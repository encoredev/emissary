//go:build lambda

package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"go.encore.dev/emissary/server/http"
)

func EmissaryHandler(ctx context.Context) {
	http.Run(ctx)
}

func main() {
	lambda.Start(EmissaryHandler)
}
