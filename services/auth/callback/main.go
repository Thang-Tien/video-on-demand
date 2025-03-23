package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	COGNITO_DOMAIN = "https://ap-southeast-2o0acq2nlj.auth.ap-southeast-2.amazoncognito.com"
	CLIENT_ID      = "caii6c0iji7rahi1dkltb8237"
	CLIENT_SECRET  = "1mupr6ngega59top0efm6l37t6f7dsl66cqnkn86olsg91h5iedu"
	REDIRECT_URI   = "https://tunv4f0t81.execute-api.ap-southeast-2.amazonaws.com/dev/auth/callback"
)

type Handler struct {
	Client *http.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (h *Handler) HandleCallback(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Query string: %v", request.QueryStringParameters)
	code := request.QueryStringParameters["code"]
	if code == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing code",
		}, nil
	}
	log.Printf("Code: %s", code)

	// Exchange code for token
	// Token endpoint for Cognito
	tokenURI := COGNITO_DOMAIN + "/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", CLIENT_ID)
	data.Set("code", code)
	data.Set("redirect_uri", REDIRECT_URI)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURI, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("Error creating token request: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error creating token request",
		}, nil
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authHeader := base64.StdEncoding.EncodeToString([]byte(CLIENT_ID + ":" + CLIENT_SECRET))
	req.Header.Set("Authorization", "Basic "+authHeader)

	// Send request
	resp, err := h.Client.Do(req)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error exchanging code for token",
		}, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Token endpoint returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Token endpoint returned non-200 status: %d", resp.StatusCode),
		}, nil
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading token response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error reading token response",
		}, nil
	}

	// Parse JSON response
	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		log.Printf("Error parsing token response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error parsing token response",
		}, nil
	}

	log.Printf("Token response: %v", body)

	// Return successful response with tokens
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil

}

func main() {
	handler := &Handler{
		Client: http.DefaultClient,
	}
	lambda.Start(handler.HandleCallback)
}
