package ports

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Range struct {
	Start int
	End   int
}

func ParseRange(s string) (Range, error) {
	parts := strings.Split(strings.TrimSpace(s), "-")
	if len(parts) != 2 {
		return Range{}, fmt.Errorf("invalid range %q (expected start-end)", s)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return Range{}, fmt.Errorf("invalid range start %q: %w", parts[0], err)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return Range{}, fmt.Errorf("invalid range end %q: %w", parts[1], err)
	}
	if start < 1 || end > 65535 || start > end {
		return Range{}, fmt.Errorf("invalid range %q (must be 1-65535 and start<=end)", s)
	}
	return Range{Start: start, End: end}, nil
}

func PickTCPPort(prefer []int, r Range) (int, error) {
	for _, p := range prefer {
		if p < 1 || p > 65535 {
			continue
		}
		if ok := probeTCP(p); ok {
			return p, nil
		}
	}
	for p := r.Start; p <= r.End; p++ {
		if ok := probeTCP(p); ok {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free TCP port found in %d-%d", r.Start, r.End)
}

func probeTCP(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

