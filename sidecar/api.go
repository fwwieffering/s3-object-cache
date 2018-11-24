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
	Router       *mux.Router
	Cache        Cache
	ObjectClient ObjectClient
}

func NewAPI(cacheSize int, cacheExpirySeconds int, url string) *API {
	router := mux.NewRouter()

	api := &API{
		Cache:        NewObjectCache(cacheSize, cacheExpirySeconds),
		Router:       router,
		ObjectClient: NewObjectServiceClient(url),
	}

	router.HandleFunc("/{category}/{object}/{version}", api.GetObject).Methods("GET")
	router.HandleFunc("/{category}/{object}", api.GetObject).Methods("GET")

	router.Use(loggingMiddleware)

	return api
}

type ObjectClient interface {
	GetObject(objectname string, objectversion string, dev bool) ([]byte, error)
}

type ObjectServiceClient struct {
	ObjectServiceURL string
}

func NewObjectServiceClient(url string) ObjectServiceClient {
	return ObjectServiceClient{
		ObjectServiceURL: url,
	}
}

func (o ObjectServiceClient) GetObject(objectname string, objectversion string, dev bool) ([]byte, error) {
	var endpoint = fmt.Sprintf("%s", objectname)
	if len(objectversion) > 0 {
		endpoint += fmt.Sprintf("/%s", objectversion)
	}

	if dev && len(objectversion) == 0 {
		endpoint += "?dev=true"
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}
	res, err := client.Get(o.ObjectServiceURL + endpoint)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 200 {
		objectcontent, _ := ioutil.ReadAll(res.Body)
		return objectcontent, nil
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

func makeKey(objectname string, objectversion string, dev bool) string {
	if len(objectversion) > 0 {
		return fmt.Sprintf("%s/%s", objectname, objectversion)
	} else if dev {
		return fmt.Sprintf("%s/dev", objectname)
	}
	return objectname
}

// resolveobject fetches a object from the cache or from the object service, if needed
func (a API) resolveObject(objectname string, objectversion string, dev bool) ([]byte, error) {
	cacheKey := makeKey(objectname, objectversion, dev)

	objectIface, exists := a.Cache.Get(cacheKey)
	var objectContent []byte
	var err error

	if exists {
		fmt.Printf("Found object %s in cache\n", objectname)
		objectContent = objectIface.([]byte)
	} else {
		fmt.Printf("Object %s not in cache, pulling from object service\n", objectname)
		objectContent, err = a.ObjectClient.GetObject(objectname, objectversion, dev)
		if err != nil {
			return nil, err
		}
		a.Cache.Add(cacheKey, objectContent)
	}

	return objectContent, nil
}

func (a API) GetObject(res http.ResponseWriter, req *http.Request) {
	routeVars := mux.Vars(req)
	categoryName := routeVars["category"]
	objectVersion := routeVars["version"]
	objectName := routeVars["object"]
	objectKey := fmt.Sprintf("%s/%s", categoryName, objectName)

	dev := req.URL.Query().Get("dev")
	devParam := strings.ToLower(dev) == "true"

	objectcontent, err := a.resolveObject(objectKey, objectVersion, devParam)
	if err == nil {
		res.Write(objectcontent)
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
