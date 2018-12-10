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

// JSONResponse a struct to ensure responses are in a consistent format
type JSONResponse struct {
	Status    string   `json:"status"`
	Error     string   `json:"error,omitempty"`
	Message   string   `json:"message,omitempty"`
	Version   string   `json:"version,omitempty"`
	NextToken string   `json:"nextToken,omitempty"`
	Items     []string `json:"items,omitempty"`
}

// RequestVars an object to hold the parameters from a request
type RequestVars struct {
	CategoryName  string
	ObjectName    string
	ObjectPath    string
	ObjectVersion string
	Dev           bool
	Token         string
}

// API the api object, which has a router and the object controller
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
	token := req.URL.Query().Get("token")

	return &RequestVars{
		CategoryName:  categoryName,
		ObjectName:    objectName,
		ObjectPath:    fmt.Sprintf("%s/%s", categoryName, objectName),
		ObjectVersion: objectVersion,
		Dev:           devParam,
		Token:         token,
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

//
func NewAPI(bucket string, path string, table string) *API {
	router := mux.NewRouter()

	api := &API{
		Objects: NewObjectController(bucket, path, table),
		Router:  router,
	}

	router.HandleFunc("/up", api.UpPageHandler).Methods("GET")
	router.HandleFunc("/", api.ListCategoriesHandler).Methods("GET")
	router.HandleFunc("/{category}", api.ListObjectsHandler).Methods("GET")
	router.HandleFunc("/{category}/{object}/versions", api.ListObjectVersionsHandler).Methods("GET")
	router.HandleFunc("/{category}/{object}/{version}", api.AddObjectHandler).Methods("POST")
	router.HandleFunc("/{category}/{object}/{version}", api.GetObjectHandler).Methods("GET")
	router.HandleFunc("/{category}/{object}/{version}", api.SetObjectVersion).Methods("PUT")
	router.HandleFunc("/{category}/{object}", api.GetObjectHandler).Methods("GET")
	router.Use(loggingMiddleware)
	return api
}

// TODO: improve HTTP response codes. All errors are passed as 5XX, but some generate from bad requests

// UpPageHandler handles up page requests, always returns happy
func (a API) UpPageHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Happy"))
}

// ListCategoriesHandler returns list of categories specified
func (a API) ListCategoriesHandler(res http.ResponseWriter, req *http.Request) {
	reqVars := processRequest(req)

	list, err := a.Objects.ListCategories(reqVars.Token)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "err",
			Error:  err.Error(),
		})
		res.Write(response)
	} else {
		response := JSONResponse{
			Status: "ok",
			Items:  list.Objects,
		}
		if len(list.Token) > 0 {
			response.NextToken = list.Token
		}
		content, _ := json.Marshal(response)
		res.Write(content)
	}
}

// ListObjectsHandler returns list of objects in a category
func (a API) ListObjectsHandler(res http.ResponseWriter, req *http.Request) {
	reqVars := processRequest(req)

	list, err := a.Objects.ListObjects(reqVars.CategoryName, reqVars.Token)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "err",
			Error:  err.Error(),
		})
		res.Write(response)
	} else {
		res.WriteHeader(http.StatusOK)
		response := JSONResponse{
			Status: "ok",
			Items:  list.Objects,
		}
		if len(list.Token) > 0 {
			response.NextToken = list.Token
		}
		content, _ := json.Marshal(response)
		res.Write(content)
	}
}

// ListObjectVersionsHandler returns a paginated list of object versions
func (a API) ListObjectVersionsHandler(res http.ResponseWriter, req *http.Request) {
	reqVars := processRequest(req)

	list, err := a.Objects.ListObjectVersions(reqVars.CategoryName, reqVars.ObjectName, reqVars.Token)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		response, _ := json.Marshal(JSONResponse{
			Status: "err",
			Error:  err.Error(),
		})
		res.Write(response)
	} else {
		res.WriteHeader(http.StatusOK)
		response := JSONResponse{
			Status: "ok",
			Items:  list.Objects,
		}
		if len(list.Token) > 0 {
			response.NextToken = list.Token
		}
		content, _ := json.Marshal(response)
		res.Write(content)
	}
}

// AddObjectHandler POST requests to add object to cache
// request body: object content
// category/object/version in url params
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

// GetObjectHandler GET requests to get object content
// category/object/version(optional) in url params
// pulls default version of map if no version is provided and version is set
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
