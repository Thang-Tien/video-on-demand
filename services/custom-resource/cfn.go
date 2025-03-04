package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

type CfnResponseBody struct {
	Status             string                 `json:"Status"`
	Reason             string                 `json:"Reason"`
	PhysicalResourceId string                 `json:"PhysicalResourceId"`
	StackId            string                 `json:"StackId"`
	RequestId          string                 `json:"RequestId"`
	LogicalResourceId  string                 `json:"LogicalResourceId"`
	Data               CustomResourceResponse `json:"Data"`
}

type CfnClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type CfnCustomResource struct {
	CfnClient CfnClient
}

func (c *CfnCustomResource) Send(event cfn.Event, responesStatus string, resposeData CustomResourceResponse) (*int, error) {
	body := CfnResponseBody{
		Status:             responesStatus,
		Reason:             "See the details in CloudWatch Log Stream: " + lambdacontext.LogStreamName,
		PhysicalResourceId: event.LogicalResourceID,
		StackId:            event.StackID,
		RequestId:          event.RequestID,
		LogicalResourceId:  event.LogicalResourceID,
		Data:               resposeData,
	}

	// Convert the response body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	// Create a buffer with the JSON body
	bodyReader := bytes.NewBuffer(jsonBody)

	req, err := http.NewRequest("PUT", event.ResponseURL, io.NopCloser(bodyReader))
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(jsonBody)))

	resp, err := c.CfnClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("CfnCustomResource.Send: Do: failed to send cfn response: %d", resp.StatusCode)
	}

	return &resp.StatusCode, nil
}
