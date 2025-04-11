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
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/mediaconvert"
)

const (
	DEFAULT_PAGE_SIZE = 5
	MAX_PAGE_SIZE     = 20
	TOKEN_EXPIRY_MINS = 60 // Token expiry time in minutes
)

var errInvalidPageNumber = errors.New("invalid page number")

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
	Count      int                `json:"count"`
	TotalCount int                `json:"totalCount,omitempty"` // Optional, requires a separate count query
	NextToken  string             `json:"nextToken,omitempty"`
	PageNumber int                `json:"pageNumber"`
	PagesCount int                `json:"pagesCount,omitempty"` // Optional, requires knowing total count
}

type PaginationToken struct {
	LastEvaluatedKey map[string]string `json:"lastEvaluatedKey"`
	ExpiresAt        int64             `json:"expiresAt"`
	PageNumber       int               `json:"pageNumber,omitempty"`
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

	// search terms
	genres := parseGenres(request.QueryStringParameters["genres"])
	casts := parseCasts(request.QueryStringParameters["casts"])
	countries := parseCountries(request.QueryStringParameters["countries"])
	productionYears := parseProductionYears(request.QueryStringParameters["productionYears"])

	// if page number is provided but not nextToken, scan to that page
	var lastEvaluatedKey map[string]types.AttributeValue
	if nextToken == "" {
		var err error
		lastEvaluatedKey, err = h.scanToPage(ctx, 1, pageNumber, pageSize, genres, casts, countries, productionYears, nil)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       fmt.Sprintf("Failed to scan to page: %v", err),
			}, nil
		}
	} else if nextToken != "" {
		// Decode the nextToken to get the last evaluated key
		paginationToken, err := decodeNextToken(nextToken)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       fmt.Sprintf("Invalid nextToken format: %v", err),
			}, nil
		}

		// Check if the token is expired
		if time.Now().Unix() > paginationToken.ExpiresAt {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "nextToken has expired",
			}, nil
		}

		// Create the last evaluated key from the decoded token
		tokenLastEvaluatedKey := map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: paginationToken.LastEvaluatedKey["PK"]},
			"SK": &types.AttributeValueMemberS{Value: paginationToken.LastEvaluatedKey["SK"]},
		}

		if paginationToken.PageNumber > pageNumber {
			lastEvaluatedKey, err = h.scanToPage(ctx, 1, pageNumber, pageSize, genres, casts, countries, productionYears, nil)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       fmt.Sprintf("Failed to scan to page: %v", err),
				}, nil
			}
		} else {
			lastEvaluatedKey, err = h.scanToPage(ctx, paginationToken.PageNumber, pageNumber, pageSize, genres, casts, countries, productionYears, tokenLastEvaluatedKey)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       fmt.Sprintf("Failed to scan to page: %v", err),
				}, nil
			}
		}
	}

	// make input
	scanInput := &dynamodb.ScanInput{
		TableName:                 aws.String(os.Getenv("DynamoDBTable")),
		Limit:                     aws.Int32(int32(pageSize)),
		ExclusiveStartKey:         lastEvaluatedKey,
		FilterExpression:          aws.String(buildFilterExpression(genres, casts, countries, productionYears)),
		ExpressionAttributeValues: buildExpressionAttributeValues(genres, casts, countries, productionYears),
	}

	result, err := h.DynamoDBClient.Scan(ctx, scanInput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to scan DynamoDB: %v", err),
		}, nil
	}

	var videos []VideoInformation
	err = attributevalue.UnmarshalListOfMaps(result.Items, &videos)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to unmarshal items: %v", err),
		}, nil
	}

	response := Response{
		Videos:     videos,
		Count:      len(videos),
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
			}, nil
		}
	}

	// Convert response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}

	log.Printf("Response: %s", responseJSON)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJSON),
	}, nil
}

func (h *Handler) scanToPage(ctx context.Context, pageStart, pageNumber, pageSize int, genres, casts, countries, productionYears []string, lastEvaluatedKey map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	for i := pageStart; i < pageNumber; i++ {
		scanInput := &dynamodb.ScanInput{
			TableName:                 aws.String(os.Getenv("DynamoDBTable")),
			Limit:                     aws.Int32(int32(pageSize)),
			ExclusiveStartKey:         lastEvaluatedKey,
			FilterExpression:          aws.String(buildFilterExpression(genres, casts, countries, productionYears)),
			ExpressionAttributeValues: buildExpressionAttributeValues(genres, casts, countries, productionYears),
		}

		result, err := h.DynamoDBClient.Scan(ctx, scanInput)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			return nil, errInvalidPageNumber
		}
	}

	return lastEvaluatedKey, nil
}

func buildFilterExpression(genres, casts, countries, productionYears []string) string {
	var filters []string

	// Default filter to ensure PK starts with VIDEO# and SK is METADATA
	filters = append(filters, "begins_with(PK, :pk)", "SK = :sk")

	if len(genres) > 0 {
		filters = append(filters, "contains(#genres, :genres)")
	}
	if len(casts) > 0 {
		filters = append(filters, "contains(#casts, :casts)")
	}
	if len(countries) > 0 {
		filters = append(filters, "contains(#countries, :countries)")
	}
	if len(productionYears) > 0 {
		filters = append(filters, "contains(#productionYears, :productionYears)")
	}

	return strings.Join(filters, " AND ")
}

func buildExpressionAttributeValues(genres, casts, countries, productionYears []string) map[string]types.AttributeValue {
	expressionValues := make(map[string]types.AttributeValue)

	// Default values for PK and SK
	expressionValues[":pk"] = &types.AttributeValueMemberS{Value: "VIDEO#"}
	expressionValues[":sk"] = &types.AttributeValueMemberS{Value: "METADATA"}

	if len(genres) > 0 {
		expressionValues[":genres"] = &types.AttributeValueMemberSS{Value: genres}
	}
	if len(casts) > 0 {
		expressionValues[":casts"] = &types.AttributeValueMemberSS{Value: casts}
	}
	if len(countries) > 0 {
		expressionValues[":countries"] = &types.AttributeValueMemberSS{Value: countries}
	}
	if len(productionYears) > 0 {
		expressionValues[":productionYears"] = &types.AttributeValueMemberSS{Value: productionYears}
	}

	return expressionValues
}

func parseGenres(input string) []string {
	if input == "" {
		return nil
	}
	genres := strings.Split(input, ",")
	return genres
}

func parseCasts(input string) []string {
	if input == "" {
		return nil
	}
	casts := strings.Split(input, ",")
	return casts
}

func parseCountries(input string) []string {
	if input == "" {
		return nil
	}
	countries := strings.Split(input, ",")
	return countries
}

func parseProductionYears(input string) []string {
	if input == "" {
		return nil
	}
	productionYears := strings.Split(input, ",")
	return productionYears
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
