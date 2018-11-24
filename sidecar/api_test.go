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

type MockMapClient struct {
	mockMapContent []byte
	mockMapError   error
}

func (m MockMapClient) GetMap(mapname string, mapversion string, dev bool) ([]byte, error) {
	return m.mockMapContent, m.mockMapError
}

func NewMockAPI(mockMapContent []byte, mockMapError error) *API {
	api := &API{
		MapClient: MockMapClient{
			mockMapError:   mockMapError,
			mockMapContent: mockMapContent,
		},
		Cache:  NewMapCache(1000, 60),
		Router: mux.NewRouter(),
	}
	return api
}

func makeRequest(mapname string, mapversion string, dev bool) *http.Request {
	target := fmt.Sprintf("/map/%s", mapname)
	if len(mapversion) > 0 {
		target = fmt.Sprintf("/map/%s/%s", mapname, mapversion)
	}
	req := httptest.NewRequest("GET", target, nil)
	urlvars := map[string]string{
		"map":     mapname,
		"version": mapversion,
	}
	req = mux.SetURLVars(req, urlvars)
	if dev {
		req.URL.Query().Add("dev", "true")
	}
	return req
}

func TestMakeKey(t *testing.T) {
	mapname := "test.map"
	mapversion := "123abc"

	expectedRes := fmt.Sprintf("%s/%s", mapname, mapversion)
	actualRes := makeKey(mapname, mapversion, false)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

	expectedRes = mapname
	actualRes = makeKey(mapname, "", false)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

	expectedRes = fmt.Sprintf("%s/dev", mapname)
	actualRes = makeKey(mapname, "", true)
	if expectedRes != actualRes {
		t.Fatalf("makeKey should match expected output. Expected: %s, Actual: %s", expectedRes, actualRes)
	}

}

func TestResolveMap(t *testing.T) {
	mockApi := NewMockAPI([]byte("whoopty doo"), nil)
	res, err := mockApi.resolveMap("ok", "", false)
	// first one should not be cached.
	if err != nil {
		t.Fatalf("resolveMap returned an error: %s", err)
	}
	if string(res) != "whoopty doo" {
		t.Fatalf("resolveMap did not return expected content: %s", string(res))
	}
	// second one should be cached
	res, err = mockApi.resolveMap("ok", "", false)
	if err != nil {
		t.Fatalf("resolveMap returned an error: %s", err)
	}
	if string(res) != "whoopty doo" {
		t.Fatalf("resolveMap did not return expected content: %s", string(res))
	}

	// make it err
	mockApi = NewMockAPI(nil, errors.New("unit test"))
	res, err = mockApi.resolveMap("ok", "", false)
	if err.Error() != "unit test" {
		t.Fatalf("resolveMap should return MapClient.GetMap error")
	}
}

func TestMapServiceClientGetMap(t *testing.T) {
	api := NewAPI(1000, 60, "https://maps-poc.spsdev.in/map/")
	// this map exists, and I know it does
	m, err := api.MapClient.GetMap("BelKaukana_co_grocery_v5010_groceryPoFedsWrite.jar", "", true)
	if err != nil {
		t.Fatalf("MapClient.GetMap returned an err: %s", err.Error())
	}
	if len(m) == 0 {
		t.Fatalf("MapClient.GetMap content length was 0")
	}
	// I know this version exists
	m, err = api.MapClient.GetMap("BelKaukana_co_grocery_v5010_groceryPoFedsWrite.jar", "56c451cd132eb85c2b433906c4d69b435dc8af4f356df760a0cbf113160aba57", false)
	if err != nil {
		t.Fatalf("MapClient.GetMap returned an err: %s", err.Error())
	}
	if len(m) == 0 {
		t.Fatalf("MapClient.GetMap content length was 0")
	}
	// I know this version of the map does not exist
	m, err = api.MapClient.GetMap("BelKaukana_co_grocery_v5010_groceryPoFedsWrite.jar", "badversion", false)
	if err == nil {
		t.Fatalf("MapClient.GetMap should have returned an error (but this test is brittle)")
	}
	if len(m) > 0 {
		t.Fatalf("MapClient.GetMap content length was 0")
	}
}

func TestAPIGetMap(t *testing.T) {
	req := makeRequest("BelKaukana_co_grocery_v5010_groceryPoFedsWrite.jar", "", false)
	res := httptest.NewRecorder()

	happyApi := NewMockAPI([]byte("super map"), nil)
	happyApi.GetMap(res, req)
	mapContent, _ := ioutil.ReadAll(res.Body)
	if string(mapContent) != "super map" {
		t.Fatalf("mapContent should be 'super map'. Was: %s", string(mapContent))
	}

	errorRes := httptest.NewRecorder()
	errorApi := NewMockAPI(nil, errors.New("unit test"))
	errorApi.GetMap(errorRes, req)
	response := &JSONResponse{}
	json.Unmarshal(errorRes.Body.Bytes(), response)
	if response.Error != "unit test" {
		t.Fatalf("GetMap error message should be 'unit test'. Is: %s", response.Error)
	}
}
