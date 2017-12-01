package main

import (
	"github.com/fwwieffering/s3-object-cache/api"
)

func main() {
	s := api.NewS3CacheApi()
	s.Run(":8080")
}
