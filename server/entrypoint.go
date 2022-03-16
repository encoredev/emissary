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

// Run starts an emissary server up. Due to how we need to package Emissary for different clouds, we have different
// `main` functions, but they all just call this function after performing any cloud specific setup. We control
// which `main` function is compiled into the binary using build tags.
func Run(ctx context.Context) error {
	// Start the main context for Emissary
	ctx, cancel := context.WithCancel(ctx)
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

	// Load the config
	config, err := proxy.LoadConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to initialise proxy layer")
		return err
	}

	return RunWithConfig(ctx, config)
}

// RunWithConfig allows end to end tests to pass in specific test config and run in parallel
func RunWithConfig(ctx context.Context, config *proxy.Config) error {
	// Start our various servers (http / tcp)
	grp, ctx := errgroup.WithContext(ctx)
	if config.HttpPort > 0 {
		grp.Go(func() error {
			if err := http.StartServer(ctx, config); err != nil {
				return errors.Wrap(err, "error running http server")
			}

			return nil
		})
	}

	if config.TcpPort > 0 {
		grp.Go(func() error {
			if err := tcp.StartServer(ctx, config); err != nil {
				return errors.Wrap(err, "error running tcp server")
			}

			return nil
		})
	}

	// Wait for one of the servers to return an error
	if err := grp.Wait(); err != nil {
		log.Err(err).Msg("there was a fatal error running emissary")
		return err
	}

	log.Info().Msg("Emissary shutdown")
	return nil
}
