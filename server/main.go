//go:build !lambda

package main

import (
	"context"

	"go.encore.dev/emissary/server/http"
)

func main() {
	http.Run(context.Background())
}
