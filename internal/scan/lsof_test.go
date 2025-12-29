package scan

import (
	"strings"
	"testing"
)

func TestParseLsofOutput(t *testing.T) {
	input := strings.TrimSpace(`
COMMAND   PID USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
node     1234 alice  23u  IPv4 0x000000000  0t0    TCP *:3000 (LISTEN)
node     1235 alice  24u  IPv6 0x000000001  0t0    TCP [::1]:3000 (LISTEN)
python   777  bob    10u  IPv4 0x000000002  0t0    TCP 127.0.0.1:8000 (LISTEN)
redis    888  bob    10u  IPv6 0x000000003  0t0    TCP [::1]:6379 (LISTEN)
nginx    999  root   11u  IPv4 0x000000004  0t0    TCP *:http (LISTEN)
`)

	listeners, err := parseLsofOutput(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseLsofOutput error: %v", err)
	}

	if len(listeners) != 4 {
		t.Fatalf("expected 4 listeners, got %d", len(listeners))
	}

	assertListener(t, listeners[0], 3000, 1234, "alice", "node", "*:3000")
	assertListener(t, listeners[1], 3000, 1235, "alice", "node", "[::1]:3000")
	assertListener(t, listeners[2], 8000, 777, "bob", "python", "127.0.0.1:8000")
	assertListener(t, listeners[3], 6379, 888, "bob", "redis", "[::1]:6379")
}

func TestParseLsofLineSkipsNonNumericPorts(t *testing.T) {
	line := "nginx 999 root 11u IPv4 0x000000004 0t0 TCP *:http (LISTEN)"
	if _, ok := parseLsofLine(line); ok {
		t.Fatalf("expected non-numeric port to be skipped")
	}
}

func assertListener(t *testing.T, got Listener, port int, pid int, user, command, addr string) {
	t.Helper()
	if got.Port != port {
		t.Fatalf("expected port %d, got %d", port, got.Port)
	}
	if got.PID != pid {
		t.Fatalf("expected pid %d, got %d", pid, got.PID)
	}
	if got.User != user {
		t.Fatalf("expected user %q, got %q", user, got.User)
	}
	if got.Command != command {
		t.Fatalf("expected command %q, got %q", command, got.Command)
	}
	if got.Address != addr {
		t.Fatalf("expected addr %q, got %q", addr, got.Address)
	}
	if got.Proto != "tcp" {
		t.Fatalf("expected proto tcp, got %q", got.Proto)
	}
}

