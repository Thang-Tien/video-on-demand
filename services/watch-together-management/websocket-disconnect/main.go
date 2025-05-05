package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type IDynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

type Handler struct {
	DynamoClient IDynamoDBClient
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the request
	requestJson, _ := json.Marshal(request)
	fmt.Printf("Received WebSocket disconnect request: %s\n", requestJson)

	// Get the connection ID
	connectionID := request.RequestContext.ConnectionID
	fmt.Printf("Processing disconnect for connection: %s\n", connectionID)

	// Find participant by connection ID
	participant, err := h.findParticipantByConnectionID(ctx, connectionID)
	if err != nil {
		fmt.Printf("Error finding participant: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error finding participant",
		}, err
	}

	// Response body with PK and SK
	responseBody := fmt.Sprintf("Disconnected successfully. PK: %s, SK: %s", participant.PK, participant.SK)

	// If participant exists, delete it
	if participant.PK != "" {
		err = h.deleteParticipant(ctx, participant.PK, participant.SK)
		if err != nil {
			fmt.Printf("Error deleting participant: %v\n", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Error deleting participant",
			}, err
		}
		fmt.Printf("Participant deleted: %s, %s\n", participant.PK, participant.SK)
	} else {
		fmt.Printf("No participant found for connection ID: %s\n", connectionID)
		responseBody = "Disconnected successfully. No participant found."
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       responseBody,
	}, nil
}

// Participant represents a user connected to a watch room
type Participant struct {
	PK           string `json:"PK" dynamodbav:"PK"`
	SK           string `json:"SK" dynamodbav:"SK"`
	RoomID       string `json:"roomId" dynamodbav:"roomId"`
	UserID       string `json:"userId" dynamodbav:"userId"`
	ConnectionID string `json:"connectionId" dynamodbav:"connectionId"`
}

// findParticipantByConnectionID finds a participant by connection ID
func (h *Handler) findParticipantByConnectionID(ctx context.Context, connectionID string) (Participant, error) {
	fmt.Printf("Finding participant with connection ID: %s\n", connectionID)

	// Since there's no index, scan the table to find the participant by connection ID
	// Use the exact attribute name as it appears in DynamoDB
	result, err := h.DynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(os.Getenv("DynamoDBTable")),
		FilterExpression: aws.String("connectionId = :connectionId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})

	if err != nil {
		return Participant{}, fmt.Errorf("failed to scan participants: %w", err)
	}

	fmt.Printf("Scan result: %d items found\n", len(result.Items))

	// Check if any participant was found
	if len(result.Items) == 0 {
		fmt.Printf("No participant found with connection ID: %s\n", connectionID)
		return Participant{}, nil
	}

	// Extract participant data
	var participant Participant
	err = attributevalue.UnmarshalMap(result.Items[0], &participant)
	if err != nil {
		return Participant{}, fmt.Errorf("failed to unmarshal participant data: %w", err)
	}

	fmt.Printf("Found participant: PK=%s, SK=%s, ConnectionID=%s\n",
		participant.PK, participant.SK, participant.ConnectionID)
	return participant, nil
}

// deleteParticipant deletes a participant from DynamoDB
func (h *Handler) deleteParticipant(ctx context.Context, pk, sk string) error {
	fmt.Printf("Deleting participant: %s, %s\n", pk, sk)

	_, err := h.DynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete participant: %w", err)
	}

	return nil
}

func main() {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS config: %v\n", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	handler := Handler{
		DynamoClient: dynamoClient,
	}

	// Handle the request
	lambda.Start(handler.HandleRequest)
}
