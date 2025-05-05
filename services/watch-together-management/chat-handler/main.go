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
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	apigwtype "github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type APIGatewayClient interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

type CognitoClient interface {
	ListUsers(ctx context.Context, params *cognitoidentityprovider.ListUsersInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.ListUsersOutput, error)
}

type Handler struct {
	DynamoDBClient   DynamoDBClient
	ApiGatewayClient APIGatewayClient
	CognitoClient    CognitoClient
}

type ChatMessage struct {
	PK        string    `json:"PK" dynamodbav:"PK"` // ROOM#<roomId>
	SK        string    `json:"SK" dynamodbav:"SK"` // CHAT#<timestamp>#<userId>
	UserID    string    `json:"userId" dynamodbav:"userId"`
	UserName  string    `json:"userName" dynamodbav:"userName"`
	Content   string    `json:"content" dynamodbav:"content"`
	Timestamp time.Time `json:"timestamp" dynamodbav:"timestamp"`
}

type WebSocketMessage struct {
	Action  string `json:"action"`
	Content string `json:"content,omitempty"`
}

type Participant struct {
	PK           string    `dynamodbav:"PK"` // ROOM#{roomId}
	SK           string    `dynamodbav:"SK"` // PARTICIPANT#{participantId}
	RoomID       string    `dynamodbav:"roomId"`
	UserID       string    `dynamodbav:"userId"`
	ConnectionID string    `dynamodbav:"connectionId"`
	JoinedAt     time.Time `dynamodbav:"joinedAt"`
}

type BroadcastChatMessage struct {
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// Message handling in your WebSocket handler
func (h *Handler) HandleChat(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJson, _ := json.Marshal(request)
	fmt.Printf("Request: %s\n", requestJson)

	connectionID := request.RequestContext.ConnectionID

	var message WebSocketMessage

	if err := json.Unmarshal([]byte(request.Body), &message); err != nil {
		log.Printf("Error parsing message: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	// Verify this is a chat message
	if message.Action != "chat" {
		// Handle other actions
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	// Get connection info from DynamoDB to know which room and user
	participant, err := h.getParticipants(ctx, connectionID)
	if err != nil {
		log.Printf("Error getting connection info: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	userName, err := h.getUserName(ctx, participant.UserID)
	if err != nil {
		log.Printf("Error getting user name: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Create the chat message
	now := time.Now()
	chatMsg := ChatMessage{
		PK:        "ROOM#" + participant.RoomID,
		SK:        "CHAT#" + now.Format(time.RFC3339Nano) + "_" + participant.UserID,
		UserID:    participant.UserID,
		UserName:  userName,
		Content:   message.Content,
		Timestamp: now,
	}

	// Store in DynamoDB
	item, err := attributevalue.MarshalMap(chatMsg)
	if err != nil {
		log.Printf("Error marshaling chat message: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	_, err = h.DynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Item:      item,
	})
	if err != nil {
		log.Printf("Error storing chat message: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Broadcast to all connections in the room
	broadcastChatMsg := BroadcastChatMessage{
		UserID:    participant.UserID,
		UserName:  userName,
		Content:   message.Content,
		Timestamp: now.Format(time.RFC3339),
	}

	if err := h.broadcastToRoom(ctx, participant.RoomID, broadcastChatMsg); err != nil {
		log.Printf("Error broadcasting message: %v", err)
		// Continue anyway - the message is stored in DynamoDB
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func (h *Handler) getParticipants(ctx context.Context, connectionID string) (*Participant, error) {
	// Query DynamoDB for the participant with the given connection ID
	input := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("DynamoDBTable")),
		IndexName:              aws.String("connectionId-index"),
		KeyConditionExpression: aws.String("connectionId = :connectionId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
		Limit: aws.Int32(1),
	}

	output, err := h.DynamoDBClient.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error querying DynamoDB: %w", err)
	}

	if len(output.Items) == 0 {
		return nil, fmt.Errorf("no participant found for connection ID: %s", connectionID)
	}

	var participant Participant
	err = attributevalue.UnmarshalMap(output.Items[0], &participant)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling DynamoDB item: %w", err)
	}

	return &participant, nil
}

func (h *Handler) getUserName(ctx context.Context, userID string) (string, error) {
	input := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(os.Getenv("CognitoUserPoolID")),
		Filter:     aws.String(fmt.Sprintf("sub = \"%s\"", userID)),
		Limit:      aws.Int32(1),
	}

	output, err := h.CognitoClient.ListUsers(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error calling ListUsers: %w", err)
	}

	if len(output.Users) == 0 {
		return "", fmt.Errorf("no user found for user ID: %s", userID)
	}

	if output.Users[0].Username != nil {
		return *output.Users[0].Username, nil
	}

	return "", fmt.Errorf("user name not found for user ID: %s", userID)
}

func (h *Handler) broadcastToRoom(ctx context.Context, roomID string, message BroadcastChatMessage) error {
	// Query DynamoDB to get all participants in the room
	input := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("DynamoDBTable")),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "ROOM#" + roomID},
			":sk": &types.AttributeValueMemberS{Value: "PARTICIPANT#"},
		},
	}

	result, err := h.DynamoDBClient.Query(ctx, input)
	if err != nil {
		return fmt.Errorf("error querying participants: %w", err)
	}

	if len(result.Items) == 0 {
		return fmt.Errorf("no participants found in room: %s", roomID)
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling broadcast message: %w", err)
	}

	// Broadcast to each participant's connection
	var broadcastErrors []error
	for _, item := range result.Items {
		var participant Participant
		if err := attributevalue.UnmarshalMap(item, &participant); err != nil {
			broadcastErrors = append(broadcastErrors, fmt.Errorf("error unmarshaling participant: %w", err))
			continue
		}

		// Skip if connection ID is empty
		if participant.ConnectionID == "" {
			continue
		}

		_, err := h.ApiGatewayClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(participant.ConnectionID),
			Data:         payload,
		})

		if err != nil {
			// Check for stale connection
			if _, ok := err.(*apigwtype.GoneException); ok {
				log.Printf("Stale connection: %s", participant.ConnectionID)
				// Could delete the stale connection here if needed
				continue
			}
			broadcastErrors = append(broadcastErrors, fmt.Errorf("error sending to connection %s: %w",
				participant.ConnectionID, err))
		}
	}

	// Return combined errors if any
	if len(broadcastErrors) > 0 {
		errMsg := fmt.Sprintf("%d broadcast errors occurred:", len(broadcastErrors))
		for i, err := range broadcastErrors {
			if i < 5 { // Limit number of errors in message to avoid excessive logs
				errMsg += fmt.Sprintf("\n  - %s", err.Error())
			} else {
				errMsg += "\n  - and more errors..."
				break
			}
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS config: %v\n", err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Create API Gateway Management API client
	endpoint := os.Getenv("WebSocketEndpoint")
	log.Printf("WebSocket endpoint: %s\n", endpoint)
	apiGatewayClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	cognitoClient := cognitoidentityprovider.NewFromConfig(cfg)

	handler := Handler{
		DynamoDBClient:   dynamoClient,
		ApiGatewayClient: apiGatewayClient,
		CognitoClient:    cognitoClient,
	}

	lambda.Start(handler.HandleChat)
}
