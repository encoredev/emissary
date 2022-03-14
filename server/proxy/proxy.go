package proxy

import (
	"context"
	golog "log"
	"net"

	"github.com/armon/go-socks5"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

var socks5Server *socks5.Server

// Init performs setup for the proxy layer and returns an error if we cannot initialise
func Init(_ context.Context) error {
	// Set up our SOCKS5 server
	server, err := socks5.New(&socks5.Config{
		Credentials: socks5.StaticCredentials{"encore": "erocne"},
		Resolver:    nil,
		Rules: &socks5.PermitCommand{
			EnableConnect:   true,
			EnableBind:      false,
			EnableAssociate: false,
		},
		Logger: golog.New(log.Logger, "ws", golog.LstdFlags),
	})
	if err != nil {
		return errors.Wrap(err, "unable to setup socks5 proxy server")
	}
	socks5Server = server

	return nil
}

// ServeConn takes a connection and runs the emissary proxy on it
func ServeConn(conn net.Conn) error {
	// FIXME: add authentication here

	// Pass the connection over to the SOCKS5 server
	if err := socks5Server.ServeConn(conn); err != nil {
		return errors.Wrap(err, "error while running socks 5 proxy")
	}

	return nil
}
