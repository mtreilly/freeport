package scan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type Listener struct {
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	User    string `json:"user,omitempty"`
	Command string `json:"command,omitempty"`
	Proto   string `json:"proto,omitempty"`
	Address string `json:"address,omitempty"`
}

func ListTCPListeners(ctx context.Context) ([]Listener, error) {
	if _, err := exec.LookPath("lsof"); err == nil {
		return listTCPListenersViaLsof(ctx)
	}
	if _, err := exec.LookPath("ss"); err == nil {
		return listTCPListenersViaSS(ctx)
	}
	return nil, errors.New("no supported port lister found (need `lsof` or `ss` in PATH)")
}

func HasTCPListenerOnPort(ctx context.Context, port int) (bool, error) {
	listeners, err := ListTCPListeners(ctx)
	if err != nil {
		return false, err
	}
	for _, l := range listeners {
		if l.Port == port {
			return true, nil
		}
	}
	return false, nil
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

