package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type Handler struct {
	CognitoClient CognitoClient
}

type CognitoClient interface {
	AdminAddUserToGroup(ctx context.Context, params *cognitoidentityprovider.AdminAddUserToGroupInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminAddUserToGroupOutput, error)
}

func (h *Handler) HandleRequest(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
	// Log the event for debugging
	eventJSON, _ := json.Marshal(event)
	log.Printf("Event: %s", eventJSON)

	// Get the user pool ID and username from the event
	userPoolID := event.UserPoolID
	username := event.UserName

	// Add the user to the "User" group
	_, err := h.CognitoClient.AdminAddUserToGroup(ctx, &cognitoidentityprovider.AdminAddUserToGroupInput{
		GroupName:  aws.String("user"),
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
	})
	if err != nil {
		log.Printf("Error adding user to user group: %v", err)
		return event, err
	}

	log.Printf("Successfully added user %s to user group", username)
	return event, nil
}

func main() {
	// Load the SDK's configuration from environment variables
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Error loading AWS config: %v", err)
	}

	// Create a Cognito service client
	cognitoClient := cognitoidentityprovider.NewFromConfig(cfg)

	handler := &Handler{
		CognitoClient: cognitoClient,
	}

	lambda.Start(handler.HandleRequest)
}
