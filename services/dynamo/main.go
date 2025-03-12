package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"unicode"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
)

type DynamoDBClient interface {
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
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

type DynamoEvent struct {
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

type DynamoOutput struct {
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

func (h *Handler) HandleRequest(event DynamoEvent) (*DynamoOutput, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("dynamo: main.Handler.HandleRequest: Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJson)

	// Update the item in DynamoDB

	values, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return nil, fmt.Errorf("dynamo: main.Handler.HandleRequest: MarshalMap: %w", err)
	}

	delete(values, "guid")

	expression := "SET "
	attributeValues := make(map[string]*dynamodb.AttributeValue)
	valuesWithNumberKey := make(map[string]*dynamodb.AttributeValue)
	counter := 1
	for key, value := range values {
		placeholder := fmt.Sprintf(":%d", counter)
		key = string(unicode.ToLower(rune(key[0]))) + key[1:]
		expression += fmt.Sprintf("%s = %s, ", key, placeholder)
		attributeValues[placeholder] = value
		valuesWithNumberKey[":"+strconv.Itoa(counter)] = value
		counter++
	}
	if len(expression) > 2 {
		expression = expression[:len(expression)-2]
	}

	log.Printf("expression:: %s", expression)
	valuesJson, _ := json.Marshal(valuesWithNumberKey)
	log.Printf("values:: %s", valuesJson)

	input := dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]*dynamodb.AttributeValue{
			"guid": {
				S: aws.String(event.GUID),
			},
		},
		UpdateExpression:          aws.String(expression),
		ExpressionAttributeValues: valuesWithNumberKey,
	}

	_, err = h.DynamoDBClient.UpdateItem(&input)
	if err != nil {
		return nil, fmt.Errorf("dynamo: main.Handler.HandleRequest: UpdateItem: %w", err)
	}

	log.Println("UPDATE:: Successfully updated item in DynamoDB")

	output := &DynamoOutput{
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
	}

	return output, nil
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	// Create DynamoDB client
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	dynamo := dynamodb.New(sess)

	log.Println("DynamoDB client initialized:", dynamo)

	handler := &Handler{
		DynamoDBClient: dynamo,
	}

	lambda.Start(handler.HandleRequest)
}
