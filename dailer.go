package emissary

import (
	"context"
	"encoding/base64"
	"net"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/auth"
	"go.encore.dev/emissary/internal/emissaryproto"
	"golang.org/x/net/proxy"
	"google.golang.org/protobuf/proto"
)

const BufSize = 1024

type transportDialer interface {
	proxy.Dialer
	proxy.ContextDialer
}

// Dialer is the primary dialer that is exposed from this library.
type Dialer struct {
	transportLayer transportDialer
	key            auth.Key
}

var _ transportDialer = (*Dialer)(nil)

// NewWebsocketDialer creates a dialer which will connect to emissary over a websocket.
func NewWebsocketDialer(server string, key auth.Key) *Dialer {
	return &Dialer{
		transportLayer: &websocketDialer{address: server},
		key:            key,
	}
}

func (e *Dialer) Dial(network, addr string) (c net.Conn, err error) {
	return e.DialContext(context.Background(), network, addr)
}

func (e *Dialer) DialContext(ctx context.Context, network, addr string) (c net.Conn, err error) {
	// Dial the transport layer
	transportLayer, err := e.transportLayer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect on emissary transport")
	}

	// Read the connect message and then verify it's the support protocol version
	connectMessage, err := readConnectMessage(transportLayer)
	if err != nil {
		_ = transportLayer.Close()
		return nil, errors.Wrap(err, "unable to read connect message")
	}
	if connectMessage.ProtocolVersion != emissaryproto.ProtocolVersion {
		_ = transportLayer.Close()
		return nil, errors.Newf("unsupported emissary protocol version: supports %d, got %d", emissaryproto.ProtocolVersion, connectMessage.ProtocolVersion) //nolint:wrapcheck
	}

	// Create the login information
	date, hmac, err := auth.SignRequest(e.key, base64.RawStdEncoding.EncodeToString(connectMessage.ConnectionNonce))
	if err != nil {
		_ = transportLayer.Close()
		return nil, errors.Wrap(err, "unable to create emissary login")
	}

	// Now upgrade the connection to a SOCKS5 client
	socks5, err := proxy.SOCKS5(network, addr, &proxy.Auth{
		User:     date,
		Password: hmac,
	}, &withOpenTransport{transportLayer})
	if err != nil {
		_ = transportLayer.Close()
		return nil, errors.Wrap(err, "unable to create socks5 proxy dialer")
	}

	// And tell the SOCKS5 proxy to now dail and authenticate
	if withCtx, ok := socks5.(proxy.ContextDialer); ok {
		c, err := withCtx.DialContext(ctx, network, addr)
		if err != nil {
			_ = transportLayer.Close()
			return nil, errors.Wrap(err, "unable to dial socks 5 proxy")
		}
		return c, nil
	} else {
		c, err := socks5.Dial(network, addr)
		if err != nil {
			_ = transportLayer.Close()
			return nil, errors.Wrap(err, "unable to dial socks 5 proxy")
		}
		return c, nil
	}
}

// readConnectMessage gets and unmarshals the connection header from the server.
func readConnectMessage(transportLayer net.Conn) (*emissaryproto.ServerConnect, error) {
	// Read the message
	buf := make([]byte, BufSize)
	n, err := transportLayer.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read connect message")
	}
	if n == 0 {
		return nil, errors.New("no connect message sent")
	}
	connectMessage := &emissaryproto.ServerConnect{}
	if err := proto.Unmarshal(buf[:n], connectMessage); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal connect message")
	}

	// Verify the constants are set as expected
	if connectMessage.ServerSoftware != emissaryproto.EmissaryServer {
		return nil, errors.Newf("invalid emissary server, got; %s", connectMessage.ServerSoftware) //nolint:wrapcheck
	}
	if connectMessage.ProtocolVersion <= 0 {
		return nil, errors.Newf("invalid emissary protocol version, got; %d", connectMessage.ProtocolVersion) //nolint:wrapcheck
	}
	if len(connectMessage.ConnectionNonce) != emissaryproto.NonceSize {
		return nil, errors.Newf("invalid length nonce, length; %d", len(connectMessage.ConnectionNonce)) //nolint:wrapcheck
	}
	if allZero(connectMessage.ConnectionNonce) {
		return nil, errors.New("connection nonce was all zeros")
	}

	log.Debug().Str("server", connectMessage.ServerSoftware).
		Str("server_version", connectMessage.ServerVersion).
		Int32("protocol_version", connectMessage.ProtocolVersion).
		Msg("connected to emissary transport layer")

	return connectMessage, nil
}

type withOpenTransport struct {
	conn net.Conn
}

func (w *withOpenTransport) Dial(_, _ string) (c net.Conn, err error) {
	return w.conn, nil
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
