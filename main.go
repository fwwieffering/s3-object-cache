package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	bucket, bucketExists := os.LookupEnv("S3_BUCKET")
	pathPrefix, _ := os.LookupEnv("S3_PATH_PREFIX")
	dynamoTable, ddbExists := os.LookupEnv("DYNAMO_TABLE")

	if !bucketExists {
		panic("S3_BUCKET environment variable is mandatory")
	}
	if !ddbExists {
		panic("DYNAMO_TABLE environment variable is mandatory")
	}

	api := NewAPI(bucket, pathPrefix, dynamoTable)
	// TODO: graceful shutdown https://github.com/gorilla/mux#graceful-shutdown
	srv := &http.Server{
		Handler:      api.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Addr:         ":80",
	}

	log.Fatal(srv.ListenAndServe())
}
