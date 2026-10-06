package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/store"
)

// captureTransport records where pushes went instead of sending them.
type captureTransport struct {
	mu   sync.Mutex
	urls []string
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.urls = append(c.urls, r.URL.String())
	c.mu.Unlock()
	return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}, Request: r}, nil
}

func (c *captureTransport) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.urls)
}

func TestPushNotifications(t *testing.T) {
	c, srv := setup(t)
	cap := &captureTransport{}
	srv.Push.Client = &http.Client{Transport: cap}

	var status struct {
		PublicKey string           `json:"public_key"`
		Devices   []map[string]any `json:"devices"`
	}
	if code := c.do("GET", "/api/push", nil, &status); code != 200 || len(status.PublicKey) != 87 {
		t.Fatalf("push status: %d %q", code, status.PublicKey)
	}
	ua, _ := ecdh.P256().GenerateKey(rand.Reader)
	keys := map[string]string{"p256dh": base64.RawURLEncoding.EncodeToString(ua.PublicKey().Bytes()), "auth": "BTBZMqHH6r4Tts7J_aSIgg"}
	sub := func(endpoint string, wants []string) int {
		return c.do("POST", "/api/push/subscribe", map[string]any{"endpoint": endpoint, "keys": keys, "wants": wants, "device": "Test phone"}, nil)
	}
	for _, bad := range []string{"http://fcm.googleapis.com/x", "https://192.168.1.1/admin", "https://fcm.googleapis.com.evil.example/x"} {
		if code := sub(bad, nil); code != 400 {
			t.Fatalf("%s must be refused: %d", bad, code)
		}
	}
	if code := sub("https://fcm.googleapis.com/fcm/send/abc", []string{"low", "nonsense", "useby"}); code != 200 {
		t.Fatalf("subscribe: %d", code)
	}
	c.do("GET", "/api/push", nil, &status)
	if len(status.Devices) != 1 || status.Devices[0]["wants"] != "useby,low" || status.Devices[0]["endpoint"] != nil {
		t.Fatalf("devices: %+v", status.Devices)
	}
	var settings map[string]any
	c.do("GET", "/api/admin/settings", nil, &settings)
	if secret := srv.Store.Setting(store.KeyPushPrivateKey); secret == "" || strings.Contains(stringify(settings), secret[:30]) {
		t.Fatal("the push key must be saved and never sent to the browser")
	}

	// Running low pushes to devices that want it.
	var it store.StockItem
	c.do("POST", "/api/stock", map[string]any{"area": "home", "name": "Toothpaste", "qty": 2, "low_at": 1}, &it)
	c.do("POST", "/api/stock/"+itoa(it.ID)+"/adjust", map[string]float64{"delta": -1}, nil)
	for i := 0; i < 50 && cap.count() == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if cap.count() != 1 || !strings.Contains(cap.urls[0], "fcm.googleapis.com") {
		t.Fatalf("running-low push: %v", cap.urls)
	}

	// The daily reminders' wording.
	now := time.Now()
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Milk", "qty": 1, "use_by": now.Format(day)}, nil)
	if m, ok := srv.useSoonMessage(now); !ok || m.Body != "Milk (today)" {
		t.Fatalf("use soon: %+v", m)
	}
	if m, ok := srv.tonightMessage(now); !ok || !strings.Contains(m.Body, "Nothing planned") {
		t.Fatalf("tonight: %+v", m)
	}
}

func stringify(v map[string]any) string {
	var b strings.Builder
	for k, x := range v {
		b.WriteString(k)
		if s, ok := x.(string); ok {
			b.WriteString(s)
		}
	}
	return b.String()
}
