package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/s3"
)

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

type ArchiveSourceEvent struct {
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
	HlsPlaylist      *string   `json:"hlsPlaylist"`
	HlsUrl           *string   `json:"hlsUrl"`
	DashPlaylist     *string   `json:"dashPlaylist"`
	DashUrl          *string   `json:"dashUrl"`
	Mp4Outputs       []*string `json:"mp4Outputs"`
	Mp4Urls          []*string `json:"mp4Urls"`
	MssPlaylist      *string   `json:"mssPlaylist"`
	MssUrl           *string   `json:"mssUrl"`
	CmafDashPlaylist *string   `json:"cmafDashPlaylist"`
	CmafDashUrl      *string   `json:"cmafDashUrl"`
	CmafHlsPlaylist  *string   `json:"cmafHlsPlaylist"`
	CmafHlsUrl       *string   `json:"cmafHlsUrl"`
	ThumbNails       []*string `json:"thumbNails"`
	ThumbNailsUrls   []*string `json:"thumbNailsUrls"`
}

type Warning struct {
	Code  int64 `json:"code"`
	Count int64 `json:"count"`
}

type UserMetadata struct {
	GUID     string `json:"guid"`
	Workflow string `json:"workflow"`
}

type S3Client interface {
	PutObjectTagging(input *s3.PutObjectTaggingInput) (*s3.PutObjectTaggingOutput, error)
}

type Handler struct {
	S3Client S3Client
}

func (h *Handler) HandleRequest(event ArchiveSourceEvent) (*ArchiveSourceEvent, error) {
	eventJson, _ := json.Marshal(event)
	log.Printf("REQUEST:: %s", eventJson)

	stackName := strings.ReplaceAll(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"), "-archive-source", "")
	input := &s3.PutObjectTaggingInput{
		Bucket: aws.String(event.SrcBucket),
		Key:    aws.String(event.SrcVideo),
		Tagging: &s3.Tagging{
			TagSet: []*s3.Tag{
				{
					Key:   aws.String("guid"),
					Value: aws.String(event.GUID),
				},
				{
					Key:   aws.String(stackName),
					Value: aws.String(event.ArchiveSource),
				},
			},
		},
	}

	_, err := h.S3Client.PutObjectTagging(input)
	if err != nil {
		log.Printf("ERROR: failed to tag source object: %v", err)
		return nil, fmt.Errorf("archive-source: main.Handler.HandleRequest: PutObjectTagging: %w", err)
	}

	return &event, nil
}

func main() {
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(os.Getenv("AWS_REGION")),
		},
	)

	if err != nil {
		log.Fatalf("failed to create a new session: %v", err)
	}

	s3Client := s3.New(sess)
	handler := &Handler{
		S3Client: s3Client,
	}
	lambda.Start(handler.HandleRequest)
}
