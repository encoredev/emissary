package http

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	router.Use(RecoveryHandler())
}

type recoveryHandler struct {
	handler http.Handler
}

func RecoveryHandler() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &recoveryHandler{handler: h}
	}
}

func (h recoveryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			stack := string(debug.Stack())

			var event *zerolog.Event
			if err2, ok := err.(error); ok {
				event = log.Err(err2)
			} else {
				event = log.Error().Interface("error", err)
			}

			event.Str("stack", stack).Msg("recovered from panic when handling request")

		}
	}()

	h.handler.ServeHTTP(w, req)
}
