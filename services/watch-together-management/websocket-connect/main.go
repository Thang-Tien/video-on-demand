package main

import (
	"context"
	"encoding/json"

	"fmt"
	"log"
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

// RoomState represents the current state of a watch room
type RoomState struct {
	RoomID        string    `json:"roomId"`
	LastStartTime time.Time `json:"lastStartTime"`
	LastVideoTime float64   `json:"lastVideoTime"`
	Status        string    `json:"status"` // "playing", "paused"
	VideoID       string    `json:"videoId"`
}

// Participant represents a user connected to a watch room
type Participant struct {
	PK           string    `dynamodbav:"PK"` // ROOM#{roomId}
	SK           string    `dynamodbav:"SK"` // PARTICIPANT#{participantId}
	RoomID       string    `dynamodbav:"roomId"`
	UserID       string    `dynamodbav:"userId"`
	ConnectionID string    `dynamodbav:"connectionId"`
	JoinedAt     time.Time `dynamodbav:"joinedAt"`
}

// IDynamoDBClient interface for DynamoDB operations
type IDynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// Handler is the WebSocket connect handler
type Handler struct {
	dynamoClient IDynamoDBClient
}

// Handle processes the WebSocket connect request
func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJson, _ := json.Marshal(request)
	fmt.Printf("Request: %s\n", requestJson)

	// Extract room ID from the request
	roomID := request.QueryStringParameters["roomID"]
	if roomID == "" {
		fmt.Println("Error: Room ID is missing from request")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Room ID is required",
		}, nil
	}
	fmt.Printf("Processing connection for room: %s\n", roomID)

	// Get user ID from Cognito
	var userID string
	if auth, ok := request.RequestContext.Authorizer.(map[string]interface{}); ok {
		userID = auth["sub"].(string)
	}

	if userID == "" {
		fmt.Println("Error: User ID is missing, unauthorized request")
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	// Check if room exists in DynamoDB
	roomKey := fmt.Sprintf("ROOM#%s", roomID)
	roomExists, err := h.checkRoomExists(ctx, roomKey)
	if err != nil {
		fmt.Printf("Error checking room existence: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error checking room existence",
		}, err
	}

	if !roomExists {
		fmt.Printf("Room does not exist: %s\n", roomID)
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       "Room not found",
		}, nil
	}

	// Get the connection ID
	connectionID := request.RequestContext.ConnectionID
	fmt.Printf("Connection ID: %s\n", connectionID)

	// Add participant to DynamoDB
	participant := Participant{
		PK:           fmt.Sprintf("ROOM#%s", roomID),
		SK:           fmt.Sprintf("PARTICIPANT#%s", userID),
		RoomID:       roomID,
		UserID:       userID,
		ConnectionID: connectionID,
		JoinedAt:     time.Now(),
	}

	err = h.addParticipant(ctx, participant)
	if err != nil {
		fmt.Printf("Error adding participant: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error adding participant to room",
		}, err
	}

	// Check if room state exists (but don't initialize it as it should have been created during room creation)
	roomStateExists, err := h.checkRoomStateExists(ctx, roomID)
	if err != nil {
		fmt.Printf("Error checking room state: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error checking room state",
		}, err
	}

	if !roomStateExists {
		fmt.Printf("Warning: Room state does not exist for room %s\n", roomID)
		// We don't initialize it here anymore, just log a warning
	}

	fmt.Printf("Successfully added participant %s to room %s\n", userID, roomID)
	// If successful, return 200
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Connected to room successfully",
	}, nil
}

// checkRoomExists checks if the room exists in DynamoDB
func (h *Handler) checkRoomExists(ctx context.Context, roomKey string) (bool, error) {
	fmt.Printf("Checking if room exists: %s\n", roomKey)

	result, err := h.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: roomKey},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})

	if err != nil {
		return false, fmt.Errorf("failed to query DynamoDB: %w", err)
	}

	// Room exists if the result item is not empty
	exists := len(result.Item) > 0
	fmt.Printf("Room exists: %v\n", exists)
	return exists, nil
}

// checkRoomStateExists checks if the room state exists in DynamoDB
func (h *Handler) checkRoomStateExists(ctx context.Context, roomID string) (bool, error) {
	roomKey := fmt.Sprintf("ROOM#%s", roomID)
	result, err := h.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: roomKey},
			"SK": &types.AttributeValueMemberS{Value: "STATE"},
		},
	})

	if err != nil {
		return false, fmt.Errorf("failed to query DynamoDB for room state: %w", err)
	}

	// Room state exists if the result item is not empty
	exists := len(result.Item) > 0
	fmt.Printf("Room state exists: %v\n", exists)
	return exists, nil
}

// addParticipant adds a participant to the DynamoDB table
func (h *Handler) addParticipant(ctx context.Context, participant Participant) error {
	fmt.Printf("Adding participant: %+v\n", participant)

	av, err := attributevalue.MarshalMap(participant)
	if err != nil {
		return fmt.Errorf("failed to marshal participant data: %w", err)
	}

	_, err = h.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to add participant to DynamoDB: %w", err)
	}

	return nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS config: %v\n", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	handler := &Handler{
		dynamoClient: dynamoClient,
	}

	lambda.Start(handler.HandleRequest)
}
