package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type DynamoDBClient interface {
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
	S3Client       S3Client
}

var guidPattern = regexp.MustCompile(`\/([a-f0-9-]{36})\/hls\/[^_]+\.m3u8`)

func (h *Handler) HandleRequest(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		log.Printf("Processing file: s3://%s/%s", bucket, key)

		// Only process CloudFront logs (usually have .gz extension)
		if !strings.HasSuffix(key, ".gz") {
			log.Printf("Skipping non-gzip file: %s", key)
			continue
		}

		resp, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("Error getting object %s/%s: %v", bucket, key, err)
			continue
		}
		defer resp.Body.Close()

		// Process the gzipped log file
		viewCounts, err := h.processLogFile(resp.Body)
		if err != nil {
			log.Printf("Error processing log file: %v", err)
			continue
		}

		log.Printf("View counts: %v", viewCounts)

		// Update DynamoDB with the view counts
		for guid, count := range viewCounts {
			err := h.updateDynamoDBViewCount(ctx, guid, count)
			if err != nil {
				log.Printf("Error updating DynamoDB for GUID %s: %v", guid, err)
			} else {
				log.Printf("Successfully updated view count for GUID %s: %d", guid, count)
			}
		}
	}
	return nil
}

func (h *Handler) processLogFile(reader io.Reader) (map[string]int, error) {
	// Create a gzip reader
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Create a map to store the view counts
	viewCounts := make(map[string]int)

	// Variables to track field positions
	var fieldNames []string
	var uriStemIdx, statusCodeIdx int

	// Read the log file line by line
	scanner := bufio.NewScanner(gzReader)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Process field headers
		if strings.HasPrefix(line, "#Fields:") {
			// Extract field names
			fields := strings.Split(line, " ")
			if len(fields) > 1 {
				fieldNames = fields[1:] // Skip the "#Fields:" part

				// Find indices for URI stem and status code
				for i, field := range fieldNames {
					if field == "cs-uri-stem" {
						uriStemIdx = i
					} else if field == "sc-status" {
						statusCodeIdx = i
					}
				}

				log.Printf("Found field indexes - URI Stem: %d, Status Code: %d", uriStemIdx, statusCodeIdx)
			}
			continue
		}

		// Skip other comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		// If we have field names, process the log entry
		if len(fieldNames) > 0 {
			fields := strings.Split(line, "\t")

			// Check if we have enough fields and proper indices
			if len(fields) > uriStemIdx && len(fields) > statusCodeIdx {
				uriStem := fields[uriStemIdx]
				statusCode := fields[statusCodeIdx]

				// Check if this is a successful request (200) for a .m3u8 file (but not a variant playlist)
				if statusCode == "200" {
					// Extract GUID from the URI path
					matches := guidPattern.FindStringSubmatch(uriStem)
					if len(matches) > 1 {
						guid := matches[1] // Extract the GUID
						viewCounts[guid]++
						log.Printf("Found view for GUID: %s, URI: %s", guid, uriStem)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning log file: %v", err)
	}

	return viewCounts, nil
}

// updateDynamoDBViewCount updates the view count for a video in DynamoDB
func (h *Handler) updateDynamoDBViewCount(ctx context.Context, guid string, views int) error {
	currentMonth := time.Now().Format("2006-01")

	// First, check if the item exists
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("VIDEO#%s", guid)},
			"SK": &types.AttributeValueMemberS{Value: "VIEWS"},
		},
		ProjectionExpression: aws.String("PK"), // We only need to know if the item exists
	}

	result, err := h.DynamoDBClient.GetItem(ctx, getInput)
	if err != nil {
		return fmt.Errorf("error checking if item exists: %v", err)
	}

	// Build different update expressions based on whether the item exists
	var updateExpr expression.UpdateBuilder

	if len(result.Item) == 0 {
		// Item doesn't exist, create new item with both total and monthly
		updateExpr = expression.Set(
			expression.Name("total"),
			expression.Value(views),
		).Set(
			expression.Name("monthly"),
			expression.Value(map[string]int{currentMonth: views}),
		)
	} else {
		// Item exists, update total and append to monthly
		updateExpr = expression.Set(
			expression.Name("total"),
			expression.Plus(
				expression.IfNotExists(expression.Name("total"), expression.Value(0)),
				expression.Value(views),
			),
		).Set(
			expression.Name("monthly."+currentMonth),
			expression.Plus(
				expression.IfNotExists(expression.Name("monthly."+currentMonth), expression.Value(0)),
				expression.Value(views),
			),
		)
	}

	expr, err := expression.NewBuilder().WithUpdate(updateExpr).Build()
	if err != nil {
		return fmt.Errorf("error building expression: %v", err)
	}

	// Build the update input
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("VIDEO#%s", guid)},
			"SK": &types.AttributeValueMemberS{Value: "VIEWS"},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	}

	// Execute the update
	updateItemOutput, err := h.DynamoDBClient.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("error updating DynamoDB: %v", err)
	}

	log.Printf("UpdateItem output: %+v", updateItemOutput)

	// Get current month's views instead of total
	var monthlyViews int
	if av, ok := updateItemOutput.Attributes["monthly"]; ok {
		if mv, ok := av.(*types.AttributeValueMemberM); ok {
			if cmv, ok := mv.Value[currentMonth]; ok {
				if n, ok := cmv.(*types.AttributeValueMemberN); ok {
					fmt.Sscanf(n.Value, "%d", &monthlyViews)
				}
			}
		}
	}

	// Create a period entry for ranking purposes
	periodKey := fmt.Sprintf("PERIOD#%s", currentMonth)
	videoKey := fmt.Sprintf("VIEW#%s", guid)

	rankInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: periodKey},
			"SK": &types.AttributeValueMemberS{Value: videoKey},
		},
		UpdateExpression: aws.String("SET videoId = :videoId, viewCount = :viewCount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":videoId":   &types.AttributeValueMemberS{Value: guid},
			":viewCount": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", monthlyViews)},
		},
	}

	if _, err := h.DynamoDBClient.UpdateItem(ctx, rankInput); err != nil {
		log.Printf("Error creating period ranking entry: %v", err)
	}

	return nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-southeast-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	handler := &Handler{
		DynamoDBClient: dynamoDBClient,
		S3Client:       s3Client,
	}

	lambda.Start(handler.HandleRequest)
}
