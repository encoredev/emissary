package proxy

import (
	"context"
	"testing"

	"github.com/armon/go-socks5"
)

func TestCustomDNSServer(t *testing.T) {
	t.Parallel()
	dns := customDNSResolver{ServerIPs: []string{"1.1.1.1"}}
	_, ip, err := dns.Resolve(context.Background(), "google.com")
	if err != nil {
		t.Fatalf("unable to resolve with custom resolver: %v", err)
	} else if len(ip) == 0 {
		t.Fatalf("got zero ip: %s", ip)
	}
}

func TestCustomDNSServerMultiple(t *testing.T) {
	t.Parallel()

	// Start with an invalid DNS server to ensure we try the second, valid one.
	dns := customDNSResolver{ServerIPs: []string{"127.127.127.127", "1.1.1.1"}}
	_, ip, err := dns.Resolve(context.Background(), "google.com")
	if err != nil {
		t.Fatalf("unable to resolve with custom resolver: %v", err)
	} else if len(ip) == 0 {
		t.Fatalf("got zero ip: %s", ip)
	}
}

func TestCustomDNSServerFallback(t *testing.T) {
	t.Parallel()

	// Start with an invalid DNS server to ensure we try the fallback
	dns := customDNSResolver{ServerIPs: []string{"127.127.127.127"}, Fallback: socks5.DNSResolver{}}
	_, ip, err := dns.Resolve(context.Background(), "google.com")
	if err != nil {
		t.Fatalf("unable to resolve with custom resolver: %v", err)
	} else if len(ip) == 0 {
		t.Fatalf("got zero ip: %s", ip)
	}
}
