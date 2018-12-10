package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type MockObjectClient struct {
	mockObjectContent []byte
	mockObjectError   error
}

func (m MockObjectClient) GetObject(objectname string, objectversion string, dev bool) ([]byte, error) {
	return m.mockObjectContent, m.mockObjectError
}

func NewMockAPI(mockObjectContent []byte, mockObjectError error) *API {
	api := &API{
		ObjectClient: MockObjectClient{
			mockObjectError:   mockObjectError,
			mockObjectContent: mockObjectContent,
		},
		Cache:  NewObjectCache(1000, 60),
		Router: mux.NewRouter(),
	}
	return api
}

func makeRequest(categoryName string, objectName string, objectVersion string, dev bool) *http.Request {
	target := fmt.Sprintf("/%s/%s", categoryName, objectName)
	if len(objectVersion) > 0 {
		target = fmt.Sprintf("/%s/%s/%s", categoryName, objectName, objectVersion)
	}
	req := httptest.NewRequest("GET", target, nil)
	urlvars := map[string]string{
		"category": categoryName,
		"object":   objectName,
		"version":  objectVersion,
	}
	req = mux.SetURLVars(req, urlvars)
	if dev {
		req.URL.RawQuery = "dev=true"
	}
	return req
}

func TestMakeKey(t *testing.T) {
	objectname := "test.map"
	objectversion := "123abc"

	expectedRes := fmt.Sprintf("%s/%s", objectname, objectversion)
	actualRes := makeKey(objectname, objectversion, false)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

	expectedRes = objectname
	actualRes = makeKey(objectname, "", false)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

	expectedRes = fmt.Sprintf("%s/dev", objectname)
	actualRes = makeKey(objectname, "", true)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

}

func TestResolveObject(t *testing.T) {
	mockApi := NewMockAPI([]byte("whoopty doo"), nil)
	res, err := mockApi.resolveObject("ok", "", false)
	// first one should not be cached.
	if err != nil {
		t.Fatalf("resolveObject returned an error: %s", err)
	}
	if string(res) != "whoopty doo" {
		t.Fatalf("resolveObject did not return expected content: %s", string(res))
	}
	// second one should be cached
	res, err = mockApi.resolveObject("ok", "", false)
	if err != nil {
		t.Fatalf("resolveObject returned an error: %s", err)
	}
	if string(res) != "whoopty doo" {
		t.Fatalf("resolveObject did not return expected content: %s", string(res))
	}

	// make it err
	mockApi = NewMockAPI(nil, errors.New("unit test"))
	res, err = mockApi.resolveObject("ok", "", false)
	if err.Error() != "unit test" {
		t.Fatalf("resolveObject should return ObjectClient.GetObject error")
	}
}

func TestAPIGetObject(t *testing.T) {
	req := makeRequest("foo", "bar.jar", "", false)
	res := httptest.NewRecorder()

	happyApi := NewMockAPI([]byte("super object"), nil)
	happyApi.GetObject(res, req)
	objectContent, _ := ioutil.ReadAll(res.Body)
	if string(objectContent) != "super object" {
		t.Fatalf("objectContent should be 'super object'. Was: %s", string(objectContent))
	}

	errorRes := httptest.NewRecorder()
	errorApi := NewMockAPI(nil, errors.New("unit test"))
	errorApi.GetObject(errorRes, req)
	response := &JSONResponse{}
	json.Unmarshal(errorRes.Body.Bytes(), response)
	if response.Error != "unit test" {
		t.Fatalf("GetObject error message should be 'unit test'. Is: %s", response.Error)
	}
}
