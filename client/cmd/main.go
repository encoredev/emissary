package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"go.encore.dev/emissary/internal/ws"

	// "github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// "go.encore.dev/emissary/internal/ws"
	"golang.org/x/net/proxy"
)

type WSProxy struct {
}

func (W WSProxy) Dial(protocol, address string) (c net.Conn, err error) {
	if protocol != "tcp" {
		return nil, errors.New("tcp only supported")
	}

	dailer := &websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
		ReadBufferSize:   409600,
		WriteBufferSize:  409600,
	}
	wsc, _, err := dailer.Dial(address, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to dail")
		return
	}

	conn := ws.NewClient(wsc)
	return conn, nil

	// return gows.Dial("ws://localhost:8000/proxy", "", "http://localhost")
}

var _ proxy.Dialer = (*WSProxy)(nil)

func main() {
	configureLogger()

	s, err := proxy.SOCKS5("tcp", "ws://localhost:8000/proxy", &proxy.Auth{
		User:     "encore",
		Password: "erocne",
	}, &WSProxy{})

	// s, err := proxy.SOCKS5("tcp", "localhost:8001", &proxy.Auth{
	// 	User:     "encore",
	// 	Password: "erocne",
	// }, &net.Dialer{
	// 	Timeout:   30 * time.Second,
	// 	KeepAlive: 30 * time.Second,
	// })
	if err != nil {
		log.Fatal().Err(err).Msg("unable to dail")
		return
	}

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}
	// set our socks5 as the dialer
	httpTransport.Dial = s.Dial
	// create a request
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to get google")
		return
	}
	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to get google")
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to read google respones")
		return
	}

	log.Info().Msg("Body: " + string(b))

	// c.WriteMessage(websocket.TextMessage, []byte("hello there"))

	// _, err = conn.Write([]byte("client"))
	// if err != nil {
	// 	log.Err(err).Msg("error writing to socket")
	// 	return
	// }
	//
	// b := make([]byte, 4096)
	// // buf := bytes.NewBuffer(b)
	// // _, err = buf.ReadFrom(conn)
	// _, err = conn.Read(b)
	// if err != nil {
	// 	log.Err(err).Msg("error reading from socket")
	// 	return
	// }
	//
	// ticker := time.NewTicker(10 * time.Second)
	//
	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		if err := conn.SendPing(); err != nil {
	// 			log.Err(err).Msg("unable to send ping")
	// 			break
	// 		}
	//
	// 		log.Info().Msg("ping")
	// 	}
	// }
	//
	// log.Info().Msgf("Got back: %s", string(b))
	//
	// if err := conn.Close(); err != nil {
	// 	log.Fatal().Err(err).Msg("unable to close connection")
	// }

}

func configureLogger() {
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()
}
