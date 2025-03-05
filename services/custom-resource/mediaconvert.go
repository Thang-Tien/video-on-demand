package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/mitchellh/mapstructure"
)

type Preset struct {
	Name string
	File string
}

type Template struct {
	Name string
	File string
}

var qvbrPresets = []Preset{
	{
		Name: "_Mp4_Avc_Aac_16x9_1280x720p_4.5Mbps_qvbr",
		File: "./presets/_Mp4_Avc_Aac_16x9_1280x720p_4.5Mbps_qvbr.json",
	},
	{
		Name: "_Mp4_Avc_Aac_16x9_1920x1080p_6Mbps_qvbr",
		File: "./presets/_Mp4_Avc_Aac_16x9_1920x1080p_6Mbps_qvbr.json",
	},
	{
		Name: "_Mp4_Hevc_Aac_16x9_3840x2160p_20Mbps_qvbr",
		File: "./presets/_Mp4_Hevc_Aac_16x9_3840x2160p_20Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_1280x720p_6.5Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_1280x720p_6.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_480x270p_0.4Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_480x270p_0.4Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_1920x1080p_8.5Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_1920x1080p_8.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_640x360p_0.6Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_640x360p_0.6Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_1280x720p_3.5Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_1280x720p_3.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_640x360p_1.2Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_640x360p_1.2Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_1280x720p_5.0Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_1280x720p_5.0Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Dash_Mp4_Avc_16x9_960x540p_3.5Mbps_qvbr",
		File: "./presets/_Ott_Dash_Mp4_Avc_16x9_960x540p_3.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_3.5Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_3.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_480x270p_0.4Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_480x270p_0.4Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_5.0Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_5.0Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_640x360p_0.6Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_640x360p_0.6Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_6.5Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_1280x720p_6.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_640x360p_1.2Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_640x360p_1.2Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_1920x1080p_8.5Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_1920x1080p_8.5Mbps_qvbr.json",
	},
	{
		Name: "_Ott_Hls_Ts_Avc_Aac_16x9_960x540p_3.5Mbps_qvbr",
		File: "./presets/_Ott_Hls_Ts_Avc_Aac_16x9_960x540p_3.5Mbps_qvbr.json",
	},
}

var qvbrTemplates = []Template{
	{
		Name: "_Ott_2160p_Avc_Aac_16x9_qvbr",
		File: "./templates/2160p_avc_aac_16x9_qvbr.json",
	},
	{
		Name: "_Ott_1080p_Avc_Aac_16x9_qvbr",
		File: "./templates/1080p_avc_aac_16x9_qvbr.json",
	},
	{
		Name: "_Ott_720p_Avc_Aac_16x9_qvbr",
		File: "./templates/720p_avc_aac_16x9_qvbr.json",
	},
}

var mediaPackageTemplates = []Template{
	{
		Name: "_Ott_2160p_Avc_Aac_16x9_mvod",
		File: "./templates/2160p_avc_aac_16x9_mvod.json",
	},
	{
		Name: "_Ott_1080p_Avc_Aac_16x9_mvod",
		File: "./templates/1080p_avc_aac_16x9_mvod.json",
	},
	{
		Name: "_Ott_720p_Avc_Aac_16x9_mvod",
		File: "./templates/720p_avc_aac_16x9_mvod.json",
	},
}

var qvbrTemplatesNoPreset = []Template{
	{
		Name: "_Ott_2160p_Avc_Aac_16x9_qvbr_no_preset",
		File: "./templates/2160p_avc_aac_16x9_qvbr_no_preset.json",
	},
	{
		Name: "_Ott_1080p_Avc_Aac_16x9_qvbr_no_preset",
		File: "./templates/1080p_avc_aac_16x9_qvbr_no_preset.json",
	},
	{
		Name: "_Ott_720p_Avc_Aac_16x9_qvbr_no_preset",
		File: "./templates/720p_avc_aac_16x9_qvbr_no_preset.json",
	},
}

var mediaPackageTemplatesNoPreset = []Template{
	{
		Name: "_Ott_2160p_Avc_Aac_16x9_mvod_no_preset",
		File: "./templates/2160p_avc_aac_16x9_mvod_no_preset.json",
	},
	{
		Name: "_Ott_1080p_Avc_Aac_16x9_mvod_no_preset",
		File: "./templates/1080p_avc_aac_16x9_mvod_no_preset.json",
	},
	{
		Name: "_Ott_720p_Avc_Aac_16x9_mvod_no_preset",
		File: "./templates/720p_avc_aac_16x9_mvod_no_preset.json",
	},
}

type MediaConvertCustomResource struct {
	MediaConvertClient MediaConvertClient
}

type MediaConvertClient interface {
	DescribeEndpoints(input *mediaconvert.DescribeEndpointsInput) (*mediaconvert.DescribeEndpointsOutput, error)
	CreateJobTemplate(input *mediaconvert.CreateJobTemplateInput) (*mediaconvert.CreateJobTemplateOutput, error)
}

func (m *MediaConvertCustomResource) GetEndpoint() (*string, error) {
	// res, err := m.MediaConvertClient.DescribeEndpoints(&mediaconvert.DescribeEndpointsInput{
	// 	MaxResults: aws.Int64(1),
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("MediaConvertCustomResource.GetEndpoint: DescribeEndpoints: %w", err)
	// }

	// return res.Endpoints[0].Url, nil
	return aws.String("https://ap-southeast-2.mediaconvert.amazonaws.com"), nil
}

type MediaConvertConfig struct {
	ServiceToken       string
	EnableMediaPackage string
	Resource           string
	EnableNewTemplates string
	EndPoint           string
	StackName          string
}

func (m *MediaConvertCustomResource) CreateTemplates(config map[string]interface{}) error {

	var mediaConvertConfig MediaConvertConfig
	if err := mapstructure.Decode(config, &mediaConvertConfig); err != nil {
		return fmt.Errorf("MediaConvertCustomResource.CreateTemplates: Decode: failed to decode config: %w", err)
	}

	for _, template := range mediaPackageTemplatesNoPreset {
		templateJSON, err := os.ReadFile(template.File)
		if err != nil {
			log.Printf("MediaConvertCustomResource.CreateTemplates: ReadFile: Error reading template file %s: %v\n", template.File, err)
			continue
		}

		input := &mediaconvert.CreateJobTemplateInput{}
		err = json.Unmarshal(templateJSON, input)
		if err != nil {
			return fmt.Errorf("MediaConvertCustomResource.CreateTemplates: Unmarshal: %w", err)

		}
		input.Name = aws.String(mediaConvertConfig.StackName + template.Name)
		input.Tags = map[string]*string{
			"SolutionId": aws.String("SO0021"),
		}

		_, err = m.MediaConvertClient.CreateJobTemplate(input)
		if err != nil {
			log.Printf("MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: Error creating template %s: %v", template.Name, err)
			return fmt.Errorf("MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: %w", err)
		}
	}

	for _, template := range qvbrTemplatesNoPreset {
		templateJSON, err := os.ReadFile(template.File)
		if err != nil {
			log.Printf("MediaConvertCustomResource.CreateTemplates: ReadFile: Error reading template file %s: %v\n", template.File, err)
			continue
		}

		input := &mediaconvert.CreateJobTemplateInput{}
		err = json.Unmarshal(templateJSON, input)
		if err != nil {
			return fmt.Errorf("MediaConvertCustomResource.CreateTemplates: Unmarshal: Error unmarshalling template file %s: %v", template.File, err)

		}
		input.Name = aws.String(mediaConvertConfig.StackName + template.Name)
		input.Tags = map[string]*string{
			"SolutionId": aws.String("SO0021"),
		}

		_, err = m.MediaConvertClient.CreateJobTemplate(input)
		if err != nil {
			log.Printf("MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: Error creating template %s: %v", template.Name, err)
			return fmt.Errorf("MediaConvertCustomResource.CreateTemplates: CreateJobTemplate: %w", err)
		}
	}

	return nil

}
