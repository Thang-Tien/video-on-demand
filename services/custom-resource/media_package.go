package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediapackagevod"
	"github.com/mitchellh/mapstructure"
)

const DEFAULT_SEGMENT_LENGTH = 6
const DEFAULT_PROGRAM_DATETIME_INTERVAL = 60
const DEFAULT_MANIFEST_NAME = "index"

type MediaPackageVodClient interface {
	CreatePackagingGroup(input *mediapackagevod.CreatePackagingGroupInput) (*mediapackagevod.CreatePackagingGroupOutput, error)
	CreatePackagingConfiguration(input *mediapackagevod.CreatePackagingConfigurationInput) (*mediapackagevod.CreatePackagingConfigurationOutput, error)
}

type MediaPackageCustomResource struct {
	MediaPackageVODClient MediaPackageVodClient
	CloudFrontHelper      CloudFrontHelper
}

type MediaPackageCustomResourceConfig struct {
	ServiceToken            string
	EnableMediaPackage      string
	Resource                string
	DistributionId          string
	PackagingConfigurations string
	StackName               string
	GroupId                 string
}

type MediaPackageResponse struct {
	GroupID         string
	GroupDomainName string
}

func getHlsParameter(groupId, configId string) *mediapackagevod.CreatePackagingConfigurationInput {
	return &mediapackagevod.CreatePackagingConfigurationInput{
		Id:               aws.String(configId),
		PackagingGroupId: aws.String(groupId),
		HlsPackage: &mediapackagevod.HlsPackage{
			HlsManifests: []*mediapackagevod.HlsManifest{
				{
					AdMarkers:                      aws.String("SCTE35_ENHANCED"),
					IncludeIframeOnlyStream:        aws.Bool(false),
					ManifestName:                   aws.String(DEFAULT_MANIFEST_NAME),
					ProgramDateTimeIntervalSeconds: aws.Int64(DEFAULT_PROGRAM_DATETIME_INTERVAL),
					RepeatExtXKey:                  aws.Bool(false),
				},
			},
			SegmentDurationSeconds: aws.Int64(DEFAULT_SEGMENT_LENGTH),
			UseAudioRenditionGroup: aws.Bool(true),
		},
	}
}

func getDashParameter(groupId, configId string) *mediapackagevod.CreatePackagingConfigurationInput {
	return &mediapackagevod.CreatePackagingConfigurationInput{
		Id:               aws.String(configId),
		PackagingGroupId: aws.String(groupId),
		DashPackage: &mediapackagevod.DashPackage{
			DashManifests: []*mediapackagevod.DashManifest{
				{
					ManifestName:         aws.String(DEFAULT_MANIFEST_NAME),
					MinBufferTimeSeconds: aws.Int64(DEFAULT_SEGMENT_LENGTH * 3),
					Profile:              aws.String("NONE"),
				},
			},
			SegmentDurationSeconds: aws.Int64(DEFAULT_SEGMENT_LENGTH),
		},
	}
}

func getMssParameter(groupId, configId string) *mediapackagevod.CreatePackagingConfigurationInput {
	return &mediapackagevod.CreatePackagingConfigurationInput{
		Id:               aws.String(configId),
		PackagingGroupId: aws.String(groupId),
		MssPackage: &mediapackagevod.MssPackage{
			MssManifests: []*mediapackagevod.MssManifest{
				{
					ManifestName: aws.String(DEFAULT_MANIFEST_NAME),
				},
			},
			SegmentDurationSeconds: aws.Int64(DEFAULT_SEGMENT_LENGTH),
		},
	}
}

func getCmafParameter(groupId, configId string) *mediapackagevod.CreatePackagingConfigurationInput {
	return &mediapackagevod.CreatePackagingConfigurationInput{
		Id:               aws.String(configId),
		PackagingGroupId: aws.String(groupId),
		CmafPackage: &mediapackagevod.CmafPackage{
			HlsManifests: []*mediapackagevod.HlsManifest{
				{
					AdMarkers:                      aws.String("SCTE35_ENHANCED"),
					IncludeIframeOnlyStream:        aws.Bool(false),
					ManifestName:                   aws.String(DEFAULT_MANIFEST_NAME),
					ProgramDateTimeIntervalSeconds: aws.Int64(DEFAULT_PROGRAM_DATETIME_INTERVAL),
					RepeatExtXKey:                  aws.Bool(false),
				},
			},
			SegmentDurationSeconds: aws.Int64(DEFAULT_SEGMENT_LENGTH),
		},
	}
}

func (m *MediaPackageCustomResource) Create(properties map[string]interface{}) (*MediaPackageResponse, error) {
	var mediaPackageConfig MediaPackageCustomResourceConfig
	if err := mapstructure.Decode(properties, &mediaPackageConfig); err != nil {
		return nil, fmt.Errorf("MediaPackageCustomResource.Create: Decode: error decoding config: %v", err)
	}

	// Create a random ID by generating 8 random bytes and converting to hex
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	randomId := hex.EncodeToString(randomBytes)

	packagingGroup, err := m.MediaPackageVODClient.CreatePackagingGroup(&mediapackagevod.CreatePackagingGroupInput{
		Id: aws.String(mediaPackageConfig.GroupId),
		Tags: map[string]*string{
			"SolutionId": aws.String("SO0021"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("MediaPackageCustomResource.Create: %w", err)
	}
	configStr := mediaPackageConfig.PackagingConfigurations
	configsWithDups := strings.Split(configStr, ",")

	// Create a map to track unique configurations
	configMap := make(map[string]bool)
	var configurations []string

	for _, cfg := range configsWithDups {
		// Trim whitespace and convert to lowercase for consistent comparison
		trimmedCfg := strings.ToLower(strings.TrimSpace(cfg))
		if trimmedCfg != "" && !configMap[trimmedCfg] {
			configMap[trimmedCfg] = true
			configurations = append(configurations, trimmedCfg)
		}
	}

	if len(configurations) == 0 {
		return nil, fmt.Errorf("MediaPackageCustomResource.Create: No valid packaging configurations provided")
	}

	created := false

	for _, cfg := range configurations {
		var input *mediapackagevod.CreatePackagingConfigurationInput
		switch strings.ToLower(cfg) {
		case "hls":
			configId := "packaging-config-" + randomId + "-hls"
			input = getHlsParameter(*packagingGroup.Id, configId)
		case "dash":
			configId := "packaging-config-" + randomId + "-dash"
			input = getDashParameter(*packagingGroup.Id, configId)
		case "mss":
			configId := "packaging-config-" + randomId + "-mss"
			input = getMssParameter(*packagingGroup.Id, configId)
		case "cmaf":
			configId := "packaging-config-" + randomId + "-cmaf"
			input = getCmafParameter(*packagingGroup.Id, configId)
		default:
			log.Printf("Unknown packaging configuration: %s", cfg)
			continue
		}

		if input != nil {
			_, err = m.MediaPackageVODClient.CreatePackagingConfiguration(input)
			if err != nil {
				return nil, fmt.Errorf("MediaPackageCustomResource.CreatePackagingConfiguration: %w", err)
			}
			created = true
		}

	}

	if !created {
		return nil, fmt.Errorf("MediaPackageCustomResource.Create: At least one valid packaging configuration must be informed")
	}

	err = m.CloudFrontHelper.AddCustomOrigin(mediaPackageConfig.DistributionId, *packagingGroup.DomainName)
	if err != nil {
		return nil, fmt.Errorf("MediaPackageCustomResource.Create: AddCustomOrigin: %w", err)
	}

	return &MediaPackageResponse{
		GroupID:         *packagingGroup.Id,
		GroupDomainName: *packagingGroup.DomainName,
	}, nil
}
