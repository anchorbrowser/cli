package cli

import "testing"

func TestNormalizeSemver(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "0.1.3", want: "v0.1.3"},
		{in: "v0.1.3", want: "v0.1.3"},
		{in: "dev", want: "v0.0.0"},
		{in: "", want: "v0.0.0"},
	}
	for _, tt := range tests {
		got := normalizeSemver(tt.in)
		if got != tt.want {
			t.Fatalf("normalizeSemver(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
