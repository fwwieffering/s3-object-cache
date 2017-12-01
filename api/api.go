package api

import (
	"log"
	"net/http"
	"os"

	"github.com/fwwieffering/s3-object-cache/cache"

	"github.com/gorilla/mux"
)

type S3CacheAPI struct {
	Router     *mux.Router
	Source     *cache.S3Source
	LocalCache *cache.LocalCache
	Redis      *cache.RedisCache
}

func NewS3CacheApi() *S3CacheAPI {
	bucket, present := os.LookupEnv("BUCKET")
	if !present {
		panic("BUCKET environment variable not set")
	}
	source := cache.NewS3Source(bucket)
	local := &cache.LocalCache{Src: source}
	redis := cache.NewRedisCache(source)

	router := mux.NewRouter()
	ret := &S3CacheAPI{
		Router:     router,
		LocalCache: local,
		Source:     source,
		Redis:      redis,
	}
	router.HandleFunc("/s3/{s3_path:.*}", ret.FetchObjectS3).Methods("GET", "OPTIONS")
	router.HandleFunc("/local/{s3_path:.*}", ret.FetchObjectLocalCache).Methods("GET", "OPTIONS")
	router.HandleFunc("/redis/{s3_path:.*}", ret.FetchObjectRedisCache).Methods("GET", "OPTIONS")
	return ret
}

func (s *S3CacheAPI) FetchObjectS3(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["s3_path"]

	obj, _, err := s.Source.FetchFromSource(key)

	if err != nil {
		w.Write([]byte(err.Error()))
	}
	w.Write(obj)
}

func (s *S3CacheAPI) FetchObjectLocalCache(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["s3_path"]

	obj, err := s.LocalCache.Fetch(key)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	w.Write(obj)
}

func (s *S3CacheAPI) FetchObjectRedisCache(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["s3_path"]

	obj, err := s.Redis.Fetch(key)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	w.Write(obj)
}

func (s *S3CacheAPI) Run(addr string) {
	log.Fatal(http.ListenAndServe(":8080", s.Router))
}
