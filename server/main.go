//go:build !lambda

package main

import (
	"context"
	golog "log"
	"net"
	"os"

	"github.com/armon/go-socks5"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/server/http"
)

func main() {
	go func() {
		s, err := net.Listen("tcp", ":8001")
		if err != nil {
			log.Fatal().Err(err).Msg("unable to listen")
			os.Exit(1)
		}

		for {
			conn, err := s.Accept()
			if err != nil {
				log.Fatal().Err(err).Msg("unable to accept")
				continue
			}

			server, err := socks5.New(&socks5.Config{
				AuthMethods: nil,
				Credentials: socks5.StaticCredentials{"encore": "erocne"},
				Resolver:    nil,
				Rules:       nil,
				Rewriter:    nil,
				BindIP:      nil,
				Logger:      golog.New(log.Logger, "ws", golog.LstdFlags),
				Dial:        nil,
			})
			if err != nil {
				log.Err(err).Msg("unable to start socks5 proxy")
			}

			if err := server.ServeConn(conn); err != nil {
				log.Err(err).Msg("unable to serve socks 5 proxy")
			}
		}
	}()

	if err := http.StartServer(context.Background(), 8000); err != nil {
		panic(err)
	}
}
