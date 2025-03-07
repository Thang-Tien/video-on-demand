package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

var ErrWorkflowStatusNotDefined = errors.New("workflow Status not defined")

var NOT_APPLICABLE_PROPERTIES = []string{
	"mp4Outputs",
	"mp4Urls",
	"hlsPlaylist",
	"hlsUrl",
	"dashPlaylist",
	"dashUrl",
	"mssPlaylist",
	"mssUrl",
	"cmafDashPlaylist",
	"cmafDashUrl",
	"cmafHlsPlaylist",
	"cmafHlsUrl",
}

type SNSClient interface {
	Publish(input *sns.PublishInput) (*sns.PublishOutput, error)
}

type Handler struct {
	snsClient SNSClient
}

type SNSNotificationEvent struct {
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
	SrcMediainfo           string `json:"srcMediainfo"`
}

type SNSNotificationOutput struct {
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
	SrcMediainfo           string `json:"srcMediainfo"`
}

type Message struct {
	Status   string `json:"workflowStatus"`
	GUID     string `json:"guid"`
	SrcVideo string `json:"srcVideo"`
}

func (h *Handler) HandleRequest(event SNSNotificationEvent) (*SNSNotificationOutput, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJSON)

	var message Message
	subject := "Workflow Status:: " + event.WorkflowStatus + ":: " + event.GUID

	if event.WorkflowStatus == "Complete" {
		// WIP - Delete some fields of the event
	} else if event.WorkflowStatus == "Ingest" {
		message = Message{
			Status:   event.WorkflowStatus,
			GUID:     event.GUID,
			SrcVideo: event.SrcVideo,
		}
	} else {
		return nil, ErrWorkflowStatusNotDefined
	}

	messageJson, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Marshal: %w", err)
	}
	log.Printf("SEND SNS:: %s", messageJson)

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Marshal: %w", err)
	}
	messageString := string(messageBytes)

	_, err = h.snsClient.Publish(&sns.PublishInput{
		Message:  aws.String(messageString),
		Subject:  aws.String(subject),
		TopicArn: aws.String(os.Getenv("SnsTopic")),
	})
	if err != nil {
		return nil,  fmt.Errorf("sns-notification: main.Handler: Publish: %w", err)
	}

	return &SNSNotificationOutput{
		GUID:                   event.GUID,
		StartTime:              event.StartTime,
		WorkflowTrigger:        event.WorkflowTrigger,
		WorkflowStatus:         event.WorkflowStatus,
		WorkflowName:           event.WorkflowName,
		SrcBucket:              event.SrcBucket,
		DestBucket:             event.DestBucket,
		CloudFront:             event.CloudFront,
		FrameCapture:           event.FrameCapture,
		ArchiveSource:          event.ArchiveSource,
		JobTemplate2160p:       event.JobTemplate2160p,
		JobTemplate1080p:       event.JobTemplate1080p,
		JobTemplate720p:        event.JobTemplate720p,
		InputRotate:            event.InputRotate,
		AcceleratedTranscoding: event.AcceleratedTranscoding,
		EnableSns:              event.EnableSns,
		EnableSqs:              event.EnableSqs,
		SrcVideo:               event.SrcVideo,
		EnableMediaPackage:     event.EnableMediaPackage,
		SrcMediainfo:           event.SrcMediainfo,
	}, nil

}

func main() {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(os.Getenv("AWS_REGION")),
		},
	))
	snsClient := sns.New(sess)
	handler := Handler{
		snsClient: snsClient,
	}

	lambda.Start(handler.HandleRequest)
}
