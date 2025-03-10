package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type StepFunctionClientMock struct {
	mock.Mock
}

func (m *StepFunctionClientMock) StartExecution(input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return &sfn.StartExecutionOutput{}, nil
}

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name             string
		event            map[string]interface{}
		expectedResponse *string
		expectedError    error
	}{
		{
			name: "should return \"success\" on Ingest Execute success",
			event: map[string]interface{}{
				"Records": []events.S3EventRecord{
					{
						S3: events.S3Entity{
							Object: events.S3Object{
								Key: "test.mp4",
							},
						},
					},
				},
			},
			expectedResponse: aws.String("success"),
			expectedError:    nil,
		},
		{
			name: "should return \"success\" on Process Execute success",
			event: map[string]interface{}{
				"guid": aws.String("123e4567-e89b-12d3-a456-426614174000"),
			},
			expectedResponse: aws.String("success"),
			expectedError:    nil,
		},
		{
			name: "should return \"success\" on Publish Execute success",
			event: map[string]interface{}{
				"detail": map[string]interface{}{
					"status": "COMPLETE",
					"jobId":  "1740305088427-714h8k",
				},
				"source":      "aws.mediaconvert",
				"detail-type": "MediaConvert Job State Change",
			},
			expectedResponse: aws.String("success"),
			expectedError:    nil,
		},
		{
			name:             "should return error on invalid event object",
			event:            map[string]interface{}{},
			expectedResponse: nil,
			expectedError:    ErrInvalidEventObject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStepFunctionClient := new(StepFunctionClientMock)
			handler := Handler{
				StepFunctionClient: mockStepFunctionClient,
			}

			mockStepFunctionClient.On("StartExecution", &sfn.StartExecutionInput{}).Return(&sfn.StartExecutionOutput{}, nil)

			response, err := handler.HandleRequest(tt.event)
			assert.Equal(t, tt.expectedResponse, response)
			assert.Equal(t, tt.expectedError, err)
		})
	}

}
