package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DynamoDBClientMock struct {
	mock.Mock
}

func (m *DynamoDBClientMock) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

func TestUpdateInformation(t *testing.T) {
	t.Run("should success update item", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"pk": "123",
			},
			Body: `{"title":"Test Movie"}`,
		}

		dynamoDBClient.On("UpdateItem", mock.Anything, mock.Anything).Return(&dynamodb.UpdateItemOutput{
			Attributes: map[string]types.AttributeValue{},
		}, nil)

		res, err := handler.HandleRequest(event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
	})

	t.Run("should return error when UpdateItem fails", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"pk": "123",
			},
			Body: `{"title":"Test Movie"}`,
		}

		dynamoDBClient.On("UpdateItem", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		res, _ := handler.HandleRequest(event)
		assert.Equal(t, 500, res.StatusCode)
		assert.Equal(t, res.Body, fmt.Sprintf("Failed to update item in DynamoDB: %v", assert.AnError.Error()))
	})

	t.Run("should return error when request body is invalid", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"pk": "123",
			},
			Body: `invalid-json`,
		}

		res, _ := handler.HandleRequest(event)
		assert.Equal(t, 400, res.StatusCode)
	})
}
