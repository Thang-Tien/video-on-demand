package main

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"
)

var (
	TestEvent = cfn.Event{
		RequestType:       "Create",
		ResponseURL:       "https://cloudformation",
		StackID:           "arn:aws:cloudformation",
		RequestID:         "63e8ffa2-3059-4607-a450-119d473c73bc",
		LogicalResourceID: "Uuid",
		ResourceType:      "Custom::UUID",
		ResourceProperties: map[string]interface{}{
			"ServiceToken": "arn:aws:lambda",
			"Resource":     "abc",
		},
	}

	TestResponseStatus = "SUCCESS"
	TestResponseData   = CustomResourceResponse{
		GroupId:         aws.String("grp-12345abcdef"),
		GroupDomainName: aws.String("example-domain"),
		EndpointUrl:     aws.String("https://api.example.com/endpoint"),
		UUID:            aws.String("550e8400-e29b-41d4-a716-446655440000"),
	}
)

type CfnClientMock struct {
	mock.Mock
}

func (m *CfnClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*http.Response), args.Error(1)
}

func TestCfn(t *testing.T) {
	t.Run("should return \"200 OK\" on a send cfn response success", func(t *testing.T) {
		cfnClientMock := new(CfnClientMock)
		cfnClientMock.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(nil),
		}, nil)

		cfnCustomResource := &CfnCustomResource{
			CfnClient: cfnClientMock,
		}
		res, err := cfnCustomResource.Send(TestEvent, TestResponseStatus, TestResponseData)
		if err != nil {
			t.Errorf("expect no error, but got %v", err)
		}
		if *res != 200 {
			t.Errorf("expect status code 200, got %v", *res)
		}
	})

	t.Run("should return error on connection timeout", func(t *testing.T) {
		cfnClientMock := new(CfnClientMock)
		cfnClientMock.On("Do", mock.Anything).Return(nil, errors.New("connection timeout"))

		cfnCustomResource := &CfnCustomResource{
			CfnClient: cfnClientMock,
		}

		_, err := cfnCustomResource.Send(TestEvent, TestResponseStatus, TestResponseData)
		if err == nil {
			t.Errorf("expect error, but got nil")
		}
		if err.Error() != "connection timeout" {
			t.Errorf("expect error message \"connection timeout\", got %v", err.Error())
		}

	})
}
