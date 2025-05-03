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

func (m *DynamoDBClientMock) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}
func TestGetTrendingVideo(t *testing.T) {
	os.Setenv("EncryptionKey", "................................")
	t.Run("should success get trending video", func(t *testing.T) {
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

		dynamoDBClient.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{}, nil)

		res, err := handler.HandleRequest(context.Background(), event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, 200, res.StatusCode)
		log.Printf("Response: %s", res.Body)
	})
}
