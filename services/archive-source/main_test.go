package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type S3ClientMock struct {
	mock.Mock
}

func (m *S3ClientMock) PutObjectTagging(input *s3.PutObjectTaggingInput) (*s3.PutObjectTaggingOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectTaggingOutput), args.Error(1)
}

func TestArchiveSource(t *testing.T) {
	t.Run("should success when s3 tags object successfully", func(t *testing.T) {
		s3ClientMock := new(S3ClientMock)
		handler := &Handler{
			S3Client: s3ClientMock,
		}

		event := ArchiveSourceEvent{
			SrcBucket:     "bucket",
			SrcVideo:      "video",
			GUID:          "guid",
			ArchiveSource: "GLACIER",
		}

		s3ClientMock.On("PutObjectTagging", mock.Anything).Return(&s3.PutObjectTaggingOutput{}, nil)

		res, err := handler.HandleRequest(event)
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		assert.Equal(t, "guid", res.GUID)

	})

	t.Run("should fail when s3 fails to tag object", func(t *testing.T) {
		s3ClientMock := new(S3ClientMock)
		handler := &Handler{
			S3Client: s3ClientMock,
		}

		event := ArchiveSourceEvent{
			SrcBucket:     "bucket",
			SrcVideo:      "video",
			GUID:          "guid",
			ArchiveSource: "GLACIER",
		}


		s3ClientMock.On("PutObjectTagging", mock.Anything).Return(nil, assert.AnError)

		_, err := handler.HandleRequest(event)
		assert.Error(t, err)
	})
}
