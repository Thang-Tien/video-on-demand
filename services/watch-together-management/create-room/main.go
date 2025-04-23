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
	"github.com/google/uuid"
)

type CreateRoomRequest struct {
	RoomName        string `json:"roomName"`
	VideoId         string `json:"videoId"`
	MaxParticipants int    `json:"maxParticipants"`
}

type RoomData struct {
	PK              string    `json:"PK"`
	SK              string    `json:"SK"`
	RoomName        string    `json:"roomName"`
	CreatedBy       string    `json:"createdBy"`
	CreatedAt       time.Time `json:"createdAt"`
	VideoId         string    `json:"videoId"`
	MaxParticipants int       `json:"maxParticipants"`
	Status          string    `json:"status"`
	ExpiresAt       time.Time `json:"expiresAt"`
	InviteLink      string    `json:"inviteLink"`
}

type Response struct {
	RoomId      string   `json:"roomId"`
	InviteLink  string   `json:"inviteLink"`
	RoomDetails RoomData `json:"roomDetails"`
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// Extract user ID from principalId
	// The principalId is expected in format: vodapi::User::"ap-southeast-2_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613"
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

	// Create room participant entry (host)
	participantItem := map[string]interface{}{
		"PK":       "ROOM#" + roomId,
		"SK":       "PARTICIPANT#" + userId,
		"joinedAt": time.Now().Format(time.RFC3339),
		"role":     "HOST",
	}

	av, err = attributevalue.MarshalMap(participantItem)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error creating room participant: %v", err),
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
			Body:       fmt.Sprintf("Error saving participant to database: %v", err),
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
		config.WithRegion("ap-southeast-2"),
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
