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

func (h *CommentHandler) DeleteComment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Delete comment request: %s", requestJSON)

	videoID := request.PathParameters["videoId"]
	rawCommentID := request.PathParameters["commentId"]

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

	// Find the comment using a query with refined key condition
	queryResult, err := h.DynamoDBClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(h.CommentTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
		FilterExpression:       aws.String("contains(commentId, :commentId)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: "VIDEO#" + videoID},
			":skPrefix":  &types.AttributeValueMemberS{Value: "COMMENT#"},
			":commentId": &types.AttributeValueMemberS{Value: rawCommentID},
		},
	})

	if err != nil {
		log.Printf("Error querying comment: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to find comment",
		}, nil
	}

	if len(queryResult.Items) == 0 {
		return events.APIGatewayProxyResponse{StatusCode: 404,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Comment not found",
		}, nil
	}

	var comment Comment
	err = attributevalue.UnmarshalMap(queryResult.Items[0], &comment)
	if err != nil {
		log.Printf("Error unmarshalling comment: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
			Body: "Internal server error",
		}, nil
	}

	log.Printf("Comment found: %v", comment)

	// Verify the user is the owner of the comment
	if comment.UserID != userID {
		// Check if the user is an admin (could add admin check here)
		return events.APIGatewayProxyResponse{StatusCode: 403,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "You can only delete your own comments",
		}, nil
	}

	// Delete the comment
	_, err = h.DynamoDBClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(h.CommentTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: comment.PK},
			"SK": &types.AttributeValueMemberS{Value: comment.SK},
		},
	})

	if err != nil {
		log.Printf("Error deleting comment: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to delete comment",
		}, nil
	}

	// If this was a reply to another comment, decrement the parent's reply count
	if comment.ParentID != nil && *comment.ParentID != "" {
		// First, find the parent comment
		queryParentResult, err := h.DynamoDBClient.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(h.CommentTable),
			KeyConditionExpression: aws.String("PK = :pk AND SK = :sk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "VIDEO#" + videoID},
				":sk": &types.AttributeValueMemberS{Value: "COMMENT#" + *comment.ParentID},
			},
		})

		if err == nil && len(queryParentResult.Items) > 0 {
			var parentComment Comment
			if err = attributevalue.UnmarshalMap(queryParentResult.Items[0], &parentComment); err == nil {
				// Update the parent comment's reply count
				_, err = h.DynamoDBClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: aws.String(h.CommentTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: parentComment.PK},
						"SK": &types.AttributeValueMemberS{Value: parentComment.SK},
					},
					UpdateExpression:    aws.String("SET replyCount = if_not_exists(replyCount, :zero) - :one"),
					ConditionExpression: aws.String("replyCount > :zero"),
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":zero": &types.AttributeValueMemberN{Value: "0"},
						":one":  &types.AttributeValueMemberN{Value: "1"},
					},
				})
				if err != nil {
					log.Printf("Error updating parent comment reply count: %v", err)
					// Continue execution even if updating parent fails
				}
			}
		}


	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: `{"success": true}`,
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

	lambda.Start(handler.DeleteComment)

}
