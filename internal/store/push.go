package store

import "strings"

// KeyPushPrivateKey holds the server's VAPID signing key (PKCS#8, base64).
// It is a secret: never sent to the browser.
const KeyPushPrivateKey = "push_vapid_private_key"

func init() { SecretKeys[KeyPushPrivateKey] = true }

// What a device can ask to be told about.
var PushKinds = []string{"timers", "useby", "low", "tonight", "updates"}

// PushSub is one phone or browser that turned on notifications.
type PushSub struct {
	ID        int64  `db:"id" json:"id"`
	UserID    int64  `db:"user_id" json:"-"`
	Endpoint  string `db:"endpoint" json:"-"`
	P256dh    string `db:"p256dh" json:"-"`
	Auth      string `db:"auth" json:"-"`
	Wants     string `db:"wants" json:"wants"`
	Device    string `db:"device" json:"device"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

const pushCols = `id, user_id, endpoint, p256dh, auth, wants, device, created_at`

// SavePushSub adds a device, or updates it if the endpoint is already known
// (it then belongs to whoever registered it last).
func (s *Store) SavePushSub(p PushSub) error {
	_, err := s.DB.Exec(`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, wants, device)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(endpoint) DO UPDATE SET user_id = excluded.user_id,
		p256dh = excluded.p256dh, auth = excluded.auth, wants = excluded.wants, device = excluded.device`,
		p.UserID, p.Endpoint, p.P256dh, p.Auth, p.Wants, p.Device)
	return err
}

func (s *Store) DeletePushSub(userID int64, endpoint string) error {
	_, err := s.DB.Exec(`DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?`, userID, endpoint)
	return err
}

// DropPushEndpoint forgets a device the push service says is gone.
func (s *Store) DropPushEndpoint(endpoint string) {
	_, _ = s.DB.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
}

// UserPushSubs lists one user's devices.
func (s *Store) UserPushSubs(userID int64) ([]PushSub, error) {
	out := []PushSub{}
	err := s.DB.Select(&out, `SELECT `+pushCols+` FROM push_subscriptions WHERE user_id = ? ORDER BY id`, userID)
	return out, err
}

// PushSubsWanting lists every device that asked for kind (one user's, if userID > 0).
func (s *Store) PushSubsWanting(kind string, userID int64) ([]PushSub, error) {
	var all []PushSub
	q, args := `SELECT `+pushCols+` FROM push_subscriptions`, []any{}
	if userID > 0 {
		q += ` WHERE user_id = ?`
		args = append(args, userID)
	}
	if err := s.DB.Select(&all, q+` ORDER BY id`, args...); err != nil {
		return nil, err
	}
	out := []PushSub{}
	for _, p := range all {
		for _, k := range strings.Split(p.Wants, ",") {
			if k == kind {
				out = append(out, p)
				break
			}
		}
	}
	return out, nil
}

// CleanWants keeps only known kinds, in order.
func CleanWants(list []string) string {
	var out []string
	for _, k := range PushKinds {
		for _, w := range list {
			if w == k {
				out = append(out, k)
				break
			}
		}
	}
	return strings.Join(out, ",")
}
