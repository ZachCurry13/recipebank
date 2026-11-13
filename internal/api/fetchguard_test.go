package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	home := []string{
		"127.0.0.1", "127.8.9.10", "10.0.0.5", "172.16.0.1", "172.31.255.254", "192.168.1.1", // this computer and home networks
		"169.254.169.254", "100.64.0.1", "100.127.255.254", "0.0.0.0", "0.1.2.3", "192.0.0.8", "198.18.0.1", // link-local, carrier NAT, reserved
		"224.0.0.1", "240.0.0.1", "255.255.255.255",
		"::1", "::", "fe80::1", "fc00::1", "fd12:3456::1", "ff02::1",
		"::ffff:127.0.0.1", "::ffff:192.168.1.1", "64:ff9b::7f00:1", "64:ff9b::c0a8:101", // IPv4 home addresses written as IPv6
	}
	for _, s := range home {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s must not count as a public address", s)
		}
	}
	for _, s := range []string{"93.184.216.34", "1.1.1.1", "8.8.8.8", "100.63.255.255", "100.128.0.1", "172.32.0.1", "198.20.0.1",
		"2606:4700:4700::1111", "2001:4860:4860::8888", "64:ff9b::808:808"} {
		if !isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s is an ordinary internet address", s)
		}
	}
}

// The page reader never connects to this computer or the home network: not
// for a typed address, not for a name that resolves to one, and not when an
// outside page redirects there.
func TestFetchRefusesHomeAddresses(t *testing.T) {
	var hits atomic.Int32
	home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("the router's settings page"))
	}))
	defer home.Close()
	port := home.URL[strings.LastIndex(home.URL, ":"):]

	noProxy(t)
	s := &Server{Fetch: newFetchClient()}
	for _, addr := range []string{home.URL, "http://localhost" + port} {
		body, _, err := s.fetch(context.Background(), addr+"/admin", 1<<20)
		if err == nil {
			t.Fatalf("%s was read: %q", addr, body)
		}
		if !strings.Contains(err.Error(), "your own network") || !strings.Contains(err.Error(), "Paste text") {
			t.Errorf("%s: the answer should say why and what to do instead, got %q", addr, err)
		}
	}

	// A redirect is followed by the same client, so it meets the same check.
	_, err := newFetchClient().Get(home.URL)
	if !errors.Is(err, errHomeAddress) {
		t.Errorf("the client itself should refuse, got %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, home.URL, nil)
	if err := checkRedirect(req, make([]*http.Request, 5)); err == nil {
		t.Error("a sixth redirect should be refused")
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the home server was reached %d times", n)
	}
}

// Where the server reaches the internet through a proxy, the reader keeps
// working as before (the proxy makes the connections).
func TestFetchClientBehindAProxyIsThePlainOne(t *testing.T) {
	noProxy(t)
	if c := newFetchClient(); c.Transport == nil {
		t.Error("without a proxy, the client should carry the checked dialer")
	}
	t.Setenv("HTTPS_PROXY", "http://proxy.example:3128")
	if c := newFetchClient(); c.Transport != nil {
		t.Error("with a proxy set, the client should use the default transport")
	}
}

// noProxy makes the test read the same on a computer that has a proxy set.
func noProxy(t *testing.T) {
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(name, "")
	}
}
