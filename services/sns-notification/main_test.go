package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSnsClient struct {
	mock.Mock
}

func (m *mockSnsClient) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	m.Called(input)
	return &sns.PublishOutput{}, nil
}

func TestHandleRequest(t *testing.T) {
	mockSns := new(mockSnsClient)
	handler := Handler{
		snsClient: mockSns,
	}

	event := SNSNotificationEvent{
		GUID:                   "597c449e-6d32-4e88-a2b4-c956f85a3d51",
		StartTime:              "2025-02-23T10:04:34.556Z",
		WorkflowTrigger:        "Video",
		WorkflowStatus:         "Ingest",
		WorkflowName:           "video-on-demand-on-aws",
		SrcBucket:              "video-on-demand-on-aws-source71e471f1-hnsa1xte0xkc",
		DestBucket:             "video-on-demand-on-aws-destination920a3c57-anvrpncde7i0",
		CloudFront:             "d2xdx30mim9k2e.cloudfront.net",
		FrameCapture:           true,
		ArchiveSource:          "GLACIER",
		JobTemplate2160p:       "video-on-demand-on-aws_Ott_2160p_Avc_Aac_16x9_mvod_no_preset",
		JobTemplate1080p:       "video-on-demand-on-aws_Ott_1080p_Avc_Aac_16x9_mvod_no_preset",
		JobTemplate720p:        "video-on-demand-on-aws_Ott_720p_Avc_Aac_16x9_mvod_no_preset",
		InputRotate:            "DEGREE_0",
		AcceleratedTranscoding: "PREFERRED",
		EnableSns:              true,
		EnableSqs:              true,
		SrcVideo:               "clang.mp4",
		EnableMediaPackage:     true,
		SrcMediainfo: `{
			"filename": "clang.mp4",
			"container": {
				"format": "MPEG-4",
				"fileSize": 1540047,
				"duration": 28.189,
				"totalBitrate": 437063
			},
			"video": [
				{
					"codec": "vp09",
					"bitrate": 305827,
					"duration": 28.09,
					"frameCount": 842,
					"width": 854,
					"height": 480,
					"framerate": 29.97,
					"aspectRatio": "1.779",
					"colorSpace": "YUV None"
				}
			],
			"audio": [
				{
					"codec": "AAC",
					"bitrate": 128000,
					"duration": 28.189,
					"frameCount": 1214,
					"bitrateMode": "CBR",
					"channels": 2,
					"samplingRate": 44100,
					"samplePerFrame": 1024
				}
			]
		}`,
	}

	output := SNSNotificationOutput{
		GUID:                   "597c449e-6d32-4e88-a2b4-c956f85a3d51",
		StartTime:              "2025-02-23T10:04:34.556Z",
		WorkflowTrigger:        "Video",
		WorkflowStatus:         "Ingest",
		WorkflowName:           "video-on-demand-on-aws",
		SrcBucket:              "video-on-demand-on-aws-source71e471f1-hnsa1xte0xkc",
		DestBucket:             "video-on-demand-on-aws-destination920a3c57-anvrpncde7i0",
		CloudFront:             "d2xdx30mim9k2e.cloudfront.net",
		FrameCapture:           true,
		ArchiveSource:          "GLACIER",
		JobTemplate2160p:       "video-on-demand-on-aws_Ott_2160p_Avc_Aac_16x9_mvod_no_preset",
		JobTemplate1080p:       "video-on-demand-on-aws_Ott_1080p_Avc_Aac_16x9_mvod_no_preset",
		JobTemplate720p:        "video-on-demand-on-aws_Ott_720p_Avc_Aac_16x9_mvod_no_preset",
		InputRotate:            "DEGREE_0",
		AcceleratedTranscoding: "PREFERRED",
		EnableSns:              true,
		EnableSqs:              true,
		SrcVideo:               "clang.mp4",
		EnableMediaPackage:     true,
		SrcMediainfo: `{
			"filename": "clang.mp4",
			"container": {
				"format": "MPEG-4",
				"fileSize": 1540047,
				"duration": 28.189,
				"totalBitrate": 437063
			},
			"video": [
				{
					"codec": "vp09",
					"bitrate": 305827,
					"duration": 28.09,
					"frameCount": 842,
					"width": 854,
					"height": 480,
					"framerate": 29.97,
					"aspectRatio": "1.779",
					"colorSpace": "YUV None"
				}
			],
			"audio": [
				{
					"codec": "AAC",
					"bitrate": 128000,
					"duration": 28.189,
					"frameCount": 1214,
					"bitrateMode": "CBR",
					"channels": 2,
					"samplingRate": 44100,
					"samplePerFrame": 1024
				}
			]
		}`,
	}

	mockSns.On("Publish", mock.Anything).Return(&sns.PublishOutput{}, nil)

	result, err := handler.HandleRequest(event)
	assert.NoError(t, err)
	assert.Equal(t, &output, result)
}
