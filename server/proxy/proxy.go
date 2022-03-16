package proxy

import (
	"crypto/rand"
	golog "log"
	"net"

	"github.com/armon/go-socks5"
	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/emissaryproto"
)

// ServeConn takes a connection and runs the emissary proxy on it
func (cfg *Config) ServeConn(conn net.Conn) error {
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
		AuthMethods: newAuthenticator(cfg, nonce),
		Rules:       cfg.AllowedProxyTargets,
		Logger:      golog.New(log.Logger, "", golog.Lshortfile),
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
