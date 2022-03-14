package emissary

import (
	"context"
	"net"

	"github.com/cockroachdb/errors"
	"golang.org/x/net/proxy"
)

type transportDialer interface {
	proxy.Dialer
	proxy.ContextDialer
}

// Dialer is the primary dialer that is exposed from this library
type Dialer struct {
	transportLayer transportDialer
}

var _ proxy.Dialer = (*Dialer)(nil)
var _ proxy.ContextDialer = (*Dialer)(nil)

// NewWebsocketDialer creates a dialer which will connect to emissary over a websocket
func NewWebsocketDialer(server string) *Dialer {
	return &Dialer{
		transportLayer: &websocketDialer{address: server},
	}
}

func (e *Dialer) Dial(network, addr string) (c net.Conn, err error) {
	return e.DialContext(context.Background(), network, addr)
}

func (e *Dialer) DialContext(ctx context.Context, network, addr string) (c net.Conn, err error) {
	// Setup the socks5 dialer with the underlying transport dialer
	socks5, err := proxy.SOCKS5(network, addr, &proxy.Auth{
		User:     "encore",
		Password: "erocne",
	}, e.transportLayer)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create socks5 proxy dialer")
	}

	// Now do the dail
	if withCtx, ok := socks5.(proxy.ContextDialer); ok {
		c, err := withCtx.DialContext(ctx, network, addr)
		if err != nil {
			return nil, errors.Wrap(err, "unable to dial socks 5 proxy")
		}
		return c, nil
	} else {
		c, err := socks5.Dial(network, addr)
		if err != nil {
			return nil, errors.Wrap(err, "unable to dial socks 5 proxy")
		}
		return c, nil
	}
}
