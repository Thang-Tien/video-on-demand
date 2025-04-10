package main

import (
	"context"
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

func (m *DynamoDBClientMock) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.ScanOutput), args.Error(1)
}

func TestGetAllVideo(t *testing.T) {
	os.Setenv("EncryptionKey", ".......nguyenhoangdieuanh.......")
	t.Run("should success get items", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{}

		dynamoDBClient.On("Scan", mock.Anything, mock.Anything).Return(&dynamodb.ScanOutput{
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

		res, err := handler.HandleRequest(context.Background(), event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
	})

	t.Run("should success get item with nextToken", func(t *testing.T) {
		dynamoDBClient := &DynamoDBClientMock{}
		handler := &Handler{
			DynamoDBClient: dynamoDBClient,
		}

		event := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"nextToken": "gA-thvPsbyCAD2HWzfPF_XhrTjhbDOH6jsdzhwRQ5lyn6Q2hHiSP2iKJr37d6TRtK9LNb5k1ZiKtd0LWjMfl5r2l1pcqFSAyW49qeQHz_jkzfIwlWeCuWhOrzsABDkYC2wtlrNgKqOkQuA==",
			},
		}

		dynamoDBClient.On("Scan", mock.Anything, mock.Anything).Return(&dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{
				{
					"PK": &types.AttributeValueMemberS{Value: "VIDEO#123"},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
			},
		}, nil)

		res, err := handler.HandleRequest(context.Background(), event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
	})
}
