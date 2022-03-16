package tcp

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/server/proxy"
)

// StartServer starts listening on the given port for TCP connections
func StartServer(ctx context.Context, cfg *proxy.Config) error {
	if cfg.TcpPort <= 0 {
		return nil
	}
	log.Info().Int("port", cfg.TcpPort).Msg("starting tcp server")

	var lc net.ListenConfig
	srv, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", cfg.TcpPort))
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
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}

				return errors.Wrap(err, "unable to accept")
			default:
			}

			log.Err(err).Msg("unable to accept tcp connection")
			continue
		}

		go handleConn(conn, cfg)
	}
}

// handleConn handles a TCP connection and recovers from panics
func handleConn(conn net.Conn, cfg *proxy.Config) {
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

	if err := cfg.ServeConn(conn); err != nil {
		l.Err(err).Msg("unable to serve socks 5 proxy")
	}

	l.Info().Msg("websocket proxy connection closed")
}
