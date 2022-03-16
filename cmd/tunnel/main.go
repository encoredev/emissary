package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary"
	"go.encore.dev/emissary/internal/auth"
)

func main() {
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()

	// Read input
	host := flag.String("url", "", "URL to the emissary server")
	keyID := flag.Uint("kid", 1, "The emissary key ID")
	key := flag.String("key", "", "The emissary key base64 encoded")
	target := flag.String("target", "", "The target host:port you want to connect to via emissary")
	listenPort := flag.Uint("port", 0, "Port that the tunnel will listen on for your local system (0 will result in a random port)")
	flag.Parse()

	if host == nil || *host == "" {
		flag.PrintDefaults()
		log.Fatal().Msg("expected a emissary server url to be passed in using `-url`")
		os.Exit(1)
	}

	if keyID == nil || *keyID == 0 {
		flag.PrintDefaults()
		log.Fatal().Msg("expected a key id passed in with `-kid`")
		os.Exit(1)
	}

	if key == nil || *key == "" {
		flag.PrintDefaults()
		log.Fatal().Msg("expected a base64 encoded key passed in using `-key`")
		os.Exit(1)
	}

	if target == nil || *target == "" {
		flag.PrintDefaults()
		log.Fatal().Msg("expected a target to be passed in using `-target`")
		os.Exit(1)
	}

	data, err := base64.StdEncoding.DecodeString(*key)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to decode emissary key")
		os.Exit(1)
	}

	// Start listening for connections
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *listenPort))
	if err != nil {
		log.Fatal().Err(err).Msg("unable to start listening locally")
		os.Exit(1)
	}
	defer func() { _ = l.Close() }()

	log.Info().Msgf("Will tunnel traffic to %s", *target)
	log.Info().Msgf("Please connect to: %s", l.Addr().String())

	for {
		// Wait for connections
		local, err := l.Accept()
		if err != nil {
			log.Fatal().Err(err).Msg("unable to accept connection")
			os.Exit(1)
		}
		log.Info().Msgf("accepted connection from %s", local.RemoteAddr().String())

		go func() {
			// Setup the dialer
			log.Info().Msgf("dialing emissary server at %s", *host)
			dialer := emissary.NewWebsocketDialer(*host, auth.Key{KeyID: uint32(*keyID), Data: data})

			remote, err := dialer.Dial("tcp", *target)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to dial target through emissary")
				os.Exit(1)
			}

			// Proxy traffic
			errs := make(chan error, 2)
			go proxy(local, remote, errs)
			go proxy(remote, local, errs)
			for i := 0; i < 2; i++ {
				e := <-errs
				if e != nil {
					log.Fatal().Err(err).Msg("error while proxying traffic through the tunnel")
					os.Exit(1)
				}
			}
		}()
	}
}

func proxy(from, to net.Conn, errs chan error) {
	_, err := io.Copy(to, from)
	errs <- err
}
