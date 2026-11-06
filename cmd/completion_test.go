package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestListeningPortCompletionLive(t *testing.T) {
	bin := buildCLI(t)
	var ports []int
	for range 3 {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		t.Cleanup(func() { listener.Close() })
		ports = append(ports, listener.Addr().(*net.TCPAddr).Port)
	}
	// Two sockets on one port must still produce just one candidate.
	duplicate, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.2", itoa(ports[0])))
	if err != nil {
		t.Logf("second loopback address unavailable; fixture test still covers deduplication: %v", err)
	} else {
		t.Cleanup(func() { duplicate.Close() })
	}

	for _, command := range []string{"who", "kill"} {
		t.Run(command, func(t *testing.T) {
			all := cliPortCompletions(t, bin, command, "")
			previous := 0
			found := make(map[int]string)
			for _, candidate := range all {
				portText, description, _ := strings.Cut(candidate, "\t")
				port, err := strconv.Atoi(portText)
				if err != nil || port <= previous {
					t.Fatalf("candidates not unique and numerically sorted: %q", all)
				}
				previous = port
				found[port] = description
			}
			for _, port := range ports {
				if !strings.Contains(found[port], fmt.Sprintf("(pid %d)", os.Getpid())) {
					t.Fatalf("port %d missing current process description: %q", port, all)
				}
			}

			portText := itoa(ports[0])
			prefix := portText[:len(portText)-1]
			matches := cliPortCompletions(t, bin, command, prefix)
			var matchedPort bool
			for _, candidate := range matches {
				if !strings.HasPrefix(candidate, prefix) {
					t.Fatalf("candidate %q does not match prefix %q", candidate, prefix)
				}
				port, _, _ := strings.Cut(candidate, "\t")
				matchedPort = matchedPort || port == portText
			}
			if !matchedPort {
				t.Fatalf("prefix %q omitted port %s: %q", prefix, portText, matches)
			}
			if got := cliPortCompletions(t, bin, command, portText); len(got) != 1 || !strings.HasPrefix(got[0], portText+"\t") {
				t.Fatalf("exact port completion: %q", got)
			}
			for _, args := range [][]string{{"not-a-port"}, {portText, ""}, {portText, prefix}} {
				if got := cliPortCompletions(t, bin, command, args...); len(got) != 0 {
					t.Fatalf("unexpected candidates for %q: %q", args, got)
				}
			}

			// Completion must observe listeners opened after earlier requests.
			fresh, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			t.Cleanup(func() { fresh.Close() })
			freshPort := itoa(fresh.Addr().(*net.TCPAddr).Port)
			if got := cliPortCompletions(t, bin, command, freshPort); len(got) != 1 || !strings.HasPrefix(got[0], freshPort+"\t") {
				t.Fatalf("new listener not offered: %q", got)
			}
			if err := fresh.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if got := cliPortCompletions(t, bin, command, freshPort); len(got) != 0 {
				t.Fatalf("closed listener still offered: %q", got)
			}
		})
	}
}

func TestListeningPortCompletionScanner(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	// Controlled scanner output exercises ordering across digit lengths and
	// aggregation of multiple processes without requiring privileged port binds.
	fixture := `#!/bin/sh
printf '%s\n' \
  'COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME' \
  'node 20 user 3u IPv4 1 0t0 TCP 127.0.0.1:10000 (LISTEN)' \
  'ruby 30 user 3u IPv4 1 0t0 TCP 127.0.0.1:900 (LISTEN)' \
  'node 20 user 4u IPv6 1 0t0 TCP [::1]:10000 (LISTEN)' \
  'worker 21 user 3u IPv4 1 0t0 TCP 127.0.0.2:10000 (LISTEN)' \
  'web 40 user 3u IPv4 1 0t0 TCP 127.0.0.1:80 (LISTEN)'
`
	if err := os.WriteFile(filepath.Join(dir, "lsof"), []byte(fixture), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	for _, command := range []string{"who", "kill"} {
		got := cliPortCompletions(t, bin, command, "")
		want := []string{"80\tweb (pid 40)", "900\truby (pid 30)", "10000\tnode (pid 20); worker (pid 21)"}
		if !slices.Equal(got, want) {
			t.Fatalf("%s candidates = %q, want %q", command, got, want)
		}
	}

	// No scanner in PATH forces ListTCPListeners to return an error.
	t.Setenv("PATH", t.TempDir())
	for _, command := range []string{"who", "kill"} {
		if got := cliPortCompletions(t, bin, command, ""); len(got) != 0 {
			t.Fatalf("%s offered candidates despite scanner failure: %q", command, got)
		}
	}
}

func cliPortCompletions(t *testing.T, bin, command string, args ...string) []string {
	t.Helper()
	cliArgs := append([]string{"__complete", command}, args...)
	code, out, errOut := runCLI(bin, cliArgs...)
	if code != 0 {
		t.Fatalf("%q exited %d (stdout=%q stderr=%q)", cliArgs, code, out, errOut)
	}
	lines := nonEmptyLines(out)
	directive := fmt.Sprintf(":%d", cobra.ShellCompDirectiveNoFileComp)
	if len(lines) == 0 || lines[len(lines)-1] != directive {
		t.Fatalf("%q missing no-file-completion directive %s: %q", cliArgs, directive, out)
	}
	return lines[:len(lines)-1]
}
