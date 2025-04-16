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
	os.Setenv("EncryptionKey", "................................")
	ctx := context.Background()
	t.Run("should success get video with default pageNumber", func(t *testing.T) {
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

		dynamoDBClient.On("Scan", mock.Anything, mock.Anything).Return(&dynamodb.ScanOutput{
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
				"nextToken":  "LUskRG2xm2xv7IHa8EcqhEiB9d8wIEfEsk2w7mz1SaT5100l5LL2kN8JiLR_nuqrLEf5eMSqI5hO0NDDUNfDg_z0Ygn3-sDlLS5Rdiw_mRatSyh5IdCafUiBJaVGJbcbUhOXYSgVI141kv21AYvwlh_yPcRlfhJOLA==",
			},
		}

		dynamoDBClient.On("Scan", mock.Anything, mock.Anything).Once().Return(&dynamodb.ScanOutput{
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

		dynamoDBClient.On("Scan", mock.Anything, mock.Anything).Once().Return(&dynamodb.ScanOutput{
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
		
	})
}
