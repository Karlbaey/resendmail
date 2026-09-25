package constants

import "testing"

func TestResendAPIBase(t *testing.T) {
	t.Parallel()

	if ResendAPIBase != "https://api.resend.com" {
		t.Fatalf("ResendAPIBase = %q, want %q", ResendAPIBase, "https://api.resend.com")
	}
}