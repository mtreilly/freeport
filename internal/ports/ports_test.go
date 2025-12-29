package ports

import "testing"

func TestPickEphemeral(t *testing.T) {
	port, ok := pickEphemeral()
	if !ok {
		t.Fatalf("expected ephemeral pick to succeed")
	}
	if port < 1 || port > 65535 {
		t.Fatalf("unexpected port %d", port)
	}
}

