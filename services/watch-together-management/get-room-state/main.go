package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type RoomState struct {
	PK            string    `json:"PK" dynamodbav:"PK"`
	SK            string    `json:"SK" dynamodbav:"SK"`
	RoomID        string    `json:"roomId" dynamodbav:"roomId"`
	LastStartTime time.Time `json:"lastStartTime" dynamodbav:"lastStartTime"`
	LastVideoTime float64   `json:"lastVideoTime" dynamodbav:"lastVideoTime"`
	Status        string    `json:"status" dynamodbav:"status"` // "playing", "paused"
	VideoID       string    `json:"videoId" dynamodbav:"videoId"`
}

type DynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type RoomStateHandler struct {
	DynamoDBClient DynamoDBClient
}

// Handle the Lambda request
func (h *RoomStateHandler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJson, _ := json.Marshal(request)
	log.Printf("Request received: %s", requestJson)

	// Extract room ID from path parameters
	roomID, exists := request.PathParameters["roomId"]
	if !exists || roomID == "" {
		log.Printf("Error: Missing roomId parameter")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Missing roomId parameter\"}",
		}, nil
	}

	// Construct the DynamoDB query
	input := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ROOM#" + roomID},
			"SK": &types.AttributeValueMemberS{Value: "STATE"},
		},
	}

	// Query DynamoDB
	result, err := h.DynamoDBClient.GetItem(ctx, input)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Error retrieving room state\"}",
		}, nil
	}

	// Check if item was found
	if len(result.Item) <= 0 {
		log.Printf("Room not found: %s", roomID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Room not found\"}",
		}, nil
	}

	// Unmarshal the DynamoDB item into RoomState
	var roomState RoomState
	err = attributevalue.UnmarshalMap(result.Item, &roomState)
	if err != nil {
		log.Printf("Error unmarshaling room state: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Error parsing room state\"}",
		}, nil
	}

	// Marshal the RoomState into JSON
	responseBody, err := json.Marshal(roomState)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Error generating response\"}",
		}, nil
	}

	// Return the successful response
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	handler := &RoomStateHandler{
		DynamoDBClient: dynamoDBClient,
	}

	// Start the Lambda handler
	lambda.Start(handler.HandleRequest)
}
