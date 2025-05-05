package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

type ChatMessage struct {
	PK        string `json:"PK" dynamodbav:"PK"` // ROOM#<roomId>
	SK        string `json:"SK" dynamodbav:"SK"` // CHAT#<timestamp>#<userId>
	UserID    string `json:"userId" dynamodbav:"userId"`
	UserName  string `json:"userName" dynamodbav:"userName"`
	Content   string `json:"content" dynamodbav:"content"`
	Timestamp string `json:"timestamp" dynamodbav:"timestamp"`
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	roomID, exists := request.PathParameters["roomId"]
	if !exists || roomID == "" {
		log.Printf("Error: Missing roomId parameter")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Missing roomId parameter\"}",
		}, nil
	}

	// Construct the DynamoDB query to get all chat messages for a room
	input := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("DynamoDBTable")),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "ROOM#" + roomID},
			":sk": &types.AttributeValueMemberS{Value: "CHAT#"},
		},
		ScanIndexForward: aws.Bool(true), // Set to false for descending order (newest first)
	}

	// Execute the query
	result, err := h.DynamoDBClient.Query(ctx, input)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Failed to retrieve chat messages\"}",
		}, nil
	}

	// Parse the results into ChatMessage objects
	var messages []ChatMessage
	err = attributevalue.UnmarshalListOfMaps(result.Items, &messages)
	if err != nil {
		log.Printf("Error unmarshaling DynamoDB response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Failed to process chat messages\"}",
		}, nil
	}

	// Convert messages to JSON
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		log.Printf("Error marshaling chat messages: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "{\"error\": \"Failed to process chat messages\"}",
		}, nil
	}

	// Return the chat messages
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(messagesJSON),
	}, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS config: %v\n", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	handler := &Handler{
		DynamoDBClient: dynamoDBClient,
	}

	// Start the Lambda function
	lambda.Start(handler.HandleRequest)
}
