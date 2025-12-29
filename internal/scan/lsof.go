package scan

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func listTCPListenersViaLsof(ctx context.Context) ([]Listener, error) {
	c := exec.CommandContext(ctx, "lsof", "-nP", "-iTCP", "-sTCP:LISTEN")
	out, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	defer c.Wait()

	listeners, err := parseLsofOutput(out)
	if err != nil {
		return nil, err
	}
	return listeners, nil
}

func parseLsofOutput(r io.Reader) ([]Listener, error) {
	var listeners []Listener
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if strings.HasPrefix(line, "COMMAND ") {
				continue
			}
		}

		listener, ok := parseLsofLine(line)
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

func parseLsofLine(line string) (Listener, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return Listener{}, false
	}

	command := fields[0]
	pid, _ := strconv.Atoi(fields[1])
	user := fields[2]

	addr, port := parseLsofAddressAndPort(fields)
	if port == 0 {
		return Listener{}, false
	}

	return Listener{
		Port:    port,
		PID:     pid,
		User:    user,
		Command: command,
		Proto:   "tcp",
		Address: addr,
	}, true
}

func parseLsofAddressAndPort(fields []string) (addr string, port int) {
	for i := len(fields) - 1; i >= 0; i-- {
		token := fields[i]
		if token == "(LISTEN)" {
			continue
		}

		// Common shapes:
		//   *:3000
		//   127.0.0.1:3000
		//   [::1]:3000
		//   localhost:3000
		lastColon := strings.LastIndex(token, ":")
		if lastColon < 0 || lastColon == len(token)-1 {
			continue
		}
		p, err := strconv.Atoi(token[lastColon+1:])
		if err != nil || p < 1 || p > 65535 {
			continue
		}
		return token, p
	}
	return "", 0
}
