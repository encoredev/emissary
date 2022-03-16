package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankban/quicktest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary"
	"go.encore.dev/emissary/internal/auth"
	"go.encore.dev/emissary/server/proxy"
	"go.uber.org/atomic"
)

func TestMain(m *testing.M) {
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()

	os.Exit(m.Run())
}

// This test checks the happy path of an end to end test using emissary's proxy
func TestProxy_WebsocketTransport(t *testing.T) {
	c := quicktest.New(t)
	c.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targetServer := mustCreateTargetServer(c, ctx)
	defer func() { _ = targetServer.socket.Close() }()

	config := &proxy.Config{
		HttpPort: mustFreePort(c),
		TcpPort:  0,
		AuthKeys: auth.Keys{
			mustCreateAuthKey(c),
		},
		AllowedProxyTargets: proxy.AllowedProxyTargets{
			{Host: "localhost", Port: targetServer.port},
		},
	}

	// Start Server
	serverShutdown := make(chan error)
	go func() {
		serverShutdown <- RunWithConfig(ctx, config)
	}()

	// Create Client
	dailer := emissary.NewWebsocketDialer(fmt.Sprintf("ws://localhost:%d", config.HttpPort), config.AuthKeys[0])

	// Now dial the server via emissary
	conn, err := dailer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", targetServer.port))
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("error while dialing target server"))
	defer func() { _ = conn.Close() }()

	// Now test the transfer of data
	_, err = conn.Write([]byte("hello world"))
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("error while writing data"))

	response, err := io.ReadAll(conn)
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("error while reading response from server"))
	c.Assert(string(response), quicktest.Equals, "goodbye to hello world", quicktest.Commentf("wrong response received"))

	// Assert the state of the server
	c.Assert(targetServer.connections.Load(), quicktest.Equals, int64(1), quicktest.Commentf("connection wasn't made to target server"))
	c.Assert(targetServer.lastError.Load(), quicktest.IsNil, quicktest.Commentf("there was an error on the target server"))

	cancel()
	c.Assert(<-serverShutdown, quicktest.IsNil, quicktest.Commentf("run with config returned error"))
}

// This test checks that emissary's proxy will reject invalid hmacs
func TestProxy_WebsocketTransport_InvalidKey(t *testing.T) {
	c := quicktest.New(t)
	c.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targetServer := mustCreateTargetServer(c, ctx)
	defer func() { _ = targetServer.socket.Close() }()

	config := &proxy.Config{
		HttpPort: mustFreePort(c),
		TcpPort:  0,
		AuthKeys: auth.Keys{
			mustCreateAuthKey(c),
		},
		AllowedProxyTargets: proxy.AllowedProxyTargets{
			{Host: "localhost", Port: targetServer.port},
		},
	}

	// Start Server
	serverShutdown := make(chan error)
	go func() {
		serverShutdown <- RunWithConfig(ctx, config)
	}()

	// Create Client
	key := mustCreateAuthKey(c)
	key.KeyID = config.AuthKeys[0].KeyID // set the same key ID to force an hmac comparison
	dailer := emissary.NewWebsocketDialer(fmt.Sprintf("ws://localhost:%d", config.HttpPort), key)

	// Now dial the server via emissary
	_, err := dailer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", targetServer.port))
	c.Assert(err, quicktest.ErrorMatches, ".* username/password authentication failed", quicktest.Commentf("expected auth error from server"))

	// Assert the state of the server
	c.Assert(targetServer.connections.Load(), quicktest.Equals, int64(0), quicktest.Commentf("target server expected no connection attempts"))
	c.Assert(targetServer.lastError.Load(), quicktest.IsNil, quicktest.Commentf("there was an error on the target server"))

	cancel()
	c.Assert(<-serverShutdown, quicktest.IsNil, quicktest.Commentf("run with config returned error"))
}

func mustFreePort(c *quicktest.C) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		c.Fatalf("unable to get free port %+v", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		c.Fatalf("unable to get free port %+v", err)
	}
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port
}

func mustCreateAuthKey(c *quicktest.C) auth.Key {
	key := auth.Key{Data: make([]byte, 32)}

	id, err := rand.Int(rand.Reader, big.NewInt(1000000))
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("unable to generate key id"))

	key.KeyID = uint32(id.Int64())

	_, err = rand.Read(key.Data)
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("unable to generate random key"))

	return key
}

type targetServer struct {
	port        int
	socket      net.Listener
	connections *atomic.Int64
	lastError   *atomic.Error
}

func mustCreateTargetServer(c *quicktest.C, ctx context.Context) *targetServer {
	targetPort := mustFreePort(c)
	var lc net.ListenConfig
	socket, err := lc.Listen(ctx, "tcp", fmt.Sprintf("localhost:%d", targetPort))
	c.Assert(err, quicktest.IsNil, quicktest.Commentf("unable to create target server"))

	rtn := &targetServer{
		port:        targetPort,
		socket:      socket,
		connections: atomic.NewInt64(0),
		lastError:   atomic.NewError(nil),
	}

	go func() {
		log.Debug().Msg("target server listening for connection...")

		conn, err := socket.Accept()
		rtn.connections.Inc()
		if err != nil {
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				// happens on socket close during shutdown
				return
			}
			log.Err(err).Msg("target server unable to accept connection")
			rtn.lastError.Store(err)
			return
		}
		defer func() { _ = conn.Close() }()

		log.Debug().Msg("target server accepted connection")

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Err(err).Msg("target server unable to read")
			rtn.lastError.Store(err)
			return
		}
		log.Debug().Msgf("target server read %s", buf[:n])

		response := fmt.Sprintf("goodbye to %s", string(buf[:n]))
		_, err = conn.Write([]byte(response))
		if err != nil {
			log.Err(err).Msg("target server unable to write")
			rtn.lastError.Store(err)
			return
		}

		log.Debug().Msgf("target server wrote back %s", response)

		err = conn.Close()
		if err != nil {
			log.Err(err).Msg("target server couldn't close the connection")
			rtn.lastError.Store(err)
			return
		}

		log.Debug().Msg("target server closed the connection")
	}()

	return rtn
}
