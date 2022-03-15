package proxy

import (
	"context"
	"crypto/rand"
	golog "log"
	"net"

	"github.com/armon/go-socks5"
	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/auth"
	"go.encore.dev/emissary/internal/emissaryproto"
)

var (
	socks5Rules socks5.RuleSet
	authKeys    auth.Keys
)

// Init performs setup for the proxy layer and returns an error if we cannot initialise
func Init(_ context.Context) error {
	socks5Rules = &socks5.PermitCommand{
		EnableConnect:   true,
		EnableBind:      false,
		EnableAssociate: false,
	}

	authKeys = auth.Keys{
		auth.Key{
			KeyID: 122,
			Data:  []byte("old-key"),
		},
		auth.Key{
			KeyID: 123,
			Data:  []byte("super-secret-key"),
		},
		auth.Key{
			KeyID: 124,
			Data:  []byte("future-key"),
		},
	}

	return nil
}

// ServeConn takes a connection and runs the emissary proxy on it
func ServeConn(conn net.Conn) error {
	nonce := make([]byte, emissaryproto.NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return errors.Wrap(err, "unable to create nonce")
	}

	// Send the emissary server version number
	connectMsg := &emissaryproto.ServerConnect{
		ServerSoftware:  emissaryproto.EmissaryServer,
		ServerVersion:   emissaryproto.EmissaryServerVersion,
		ProtocolVersion: emissaryproto.ProtocolVersion,
		ConnectionNonce: nonce,
	}
	bytes, err := proto.Marshal(connectMsg)
	if err != nil {
		return errors.Wrap(err, "unable to marshal connect message")
	}
	_, err = conn.Write(bytes)
	if err != nil {
		return errors.Wrap(err, "unable to send connect message")
	}

	// Set up our SOCKS5 server
	server, err := socks5.New(&socks5.Config{
		AuthMethods: newAuthenticator(nonce),
		Rules:       socks5Rules,
		Logger:      golog.New(log.Logger, "ws", golog.LstdFlags),
	})
	if err != nil {
		return errors.Wrap(err, "unable to setup socks5 proxy server")
	}

	// Pass the connection over to the SOCKS5 server
	if err := server.ServeConn(conn); err != nil {
		return errors.Wrap(err, "error while running socks 5 proxy")
	}

	return nil
}
