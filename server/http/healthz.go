package http

import (
	"net/http"
)

func init() {
	router.Methods("GET").PathPrefix("/healthz").Handler(http.HandlerFunc(handleHealth))
}
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{ "ok": true }`))
}
