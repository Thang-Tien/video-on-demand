package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type VideoInformation struct {
	PK                     string    `json:"pk"`
	SK                     string    `json:"sk"`
	StartTime              string    `json:"startTime"`
	WorkflowTrigger        string    `json:"workflowTrigger"`
	WorkflowStatus         string    `json:"workflowStatus"`
	WorkflowName           string    `json:"workflowName"`
	SrcBucket              string    `json:"srcBucket"`
	DestBucket             string    `json:"destBucket"`
	CloudFront             string    `json:"cloudFront"`
	FrameCapture           bool      `json:"frameCapture"`
	ArchiveSource          string    `json:"archiveSource"`
	JobTemplate2160p       string    `json:"jobTemplate_2160p"`
	JobTemplate1080p       string    `json:"jobTemplate_1080p"`
	JobTemplate720p        string    `json:"jobTemplate_720p"`
	InputRotate            string    `json:"inputRotate"`
	AcceleratedTranscoding string    `json:"acceleratedTranscoding"`
	EnableSns              bool      `json:"enableSns"`
	EnableSqs              bool      `json:"enableSqs"`
	SrcVideo               string    `json:"srcVideo"`
	EnableMediaPackage     bool      `json:"enableMediaPackage"`
	SrcMediainfo           string    `json:"srcMediainfo"`
	EncodeJobId            string    `json:"encodeJobId"`
	EndTime                time.Time `json:"endTime"`

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

	// Basic Information
	Title           *string    `json:"title"`
	OriginalTitle   *string    `json:"originalTitle"`
	Description     *string    `json:"description"`
	PlotSummary     *string    `json:"plotSummary"`
	ReleaseDate     *time.Time `json:"releaseDate"`
	ProductionYear  *int       `json:"productionYear"`
	Languages       *[]string  `json:"languages"`
	CountryOfOrigin *string    `json:"countryOfOrigin"`
	AgeRating       *string    `json:"ageRating"`

	// Creative Elements
	Directors          *[]string     `json:"directors"`
	Producers          *[]string     `json:"producers"`
	Writers            *[]string     `json:"writers"`
	Cast               *[]CastMember `json:"cast"`
	Cinematographer    *string       `json:"cinematographer"`
	MusicComposer      *string       `json:"musicComposer"`
	Editor             *string       `json:"editor"`
	ProductionDesigner *string       `json:"productionDesigner"`
	CostumeDesigner    *string       `json:"costumeDesigner"`

	// Technical Information
	SubtitleLanguages *[]string `json:"subtitleLanguages"`

	// Categorization
	Genres            *[]string          `json:"genres"`
	Tags              *[]string          `json:"tags"`
	Themes            *[]string          `json:"themes"`
	SeriesInformation *SeriesInfo        `json:"seriesInformation"`
	SequelPrequel     *SequelPrequelInfo `json:"sequelPrequel"`
	SimilarMovies     *[]string          `json:"similarMovies"`

	// Supplementary Content
	PosterURLs       *[]string `json:"posterUrls"`
	TrailerURLs      *[]string `json:"trailerUrls"`
	BehindTheScenes  *[]string `json:"behindTheScenes"`
	CommentaryTracks *[]string `json:"commentaryTracks"`
	DeletedScenes    *[]string `json:"deletedScenes"`
	Interviews       *[]string `json:"interviews"`

	// Reception and Metrics
	Awards               *[]string           `json:"awards"`
	CriticRatings        *map[string]float64 `json:"criticRatings"`
	UserRating           *float64            `json:"userRating"`
	BoxOfficePerformance *float64            `json:"boxOfficePerformance"`
	Views                *int                `json:"views"`
}

type CastMember struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type SeriesInfo struct {
	Franchise          string `json:"franchise"`
	ChronologicalOrder int    `json:"chronologicalOrder"`
	ReleaseOrder       int    `json:"releaseOrder"`
}

type SequelPrequelInfo struct {
	Prequels []string `json:"prequels"`
	Sequels  []string `json:"sequels"`
}

type Views struct {
	PK      string `json:"PK"`
	SK      string `json:"SK"`
	VideoID string `json:"videoId"`
}

// New response structure that includes full video details
type VideoWithViews struct {
	Views       Views            `json:"views"`
	Information VideoInformation `json:"information"`
}

type Response struct {
	Videos []VideoWithViews `json:"videos"`
	Count  int              `json:"count"`
}

// Modified DynamoDBClient interface to include GetItem for fetching individual videos
type DynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

// Modified to get full video information for each trending video
func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// Validate request
	monthAndYear := request.QueryStringParameters["monthAndYear"]

	expr, err := buildKeyConditionExpression(monthAndYear)
	if err != nil {
		log.Printf("Failed to build key condition expression: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to build key condition expression: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("DynamoDBTable")),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(false), // Descending order
	}

	jsonInput, _ := json.Marshal(queryInput)
	log.Printf("QueryInput: %s", jsonInput)

	result, err := h.DynamoDBClient.Query(ctx, queryInput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to query DynamoDB: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	var views []Views
	err = attributevalue.UnmarshalListOfMaps(result.Items, &views)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to unmarshal items: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	// Get detailed information for each video
	videosWithDetails := []VideoWithViews{}

	for _, view := range views {
		// Extract videoId from the view item
		videoID := view.VideoID

		// Get full video information
		videoInfo, err := h.getVideoInformation(ctx, videoID)
		if err != nil {
			log.Printf("Error retrieving video information for %s: %v", videoID, err)
			continue
		}

		// Add to result list
		videosWithDetails = append(videosWithDetails, VideoWithViews{
			Views:       view,
			Information: videoInfo,
		})
	}

	// Prepare the enhanced response
	response := Response{
		Videos: videosWithDetails,
		Count:  len(videosWithDetails),
	}

	// Convert response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	log.Printf("Response: %s", responseJSON)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJSON),
		Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
	}, nil
}

// Helper function to get full video information for a given videoId
func (h *Handler) getVideoInformation(ctx context.Context, videoID string) (VideoInformation, error) {
	var videoInfo VideoInformation

	// First get the video information
	videoResult, err := h.DynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "VIDEO#" + videoID},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return videoInfo, fmt.Errorf("failed to get video information: %w", err)
	}

	if videoResult.Item == nil {
		return videoInfo, fmt.Errorf("video not found")
	}

	// Unmarshal the video information
	err = attributevalue.UnmarshalMap(videoResult.Item, &videoInfo)
	if err != nil {
		return videoInfo, fmt.Errorf("failed to unmarshal video information: %w", err)
	}

	return videoInfo, nil
}

func buildKeyConditionExpression(monthAndYear string) (expression.Expression, error) {
	var keyCondition expression.KeyConditionBuilder

	if monthAndYear != "" {
		// Query for period-based trending videos
		keyCondition = expression.Key("PK").Equal(expression.Value(fmt.Sprintf("PERIOD#%s", monthAndYear))).
			And(expression.Key("SK").BeginsWith("VIEWS#"))
	} else {
		// Default to show most recent period if not specified
		currentTime := time.Now()
		defaultPeriod := fmt.Sprintf("%d-%02d", currentTime.Year(), currentTime.Month())

		keyCondition = expression.Key("PK").Equal(expression.Value(fmt.Sprintf("PERIOD#%s", defaultPeriod))).
			And(expression.Key("SK").BeginsWith("VIEWS#"))
	}

	// Return final expression
	return expression.NewBuilder().WithKeyCondition(keyCondition).Build()
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-southeast-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	handler := &Handler{
		DynamoDBClient: dynamoDBClient,
	}

	// Start the Lambda function
	lambda.Start(handler.HandleRequest)
}
