package emissary

import (
	"context"
	"net"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"go.encore.dev/emissary/internal/ws"
)

// The websocket dialer is one way of accessing an Emissary server
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
		HandshakeTimeout: 45 * time.Second,
	}
	wsc, _, err := dialer.DialContext(ctx, w.address, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to the emissary websocket")
	}
	wsc.EnableWriteCompression(true)

	// Wrap the websocket so it can be used like a net.Conn and return it
	conn := ws.NewClient(wsc)

	// FIXME: Start background routine for ping/pong admin messages

	return conn, nil
}
