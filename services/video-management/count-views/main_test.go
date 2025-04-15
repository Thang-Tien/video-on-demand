package main

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DynamoDBClientMock struct {
	mock.Mock
}

type S3ClientMock struct {
	mock.Mock
}

func (m *DynamoDBClientMock) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}
func (m *S3ClientMock) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func TestProcessLogFile(t *testing.T) {
	filePath := "./test-log-files/E2ACH0FFGSSCH8.2025-04-15-01.321b0a98.gz"

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	defer file.Close()

	handler := &Handler{}

	views, err := handler.processLogFile(file)
	assert.NoError(t, err, "expected no error while processing log file")
	assert.Equal(t, 1, views["350337bf-db60-48d8-b04c-4b1b2c43fb6f"], "expected view count to be 1 for E2ACH0350337bf-db60-48d8-b04c-4b1b2c43fb6fFFGSSCH8")

}

func TestCountView(t *testing.T) {
	t.Run("should success counting view", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		s3Client := &S3ClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
			S3Client:       s3Client,
		}

		event := events.S3Event{
			Records: []events.S3EventRecord{
				{
					S3: events.S3Entity{
						Bucket: events.S3Bucket{
							Name: "test-bucket",
						},
						Object: events.S3Object{
							Key: "test-key.gz",
						},
					},
				},
			},
		}

		filePath := "./test-log-files/E2ACH0FFGSSCH8.2025-04-15-01.321b0a98.gz"

		file, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("failed to open log file: %v", err)
		}
		defer file.Close()

		dynamoDBClient.On("UpdateItem", mock.Anything, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)
		s3Client.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
			Body: file,
		}, nil)

		err = handler.HandleRequest(context.Background(), event)
		assert.NoError(t, err, "expected no error while handling request")
	})

	t.Run("should skip error when GetObject fails", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		s3Client := &S3ClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
			S3Client:       s3Client,
		}

		event := events.S3Event{
			Records: []events.S3EventRecord{
				{
					S3: events.S3Entity{
						Bucket: events.S3Bucket{
							Name: "test-bucket",
						},
						Object: events.S3Object{
							Key: "test-key.gz",
						},
					},
				},
			},
		}

		s3Client.On("GetObject", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		err := handler.HandleRequest(context.Background(), event)
		assert.NoError(t, err, "expected no error while handling request")
	})

	t.Run("should skip error when UpdateItem fails", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		s3Client := &S3ClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
			S3Client:       s3Client,
		}

		event := events.S3Event{
			Records: []events.S3EventRecord{
				{
					S3: events.S3Entity{
						Bucket: events.S3Bucket{
							Name: "test-bucket",
						},
						Object: events.S3Object{
							Key: "test-key.gz",
						},
					},
				},
			},
		}

		filePath := "./test-log-files/E2ACH0FFGSSCH8.2025-04-15-01.321b0a98.gz"

		file, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("failed to open log file: %v", err)
		}
		defer file.Close()

		dynamoDBClient.On("UpdateItem", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		s3Client.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
			Body: file,
		}, nil)

		err = handler.HandleRequest(context.Background(), event)
		assert.NoError(t, err, "expected no error while handling request")
	})
}
