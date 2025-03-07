package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/mock"
)

type S3ClientMock struct {
	mock.Mock
	Output      *s3.PutBucketNotificationConfigurationOutput
	ErrorOutput error
}

func (m *S3ClientMock) PutBucketNotificationConfiguration(input *s3.PutBucketNotificationConfigurationInput) (*s3.PutBucketNotificationConfigurationOutput, error) {
	return m.Output, m.ErrorOutput
}

func TestS3CustomResource(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]interface{}
		s3Client         S3Client
		expectedResponse *string
		expectedError    error
	}{
		{
			name: "should success on VideoFile trigger",
			config: map[string]interface{}{
				"WorkflowTrigger": "VideoFile",
				"IngestArn":       "arn",
				"Source":          "srcBucket",
			},
			s3Client: &S3ClientMock{
				Output:      &s3.PutBucketNotificationConfigurationOutput{},
				ErrorOutput: nil,
			},
			expectedResponse: aws.String("success"),
			expectedError:    nil,
		},
		{
			name: "should return error when PutBucketNotificationConfiguration fails",
			config: map[string]interface{}{
				"WorkflowTrigger": "VideoFile",
				"IngestArn":       "arn",
				"Source":          "srcBucket",
			},
			s3Client: &S3ClientMock{
				Output:      nil,
				ErrorOutput: errors.New("s3 error"),
			},
			expectedResponse: nil,
			expectedError:    errors.New("S3CustomResource.PutNotification: s3 error"),
		},
		{
			name:             "should return error when trigger is unknown",
			s3Client:         &S3ClientMock{},
			expectedResponse: nil,
			expectedError:    fmt.Errorf("S3CustomResource.PutNotification: %w", ErrInvalidWorkflowTrigger),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3ClientMock := tt.s3Client.(*S3ClientMock)
			s3CustomResource := S3CustomResource{
				S3Client: s3ClientMock,
			}

			// Set expectations on mock if tt.name is not about unknown triggers
			if tt.name != "should return error when trigger is unknown" {
				s3ClientMock.On("PutBucketNotificationConfiguration", mock.Anything).Return(s3ClientMock.Output, s3ClientMock.ErrorOutput)
			}

			response, err := s3CustomResource.PutNotification(tt.config)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectedResponse == nil && response != nil {
				t.Errorf("Expected no response, got %v", *response)
			} else if tt.expectedResponse != nil && (response == nil || *response != *tt.expectedResponse) {
				t.Errorf("Expected response %v, got %v", *tt.expectedResponse, response)
			}

		})
	}
}
