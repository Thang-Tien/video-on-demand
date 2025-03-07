package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/mitchellh/mapstructure"
)

type MetricCustomResource struct {
	MetricClient MetricClient
}

type MetricClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

type MetricCustomResourceConfig struct {
	SolutionId           string
	Version              string
	UUID                 string
	SendAnonymizedMetric string
	Glacier              string
	EnableMediaPackage   string
	FrameCapture         string
	WorkflowTrigger      string
	Transcoder           string
	ServiceToken         string
	Resource             string
}

type Metric struct {
	Solution  string
	UUID      string
	TimeStamp time.Time
	Data      MetricData
}

type MetricData struct {
	SolutionId           string
	Version              string
	UUID                 string
	SendAnonymizedMetric string
	Glacier              string
	EnableMediaPackage   string
	FrameCapture         string
	WorkflowTrigger      string
	Transcoder           string
}

func (m *MetricCustomResource) Send(config map[string]interface{}) (*string, error) {

	// convert map to struct
	var metricConfig MetricCustomResourceConfig
	if err := mapstructure.Decode(config, &metricConfig); err != nil {
		return nil, fmt.Errorf("MetricCustomResource.Send: Decode: error decoding config: %v", err)
	}

	metrics := Metric{
		Solution:  metricConfig.SolutionId,
		UUID:      metricConfig.UUID,
		TimeStamp: time.Now().UTC(),
		Data: MetricData{
			SolutionId:           metricConfig.SolutionId,
			Version:              metricConfig.Version,
			UUID:                 metricConfig.UUID,
			SendAnonymizedMetric: metricConfig.SendAnonymizedMetric,
			Glacier:              metricConfig.Glacier,
			EnableMediaPackage:   metricConfig.EnableMediaPackage,
			FrameCapture:         metricConfig.FrameCapture,
			WorkflowTrigger:      metricConfig.WorkflowTrigger,
			Transcoder:           metricConfig.Transcoder,
		},
	}

	// convert to JSON
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("MetricCustomResource.Marshal: error marshalling metrics: %v", err)
	}

	// send HTTP request
	resp, err := m.MetricClient.Post(
		"https://metrics.awssolutionsbuilder.com/generic",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("MetricCustomResource.Send: Post: error sending metrics: %v", err)
	}
	defer resp.Body.Close()

	// return status code as string
	return aws.String(strconv.Itoa(resp.StatusCode)), nil
}
