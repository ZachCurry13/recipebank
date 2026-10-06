// Package mail sends a recipe or the shopping list by email through the
// family's own mail server (SMTP), set up under Admin → Email.
package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Config is the mail server RecipeBank sends through.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (c Config) Validate() error {
	if c.Host == "" || c.Port == "" || c.From == "" {
		return errors.New("email isn't set up yet: an admin can add a mail server under Admin → Email")
	}
	if strings.ContainsAny(c.From+c.Host+c.Port, "\r\n") {
		return errors.New("the email settings aren't valid")
	}
	return nil
}

// Address checks one recipient address ("name@example.com").
func Address(to string) (string, error) {
	a, err := netmail.ParseAddress(strings.TrimSpace(to))
	if err != nil || strings.ContainsAny(a.Address, "\r\n,;") {
		return "", errors.New("type one email address, like name@example.com")
	}
	return a.Address, nil
}

// Message is an email with a plain-text and an HTML version.
type Message struct {
	To, Subject, Text, HTML string
}

// Send delivers m. Port 465 uses TLS from the start; other ports switch to
// TLS when the server offers it.
func Send(c Config, m Message) error {
	c = c.Clean()
	if err := c.Validate(); err != nil {
		return err
	}
	to, err := Address(m.To)
	if err != nil {
		return err
	}
	return deliver(c, to, build(c.From, to, m))
}

func build(from, to string, m Message) []byte {
	boundary := fmt.Sprintf("recipebank-%d", time.Now().UnixNano())
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\n", from, to, mime.QEncoding.Encode("utf-8", oneLine(m.Subject)))
	fmt.Fprintf(&b, "Date: %s\r\nMIME-Version: 1.0\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	for _, part := range []struct{ kind, body string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		fmt.Fprintf(&b, "--%s\r\nContent-Type: %s; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", boundary, part.kind)
		w := quotedprintable.NewWriter(&b)
		_, _ = w.Write([]byte(strings.ReplaceAll(part.body, "\n", "\r\n")))
		_ = w.Close()
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// dial connects and signs in; the caller sends and closes.
func dial(c Config) (*smtp.Client, error) {
	addr := net.JoinHostPort(c.Host, c.Port)
	tlsCfg := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 20 * time.Second}
	var conn net.Conn
	var err error
	if c.Port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("couldn't reach the mail server %s: check its name and port under Admin → Email", addr)
	}
	_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
	cl, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if ok, _ := cl.Extension("STARTTLS"); ok && c.Port != "465" {
		if err := cl.StartTLS(tlsCfg); err != nil {
			cl.Close()
			return nil, err
		}
	}
	if c.Username != "" {
		if err := cl.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			cl.Close()
			return nil, authError(c, err)
		}
	}
	return cl, nil
}

func deliver(c Config, to string, msg []byte) error {
	cl, err := dial(c)
	if err != nil {
		return err
	}
	defer cl.Close()
	if err := cl.Mail(c.From); err != nil {
		return err
	}
	if err := cl.Rcpt(to); err != nil {
		return err
	}
	wc, err := cl.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return cl.Quit()
}
