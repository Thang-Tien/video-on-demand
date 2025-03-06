package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	os.Setenv("WorkflowName", "TestWorkflow")
	os.Setenv("Source", "source_bucket")
	os.Setenv("Destination", "destination-bucket")
	os.Setenv("CloudFront", "cloudfront-url")
	os.Setenv("FrameCapture", "true")
	os.Setenv("EnableSns", "true")
	os.Setenv("EnableSqs", "true")
	os.Setenv("EnableMediaPackage", "true")
	os.Setenv("ArchiveSource", "true")
	os.Setenv("JMediaConvert_Template_2160p", "template-2160p")
	os.Setenv("MediaConvert_Template_1080p", "template-1080p")
	os.Setenv("MediaConvert_Template_720p", "template-720p")
	os.Setenv("InputRotate", "DEGREE_0")
	os.Setenv("AcceleratedTranscoding", "DISABLED")

	cases := []struct {
		name          string
		event         InputValidateEvent
		expectedError error
		expectedData  *InputValidateData
	}{
		{
			name: "Valid Video WorkflowTrigger",
			event: InputValidateEvent{
				GUID:            "1234",
				WorkflowTrigger: "Video",
				Records: []events.S3EventRecord{
					{
						S3: events.S3Entity{
							Object: events.S3Object{
								Key: "video+file.mp4",
							},
						},
					},
				},
			},
			expectedError: nil,
			expectedData: &InputValidateData{
				GUID:                   "1234",
				StartTime:              time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
				WorkflowTrigger:        "Video",
				WorkflowStatus:         "Ingest",
				WorkflowName:           "TestWorkflow",
				SrcBucket:              "source_bucket",
				DestBucket:             "destination-bucket",
				CloudFront:             "cloudfront-url",
				FrameCapture:           true,
				ArchiveSource:          "true",
				JobTemplate2160p:       "template-2160p",
				JobTemplate1080p:       "template-1080p",
				JobTemplate720p:        "template-720p",
				InputRotate:            "DEGREE_0",
				AcceleratedTranscoding: "DISABLED",
				EnableSns:              true,
				EnableSqs:              true,
				EnableMediaPackage:     true,
				SrcVideo:               "video file.mp4",
			},
		},
		{
			name: "Invalid WorkflowTrigger",
			event: InputValidateEvent{
				GUID:            "1234",
				WorkflowTrigger: "InvalidTrigger",
			},
			expectedError: fmt.Errorf("input-validate: main.Handler: %w", ErrEventWorkflowTriggerNotDefined),
			expectedData:  nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := Handler(c.event)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				assert.Equal(t, c.expectedData.GUID, data.GUID)
				assert.Equal(t, c.expectedData.WorkflowTrigger, data.WorkflowTrigger)
				assert.Equal(t, c.expectedData.WorkflowStatus, data.WorkflowStatus)
				assert.Equal(t, c.expectedData.WorkflowName, data.WorkflowName)
				assert.Equal(t, c.expectedData.SrcBucket, data.SrcBucket)
				assert.Equal(t, c.expectedData.DestBucket, data.DestBucket)
				assert.Equal(t, c.expectedData.CloudFront, data.CloudFront)
				assert.Equal(t, c.expectedData.FrameCapture, data.FrameCapture)
				assert.Equal(t, c.expectedData.ArchiveSource, data.ArchiveSource)
				assert.Equal(t, c.expectedData.JobTemplate2160p, data.JobTemplate2160p)
				assert.Equal(t, c.expectedData.JobTemplate1080p, data.JobTemplate1080p)
				assert.Equal(t, c.expectedData.JobTemplate720p, data.JobTemplate720p)
				assert.Equal(t, c.expectedData.InputRotate, data.InputRotate)
				assert.Equal(t, c.expectedData.AcceleratedTranscoding, data.AcceleratedTranscoding)
				assert.Equal(t, c.expectedData.EnableSns, data.EnableSns)
				assert.Equal(t, c.expectedData.EnableSqs, data.EnableSqs)
				assert.Equal(t, c.expectedData.EnableMediaPackage, data.EnableMediaPackage)
				assert.Equal(t, c.expectedData.SrcVideo, data.SrcVideo)
			}
		})
	}
}
