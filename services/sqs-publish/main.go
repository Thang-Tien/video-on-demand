package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SqsClient interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
}
type Handler struct {
	SqsClient SqsClient
}

type EventDetail struct {
	Timestamp          int64                `json:"timestamp"`
	AccountId          string               `json:"accountId"`
	Queue              string               `json:"queue"`
	JobId              string               `json:"jobId"`
	Status             string               `json:"status"`
	UserMetadata       UserMetadata         `json:"userMetadata"`
	OutputGroupDetails []*OutputGroupDetail `json:"outputGroupDetails"`
	PaddingInserted    int64                `json:"paddingInserted"`
	BlackVideoDetected int64                `json:"blackVideoDetected"`
	Warnings           []*Warning           `json:"warnings"`
}

type OutputGroupDetail struct {
	OutputDetails     []*OutputDetail `json:"outputDetails"`
	PlaylistFilePaths []*string       `json:"playlistFilePaths"`
	Type              string          `json:"type"`
}

type OutputDetail struct {
	OutputFilePaths []*string    `json:"outputFilePaths"`
	DurationInMs    int64        `json:"durationInMs"`
	VideoDetails    *VideoDetail `json:"videoDetails"`
}

type VideoDetail struct {
	WidthInPx              int64   `json:"widthInPx"`
	HeightInPx             int64   `json:"heightInPx"`
	AverageBitrate         float64 `json:"averageBitrate"`
	QvbrAvgQuality         float64 `json:"qvbrAvgQuality"`
	QvbrMinQuality         float64 `json:"qvbrMinQuality"`
	QvbrMaxQuality         float64 `json:"qvbrMaxQuality"`
	QvbrMinQualityLocation float64 `json:"qvbrMinQualityLocation"`
	QvbrMaxQualityLocation float64 `json:"qvbrMaxQualityLocation"`
}

type SqsPublishEvent struct {
	GUID                   string                      `json:"guid"`
	StartTime              string                      `json:"startTime"`
	WorkflowTrigger        string                      `json:"workflowTrigger"`
	WorkflowStatus         string                      `json:"workflowStatus"`
	WorkflowName           string                      `json:"workflowName"`
	SrcBucket              string                      `json:"srcBucket"`
	DestBucket             string                      `json:"destBucket"`
	CloudFront             string                      `json:"cloudFront"`
	FrameCapture           bool                        `json:"frameCapture"`
	ArchiveSource          string                      `json:"archiveSource"`
	JobTemplate2160p       string                      `json:"jobTemplate_2160p"`
	JobTemplate1080p       string                      `json:"jobTemplate_1080p"`
	JobTemplate720p        string                      `json:"jobTemplate_720p"`
	InputRotate            string                      `json:"inputRotate"`
	AcceleratedTranscoding string                      `json:"acceleratedTranscoding"`
	EnableSns              bool                        `json:"enableSns"`
	EnableSqs              bool                        `json:"enableSqs"`
	SrcVideo               string                      `json:"srcVideo"`
	EnableMediaPackage     bool                        `json:"enableMediaPackage"`
	SrcMediainfo           string                      `json:"srcMediainfo"`
	EncodingJob            mediaconvert.CreateJobInput `json:"encodingJob"`
	EncodeJobId            string                      `json:"encodeJobId"`
	EncodingOutput         EventDetail                 `json:"encodingOutput"`
	EndTime                time.Time                   `json:"endTime"`

	// Output
	HlsPlaylist            *string           `json:"hlsPlaylist"`
	HlsUrl                 *string           `json:"hlsUrl"`
	DashPlaylist           *string           `json:"dashPlaylist"`
	DashUrl                *string           `json:"dashUrl"`
	Mp4Outputs             []*string         `json:"mp4Outputs"`
	Mp4Urls                []*string         `json:"mp4Urls"`
	MssPlaylist            *string           `json:"mssPlaylist"`
	MssUrl                 *string           `json:"mssUrl"`
	CmafDashPlaylist       *string           `json:"cmafDashPlaylist"`
	CmafDashUrl            *string           `json:"cmafDashUrl"`
	CmafHlsPlaylist        *string           `json:"cmafHlsPlaylist"`
	CmafHlsUrl             *string           `json:"cmafHlsUrl"`
	ThumbNails             []*string         `json:"thumbNails"`
	ThumbNailsUrls         []*string         `json:"thumbNailsUrls"`
	MediaPackageResourceId string            `json:"mediaPackageResourceId"`
	EgressEndpoints        map[string]string `json:"egressEndpoints"`
}

type Warning struct {
	Code  int64 `json:"code"`
	Count int64 `json:"count"`
}

type UserMetadata struct {
	GUID     string `json:"guid"`
	Workflow string `json:"workflow"`
}

func (h *Handler) HandleRequest(event SqsPublishEvent) (*SqsPublishEvent, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("sqs-publish: main.Handler: Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJSON)

	_, err = h.SqsClient.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(eventJSON)),
		QueueUrl:    aws.String(os.Getenv("SqsQueue")),
	})

	return &event, err
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalf("sqs-publish: main: NewSession: %v", err)
	}

	sqsClient := sqs.New(sess)

	handler := &Handler{
		SqsClient: sqsClient,
	}

	lambda.Start(handler.HandleRequest)
}
