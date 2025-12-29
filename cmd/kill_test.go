package cmd

import "testing"

func TestParseSignal(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"TERM", true},
		{"SIGTERM", true},
		{"INT", true},
		{"SIGINT", true},
		{"KILL", true},
		{"SIGKILL", true},
		{"", false},
		{"HUP", false},
	}

	for _, tc := range cases {
		_, err := parseSignal(tc.in)
		if tc.valid && err != nil {
			t.Fatalf("expected %q to be valid: %v", tc.in, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("expected %q to be invalid", tc.in)
		}
	}
}

