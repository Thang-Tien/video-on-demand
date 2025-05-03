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

type DynamoDBClient interface {
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

func (h *Handler) HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	ctx := context.Background()
	var videoInfo VideoInformation
	err := json.Unmarshal([]byte(request.Body), &videoInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	// Update the item in DynamoDB
	values, err := attributevalue.MarshalMap(videoInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Failed to marshal request: %v", err),
		}, nil
	}

	valuesJSON, _ := json.Marshal(values)
	log.Printf("Values: %s", valuesJSON)

	var update expression.UpdateBuilder
	for key, value := range values {
		uncapitalizedKey := string(key[0]+32) + key[1:]
		update = update.Set(expression.Name(uncapitalizedKey), expression.Value(value))
	}
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Failed to build expression: %v", err),
		}, nil
	}

	input := dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "VIDEO#" + request.PathParameters["pk"]},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ReturnValues:              types.ReturnValueAllNew,
	}

	inputJSON, _ := json.Marshal(input)
	log.Printf("UpdateItemInput: %s", inputJSON)

	res, err := h.DynamoDBClient.UpdateItem(ctx, &input)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Failed to update item in DynamoDB: %v", err),
		}, nil
	}

	log.Println("UPDATE:: Successfully updated item in DynamoDB")

	var updatedVideoInfo VideoInformation
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedVideoInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Failed to unmarshal updated item: %v", err),
		}, nil
	}

	updatedVideoInfoJSON, err := json.Marshal(updatedVideoInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf("Failed to marshal updated item to JSON: %v", err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "application/json",
		},
		Body: string(updatedVideoInfoJSON),
	}, nil
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
