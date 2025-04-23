package main

import (
	"context"
	"crypto/tls"
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
	"github.com/redis/go-redis/v9"
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
	PK           string    `json:"PK"` // PARTICIPANT#{participantId}
	SK           string    `json:"SK"` // ROOM#{roomId}
	RoomID       string    `json:"roomId"`
	UserID       string    `json:"userId"`
	ConnectionID string    `json:"connectionId"`
	JoinedAt     time.Time `json:"joinedAt"`
}

// IDynamoDBClient interface for DynamoDB operations
type IDynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// IRedisClient interface for Redis operations
type IRedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// Handler is the WebSocket connect handler
type Handler struct {
	dynamoClient IDynamoDBClient
	redisClient  IRedisClient
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
		PK:           fmt.Sprintf("PARTICIPANT#%s", userID),
		SK:           fmt.Sprintf("ROOM#%s", roomID),
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

	// Get or initialize room state in Valkey
	err = h.initializeRoomState(ctx, roomID)
	if err != nil {
		fmt.Printf("Error initializing room state: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error initializing room state",
		}, err
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

// initializeRoomState gets or initializes the room state in Valkey
func (h *Handler) initializeRoomState(ctx context.Context, roomID string) error {
	fmt.Printf("Initializing room state for room: %s\n", roomID)

	// Check if room state exists in Valkey
	roomStateKey := fmt.Sprintf("room:%s", roomID)
	_, err := h.redisClient.Get(ctx, roomStateKey).Result()

	// If room state doesn't exist or there was an error other than redis.Nil
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error fetching room state from Valkey: %w", err)
	}

	// If room state doesn't exist, initialize it
	if err == redis.Nil {
		fmt.Println("Room state not found in Valkey, initializing new state")

		// Get room details from DynamoDB to get the videoID
		roomKey := fmt.Sprintf("ROOM#%s", roomID)
		result, err := h.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(os.Getenv("DynamoDBTable")),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: roomKey},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
		})

		if err != nil {
			return fmt.Errorf("failed to get room details from DynamoDB: %w", err)
		}

		// Extract videoID from the room item
		var videoID string
		if v, ok := result.Item["videoId"]; ok {
			if s, ok := v.(*types.AttributeValueMemberS); ok {
				videoID = s.Value
			}
		}

		// Initialize room state
		roomState := RoomState{
			RoomID:        roomID,
			LastStartTime: time.Now(),
			LastVideoTime: 0,
			Status:        "paused",
			VideoID:       videoID,
		}

		roomStateBytes, err := json.Marshal(roomState)
		if err != nil {
			return fmt.Errorf("failed to marshal room state: %w", err)
		}

		err = h.redisClient.Set(ctx, roomStateKey, string(roomStateBytes), 24*time.Hour).Err()
		if err != nil {
			return fmt.Errorf("failed to save room state to Valkey: %w", err)
		}

		fmt.Printf("Initialized new room state: %+v\n", roomState)
	} else {
		fmt.Println("Room state found in Valkey")
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

	// Create Redis client for Valkey
	valkeyEndpoint := os.Getenv("RedisEndpoint")
	if valkeyEndpoint == "" {
		log.Fatalf("Error: RedisEndpoint environment variable not set")
	}
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: valkeyEndpoint,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})
	
	// Test connection
	res, err := redisClient.Ping(context.TODO()).Result()
	if err != nil {
		log.Printf("Error connecting to Valkey: %v", err)
	}
	fmt.Printf("Connected to Valkey: %s\n", res)

	handler := &Handler{
		dynamoClient: dynamoClient,
		redisClient:  redisClient,
	}

	lambda.Start(handler.HandleRequest)
}
