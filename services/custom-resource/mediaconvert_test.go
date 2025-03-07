package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/mock"
)

var (
	TestCreateJobTemplateOutput = &mediaconvert.CreateJobTemplateOutput{
		JobTemplate: &mediaconvert.JobTemplate{
			Name: aws.String("name"),
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

type MediaConvertS3ClientMock struct {
	mock.Mock
}

func (m *MediaConvertClientMock) CreateJobTemplate(input *mediaconvert.CreateJobTemplateInput) (*mediaconvert.CreateJobTemplateOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediaconvert.CreateJobTemplateOutput), args.Error(1)
}

func (m *MediaConvertS3ClientMock) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	
	// Handle function return values
	if fn, ok := args.Get(0).(func(*s3.GetObjectInput) *s3.GetObjectOutput); ok {
		return fn(input), args.Error(1)
	}
	
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func TestMediaConvert(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("should success on create templates", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			MediaConvertS3ClientMock := new(MediaConvertS3ClientMock)

			mediaConvertClientMock.On("CreateJobTemplate", mock.Anything).Return(TestCreateJobTemplateOutput, nil)
			MediaConvertS3ClientMock.On("GetObject", mock.Anything).Return(func(input *s3.GetObjectInput) *s3.GetObjectOutput {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader([]byte(`{"Name": "name", "Category": "category", "Description": "description"}`))),
				}
			}, nil)

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
				S3Client:           MediaConvertS3ClientMock,
			}

			err := mediaConvertCustomResource.CreateTemplates(TestConfig)
			if err != nil {
				t.Errorf("expect no error, got %v", err)
			}
		})

		t.Run("should fail when CreateJobTemplate fails", func(t *testing.T) {
			mediaConvertClientMock := new(MediaConvertClientMock)
			MediaConvertS3ClientMock := new(MediaConvertS3ClientMock)

			mediaConvertClientMock.On("CreateJobTemplate", mock.Anything).Return(nil, errors.New("error"))
			MediaConvertS3ClientMock.On("GetObject", mock.Anything).Return(func(input *s3.GetObjectInput) *s3.GetObjectOutput {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader([]byte(`{"Name": "name", "Category": "category", "Description": "description"}`))),
				}
			}, nil)

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
				S3Client:           MediaConvertS3ClientMock,
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

			mediaConvertCustomResource := &MediaConvertCustomResource{
				MediaConvertClient: mediaConvertClientMock,
			}

			res, err := mediaConvertCustomResource.GetEndpoint()
			if err != nil {
				t.Errorf("expect no error, got %v", err)
			}
			if *res != "https://ap-southeast-2.mediaconvert.amazonaws.com" {
				t.Errorf("expect %s, got %s", "https://ap-southeast-2.mediaconvert.amazonaws.com", *res)
			}

		})

	})
}
