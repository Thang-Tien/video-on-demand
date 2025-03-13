package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

type DynamoData struct {
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

type DynamoDBClient interface {
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}
type S3Client interface {
	ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
	S3Client       S3Client
}

func (h *Handler) HandleRequest(event events.EventBridgeEvent) (*DynamoData, error) {
	eventJson, _ := json.Marshal(event)
	log.Printf("REQUEST:: %s", eventJson)

	var eventDetail EventDetail
	err := json.Unmarshal(event.Detail, &eventDetail)
	if err != nil {
		return nil, fmt.Errorf("output-validate: main.Handler.HandleRequest: json.Unmarshal: %w", err)
	}

	data, err := h.DynamoDBClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]*dynamodb.AttributeValue{
			"guid": {
				S: aws.String(eventDetail.UserMetadata.GUID),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("output-validate: main.Handler.HandlerRequest: dynamodb.GetItem: %w", err)
	}

	// Map DynamoDB item to DynamoData struct
	var dynamoData DynamoData
	if len(data.Item) == 0 {
		return nil, fmt.Errorf("output-validate: main.Handler.HandlerRequest: dynamodb data (guid = %s) is empty", eventDetail.UserMetadata.GUID)
	}

	err = dynamodbattribute.UnmarshalMap(data.Item, &dynamoData)
	if err != nil {
		return nil, fmt.Errorf("output-validate: main.Handler.HandlerRequest: dynamodbattribute.UnmarshalMap %w", err)
	}

	dynamoData.EncodingOutput = eventDetail
	dynamoData.EndTime = time.Now().UTC()
	dynamoData.WorkflowStatus = "Complete"

	if len(eventDetail.OutputGroupDetails) == 0 {
		return nil, fmt.Errorf("output-validate: main.Handler.HandleRequest: no output group details found")
	}

	for _, outputGroupDetail := range eventDetail.OutputGroupDetails {
		log.Printf("%s found in outputs", outputGroupDetail.Type)

		switch outputGroupDetail.Type {
		case "HLS_GROUP":
			dynamoData.HlsPlaylist = outputGroupDetail.PlaylistFilePaths[0]
			dynamoData.HlsUrl = aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*dynamoData.HlsPlaylist)))
		case "DASH_ISO_GROUP":
			dynamoData.DashPlaylist = outputGroupDetail.PlaylistFilePaths[0]
			dynamoData.DashUrl = aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*dynamoData.DashPlaylist)))
		case "FILE_GROUP":
			files, urls := []*string{}, []*string{}
			for _, outputDetail := range outputGroupDetail.OutputDetails {
				if outputDetail.OutputFilePaths != nil {
					files = append(files, outputDetail.OutputFilePaths[0])
					urls = append(urls, aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*outputDetail.OutputFilePaths[0]))))
				}
			}

			if len(files) > 0 {
				if ext := strings.Split(*files[0], "."); ext[len(ext)-1] == "mp4" {
					dynamoData.Mp4Outputs = files
					dynamoData.Mp4Urls = urls
				}
			}
		case "MS_SMOOTH_GROUP":
			dynamoData.MssPlaylist = outputGroupDetail.PlaylistFilePaths[0]
			dynamoData.MssUrl = aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*dynamoData.MssPlaylist)))
		case "CMAF_GROUP":
			dynamoData.CmafDashPlaylist = outputGroupDetail.PlaylistFilePaths[0]
			dynamoData.CmafDashUrl = aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*dynamoData.CmafDashPlaylist)))

			dynamoData.CmafHlsPlaylist = outputGroupDetail.PlaylistFilePaths[1]
			dynamoData.CmafHlsUrl = aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, buildUrl(*dynamoData.CmafHlsPlaylist)))
		default:
			return nil, fmt.Errorf("output-validate: main.Handler.HandleRequest: unknown output group type: %s", outputGroupDetail.Type)
		}
	}

	if dynamoData.FrameCapture {
		thumbNails := []*string{}
		thumbNailsUrls := []*string{}

		thumbNailsData, err := h.S3Client.ListObjects(&s3.ListObjectsInput{
			Bucket: aws.String(dynamoData.DestBucket),
			Prefix: aws.String(fmt.Sprintf("%s/thumbnails", dynamoData.GUID)),
		})
		if err != nil {
			return nil, fmt.Errorf("output-validate: main.Handler.HandleRequest: s3.ListObjects: %w", err)
		}

		if len(thumbNailsData.Contents) > 0 {
			lastImg := thumbNailsData.Contents[len(thumbNailsData.Contents)-1]
			thumbNails = append(thumbNails, aws.String(fmt.Sprintf("s3://%s/%s", dynamoData.DestBucket, *lastImg.Key)))
			thumbNailsUrls = append(thumbNailsUrls, aws.String(fmt.Sprintf("https://%s/%s", dynamoData.CloudFront, *lastImg.Key)))
		} else {
			return nil, fmt.Errorf("output-validate: main.Handler.HandleRequest: no thumbnails found in S3")
		}

		dynamoData.ThumbNails = thumbNails
		dynamoData.ThumbNailsUrls = thumbNailsUrls
	}

	return &dynamoData, nil
}

func buildUrl(s3Path string) string {
	s := strings.Split(s3Path, "/")
	return fmt.Sprintf("%s/%s/%s", s[len(s)-3], s[len(s)-2], s[len(s)-1])
}

func main() {
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(os.Getenv("AWS_REGION")),
		},
	)
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}

	dynamoClient := dynamodb.New(sess)
	s3Client := s3.New(sess)

	handler := Handler{
		DynamoDBClient: dynamoClient,
		S3Client:       s3Client,
	}

	lambda.Start(handler.HandleRequest)
}
