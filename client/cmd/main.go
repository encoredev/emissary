package main

import (
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary"
)

func main() {
	configureLogger()
	emissaryClient := emissary.NewWebsocketDialer("ws://localhost:8000/proxy")

	// setup a http client
	httpTransport := &http.Transport{
		DisableKeepAlives: true,
	}
	httpClient := &http.Client{Transport: httpTransport}

	// Use the emissary client to dial
	httpTransport.DialContext = emissaryClient.DialContext

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
		log.Fatal().Err(err).Msg("unable to read google response")
		return
	}

	log.Info().Msg("Body: " + string(b))
}

func configureLogger() {
	log.Logger = zerolog.New(
		zerolog.NewConsoleWriter(),
	).With().Caller().Timestamp().Logger()
}
