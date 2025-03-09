package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

const originId = "vodMPOrigin"

var (
	ErrDistributionNotFound     = errors.New("distribution not found")
	ErrOriginDomainNameNotFound = errors.New("origin domain name not found")
)

type CloudFrontClient interface {
	GetDistributionConfig(input *cloudfront.GetDistributionConfigInput) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(input *cloudfront.UpdateDistributionInput) (*cloudfront.UpdateDistributionOutput, error)
}

type CloudFrontHelper struct {
	CloudFrontClient CloudFrontClient
}

func (c *CloudFrontHelper) AddCustomOrigin(distributionId, domainName string) error {
	if distributionId == "" {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: %w", ErrDistributionNotFound)
	}
	if domainName == "" {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: %w", ErrOriginDomainNameNotFound)
	}

	response, err := c.CloudFrontClient.GetDistributionConfig(&cloudfront.GetDistributionConfigInput{
		Id: aws.String(distributionId),
	})
	if err != nil {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: GetDistributionConfig: %w", err)
	}

	config := response.DistributionConfig
	originExists := false
	for _, item := range config.Origins.Items {
		if *item.Id == originId {
			originExists = true
			break
		}
	}

	if originExists {
		log.Printf("CloudFrontHelper.AddCustomOrigin: Origin %s has already been added to distribution %s", originId, distributionId)
		return nil
	}

	log.Printf("CloudFrontHelper.AddCustomOrigin: Adding MediaPackage as origin to distribution %s", distributionId)

	// Parse the domain name from the URL
	u, err := url.Parse(domainName)
	if err != nil {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: Parse: %w", err)
	}

	customOrigin := cloudfront.Origin{
		Id:         aws.String(originId),
		DomainName: aws.String(u.Hostname()),
		OriginPath: aws.String(""),
		CustomHeaders: &cloudfront.CustomHeaders{
			Quantity: aws.Int64(0),
		},
		CustomOriginConfig: &cloudfront.CustomOriginConfig{
			HTTPPort:             aws.Int64(80),
			HTTPSPort:            aws.Int64(443),
			OriginProtocolPolicy: aws.String("https-only"),
			OriginSslProtocols: &cloudfront.OriginSslProtocols{
				Items: []*string{
					aws.String("TLSv1.2"),
				},
				Quantity: aws.Int64(1),
			},
			OriginKeepaliveTimeout: aws.Int64(5),
			OriginReadTimeout:      aws.Int64(30),
		},
	}

	config.Origins.Items = append(config.Origins.Items, &customOrigin)
	config.Origins.Quantity = aws.Int64(int64(len(config.Origins.Items)))

	customBehavior := cloudfront.CacheBehavior{
		PathPattern:    aws.String("out/*"),
		TargetOriginId: aws.String(originId),
		ForwardedValues: &cloudfront.ForwardedValues{
			QueryString: aws.Bool(true),
			Cookies: &cloudfront.CookiePreference{
				Forward: aws.String("none"),
			},
			Headers: &cloudfront.Headers{
				Quantity: aws.Int64(4),
				Items: []*string{
					aws.String("Access-Control-Request-Headers"),
					aws.String("Access-Control-Request-Method"),
					aws.String("Origin"),
					aws.String("Access-Control-Allow-Origin"),
				},
			},
			QueryStringCacheKeys: &cloudfront.QueryStringCacheKeys{
				Quantity: aws.Int64(1),
				Items: []*string{
					aws.String("aws.manifestfilter"),
				},
			},
		},
		TrustedSigners: &cloudfront.TrustedSigners{
			Enabled:  aws.Bool(false),
			Quantity: aws.Int64(0),
		},
		ViewerProtocolPolicy: aws.String("redirect-to-https"),
		MinTTL:               aws.Int64(0),
		AllowedMethods: &cloudfront.AllowedMethods{
			Quantity: aws.Int64(2),
			Items: []*string{
				aws.String("GET"),
				aws.String("HEAD"),
			},
			CachedMethods: &cloudfront.CachedMethods{
				Quantity: aws.Int64(2),
				Items: []*string{
					aws.String("GET"),
					aws.String("HEAD"),
				},
			},
		},
		SmoothStreaming: aws.Bool(false),
		DefaultTTL:      aws.Int64(86400),
		MaxTTL:          aws.Int64(31536000),
		Compress:        aws.Bool(false),
		LambdaFunctionAssociations: &cloudfront.LambdaFunctionAssociations{
			Quantity: aws.Int64(0),
		},
		FieldLevelEncryptionId: aws.String(""),
	}

	input := cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionId),
		DistributionConfig: config,
		IfMatch:            response.ETag,
	}

	config.CacheBehaviors.Items = append(config.CacheBehaviors.Items, &customBehavior)
	config.CacheBehaviors.Quantity = aws.Int64(int64(len(config.CacheBehaviors.Items)))

	_, err = c.CloudFrontClient.UpdateDistribution(&input)
	if err != nil {
		var awsErr awserr.Error
		if !errors.As(err, &awsErr) || awsErr.Code() != cloudfront.ErrCodePreconditionFailed {
			return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: UpdateDistribution: %w", err)
		}
	}

	originItemJson, err := json.Marshal(config.Origins.Items)
	if err != nil {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: json.Marshal: %w", err)
	}
	cacheBehaviorJson, err := json.Marshal(config.CacheBehaviors.Items)
	if err != nil {
		return fmt.Errorf("CloudFrontHelper.AddCustomOrigin: json.Marshal: %w", err)
	}
	log.Printf("Origins:: %s", originItemJson)
	log.Printf("Cache behaviors:: %s", cacheBehaviorJson)

	return nil
}
