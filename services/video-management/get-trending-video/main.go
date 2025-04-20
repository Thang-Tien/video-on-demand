package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	DEFAULT_PAGE_SIZE = 5
	MAX_PAGE_SIZE     = 20
	TOKEN_EXPIRY_MINS = 60 // Token expiry time in minutes
)

var errInvalidPageNumber = errors.New("invalid page number")

type Views struct {
	PK      string `json:"PK"`
	SK      string `json:"SK"`
	VideoID string `json:"videoId"`
}

type Response struct {
	Videos     []Views `json:"views"`
	Count      int     `json:"count"`
	TotalCount int     `json:"totalCount,omitempty"` // Optional, requires a separate count query
	NextToken  string  `json:"nextToken,omitempty"`
	PageNumber int     `json:"pageNumber"`
	PagesCount int     `json:"pagesCount,omitempty"` // Optional, requires knowing total count
}

type PaginationToken struct {
	LastEvaluatedKey map[string]string `json:"lastEvaluatedKey"`
	ExpiresAt        int64             `json:"expiresAt"`
	PageNumber       int               `json:"pageNumber,omitempty"`
}

type DynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// TODO: Validate request

	// get query parameters
	// get page number from query parameters
	pageNumber, _ := strconv.Atoi(request.QueryStringParameters["pageNumber"])
	if pageNumber <= 0 {
		pageNumber = 1
	}

	pageSize, _ := strconv.Atoi(request.QueryStringParameters["pageSize"])
	if pageSize <= 0 {
		pageSize = DEFAULT_PAGE_SIZE
	} else if pageSize > MAX_PAGE_SIZE {
		pageSize = MAX_PAGE_SIZE
	}

	nextToken := request.QueryStringParameters["nextToken"]

	monthAndYear := request.QueryStringParameters["monthAndYear"]

	// if page number is provided but not nextToken, query to that page
	var lastEvaluatedKey map[string]types.AttributeValue
	if nextToken == "" {
		var err error
		lastEvaluatedKey, err = h.queryToPage(ctx, 1, pageNumber, pageSize, monthAndYear, nil)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       fmt.Sprintf("Failed to query to page: %v", err),
				Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
			}, nil
		}
	} else if nextToken != "" {
		// Decode the nextToken to get the last evaluated key
		paginationToken, err := decodeNextToken(nextToken)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       fmt.Sprintf("Invalid nextToken format: %v", err),
				Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
			}, nil
		}

		// Check if the token is expired
		if time.Now().Unix() > paginationToken.ExpiresAt {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "nextToken has expired",
				Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
			}, nil
		}

		// Create the last evaluated key from the decoded token
		tokenLastEvaluatedKey := map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: paginationToken.LastEvaluatedKey["PK"]},
			"SK": &types.AttributeValueMemberS{Value: paginationToken.LastEvaluatedKey["SK"]},
		}

		if paginationToken.PageNumber > pageNumber {
			lastEvaluatedKey, err = h.queryToPage(ctx, 1, pageNumber, pageSize, monthAndYear, nil)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       fmt.Sprintf("Failed to query to page: %v", err),
					Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
				}, nil
			}
		} else {
			lastEvaluatedKey, err = h.queryToPage(ctx, paginationToken.PageNumber, pageNumber, pageSize, monthAndYear, tokenLastEvaluatedKey)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       fmt.Sprintf("Failed to query to page: %v", err),
					Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
				}, nil
			}
		}
	}

	expr, err := buildKeyConditionExpression(monthAndYear)
	if err != nil {
		log.Printf("Failed to build key condition expression: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to build key condition expression: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("DynamoDBTable")),
		Limit:                     aws.Int32(int32(pageSize)),
		ExclusiveStartKey:         lastEvaluatedKey,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(false), // Descending order
	}

	jsonInput, _ := json.Marshal(queryInput)
	log.Printf("QueryInput: %s", jsonInput)

	result, err := h.DynamoDBClient.Query(ctx, queryInput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to query DynamoDB: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	var views []Views
	err = attributevalue.UnmarshalListOfMaps(result.Items, &views)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to unmarshal items: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	response := Response{
		Videos:     views,
		Count:      len(views),
		PageNumber: pageNumber,
	}

	// Add nextToken if there are more results
	if result.LastEvaluatedKey != nil {
		lastKey := PaginationToken{
			LastEvaluatedKey: make(map[string]string),
			ExpiresAt:        time.Now().Add(TOKEN_EXPIRY_MINS * time.Minute).Unix(),
			PageNumber:       pageNumber + 1,
		}
		if pk, ok := result.LastEvaluatedKey["PK"].(*types.AttributeValueMemberS); ok {
			lastKey.LastEvaluatedKey["PK"] = pk.Value
		}
		if sk, ok := result.LastEvaluatedKey["SK"].(*types.AttributeValueMemberS); ok {
			lastKey.LastEvaluatedKey["SK"] = sk.Value
		}

		response.NextToken, err = encodeNextToken(lastKey)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       fmt.Sprintf("Failed to encode nextToken: %v", err),
				Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
			}, nil
		}
	}

	// Convert response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
			Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
		}, nil
	}

	log.Printf("Response: %s", responseJSON)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJSON),
		Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
	}, nil
}

func (h *Handler) queryToPage(ctx context.Context, pageStart, pageNumber, pageSize int, monthAndYear string, lastEvaluatedKey map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	for i := pageStart; i < pageNumber; i++ {
		expr, err := buildKeyConditionExpression(monthAndYear)
		if err != nil {
			log.Printf("Failed to build key condition expression: %v", err)
			return nil, err
		}

		queryInput := &dynamodb.QueryInput{
			TableName:                 aws.String(os.Getenv("DynamoDBTable")),
			Limit:                     aws.Int32(int32(pageSize)),
			ExclusiveStartKey:         lastEvaluatedKey,
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			ScanIndexForward:          aws.Bool(false), // Descending order
		}

		result, err := h.DynamoDBClient.Query(ctx, queryInput)
		if err != nil {
			return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			return nil, errInvalidPageNumber
		}
	}

	return lastEvaluatedKey, nil
}

func buildKeyConditionExpression(monthAndYear string) (expression.Expression, error) {
	var keyCondition expression.KeyConditionBuilder

	if monthAndYear != "" {
		// Query for period-based trending videos
		keyCondition = expression.Key("PK").Equal(expression.Value(fmt.Sprintf("PERIOD#%s", monthAndYear))).
			And(expression.Key("SK").BeginsWith("VIEWS#"))
	} else {
		// Default to show most recent period if not specified
		currentTime := time.Now()
		defaultPeriod := fmt.Sprintf("%d-%02d", currentTime.Year(), currentTime.Month())

		keyCondition = expression.Key("PK").Equal(expression.Value(fmt.Sprintf("PERIOD#%s", defaultPeriod))).
			And(expression.Key("SK").BeginsWith("VIEWS#"))
	}

	// Return final expression
	return expression.NewBuilder().WithKeyCondition(keyCondition).Build()
}

// encodeNextToken encrypts and base64 encodes a pagination token
func encodeNextToken(token PaginationToken) (string, error) {
	var encryptionKey = []byte(os.Getenv("EncryptionKey"))

	// Marshal the token to JSON
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	// Create a GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, tokenBytes, nil)

	// Base64 encode the result
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decodeNextToken decrypts and decodes a base64 encoded pagination token
func decodeNextToken(encodedToken string) (PaginationToken, error) {
	var encryptionKey = []byte(os.Getenv("EncryptionKey"))

	var token PaginationToken

	// Base64 decode the token
	ciphertext, err := base64.URLEncoding.DecodeString(encodedToken)
	if err != nil {
		return token, err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return token, err
	}

	// Create a GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return token, err
	}

	// Check that the ciphertext is long enough
	if len(ciphertext) < gcm.NonceSize() {
		return token, errors.New("ciphertext too short")
	}

	// Extract the nonce
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return token, err
	}

	// Unmarshal the JSON
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return token, err
	}

	return token, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-southeast-2"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	handler := &Handler{
		DynamoDBClient: dynamoDBClient,
	}

	// Start the Lambda function
	lambda.Start(handler.HandleRequest)
}
