package cache

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// LocalCache ...
// Stores objects on local filesystem once pulled
type LocalCache struct {
	Src ObjectSource
}

func storeObjectLocalCache(filename string, body []byte) error {
	// make sure directory path exists
	pathParts := strings.Split(filename, "/")
	path := strings.Join(pathParts[0:len(pathParts)-1], "/")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Print("Error creating directory")
		log.Print(err)
		return err
	}
	err = ioutil.WriteFile(filename, body, 0644)
	// err is likely nil
	if err != nil {
		log.Print("Error writing file")
		log.Print(err)
		return err
	}
	return nil
}

// Fetch ...
// Checks for object stored on FS. Otherwise, pulls from S3 and returns
func (l *LocalCache) Fetch(key string) ([]byte, error) {
	// get tag from S3
	version, chkErr := l.Src.CheckSource(key)
	if chkErr != nil {
		return nil, chkErr
	}
	filename := key + "-" + version
	if _, osErr := os.Stat(filename); os.IsNotExist(osErr) {
		log.Printf("%v not stored locally", filename)
		log.Printf("Fetching %s from S3", key)
		body, id, SrcErr := l.Src.FetchFromSource(key)
		if SrcErr != nil {
			return nil, SrcErr
		}
		log.Printf("Storing %s in local cache...", key)
		storeObjectLocalCache(key+"-"+id, body)
		return body, nil
	}

	log.Printf("%s found stored locally. Reading from disk...", key)
	body, err := ioutil.ReadFile(key + "-" + version)
	if err != nil {
		return nil, err
	}
	return body, nil

}
