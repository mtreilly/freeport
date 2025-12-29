package cmd

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	code, out, errOut = runCLI(bin, "check", itoa(port))
	if code != 0 {
		t.Fatalf("expected exit 0 for free, got %d (out=%q err=%q)", code, out, errOut)
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
	if !strings.Contains(errOut, "freeport: using port") {
		t.Fatalf("expected chosen port message in stderr, got %q", errOut)
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
	bin := filepath.Join(tmp, "freeport")

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
