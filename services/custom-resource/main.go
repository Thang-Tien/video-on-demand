package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/mediapackagevod"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

type CustomResourceResponse struct {
	GroupId         *string
	GroupDomainName *string
	EndpointUrl     *string
	UUID            *string
}

type Handler struct {
	S3CustomResource           S3CustomResource
	MediaPackageCustomResource MediaPackageCustomResource
	MetricCustomResource       MetricCustomResource
	MediaConvertCustomResource MediaConvertCustomResource
	CfnCustomResource          CfnCustomResource
}

func (h *Handler) HandleRequest(event cfn.Event) (*CustomResourceResponse, error) {
	log.Printf("REQUEST:: %v", event)

	config := event.ResourceProperties
	responseData := CustomResourceResponse{}

	if event.RequestType == cfn.RequestCreate {
		resourceValue, ok := config["Resource"]
		if !ok {
			return nil, fmt.Errorf("custom-resource: Resource property is missing")
		}

		resourceStr, ok := resourceValue.(string)
		if !ok {
			return nil, fmt.Errorf("custom-resource: Resource property must be a string")
		}

		switch resourceStr {
		case "S3Notification":
			_, err := h.S3CustomResource.PutNotification(config)
			if err != nil {
				return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: PutNotification: %w", err)
			}

		case "EndPoint":
			url, err := h.MediaConvertCustomResource.GetEndpoint()
			if err != nil {
				return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: GetEndpoint: %w", err)
			}
			responseData.EndpointUrl = url

		case "MediaConvertTemplates":
			err := h.MediaConvertCustomResource.CreateTemplates(config)
			if err != nil {
				return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: CreateTemplates: %w", err)
			}
		case "UUID":
			uuid := uuid.New().String()
			responseData.UUID = &uuid
		case "AnonymizedMetric":
			sendAnonymizedMetricValue, ok := config["SendAnonymizedMetric"]
			if !ok {
				return nil, fmt.Errorf("custom-resource: SendAnonymizedMetric property is missing")
			}

			sendAnonymizedMetricStr, ok := sendAnonymizedMetricValue.(string)
			if !ok {
				return nil, fmt.Errorf("custom-resource: SendAnonymizedMetric property must be a string")
			}

			if sendAnonymizedMetricStr == "Yes" {
				_, err := h.MetricCustomResource.Send(config)
				if err != nil {
					return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: Send: %w", err)
				}
			}
		case "MediaPackageVod":
			enableMediaPackageValue, ok := config["EnableMediaPackage"]
			if !ok {
				return nil, fmt.Errorf("custom-resource: EnableMediaPackage property is missing")
			}

			enableMediaPackageStr, ok := enableMediaPackageValue.(string)
			if !ok {
				return nil, fmt.Errorf("custom-resource: EnableMediaPackage property must be a string")
			}

			if enableMediaPackageStr == "true" {
				res, err := h.MediaPackageCustomResource.Create(config)
				if err != nil {
					return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: Create: %w", err)
				}
				responseData.GroupId = &res.GroupID
				responseData.GroupDomainName = &res.GroupDomainName
			}
		default:
			log.Printf("custom-resource: main.Handler.HandleRequest: %s not defined as a custom resource, sending success response", resourceStr)
		}
	}

	res, err := h.CfnCustomResource.Send(event, "SUCCESS", responseData)
	if err != nil {
		return nil, fmt.Errorf("custom-resource: main.Handler.HandleRequest: Send: %w", err)
	}

	log.Printf("RESPONSE:: %v", responseData)
	log.Printf("CFN STATUS:: %d", *res)

	return &responseData, nil
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}

	mediaPackageClient := mediapackagevod.New(sess)
	CloudFrontClient := cloudfront.New(sess)
	s3Client := s3.New(sess)
	metricClient := &http.Client{}
	mediaconvertClient := mediaconvert.New(sess)
	cfnClient := &http.Client{}

	mediaPackageCustomResource := MediaPackageCustomResource{
		MediaPackageVODClient: mediaPackageClient,
		CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: CloudFrontClient},
	}
	s3CustomResource := S3CustomResource{
		S3Client: s3Client,
	}
	metricCustomResource := MetricCustomResource{
		MetricClient: metricClient,
	}
	mediaConvertCustomResource := MediaConvertCustomResource{
		MediaConvertClient: mediaconvertClient,
	}
	cfnCustomResource := CfnCustomResource{
		CfnClient: cfnClient,
	}

	handler := &Handler{
		S3CustomResource:           s3CustomResource,
		MediaPackageCustomResource: mediaPackageCustomResource,
		MetricCustomResource:       metricCustomResource,
		MediaConvertCustomResource: mediaConvertCustomResource,
		CfnCustomResource:          cfnCustomResource,
	}
	lambda.Start(handler.HandleRequest)
}
