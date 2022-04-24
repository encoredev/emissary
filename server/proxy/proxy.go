package proxy

import (
	"context"
	"crypto/rand"
	golog "log"
	"net"
	"time"

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

	var resolver socks5.NameResolver = socks5.DNSResolver{}
	if len(cfg.DNSServers) > 0 {
		resolver = customDNSResolver{ServerIPs: cfg.DNSServers, Fallback: socks5.DNSResolver{}}
	}

	// Set up our SOCKS5 server
	server, err := socks5.New(&socks5.Config{
		AuthMethods: newAuthenticator(cfg, nonce),
		Rules:       cfg.AllowedProxyTargets,
		Logger:      golog.New(log.Logger, "", golog.Lshortfile),
		Resolver:    resolver,
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

type customDNSResolver struct {
	// ServerIPs are the IPs to dial to do DNS lookups, in order.
	ServerIPs []string
	// Fallback specifies the fallback name resolver to use, if the given resolvers don't return anything.
	Fallback socks5.NameResolver
}

func (d customDNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	var lastErr error
	for _, serverIP := range d.ServerIPs {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, serverIP+":53")
			},
		}

		// Use a context with a short timeout so we don't block forever.
		lookupCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		addrs, err := r.LookupIPAddr(lookupCtx, name)
		cancel()
		if len(addrs) > 0 {
			return ctx, addrs[0].IP, nil
		}
		if err != nil {
			lastErr = err
			continue
		}
	}

	if d.Fallback != nil {
		return d.Fallback.Resolve(ctx, name)
	}

	// We failed; return the last error, if any.
	if lastErr != nil {
		return ctx, nil, lastErr
	}
	// Otherwise replicate the default behavior of net.Resolver when there are no addresses.
	err := &net.AddrError{
		Addr: name,
		Err:  "no suitable address found",
	}
	return ctx, nil, err
}
