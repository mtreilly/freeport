package lock

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"fp/internal/ports"
	"golang.org/x/sys/unix"
)

type Handle struct {
	f *os.File
}

func (h *Handle) Close() error {
	if h == nil || h.f == nil {
		return nil
	}
	_ = unix.Flock(int(h.f.Fd()), unix.LOCK_UN)
	return h.f.Close()
}

func PickAndLockTCPPort(prefer []int, r ports.Range) (int, *Handle, error) {
	dir, err := lockDir()
	if err != nil {
		return 0, nil, err
	}

	tryPort := func(p int) (int, *Handle, bool) {
		h, err := tryLockPortFile(dir, p)
		if err != nil {
			return 0, nil, false
		}
		if ok := portsPickProbe(p); !ok {
			_ = h.Close()
			return 0, nil, false
		}
		return p, h, true
	}

	for _, p := range prefer {
		if p < 1 || p > 65535 {
			continue
		}
		if chosen, h, ok := tryPort(p); ok {
			return chosen, h, nil
		}
	}
	for p := r.Start; p <= r.End; p++ {
		if chosen, h, ok := tryPort(p); ok {
			return chosen, h, nil
		}
	}
	return 0, nil, fmt.Errorf("no free TCP port found in %d-%d", r.Start, r.End)
}

func lockDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "fp", "locks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func tryLockPortFile(dir string, port int) (*Handle, error) {
	path := filepath.Join(dir, fmt.Sprintf("%d.lock", port))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Handle{f: f}, nil
}

// Duplicate of ports.probeTCP but kept local so PickAndLock can remain race-minimizing:
// hold lock while probing so concurrent `fp run` calls don't pick the same port.
func portsPickProbe(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
