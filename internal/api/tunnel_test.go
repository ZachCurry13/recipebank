package api

import "testing"

func TestTunnelSaveAddress(t *testing.T) {
	c, srv := setup(t)
	for _, h := range []string{"https://recipes.example.com/", "recipes.example.com", "http://Recipes.Example.org"} {
		if code := c.do("PUT", "/api/admin/tunnel", map[string]any{"enabled": false, "hostname": h}, nil); code != 200 {
			t.Errorf("%q: %d", h, code)
		}
	}
	if got := srv.Store.Setting("tunnel_hostname"); got != "Recipes.Example.org" {
		t.Errorf("saved address %q", got)
	}
	if code := c.do("PUT", "/api/admin/tunnel", map[string]any{"enabled": false, "hostname": "a b/c"}, nil); code != 400 {
		t.Errorf("bad address accepted: %d", code)
	}
	var status struct {
		Hostname string `json:"hostname"`
		HasToken bool   `json:"has_token"`
	}
	c.do("PUT", "/api/admin/tunnel", map[string]any{"enabled": false, "token": "cloudflared service install eyJabc"}, nil)
	c.do("GET", "/api/admin/tunnel", nil, &status)
	if !status.HasToken || srv.Store.Setting("tunnel_token") != "eyJabc" {
		t.Errorf("token from the pasted command: %q", srv.Store.Setting("tunnel_token"))
	}
}
