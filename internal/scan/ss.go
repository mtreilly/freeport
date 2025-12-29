package scan

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var ssPid = regexp.MustCompile(`pid=(\d+)`)
var ssProc = regexp.MustCompile(`\"([^\"]+)\"`)

func listTCPListenersViaSS(ctx context.Context) ([]Listener, error) {
	// Example:
	// LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("node",pid=12345,fd=22))
	c := exec.CommandContext(ctx, "ss", "-ltnpH")
	out, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	defer c.Wait()

	listeners, err := parseSSOutput(out)
	if err != nil {
		return nil, err
	}
	return listeners, nil
}

func parseSSOutput(r io.Reader) ([]Listener, error) {
	var listeners []Listener
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		listener, ok := parseSSLine(line)
		if !ok {
			continue
		}
		listeners = append(listeners, listener)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return listeners, nil
}

func parseSSLine(line string) (Listener, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return Listener{}, false
	}

	local, ok := extractSSLocal(fields)
	if !ok {
		return Listener{}, false
	}

	p, ok := parsePortFromAddress(local)
	if !ok {
		return Listener{}, false
	}

	pid := 0
	if pm := ssPid.FindStringSubmatch(line); len(pm) == 2 {
		pid, _ = strconv.Atoi(pm[1])
	}

	cmdName := ""
	if cm := ssProc.FindStringSubmatch(line); len(cm) == 2 {
		cmdName = cm[1]
	}

	return Listener{
		Port:    p,
		PID:     pid,
		Command: cmdName,
		Proto:   "tcp",
		Address: local,
	}, true
}

func extractSSLocal(fields []string) (string, bool) {
	// ss output usually: State Recv-Q Send-Q Local Address:Port Peer Address:Port
	// After splitting, Local is often fields[3] but may appear later when fields vary.
	if len(fields) > 3 && strings.Contains(fields[3], ":") {
		return fields[3], true
	}
	for i := 0; i < len(fields); i++ {
		if strings.Contains(fields[i], ":") && !strings.Contains(fields[i], "users:") {
			return fields[i], true
		}
	}
	return "", false
}

func parsePortFromAddress(addr string) (int, bool) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 || lastColon == len(addr)-1 {
		return 0, false
	}
	portStr := addr[lastColon+1:]
	p, err := strconv.Atoi(portStr)
	if err != nil || p < 1 || p > 65535 {
		return 0, false
	}
	return p, true
}
