package send

import (
	"strings"
	"testing"
)

func TestEmailPayloadValidate(t *testing.T) {
	t.Parallel()

	valid := &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body text",
	}

	tests := []struct {
		name    string
		mutate  func(p *EmailPayload)
		wantErr string // substring expected in error; empty means no error
	}{
		{
			name:    "valid with text",
			mutate:  func(p *EmailPayload) {},
			wantErr: "",
		},
		{
			name:    "valid with html",
			mutate:  func(p *EmailPayload) { p.Text = ""; p.HTML = "<p>hi</p>" },
			wantErr: "",
		},
		{
			name:    "missing from",
			mutate:  func(p *EmailPayload) { p.From = "" },
			wantErr: "sender is required",
		},
		{
			name:    "missing to",
			mutate:  func(p *EmailPayload) { p.To = nil },
			wantErr: "receiver",
		},
		{
			name:    "empty to slice",
			mutate:  func(p *EmailPayload) { p.To = []string{} },
			wantErr: "receiver",
		},
		{
			name:    "missing subject",
			mutate:  func(p *EmailPayload) { p.Subject = "" },
			wantErr: "subject is required",
		},
		{
			name:    "both text and html empty",
			mutate:  func(p *EmailPayload) { p.Text = ""; p.HTML = "" },
			wantErr: "cannot both be empty",
		},
		{
			name:    "text and html both set",
			mutate:  func(p *EmailPayload) { p.HTML = "<p>hi</p>" },
			wantErr: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload := *valid
			tt.mutate(&payload)

			err := payload.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateAttachments(t *testing.T) {
	t.Parallel()

	payload := &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body text",
	}

	t.Run("valid attachments", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "a.txt", Content: "AAAA"},
			{Filename: "b.bin", Content: "BBBB"},
		}
		if err := payload.Validate(); err != nil {
			t.Fatalf("Validate() expected nil error, got %v", err)
		}
	})

	t.Run("empty filename", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "", Content: "AAAA"},
		}
		err := payload.Validate()
		if err == nil || !strings.Contains(err.Error(), "attachment number 0 does not exist") {
			t.Fatalf("Validate() expected error about missing filename, got %v", err)
		}
	})

	t.Run("content and path both empty", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "a.txt"},
		}
		err := payload.Validate()
		if err == nil || !strings.Contains(err.Error(), "content and path cannot both be empty") {
			t.Fatalf("Validate() expected error about empty content/path, got %v", err)
		}
	})

	t.Run("content and path both set", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "a.txt", Content: "AAAA", Path: "/tmp/a.txt"},
		}
		err := payload.Validate()
		if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("Validate() expected error about mutually exclusive content/path, got %v", err)
		}
	})

	t.Run("size over 40MB limit", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "big.bin", Content: strings.Repeat("A", 40*1024*1024)},
		}
		err := payload.Validate()
		if err == nil || !strings.Contains(err.Error(), "over the limit") {
			t.Fatalf("Validate() expected size limit error, got %v", err)
		}
	})

	t.Run("size exactly at 40MB is allowed", func(t *testing.T) {
		t.Parallel()
		payload.Attachments = []Attachment{
			{Filename: "big.bin", Content: strings.Repeat("A", 40*1024*1024-1)},
		}
		if err := payload.Validate(); err != nil {
			t.Fatalf("Validate() unexpected error: %v", err)
		}
	})
}

func TestEmailResponseErrorError(t *testing.T) {
	t.Parallel()

	err := &EmailResponseError{StatusCode: 401, Message: "invalid api key", Name: "unauthorized_error"}
	want := "Resend API error [unauthorized_error] (HTTP 401): invalid api key"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}