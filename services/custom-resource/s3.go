package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mitchellh/mapstructure"
)

var suffixList = []string{
	".mpg",
	".mp4",
	".m4v",
	".mov",
	".m2ts",
	".wmv",
	".mxf",
	".mkv",
	".m3u8",
	".mpeg",
	".webm",
	".h264",
}

var ErrInvalidWorkflowTrigger = errors.New("invalid workflow trigger")

type S3CustomResource struct {
	S3Client S3Client
}

type S3Client interface {
	PutBucketNotificationConfiguration(input *s3.PutBucketNotificationConfigurationInput) (*s3.PutBucketNotificationConfigurationOutput, error)
}

type S3CustomResourceConfig struct {
	ServiceToken    string
	IngestArn       string
	Resource        string
	WorkflowTrigger string
	Source          string
}

func generateConfigurations(suffix string, lambdaArn string) *s3.LambdaFunctionConfiguration {
	return &s3.LambdaFunctionConfiguration{
		Events:            aws.StringSlice([]string{"s3:ObjectCreated:*"}),
		LambdaFunctionArn: aws.String(lambdaArn),
		Filter: &s3.NotificationConfigurationFilter{
			Key: &s3.KeyFilter{
				FilterRules: []*s3.FilterRule{
					{
						Name:  aws.String("suffix"),
						Value: aws.String(suffix),
					},
				},
			},
		},
	}
}

func (s *S3CustomResource) PutNotification(config map[string]interface{}) (*string, error) {
	var s3Config S3CustomResourceConfig
	if err := mapstructure.Decode(config, &s3Config); err != nil {
		return nil, fmt.Errorf("S3CustomResource.PutNotification: Decode: error decoding config: %v", err)
	}

	switch s3Config.WorkflowTrigger {
	case "VideoFile":
		generateAllConfigurations := func() []*s3.LambdaFunctionConfiguration {
			lambdaArn := s3Config.IngestArn
			var configurations []*s3.LambdaFunctionConfiguration

			for _, suffix := range suffixList {
				configurations = append(configurations, generateConfigurations(suffix, lambdaArn))
				configurations = append(configurations, generateConfigurations(strings.ToUpper(suffix), lambdaArn))
			}

			return configurations
		}

		_, err := s.S3Client.PutBucketNotificationConfiguration(
			&s3.PutBucketNotificationConfigurationInput{
				Bucket: aws.String(s3Config.Source),
				NotificationConfiguration: &s3.NotificationConfiguration{
					LambdaFunctionConfigurations: generateAllConfigurations(),
				},
			},
		)

		if err != nil {
			return nil, fmt.Errorf("S3CustomResource.PutNotification: %w", err)
		}
	default:
		return nil, fmt.Errorf("S3CustomResource.PutNotification: %w", ErrInvalidWorkflowTrigger)
	}

	return aws.String("success"), nil
}
