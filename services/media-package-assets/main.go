package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/mediapackagevod"
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

type MediaPackageAssetsEvent struct {
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

type MediaPackageVodClient interface {
	CreateAsset(input *mediapackagevod.CreateAssetInput) (*mediapackagevod.CreateAssetOutput, error)
}

type Handler struct {
	MediaPackageVodClient MediaPackageVodClient
}

func (h *Handler) HanleRequest(event MediaPackageAssetsEvent) (*MediaPackageAssetsEvent, error) {
	eventJson, _ := json.Marshal(event)
	log.Printf("REQUEST:: %s", eventJson)

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	randomId := hex.EncodeToString(randomBytes)

	arn, err := buildArnFromUri(*event.HlsPlaylist)
	if err != nil {
		return nil, fmt.Errorf("media-package-assets: main.Handler.HandleRequest: buildArnFromUri: %w", err)
	}
	input := &mediapackagevod.CreateAssetInput{
		Id:               aws.String(randomId),
		PackagingGroupId: aws.String(os.Getenv("GroupId")),
		SourceArn:        aws.String(arn),
		SourceRoleArn:    aws.String(os.Getenv("MediaPackageVodRole")),
		ResourceId:       aws.String(os.Getenv(randomId)),
	}
	input.Tags = map[string]*string{
		"SolutionId": aws.String("vod-solution"),
	}

	inputJson, _ := json.Marshal(input)
	log.Printf("Ingesting asset:: %s", inputJson)

	res, err := h.MediaPackageVodClient.CreateAsset(input)
	if err != nil {
		return nil, fmt.Errorf("media-package-assets: main.Handler.HandleRequest: CreateAsset: %w", err)
	}

	event.MediaPackageResourceId = randomId
	event.EgressEndpoints, err = convertEndpoint(res.EgressEndpoints, event.CloudFront)
	if err != nil {
		return nil, fmt.Errorf("media-package-assets: main.Handler.HandleRequest: convertEndpoint: %w", err)
	}

	endpointJson, _ := json.Marshal(event.EgressEndpoints)
	log.Printf("ENDPOINTS:: %s", endpointJson)

	return &event, nil
}

func buildArnFromUri(s3Uri string) (string, error) {
	const S3_URI_ID = "s3://"

	if s3Uri[:len(S3_URI_ID)] != S3_URI_ID {
		return "", fmt.Errorf("invalid S3 URI: %s", s3Uri)
	}

	source := strings.ReplaceAll(s3Uri, S3_URI_ID, "")
	return fmt.Sprintf("arn:aws:s3:::%s", source), nil
}

func convertEndpoint(egressEndpoints []*mediapackagevod.EgressEndpoint, cloudFrontEndpoint string) (map[string]string, error) {
	url, err := url.Parse(os.Getenv("GroupDomainName"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CloudFront endpoint: %s", err)
	}

	updatedEndpoints := make(map[string]string)
	for _, endpoint := range egressEndpoints {
		if endpoint.PackagingConfigurationId == nil {
			return nil, fmt.Errorf("packagingConfigurationId is nil")
		}
		splited := strings.Split(*endpoint.PackagingConfigurationId, "-")
		config := strings.ToUpper(splited[len(splited)-1])

		if endpoint.Url == nil {
			return nil, fmt.Errorf("url is nil")
		}

		updatedEndpoints[config] = strings.ReplaceAll(*endpoint.Url, url.Host, cloudFrontEndpoint)
	}

	return updatedEndpoints, nil
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalf("Error creating session: %s", err)
	}
	mediaPackageVodClient := mediapackagevod.New(sess)

	handler := &Handler{
		MediaPackageVodClient: mediaPackageVodClient,
	}

	lambda.Start(handler.HanleRequest)
}
