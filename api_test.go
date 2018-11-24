package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
)

func makeRequest(categoryName string, objectName string, objectVersion string, method string, dev string, body io.Reader) *http.Request {
	target := fmt.Sprintf("/%s/%s", categoryName, objectName)
	if len(objectVersion) > 0 {
		target = fmt.Sprintf("/%s/%s/%s", categoryName, objectName, objectVersion)
	}
	req := httptest.NewRequest(method, target, body)
	urlvars := map[string]string{
		"category": categoryName,
		"object":   objectName,
		"version":  objectVersion,
	}
	req = mux.SetURLVars(req, urlvars)
	if len(dev) > 0 {
		req.URL.RawQuery = fmt.Sprintf("dev=%s", dev)
	}
	return req
}

func NewMockAPI() *API {
	return &API{
		Objects: &ObjectController{
			bucket: aws.String("unit test"),
			path:   "dang",
			table:  aws.String("unit test"),
			s3: &MockS3{
				bucket: make(map[string]string),
			},
			ddb: &MockDynamo{
				items: []map[string]*dynamodb.AttributeValue{},
			},
		},
	}
}

func TestUpPageHandler(t *testing.T) {
	api := NewMockAPI()
	res := httptest.NewRecorder()

	api.UpPageHandler(res, &http.Request{})

	content := res.Body.String()
	if content != "Happy" {
		t.Fatalf("Up page should always return 'Happy'. Returned: %s", content)
	}
}

func TestAddObjectHandler(t *testing.T) {
	api := NewMockAPI()
	res := httptest.NewRecorder()

	req := makeRequest("foo", "test.map.yo", "123ABC", "POST", "", aws.ReadSeekCloser(strings.NewReader("secret sauce")))

	api.AddObjectHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("AddObjectHandler should have returned a success. Status code: %d", res.Code)
	}
	response := &JSONResponse{}
	_ = json.Unmarshal(res.Body.Bytes(), response)
	if response.Status != "ok" {
		t.Fatalf("reponse status should be ok on successful response. Was: %s", response.Status)
	}

	objectContent, err := api.Objects.GetObject("foo/test.map.yo", "123ABC", false)
	if err != nil {
		t.Fatalf("Unable to get map after storing it. Error: %s", err.Error())
	}
	content, _ := ioutil.ReadAll(objectContent)
	if string(content) != "secret sauce" {
		t.Fatalf("Stored map should have the same content as retrieved map. Should be 'secret_sauce'. Was: '%s'", string(content))
	}
}

func TestGetObjectHandler(t *testing.T) {
	api := NewMockAPI()
	res := httptest.NewRecorder()

	req := makeRequest("foo", "test.map.yo", "123ABC", "POST", "", aws.ReadSeekCloser(strings.NewReader("secret sauce")))

	api.AddObjectHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("AddObjectHandler should have returned a success. Status code: %d", res.Code)
	}
	response := &JSONResponse{}
	_ = json.Unmarshal(res.Body.Bytes(), response)
	if response.Status != "ok" {
		t.Fatalf("reponse status should be ok on successful response. Was: %s", response.Status)
	}

	getReq := makeRequest("foo", "test.map.yo", "123ABC", "GET", "", nil)
	getRes := httptest.NewRecorder()
	api.GetObjectHandler(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GetObject request should be successful. Response code: %d", getRes.Code)
	}
	if getRes.Header().Get("Content-Type") != "application/java-archive" {
		t.Fatalf("GetObjectHandler must set content type to application/java-archive. Is: %s", res.Header().Get("content-type"))
	}
	content := getRes.Body.String()
	if string(content) != "secret sauce" {
		t.Fatalf("Stored map should have the same content as retrieved map. Should be 'secret_sauce'. Was: '%s'", string(content))
	}
}

func TestSetObjectVznHandler(t *testing.T) {
	api := NewMockAPI()
	// add object, dont set version
	res := httptest.NewRecorder()
	req := makeRequest("foo", "test.map.yo", "123ABC", "POST", "", aws.ReadSeekCloser(strings.NewReader("secret sauce")))
	api.AddObjectHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("AddObjectHandler should have returned a success. Status code: %d", res.Code)
	}
	// Get unversioned object. SHOULD FAIL b/c version has not been set
	getObjReq := makeRequest("foo", "test.map.yo", "", "GET", "", nil)
	getObjRes := httptest.NewRecorder()

	api.GetObjectHandler(getObjRes, getObjReq)
	if getObjRes.Code == http.StatusOK {
		t.Fatalf("GetObjectHandler should fail when making an unversioned request to an object without a set version")
	}
	// set object dev version
	setVznReqDev := makeRequest("foo", "test.map.yo", "123ABC", "PUT", "true", nil)
	setVznResDev := httptest.NewRecorder()
	api.SetObjectVersion(setVznResDev, setVznReqDev)
	if setVznResDev.Code != http.StatusOK {
		t.Fatalf("SetObjectVersion should have returned a success. Status code: %d", setVznResDev.Code)
	}
	// Get Object Dev Version should succeed
	res = httptest.NewRecorder()
	req = makeRequest("foo", "test.map.yo", "", "GET", "true", aws.ReadSeekCloser(strings.NewReader("secret sauce")))
	api.GetObjectHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GetObjectHandler should have returned a success. Status code: %d", res.Code)
	}
	// Get Object Prod Version should fail
	res = httptest.NewRecorder()
	req = makeRequest("foo", "test.map.yo", "", "GET", "", aws.ReadSeekCloser(strings.NewReader("secret sauce")))
	api.GetObjectHandler(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("GetObjectHandler should have failed because prod version was not set. Status code: %d", res.Code)
	}
	// set object prod version
	setVznReq := makeRequest("foo", "test.map.yo", "123ABC", "PUT", "", nil)
	setVznRes := httptest.NewRecorder()
	api.SetObjectVersion(setVznRes, setVznReq)
	if setVznRes.Code != http.StatusOK {
		t.Fatalf("SetObjectVersion should have returned a success. Status code: %d", setVznResDev.Code)
	}
	// get object prod version should succeed
	res = httptest.NewRecorder()
	req = makeRequest("foo", "test.map.yo", "", "GET", "", aws.ReadSeekCloser(strings.NewReader("secret sauce")))
	api.GetObjectHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GetObjectHandler should have succeeded. Status code: %d", res.Code)
	}
}
