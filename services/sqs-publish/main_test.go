package main

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type SqsClientMock struct {
	mock.Mock
}

func (m *SqsClientMock) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func TestSqsPublish(t *testing.T) {
	os.Setenv("SqsQueue", "https://sqs.amazonaws.com/1234")
	t.Run("should success on sqs send message", func(t *testing.T) {
		sqsClientMock := new(SqsClientMock)
		handler := &Handler{
			SqsClient: sqsClientMock,
		}

		event := SqsPublishEvent{
			GUID:      "guid",
			StartTime: "2025-01-01T00:00:00Z",
			SrcVideo:  "video.mp4",
		}

		sqsClientMock.On("SendMessage", mock.Anything).Return(&sqs.SendMessageOutput{}, nil)

		res, err := handler.HandleRequest(event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, "video.mp4", res.SrcVideo)
	})
	t.Run("should fail on sqs send message fails", func(t *testing.T) {
		sqsClientMock := new(SqsClientMock)
		handler := &Handler{
			SqsClient: sqsClientMock,
		}

		event := SqsPublishEvent{
			GUID:      "guid",
			StartTime: "2025-01-01T00:00:00Z",
			SrcVideo:  "video.mp4",
		}

		sqsClientMock.On("SendMessage", mock.Anything).Return(nil, assert.AnError)

		_, err := handler.HandleRequest(event)
		assert.Error(t, err)
		
	})
}
