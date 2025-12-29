package scan

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func EnrichListenersWithProcessInfo(ctx context.Context, listeners []Listener) {
	byPID := map[int]*Listener{}
	for i := range listeners {
		if listeners[i].PID <= 0 {
			continue
		}
		if _, ok := byPID[listeners[i].PID]; ok {
			continue
		}
		byPID[listeners[i].PID] = &listeners[i]
	}
	if len(byPID) == 0 {
		return
	}

	fillFromPS(ctx, byPID)
	fillProcPaths(ctx, byPID)
}

func fillFromPS(ctx context.Context, byPID map[int]*Listener) {
	if _, err := exec.LookPath("ps"); err != nil {
		return
	}

	var pids []string
	for pid := range byPID {
		pids = append(pids, strconv.Itoa(pid))
	}
	cmd := exec.CommandContext(ctx, "ps", "-p", strings.Join(pids, ","), "-o", "pid=", "-o", "ppid=", "-o", "command=")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields, rest := splitFieldsWithRemainder(line, 2)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, _ := strconv.Atoi(fields[1])
		listener := byPID[pid]
		if listener == nil {
			continue
		}
		if ppid > 0 {
			listener.PPID = ppid
		}
		if rest != "" {
			listener.CommandLine = strings.TrimSpace(rest)
		}
	}
}

func fillProcPaths(ctx context.Context, byPID map[int]*Listener) {
	if runtime.GOOS == "linux" {
		for pid, listener := range byPID {
			cwd, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "cwd"))
			if err == nil && cwd != "" {
				listener.CWD = cwd
			}
			exe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
			if err == nil && exe != "" {
				listener.Executable = exe
			}
		}
		return
	}

	if _, err := exec.LookPath("lsof"); err != nil {
		return
	}
	for pid, listener := range byPID {
		cwd, exe := lsofProcPaths(ctx, pid)
		if cwd != "" {
			listener.CWD = cwd
		}
		if exe != "" {
			listener.Executable = exe
		}
	}
}

func lsofProcPaths(ctx context.Context, pid int) (string, string) {
	cmd := exec.CommandContext(ctx, "lsof", "-p", strconv.Itoa(pid), "-a", "-d", "cwd,txt", "-Fn")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", ""
	}
	if err := cmd.Start(); err != nil {
		return "", ""
	}
	defer cmd.Wait()

	var cwd string
	var exe string
	currentFD := ""
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'f':
			currentFD = strings.TrimPrefix(line, "f")
		case 'n':
			if currentFD == "cwd" {
				cwd = strings.TrimPrefix(line, "n")
			}
			if currentFD == "txt" {
				exe = strings.TrimPrefix(line, "n")
			}
		}
	}
	return cwd, exe
}

func splitFieldsWithRemainder(line string, n int) ([]string, string) {
	fields := make([]string, 0, n)
	i := 0
	for len(fields) < n && i < len(line) {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		start := i
		for i < len(line) && line[i] != ' ' {
			i++
		}
		if start < i {
			fields = append(fields, line[start:i])
		}
	}
	for i < len(line) && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return fields, ""
	}
	return fields, line[i:]
}
