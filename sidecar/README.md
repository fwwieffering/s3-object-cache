# object-service Sidecar
A sidecar container to handle fetching objects from object service and implementing a local cache.

## How To Use
Run this container as a daemonset or as a sidecar container.

To request an object:
- `GET` `/{category}/{object_name}` get default map version. Can get dev default version by providing query parameter `?dev=true`. Returns map binary
- `GET` `/{category}/{object_name}/{object_version}` get specific map version. Returns map binary

## Caching
The container implements an LRU cache to store objects locally. If the requested object/version is not present in the in-memory cache it is fetched from object-service and placed in the cache. The cache implementation used is the TwoQueueCache from [hashicorps golang-lru cache implentation](https://github.com/hashicorp/golang-lru).

>TwoQueueCache tracks frequently used and recently used entries separately. This avoids a burst of accesses from taking out frequently used entries

In addition to the LRU cache, each item in the cache expires in the configurable `CACHE_EXPIRY_SECONDS` to force an update.

## Configuration
Can configure
- number of entries in cache
- cache item expiration

Environment variable configuration:

| variable name | default  | description |
|---------------|----------|-------------|
| `CACHE_SIZE` | 1000 | number of entries to keep in the cache |
| `CACHE_EXPIRY_SECONDS` | 300 | seconds to keep maps cached |
