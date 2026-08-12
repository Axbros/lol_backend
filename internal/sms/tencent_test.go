package sms

import "testing"

func TestFormatMobile(t *testing.T) {
	tests := map[string]string{
		"18706078522":     "+8618706078522",
		"+8618706078522":  "+8618706078522",
		"008618706078522": "008618706078522",
	}
	for input, want := range tests {
		if got := FormatMobile(input, "+86"); got != want {
			t.Fatalf("formatMobile(%q) = %q, want %q", input, got, want)
		}
	}
}
