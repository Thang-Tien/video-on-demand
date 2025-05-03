package main

import (
	"context"
	"encoding/json"
	"log"
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

func (h *CommentHandler) GetComments(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Delete comment request: %s", requestJSON)

	// Get video ID from path parameter
	videoID := request.PathParameters["videoId"]

	// Check if we're looking for top-level comments or replies
	parentID := request.QueryStringParameters["parentId"]

	var expressionAttributeValues map[string]types.AttributeValue
	var keyConditionExpression string
	var filterExpression *string

	// Base query for video comments
	keyConditionExpression = "PK = :pk AND begins_with(SK, :sk)"
	expressionAttributeValues = map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: "VIDEO#" + videoID},
		":sk": &types.AttributeValueMemberS{Value: "COMMENT#"},
	}

	if parentID != "" {
		// Query for replies to a specific comment
		filterExp := "parentId = :parentId"
		filterExpression = &filterExp
		expressionAttributeValues[":parentId"] = &types.AttributeValueMemberS{Value: parentID}
	} else {
		// Query for top-level comments (no parent)
		filterExp := "attribute_not_exists(parentId)"
		filterExpression = &filterExp
	}

	// Query comments from DynamoDB
	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(h.CommentTable),
		KeyConditionExpression:    aws.String(keyConditionExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		FilterExpression:          filterExpression,
	}

	result, err := h.DynamoDBClient.Query(ctx, queryInput)

	if err != nil {
		log.Printf("Error querying comments: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to retrieve comments",
		}, nil
	}

	// Unmarshal the comments
	var comments []Comment
	err = attributevalue.UnmarshalListOfMaps(result.Items, &comments)
	if err != nil {
		log.Printf("Error unmarshalling comments: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Internal server error",
		}, nil
	}

	// Convert to response format
	var commentResponses []CommentResponse
	for _, comment := range comments {
		commentResponses = append(commentResponses, commentToResponse(comment))
	}

	responseBody, err := json.Marshal(commentResponses)
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
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
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

	lambda.Start(handler.GetComments)

}
