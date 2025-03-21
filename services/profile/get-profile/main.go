package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type CognitoClient interface {
	GetUser(ctx context.Context, params *cognitoidentityprovider.GetUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.GetUserOutput, error)
}

type Handler struct {
	CognitoClient CognitoClient
}

func (h *Handler) GetProfile(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract the access token from the Authorization header
	accessToken := ""
	if authHeader, ok := request.Headers["Authorization"]; ok {
		// The Authorization header typically has format "Bearer <token>"
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			accessToken = authHeader[7:]
		}
	}

	// If the access token is not provided, return unauthorized
	if accessToken == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Access token not provided",
		}, nil
	}

	params := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	}

	// Get user details from Cognito using the identity ID
	userInfo, err := h.CognitoClient.GetUser(ctx, params)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error retrieving user information: %v", err),
		}, nil
	}

	// Marshal the user info to JSON
	responseBody, err := json.Marshal(userInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Error creating response",
		}, nil
	}

	// Return the response
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	handler := &Handler{
		CognitoClient: cognitoidentityprovider.NewFromConfig(aws.Config{
			Region: "ap-southeast-2",
		}),
	}
	lambda.Start(handler.GetProfile)
}
