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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type CreateRoomRequest struct {
	RoomName        string `json:"roomName"`
	VideoId         string `json:"videoId"`
	MaxParticipants int    `json:"maxParticipants"`
}
type RoomData struct {
	PK              string    `json:"PK" dynamodbav:"PK"`
	SK              string    `json:"SK" dynamodbav:"SK"`
	RoomName        string    `json:"roomName" dynamodbav:"roomName"`
	CreatedBy       string    `json:"createdBy" dynamodbav:"createdBy"`
	CreatedAt       time.Time `json:"createdAt" dynamodbav:"createdAt"`
	VideoId         string    `json:"videoId" dynamodbav:"videoId"`
	MaxParticipants int       `json:"maxParticipants" dynamodbav:"maxParticipants"`
	Status          string    `json:"status" dynamodbav:"status"`
	ExpiresAt       time.Time `json:"expiresAt" dynamodbav:"expiresAt"`
	InviteLink      string    `json:"inviteLink" dynamodbav:"inviteLink"`
}

// RoomState represents the current state of a watch room
type RoomState struct {
	PK            string    `json:"PK" dynamodbav:"PK"`
	SK            string    `json:"SK" dynamodbav:"SK"`
	RoomID        string    `json:"roomId" dynamodbav:"roomId"`
	LastStartTime time.Time `json:"lastStartTime" dynamodbav:"lastStartTime"`
	LastVideoTime float64   `json:"lastVideoTime" dynamodbav:"lastVideoTime"`
	Status        string    `json:"status" dynamodbav:"status"` // "playing", "paused"
	VideoID       string    `json:"videoId" dynamodbav:"videoId"`
}

type Response struct {
	RoomId      string   `json:"roomId"`
	InviteLink  string   `json:"inviteLink"`
	RoomDetails RoomData `json:"roomDetails"`
}

type Handler struct {
	DynamoDBClient DynamoDBClient
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

type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// Extract user ID from principalId
	// The principalId is expected in format: vodapi::User::"ap-southeast-1_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613"
	var userId string
	if principalID, ok := request.RequestContext.Authorizer["principalId"].(string); ok {
		// Find the last pipe character and extract the UUID part
		if lastPipeIndex := strings.LastIndex(principalID, "|"); lastPipeIndex != -1 && lastPipeIndex < len(principalID)-1 {
			// Extract everything after the pipe
			userId = principalID[lastPipeIndex+1:]
			// Remove any trailing quotes if present
			userId = strings.Trim(userId, "\"")
		}
	}

	// If user ID couldn't be extracted, return bad request
	if userId == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Could not extract user ID from token",
		}, nil
	}

	// Parse request body
	var createRequest CreateRoomRequest
	err := json.Unmarshal([]byte(request.Body), &createRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request: %v", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	// Check if the video exists
	videoResult, err := h.DynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "VIDEO#" + createRequest.VideoId},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil || videoResult.Item == nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       fmt.Sprintf("Video with ID %s not found", createRequest.VideoId),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	// Create room ID
	roomId := uuid.New().String()

	room := RoomData{
		PK:              "ROOM#" + roomId,
		SK:              "METADATA",
		RoomName:        createRequest.RoomName,
		CreatedBy:       userId,
		CreatedAt:       time.Now(),
		VideoId:         "VIDEO#" + createRequest.VideoId,
		MaxParticipants: createRequest.MaxParticipants,
		Status:          "ACTIVE",
		ExpiresAt:       time.Now().Add(24 * time.Hour), // Rooms expire after 24 hours
	}

	av, err := attributevalue.MarshalMap(room)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error creating room: %v", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	_, err = h.DynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Item:      av,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error saving room to database: %v", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	// Initialize room state
	roomState := RoomState{
		PK:            "ROOM#" + roomId,
		SK:            "STATE",
		RoomID:        roomId,
		LastStartTime: time.Now(),
		LastVideoTime: 0,
		Status:        "paused",
		VideoID:       createRequest.VideoId,
	}

	stateAV, err := attributevalue.MarshalMap(roomState)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error creating room state: %v", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	_, err = h.DynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Item:      stateAV,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error saving room state to database: %v", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	// Create response
	response := Response{
		RoomId:      roomId,
		InviteLink:  os.Getenv("InviteDomain") + roomId,
		RoomDetails: room,
	}

	responseBody, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(responseBody),
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

	handler := Handler{
		DynamoDBClient: dynamoDBClient,
	}

	lambda.Start(handler.HandleRequest)
}
