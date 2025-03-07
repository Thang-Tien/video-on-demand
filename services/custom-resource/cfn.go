package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

type CfnResponseBody struct {
	Status             string            `json:"Status"`
	Reason             string            `json:"Reason"`
	PhysicalResourceId string            `json:"PhysicalResourceId"`
	StackId            string            `json:"StackId"`
	RequestId          string            `json:"RequestId"`
	LogicalResourceId  string            `json:"LogicalResourceId"`
	Data               map[string]string `json:"Data"`
}

type CfnClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type CfnCustomResource struct {
	CfnClient CfnClient
}

func (c *CfnCustomResource) Send(event cfn.Event, responseStatus string, responseData CustomResourceResponse) (*int, error) {
	// Convert CustomResourceResponse to map[string]string
	responseMap := make(map[string]string)
	if responseData.GroupId != nil {
		responseMap["GroupId"] = *responseData.GroupId
	}
	if responseData.GroupDomainName != nil {
		responseMap["GroupDomainName"] = *responseData.GroupDomainName
	}
	if responseData.EndpointUrl != nil {
		responseMap["EndpointUrl"] = *responseData.EndpointUrl
	}
	if responseData.UUID != nil {
		responseMap["UUID"] = *responseData.UUID
	}
	body := CfnResponseBody{
		Status:             responseStatus,
		Reason:             "See the details in CloudWatch Log Stream: " + lambdacontext.LogStreamName,
		PhysicalResourceId: event.LogicalResourceID,
		StackId:            event.StackID,
		RequestId:          event.RequestID,
		LogicalResourceId:  event.LogicalResourceID,
		Data:               responseMap,
	}

	// Convert the response body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("CfnCustomResource.Send: Marshal: %w", err)
	}

	req, err := http.NewRequest("PUT", event.ResponseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.CfnClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CfnCustomResource.Send: Do: %w", err)
	}
	defer resp.Body.Close()

	// Log the response body
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("CfnCustomResource.Send: ReadAll: failed to read response body: %v", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "application/json" {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBodyBytes, "", "  "); err != nil {
			log.Printf("Failed to pretty-print JSON: %v", err)
			log.Printf("Response body: %s", respBodyBytes)
		} else {
			log.Printf("Response body:\n%s", prettyJSON.String())
		}
	} else if contentType == "text/html" {
		log.Printf("Response body:\n%s", respBodyBytes)
	} else {
		log.Printf("Response body: %s", respBodyBytes)
	}

	// Recreate the response body
	resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("CfnCustomResource.Send: Do: failed to send cfn response: %d, response body: %s", resp.StatusCode, string(respBodyBytes))
	}

	return &resp.StatusCode, nil
}
