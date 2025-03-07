package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

var (
	TestConfigurationWithS3 = cloudfront.GetDistributionConfigOutput{
		DistributionConfig: &cloudfront.DistributionConfig{
			CallerReference: aws.String("some-caller-reference"),
			Origins: &cloudfront.Origins{
				Quantity: aws.Int64(1),
				Items: []*cloudfront.Origin{
					{
						Id:         aws.String("s3Origin"),
						DomainName: aws.String("some-bucket.s3.us-east-1.amazonaws.com"),
						OriginPath: aws.String(""),
						CustomHeaders: &cloudfront.CustomHeaders{
							Quantity: aws.Int64(0),
							Items:    []*cloudfront.OriginCustomHeader{},
						},
						S3OriginConfig: &cloudfront.S3OriginConfig{
							OriginAccessIdentity: aws.String("origin-access-identity/cloudfront/some-oai"),
						},
					},
				},
			},
			DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
				TargetOriginId: aws.String("s3Origin"),
				ForwardedValues: &cloudfront.ForwardedValues{
					QueryString: aws.Bool(false),
					Cookies: &cloudfront.CookiePreference{
						Forward: aws.String("none"),
					},
					Headers: &cloudfront.Headers{
						Quantity: aws.Int64(3),
						Items: []*string{
							aws.String("Access-Control-Request-Headers"),
							aws.String("Access-Control-Request-Method"),
							aws.String("Origin"),
						},
					},
					QueryStringCacheKeys: &cloudfront.QueryStringCacheKeys{
						Quantity: aws.Int64(0),
						Items:    []*string{},
					},
				},
				TrustedSigners: &cloudfront.TrustedSigners{
					Enabled:  aws.Bool(false),
					Quantity: aws.Int64(0),
					Items:    []*string{},
				},
				ViewerProtocolPolicy: aws.String("allow-all"),
				MinTTL:               aws.Int64(0),
			},
			CacheBehaviors: &cloudfront.CacheBehaviors{
				Quantity: aws.Int64(0),
				Items:    []*cloudfront.CacheBehavior{},
			},
			Comment: aws.String(""),
			Enabled: aws.Bool(true),
		},
		ETag: aws.String("some-etag"),
	}

	TestConfigurationWithMP = cloudfront.GetDistributionOutput{
		Distribution: &cloudfront.Distribution{
			DistributionConfig: &cloudfront.DistributionConfig{
				Origins: &cloudfront.Origins{
					Quantity: aws.Int64(1),
					Items: []*cloudfront.Origin{
						{
							Id: aws.String("vodMPOrigin"),
						},
					},
				},
			},
		},
	}
)

const TestDistributionId = "distribution-id"
const TestDomainName = "https://random-id.egress.mediapackage-vod.us-east-1.amazonaws.com"

func GetTestConfigurationWithS3() cloudfront.GetDistributionConfigOutput {
	return cloudfront.GetDistributionConfigOutput{
		DistributionConfig: &cloudfront.DistributionConfig{
			CallerReference: aws.String(*TestConfigurationWithS3.DistributionConfig.CallerReference),
			Origins: &cloudfront.Origins{
				Quantity: aws.Int64(*TestConfigurationWithS3.DistributionConfig.Origins.Quantity),
				Items: []*cloudfront.Origin{
					{
						Id:         aws.String(*TestConfigurationWithS3.DistributionConfig.Origins.Items[0].Id),
						DomainName: aws.String(*TestConfigurationWithS3.DistributionConfig.Origins.Items[0].DomainName),
						OriginPath: aws.String(*TestConfigurationWithS3.DistributionConfig.Origins.Items[0].OriginPath),
						CustomHeaders: &cloudfront.CustomHeaders{
							Quantity: aws.Int64(*TestConfigurationWithS3.DistributionConfig.Origins.Items[0].CustomHeaders.Quantity),
							Items:    []*cloudfront.OriginCustomHeader{},
						},
						S3OriginConfig: &cloudfront.S3OriginConfig{
							OriginAccessIdentity: aws.String(*TestConfigurationWithS3.DistributionConfig.Origins.Items[0].S3OriginConfig.OriginAccessIdentity),
						},
					},
				},
			},
			DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
				TargetOriginId: aws.String(*TestConfigurationWithS3.DistributionConfig.DefaultCacheBehavior.TargetOriginId),
				TrustedSigners: &cloudfront.TrustedSigners{
					Enabled:  aws.Bool(*TestConfigurationWithS3.DistributionConfig.DefaultCacheBehavior.TrustedSigners.Enabled),
					Quantity: aws.Int64(*TestConfigurationWithS3.DistributionConfig.DefaultCacheBehavior.TrustedSigners.Quantity),
					Items:    []*string{},
				},
				ViewerProtocolPolicy: aws.String(*TestConfigurationWithS3.DistributionConfig.DefaultCacheBehavior.ViewerProtocolPolicy),
			},
			CacheBehaviors: &cloudfront.CacheBehaviors{
				Quantity: aws.Int64(*TestConfigurationWithS3.DistributionConfig.CacheBehaviors.Quantity),
				Items:    []*cloudfront.CacheBehavior{},
			},
			Comment: aws.String(*TestConfigurationWithS3.DistributionConfig.Comment),
			Enabled: aws.Bool(*TestConfigurationWithS3.DistributionConfig.Enabled),
		},
		ETag: aws.String(*TestConfigurationWithS3.ETag),
	}
}

func GetTestConfigurationWithMP() cloudfront.GetDistributionConfigOutput {
	return cloudfront.GetDistributionConfigOutput{
		DistributionConfig: &cloudfront.DistributionConfig{
			Origins: &cloudfront.Origins{
				Quantity: aws.Int64(*TestConfigurationWithMP.Distribution.DistributionConfig.Origins.Quantity),
				Items: []*cloudfront.Origin{
					{
						Id: aws.String(*TestConfigurationWithMP.Distribution.DistributionConfig.Origins.Items[0].Id),
					},
				},
			},
		},
	}
}

func GetTestDistributionId() string {
	return TestDistributionId
}

func GetTestDomainName() string {
	return TestDomainName
}
