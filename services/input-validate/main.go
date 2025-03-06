package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	ErrEventWorkflowTriggerNotDefined = errors.New("event.workflowTrigger is not defined")
)

// InputValidateEvent represents the input event structure
type InputValidateEvent struct {
	Records         []events.S3EventRecord `json:"Records"`
	GUID            string                 `json:"guid"`
	WorkflowTrigger string                 `json:"workflowTrigger"`
}

type InputValidateData struct {
	GUID                   string `json:"guid"`
	StartTime              string `json:"startTime"`
	WorkflowTrigger        string `json:"workflowTrigger"`
	WorkflowStatus         string `json:"workflowStatus"`
	WorkflowName           string `json:"workflowName"`
	SrcBucket              string `json:"srcBucket"`
	DestBucket             string `json:"destBucket"`
	CloudFront             string `json:"cloudFront"`
	FrameCapture           bool   `json:"frameCapture"`
	ArchiveSource          string `json:"archiveSource"`
	JobTemplate2160p       string `json:"jobTemplate_2160p"`
	JobTemplate1080p       string `json:"jobTemplate_1080p"`
	JobTemplate720p        string `json:"jobTemplate_720p"`
	InputRotate            string `json:"inputRotate"`
	AcceleratedTranscoding string `json:"acceleratedTranscoding"`
	EnableSns              bool   `json:"enableSns"`
	EnableSqs              bool   `json:"enableSqs"`
	SrcVideo               string `json:"srcVideo"`
	EnableMediaPackage     bool   `json:"enableMediaPackage"`
}

func Handler(event InputValidateEvent) (*InputValidateData, error) {
	log.Printf("newest version")

	eventJson, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return nil, fmt.Errorf("input-validate: main.Handler: MarshalIndent: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJson)

	frameCapture := os.Getenv("FrameCapture") == "true"
	enableSns := os.Getenv("EnableSns") == "true"
	enableSqs := os.Getenv("EnableSqs") == "true"
	enableMediaPackage := os.Getenv("EnableMediaPackage") == "true"

	inputValidateData := InputValidateData{
		GUID:                   event.GUID,
		StartTime:              time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		WorkflowTrigger:        event.WorkflowTrigger,
		WorkflowStatus:         "Ingest",
		WorkflowName:           os.Getenv("WorkflowName"),
		SrcBucket:              os.Getenv("Source"),
		DestBucket:             os.Getenv("Destination"),
		CloudFront:             os.Getenv("CloudFront"),
		FrameCapture:           frameCapture,
		ArchiveSource:          os.Getenv("ArchiveSource"),
		JobTemplate2160p:       os.Getenv("JMediaConvert_Template_2160p"),
		JobTemplate1080p:       os.Getenv("MediaConvert_Template_1080p"),
		JobTemplate720p:        os.Getenv("MediaConvert_Template_720p"),
		InputRotate:            os.Getenv("InputRotate"),
		AcceleratedTranscoding: os.Getenv("AcceleratedTranscoding"),
		EnableSns:              enableSns,
		EnableSqs:              enableSqs,
		EnableMediaPackage:     enableMediaPackage,
	}

	switch event.WorkflowTrigger {
	case "Video":
		inputValidateData.SrcVideo = strings.Replace(event.Records[0].S3.Object.Key, "+", " ", -1)
	default:
		return nil, fmt.Errorf("input-validate: main.Handler: %w", ErrEventWorkflowTriggerNotDefined)
	}

	return &inputValidateData, nil

}

func main() {
	lambda.Start(Handler)
}
