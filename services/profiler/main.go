package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ProfilerInput struct {
	GUID        string `json:"guid"`
	JobTemplate *string `json:"jobTemplate,omitempty"`
}

type ProfilerOutput struct {
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
	SrcHeight              int    `json:"srcHeight"`
	SrcWidth               int    `json:"srcWidth"`
	EncodingProfile        int    `json:"encodingProfile"`
	FrameCaptureHeight     int    `json:"frameCaptureHeight"`
	FrameCaptureWidth      int    `json:"frameCaptureWidth"`
	JobTemplate            string `json:"jobTemplate"`
	IsCustomTemplate       bool   `json:"isCustomTemplate"`
}

type MediaInfo struct {
	Filename  string    `json:"filename"`
	Container Container `json:"container"`
	Video     []Video   `json:"video"`
	Audio     []Audio   `json:"audio"`
}

type Container struct {
	Format       string  `json:"format"`
	FileSize     int     `json:"fileSize"`
	Duration     float64 `json:"duration"`
	TotalBitrate int     `json:"totalBitrate"`
}

type Video struct {
	Codec       string  `json:"codec"`
	Bitrate     int     `json:"bitrate"`
	Duration    float64 `json:"duration"`
	FrameCount  int     `json:"frameCount"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Framerate   float64 `json:"framerate"`
	AspectRatio string  `json:"aspectRatio"`
	ColorSpace  string  `json:"colorSpace"`
}

type Audio struct {
	Codec          string  `json:"codec"`
	Bitrate        int     `json:"bitrate"`
	Duration       float64 `json:"duration"`
	FrameCount     int     `json:"frameCount"`
	BitrateMode    string  `json:"bitrateMode"`
	Channels       int     `json:"channels"`
	SamplingRate   int     `json:"samplingRate"`
	SamplePerFrame int     `json:"samplePerFrame"`
}

type DynamoDBClient interface {
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

func (h *Handler) HandleRequest(event ProfilerInput) (*ProfilerOutput, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("profiler: main.Handler: json.Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJson)

	data, err := h.DynamoDBClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]*dynamodb.AttributeValue{
			"guid": {
				S: aws.String(event.GUID),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("profiler: main.Handler: GetItem: %w", err)
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("profiler: main.Handler: json.Marshal: %w", err)
	}
	log.Printf("Dynamo Data:: %s", dataJson)

	// Check if the item was found
	if data.Item == nil {
		return nil, fmt.Errorf("profiler: main.Handler: item with GUID %s not found", event.GUID)
	}

	output := &ProfilerOutput{
		GUID:                   getStringValue(data.Item, "guid"),
		StartTime:              getStringValue(data.Item, "startTime"),
		WorkflowTrigger:        getStringValue(data.Item, "workflowTrigger"),
		WorkflowStatus:         getStringValue(data.Item, "workflowStatus"),
		WorkflowName:           getStringValue(data.Item, "workflowName"),
		SrcBucket:              getStringValue(data.Item, "srcBucket"),
		DestBucket:             getStringValue(data.Item, "destBucket"),
		CloudFront:             getStringValue(data.Item, "cloudFront"),
		FrameCapture:           getBoolValue(data.Item, "frameCapture"),
		ArchiveSource:          getStringValue(data.Item, "archiveSource"),
		JobTemplate2160p:       getStringValue(data.Item, "jobTemplate_2160p"),
		JobTemplate1080p:       getStringValue(data.Item, "jobTemplate_1080p"),
		JobTemplate720p:        getStringValue(data.Item, "jobTemplate_720p"),
		InputRotate:            getStringValue(data.Item, "inputRotate"),
		AcceleratedTranscoding: getStringValue(data.Item, "acceleratedTranscoding"),
		EnableSns:              getBoolValue(data.Item, "enableSns"),
		EnableSqs:              getBoolValue(data.Item, "enableSqs"),
		SrcVideo:               getStringValue(data.Item, "srcVideo"),
		EnableMediaPackage:     getBoolValue(data.Item, "enableMediaPackage"),
		SrcMediainfo:           getStringValue(data.Item, "srcMediainfo"),
	}

	formatedSrcMediainfo := output.SrcMediainfo
	formatedSrcMediainfo = strings.ReplaceAll(formatedSrcMediainfo, "\n", "")
	formatedSrcMediainfo = strings.ReplaceAll(formatedSrcMediainfo, `\"`, `"`)
	formatedSrcMediainfo = strings.ReplaceAll(formatedSrcMediainfo, " ", "")
	log.Printf("SRC MEDIAINFO:: %s", formatedSrcMediainfo)

	// Parse mediainfo if available
	var mediainfo MediaInfo
	if output.SrcMediainfo != "" {
		err = json.Unmarshal([]byte(formatedSrcMediainfo), &mediainfo)
		if err != nil {
			return nil, fmt.Errorf("profile: main.Handler: Unmarshal %w", err)
		}
	}

	log.Printf("MediaInfo:: %+v", mediainfo)

	output.SrcHeight = mediainfo.Video[0].Height
	output.SrcWidth = mediainfo.Video[0].Width

	profiles := []int{2160, 1080, 720}
	var encodingProfile int
	minProfileDiff := math.MaxInt32

	for _, profile := range profiles {
		profileDiff := int(math.Abs(float64(output.SrcHeight - profile)))
		if profileDiff < minProfileDiff {
			minProfileDiff = profileDiff
			encodingProfile = profile
		}
	}

	output.EncodingProfile = encodingProfile
	if output.FrameCapture {
		ratio := map[int]int{
			2160: 3840,
			1080: 1920,
			720:  1280,
		}

		output.FrameCaptureHeight = encodingProfile
		output.FrameCaptureWidth = ratio[encodingProfile]
	}

	if event.JobTemplate == nil {
		jobTemplates := map[int]string{
			2160: output.JobTemplate2160p,
			1080: output.JobTemplate1080p,
			720:  output.JobTemplate720p,
		}
		output.JobTemplate = jobTemplates[encodingProfile]
		log.Printf("Chosen template:: %s", output.JobTemplate)
		output.IsCustomTemplate = false
	} else {
		output.IsCustomTemplate = true
	}
	

	outputJson, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("profiler: main.Handler: json.Marshal: %w", err)
	}
	log.Printf("RESPONSE:: %s", outputJson)

	return output, nil
}

func getStringValue(item map[string]*dynamodb.AttributeValue, key string) string {
	if val, exists := item[key]; exists && val.S != nil {
		return *val.S
	}
	return ""
}

func getBoolValue(item map[string]*dynamodb.AttributeValue, key string) bool {
	if val, exists := item[key]; exists && val.BOOL != nil {
		return *val.BOOL
	}
	return false
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}

	handler := Handler{
		DynamoDBClient: dynamodb.New(sess),
	}

	lambda.Start(handler.HandleRequest)
}
