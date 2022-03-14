package http

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

type errorResponse struct {
	Ok     bool  `json:"ok"`
	Reason error `json:"reason"`
}

func respondWithError(w http.ResponseWriter, req *http.Request, status int, reason error) {
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)

	// Write response
	bytes, err := json.Marshal(errorResponse{
		Ok:     false,
		Reason: reason,
	})
	if err != nil {
		log.Err(err).Msg("failed to encode error response as JSON")
		return
	}
	_, err = w.Write(bytes)
	if err != nil {
		log.Err(err).Msg("failed to write error response")
	}

	log.Warn().Str("remote", req.RemoteAddr).Str("uri", req.RequestURI).Bytes("json", bytes).Msg("responded to request with error")
}
