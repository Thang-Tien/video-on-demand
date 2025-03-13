package main

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DynamoClientMock struct {
	mock.Mock
}

type S3ClientMock struct {
	mock.Mock
}

func (m *DynamoClientMock) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *S3ClientMock) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsOutput), args.Error(1)
}

func TestOutputValidate(t *testing.T) {
	t.Run("should success on parsing CMAF MSS output", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		cmafMssBytes, _ := json.Marshal(CmafMss)

		event := events.CloudWatchEvent{
			Detail: cmafMssBytes,
		}

		data := &dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				"guid": {
					S: aws.String("guid"),
				},
				"cloudFront": {
					S: aws.String("cloudfront"),
				},
				"destBucket": {
					S: aws.String("vod-destination"),
				},
				"frameCapture": {
					BOOL: aws.Bool(false),
				},
			},
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(data, nil)

		res, err := handler.HandleRequest(event)
		assert.Nil(t, err)
		assert.Equal(t, *res.MssPlaylist, "s3://vod-destination/12345/mss/big_bunny.ism")
		assert.Equal(t, *res.MssUrl, "https://cloudfront/12345/mss/big_bunny.ism")
		assert.Equal(t, *res.CmafDashPlaylist, "s3://vod-destination/12345/cmaf/big_bunny.mpd")
		assert.Equal(t, *res.CmafDashUrl, "https://cloudfront/12345/cmaf/big_bunny.mpd")
	})

	t.Run("should success on parsing CMAF HLS output", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		hlsBytes, _ := json.Marshal(HlsDash)

		event := events.CloudWatchEvent{
			Detail: hlsBytes,
		}

		data := &dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				"guid": {
					S: aws.String("guid"),
				},
				"cloudFront": {
					S: aws.String("cloudfront"),
				},
				"destBucket": {
					S: aws.String("vod-destination"),
				},
				"frameCapture": {
					BOOL: aws.Bool(false),
				},
			},
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(data, nil)

		res, err := handler.HandleRequest(event)
		assert.Nil(t, err)
		assert.Equal(t, *res.HlsPlaylist, "s3://vod-destination/12345/hls/dude.m3u8")
		assert.Equal(t, *res.HlsUrl, "https://cloudfront/12345/hls/dude.m3u8")
		assert.Equal(t, *res.DashPlaylist, "s3://vod-destination/12345/dash/dude.mpd")
		assert.Equal(t, *res.DashUrl, "https://cloudfront/12345/dash/dude.mpd")
	})

	t.Run("should success on parsing MP4 output", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		mp4EventBytes, _ := json.Marshal(Mp4)
		event := events.CloudWatchEvent{
			Detail: mp4EventBytes,
		}
		data := &dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				"guid": {
					S: aws.String("guid"),
				},
				"cloudFront": {
					S: aws.String("cloudfront"),
				},
				"destBucket": {
					S: aws.String("vod-destination"),
				},
				"frameCapture": {
					BOOL: aws.Bool(false),
				},
			},
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(data, nil)
		res, err := handler.HandleRequest(event)
		assert.Nil(t, err)
		assert.Equal(t, *res.Mp4Outputs[0], "s3://vod-destination/12345/mp4/dude_3.0Mbps.mp4")
		assert.Equal(t, *res.Mp4Urls[0], "https://cloudfront/12345/mp4/dude_3.0Mbps.mp4")
	})

	t.Run("should fail when DynamoDB GetItem failed", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		cmafMssBytes, _ := json.Marshal(CmafMss)

		event := events.CloudWatchEvent{
			Detail: cmafMssBytes,
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(nil, assert.AnError)
		_, err := handler.HandleRequest(event)
		assert.Error(t, err, assert.AnError)
	})

	t.Run("should fail when output parse fails", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		errorEventDetail := EventDetail{
			JobId:  "htprrb",
			Status: "COMPLETE",
			UserMetadata: UserMetadata{
				Workflow: "vod10",
				GUID:     "guid",
			},
			OutputGroupDetails: []*OutputGroupDetail{},
		}

		errorEventBytes, _ := json.Marshal(errorEventDetail)

		event := events.CloudWatchEvent{
			Detail: errorEventBytes,
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(&dynamodb.GetItemOutput{}, nil)

		_, err := handler.HandleRequest(event)
		assert.Error(t, err)
		assert.Error(t, err, "output-validate: main.Handler.HandleRequest: no output group details found")
	})

	t.Run("should success when frame capture enabled", func(t *testing.T) {
		dynamoClientMock := new(DynamoClientMock)
		s3ClientMock := new(S3ClientMock)

		handler := Handler{
			DynamoDBClient: dynamoClientMock,
			S3Client:       s3ClientMock,
		}

		mp4EventBytes, _ := json.Marshal(Mp4)
		event := events.CloudWatchEvent{
			Detail: mp4EventBytes,
		}

		data := &dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				"guid": {
					S: aws.String("guid"),
				},
				"cloudFront": {
					S: aws.String("cloudfront"),
				},
				"destBucket": {
					S: aws.String("vod-destination"),
				},
				"frameCapture": {
					BOOL: aws.Bool(true),
				},
			},
		}

		imageData := &s3.ListObjectsOutput{
			Contents: []*s3.Object{
				{
					Key: aws.String("12345/thumbnails/dude3.000.jpg"),
				},
			},
		}

		dynamoClientMock.On("GetItem", mock.Anything).Return(data, nil)
		s3ClientMock.On("ListObjects", mock.Anything).Return(imageData, nil)

		res, err := handler.HandleRequest(event)
		assert.Nil(t, err)
		assert.Equal(t, *res.ThumbNails[0], "s3://vod-destination/12345/thumbnails/dude3.000.jpg")
		assert.Equal(t, *res.ThumbNailsUrls[0], "https://cloudfront/12345/thumbnails/dude3.000.jpg")
	})

}
