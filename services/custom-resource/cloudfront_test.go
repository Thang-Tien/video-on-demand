package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/stretchr/testify/mock"
)

type CloudFrontClientMock struct {
	mock.Mock
}

func (m *CloudFrontClientMock) GetDistributionConfig(input *cloudfront.GetDistributionConfigInput) (*cloudfront.GetDistributionConfigOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudfront.GetDistributionConfigOutput), args.Error(1)
}

func (m *CloudFrontClientMock) UpdateDistribution(input *cloudfront.UpdateDistributionInput) (*cloudfront.UpdateDistributionOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudfront.UpdateDistributionOutput), args.Error(1)
}

func TestCloudFront(t *testing.T) {
	t.Parallel()
	t.Run("Validation", func(t *testing.T) {
		t.Parallel()

		cloudFrontHelper := CloudFrontHelper{
			CloudFrontClient: &CloudFrontClientMock{},
		}

		t.Run("should return error when distributionId is empty", func(t *testing.T) {
			err := cloudFrontHelper.AddCustomOrigin("", TestDomainName)
			if err == nil {
				t.Error("expected an error")
			}
		})

		t.Run("should return error when domainName is empty", func(t *testing.T) {
			err := cloudFrontHelper.AddCustomOrigin(TestDistributionId, "")
			if err == nil {
				t.Error("expected an error")
			}
		})
	})

	t.Run("Api", func(t *testing.T) {
		t.Parallel()

		t.Run("should return error when UpdateDistribution fails with something other than PreconditionFailed", func(t *testing.T) {
			mockClient := new(CloudFrontClientMock)
			cloudFrontHelper := CloudFrontHelper{
				CloudFrontClient: mockClient,
			}

			config := GetTestConfigurationWithS3()

			mockClient.On("GetDistributionConfig", mock.Anything).Return(&config, nil)
			mockClient.On("UpdateDistribution", mock.Anything).Return(&cloudfront.UpdateDistributionOutput{}, errors.New("update distribution error"))

			cloudFrontHelper.CloudFrontClient = mockClient

			err := cloudFrontHelper.AddCustomOrigin(TestDistributionId, TestDomainName)
			if err == nil {
				t.Error("expected an error")
			}
			mockClient.AssertExpectations(t)
		})

		t.Run("should not return error when UpdateDistribution fails with PreconditionFailed", func(t *testing.T) {
			mockClient := new(CloudFrontClientMock)
			cloudFrontHelper := CloudFrontHelper{
				CloudFrontClient: mockClient,
			}

			config := GetTestConfigurationWithS3()
			mockClient.On("GetDistributionConfig", mock.Anything).Return(&config, nil)
			mockClient.On("UpdateDistribution", mock.Anything).Return(&cloudfront.UpdateDistributionOutput{}, awserr.New(cloudfront.ErrCodePreconditionFailed, "Precondition Failed", nil))

			err := cloudFrontHelper.AddCustomOrigin(TestDistributionId, TestDomainName)
			if err != nil {
				t.Error("expected no error")
			}
		})

		t.Run("should not add origin if it already exists", func(t *testing.T) {
			mockClient := new(CloudFrontClientMock)
			cloudFrontHelper := CloudFrontHelper{
				CloudFrontClient: mockClient,
			}

			config := GetTestConfigurationWithS3()
			config.DistributionConfig.Origins.Items = append(config.DistributionConfig.Origins.Items, &cloudfront.Origin{
				Id: aws.String("vodMPOrigin"),
			})
			config.DistributionConfig.Origins.Quantity = aws.Int64(2)

			mockClient.On("GetDistributionConfig", mock.Anything).Return(&config, nil)

			err := cloudFrontHelper.AddCustomOrigin(TestDistributionId, TestDomainName)
			if err != nil {
				t.Error("expected no error")
			}

			mockClient.AssertNotCalled(t, "UpdateDistribution")
		})

		t.Run("should succeed with valid parameter", func(t *testing.T) {
			mockClient := new(CloudFrontClientMock)
			cloudFrontHelper := CloudFrontHelper{
				CloudFrontClient: mockClient,
			}
			
			config := GetTestConfigurationWithS3()
			mockClient.On("GetDistributionConfig", mock.Anything).Return(&config, nil)
			mockClient.On("UpdateDistribution", mock.Anything).Return(&cloudfront.UpdateDistributionOutput{}, nil)

			err := cloudFrontHelper.AddCustomOrigin(TestDistributionId, TestDomainName)
			if err != nil {
				t.Error("expected no error")
			}
			mockClient.AssertExpectations(t)
		})
	})
}
