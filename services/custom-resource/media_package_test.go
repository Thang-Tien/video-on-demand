package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediapackagevod"
	"github.com/stretchr/testify/mock"
)

const testStackName = "test-stack"
const testGroupId = "test-packaging-group"

var ValidParameter = map[string]interface{}{
	"StackName":               testStackName,
	"GroupId":                 testGroupId,
	"PackagingConfigurations": "HLS,DASH",
	"DistributionId":          TestDistributionId,
	"EnableMediaPackage":      "true",
}

type MediaPackageVodClientMock struct {
	mock.Mock
}

func (m *MediaPackageVodClientMock) CreatePackagingGroup(input *mediapackagevod.CreatePackagingGroupInput) (*mediapackagevod.CreatePackagingGroupOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediapackagevod.CreatePackagingGroupOutput), args.Error(1)
}

func (m *MediaPackageVodClientMock) CreatePackagingConfiguration(input *mediapackagevod.CreatePackagingConfigurationInput) (*mediapackagevod.CreatePackagingConfigurationOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediapackagevod.CreatePackagingConfigurationOutput), args.Error(1)
}

func TestMediaPackage(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("should succeed with valid parameters", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testGroupId),
				DomainName: aws.String(TestDomainName),
			}
			getDistributionConfigOutputMock := GetTestConfigurationWithMP()

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)
			mediaPackageVodClientMock.On("CreatePackagingConfiguration", mock.Anything).Return(&mediapackagevod.CreatePackagingConfigurationOutput{}, nil)
			cloudFrontClientMock.On("GetDistributionConfig", mock.Anything).Return(&getDistributionConfigOutputMock, nil)

			res, err := mediaPackageCustomResource.Create(ValidParameter)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if res == nil {
				t.Fatal("Expected non-nil response")
			}

			if res.GroupID != testGroupId {
				t.Errorf("Expected GroupID %s, got %s", testGroupId, res.GroupID)
			}
		})

		t.Run("should ignore duplicate configurations", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testGroupId),
				DomainName: aws.String(TestDomainName),
			}
			getDistributionConfigOutputMock := GetTestConfigurationWithMP()

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)
			mediaPackageVodClientMock.On("CreatePackagingConfiguration", mock.Anything).Return(&mediapackagevod.CreatePackagingConfigurationOutput{}, nil)
			cloudFrontClientMock.On("GetDistributionConfig", mock.Anything).Return(&getDistributionConfigOutputMock, nil)

			duplicateConfigParams := map[string]interface{}{
				"StackName":               testStackName,
				"GroupId":                 testGroupId,
				"PackagingConfigurations": "HLS,DASH,HLS",
				"DistributionId":          TestDistributionId,
			}

			res, err := mediaPackageCustomResource.Create(duplicateConfigParams)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if res == nil {
				t.Fatal("Expected non-nil response")
			}

			if res.GroupID != testGroupId {
				t.Errorf("Expected GroupID %s, got %s", testGroupId, res.GroupID)
			}

			mediaPackageVodClientMock.AssertNumberOfCalls(t, "CreatePackagingConfiguration", 2)
		})

		t.Run("should succeed when at least one valid configuration is informed", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testGroupId),
				DomainName: aws.String(TestDomainName),
			}
			getDistributionConfigOutputMock := GetTestConfigurationWithMP()

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)
			mediaPackageVodClientMock.On("CreatePackagingConfiguration", mock.Anything).Return(&mediapackagevod.CreatePackagingConfigurationOutput{}, nil)
			cloudFrontClientMock.On("GetDistributionConfig", mock.Anything).Return(&getDistributionConfigOutputMock, nil)

			singleInvalidParams := map[string]interface{}{
				"StackName":               testStackName,
				"GroupId":                 testGroupId,
				"PackagingConfigurations": "HLS,sickduck",
				"DistributionId":          TestDistributionId,
			}

			res, err := mediaPackageCustomResource.Create(singleInvalidParams)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if res == nil {
				t.Fatal("Expected non-nil response")
			}

			if res.GroupID != testGroupId {
				t.Errorf("Expected GroupID %s, got %s", testGroupId, res.GroupID)
			}
		})

		t.Run("should fail when CreatePackagingGroup fails", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(nil, errors.New("create packaging group error"))

			_, err := mediaPackageCustomResource.Create(ValidParameter)
			if err == nil {
				t.Fatal("Expected error")
			}
			if err.Error() != "MediaPackageCustomResource.Create: create packaging group error"{
				t.Errorf("Expected error message \"MediaPackageCustomResource.Create: create packaging group error\", got \"%v\"", err)
			}
		})

		t.Run("should fail when no valid configuration are informed", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testGroupId),
				DomainName: aws.String(TestDomainName),
			}
			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)

			invalidParameters := map[string]interface{}{
				"StackName":               testStackName,
				"GroupId":                 testGroupId,
				"PackagingConfigurations": "sickduck",
				"DistributionId":          TestDistributionId,
			}

			_, err := mediaPackageCustomResource.Create(invalidParameters)
			if err == nil {
				t.Fatal("Expected error")
			}
		})

		t.Run("should fail when CreatePackagingConfiguration fails", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testStackName),
				DomainName: aws.String(TestDomainName),
			}

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)
			mediaPackageVodClientMock.On("CreatePackagingConfiguration", mock.Anything).Return(nil, errors.New("create packaging configuration error"))

			_, err := mediaPackageCustomResource.Create(ValidParameter)
			if err == nil {
				t.Fatal("Expected error")
			}
			if err.Error() != "MediaPackageCustomResource.CreatePackagingConfiguration: create packaging configuration error"{
				t.Errorf("Expected error message \"c MediaPackageCustomResource.CreatePackagingConfiguration: create packaging configuration error\", got \"%v\"", err)
			}
		})

		t.Run("should fail when GetDistributionConfig fails", func(t *testing.T) {
			cloudFrontClientMock := new(CloudFrontClientMock)
			mediaPackageVodClientMock := new(MediaPackageVodClientMock)

			mediaPackageCustomResource := MediaPackageCustomResource{
				MediaPackageVODClient: mediaPackageVodClientMock,
				CloudFrontHelper:      CloudFrontHelper{CloudFrontClient: cloudFrontClientMock},
			}

			testGroupResponse := mediapackagevod.CreatePackagingGroupOutput{
				Id:         aws.String(testStackName),
				DomainName: aws.String(TestDomainName),
			}

			mediaPackageVodClientMock.On("CreatePackagingGroup", mock.Anything).Return(&testGroupResponse, nil)
			mediaPackageVodClientMock.On("CreatePackagingConfiguration", mock.Anything).Return(&mediapackagevod.CreatePackagingConfigurationOutput{}, nil)
			cloudFrontClientMock.On("GetDistributionConfig", mock.Anything).Return(nil, errors.New("get distribution config error"))

			_, err := mediaPackageCustomResource.Create(ValidParameter)
			if err == nil {
				t.Fatal("Expected error")
			}
			if err.Error() != "MediaPackageCustomResource.Create: AddCustomOrigin: CloudFrontHelper.AddCustomOrigin: GetDistributionConfig: get distribution config error"{
				t.Errorf("MediaPackageCustomResource.Create: AddCustomOrigin: CloudFrontHelper.AddCustomOrigin: GetDistributionConfig: get distribution config error\", got \"%v\"", err)
			} 
		})
	})

}
