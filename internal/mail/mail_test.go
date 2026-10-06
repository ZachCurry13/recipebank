package mail

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// fakeSMTP accepts one message and returns what it received.
func fakeSMTP(t *testing.T) (port string, got chan string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	got = make(chan string, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { conn.Write([]byte(s + "\r\n")) }
		say("220 fake")
		var log strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			log.WriteString(line)
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case cmd == "DATA":
				say("354 go on")
				for {
					l, err := r.ReadString('\n')
					if err != nil || l == ".\r\n" {
						break
					}
					log.WriteString(l)
				}
				say("250 queued")
			case cmd == "QUIT":
				say("221 bye")
				got <- log.String()
				return
			default:
				say("250 OK")
			}
		}
	}()
	_, port, _ = net.SplitHostPort(ln.Addr().String())
	return port, got
}

func TestSend(t *testing.T) {
	port, got := fakeSMTP(t)
	c := Config{Host: "127.0.0.1", Port: port, From: "kitchen@example.com"}
	err := Send(c, Message{To: "Grandma <grandma@example.com>", Subject: "Crêpes from RecipeBank",
		Text: "Crêpes\n\n- 1 cup flour", HTML: "<h1>Crêpes</h1>"})
	if err != nil {
		t.Fatal(err)
	}
	log := <-got
	for _, want := range []string{"RCPT TO:<grandma@example.com>", "MAIL FROM:<kitchen@example.com>", "Subject: =?utf-8?q?",
		"Content-Type: text/plain; charset=utf-8", "Content-Type: text/html; charset=utf-8", "1 cup flour"} {
		if !strings.Contains(log, want) {
			t.Errorf("missing %q in:\n%s", want, log)
		}
	}
}

func TestAddress(t *testing.T) {
	for _, bad := range []string{"", "nobody", "a@example.com, b@example.com", "a@example.com\r\nBcc: c@example.com"} {
		if _, err := Address(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
	if a, err := Address(" Mom <mom@example.com> "); err != nil || a != "mom@example.com" {
		t.Errorf("got %q, %v", a, err)
	}
	if err := (Config{}).Validate(); err == nil || !strings.Contains(err.Error(), "Admin") {
		t.Errorf("an empty setup: %v", err)
	}
}
