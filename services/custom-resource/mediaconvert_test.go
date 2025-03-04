package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/stretchr/testify/mock"
)

var (
	TestCreateJobTemplateOutput = &mediaconvert.CreateJobTemplateOutput{
		JobTemplate: &mediaconvert.JobTemplate{
			Name: aws.String("name"),
		},
	}
	TestDescribeEndpointsOutput = &mediaconvert.DescribeEndpointsOutput{
		Endpoints: []*mediaconvert.Endpoint{
			{
				Url: aws.String("https://test.com"),
			},
		},
	}
	TestConfig = map[string]interface{}{
		"StackName":          "test",
		"EndPoint":           "https://test.com",
		"EnableMediaPackage": "false",
	}
)

type MediaConvertClientMock struct {
	mock.Mock
}

func (m *MediaConvertClientMock) DescribeEndpoints(input *mediaconvert.DescribeEndpointsInput) (*mediaconvert.DescribeEndpointsOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediaconvert.DescribeEndpointsOutput), args.Error(1)
}

func (m *MediaConvertClientMock) CreateJobTemplate(input *mediaconvert.CreateJobTemplateInput) (*mediaconvert.CreateJobTemplateOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediaconvert.CreateJobTemplateOutput), args.Error(1)
}

func TestMediaConvert(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("should success on create templates", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			mediaConvertClientMock.On("CreateJobTemplate", mock.Anything).Return(TestCreateJobTemplateOutput, nil)

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
			}

			err := mediaConvertCustomResource.CreateTemplates(TestConfig)
			if err != nil {
				t.Errorf("expect no error, got %v", err)
			}
		})

		t.Run("should fail when CreateJobTemplate fails", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			mediaConvertClientMock.On("CreateJobTemplate", mock.Anything).Return(nil, errors.New("error"))

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
			}

			err := mediaConvertCustomResource.CreateTemplates(TestConfig)
			if err == nil {
				t.Error("expect error, got nil")
			}
			if err.Error() != "MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: error" {
				t.Errorf("expect error %s, got %s", "MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: error", err.Error())
			}
		})
	})

	t.Run("Describe", func(t *testing.T) {
		t.Run("should success on describe endpoints", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			mediaConvertClientMock.On("DescribeEndpoints", mock.Anything).Return(TestDescribeEndpointsOutput, nil)

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
			}

			res, err := mediaConvertCustomResource.GetEndpoint()
			if err != nil {
				t.Errorf("expect no error, got %v", err)
			}
			if *res != "https://test.com" {
				t.Errorf("expect %s, got %s", "https://test.com", *res)
			}

		})

		t.Run("should fail when DescribeEndpoints fails", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			mediaConvertClientMock.On("DescribeEndpoints", mock.Anything).Return(nil, errors.New("error"))

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
			}

			_, err := mediaConvertCustomResource.GetEndpoint()
			if err == nil {
				t.Error("expect error, got nil")
			}
			if err.Error() != "MediaConvertCustomResource.GetEndpoint: DescribeEndpoints: error" {
				t.Errorf("expect error %s, got %s", "MediaConvertCustomResource.GetEndpoint: DescribeEndpoints: error", err.Error())
			}
		})
	})
}
