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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3PresignClient interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type GetPresignedURLHandler struct {
	s3PresignClient S3PresignClient
}

type Response struct {
	UploadURL string    `json:"uploadUrl"`
	ExpiresIn time.Time `json:"expiresIn"`
}

func (h *GetPresignedURLHandler) HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	ctx := context.Background()

	// Extract file name from query parameters
	fileName, ok := request.QueryStringParameters["fileName"]
	if !ok || fileName == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "File name not provided in query parameters",
		}, nil
	}

	// Extract user ID from principalId
	// The principalId is expected in format: vodapi::User::"ap-southeast-2_opagHcslJ|996ev488-21f1-7gc6-da0f-28ag6acb3613"
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

	// Generate the S3 key (path) where the video will be stored
	s3Key := userID + "/" + uuid.New().String() + "/" + fileName
	// Get the bucket name from environment variable
	bucket := os.Getenv("Source")

	// Create the presigned URL request
	putObjectInput := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &s3Key,
	}

	// Set expiration time for the URL (15 minutes)
	presignOptions := func(po *s3.PresignOptions) {
		po.Expires = 15 * time.Minute
	}
	// Generate the presigned URL
	presignedReq, err := h.s3PresignClient.PresignPutObject(ctx, putObjectInput, presignOptions)
	if err != nil {
		log.Printf("Error creating presigned URL: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to generate upload URL: " + err.Error(),
		}, nil
	}

	// Construct the response
	response := Response{
		UploadURL: presignedReq.URL,
		ExpiresIn: time.Now().Add(15 * time.Minute),
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Body: "Failed to generate response",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(responseJSON),
	}, nil
}

func main() {
	// Create an S3 client
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-southeast-2"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	// Create a presign client from the S3 client
	s3PresignClient := s3.NewPresignClient(s3Client)

	handler := GetPresignedURLHandler{
		s3PresignClient: s3PresignClient,
	}

	// Start the Lambda function
	lambda.Start(handler.HandleRequest)
}
