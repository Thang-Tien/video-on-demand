package main

import "github.com/aws/aws-sdk-go/aws"

var (
	CmafMss = EventDetail{
		Queue: "arn:aws:mediaconvert:us-east-1::queues/Default",
		JobId: "htprrb",
		UserMetadata: UserMetadata{
			Workflow: "CMAF",
			GUID:     "guid",
		},
		OutputGroupDetails: []*OutputGroupDetail{
			{
				PlaylistFilePaths: []*string{
					aws.String("s3://vod-destination/12345/cmaf/big_bunny.mpd"),
					aws.String("s3://vod-destination/12345/cmaf/big_bunny.m3u8"),
				},
				Type: "CMAF_GROUP",
			},
			{
				OutputDetails: []*OutputDetail{
					{
						OutputFilePaths: []*string{
							aws.String("s3://vod-destination/12345/mss/big_bunny.ismv"),
						},
					},
				},
				PlaylistFilePaths: []*string{
					aws.String("s3://vod-destination/12345/mss/big_bunny.ism"),
				},
				Type: "MS_SMOOTH_GROUP",
			},
		},
	}

	HlsDash = EventDetail{
		Queue: "arn:aws:mediaconvert:us-east-1::queues/Default",
		JobId: "htprrb",
		UserMetadata: UserMetadata{
			Workflow: "vod10",
			GUID:     "guid",
		},
		OutputGroupDetails: []*OutputGroupDetail{
			{
				PlaylistFilePaths: []*string{
					aws.String("s3://vod-destination/12345/hls/dude.m3u8"),
				},
				Type: "HLS_GROUP",
			},
			{
				PlaylistFilePaths: []*string{
					aws.String("s3://vod-destination/12345/dash/dude.mpd"),
				},
				Type: "DASH_ISO_GROUP",
			},
		},
	}

	Mp4 = EventDetail{
		Queue:  "arn:aws:mediaconvert:us-east-1::queues/Default",
		JobId:  "htprrb",
		Status: "COMPLETE",
		UserMetadata: UserMetadata{
			Workflow: "vod10",
			GUID:     "guid",
		},
		OutputGroupDetails: []*OutputGroupDetail{
			{
				OutputDetails: []*OutputDetail{
					{
						OutputFilePaths: []*string{
							aws.String("s3://vod-destination/12345/mp4/dude_3.0Mbps.mp4"),
						},
						DurationInMs: 13471,
						VideoDetails: &VideoDetail{
							WidthInPx:  1280,
							HeightInPx: 720,
						},
					},
				},
				Type: "FILE_GROUP",
			},
		},
	}
)
