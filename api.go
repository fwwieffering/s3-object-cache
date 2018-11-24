package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type JSONResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

type RequestVars struct {
	CategoryName  string
	ObjectName    string
	ObjectPath    string
	ObjectVersion string
	Dev           bool
}

type API struct {
	Objects *ObjectController
	Router  *mux.Router
}

func processRequest(req *http.Request) *RequestVars {
	routeVars := mux.Vars(req)

	categoryName := routeVars["category"]
	objectVersion := routeVars["version"]
	objectName := routeVars["object"]
	dev := req.URL.Query().Get("dev")
	devParam := strings.ToLower(dev) == "true"

	return &RequestVars{
		CategoryName:  categoryName,
		ObjectName:    objectName,
		ObjectPath:    fmt.Sprintf("%s/%s", categoryName, objectName),
		ObjectVersion: objectVersion,
		Dev:           devParam,
	}
}

// TODO: add cors
// TODO: Authentication

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(fmt.Sprintf("%s %s", r.RequestURI, r.Method))
		next.ServeHTTP(w, r)
	})
}

func NewAPI(bucket string, path string, table string) *API {
	router := mux.NewRouter()

	api := &API{
		Objects: NewObjectController(bucket, path, table),
		Router:  router,
	}

	router.HandleFunc("/up", api.UpPageHandler).Methods("GET")
	router.HandleFunc("/{category}/{object}/{version}", api.AddObjectHandler).Methods("POST")
	router.HandleFunc("/{category}/{object}/{version}", api.GetObjectHandler).Methods("GET")
	router.HandleFunc("/{category}/{object}/{version}", api.SetObjectVersion).Methods("PUT")
	router.HandleFunc("/{category}/{object}", api.GetObjectHandler).Methods("GET")
	router.Use(loggingMiddleware)
	return api
}

// TODO: improve HTTP response codes. All errors are passed as 5XX, but some generate from bad requests

func (a API) UpPageHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Happy"))
}

func (a API) AddObjectHandler(res http.ResponseWriter, req *http.Request) {
	objectContent := req.Body
	reqVars := processRequest(req)

	addObjectErr := a.Objects.AddObject(reqVars.ObjectPath, objectContent, false, false, reqVars.ObjectVersion)

	// return json response for addobject
	if addObjectErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "error",
			Error:  addObjectErr.Error(),
		})
		res.Write(response)
	} else {
		res.WriteHeader(http.StatusOK)
		response, _ := json.Marshal(JSONResponse{
			Status: "ok",
		})
		res.Write(response)
	}
}

func (a API) GetObjectHandler(res http.ResponseWriter, req *http.Request) {
	reqVars := processRequest(req)

	objectReader, getObjectErr := a.Objects.GetObject(reqVars.ObjectPath, reqVars.ObjectVersion, reqVars.Dev)

	if getObjectErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "error",
			Error:  getObjectErr.Error(),
		})
		res.Write(response)
	} else {
		objectContent, objectReadErr := ioutil.ReadAll(objectReader)
		if objectReadErr != nil {
			res.WriteHeader(http.StatusInternalServerError)
			response, _ := json.Marshal(JSONResponse{
				Status: "error",
				Error:  objectReadErr.Error(),
			})
			res.Write(response)
		} else {
			res.WriteHeader(http.StatusOK)
			res.Write(objectContent)
			res.Header().Set("Content-Type", "application/java-archive")
		}
	}
}

func (a API) SetObjectVersion(res http.ResponseWriter, req *http.Request) {
	reqVars := processRequest(req)

	var setvznerr error
	if reqVars.Dev {
		setvznerr = a.Objects.SetObjectDevVersion(reqVars.ObjectPath, reqVars.ObjectVersion)
	} else {
		setvznerr = a.Objects.SetObjectVersion(reqVars.ObjectPath, reqVars.ObjectVersion)
	}

	if setvznerr != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "error",
			Error:  setvznerr.Error(),
		})
		res.Write(response)
	} else {
		res.WriteHeader(http.StatusOK)
		response, _ := json.Marshal(JSONResponse{
			Status: "ok",
		})
		res.Write(response)
	}
}
