package proxy

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.encore.dev/emissary/internal/auth"
)

type Config struct {
	HttpPort            int                 // What port should this server listen for HTTP/websocket connections on (0 == disabled)
	TcpPort             int                 // What port should this server listen for raw TCP connections on (0 == disabled)
	AuthKeys            auth.Keys           // What auth keys can be used when talking with this Emissary server
	AllowedProxyTargets AllowedProxyTargets // What proxy targets are allowed through this Emissary server
}

// LoadConfig performs setup for the proxy layer and returns an error if we cannot initialise
func LoadConfig(_ context.Context) (*Config, error) {
	// Load the .env file if present
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.Wrap(err, "unable to load env")
	}

	// Now configure viper with our default config and bind it to read from the environment
	viper.SetDefault("http_port", 8080)
	viper.SetEnvPrefix("emissary")
	viper.AutomaticEnv()

	// Read the config file
	viper.SetConfigFile("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/emissary")
	viper.AddConfigPath("$HOME/.emissary")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Ignore file not found errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errors.Wrap(err, "unable to read config.yaml")
		}
	}

	// Load the allowed proxy target list
	allowedProxyTargets := make(AllowedProxyTargets, 0)
	allowedHostsJSON := viper.GetString("allowed_proxy_targets")
	if allowedHostsJSON != "" {
		if err := json.Unmarshal([]byte(allowedHostsJSON), &allowedProxyTargets); err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal allowed proxy targets")
		}
	}
	if len(allowedProxyTargets) == 0 {
		return nil, errors.New("no allowed proxy targets loaded from environment")
	}

	// Load the auth keys
	authKeys := make(auth.Keys, 0)
	authKeysJSON := viper.GetString("auth_keys")
	if authKeysJSON != "" {
		if err := json.Unmarshal([]byte(authKeysJSON), &authKeys); err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal auth keys")
		}
	}
	if len(authKeys) == 0 {
		return nil, errors.New("no auth keys loaded from environment")
	}

	log.Info().
		Int("num_allowed_proxy_targets", len(allowedProxyTargets)).
		Int("num_auth_keys", len(authKeys)).
		Msg("loaded emissary proxy config")

	return &Config{
		HttpPort:            viper.GetInt("http_port"),
		TcpPort:             viper.GetInt("tcp_port"),
		AuthKeys:            authKeys,
		AllowedProxyTargets: allowedProxyTargets,
	}, nil
}
