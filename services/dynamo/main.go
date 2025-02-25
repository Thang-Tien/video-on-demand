package main

import (
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoDBClient interface {
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

type Handler struct {
	DynamoDBClient DynamoDBClient
}

type DynamoEvent struct {
	GUID                   string `json:"guid"`
	StartTime              string `json:"startTime"`
	WorkflowTrigger        string `json:"workflowTrigger"`
	WorkflowStatus         string `json:"workflowStatus"`
	WorkflowName           string `json:"workflowName"`
	SrcBucket              string `json:"srcBucket"`
	DestBucket             string `json:"destBucket"`
	CloudFront             string `json:"cloudFront"`
	FrameCapture           bool   `json:"frameCapture"`
	ArchiveSource          string `json:"archiveSource"`
	JobTemplate2160p       string `json:"jobTemplate_2160p"`
	JobTemplate1080p       string `json:"jobTemplate_1080p"`
	JobTemplate720p        string `json:"jobTemplate_720p"`
	InputRotate            string `json:"inputRotate"`
	AcceleratedTranscoding string `json:"acceleratedTranscoding"`
	EnableSns              bool   `json:"enableSns"`
	EnableSqs              bool   `json:"enableSqs"`
	SrcVideo               string `json:"srcVideo"`
	EnableMediaPackage     bool   `json:"enableMediaPackage"`
	SrcMediainfo           string `json:"srcMediainfo"`
}

type DynamoOutput struct {
	GUID                   string `json:"guid"`
	StartTime              string `json:"startTime"`
	WorkflowTrigger        string `json:"workflowTrigger"`
	WorkflowStatus         string `json:"workflowStatus"`
	WorkflowName           string `json:"workflowName"`
	SrcBucket              string `json:"srcBucket"`
	DestBucket             string `json:"destBucket"`
	CloudFront             string `json:"cloudFront"`
	FrameCapture           bool   `json:"frameCapture"`
	ArchiveSource          string `json:"archiveSource"`
	JobTemplate2160p       string `json:"jobTemplate_2160p"`
	JobTemplate1080p       string `json:"jobTemplate_1080p"`
	JobTemplate720p        string `json:"jobTemplate_720p"`
	InputRotate            string `json:"inputRotate"`
	AcceleratedTranscoding string `json:"acceleratedTranscoding"`
	EnableSns              bool   `json:"enableSns"`
	EnableSqs              bool   `json:"enableSqs"`
	SrcVideo               string `json:"srcVideo"`
	EnableMediaPackage     bool   `json:"enableMediaPackage"`
	SrcMediainfo           string `json:"srcMediainfo"`
}


func (h *Handler) HandleRequest(event DynamoEvent) (*DynamoOutput, error) {
	log.Printf("REQUEST:: %v", event)

	expression := ""
	values := map[string]*dynamodb.AttributeValue{}
	v := reflect.ValueOf(event)
	typeOfEvent := v.Type()

	for i := range v.NumField() {
		if typeOfEvent.Field(i).Name == "GUID" {
			continue
		}

		expression += typeOfEvent.Field(i).Name + " = :" + strconv.Itoa(i) + ", "
		values[":"+strconv.Itoa(i)] = &dynamodb.AttributeValue{
			S: aws.String(v.Field(i).String()),
		}
	}

	input := dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("DynamoDBTable")),
		Key: map[string]*dynamodb.AttributeValue{
			"guid": {
				S: aws.String(event.GUID),
			},
		},
		UpdateExpression:          aws.String(expression),
		ExpressionAttributeValues: values,
	}

	_, err := h.DynamoDBClient.UpdateItem(&input)
	if err != nil {
		return nil, err
	}

	log.Println("UPDATE:: Successfully updated item in DynamoDB")

	output := &DynamoOutput{
		GUID:                   event.GUID,
		StartTime:              event.StartTime,
		WorkflowTrigger:        event.WorkflowTrigger,
		WorkflowStatus:         event.WorkflowStatus,
		WorkflowName:           event.WorkflowName,
		SrcBucket:              event.SrcBucket,
		DestBucket:             event.DestBucket,
		CloudFront:             event.CloudFront,
		FrameCapture:           event.FrameCapture,
		ArchiveSource:          event.ArchiveSource,
		JobTemplate2160p:       event.JobTemplate2160p,
		JobTemplate1080p:       event.JobTemplate1080p,
		JobTemplate720p:        event.JobTemplate720p,
		InputRotate:            event.InputRotate,
		AcceleratedTranscoding: event.AcceleratedTranscoding,
		EnableSns:              event.EnableSns,
		EnableSqs:              event.EnableSqs,
		SrcVideo:               event.SrcVideo,
		EnableMediaPackage:     event.EnableMediaPackage,
		SrcMediainfo:           event.SrcMediainfo,
	}

	return output, nil
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	// Create DynamoDB client
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	dynamo := dynamodb.New(sess)

	log.Println("DynamoDB client initialized:", dynamo)

	handler := &Handler{
		DynamoDBClient: dynamo,
	}

	lambda.Start(handler.HandleRequest)
}
