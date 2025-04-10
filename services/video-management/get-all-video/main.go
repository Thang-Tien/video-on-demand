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
	"github.com/aws/aws-sdk-go-v2/service/mediaconvert"
)

const (
	DefaultLimit   = 5
	MaxLimit       = 20
	TokenExpiryMin = 60 // Token expiry time in minutes
)

type VideoInformation struct {
	PK                     string                      `json:"pk"`
	SK                     string                      `json:"sk"`
	StartTime              string                      `json:"startTime"`
	WorkflowTrigger        string                      `json:"workflowTrigger"`
	WorkflowStatus         string                      `json:"workflowStatus"`
	WorkflowName           string                      `json:"workflowName"`
	SrcBucket              string                      `json:"srcBucket"`
	DestBucket             string                      `json:"destBucket"`
	CloudFront             string                      `json:"cloudFront"`
	FrameCapture           bool                        `json:"frameCapture"`
	ArchiveSource          string                      `json:"archiveSource"`
	JobTemplate2160p       string                      `json:"jobTemplate_2160p"`
	JobTemplate1080p       string                      `json:"jobTemplate_1080p"`
	JobTemplate720p        string                      `json:"jobTemplate_720p"`
	InputRotate            string                      `json:"inputRotate"`
	AcceleratedTranscoding string                      `json:"acceleratedTranscoding"`
	EnableSns              bool                        `json:"enableSns"`
	EnableSqs              bool                        `json:"enableSqs"`
	SrcVideo               string                      `json:"srcVideo"`
	EnableMediaPackage     bool                        `json:"enableMediaPackage"`
	SrcMediainfo           string                      `json:"srcMediainfo"`
	EncodingJob            mediaconvert.CreateJobInput `json:"encodingJob"`
	EncodeJobId            string                      `json:"encodeJobId"`
	EncodingOutput         EventDetail                 `json:"encodingOutput"`
	EndTime                time.Time                   `json:"endTime"`

	// Output
	HlsPlaylist            *string           `json:"hlsPlaylist"`
	HlsUrl                 *string           `json:"hlsUrl"`
	DashPlaylist           *string           `json:"dashPlaylist"`
	DashUrl                *string           `json:"dashUrl"`
	Mp4Outputs             []*string         `json:"mp4Outputs"`
	Mp4Urls                []*string         `json:"mp4Urls"`
	MssPlaylist            *string           `json:"mssPlaylist"`
	MssUrl                 *string           `json:"mssUrl"`
	CmafDashPlaylist       *string           `json:"cmafDashPlaylist"`
	CmafDashUrl            *string           `json:"cmafDashUrl"`
	CmafHlsPlaylist        *string           `json:"cmafHlsPlaylist"`
	CmafHlsUrl             *string           `json:"cmafHlsUrl"`
	ThumbNails             []*string         `json:"thumbNails"`
	ThumbNailsUrls         []*string         `json:"thumbNailsUrls"`
	MediaPackageResourceId string            `json:"mediaPackageResourceId"`
	EgressEndpoints        map[string]string `json:"egressEndpoints"`
	// Basic Information
	Title           *string    `json:"title"`
	OriginalTitle   *string    `json:"originalTitle"`
	Description     *string    `json:"description"`
	PlotSummary     *string    `json:"plotSummary"`
	ReleaseDate     *time.Time `json:"releaseDate"`
	ProductionYear  *int       `json:"productionYear"`
	Languages       *[]string  `json:"languages"`
	CountryOfOrigin *string    `json:"countryOfOrigin"`
	AgeRating       *string    `json:"ageRating"`

	// Creative Elements
	Directors          *[]string     `json:"directors"`
	Producers          *[]string     `json:"producers"`
	Writers            *[]string     `json:"writers"`
	Cast               *[]CastMember `json:"cast"`
	Cinematographer    *string       `json:"cinematographer"`
	MusicComposer      *string       `json:"musicComposer"`
	Editor             *string       `json:"editor"`
	ProductionDesigner *string       `json:"productionDesigner"`
	CostumeDesigner    *string       `json:"costumeDesigner"`

	// Technical Information
	SubtitleLanguages *[]string `json:"subtitleLanguages"`

	// Categorization
	Genres            *[]string `json:"genres"`
	Tags              *[]string `json:"tags"`
	Themes            *[]string `json:"themes"`
	SeriesInformation *string   `json:"seriesInformation"`
	SequelPrequel     *string   `json:"sequelPrequel"`
	SimilarMovies     *[]string `json:"similarMovies"`

	// Supplementary Content
	PosterURLs       *[]string `json:"posterUrls"`
	TrailerURLs      *[]string `json:"trailerUrls"`
	BehindTheScenes  *[]string `json:"behindTheScenes"`
	CommentaryTracks *[]string `json:"commentaryTracks"`
	DeletedScenes    *[]string `json:"deletedScenes"`
	Interviews       *[]string `json:"interviews"`

	// Reception and Metrics
	Awards               *[]string           `json:"awards"`
	CriticRatings        *map[string]float64 `json:"criticRatings"`
	UserRating           *float64            `json:"userRating"`
	BoxOfficePerformance *float64            `json:"boxOfficePerformance"`
	Views                *int                `json:"views"`
}

type CastMember struct {
	ActorName     string
	CharacterName string
}

type EventDetail struct {
	Timestamp          int64                `json:"timestamp"`
	AccountId          string               `json:"accountId"`
	Queue              string               `json:"queue"`
	JobId              string               `json:"jobId"`
	Status             string               `json:"status"`
	UserMetadata       UserMetadata         `json:"userMetadata"`
	OutputGroupDetails []*OutputGroupDetail `json:"outputGroupDetails"`
	PaddingInserted    int64                `json:"paddingInserted"`
	BlackVideoDetected int64                `json:"blackVideoDetected"`
	Warnings           []*Warning           `json:"warnings"`
}

type Warning struct {
	Code  int64 `json:"code"`
	Count int64 `json:"count"`
}

type UserMetadata struct {
	GUID     string `json:"guid"`
	Workflow string `json:"workflow"`
}

type OutputGroupDetail struct {
	OutputDetails     []*OutputDetail `json:"outputDetails"`
	PlaylistFilePaths []*string       `json:"playlistFilePaths"`
	Type              string          `json:"type"`
}

type OutputDetail struct {
	OutputFilePaths []*string    `json:"outputFilePaths"`
	DurationInMs    int64        `json:"durationInMs"`
	VideoDetails    *VideoDetail `json:"videoDetails"`
}

type VideoDetail struct {
	WidthInPx              int64   `json:"widthInPx"`
	HeightInPx             int64   `json:"heightInPx"`
	AverageBitrate         float64 `json:"averageBitrate"`
	QvbrAvgQuality         float64 `json:"qvbrAvgQuality"`
	QvbrMinQuality         float64 `json:"qvbrMinQuality"`
	QvbrMaxQuality         float64 `json:"qvbrMaxQuality"`
	QvbrMinQualityLocation float64 `json:"qvbrMinQualityLocation"`
	QvbrMaxQualityLocation float64 `json:"qvbrMaxQualityLocation"`
}

type Response struct {
	Videos     []VideoInformation `json:"videos"`
	NextToken  string             `json:"nextToken,omitempty"`
	TotalCount int                `json:"totalCount"`
}

type PaginationToken struct {
	LastEvaluatedKey map[string]string `json:"lastEvaluatedKey"`
	ExpiresAt        int64             `json:"expiresAt"`
}

type DynamoDBClient interface {
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestJSON, _ := json.Marshal(request)
	log.Printf("Request: %s", requestJSON)

	// Parse query parameters
	limit := DefaultLimit
	if limitParam, ok := request.QueryStringParameters["limit"]; ok {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			if parsedLimit > 0 {
				limit = parsedLimit
			}
			// Cap the limit to avoid excessive resource usage
			if limit > MaxLimit {
				limit = MaxLimit
			}
		}
	}

	// Get last evaluated key from nextToken if provided
	var exclusiveStartKey map[string]types.AttributeValue
	if nextToken, ok := request.QueryStringParameters["nextToken"]; ok && nextToken != "" {
		// Decode and decrypt the nextToken
		decodedToken, err := decodeNextToken(nextToken)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "Error decoding nextToken: " + err.Error(),
			}, nil
		}

		// Check if token has expired
		if decodedToken.ExpiresAt < time.Now().Unix() {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "Token has expired",
			}, nil
		}

		// Convert to DynamoDB attribute values
		exclusiveStartKey = map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: decodedToken.LastEvaluatedKey["PK"]},
			"SK": &types.AttributeValueMemberS{Value: decodedToken.LastEvaluatedKey["SK"]},
		}
	}

	// Create a filter expression for items with PK starting with "VIDEO#" and SK = "METADATA"
	filterBuilder := expression.Name("PK").BeginsWith("VIDEO#").And(expression.Name("SK").Equal(expression.Value("METADATA")))

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to build expression: %v", err),
		}, nil
	}

	// Execute scan
	scanInput := &dynamodb.ScanInput{
		TableName:                 aws.String(os.Getenv("DynamoDBTable")),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     aws.Int32(int32(limit)),
	}

	// Add exclusiveStartKey if we have a nextToken
	if exclusiveStartKey != nil {
		scanInput.ExclusiveStartKey = exclusiveStartKey
		log.Printf("ExclusiveStartKey: %v", exclusiveStartKey)
	}

	result, err := h.DynamoDBClient.Scan(ctx, scanInput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to scan DynamoDB: %v", err),
		}, nil
	}

	videos := make([]VideoInformation, 0, len(result.Items))
	for _, item := range result.Items {
		var video VideoInformation
		if err := attributevalue.UnmarshalMap(item, &video); err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       fmt.Sprintf("Failed to unmarshal item: %v", err),
			}, nil
		}
		videos = append(videos, video)
	}

	// Prepare the response
	response := Response{
		Videos:     videos,
		TotalCount: len(videos),
	}

	// Set nextToken if there are more results
	if result.LastEvaluatedKey != nil {
		nextKey := map[string]string{
			"PK": result.LastEvaluatedKey["PK"].(*types.AttributeValueMemberS).Value,
			"SK": result.LastEvaluatedKey["SK"].(*types.AttributeValueMemberS).Value,
		}

		// Create token with expiry time
		token := PaginationToken{
			LastEvaluatedKey: nextKey,
			ExpiresAt:        time.Now().Add(time.Minute * TokenExpiryMin).Unix(),
		}

		// Encode the token
		encodedToken, err := encodeNextToken(token)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       fmt.Sprintf("Failed to encode nextToken: %v", err),
			}, nil
		}

		response.NextToken = encodedToken
	}

	// Convert response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJSON),
	}, nil
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
