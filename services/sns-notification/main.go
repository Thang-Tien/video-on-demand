package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
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

type Warning struct {
	Code  int64 `json:"code"`
	Count int64 `json:"count"`
}

type UserMetadata struct {
	GUID     string `json:"guid"`
	Workflow string `json:"workflow"`
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

type SNSNotificationOutput struct {
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

type Message struct {
	Status   string `json:"workflowStatus"`
	GUID     string `json:"guid"`
	SrcVideo string `json:"srcVideo"`
}

type CompleteMessage struct {
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

	EncodeJobId            string            `json:"encodeJobId"`
	EndTime                time.Time         `json:"endTime"`
	ThumbNails             []*string         `json:"thumbNails"`
	ThumbNailsUrls         []*string         `json:"thumbNailsUrls"`
	MediaPackageResourceId string            `json:"mediaPackageResourceId"`
	EgressEndpoints        map[string]string `json:"egressEndpoints"`
}

func (h *Handler) HandleRequest(event SNSNotificationEvent) (*SNSNotificationOutput, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJSON)

	var message interface{}
	subject := "Workflow Status:: " + event.WorkflowStatus + ":: " + event.GUID

	if event.WorkflowStatus == "Complete" {
		message = CompleteMessage{
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
			EncodeJobId:            event.EncodeJobId,
			EndTime:                event.EndTime,
			ThumbNails:             event.ThumbNails,
			ThumbNailsUrls:         event.ThumbNailsUrls,
			MediaPackageResourceId: event.MediaPackageResourceId,
			EgressEndpoints:        event.EgressEndpoints,
		}

	} else if event.WorkflowStatus == "Ingest" {
		message = Message{
			Status:   event.WorkflowStatus,
			GUID:     event.GUID,
			SrcVideo: event.SrcVideo,
		}
	} else {
		return nil, ErrWorkflowStatusNotDefined
	}

	messageJson, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Marshal: %w", err)
	}
	log.Printf("MESSAGE:: %s", messageJson)

	_, err = h.snsClient.Publish(&sns.PublishInput{
		Message:  aws.String(string(messageJson)),
		Subject:  aws.String(subject),
		TopicArn: aws.String(os.Getenv("SnsTopic")),
	})
	if err != nil {
		return nil, fmt.Errorf("sns-notification: main.Handler: Publish: %w", err)
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
		EncodingJob:            event.EncodingJob,
		EncodeJobId:            event.EncodeJobId,
		EncodingOutput:         event.EncodingOutput,
		EndTime:                event.EndTime,
		HlsPlaylist:            event.HlsPlaylist,
		HlsUrl:                 event.HlsUrl,
		DashPlaylist:           event.DashPlaylist,
		DashUrl:                event.DashUrl,
		Mp4Outputs:             event.Mp4Outputs,
		Mp4Urls:                event.Mp4Urls,
		MssPlaylist:            event.MssPlaylist,
		MssUrl:                 event.MssUrl,
		CmafDashPlaylist:       event.CmafDashPlaylist,
		CmafDashUrl:            event.CmafDashUrl,
		CmafHlsPlaylist:        event.CmafHlsPlaylist,
		CmafHlsUrl:             event.CmafHlsUrl,
		ThumbNails:             event.ThumbNails,
		ThumbNailsUrls:         event.ThumbNailsUrls,
		MediaPackageResourceId: event.MediaPackageResourceId,
		EgressEndpoints:        event.EgressEndpoints,
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
