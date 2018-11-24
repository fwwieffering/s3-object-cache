package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	var cacheSize int
	var cacheExpirySeconds int
	var mapServiceURL string

	cacheSizeParam, sizeOk := os.LookupEnv("CACHE_SIZE")
	cacheExpirySecondsParam, expiryOk := os.LookupEnv("CACHE_EXPIRY_SECONDS")
	mapserviceParam, mapserviceparamOK := os.LookupEnv("MAP_SERVICE_URL")

	cacheSize = 1000
	if sizeOk {
		cacheSize, err := strconv.Atoi(cacheSizeParam)
		if err != nil {
			fmt.Printf("Unable to parse CACHE_SIZE %s as int. Using default %d", cacheSizeParam, cacheSize)
		}
	}

	cacheExpirySeconds = 300
	if expiryOk {
		cacheExpirySeconds, err := strconv.Atoi(cacheExpirySecondsParam)
		if err != nil {
			cacheExpirySeconds = 1000
			fmt.Printf("Unable to parse CACHE_SIZE %s as int. Using default %d", cacheExpirySecondsParam, cacheExpirySeconds)
		}
	}

	mapServiceURL = "https://maps-poc.spsdev.in/map/"
	if mapserviceparamOK {
		mapServiceURL = mapserviceParam
	}

	api := NewAPI(cacheSize, cacheExpirySeconds, mapServiceURL)

	srv := &http.Server{
		Handler:      api.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Addr:         ":80",
	}

	log.Fatal(srv.ListenAndServe())
}
