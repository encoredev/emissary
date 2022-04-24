package proxy

import (
	"context"

	"github.com/armon/go-socks5"
	"github.com/rs/zerolog/log"
)

type AllowedHost struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type AllowedProxyTargets []AllowedHost

var _ socks5.RuleSet = (AllowedProxyTargets)(nil)

func (a AllowedProxyTargets) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	if req.Command != socks5.ConnectCommand {
		log.Warn().Uint8("command", req.Command).Msg("only connect commands are allowed")
		return ctx, false
	}

	l := log.With().Str("to_host", req.DestAddr.FQDN).Str("to_ip", req.DestAddr.IP.String()).Int("to_port", req.DestAddr.Port).Str("remote", req.RemoteAddr.String()).Logger()

	for _, allowedHost := range a {
		if allowedHost.Allow(req) {
			l.Info().Msg("allowing proxy connection")
			return ctx, true
		}
	}

	l.Warn().Msg("disallowing proxy connection")
	return ctx, false
}

func (a AllowedHost) Allow(req *socks5.Request) bool {
	return a.Port == req.DestAddr.Port && (a.Host == req.DestAddr.FQDN ||
		(a.Host == req.DestAddr.IP.String() && len(req.DestAddr.IP) > 0))
}
