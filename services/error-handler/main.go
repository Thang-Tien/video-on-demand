package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event interface{}) (string, error) {
	return "hello from error-handle", nil
}

func main() {
	lambda.Start(handler)
}
