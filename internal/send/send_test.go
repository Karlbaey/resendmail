package send

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSendValidationShortCircuit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		email   *EmailPayload
		wantErr string
	}{
		{
			name: "missing from",
			email: &EmailPayload{
				To:      []string{"receiver@example.com"},
				Subject: "Hello",
				Text:    "body",
			},
			wantErr: "sender is required",
		},
		{
			name: "missing receiver",
			email: &EmailPayload{
				From:    "sender@example.com",
				Subject: "Hello",
				Text:    "body",
			},
			wantErr: "receiver",
		},
		{
			name: "missing subject",
			email: &EmailPayload{
				From: "sender@example.com",
				To:   []string{"receiver@example.com"},
				Text: "body",
			},
			wantErr: "subject is required",
		},
		{
			name: "missing body",
			email: &EmailPayload{
				From:    "sender@example.com",
				To:      []string{"receiver@example.com"},
				Subject: "Hello",
			},
			wantErr: "cannot both be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Send(context.Background(), "test-key", tt.email)
			if err == nil {
				t.Fatalf("Send() expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Send() error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// Send does not expose a base URL override, so the happy path cannot run
// against a fake server. The invalid-payload cases above prove validation
// runs before any network call; the network layer itself is covered by the
// SendEmail tests in client_test.go.
func TestSendInvalidEmailDoesNotHitNetwork(t *testing.T) {
	t.Parallel()

	// A payload that is valid except for attachments: validation must fail
	// before client creation, so no HTTP request is ever attempted.
	email := &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
		Attachments: []Attachment{
			{Filename: "a.txt", Content: "", Path: ""},
		},
	}
	_, err := Send(context.Background(), "test-key", email)
	if err == nil {
		t.Fatal("Send() expected attachment validation error, got nil")
	}
	var apiErr *EmailResponseError
	if errors.As(err, &apiErr) {
		t.Fatalf("Send() must fail during validation, not with an API error: %v", err)
	}
}