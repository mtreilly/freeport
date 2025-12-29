package scan

import (
	"strings"
	"testing"
)

func TestParseSSOutput(t *testing.T) {
	input := strings.TrimSpace(`
LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("node",pid=12345,fd=22))
LISTEN 0 128 [::1]:6379 [::]:* users:(("redis-server",pid=555,fd=7))
LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1,fd=3))
LISTEN 0 128 [::]:443 [::]:* users:(("nginx",pid=2000,fd=9))
`)

	listeners, err := parseSSOutput(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSSOutput error: %v", err)
	}

	if len(listeners) != 4 {
		t.Fatalf("expected 4 listeners, got %d", len(listeners))
	}

	assertListener(t, listeners[0], 3000, 12345, "", "node", "127.0.0.1:3000")
	assertListener(t, listeners[1], 6379, 555, "", "redis-server", "[::1]:6379")
	assertListener(t, listeners[2], 22, 1, "", "sshd", "0.0.0.0:22")
	assertListener(t, listeners[3], 443, 2000, "", "nginx", "[::]:443")
}

func TestParseSSLineWithoutProcessInfo(t *testing.T) {
	line := "LISTEN 0 4096 127.0.0.1:8080 0.0.0.0:*"
	listener, ok := parseSSLine(line)
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if listener.PID != 0 {
		t.Fatalf("expected pid 0, got %d", listener.PID)
	}
	if listener.Command != "" {
		t.Fatalf("expected empty command, got %q", listener.Command)
	}
	if listener.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", listener.Port)
	}
}

