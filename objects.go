package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// ObjectController object to handle the storage, retrieval, and versioning of objects in s3 and dynamo
type ObjectController struct {
	bucket *string
	path   string
	table  *string
	s3     s3iface.S3API
	ddb    dynamodbiface.DynamoDBAPI
}

// NewObjectController returns a new object controller
func NewObjectController(bucket string, pathPrefix string, table string) *ObjectController {
	var sess = session.Must(session.NewSession())
	return &ObjectController{
		bucket: aws.String(bucket),
		path:   pathPrefix,
		table:  aws.String(table),
		s3:     s3.New(sess),
		ddb:    dynamodb.New(sess),
	}
}

// GetObject Orchestrator for getting objects.
// if version is supplied attempt to pull directly from S3
// else, look up version in dynamo and return that
// TODO: add redis cache
func (o ObjectController) GetObject(objectName string, version string, dev bool) (io.ReadCloser, error) {
	if len(version) > 0 {
		// passes s3 errors upwards
		return o.getObjectFromS3(objectName, version)
	}
	version, err := o.getObjectVersion(objectName, dev)
	if err != nil {
		return nil, fmt.Errorf("Error looking up version for object %s. Error:%s", objectName, err.Error())
	}
	return o.getObjectFromS3(objectName, version)
}

// SetObjectVersion sets default prod/dev version of object objectName to version version
func (o ObjectController) SetObjectVersion(objectName string, version string) error {
	err := o.addObjectToDynamo(objectName, false, version)
	if err != nil {
		return fmt.Errorf("Unable to write object %s version %s info to dynamo. %s", objectName, version, err.Error())
	}
	return nil
}

// SetObjectDevVersion sets default dev version of object objectName to version version
func (o ObjectController) SetObjectDevVersion(objectName string, version string) error {
	err := o.addObjectToDynamo(objectName, true, version)
	if err != nil {
		return fmt.Errorf("Unable to write object %s version %s info to dynamo. %s", objectName, version, err.Error())
	}
	return nil
}

// AddObject Orchestrator for adding objects
// checks if object version already written to s3
// attempts to write objects to s3. Will not overwrite objects in S3, returns error
// sets versions in database if dev/prod flags supplied
func (o ObjectController) AddObject(objectName string, objectContent io.Reader, dev bool, prod bool, version string) error {
	objectexists, err := o.checkVersionS3(objectName, version)
	if err != nil {
		return fmt.Errorf("Unexpected error looking up object %s version %s in S3: %s", objectName, version, err.Error())
	}
	// return error if trying to redeploy same version of object
	if objectexists && !(dev || prod) {
		return fmt.Errorf("Object %s version %s already exists in S3. Not overwriting", objectName, version)
	} else if !objectexists {
		// write object to S3 if not already there
		err := o.addObjectToS3(objectName, version, objectContent)
		if err != nil {
			return fmt.Errorf("Unable to write object %s version %s to S3. Error: %s", objectName, version, err.Error())
		}
	}
	// update dynamo if dev/prod is set
	if dev {
		return o.SetObjectDevVersion(objectName, version)
	} else if prod {
		return o.SetObjectVersion(objectName, version)
	}

	return nil
}

// generateItemContent is a helper to generate the dynamodb putItem input
func generateItemContent(objectName string, dev bool, version string) map[string]*dynamodb.AttributeValue {
	item := map[string]*dynamodb.AttributeValue{
		"name": &dynamodb.AttributeValue{
			S: aws.String(objectName),
		},
	}
	// "dev" objects are for lower environments. If the dev bool is passed only set the dev version
	if dev {
		item["dev"] = &dynamodb.AttributeValue{
			S: aws.String(version),
		}
	} else { // otherwise we're setting the prod version, and the dev version should be set to the same as the prod version
		item["version"] = &dynamodb.AttributeValue{
			S: aws.String(version),
		}
		item["dev"] = &dynamodb.AttributeValue{
			S: aws.String(version),
		}
	}
	return item
}

// isRetryable helper function for determining whether an aws error is retryable
func isRetryable(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case dynamodb.ErrCodeProvisionedThroughputExceededException:
			return true
		case dynamodb.ErrCodeInternalServerError:
			return true
		default:
			return false
		}
	}
	return false
}

// puts item in dynamodb
// item primary key is name, also has columns dev and version that are versions of the item
func (o ObjectController) addObjectToDynamo(objectName string, dev bool, version string) error {
	// inline function for running put item
	putItem := func() (*dynamodb.PutItemOutput, error) {
		return o.ddb.PutItem(&dynamodb.PutItemInput{
			TableName: o.table,
			Item:      generateItemContent(objectName, dev, version),
		})
	}
	// backoff for put item
	retries := 3
	for i := 1; i < retries; i++ {
		_, err := putItem()
		if err != nil {
			if !isRetryable(err) {
				return err
			} else if i+1 < retries {
				time.Sleep(time.Duration(i) * time.Second)
			} else {
				return err
			}
		} else {
			return nil
		}
	}
	return nil
}

func (o ObjectController) getObjectFromDynamo(objectName string) (map[string]*dynamodb.AttributeValue, error) {
	// function for making aws call
	getObject := func() (*dynamodb.GetItemOutput, error) {
		return o.ddb.GetItem(&dynamodb.GetItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"name": &dynamodb.AttributeValue{S: aws.String(objectName)},
			},
			TableName: o.table,
		})
	}
	// backoff for get item
	retries := 3
	for i := 1; i < retries; i++ {
		res, err := getObject()
		if err != nil {
			if !isRetryable(err) {
				return nil, err
			} else if i+1 < retries {
				time.Sleep(time.Duration(i) * time.Second)
			} else {
				return nil, err
			}
		} else {
			return res.Item, nil
		}
	}
	return nil, nil
}

func (o ObjectController) getObjectVersion(objectName string, dev bool) (string, error) {
	item, err := o.getObjectFromDynamo(objectName)
	if err != nil {
		return "", err
	}
	if dev {
		val, ok := item["dev"]
		if ok {
			return *val.S, nil
		} else {
			return "", fmt.Errorf("No dev version set for object %s", objectName)
		}
	}
	val, ok := item["version"]
	if ok {
		return *val.S, nil
	} else {
		return "", fmt.Errorf("No version set for object %s", objectName)
	}
}

// generates the key for a object to be stored / retrieved from
func (o ObjectController) getObjectKey(objectName string, version string) string {
	// add path if present to s3 object key
	key := fmt.Sprintf("%s/%s", objectName, version)
	if len(o.path) > 0 {
		key = fmt.Sprintf("%s/%s", o.path, key)
	}
	return key
}

// returns true if version is already stored in s3, false otherwise
func (o ObjectController) checkVersionS3(objectName string, version string) (bool, error) {
	key := o.getObjectKey(objectName, version)

	_, err := o.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: o.bucket,
		Key:    aws.String(key),
	})

	// head object returns error if object does not exist
	aerr, ok := err.(awserr.Error)

	// head object doesn't return ErrCodeNoSuchKey
	if ok && aerr.Code() == "NotFound" {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (o ObjectController) addObjectToS3(objectName string, version string, objectContent io.Reader) error {
	// add path if present to s3 object key
	key := o.getObjectKey(objectName, version)

	// have to know ContentLength
	byteArray, readErr := ioutil.ReadAll(objectContent)
	if readErr != nil {
		fmt.Printf("readerr %v\n", readErr)
		return readErr
	}

	byteReader := bytes.NewReader(byteArray)
	// I don't think there are retryable errors for s3.putobject
	_, err := o.s3.PutObject(&s3.PutObjectInput{
		Bucket:        o.bucket,
		Key:           aws.String(key),
		Body:          aws.ReadSeekCloser(byteReader),
		ContentLength: aws.Int64(int64(byteReader.Len())),
	})

	return err
}

func (o ObjectController) getObjectFromS3(objectName string, version string) (io.ReadCloser, error) {
	key := o.getObjectKey(objectName, version)
	res, err := o.s3.GetObject(&s3.GetObjectInput{
		Bucket: o.bucket,
		Key:    aws.String(key),
	})

	if err != nil {
		aerr, ok := err.(awserr.Error)
		// format not found errors nicely
		if ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, fmt.Errorf("Object %s version %s does not exist", objectName, version)
		}
		return nil, err
	}
	return res.Body, nil
}
