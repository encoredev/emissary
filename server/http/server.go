package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

var router = mux.NewRouter()

// StartServer starts listening on the given port for HTTP requests
func StartServer(ctx context.Context, port int) error {
	log.Info().Int("port", port).Msg("starting http server")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
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
		return errors.Wrap(err, "error listening to http server")
	}

	return nil
}
