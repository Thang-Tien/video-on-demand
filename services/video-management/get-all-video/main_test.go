package main

import (
	"context"
	"log"
	"os"
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

func (m *DynamoDBClientMock) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func TestGetAllVideo(t *testing.T) {
	os.Setenv("EncryptionKey", "................................")
	ctx := context.Background()
	t.Run("should success get video with default pageNumber", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{}

		dynamoDBClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{
					"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
			},
			LastEvaluatedKey: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
		}, nil)

		res, err := handler.HandleRequest(ctx, event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
	})

	t.Run("should success get video with pageNumber", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"pageNumber": "1",
			},
		}

		dynamoDBClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{
					"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
			},
		}, nil)

		res, err := handler.HandleRequest(ctx, event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
	})

	t.Run("should success get video with pageNumber and nextToken", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"pageNumber": "3",
				"nextToken":  "z6oWbz7VJKtpQJ9-J0hmMjS1V4jjeLWeFNWR0N7zHekbbnviX_dx_0vuHUCInQEvDLCGt59XGjj3_Hegx6SUBBsILc5gTvEsd625oSyjHrH6ELfYlEdTL0PYWBwYW3oT7giEDvIxJHiPkXE4qvMV_tee3OQIwbBF3w==",
			},
		}

		dynamoDBClient.On("Query", mock.Anything, mock.Anything).Once().Return(&dynamodb.QueryOutput{
			LastEvaluatedKey: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
			Items: []map[string]types.AttributeValue{
				{
					"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
			},
		}, nil)

		dynamoDBClient.On("Query", mock.Anything, mock.Anything).Once().Return(&dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{
					"PK": &types.AttributeValueMemberS{Value: "VIDEO#456"},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
			},
		}, nil)

		res, err := handler.HandleRequest(ctx, event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
		log.Printf("Response: %s", res.Body)
	})
}
