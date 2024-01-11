package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/server/proxy"
)

// StartServer starts listening on the given port for HTTP requests
func StartServer(ctx context.Context, config *proxy.Config) error {
	if config.HttpPort <= 0 {
		return nil
	}

	// Setup the router
	var router = mux.NewRouter()
	router.Use(PanicRecovery(), RequestLogger())
	router.Methods("GET").PathPrefix(config.HealthPath).Handler(http.HandlerFunc(handleHealth(config)))
	router.Methods("GET").PathPrefix("/").Handler(handleProxy(config))

	// Start the server
	log.Info().Int("port", config.HttpPort).Msg("starting http server")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.HttpPort),
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		log.Warn().Msg("shutting down http server")
		if err := srv.Close(); err != nil {
			log.Err(err).Msg("error shutting down http server")
		}
	}()

	err := srv.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return errors.Wrap(err, "error listening to http server")
	}

	return nil
}
