package emissary

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/ws"
)

const HandshakeTimeout = 20 * time.Second
const PingTime = 30 * time.Second

// The websocket dialer is one way of accessing an Emissary server.
type websocketDialer struct {
	address string
}

var _ transportDialer = (*websocketDialer)(nil)

func (w *websocketDialer) Dial(network, addr string) (c net.Conn, err error) {
	return w.DialContext(context.Background(), network, addr)
}

func (w *websocketDialer) DialContext(ctx context.Context, network, _ string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("tcp only supported")
	}

	// Dial the basic websocket
	dialer := &websocket.Dialer{
		HandshakeTimeout: HandshakeTimeout,
	}
	wsc, _, err := dialer.DialContext(ctx, w.address, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to the emissary websocket")
	}
	wsc.EnableWriteCompression(true)

	// Wrap the websocket so it can be used like a net.Conn and return it
	conn := ws.NewClient(wsc)

	// Startup a background keep-alive so the socket doesn't close when there's no traffic
	go func() {
		t := time.NewTicker(PingTime)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-t.C:
				if err := conn.SendPing(); err != nil {
					if !errors.Is(err, io.EOF) {
						log.Err(err).Msg("unable to send websocket keep-alive")
					}
					return
				}
			}
		}
	}()
	return conn, nil
}
