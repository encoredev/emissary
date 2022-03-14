package http

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func init() {
	router.Use(RequestLogger())
}

type requestLogger struct {
	handler http.Handler
}

func RequestLogger() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &requestLogger{handler: h}
	}
}

func (h requestLogger) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Debug().Str("remote", req.RemoteAddr).Str("uri", req.RequestURI).Msg("request received")
	h.handler.ServeHTTP(w, req)
}
