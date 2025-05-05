package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

// CognitoAPI defines the interface for Cognito operations
type CognitoAPI interface {
	GetUser(ctx context.Context, params *cognito.GetUserInput, optFns ...func(*cognito.Options)) (*cognito.GetUserOutput, error)
}

// AuthorizerHandler handles the custom authorizer logic
type AuthorizerHandler struct {
	cognitoClient CognitoAPI
	userPoolID    string
}

// WebSocketAuthorizerResponse represents the structure for API Gateway WebSocket authorizer
type WebSocketAuthorizerResponse struct {
	PrincipalID    string                 `json:"principalId"`
	PolicyDocument PolicyDocument         `json:"policyDocument"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// PolicyDocument represents the IAM policy for the authorizer
type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a statement in the IAM policy
type Statement struct {
	Action   string `json:"Action"`
	Effect   string `json:"Effect"`
	Resource string `json:"Resource"`
}

// NewAuthorizerHandler creates a new instance of AuthorizerHandler
func NewAuthorizerHandler(cognitoClient CognitoAPI, userPoolID string) *AuthorizerHandler {
	return &AuthorizerHandler{
		cognitoClient: cognitoClient,
		userPoolID:    userPoolID,
	}
}

// HandleRequest processes the authorizer request
func (h *AuthorizerHandler) HandleRequest(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	// Log the incoming request
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s\n", requestJSON)

	// Extract token from the request
	token := extractToken(request.Headers["Authorization"])
	if token == "" {
		// If token is not in the header, check the query parameters
		token = request.QueryStringParameters["token"]
		if token == "" {
			log.Println("Empty authorization token")
			return generateDenyPolicy("user", request.MethodArn), nil
		}
	}

	// Validate token with Cognito
	userSub, err := h.validateCognitoToken(ctx, token)
	if err != nil {
		log.Printf("Token validation error: %v", err)
		return generateDenyPolicy("user", request.MethodArn), nil
	}

	// Get the account ID from the ARN
	arnParts := strings.Split(request.MethodArn, ":")
	if len(arnParts) < 6 {
		log.Printf("Invalid ARN format: %s", request.MethodArn)
		return generateDenyPolicy("user", request.MethodArn), nil
	}

	// Create the resource pattern for the policy
	// This allows access to all WebSocket routes (connect, disconnect, default)
	resourcePattern := strings.Replace(request.MethodArn, request.MethodArn[strings.LastIndex(request.MethodArn, "/"):], "/*", 1)

	// If token is valid, generate an Allow policy
	return generateAllowPolicy(userSub, resourcePattern, userSub), nil
}

// validateCognitoToken validates the provided token with Cognito
func (h *AuthorizerHandler) validateCognitoToken(ctx context.Context, token string) (string, error) {
	// Use GetUser API to validate the token
	input := &cognito.GetUserInput{
		AccessToken: aws.String(token),
	}

	result, err := h.cognitoClient.GetUser(ctx, input)
	if err != nil {
		return "", err
	}

	// Extract and return the user's sub (unique identifier)
	for _, attr := range result.UserAttributes {
		if *attr.Name == "sub" {
			return *attr.Value, nil
		}
	}

	return "", errors.New("user sub not found in token")
}

// extractToken extracts the token from the authorization header
func extractToken(authHeader string) string {
	// For Bearer token format: "Bearer <token>"
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return authHeader
}

// generateAllowPolicy generates an IAM policy that allows access
func generateAllowPolicy(principalID string, resource string, userSub string) events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   "Allow",
					Resource: []string{resource},
				},
			},
		},
		Context: map[string]interface{}{
			"sub":    userSub,
			"userId": userSub,
		},
	}
}

// generateDenyPolicy generates an IAM policy that denies access
func generateDenyPolicy(principalID string, resource string) events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   "Deny",
					Resource: []string{resource},
				},
			},
		},
	}
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-southeast-1"),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	cognitoClient := cognito.NewFromConfig(cfg)

	userPoolID := "ap-southeast-1_o0aCq2NlJ"
	handler := NewAuthorizerHandler(cognitoClient, userPoolID)

	lambda.Start(handler.HandleRequest)
}
