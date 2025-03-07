package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DynamoDBClientMock struct {
	mock.Mock
}

func (m *DynamoDBClientMock) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func TestProfiler(t *testing.T) {
	t.Run("should success on profile set", func(t *testing.T) {
		dynamoDBClientMock := new(DynamoDBClientMock)
		dynamoDBClientMock.On("GetItem", mock.Anything).Return(&dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				"guid": {
					S: aws.String("123e4567-e89b-12d3-a456-426614174000"),
				},
				"srcMediainfo": {
					S: aws.String("{\n  \"filename\": \"clang.mp4\",\n  \"container\": {\n    \"format\": \"MPEG-4\",\n    \"fileSize\": 1540047,\n    \"duration\": 28.189,\n    \"totalBitrate\": 437063\n  },\n  \"video\": [\n    {\n      \"codec\": \"vp09\",\n      \"bitrate\": 305827,\n      \"duration\": 28.09,\n      \"frameCount\": 842,\n      \"width\": 854,\n      \"height\": 480,\n      \"framerate\": 29.97,\n      \"aspectRatio\": \"1.779\",\n      \"colorSpace\": \"YUV None\"\n    }\n  ],\n  \"audio\": [\n    {\n      \"codec\": \"AAC\",\n      \"bitrate\": 128000,\n      \"duration\": 28.189,\n      \"frameCount\": 1214,\n      \"bitrateMode\": \"CBR\",\n      \"channels\": 2,\n      \"samplingRate\": 44100,\n      \"samplePerFrame\": 1024\n    }\n  ]\n}"),
				},
				"jobTemplate_2160p": {
					S: aws.String("tmpl1"),
				},
				"jobTemplate_1080p": {
					S: aws.String("tmpl2"),
				},
				"jobTemplate_720p": {
					S: aws.String("tmpl3"),
				},
				"frameCapture": {
					BOOL: aws.Bool(true),
				},
			},
		}, nil)

		handler := &Handler{
			DynamoDBClient: dynamoDBClientMock,
		}

		output, err := handler.HandleRequest(ProfilerInput{
			GUID: "123e4567-e89b-12d3-a456-426614174000",
		})

		assert.Nil(t, err)
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", output.GUID)
		assert.Equal(t, "tmpl1", output.JobTemplate2160p)
		assert.Equal(t, "tmpl2", output.JobTemplate1080p)
		assert.Equal(t, "tmpl3", output.JobTemplate720p)
		assert.Equal(t, true, output.FrameCapture)
	})

	t.Run("should retuirn error when db get fails", func(t *testing.T) {
		dynamoDBClientMock := new(DynamoDBClientMock)
		dynamoDBClientMock.On("GetItem", mock.Anything).Return(nil, assert.AnError)

		handler := &Handler{
			DynamoDBClient: dynamoDBClientMock,
		}

		output, err := handler.HandleRequest(ProfilerInput{
			GUID: "123e4567-e89b-12d3-a456-426614174000",
		})

		assert.NotNil(t, err)
		assert.Nil(t, output)
	})
}
