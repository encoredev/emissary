package proxy

import (
	"context"
	"crypto/rand"
	"encoding/json"
	golog "log"
	"net"
	"os"

	"github.com/armon/go-socks5"
	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/auth"
	"go.encore.dev/emissary/internal/emissaryproto"
)

var (
	allowedProxyTargets AccessList
	authKeys            auth.Keys
)

// Init performs setup for the proxy layer and returns an error if we cannot initialise
func Init(_ context.Context) error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "unable to load env")
	}

	// Load the allowed proxy target list
	allowedProxyTargets = make(AccessList, 0)
	allowedHostsJSON := os.Getenv("EMISSARY_ALLOWED_PROXY_TARGETS")
	if allowedHostsJSON != "" {
		if err := json.Unmarshal([]byte(allowedHostsJSON), &allowedProxyTargets); err != nil {
			return errors.Wrap(err, "unable to unmarshal allowed proxy targets")
		}
	}
	if len(allowedProxyTargets) == 0 {
		return errors.New("no allowed proxy targets loaded from environment")
	}

	// Load the auth keys
	authKeys = auth.Keys{}
	authKeysJSON := os.Getenv("EMISSARY_AUTH_KEYS")
	if authKeysJSON != "" {
		if err := json.Unmarshal([]byte(authKeysJSON), &authKeys); err != nil {
			return errors.Wrap(err, "unable to unmarshal auth keys")
		}
	}
	if len(authKeys) == 0 {
		return errors.New("no auth keys loaded from environment")
	}

	log.Info().
		Int("num_allowed_proxy_targets", len(allowedProxyTargets)).
		Int("num_auth_keys", len(authKeys)).
		Msg("loaded emissary proxy config")

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
		Rules:       allowedProxyTargets,
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
