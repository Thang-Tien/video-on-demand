package main

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type S3PresignClientMock struct {
	mock.Mock
}

func (m *S3PresignClientMock) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func TestGetPresignedURL(t *testing.T) {
	os.Setenv("Source", "test-bucket")

	t.Run("should success on presigning URL", func(t *testing.T) {
		s3PresignClient := &S3PresignClientMock{}
		s3PresignClient.On("PresignPutObject", mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
			URL: "https://example.com",
		}, nil)
		handler := GetPresignedURLHandler{
			s3PresignClient: s3PresignClient,
		}

		request := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"fileName": "test.mp4",
			},
			RequestContext: events.APIGatewayProxyRequestContext{
				Authorizer: map[string]interface{}{
					"principalId": "vodapi::User::\"ap-southeast-2_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613\"",
				},
			},
		}

		_, err := handler.HandleRequest(request)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assert.NoError(t, err)
	})

	t.Run("should fails when authenticate fails", func(t *testing.T) {
		s3PresignClient := &S3PresignClientMock{}
		s3PresignClient.On("PresignPutObject", mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
			URL: "https://example.com",
		}, nil)
		handler := GetPresignedURLHandler{
			s3PresignClient: s3PresignClient,
		}

		request := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"fileName": "test.mp4",
			},
			RequestContext: events.APIGatewayProxyRequestContext{
				Authorizer: map[string]interface{}{
					"principalId": "vodapi::User::\"ap-southeast-2_opagHcslJ|\"",
				},
			},
		}

		res, err := handler.HandleRequest(request)
		assert.NoError(t, err)
		assert.Equal(t, events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Could not extract user ID from token",
		}, res)
	})

	t.Run("should fails when generating presigned url fails", func(t *testing.T) {
		s3PresignClient := &S3PresignClientMock{}
		s3PresignClient.On("PresignPutObject", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		handler := GetPresignedURLHandler{
			s3PresignClient: s3PresignClient,
		}

		request := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"fileName": "test.mp4",
			},
			RequestContext: events.APIGatewayProxyRequestContext{
				Authorizer: map[string]interface{}{
					"principalId": "vodapi::User::\"ap-southeast-2_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613\"",
				},
			},
		}

		res, err := handler.HandleRequest(request)
		assert.NoError(t, err)
		assert.Equal(t, events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to generate upload URL: " + assert.AnError.Error(),
		}, res)
	})
}
