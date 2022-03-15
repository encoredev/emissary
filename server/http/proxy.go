package http

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/ws"
	"go.encore.dev/emissary/server/proxy"
)

func init() {
	router.Methods("GET", "POST").PathPrefix("/proxy").Handler(http.HandlerFunc(handleProxy))
}

var (
	upgrader = websocket.Upgrader{
		Error: respondWithError,
	}
)

func handleProxy(w http.ResponseWriter, r *http.Request) {
	l := log.With().Str("remote", r.RemoteAddr).Str("uri", r.RequestURI).Str("proxy-method", "http").Logger()

	// Upgrade the request to a websocket connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.Err(err).Msg("error upgrading websocket")
		respondWithError(w, r, http.StatusInternalServerError, err)
		return
	}

	// Wrap the Gorilla websocket so we can use it as a net.Conn
	conn := ws.NewClient(c)
	defer func() {
		if err := conn.Close(); err != nil {
			l.Err(err).Msg("error closing websocket connection")
		}
	}()

	if err := proxy.ServeConn(conn); err != nil {
		l.Err(err).Msg("error serving websocket proxy request")
		return
	}

	l.Debug().Msg("websocket proxy connection closed")
}
