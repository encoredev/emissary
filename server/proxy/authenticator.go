package proxy

import (
	"encoding/base64"

	"github.com/armon/go-socks5"
	"github.com/rs/zerolog/log"
	"go.encore.dev/emissary/internal/auth"
)

type authenticator struct {
	nonce []byte
}

var _ socks5.CredentialStore = (*authenticator)(nil)

func newAuthenticator(nonce []byte) []socks5.Authenticator {
	return []socks5.Authenticator{&socks5.UserPassAuthenticator{Credentials: &authenticator{nonce: nonce}}}
}

func (a authenticator) Valid(user, password string) bool {
	if err := auth.ValidateRequest(authKeys, user, base64.RawStdEncoding.EncodeToString(a.nonce), password); err != nil {
		log.Warn().Err(err).Msg("invalid hmac sent for emissary connection")
		return false
	}

	return true
}
