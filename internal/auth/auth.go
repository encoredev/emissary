package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
)

const keyIDLen = 4

// Key is a MAC key for authenticating communication between
// an Encore app and the Encore Platform. It is designed to be
// JSON marshalable, but as it contains secret material care
// must be taken when using it.
type Key struct {
	KeyID uint32 `json:"kid"`
	Data  []byte `json:"data"` // secret key data
}

type Keys []Key

// SignRequest signs our request using the Encore hmac standard.
func SignRequest(key Key, content string) (date string, sig string, err error) {
	date = time.Now().UTC().Format(http.TimeFormat)
	mac := hmac.New(sha256.New, key.Data)

	if _, err := fmt.Fprintf(mac, "%s\x00%s", date, content); err != nil {
		return "", "", errors.Wrap(err, "unable to hmac data")
	}

	bytes := make([]byte, keyIDLen, keyIDLen+sha256.Size)
	binary.BigEndian.PutUint32(bytes[0:keyIDLen], key.KeyID)
	bytes = mac.Sum(bytes)
	auth := base64.RawStdEncoding.EncodeToString(bytes)

	return date, auth, nil
}

func ValidateRequest(keys Keys, date, content, sig string) error {
	macBytes, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		return errors.New("invalid signature format")
	}

	if len(macBytes) < keyIDLen {
		return errors.New("signature too short")
	}
	keyID := binary.BigEndian.Uint32(macBytes[:keyIDLen])
	mac := macBytes[keyIDLen:]

	for _, k := range keys {
		if k.KeyID == keyID {
			if checkAuth(k, date, content, mac) {
				return nil
			}

			return errors.New("bad signature")
		}
	}

	return errors.New("no matching key ID found")
}

func checkAuth(key Key, dateStr, content string, gotMac []byte) bool {
	if dateStr == "" {
		return false
	}
	date, err := http.ParseTime(dateStr)
	if err != nil {
		return false
	}
	const threshold = 15 * time.Minute
	if diff := time.Since(date); diff > threshold || diff < -threshold {
		return false
	}

	mac := hmac.New(sha256.New, key.Data)
	_, _ = fmt.Fprintf(mac, "%s\x00%s", dateStr, content)
	expected := mac.Sum(nil)
	return hmac.Equal(expected, gotMac)
}
