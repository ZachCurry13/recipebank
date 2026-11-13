package api

import (
	"errors"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

// errHomeAddress is what the outside-page reader answers when an address
// turns out to be this computer or the home network. People type the address
// to read, and a page can point at more (its photo, a redirect), so without
// this check a recipe link could be used to look at a router or another app
// at home through RecipeBank.
var errHomeAddress = errors.New("that address is on your own network")

// publicOnly is called just before a connection is made, with the address
// already looked up, so a name that resolves to a home address and a redirect
// to one are refused the same way as a typed one.
func publicOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); ip == nil || !isPublicIP(ip) {
		return errHomeAddress
	}
	return nil
}

// nat64 is the well-known prefix that carries an IPv4 address inside an IPv6 one.
var nat64 = &net.IPNet{IP: net.ParseIP("64:ff9b::"), Mask: net.CIDRMask(96, 128)}

// isPublicIP reports whether ip is an ordinary internet address: not this
// computer, not a home or carrier network, not a reserved range.
func isPublicIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 == nil && nat64.Contains(ip) {
		ip = net.IP(ip[12:16])
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 0, // "this network"
			ip4[0] == 100 && ip4[1]&0xc0 == 64,          // carrier-grade NAT, 100.64.0.0/10
			ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0, // protocol assignments, 192.0.0.0/24
			ip4[0] == 198 && ip4[1]&0xfe == 18,          // benchmarking, 198.18.0.0/15
			ip4[0] >= 240:                               // reserved and broadcast
			return false
		}
	}
	return true
}

// newFetchClient is the client for outside pages and images. It only connects
// to public internet addresses. Where the server reaches the internet through
// a proxy (HTTP_PROXY and friends), the proxy makes the connections and this
// check cannot see them, so the plain client is used as before.
func newFetchClient() *http.Client {
	c := &http.Client{Timeout: 30 * time.Second, CheckRedirect: checkRedirect}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "ALL_PROXY", "all_proxy"} {
		if os.Getenv(name) != "" {
			return c
		}
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second, Control: publicOnly}).DialContext
	c.Transport = t
	return c
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return errors.New("only web addresses can be read")
	}
	return nil
}
