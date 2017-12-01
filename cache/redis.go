package cache

import (
	"log"
	"os"

	"gopkg.in/redis.v3"
)

// RedisCache pulls from source and stores items in redis database
type RedisCache struct {
	Src   ObjectSource
	Redis *redis.Client
}

// NewRedisCache creates an instance of RedisCache with the url REDIS_URL
func NewRedisCache(source ObjectSource) *RedisCache {
	redisURL, present := os.LookupEnv("REDIS_URL")
	if !present {
		panic("Environment Variable REDIS_URL is not defined!!")
	}
	log.Printf("Redis url: %s", redisURL)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	ret := &RedisCache{Src: source, Redis: redisClient}
	return ret
}

func (r *RedisCache) storeObjectRedis(key string, body []byte) error {
	r.Redis.Append(key, string(body))
	// err := res.Err()
	return nil
}

func (r *RedisCache) getObjectRedis(key string) ([]byte, error) {
	res := r.Redis.Get(key)
	val, _ := res.Bytes()
	return val, nil
}

func (r *RedisCache) Fetch(key string) ([]byte, error) {
	version, chkErr := r.Src.CheckSource(key)
	if chkErr != nil {
		return nil, chkErr
	}
	val, err := r.getObjectRedis(version)
	if err != nil {
		log.Printf("ERROR ON REDIS GET: %s", err)
		return nil, err
	}
	if val == nil {
		log.Printf("%s not found in Redis", key)
	} else {
		log.Printf("%s found in Redis", key)
		return val, nil
	}
	body, id, srcErr := r.Src.FetchFromSource(key)
	if srcErr != nil {
		return nil, srcErr
	}
	storeErr := r.storeObjectRedis(id, body)
	if storeErr != nil {
		log.Printf("ERROR ON REDIS APPEND: %s", storeErr)
		return nil, storeErr
	}
	return body, nil
}
