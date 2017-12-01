package cache

// ObjectSource ...
type ObjectSource interface {
	FetchFromSource(string) ([]byte, string, error)
	CheckSource(string) (string, error)
}

// Cache ...
// Cache's pull from S3 but store in separate locations
type Cache interface {
	Fetch(string) ([]byte, error)
}
