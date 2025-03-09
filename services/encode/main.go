package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"dario.cat/mergo"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
)

type EncodeInput struct {
	GUID                   string `json:"guid"`
	StartTime              string `json:"startTime"`
	WorkflowTrigger        string `json:"workflowTrigger"`
	WorkflowStatus         string `json:"workflowStatus"`
	WorkflowName           string `json:"workflowName"`
	SrcBucket              string `json:"srcBucket"`
	DestBucket             string `json:"destBucket"`
	CloudFront             string `json:"cloudFront"`
	FrameCapture           bool   `json:"frameCapture"`
	ArchiveSource          string `json:"archiveSource"`
	JobTemplate2160p       string `json:"jobTemplate_2160p"`
	JobTemplate1080p       string `json:"jobTemplate_1080p"`
	JobTemplate720p        string `json:"jobTemplate_720p"`
	InputRotate            string `json:"inputRotate"`
	AcceleratedTranscoding string `json:"acceleratedTranscoding"`
	EnableSns              bool   `json:"enableSns"`
	EnableSqs              bool   `json:"enableSqs"`
	SrcVideo               string `json:"srcVideo"`
	EnableMediaPackage     bool   `json:"enableMediaPackage"`
	SrcMediainfo           string `json:"srcMediainfo"`
	SrcHeight              int    `json:"srcHeight"`
	SrcWidth               int    `json:"srcWidth"`
	EncodingProfile        int    `json:"encodingProfile"`
	FrameCaptureHeight     int    `json:"frameCaptureHeight"`
	FrameCaptureWidth      int    `json:"frameCaptureWidth"`
	JobTemplate            string `json:"jobTemplate"`
	IsCustomTemplate       bool   `json:"isCustomTemplate"`
}

type EncodeResponse struct {
	GUID                   string                      `json:"guid"`
	StartTime              string                      `json:"startTime"`
	WorkflowTrigger        string                      `json:"workflowTrigger"`
	WorkflowStatus         string                      `json:"workflowStatus"`
	WorkflowName           string                      `json:"workflowName"`
	SrcBucket              string                      `json:"srcBucket"`
	DestBucket             string                      `json:"destBucket"`
	CloudFront             string                      `json:"cloudFront"`
	FrameCapture           bool                        `json:"frameCapture"`
	ArchiveSource          string                      `json:"archiveSource"`
	JobTemplate2160p       string                      `json:"jobTemplate_2160p"`
	JobTemplate1080p       string                      `json:"jobTemplate_1080p"`
	JobTemplate720p        string                      `json:"jobTemplate_720p"`
	InputRotate            string                      `json:"inputRotate"`
	AcceleratedTranscoding string                      `json:"acceleratedTranscoding"`
	EnableSns              bool                        `json:"enableSns"`
	EnableSqs              bool                        `json:"enableSqs"`
	SrcVideo               string                      `json:"srcVideo"`
	EnableMediaPackage     bool                        `json:"enableMediaPackage"`
	SrcMediainfo           string                      `json:"srcMediainfo"`
	SrcHeight              int                         `json:"srcHeight"`
	SrcWidth               int                         `json:"srcWidth"`
	EncodingProfile        int                         `json:"encodingProfile"`
	FrameCaptureHeight     int                         `json:"frameCaptureHeight"`
	FrameCaptureWidth      int                         `json:"frameCaptureWidth"`
	JobTemplate            string                      `json:"jobTemplate"`
	IsCustomTemplate       bool                        `json:"isCustomTemplate"`
	EncodingJob            mediaconvert.CreateJobInput `json:"encodingJob"`
	EncodeJobId            string                      `json:"encodeJobId"`
}

type MediaConvertClient interface {
	GetJobTemplate(input *mediaconvert.GetJobTemplateInput) (*mediaconvert.GetJobTemplateOutput, error)
	CreateJob(input *mediaconvert.CreateJobInput) (*mediaconvert.CreateJobOutput, error)
}

type Handler struct {
	MediaConvertClient MediaConvertClient
}

func (h *Handler) HandleRequest(event EncodeInput) (*EncodeResponse, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode: main.Handler.HandleRequest: json.Marshal: %w", err)
	}
	log.Printf("REQUEST:: %s", eventJson)

	inputPath := fmt.Sprintf("s3://%s/%s", event.SrcBucket, event.SrcVideo)
	outputPath := fmt.Sprintf("s3://%s/%s", event.DestBucket, event.GUID)

	// init job to create
	job := mediaconvert.CreateJobInput{
		JobTemplate: &event.JobTemplate,
		Role:        aws.String(os.Getenv("MediaConvertRole")),
		UserMetadata: map[string]*string{
			"guid":     aws.String(event.GUID),
			"workflow": aws.String(event.WorkflowName),
		},
		Settings: &mediaconvert.JobSettings{
			Inputs: []*mediaconvert.Input{
				{
					AudioSelectors: map[string]*mediaconvert.AudioSelector{
						"Audio Selector 1": {
							Offset:           aws.Int64(0),
							DefaultSelection: aws.String("NOT_DEFAULT"),
							ProgramSelection: aws.Int64(1),
						},
					},
					VideoSelector: &mediaconvert.VideoSelector{
						ColorSpace: aws.String("FOLLOW"),
						Rotate:     &event.InputRotate,
					},
					FilterEnable:   aws.String("AUTO"),
					PsiControl:     aws.String("USE_PSI"),
					FilterStrength: aws.Int64(0),
					DeblockFilter:  aws.String("DISABLED"),
					DenoiseFilter:  aws.String("DISABLED"),
					TimecodeSource: aws.String("EMBEDDED"),
					FileInput:      &inputPath,
				},
			},
			OutputGroups: []*mediaconvert.OutputGroup{},
		},
	}

	mp4Group := getMp4Group(outputPath)
	hlsGroup := getHlsGroup(outputPath)
	dashGroup := getDashGroup(outputPath)
	cmafGroup := getCmafGroup(outputPath)
	mssGroup := getMssGroup(outputPath)
	frameCaptureGroup := getFrameGroup(event, outputPath)

	template, err := h.MediaConvertClient.GetJobTemplate(&mediaconvert.GetJobTemplateInput{
		Name: aws.String(event.JobTemplate),
	})
	if err != nil {
		return nil, fmt.Errorf("encode: main.Handler.HandleRequest: GetJobTemplate: %w", err)
	}
	templateJson, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("encode: main.Handler.HandleRequest: json.Marshal: %w", err)
	}
	log.Printf("TEMPLATE:: %s", templateJson)

	for _, group := range template.JobTemplate.Settings.OutputGroups {
		found := false
		var defaultGroup *mediaconvert.OutputGroup
		switch *group.OutputGroupSettings.Type {
		case "FILE_GROUP_SETTINGS":
			defaultGroup = mp4Group
			found = true
		case "HLS_GROUP_SETTINGS":
			defaultGroup = hlsGroup
			found = true
		case "DASH_ISO_GROUP_SETTINGS":
			defaultGroup = dashGroup
			found = true
		case "CMAF_GROUP_SETTINGS":
			defaultGroup = cmafGroup
			found = true
		case "MS_SMOOTH_GROUP_SETTINGS":
			defaultGroup = mssGroup
			found = true
		}

		if found {
			log.Printf("%s found in Job Template", *defaultGroup.Name)
			outputGroup := defaultGroup
			err := mergo.MergeWithOverwrite(outputGroup, group)
			if err != nil {
				return nil, fmt.Errorf("encode: main.Handler.HandleRequest: mergo.Merge: %w", err)
			}
			job.Settings.OutputGroups = append(job.Settings.OutputGroups, outputGroup)
		}
	}

	if event.FrameCapture {
		job.Settings.OutputGroups = append(job.Settings.OutputGroups, frameCaptureGroup)
	}

	if event.AcceleratedTranscoding == "PREFERRED" || event.AcceleratedTranscoding == "ENABLED" {
		job.AccelerationSettings = &mediaconvert.AccelerationSettings{
			Mode: aws.String(event.AcceleratedTranscoding),
		}
		job.Settings.TimecodeConfig = &mediaconvert.TimecodeConfig{
			Source: aws.String("ZEROBASED"),
		}
		job.Settings.Inputs[0].TimecodeSource = aws.String("ZEROBASED")
	}

	data, err := h.MediaConvertClient.CreateJob(&job)
	if err != nil {
		return nil, fmt.Errorf("encode: main.Handler.HandleRequest: CreateJob: %w", err)
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode: main.Handler.HandleRequest: json.Marshal: %w", err)
	}
	log.Printf("JOB:: %s", dataJson)

	EncodeReponse := EncodeResponse{
		GUID:                   event.GUID,
		StartTime:              event.StartTime,
		WorkflowTrigger:        event.WorkflowTrigger,
		WorkflowStatus:         event.WorkflowStatus,
		WorkflowName:           event.WorkflowName,
		SrcBucket:              event.SrcBucket,
		DestBucket:             event.DestBucket,
		CloudFront:             event.CloudFront,
		FrameCapture:           event.FrameCapture,
		ArchiveSource:          event.ArchiveSource,
		JobTemplate2160p:       event.JobTemplate2160p,
		JobTemplate1080p:       event.JobTemplate1080p,
		JobTemplate720p:        event.JobTemplate720p,
		InputRotate:            event.InputRotate,
		AcceleratedTranscoding: event.AcceleratedTranscoding,
		EnableSns:              event.EnableSns,
		EnableSqs:              event.EnableSqs,
		SrcVideo:               event.SrcVideo,
		EnableMediaPackage:     event.EnableMediaPackage,
		SrcMediainfo:           event.SrcMediainfo,
		SrcHeight:              event.SrcHeight,
		SrcWidth:               event.SrcWidth,
		EncodingProfile:        event.EncodingProfile,
		FrameCaptureHeight:     event.FrameCaptureHeight,
		FrameCaptureWidth:      event.FrameCaptureWidth,
		JobTemplate:            event.JobTemplate,
		IsCustomTemplate:       event.IsCustomTemplate,
		EncodingJob:            job,
		EncodeJobId:            *data.Job.Id,
	}

	return &EncodeReponse, nil

}

func getMp4Group(outputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		Name: aws.String("File Group"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("FILE_GROUP_SETTINGS"),
			FileGroupSettings: &mediaconvert.FileGroupSettings{
				Destination: aws.String(outputPath + "/mp4/"),
			},
		},
		Outputs: []*mediaconvert.Output{},
	}
}

func getHlsGroup(outputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		Name: aws.String("HLS Group"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("HLS_GROUP_SETTINGS"),
			HlsGroupSettings: &mediaconvert.HlsGroupSettings{
				SegmentLength:    aws.Int64(5),
				MinSegmentLength: aws.Int64(0),
				Destination:      aws.String(outputPath + "/hls/"),
			},
		},
		Outputs: []*mediaconvert.Output{},
	}
}

func getDashGroup(outputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		Name: aws.String("DASH ISO"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("DASH_ISO_GROUP_SETTINGS"),
			DashIsoGroupSettings: &mediaconvert.DashIsoGroupSettings{
				SegmentLength:  aws.Int64(30),
				FragmentLength: aws.Int64(3),
				Destination:    aws.String(outputPath + "/dash/"),
			},
		},
		Outputs: []*mediaconvert.Output{},
	}
}

func getCmafGroup(outputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		Name: aws.String("CMAF"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("CMAF_GROUP_SETTINGS"),
			CmafGroupSettings: &mediaconvert.CmafGroupSettings{
				SegmentLength:  aws.Int64(30),
				FragmentLength: aws.Int64(3),
				Destination:    aws.String(outputPath + "/cmaf/"),
			},
		},
		Outputs: []*mediaconvert.Output{},
	}
}

func getMssGroup(outputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		Name: aws.String("MS Smooth"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("MS_SMOOTH_GROUP_SETTINGS"),
			MsSmoothGroupSettings: &mediaconvert.MsSmoothGroupSettings{
				FragmentLength:   aws.Int64(2),
				ManifestEncoding: aws.String("UTF8"),
				Destination:      aws.String(outputPath + "/mss/"),
			},
		},
		Outputs: []*mediaconvert.Output{},
	}
}

func getFrameGroup(event EncodeInput, ouputPath string) *mediaconvert.OutputGroup {
	return &mediaconvert.OutputGroup{
		CustomName: aws.String("Frame Capture"),
		Name:       aws.String("File Group"),
		OutputGroupSettings: &mediaconvert.OutputGroupSettings{
			Type: aws.String("FILE_GROUP_SETTINGS"),
			FileGroupSettings: &mediaconvert.FileGroupSettings{
				Destination: aws.String(ouputPath + "/thumbnails/"),
			},
		},
		Outputs: []*mediaconvert.Output{
			{
				NameModifier: aws.String("_thumb"),
				ContainerSettings: &mediaconvert.ContainerSettings{
					Container: aws.String("RAW"),
				},
				VideoDescription: &mediaconvert.VideoDescription{
					ColorMetadata:     aws.String("INSERT"),
					AfdSignaling:      aws.String("NONE"),
					Sharpness:         aws.Int64(100),
					Height:            aws.Int64(int64(event.FrameCaptureHeight)),
					RespondToAfd:      aws.String("NONE"),
					TimecodeInsertion: aws.String("DISABLED"),
					Width:             aws.Int64(int64(event.FrameCaptureWidth)),
					ScalingBehavior:   aws.String("DEFAULT"),
					AntiAlias:         aws.String("ENABLED"),
					CodecSettings: &mediaconvert.VideoCodecSettings{
						FrameCaptureSettings: &mediaconvert.FrameCaptureSettings{
							MaxCaptures:          aws.Int64(10000000),
							Quality:              aws.Int64(80),
							FramerateDenominator: aws.Int64(5),
							FramerateNumerator:   aws.Int64(1),
						},
						Codec: aws.String("FRAME_CAPTURE"),
					},
					DropFrameTimecode: aws.String("ENABLED"),
				},
			},
		},
	}
}

func main() {
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(os.Getenv("AWS_REGION")),
		},
	)
	if err != nil {
		log.Fatalf("encode: main: session.NewSession: %v", err)
	}

	mediaConvertClient := mediaconvert.New(sess)

	handler := Handler{
		MediaConvertClient: mediaConvertClient,
	}

	lambda.Start(handler.HandleRequest)
}
