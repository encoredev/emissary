//go:build !lambda

package main

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	if err := Run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("emissary server exiting due to error")
		os.Exit(1)
	}
}
