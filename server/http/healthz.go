package http

import (
	"encoding/json"
	"net/http"

	"go.encore.dev/emissary/server/proxy"
)

func handleHealth(cfg *proxy.Config) func(w http.ResponseWriter, _ *http.Request) {
	keys := make([]uint32, 0, len(cfg.AuthKeys))
	for _, key := range cfg.AuthKeys {
		keys = append(keys, key.KeyID)
	}
	healthResponse, _ := json.Marshal(map[string]interface{}{
		"ok":      true,
		"key_ids": keys,
	})
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(healthResponse)
	}
}
