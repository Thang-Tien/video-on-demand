package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	apigwtype "github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type VideoEvent struct {
	Action    string  `json:"action"` // "play", "pause", "seek"
	RoomID    string  `json:"roomId"`
	VideoTime float64 `json:"videoTime"` // Current time in the video
	Timestamp int64   `json:"timestamp"` // Client-side timestamp
}

type RoomState struct {
	RoomID        string    `json:"roomId"`
	LastStartTime time.Time `json:"lastStartTime"`
	LastVideoTime float64   `json:"lastVideoTime"`
	Status        string    `json:"status"` // "playing", "paused"
	VideoID       string    `json:"videoId"`
}

type Participant struct {
	PK           string    `dynamodbav:"PK"` // ROOM#{roomId}
	SK           string    `dynamodbav:"SK"` // PARTICIPANT#{participantId}
	RoomID       string    `dynamodbav:"roomId"`
	UserID       string    `dynamodbav:"userId"`
	ConnectionID string    `dynamodbav:"connectionId"`
	JoinedAt     time.Time `dynamodbav:"joinedAt"`
}

type IDynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type IAPIGatewayClient interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

// Handler is the WebSocket message handler
type Handler struct {
	DynamoClient     IDynamoDBClient
	ApiGatewayClient IAPIGatewayClient
}

// Handle processes the WebSocket message
func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the request
	requestJSON, _ := json.Marshal(request)
	fmt.Printf("Received WebSocket message: %s\n", requestJSON)

	// Parse the message
	var videoEvent VideoEvent
	err := json.Unmarshal([]byte(request.Body), &videoEvent)
	if err != nil {
		fmt.Printf("Error parsing message: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid message format",
		}, nil
	}

	// Validate the message
	if videoEvent.RoomID == "" {
		fmt.Println("Error: Room ID is missing from message")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Room ID is required",
		}, nil
	}

	// Get sender connection ID
	senderConnectionID := request.RequestContext.ConnectionID
	fmt.Printf("Sender connection ID: %s\n", senderConnectionID)

	// Update room state in Valkey
	err = h.updateRoomState(ctx, videoEvent)
	if err != nil {
		fmt.Printf("Error updating room state: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error updating room state",
		}, err
	}

	// Get participants in the room
	participants, err := h.getParticipants(ctx, videoEvent.RoomID)
	if err != nil {
		fmt.Printf("Error getting participants: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error getting participants",
		}, err
	}

	// Broadcast the message to all participants except the sender
	messageJSON, err := json.Marshal(videoEvent)
	if err != nil {
		fmt.Printf("Error marshaling message: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error marshaling message",
		}, err
	}

	fmt.Printf("Broadcasting message to %d participants\n", len(participants))
	for _, participant := range participants {
		// Skip the sender
		if participant.ConnectionID == senderConnectionID {
			continue
		}

		// Send the message
		err = h.sendMessage(ctx, participant.ConnectionID, messageJSON)
		if err != nil {
			// Just log the error and continue with other participants
			fmt.Printf("Error sending message to %s: %v\n", participant.ConnectionID, err)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

// updateRoomState updates the room state in Valkey
func (h *Handler) updateRoomState(ctx context.Context, event VideoEvent) error {
	fmt.Printf("Updating room state for room %s: %s at %f\n", event.RoomID, event.Action, event.VideoTime)

	// Define the room state key
	roomKey := fmt.Sprintf("ROOM#%s", event.RoomID)

	// Fetch the current room state from DynamoDB
	result, err := h.DynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: roomKey},
			"SK": &types.AttributeValueMemberS{Value: "STATE"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get room state from DynamoDB: %w", err)
	}

	// Parse the room state
	var roomState RoomState
	if len(result.Item) > 0 {
		err = attributevalue.UnmarshalMap(result.Item, &roomState)
		if err != nil {
			return fmt.Errorf("failed to parse room state: %w", err)
		}
	} else {
		// Initialize a new room state if none exists
		roomState = RoomState{
			RoomID: event.RoomID,
			Status: "paused",
		}
	}

	// Update room state based on the event
	switch event.Action {
	case "play":
		roomState.Status = "playing"
		roomState.LastStartTime = time.Now()
		roomState.LastVideoTime = event.VideoTime
	case "pause":
		roomState.Status = "paused"
		roomState.LastVideoTime = event.VideoTime
	case "seek":
		roomState.LastVideoTime = event.VideoTime
		// If video was already playing, update the start time
		if roomState.Status == "playing" {
			roomState.LastStartTime = time.Now()
		}
	default:
		return fmt.Errorf("unknown action: %s", event.Action)
	}

	// Save the updated room state to DynamoDB
	updatedRoomState, err := attributevalue.MarshalMap(roomState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated room state: %w", err)
	}

	// Add required DynamoDB keys
	updatedRoomState["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("ROOM#%s", roomState.RoomID)}
	updatedRoomState["SK"] = &types.AttributeValueMemberS{Value: "STATE"}

	_, err = h.DynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Item:      updatedRoomState,
	})
	if err != nil {
		return fmt.Errorf("failed to save updated room state to DynamoDB: %w", err)
	}

	fmt.Printf("Room state updated: %+v\n", roomState)
	return nil
}

// getParticipants gets all participants in a room
func (h *Handler) getParticipants(ctx context.Context, roomID string) ([]Participant, error) {
	fmt.Printf("Getting participants for room: %s\n", roomID)

	// Query participants by room
	result, err := h.DynamoClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("DynamoDBTable")),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":       &types.AttributeValueMemberS{Value: fmt.Sprintf("ROOM#%s", roomID)},
			":skPrefix": &types.AttributeValueMemberS{Value: "PARTICIPANT#"},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}

	// Unmarshal participants
	var participants []Participant
	for _, item := range result.Items {
		var participant Participant
		err := attributevalue.UnmarshalMap(item, &participant)
		if err != nil {
			fmt.Printf("Error unmarshaling participant: %v\n", err)
			continue
		}

		participants = append(participants, participant)
	}

	fmt.Printf("Found %d participants\n", len(participants))
	return participants, nil
}

// sendMessage sends a message to a connection
func (h *Handler) sendMessage(ctx context.Context, connectionID string, message []byte) error {
	fmt.Printf("Sending message to connection: %s\n", connectionID)

	_, err := h.ApiGatewayClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         message,
	})
	if err != nil {
		// Log detailed error
		log.Printf("Error sending to connection %s: %v", connectionID, err)

		// Check for specific error types
		var forbiddenException *apigwtype.ForbiddenException
		var goneException *apigwtype.GoneException

		if errors.As(err, &forbiddenException) {
			message := "Unknown error"
			if forbiddenException.Message != nil {
				message = *forbiddenException.Message
			}
			log.Printf("Permission denied. Check IAM policies: %s", message)
		} else if errors.As(err, &goneException) {
			log.Printf("Connection no longer available, removing from state")
			// Remove connection from your connections store
		}

		return err
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

	// Create API Gateway Management API client
	endpoint := os.Getenv("WebSocketEndpoint")
	log.Printf("WebSocket endpoint: %s\n", endpoint)
	apiGatewayClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	handler := Handler{
		DynamoClient:     dynamoClient,
		ApiGatewayClient: apiGatewayClient,
	}

	lambda.Start(handler.HandleRequest)
}
