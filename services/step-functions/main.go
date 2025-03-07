package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/google/uuid"
)

var (
	ErrInvalidEventObject = errors.New("invalid event object")
)

type StepFunctionEvent struct {
	Records                []events.S3EventRecord `json:"Records"`
	GUID                   *string                 `json:"guid"`
	StartTime              *string                 `json:"startTime"`
	WorkflowTrigger        *string                 `json:"workflowTrigger"`
	WorkflowStatus         *string                 `json:"workflowStatus"`
	WorkflowName           *string                 `json:"workflowName"`
	SrcBucket              *string                 `json:"srcBucket"`
	DestBucket             *string                 `json:"destBucket"`
	CloudFront             *string                 `json:"cloudFront"`
	FrameCapture           *bool                   `json:"frameCapture"`
	ArchiveSource          *string                 `json:"archiveSource"`
	JobTemplate2160p       *string                 `json:"jobTemplate_2160p"`
	JobTemplate1080p       *string                 `json:"jobTemplate_1080p"`
	JobTemplate720p        *string                 `json:"jobTemplate_720p"`
	InputRotate            *string                 `json:"inputRotate"`
	AcceleratedTranscoding *string                 `json:"acceleratedTranscoding"`
	EnableSns              *bool                   `json:"enableSns"`
	EnableSqs              *bool                   `json:"enableSqs"`
	SrcVideo               *string                 `json:"srcVideo"`
	EnableMediaPackage     *bool                   `json:"enableMediaPackage"`
	SrcMediainfo           *string                 `json:"srcMediainfo"`
}

type StepFunctionClent interface {
	StartExecution(input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error)
}

type Handler struct {
	StepFunctionClient StepFunctionClent
}

func (h *Handler) HandleRequest(event StepFunctionEvent) (*string, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		log.Printf("step-function: main.Handler: Error marshalling event: %v", err)
	}
	log.Printf("REQUEST:: %s", eventJson)

	var response string
	var startExecutionInput sfn.StartExecutionInput

	switch {
	case event.Records != nil:
		// Ingest workflow triggerd by s3 event::
		event.GUID = aws.String(uuid.New().String())
		event.WorkflowTrigger = aws.String("Video")

		inputBytes, err := json.Marshal(event)
		if err != nil {
			log.Printf("step-function: main.Handler: Error marshalling event: %v", err)
		}

		startExecutionInput = sfn.StartExecutionInput{
			Name:            event.GUID,
			Input:           aws.String(string(inputBytes)),
			StateMachineArn: aws.String(os.Getenv("IngestWorkflow")),
		}
		response = "success"
	case event.GUID != nil:
		inputBytes, err := json.Marshal(event.GUID)
		if err != nil {
			log.Printf("step-function: main.Handler: Error marshalling event: %v", err)
		}
		// Process workflow trigger
		startExecutionInput = sfn.StartExecutionInput{
			Name:            event.GUID,
			Input:           aws.String(string(inputBytes)),
			StateMachineArn: aws.String(os.Getenv("ProcessWorkflow")),
		}
		response = "success"
	default:
		return nil, ErrInvalidEventObject
	}

	data, err := h.StepFunctionClient.StartExecution(&startExecutionInput)
	if err != nil {
		log.Printf("step-function: main.Handler: Error starting execution: %v", err)
	}


	dataJson, err := json.Marshal(data) 
	if err != nil {
		log.Printf("step-function: main.Handler: Error marshalling data: %v", err)
	}
	log.Printf("STATEMACHINE EXECUTE:: %s", dataJson)

	return &response, nil
}

func main() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}))
	stepFunctionClient := sfn.New(sess)
	handler := &Handler{
		StepFunctionClient: stepFunctionClient,
	}

	lambda.Start(handler.HandleRequest)
}
