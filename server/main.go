//go:build !lambda

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/server/http"
	"go.encore.dev/emissary/server/proxy"
	"go.encore.dev/emissary/server/tcp"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Start the main context for Emissary
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialise our logging library
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()

	// Listen for OS level signals to shutdown and then cancel our main context
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-done
		log.Warn().Str("signal", s.String()).Msg("received signal to shutdown")
		cancel()
	}()

	// Initialise the proxy layer
	if err := proxy.Init(ctx); err != nil {
		log.Fatal().Err(err).Msg("unable to initialise proxy layer")
		os.Exit(1)
	}

	// Start our various servers (http / tcp)
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		if err := http.StartServer(ctx, 8000); err != nil {
			return errors.Wrap(err, "error running http server")
		}

		return nil
	})
	grp.Go(func() error {
		if err := tcp.StartServer(ctx, 8001); err != nil {
			return errors.Wrap(err, "error running tcp server")
		}

		return nil
	})

	// Wait for one of the servers to return an error
	if err := grp.Wait(); err != nil {
		log.Err(err).Msg("there was a fatal error running emissary")
	}
	log.Info().Msg("Emissary shutdown")
}
