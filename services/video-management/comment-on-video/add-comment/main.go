package main

import (
	"context"
	"encoding/json"
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

// Comment represents a comment on a video
type Comment struct {
	PK         string    `json:"PK" dynamodbav:"PK"` // VIDEO#<videoId>
	SK         string    `json:"SK" dynamodbav:"SK"` // COMMENT#<timestamp>#<uuid>
	VideoID    string    `json:"videoId" dynamodbav:"videoId"`
	CommentID  string    `json:"commentId" dynamodbav:"commentId"`
	UserID     string    `json:"userId" dynamodbav:"userId"`
	Content    string    `json:"content" dynamodbav:"content"`
	Timestamp  time.Time `json:"timestamp" dynamodbav:"timestamp"`
	LikeCount  int       `json:"likeCount" dynamodbav:"likeCount"`
	ReplyCount int       `json:"replyCount" dynamodbav:"replyCount"`
	ParentID   *string   `json:"parentId,omitempty" dynamodbav:"parentId,omitempty"` // For replies to comments
}

// CommentRequest is the structure for incoming comment requests
type CommentRequest struct {
	VideoID  string  `json:"videoId"`
	Content  string  `json:"content"`
	ParentID *string `json:"parentId,omitempty"`
}

// CommentResponse is the structure for comment responses with extracted comment ID
type CommentResponse struct {
	PK         string    `json:"PK"`
	SK         string    `json:"SK"`
	CommentID  string    `json:"commentId"`
	VideoID    string    `json:"videoId"`
	UserID     string    `json:"userId"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	LikeCount  int       `json:"likeCount"`
	ReplyCount int       `json:"replyCount"`
	ParentID   *string   `json:"parentId,omitempty"`
}

// DynamoDBAPI interface for DynamoDB operations
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

// CommentHandler contains all the dependencies for comment operations
type CommentHandler struct {
	DynamoDBClient DynamoDBAPI
	CommentTable   string
}

// extractCommentIDFromSK extracts the UUID part from the SK
func extractCommentIDFromSK(sk string) string {
	parts := strings.Split(sk, "#")
	if len(parts) >= 3 {
		return parts[2] // Return the UUID part
	}
	return ""
}

// commentToResponse converts a Comment to CommentResponse
func commentToResponse(comment Comment) CommentResponse {
	commentID := extractCommentIDFromSK(comment.SK)
	return CommentResponse{
		PK:         comment.PK,
		SK:         comment.SK,
		CommentID:  commentID,
		VideoID:    comment.VideoID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		Timestamp:  comment.Timestamp,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		ParentID:   comment.ParentID,
	}
}

// AddComment handles adding a new comment to a video
func (h *CommentHandler) AddComment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Delete comment request: %s", requestJSON)

	// Parse request body
	var commentRequest CommentRequest
	if err := json.Unmarshal([]byte(request.Body), &commentRequest); err != nil {
		log.Printf("Error unmarshalling comment request: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Invalid request body",
		}, nil
	}

	// Extract user ID from principalId
	// The principalId is expected in format: vodapi::User::"ap-southeast-1_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613"
	var userID string
	if principalID, ok := request.RequestContext.Authorizer["principalId"].(string); ok {
		// Find the last pipe character and extract the UUID part
		if lastPipeIndex := strings.LastIndex(principalID, "|"); lastPipeIndex != -1 && lastPipeIndex < len(principalID)-1 {
			// Extract everything after the pipe
			userID = principalID[lastPipeIndex+1:]
			// Remove any trailing quotes if present
			userID = strings.Trim(userID, "\"")
		}
	}

	// If user ID couldn't be extracted, return bad request
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Could not extract user ID from token",
		}, nil
	}

	// Create a new comment
	commentUUID := uuid.New().String()
	timestamp := time.Now()

	comment := Comment{
		PK:         "VIDEO#" + commentRequest.VideoID,
		SK:         "COMMENT#" + timestamp.Format(time.RFC3339Nano) + "#" + commentUUID,
		VideoID:    commentRequest.VideoID,
		CommentID:  commentUUID,
		UserID:     userID,
		Content:    commentRequest.Content,
		Timestamp:  timestamp,
		LikeCount:  0,
		ReplyCount: 0,
		ParentID:   commentRequest.ParentID,
	}

	// Marshal comment to DynamoDB format
	commentItem, err := attributevalue.MarshalMap(comment)
	if err != nil {
		log.Printf("Error marshalling comment: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Internal server error",
		}, nil
	}

	// Put comment in DynamoDB
	_, err = h.DynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(h.CommentTable),
		Item:      commentItem,
	})

	if err != nil {
		log.Printf("Error storing comment: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to save comment"}, nil
	}

	// Increment the ReplyCount of the parent comment if applicable
	if commentRequest.ParentID != nil {
		// Increment the ReplyCount of the parent comment
		updateExpression := "SET replyCount = replyCount + :inc"
		expressionAttributeValues := map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		}

		_, err := h.DynamoDBClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(h.CommentTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "VIDEO#" + commentRequest.VideoID},
				"SK": &types.AttributeValueMemberS{Value: "COMMENT#" + *commentRequest.ParentID},
			},
			UpdateExpression:          aws.String(updateExpression),
			ExpressionAttributeValues: expressionAttributeValues,
		})

		if err != nil {
			log.Printf("Error updating parent comment's reply count: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Access-Control-Allow-Origin": "*",
				},
				Body: "Failed to update parent comment's reply count",
			}, nil
		}
	}

	// Convert to response format
	response := commentToResponse(comment)

	// Return the created comment
	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Internal server error",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "application/json",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	log.Printf("Starting video comment handler")

	// Initialize AWS SDK config
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Create handler with dependencies
	handler := &CommentHandler{
		DynamoDBClient: dynamoClient,
		CommentTable:   os.Getenv("DynamoDBTable"),
	}

	lambda.Start(handler.AddComment)
}
