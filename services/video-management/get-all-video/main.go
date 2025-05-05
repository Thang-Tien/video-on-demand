package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type VideoInformation struct {
	PK string `json:"PK"` // VIDEO#{videoId}
	SK string `json:"SK"` // METADATA

	// Metadata Information
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
	Views                int                 `json:"views"`
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

type Response struct {
	Videos []VideoInformation `json:"videos"`
	Count  int                `json:"count"`
}

type DynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// TODO: Validate request

	// search terms
	genres := parseGenres(request.QueryStringParameters["genres"])
	casts := parseCasts(request.QueryStringParameters["casts"])
	countries := parseCountries(request.QueryStringParameters["countries"])
	productionYears := parseProductionYears(request.QueryStringParameters["productionYears"])
	title := parseTitle(request.QueryStringParameters["title"])

	filterExpr, err := buildFilterExpression(genres, casts, countries, productionYears, title)
	if err != nil {
		log.Printf("Failed to build expression: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to build expression: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	// Create key condition expression for the GSI where SK is the partition key and PK is the sort key
	keyCondExpr := expression.Key("SK").Equal(expression.Value("METADATA")).And(
		expression.Key("PK").BeginsWith("VIDEO#"),
	)

	// Create a projection expression to only retrieve the fields we need
	// Create a projection expression to include all fields from VideoInformation
	projExpr := expression.NamesList(
		expression.Name("PK"),
		expression.Name("SK"),
		expression.Name("startTime"),
		expression.Name("workflowTrigger"),
		expression.Name("workflowStatus"),
		expression.Name("workflowName"),
		expression.Name("srcBucket"),
		expression.Name("destBucket"),
		expression.Name("cloudFront"),
		expression.Name("frameCapture"),
		expression.Name("archiveSource"),
		expression.Name("jobTemplate_2160p"),
		expression.Name("jobTemplate_1080p"),
		expression.Name("jobTemplate_720p"),
		expression.Name("inputRotate"),
		expression.Name("acceleratedTranscoding"),
		expression.Name("enableSns"),
		expression.Name("enableSqs"),
		expression.Name("srcVideo"),
		expression.Name("enableMediaPackage"),
		expression.Name("srcMediainfo"),
		expression.Name("encodeJobId"),
		expression.Name("endTime"),
		expression.Name("hlsPlaylist"),
		expression.Name("hlsUrl"),
		expression.Name("dashPlaylist"),
		expression.Name("dashUrl"),
		expression.Name("mp4Outputs"),
		expression.Name("mp4Urls"),
		expression.Name("mssPlaylist"),
		expression.Name("mssUrl"),
		expression.Name("cmafDashPlaylist"),
		expression.Name("cmafDashUrl"),
		expression.Name("cmafHlsPlaylist"),
		expression.Name("cmafHlsUrl"),
		expression.Name("thumbNails"),
		expression.Name("thumbNailsUrls"),
		expression.Name("mediaPackageResourceId"),
		expression.Name("egressEndpoints"),
		expression.Name("title"),
		expression.Name("originalTitle"),
		expression.Name("description"),
		expression.Name("plotSummary"),
		expression.Name("releaseDate"),
		expression.Name("productionYear"),
		expression.Name("languages"),
		expression.Name("countryOfOrigin"),
		expression.Name("ageRating"),
		expression.Name("directors"),
		expression.Name("producers"),
		expression.Name("writers"),
		expression.Name("cast"),
		expression.Name("cinematographer"),
		expression.Name("musicComposer"),
		expression.Name("editor"),
		expression.Name("productionDesigner"),
		expression.Name("costumeDesigner"),
		expression.Name("subtitleLanguages"),
		expression.Name("genres"),
		expression.Name("tags"),
		expression.Name("themes"),
		expression.Name("seriesInformation"),
		expression.Name("sequelPrequel"),
		expression.Name("similarMovies"),
		expression.Name("posterUrls"),
		expression.Name("trailerUrls"),
		expression.Name("behindTheScenes"),
		expression.Name("commentaryTracks"),
		expression.Name("deletedScenes"),
		expression.Name("interviews"),
		expression.Name("awards"),
		expression.Name("criticRatings"),
		expression.Name("userRating"),
		expression.Name("boxOfficePerformance"),
		expression.Name("views"),
	)

	// Build the query expression with the key condition, filter and projection
	queryExpr, err := expression.NewBuilder().
		WithKeyCondition(keyCondExpr).
		WithFilter(filterExpr).
		WithProjection(projExpr).
		Build()
	if err != nil {
		log.Printf("Failed to build query expression: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to build query expression: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("DynamoDBTable")),
		IndexName:                 aws.String("SK-PK-index"),
		KeyConditionExpression:    queryExpr.KeyCondition(),
		FilterExpression:          queryExpr.Filter(),
		ProjectionExpression:      queryExpr.Projection(),
		ExpressionAttributeNames:  queryExpr.Names(),
		ExpressionAttributeValues: queryExpr.Values(),
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

	var videos []VideoInformation
	err = attributevalue.UnmarshalListOfMaps(result.Items, &videos)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to unmarshal items: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	response := Response{
		Videos: videos,
		Count:  len(videos),
	}

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

func buildFilterExpression(genres, casts, countries, productionYears []string, title string) (expression.ConditionBuilder, error) {
	var filter expression.ConditionBuilder
	var hasFilter bool

	addORContains := func(field string, values []string) expression.ConditionBuilder {
		var orCond expression.ConditionBuilder
		for i, val := range values {
			cond := expression.Name(field).Contains(val)
			if i == 0 {
				orCond = cond
			} else {
				orCond = orCond.Or(cond)
			}
		}
		return orCond
	}

	if len(genres) > 0 {
		if hasFilter {
			filter = filter.And(addORContains("genres", genres))
		} else {
			filter = addORContains("genres", genres)
			hasFilter = true
		}
	}
	if len(casts) > 0 {
		if hasFilter {
			filter = filter.And(addORContains("casts", casts))
		} else {
			filter = addORContains("casts", casts)
			hasFilter = true
		}
	}
	if len(countries) > 0 {
		if hasFilter {
			filter = filter.And(addORContains("countryOfOrigin", countries))
		} else {
			filter = addORContains("countryOfOrigin", countries)
			hasFilter = true
		}
	}
	if len(productionYears) > 0 {
		if hasFilter {
			filter = filter.And(addORContains("productionYear", productionYears))
		} else {
			filter = addORContains("productionYear", productionYears)
			hasFilter = true
		}
	}
	if title != "" {
		titleCond := expression.Name("title").Contains(title)
		if hasFilter {
			filter = filter.And(titleCond)
		} else {
			filter = titleCond
			hasFilter = true
		}
	}

	if !hasFilter {
		// Return a condition that will always be true, using a non-key attribute
		// We're checking if workflowName attribute exists which should be true for all items
		return expression.AttributeExists(expression.Name("workflowName")), nil
	}

	return filter, nil
}

func parseGenres(input string) []string {
	if input == "" {
		return nil
	}
	genres := strings.Split(input, ",")
	return genres
}

func parseCasts(input string) []string {
	if input == "" {
		return nil
	}
	casts := strings.Split(input, ",")
	return casts
}

func parseCountries(input string) []string {
	if input == "" {
		return nil
	}
	countries := strings.Split(input, ",")
	return countries
}

func parseProductionYears(input string) []string {
	if input == "" {
		return nil
	}
	productionYears := strings.Split(input, ",")
	return productionYears
}

func parseTitle(input string) string {
	return strings.TrimSpace(input)
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
