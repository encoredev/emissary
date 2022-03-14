package tcp

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/server/proxy"
)

// StartServer starts listening on the given port for TCP connections
func StartServer(ctx context.Context, port int) error {
	log.Info().Int("port", port).Msg("starting tcp server")

	var lc net.ListenConfig
	srv, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return errors.Wrap(err, "unable to listen on tcp port")
	}

	// Close the socket if the context is cancelled
	go func() {
		<-ctx.Done()
		log.Warn().Msg("shutting down tcp server")
		if err := srv.Close(); err != nil {
			log.Err(err).Msg("error shutting down tcp server")
		}
	}()

	// Enter the listen loop
	for {
		conn, err := srv.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return errors.Wrap(err, "unable to accept")
			default:
			}

			log.Err(err).Msg("unable to accept tcp connection")
			continue
		}

		go handleConn(conn)
	}
}

// handleConn handles a TCP connection and recovers from panics
func handleConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			stack := string(debug.Stack())

			var event *zerolog.Event
			if err2, ok := err.(error); ok {
				event = log.Err(err2)
			} else {
				event = log.Error().Interface("error", err)
			}

			event.Str("stack", stack).Msg("recovered from panic when handling tcp proxy")
		}
	}()

	l := log.With().Str("remote", conn.RemoteAddr().String()).Str("proxy-method", "tcp").Logger()
	l.Info().Msg("accepting tcp proxy request")

	if err := proxy.ServeConn(conn); err != nil {
		l.Err(err).Msg("unable to serve socks 5 proxy")
	}

	l.Info().Msg("websocket proxy connection closed")
}
