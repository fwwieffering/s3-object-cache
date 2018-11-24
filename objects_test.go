package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type MockDynamo struct {
	dynamodbiface.DynamoDBAPI
	items      []map[string]*dynamodb.AttributeValue
	putItemErr []error
	getItemErr []error
}

func (d *MockDynamo) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	if len(d.putItemErr) > 0 && d.putItemErr[0] == nil {
		d.items = append(d.items, input.Item)
		return nil, nil
	} else if len(d.putItemErr) == 0 {
		d.items = append(d.items, input.Item)
		return nil, nil
	} else {
		err := d.putItemErr[0]
		d.putItemErr = d.putItemErr[1:]
		return nil, err
	}
}

func (d *MockDynamo) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	var returnItem map[string]*dynamodb.AttributeValue

	findItem := func() (*dynamodb.GetItemOutput, error) {
		for _, item := range d.items {
			if *item["name"].S == *input.Key["name"].S {
				returnItem = item
			}
		}
		return &dynamodb.GetItemOutput{
			Item: returnItem,
		}, nil
	}

	if len(d.getItemErr) > 0 && d.getItemErr[0] == nil {
		return findItem()
	} else if len(d.getItemErr) == 0 {
		return findItem()
	} else {
		err := d.getItemErr[0]
		d.getItemErr = d.getItemErr[1:]
		return nil, err
	}
}

type MockS3 struct {
	s3iface.S3API
	bucket        map[string]string
	putObjectErr  error
	getObjectErr  error
	headObjectErr error
}

func (m *MockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if m.putObjectErr != nil {
		return nil, m.putObjectErr
	}
	content, _ := ioutil.ReadAll(input.Body)
	m.bucket[*input.Key] = string(content)
	return &s3.PutObjectOutput{}, nil
}

func (m MockS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if m.getObjectErr != nil {
		return nil, m.getObjectErr
	}
	body, ok := m.bucket[*input.Key]

	if ok {
		return &s3.GetObjectOutput{
			Body: aws.ReadSeekCloser(strings.NewReader(body)),
		}, nil
	}
	return nil, awserr.New(s3.ErrCodeNoSuchKey, fmt.Sprintf("object %s does not exist", *input.Key), errors.New("the heck happened"))
}

func (m MockS3) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	if m.headObjectErr != nil {
		return nil, m.headObjectErr
	}
	_, ok := m.bucket[*input.Key]
	if !ok {
		return nil, awserr.New("NotFound", "no such key", errors.New("ok"))
	}
	return nil, nil
}

func TestAddObjectToDynamo(t *testing.T) {
	mocker := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			items: []map[string]*dynamodb.AttributeValue{},
		},
	}

	mocker.addObjectToDynamo("prod object", false, "123")
	mocker.addObjectToDynamo("dev object", true, "456")

	prodObjectVersion, err := mocker.getObjectVersion("prod object", false)
	if err != nil || prodObjectVersion != "123" {
		t.Fatalf("addObjectToDynamo should have set version to: %+v. Is: %+v", "123", prodObjectVersion)
	}
	prodObjectDevVersion, err := mocker.getObjectVersion("prod object", true)
	if err != nil || prodObjectDevVersion != "123" {
		t.Fatalf("addObjectToDynamo should have set dev version to: %+v. Is: %+v", "123", prodObjectDevVersion)
	}

	devObjectVersion, err := mocker.getObjectVersion("dev object", false)
	if err == nil {
		t.Fatalf("addObjectToDynamo should not have set prod version when new object is created for dev: Prod version: %s", devObjectVersion)
	}
	devObjectDevVersion, err := mocker.getObjectVersion("dev object", true)
	if err != nil || devObjectDevVersion != "456" {
		t.Fatalf("addObjectToDynamo should have set dev version to: %+v. Is: %+v", "456", devObjectDevVersion)
	}
}

func TestAddObjectToDynamoRetries(t *testing.T) {
	retryable := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			putItemErr: []error{
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
			},
		},
	}

	err := retryable.addObjectToDynamo("unite test", false, "yup")
	if err != nil {
		t.Fatalf("ProvisionedThroughPutExceeded errors should be retried. Received error: %v", err.Error())
	}

	notRetryable := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			putItemErr: []error{
				errors.New("hot dang"),
			},
		},
	}
	err = notRetryable.addObjectToDynamo("unite test", false, "yup")
	if err == nil {
		t.Fatalf("non aws errors should returned. Did not receive error")
	}

	exceedRetries := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			putItemErr: []error{
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeInternalServerError, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeBackupInUseException, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
			},
		},
	}
	err = exceedRetries.addObjectToDynamo("unite test", false, "yup")
	if err == nil {
		t.Fatalf("error should be returned when retries are exceeded. Did not receive error")
	}
}

func TestGetObjectFromDynamoRetries(t *testing.T) {
	retryable := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			getItemErr: []error{
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
			},
		},
	}
	retryable.addObjectToDynamo("unit test", false, "123")
	_, err := retryable.getObjectFromDynamo("unit test")
	if err != nil {
		t.Fatalf("ProvisionedThroughPutExceeded errors should be retried. Received error: %v", err.Error())
	}

	notRetryable := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			getItemErr: []error{
				errors.New("hot dang"),
			},
		},
	}
	notRetryable.addObjectToDynamo("unit test", false, "123")
	_, err = notRetryable.getObjectFromDynamo("unit test")
	if err == nil {
		t.Fatalf("non aws errors should returned. Did not receive error")
	}

	exceedRetries := ObjectController{
		table: aws.String("unit test"),
		ddb: &MockDynamo{
			getItemErr: []error{
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeInternalServerError, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeBackupInUseException, "poo", errors.New("ok")),
				awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "poo", errors.New("ok")),
			},
		},
	}
	err = exceedRetries.addObjectToDynamo("unit test", false, "yup")
	_, err = exceedRetries.getObjectFromDynamo("unit test")
	if err == nil {
		t.Fatalf("error should be returned when retries are exceeded. Did not receive error")
	}
}

func TestAddObjectToS3(t *testing.T) {
	mocker := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		s3: &MockS3{
			bucket: make(map[string]string),
		},
	}

	err := mocker.addObjectToS3("unit test", "123", strings.NewReader("heyyaaaaa"))

	if err != nil {
		t.Fatalf("received unexpected error putting object: %s", err.Error())
	}
}

func TestGetObjectFromS3(t *testing.T) {
	mocker := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		s3: &MockS3{
			bucket: make(map[string]string),
		},
	}
	// this object DNE
	body, err := mocker.getObjectFromS3("someobject", "123")
	if err == nil || body != nil {
		t.Fatalf("getObjectFromS3 should return no body and an error when the key does not exist. %v, %v", body, err)
	}

	mocker.addObjectToS3("someobject", "123", strings.NewReader("ok"))
	body, err = mocker.getObjectFromS3("someobject", "123")
	if err != nil {
		t.Fatalf("getObjectFromS3 should not return an error when the key exists: %v", err)
	}
}

func TestAddObjectHappy(t *testing.T) {
	mocker := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			bucket: make(map[string]string),
		},
		ddb: &MockDynamo{
			items: []map[string]*dynamodb.AttributeValue{},
		},
	}

	err := mocker.AddObject("happy object", strings.NewReader("happy jar stuff"), true, false, "abc")

	if err != nil {
		t.Fatalf("AddObject should not return error when its on the happy path: %s", err.Error())
	}
	objectBody, err := mocker.getObjectFromS3("happy object", "abc")
	content, err := ioutil.ReadAll(objectBody)
	if string(content) != "happy jar stuff" {
		t.Fatalf("AddObject: had trouble pulling object content after AddObject. Is: %s. Should be: %s", content, "happy jar stuff")
	}
	devVersion, err := mocker.getObjectVersion("happy object", true)
	if devVersion != "abc" {
		t.Fatalf("Addobject: added object should have dev version of abc. Is: %s", devVersion)
	}
	// add it again to trigger not overwriting error
	err = mocker.AddObject("happy object", strings.NewReader("happy jar stuff"), false, false, "abc")
	if err == nil {
		t.Fatalf("AddObject: Should not overwrite objects if they already exist. Should return error, no error was returned")
	}
}

func TestAddObjectFailureScenarios(t *testing.T) {
	headFailure := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			headObjectErr: errors.New("whoa"),
		},
	}
	err := headFailure.AddObject("sad object", strings.NewReader("i am so sad"), false, true, "123")
	if err == nil {
		t.Fatalf("AddObject should return an error when the s3HeadObject call fails")
	}

	s3WriteFailure := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			putObjectErr: errors.New("whoa"),
		},
	}
	err = s3WriteFailure.AddObject("sad object", strings.NewReader("i am so sad"), false, true, "123")
	if err == nil {
		t.Fatalf("AddObject should return an error when the s3GetObject call fails")
	}

	dynamoWriteFailure := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			bucket: make(map[string]string),
		},
		ddb: &MockDynamo{
			putItemErr: []error{errors.New("whoa")},
		},
	}
	err = dynamoWriteFailure.AddObject("sad object", strings.NewReader("i am so sad"), false, true, "123")
	if err == nil {
		t.Fatalf("AddObject should return an error when the ddbPutObject call fails")
	}
}

func TestGetObject(t *testing.T) {
	mocker := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			bucket: make(map[string]string),
		},
		ddb: &MockDynamo{
			items: []map[string]*dynamodb.AttributeValue{},
		},
	}

	mocker.AddObject("happy object", strings.NewReader("party time"), false, false, "123")

	body, err := mocker.GetObject("happy object", "123", false)
	if err != nil {
		t.Fatalf("GetObject returned an error %s", err.Error())
	}
	content, _ := ioutil.ReadAll(body)
	if string(content) != "party time" {
		t.Fatalf("GetObject with specific version did not return correct body. Should be: happy object. Is: %s", content)
	}

	mocker.SetObjectVersion("happy object", "123")
	nextBody, err := mocker.GetObject("happy object", "", false)
	if err != nil {
		t.Fatalf("Error calling GetObject: %s", err.Error())
	}
	content, _ = ioutil.ReadAll(nextBody)
	if string(content) != "party time" {
		t.Fatalf("GetObject with default version did not return correct body. Should be: happy object. Is: %s", string(content))
	}

	failmocker := ObjectController{
		bucket: aws.String("unit test"),
		path:   "dang",
		table:  aws.String("unit test"),
		s3: &MockS3{
			bucket: make(map[string]string),
		},
		ddb: &MockDynamo{
			items:      []map[string]*dynamodb.AttributeValue{},
			getItemErr: []error{errors.New("fart")},
		},
	}
	mocker.AddObject("sad object", strings.NewReader("not party time"), false, true, "123")

	body, err = failmocker.GetObject("sad object", "", false)
	if err == nil {
		t.Fatalf("GetObject should return an error when it cannot look up a object version")
	}
}
