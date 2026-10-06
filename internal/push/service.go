package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/zachcurry13/recipebank/internal/safe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Subject is the contact push services see in our VAPID claims.
const Subject = "https://github.com/ZachCurry13/recipebank"

// Message is what the phone shows. Messages with the same Tag replace each
// other instead of piling up.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"` // in-app page, e.g. "/#/shopping"
	Tag   string `json:"tag,omitempty"`
}

// Service sends RecipeBank's phone notifications (from NovelCheck).
type Service struct {
	Store  *store.Store
	Client *http.Client // nil = a client with a 15 s timeout

	mu  sync.Mutex
	key *ecdsa.PrivateKey
}

func New(st *store.Store) *Service { return &Service{Store: st} }

// signingKey loads the VAPID key, creating and saving one on first use.
func (p *Service) signingKey() (*ecdsa.PrivateKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.key != nil {
		return p.key, nil
	}
	if saved := p.Store.Setting(store.KeyPushPrivateKey); saved != "" {
		der, err := base64.StdEncoding.DecodeString(saved)
		if err == nil {
			if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
				if ec, ok := k.(*ecdsa.PrivateKey); ok && ec.Curve == elliptic.P256() {
					p.key = ec
					return ec, nil
				}
			}
		}
		return nil, errors.New("the saved push key is damaged")
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return nil, err
	}
	if err := p.Store.SetSetting(store.KeyPushPrivateKey, base64.StdEncoding.EncodeToString(der)); err != nil {
		return nil, err
	}
	p.key = k
	return k, nil
}

// PublicKey is the VAPID key browsers subscribe with (base64url, 65 bytes).
func (p *Service) PublicKey() (string, error) {
	k, err := p.signingKey()
	if err != nil {
		return "", err
	}
	pub, err := k.PublicKey.ECDH()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(pub.Bytes()), nil
}

// Notify sends m in the background to every device that wants kind (one
// user's devices when userID > 0).
func (p *Service) Notify(kind string, userID int64, m Message) {
	if p == nil {
		return
	}
	safe.Go("push "+kind, func() {
		subs, err := p.Store.PushSubsWanting(kind, userID)
		if err != nil || len(subs) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, err := p.SendTo(ctx, subs, m); err != nil {
			log.Printf("push: %v", err)
		}
	})
}

// SendTo delivers m to each device now, forgetting any the push service says
// are gone. It reports how many got it and the first error.
func (p *Service) SendTo(ctx context.Context, subs []store.PushSub, m Message) (int, error) {
	k, err := p.signingKey()
	if err != nil {
		return 0, err
	}
	if r := []rune(m.Body); len(r) > 600 {
		m.Body = string(r[:599]) + "…"
	}
	payload, _ := json.Marshal(m)
	s := &Sender{Key: k, Subject: Subject, Client: p.Client}
	sent := 0
	var first error
	for _, sub := range subs {
		err := s.Send(ctx, Subscription{Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth}, payload, 24*time.Hour)
		switch {
		case err == nil:
			sent++
		case errors.Is(err, ErrGone):
			p.Store.DropPushEndpoint(sub.Endpoint)
		case first == nil:
			first = err
		}
	}
	return sent, first
}

// Timers are cook-mode timers that buzz the phone when they end, even with
// the screen off. They live in memory: a restart forgets them (the page's
// own timer still rings).
type Timers struct {
	mu   sync.Mutex
	next int64
	on   map[int64]*time.Timer
}

// Start sends m to the user's timer devices after d; it returns an id for Stop.
func (t *Timers) Start(p *Service, userID int64, d time.Duration, m Message) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.on == nil {
		t.on = map[int64]*time.Timer{}
	}
	t.next++
	id := t.next
	t.on[id] = time.AfterFunc(d, func() {
		t.mu.Lock()
		delete(t.on, id)
		t.mu.Unlock()
		p.Notify("timers", userID, m)
	})
	return id
}

// Stop cancels a timer; false if it already rang or never existed.
func (t *Timers) Stop(id int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	tm, ok := t.on[id]
	if ok {
		tm.Stop()
		delete(t.on, id)
	}
	return ok
}
