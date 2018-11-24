package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type JSONResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

type API struct {
	Router    *mux.Router
	Cache     Cache
	MapClient MapClient
}

func NewAPI(cacheSize int, cacheExpirySeconds int, url string) *API {
	router := mux.NewRouter()

	api := &API{
		Cache:     NewMapCache(cacheSize, cacheExpirySeconds),
		Router:    router,
		MapClient: NewMapServiceClient(url),
	}

	router.HandleFunc("/map/{map}/{version}", api.GetMap).Methods("GET")
	router.HandleFunc("/map/{map}", api.GetMap).Methods("GET")

	router.Use(loggingMiddleware)

	return api
}

type MapClient interface {
	GetMap(mapname string, mapversion string, dev bool) ([]byte, error)
}

type MapServiceClient struct {
	MapServiceURL string
}

// TODO: configurable map service url
func NewMapServiceClient(url string) MapServiceClient {
	return MapServiceClient{
		MapServiceURL: url,
	}
}

func (m MapServiceClient) GetMap(mapname string, mapversion string, dev bool) ([]byte, error) {
	var endpoint = fmt.Sprintf("%s", mapname)
	if len(mapversion) > 0 {
		endpoint += fmt.Sprintf("/%s", mapversion)
	}

	if dev && len(mapversion) == 0 {
		endpoint += "?dev=true"
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}
	res, err := client.Get(m.MapServiceURL + endpoint)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 200 {
		mapcontent, _ := ioutil.ReadAll(res.Body)
		return mapcontent, nil
	} else {
		body, _ := ioutil.ReadAll(res.Body)

		errMsg := &JSONResponse{}
		json.Unmarshal(body, errMsg)
		return nil, errors.New(errMsg.Error)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(fmt.Sprintf("%s %s", r.RequestURI, r.Method))
		next.ServeHTTP(w, r)
	})
}

func makeKey(mapname string, mapversion string, dev bool) string {
	if len(mapversion) > 0 {
		return fmt.Sprintf("%s/%s", mapname, mapversion)
	} else if dev {
		return fmt.Sprintf("%s/dev", mapname)
	}
	return mapname
}

// resolvemap fetches a map from the cache or from the map service, if needed
func (a API) resolveMap(mapname string, mapversion string, dev bool) ([]byte, error) {
	cacheKey := makeKey(mapname, mapversion, dev)

	mapIface, exists := a.Cache.Get(cacheKey)
	var mapContent []byte
	var err error

	if exists {
		fmt.Printf("Found map %s in cache\n", mapname)
		mapContent = mapIface.([]byte)
	} else {
		fmt.Printf("Map %s not in cache, pulling from map service\n", mapname)
		mapContent, err = a.MapClient.GetMap(mapname, mapversion, dev)
		if err != nil {
			return nil, err
		}
		a.Cache.Add(cacheKey, mapContent)
	}

	return mapContent, nil
}

func (a API) GetMap(res http.ResponseWriter, req *http.Request) {
	routeVars := mux.Vars(req)
	mapVersion := routeVars["version"]
	mapName := routeVars["map"]

	dev := req.URL.Query().Get("dev")
	devParam := strings.ToLower(dev) == "true"

	mapcontent, err := a.resolveMap(mapName, mapVersion, devParam)
	if err == nil {
		res.Write(mapcontent)
		res.Header().Set("Content-Type", "application/java-archive")
	} else {
		responseBody, _ := json.Marshal(JSONResponse{
			Status: "error",
			Error:  err.Error(),
		})
		res.Write(responseBody)
		res.WriteHeader(http.StatusInternalServerError)
	}
}
