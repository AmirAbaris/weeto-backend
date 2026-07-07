package sms

import "testing"

func TestNormalizeMobile(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"09121234567", "9121234567"},
		{"+989121234567", "9121234567"},
		{"989121234567", "9121234567"},
		{"9121234567", "9121234567"},
		{" 0912 123 4567 ", "9121234567"},
	}

	for _, tc := range tests {
		got, err := NormalizeMobile(tc.in)
		if err != nil {
			t.Fatalf("NormalizeMobile(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeMobile(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeMobileInvalid(t *testing.T) {
	invalid := []string{"123", "08121234567", "abc"}
	for _, in := range invalid {
		if _, err := NormalizeMobile(in); err == nil {
			t.Fatalf("NormalizeMobile(%q) expected error", in)
		}
	}
}
