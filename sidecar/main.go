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
	var objectServiceURL string

	cacheSizeParam, sizeOk := os.LookupEnv("CACHE_SIZE")
	cacheExpirySecondsParam, expiryOk := os.LookupEnv("CACHE_EXPIRY_SECONDS")
	objectserviceParam, objectserviceparamOK := os.LookupEnv("OBJECT_SERVICE_URL")

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

	if objectserviceparamOK {
		objectServiceURL = objectserviceParam
	} else {
		log.Fatal("OBJECT_SERVICE_URL must be provided as an environment variable")
	}

	api := NewAPI(cacheSize, cacheExpirySeconds, objectServiceURL)

	srv := &http.Server{
		Handler:      api.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Addr:         ":80",
	}

	log.Fatal(srv.ListenAndServe())
}
