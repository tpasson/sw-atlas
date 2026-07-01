package store

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// privateFetchAllowed lets an operator opt out of the SSRF guard for deployments
// that legitimately mirror an internal Git server (e.g. a Gitea on a private
// LAN / GitHub Enterprise): ATLAS_ALLOW_PRIVATE_FETCH=1. Off by default.
func privateFetchAllowed() bool {
	v := os.Getenv("ATLAS_ALLOW_PRIVATE_FETCH")
	return v == "1" || strings.EqualFold(v, "true")
}

// safeHTTPClient returns an HTTP client whose dialer refuses to connect to
// non-public addresses — loopback, private (RFC1918 / ULA), link-local (incl.
// the cloud metadata endpoint 169.254.169.254), unspecified and multicast. This
// is the SSRF guard for user-supplied source / subscription URLs.
//
// The check runs on every dial (so HTTP redirects are covered too) and dials the
// exact resolved IP to avoid a DNS-rebinding TOCTOU between check and connect.
func safeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				block := !privateFetchAllowed()
				var target net.IP
				for _, ip := range ips {
					if block && isBlockedIP(ip.IP) {
						return nil, fmt.Errorf("refusing to connect to non-public address %s (SSRF guard; set ATLAS_ALLOW_PRIVATE_FETCH=1 to allow internal hosts)", ip.IP)
					}
					if target == nil {
						target = ip.IP
					}
				}
				if target == nil {
					return nil, fmt.Errorf("no address for host %q", host)
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(target.String(), port))
			},
		},
	}
}

// isBlockedIP reports whether an address must never be dialed from a user-driven
// fetch (defends against SSRF to internal services & cloud metadata).
func isBlockedIP(ip net.IP) bool {
	return ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
