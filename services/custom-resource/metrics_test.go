package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"
)

type MetricClientMock struct {
	mock.Mock
	resp *http.Response
	err  error
}

func (m *MetricClientMock) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return m.resp, m.err
}

func TestMetrics(t *testing.T) {
	statusTests := []struct {
		name             string
		config           map[string]interface{}
		metricClient     MetricClient
		expectedResponse *string
		expectedError    error
	}{
		{
			name: "should return status code \"200\" on successful metrics post",
			config: map[string]interface{}{
				"SolutionId":   "solution",
				"UUID":         "uuid",
				"ServiceToken": "lambda-arn",
				"Resource":     "AnonymizedMetric",
			},
			metricClient: &MetricClientMock{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(nil),
				},
				err: nil,
			},
			expectedResponse: aws.String("200"),
			expectedError:    nil,
		},
		{
			name: "should return \" Network Error\" on connection tiomeout",
			config: map[string]interface{}{
				"SolutionId":   "solution",
				"UUID":         "uuid",
				"ServiceToken": "lambda-arn",
				"Resource":     "AnonymizedMetric",
			},
			metricClient: &MetricClientMock{
				resp: &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(nil),
				},
				err: nil,
			},
			expectedResponse: aws.String("404"),
			expectedError:    nil,
		},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			MetricClientMock := tt.metricClient.(*MetricClientMock)
			metricCustomResource := MetricCustomResource{
				MetricClient: tt.metricClient,
			}

			MetricClientMock.On("Post", mock.Anything, mock.Anything, mock.Anything).Return(MetricClientMock.resp, MetricClientMock.err)

			response, err := metricCustomResource.Send(tt.config)

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
