package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediapackagevod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MediaPackageVodClientMock struct {
	mock.Mock
}

func (m *MediaPackageVodClientMock) CreateAsset(input *mediapackagevod.CreateAssetInput) (*mediapackagevod.CreateAssetOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mediapackagevod.CreateAssetOutput), args.Error(1)
}

const domainName = "https://random-id.egress.mediapackage-vod.ap-southeast-1.amazonaws.com"

func TestMediaPackageAssets(t *testing.T) {
	os.Setenv("DistributionId", "distributionId")
	os.Setenv("GroupId", "groupId")
	os.Setenv("GroupDomainName", domainName)
	os.Setenv("MediaPackageVodRole", "role")

	t.Run("should success with valid parameters", func(t *testing.T) {
		event := MediaPackageAssetsEvent{
			GUID:        "guid",
			SrcVideo:    "video.mp4",
			HlsPlaylist: aws.String("s3://my-bucket/video.m3u8"),
			CloudFront:  "random-id.cloudfront.net",
		}

		createAssetResponse := mediapackagevod.CreateAssetOutput{
			Arn: aws.String("arn"),
			EgressEndpoints: []*mediapackagevod.EgressEndpoint{
				{
					PackagingConfigurationId: aws.String("packaging-config-hls"),
					Url:                      aws.String(fmt.Sprintf("%s/out/index.m3u8", domainName)),
				},
				{
					PackagingConfigurationId: aws.String("packaging-config-dash"),
					Url:                      aws.String(fmt.Sprintf("%s/out/index.mpd", domainName)),
				},
			},
			Id:               aws.String("asset-id"),
			PackagingGroupId: aws.String("packaging-group-id"),
			ResourceId:       aws.String("resource-id"),
			SourceArn:        aws.String("source-file-arn"),
			SourceRoleArn:    aws.String("source-role-arn"),
		}

		mediaPackageVodClientMock := new(MediaPackageVodClientMock)
		handler := &Handler{
			MediaPackageVodClient: mediaPackageVodClientMock,
		}

		mediaPackageVodClientMock.On("CreateAsset", mock.Anything).Return(&createAssetResponse, nil)

		res, err := handler.HanleRequest(event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, event.GUID, res.GUID)
		assert.Equal(t, event.SrcVideo, res.SrcVideo)
		assert.Equal(t, "https://random-id.cloudfront.net/out/index.m3u8", res.EgressEndpoints["HLS"])
		assert.Equal(t, "https://random-id.cloudfront.net/out/index.mpd", res.EgressEndpoints["DASH"])
	})

	t.Run("should fail when CreateAsset fails", func(t *testing.T) {
		event := MediaPackageAssetsEvent{
			GUID:        "guid",
			SrcVideo:    "video.mp4",
			HlsPlaylist: aws.String("s3://my-bucket/video.m3u8"),
		}

		mediaPackageVodClientMock := new(MediaPackageVodClientMock)
		handler := &Handler{
			MediaPackageVodClient: mediaPackageVodClientMock,
		}

		mediaPackageVodClientMock.On("CreateAsset", mock.Anything).Return(nil, assert.AnError)

		_, err := handler.HanleRequest(event)
		assert.NotNil(t, err)
	})

	t.Run("should correctly parse s3Uri without subfolders", func(t *testing.T) {
		arn, err := buildArnFromUri("s3://my-bucket/video.m3u8")
		assert.Nil(t, err)
		assert.Equal(t, "arn:aws:s3:::my-bucket/video.m3u8", arn)
	})
	t.Run("should correctly parse s3Uri with subfolders", func(t *testing.T) {
		arn, err := buildArnFromUri("s3://my-bucket/output/hls/video.m3u8")
		assert.Nil(t, err)
		assert.Equal(t, "arn:aws:s3:::my-bucket/output/hls/video.m3u8", arn)
	})
	t.Run("should fail when s3Uri is not in correct format", func(t *testing.T) {
		_, err := buildArnFromUri("my-bucket/video.m3u8")
		assert.NotNil(t, err)
	})
}
