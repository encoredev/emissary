package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var router = mux.NewRouter()

func StartServer(ctx context.Context, port int) error {
	configureLogger()
	log.Info().Int("port", port).Msg("starting server")

	err := http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		router,
	)
	if err != nil {
		err = errors.Wrap(err, "error listening to http server")
		log.Err(err).Msg("error from http server")
		return err
	}

	log.Warn().Msg("server shutdown")
	return nil
}

func configureLogger() {
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()
}
