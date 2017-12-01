package cache

import (
	"bytes"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Source ...
// An implementation of object source with S3 as the backend
type S3Source struct {
	bucket  string
	fetcher fetcher
}

func NewS3Source(bucket string) *S3Source {
	fet := newS3Fetcher()
	ret := &S3Source{bucket: bucket, fetcher: fet}
	return ret
}

// fetcher interface enables abstracting S3 calls
type fetcher interface {
	GetObject(bucket string, key string) ([]byte, string, error)
	HeadObject(bucket string, key string) (string, error)
}

type s3Fetcher struct {
	s3 *s3.S3
}

// Initialize ...
// Sets up ObjectSource interface
func newS3Fetcher() *s3Fetcher {
	// create S3 interface
	region, present := os.LookupEnv("AWS_DEFAULT_REGION")
	if !present {
		region = "us-east-1"
	}
	ret := &s3Fetcher{}
	ret.s3 = s3.New(session.New(&aws.Config{
		Region: aws.String(region),
	}))
	return ret
}

func (s *s3Fetcher) GetObject(bucket string, key string) ([]byte, string, error) {
	res, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	// TODO: superior error handling https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#S3.GetObject examples
	if err != nil {
		return nil, "", err
	}

	// process S3 result body
	defer res.Body.Close()

	buf := bytes.NewBuffer(nil)

	if _, err = io.Copy(buf, res.Body); err != nil {
		return nil, "", err
	}

	// remove quotes from res.Etag
	etag := *res.ETag
	etag = etag[1 : len(etag)-1]

	return buf.Bytes(), etag, nil
}

func (s *s3Fetcher) HeadObject(bucket string, key string) (string, error) {
	res, err := s.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return "", err
	}
	// remove quotes from res.Etag
	etag := *res.ETag
	etag = etag[1 : len(etag)-1]
	return etag, nil
}

// FetchFromSource ...
// Grabs object from S3
func (s *S3Source) FetchFromSource(key string) ([]byte, string, error) {
	obj, tag, err := s.fetcher.GetObject(s.bucket, key)
	return obj, tag, err
}

// CheckSource ...
// checkSource finds the etag of the provided s3 key
func (s *S3Source) CheckSource(key string) (string, error) {
	tag, err := s.fetcher.HeadObject(s.bucket, key)
	return tag, err
}
