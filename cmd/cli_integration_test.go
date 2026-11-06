package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"fp/internal/scan"
)

func TestCheckExitCodes(t *testing.T) {
	bin := buildCLI(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	code, out, errOut := runCLI(bin, "check", itoa(port))
	if code != 1 {
		t.Fatalf("expected exit 1 for in-use, got %d (out=%q err=%q)", code, out, errOut)
	}
	code, out, errOut = runCLI(bin, "check", itoa(port), "-q")
	if code != 1 || out != "" || errOut != "" {
		t.Fatalf("quiet in-use: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	code, out, errOut = runCLI(bin, "check", itoa(port))
	if code != 0 {
		t.Fatalf("expected exit 0 for free, got %d (out=%q err=%q)", code, out, errOut)
	}
	code, out, errOut = runCLI(bin, "check", itoa(port), "--quiet")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("quiet free: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

func TestCheckQuietErrors(t *testing.T) {
	bin := buildCLI(t)
	for _, args := range [][]string{
		{"check", "--quiet", "invalid"},
		{"check", "--quiet"},
		{"check", "--quiet", "3000", "--unknown"},
		{"check", "--quiet", "--json", "3000"},
		{"--json", "check", "--quiet", "3000"},
	} {
		code, out, errOut := runCLI(bin, args...)
		if code != 2 || out != "" || errOut == "" {
			t.Errorf("%v: expected exit 2 and stderr only, got exit=%d stdout=%q stderr=%q", args, code, out, errOut)
		}
	}
}

func TestRunRequiresDashDash(t *testing.T) {
	bin := buildCLI(t)

	code, _, _ := runCLI(bin, "run", "echo", "ok")
	if code == 0 {
		t.Fatalf("expected non-zero exit for missing --")
	}
}

func TestRunSetsPort(t *testing.T) {
	bin := buildCLI(t)

	code, _, errOut := runCLI(bin, "run", "--", "/bin/sh", "-c", "test -n \"$PORT\"")
	if code != 0 {
		t.Fatalf("expected exit 0 for run, got %d (stderr=%q)", code, errOut)
	}
	if !strings.Contains(errOut, "fp: using port") {
		t.Fatalf("expected chosen port message in stderr, got %q", errOut)
	}
}

func TestRunPreservesExitStatus(t *testing.T) {
	bin := buildCLI(t)
	for _, tc := range []struct {
		script string
		code   int
	}{
		{"exit 0", 0},
		{"exit 7", 7},
		{"exit 42", 42},
		{"kill -TERM $$", 143},
	} {
		code, out, errOut := runCLI(bin, "run", "--", "/bin/sh", "-c", "echo child-output; echo child-error >&2; "+tc.script)
		if code != tc.code || out != "child-output\n" || !strings.Contains(errOut, "child-error\n") {
			t.Errorf("%q: exit=%d stdout=%q stderr=%q", tc.script, code, out, errOut)
		}
		if strings.Contains(errOut, "Usage:") || strings.Contains(errOut, "Error:") {
			t.Errorf("%q: unexpected wrapper diagnostics: %q", tc.script, errOut)
		}
	}
}

func TestListWorkingDirectory(t *testing.T) {
	bin := buildCLI(t)
	base := t.TempDir()
	project := filepath.Join(base, "project")
	subdir := filepath.Join(project, "subdir")
	sibling := filepath.Join(base, "project-other")
	for _, dir := range []string{subdir, sibling} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	projectPorts := startListenerProcess(t, project, false)
	subPorts := startListenerProcess(t, subdir, false)
	siblingPorts := startListenerProcess(t, sibling, false)
	link := filepath.Join(base, "linked-project")
	if err := os.Symlink(project, link); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{project, link, "."} {
		cmd := exec.Command(bin, "list", "--cwd", dir, "--json")
		cmd.Dir = project
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("list --cwd %s: %v\n%s", dir, err, out)
		}
		var listeners []scan.Listener
		if err := json.Unmarshal(out, &listeners); err != nil {
			t.Fatal(err)
		}
		found := make(map[int]bool)
		for _, l := range listeners {
			found[l.Port] = true
		}
		for _, port := range append(projectPorts, subPorts...) {
			if !found[port] {
				t.Errorf("--cwd %s omitted project port %d: %s", dir, port, out)
			}
		}
		for _, port := range siblingPorts {
			if found[port] {
				t.Errorf("--cwd %s included sibling port %d", dir, port)
			}
		}
	}
	code, out, errOut := runCLI(bin, "list", "--cwd", project, "--port", itoa(subPorts[1]), "--unique", "-v", "--json")
	var listeners []scan.Listener
	if err := json.Unmarshal([]byte(out), &listeners); err != nil || code != 0 || len(listeners) != 1 || listeners[0].Port != subPorts[1] {
		t.Fatalf("combined filters: exit=%d stdout=%q stderr=%q decode=%v", code, out, errOut, err)
	}
	code, out, errOut = runCLI(bin, "list", "definitely-not-a-process-name", "--cwd", project, "--json")
	if code != 0 || !json.Valid([]byte(out)) || strings.TrimSpace(out) != "[]" {
		t.Fatalf("combined name filter: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	for _, dir := range []string{filepath.Join(base, "missing"), bin} {
		code, _, errOut := runCLI(bin, "list", "--cwd", dir)
		if code == 0 || errOut == "" {
			t.Errorf("expected error for --cwd %s, exit=%d stderr=%q", dir, code, errOut)
		}
	}
}

func TestKillJSON(t *testing.T) {
	bin := buildCLI(t)
	for _, tc := range []struct {
		name       string
		flags      []string
		ignoreTERM bool
		status     string
	}{
		{"term", nil, false, "signaled"},
		{"escalation", []string{"--timeout", "10ms"}, true, "signaled"},
		{"no-wait", []string{"--timeout", "0"}, false, "signaled"},
		{"kill", []string{"--signal", "KILL"}, false, "signaled"},
		{"dry-run", []string{"--dry-run"}, false, "dry-run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ports := startListenerProcess(t, t.TempDir(), tc.ignoreTERM)
			args := []string{"--json", "kill", itoa(ports[0])}
			args = append(args, tc.flags...)
			code, out, errOut := runCLI(bin, args...)
			var result struct {
				Port     int             `json:"port"`
				Status   string          `json:"status"`
				Signaled int             `json:"signaled"`
				Targets  []scan.Listener `json:"targets"`
			}
			if err := json.Unmarshal([]byte(out), &result); err != nil || code != 0 || result.Port != ports[0] || result.Status != tc.status {
				t.Fatalf("exit=%d stdout=%q stderr=%q decode=%v", code, out, errOut, err)
			}
			if tc.status == "dry-run" {
				if len(result.Targets) != 1 {
					t.Fatalf("expected one target, got %+v", result)
				}
				code, _, _ = runCLI(bin, "check", itoa(ports[0]), "-q")
				if code != 1 {
					t.Fatal("dry-run stopped the listener")
				}
			} else {
				if result.Signaled != 1 {
					t.Fatalf("expected one signaled process, got %+v", result)
				}
				code, _, errOut = runCLI(bin, "check", itoa(ports[0]), "--wait", "1s", "-q")
				if code != 0 {
					t.Fatalf("listener still running: exit=%d stderr=%q", code, errOut)
				}
				code, out, errOut = runCLI(bin, "kill", itoa(ports[0]), "--json")
				if err := json.Unmarshal([]byte(out), &result); err != nil || code != 0 || result.Status != "idle" || result.Signaled != 0 {
					t.Fatalf("idle: exit=%d stdout=%q stderr=%q decode=%v", code, out, errOut, err)
				}
			}
		})
	}
}

// Run this test binary as a short-lived listener fixture, without another tool dependency.
func TestListenerProcess(t *testing.T) {
	if os.Getenv("FP_TEST_LISTENER") == "" {
		return
	}
	if os.Getenv("FP_TEST_IGNORE_TERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	var ports []int
	for range 2 {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		ports = append(ports, ln.Addr().(*net.TCPAddr).Port)
	}
	if err := json.NewEncoder(os.Stdout).Encode(ports); err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func startListenerProcess(t *testing.T, dir string, ignoreTERM bool) []int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestListenerProcess$")
	child.Dir = dir
	child.Env = append(os.Environ(), "FP_TEST_LISTENER=1", "FP_TEST_IGNORE_TERM=0")
	if ignoreTERM {
		child.Env = append(child.Env, "FP_TEST_IGNORE_TERM=1")
	}
	child.Stderr = os.Stderr
	stdin, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	var ports []int
	if err := json.NewDecoder(stdout).Decode(&ports); err != nil {
		t.Fatalf("listener fixture: %v", err)
	}
	return ports
}

func TestListUniqueFiltersDuplicates(t *testing.T) {
	bin := buildCLI(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	code, out, errOut := runCLI(bin, "list", "--port", itoa(port), "--unique")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (out=%q err=%q)", code, out, errOut)
	}

	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 line, got %d lines: %q", len(lines), lines)
	}
}

func TestKillDryRunDoesNotError(t *testing.T) {
	bin := buildCLI(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	code, out, errOut := runCLI(bin, "kill", itoa(port), "--dry-run")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (out=%q err=%q)", code, out, errOut)
	}
	if !strings.Contains(out, "would signal") {
		t.Fatalf("expected dry-run output, got %q", out)
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Dir(cwd)

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "fp")

	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}

	return bin
}

func runCLI(bin string, args ...string) (int, string, string) {
	cmd := exec.Command(bin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = 2
		}
	}
	return code, stdout.String(), stderr.String()
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
