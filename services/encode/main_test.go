package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MediaConvertClientMock struct {
	mock.Mock
}

func (m *MediaConvertClientMock) CreateJob(input *mediaconvert.CreateJobInput) (*mediaconvert.CreateJobOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediaconvert.CreateJobOutput), args.Error(1)
}

func (m *MediaConvertClientMock) GetJobTemplate(input *mediaconvert.GetJobTemplateInput) (*mediaconvert.GetJobTemplateOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediaconvert.GetJobTemplateOutput), args.Error(1)
}

func TestEncode(t *testing.T) {
	os.Setenv("MediaConvertRole", "Role")
	os.Setenv("Workflow", "vod")

	t.Run("should success when FrameCapture is disabled", func(t *testing.T) {
		template := mediaconvert.GetJobTemplateOutput{
			JobTemplate: &mediaconvert.JobTemplate{
				Settings: &mediaconvert.JobTemplateSettings{
					OutputGroups: []*mediaconvert.OutputGroup{
						{
							OutputGroupSettings: &mediaconvert.OutputGroupSettings{
								Type: aws.String("HLS_GROUP_SETTINGS"),
							},
							Name: aws.String("test-output-group"),
						},
					},
				},
			},
		}

		data := mediaconvert.CreateJobOutput{
			Job: &mediaconvert.Job{
				Id: aws.String("12345"),
			},
		}

		event := EncodeInput{
			GUID:                   "GUID",
			JobTemplate:            "JobTemplate",
			SrcVideo:               "video.mp4",
			SrcBucket:              "src",
			DestBucket:             "dest",
			AcceleratedTranscoding: "PREFERRED",
		}

		mediaConvertClientMock := new(MediaConvertClientMock)
		handler := Handler{
			MediaConvertClient: mediaConvertClientMock,
		}

		mediaConvertClientMock.On("GetJobTemplate", mock.Anything).Return(&template, nil)
		mediaConvertClientMock.On("CreateJob", mock.Anything).Return(&data, nil)

		res, err := handler.HandleRequest(event)
		if err != nil {
			t.Errorf("Expect no error, but got %v", err)
		}
		assert.Equal(t, "12345", res.EncodeJobId)
		assert.Equal(t, "HLS_GROUP_SETTINGS", *res.EncodingJob.Settings.OutputGroups[0].OutputGroupSettings.Type)
	})
	t.Run("should succeed when FrameCapture is enabled", func(t *testing.T) {
		template := mediaconvert.GetJobTemplateOutput{
			JobTemplate: &mediaconvert.JobTemplate{
				Settings: &mediaconvert.JobTemplateSettings{
					OutputGroups: []*mediaconvert.OutputGroup{
						{
							OutputGroupSettings: &mediaconvert.OutputGroupSettings{
								Type: aws.String("HLS_GROUP_SETTINGS"),
							},
							Name: aws.String("test-output-group"),
						},
					},
				},
			},
		}

		data := mediaconvert.CreateJobOutput{
			Job: &mediaconvert.Job{
				Id: aws.String("12345"),
			},
		}
	
		withFrame := EncodeInput{
			GUID:                   "GUID",
			JobTemplate:            "JobTemplate",
			SrcVideo:               "video.mp4",
			SrcBucket:              "src",
			DestBucket:             "dest",
			FrameCapture:           true,
			AcceleratedTranscoding: "ENABLED",
		}

		mediaConvertClientMock := new(MediaConvertClientMock)
		handler := Handler{
			MediaConvertClient: mediaConvertClientMock,
		}

		mediaConvertClientMock.On("GetJobTemplate", mock.Anything).Return(&template, nil)
		mediaConvertClientMock.On("CreateJob", mock.Anything).Return(&data, nil)

		res, err := handler.HandleRequest(withFrame)
		if err != nil {
			t.Errorf("Expect no error, but got %v", err)
		}
		assert.Equal(t, "12345", res.EncodeJobId)
		assert.Equal(t, "Frame Capture", *res.EncodingJob.Settings.OutputGroups[1].CustomName)
	})

	t.Run("should apply custom settings when template is custom", func(t *testing.T) {
		event := EncodeInput{
			GUID:                   "12345678",
			JobTemplate:            "custom-template",
			SrcVideo:               "video.mp4",
			SrcBucket:              "src",
			DestBucket:             "dest",
			IsCustomTemplate:       true,
			AcceleratedTranscoding: "DISABLED",
		}

		customTemplate := mediaconvert.GetJobTemplateOutput{
			JobTemplate: &mediaconvert.JobTemplate{
				Name: aws.String("custom-template"),
				Type: aws.String("CUSTOM"),
				Settings: &mediaconvert.JobTemplateSettings{
					OutputGroups: []*mediaconvert.OutputGroup{
						{
							OutputGroupSettings: &mediaconvert.OutputGroupSettings{
								Type: aws.String("HLS_GROUP_SETTINGS"),
								HlsGroupSettings: &mediaconvert.HlsGroupSettings{
									SegmentLength:    aws.Int64(10),
									MinSegmentLength: aws.Int64(2),
								},
							},
							Name: aws.String("custom-output-group"),
						},
					},
				},
			},
		}

		newJob := mediaconvert.CreateJobOutput{
			Job: &mediaconvert.Job{
				Id: aws.String("12345678"),
			},
		}
		mediaConvertClientMock := new(MediaConvertClientMock)
		handler := Handler{
			MediaConvertClient: mediaConvertClientMock,
		}

		mediaConvertClientMock.On("GetJobTemplate", mock.Anything).Return(&customTemplate, nil)
		mediaConvertClientMock.On("CreateJob", mock.Anything).Return(&newJob, nil)

		res, err := handler.HandleRequest(event)
		if err != nil {
			t.Errorf("Expect no error, but got %v", err)
		}

		output := res.EncodingJob.Settings.OutputGroups[0]
		settings := output.OutputGroupSettings.HlsGroupSettings

		assert.NotNil(t, settings, "HLS group settings should not be nil")
		fmt.Println(*settings.SegmentLength, *settings.MinSegmentLength)
		assert.Equal(t, int64(10), *settings.SegmentLength)
		assert.Equal(t, int64(2), *settings.MinSegmentLength)
	})

	t.Run("should fail when GetJobTemplate failed", func(t *testing.T) {
		event := EncodeInput{
			GUID:                   "GUID",
			JobTemplate:            "JobTemplate",
			SrcVideo:               "video.mp4",
			SrcBucket:              "src",
			DestBucket:             "dest",
			AcceleratedTranscoding: "PREFERRED",
		}

		mediaConvertClientMock := new(MediaConvertClientMock)
		handler := Handler{
			MediaConvertClient: mediaConvertClientMock,
		}
		mediaConvertClientMock.On("GetJobTemplate", mock.Anything).Return(nil, assert.AnError)

		_, err := handler.HandleRequest(event)
		assert.Error(t, err)
	})

	t.Run("should fail when CreateJob failed", func(t *testing.T) {
		template := mediaconvert.GetJobTemplateOutput{
			JobTemplate: &mediaconvert.JobTemplate{
				Settings: &mediaconvert.JobTemplateSettings{
					OutputGroups: []*mediaconvert.OutputGroup{
						{
							OutputGroupSettings: &mediaconvert.OutputGroupSettings{
								Type: aws.String("HLS_GROUP_SETTINGS"),
							},
							Name: aws.String("test-output-group"),
						},
					},
				},
			},
		}

		event := EncodeInput{
			GUID:                   "GUID",
			JobTemplate:            "JobTemplate",
			SrcVideo:               "video.mp4",
			SrcBucket:              "src",
			DestBucket:             "dest",
			AcceleratedTranscoding: "PREFERRED",
		}

		mediaConvertClientMock := new(MediaConvertClientMock)
		handler := Handler{
			MediaConvertClient: mediaConvertClientMock,
		}

		mediaConvertClientMock.On("GetJobTemplate", mock.Anything).Return(&template, nil)
		mediaConvertClientMock.On("CreateJob", mock.Anything).Return(nil, assert.AnError)

		_, err := handler.HandleRequest(event)
		assert.Error(t, err)
	})
}
