package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunSendSuccess(t *testing.T) {
	t.Parallel()

	var gotReq SendRequest
	var gotAPIKey string
	var gotTimeout time.Duration

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := run([]string{
		"send",
		"--from", "sender@example.com",
		"--to", "first@example.com,second@example.com",
		"--cc", "copy@example.com",
		"--subject", "hello",
		"--text", "plain body",
		"--timeout", "15s",
	}, runtimeDeps{
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(key string) string {
			if key == envResendAPIKey {
				return "test-api-key"
			}
			return ""
		},
		readFile: func(string) ([]byte, error) {
			return nil, errors.New("unexpected file read")
		},
		newSender: func(apiKey string, timeout time.Duration) emailSender {
			gotAPIKey = apiKey
			gotTimeout = timeout
			return sendFunc(func(_ context.Context, req SendRequest) (SendResponse, error) {
				gotReq = req
				return SendResponse{ID: "email_123"}, nil
			})
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if gotAPIKey != "test-api-key" {
		t.Fatalf("expected API key to be passed through, got %q", gotAPIKey)
	}
	if gotTimeout != 15*time.Second {
		t.Fatalf("expected timeout 15s, got %s", gotTimeout)
	}
	if gotReq.From != "sender@example.com" {
		t.Fatalf("unexpected from: %q", gotReq.From)
	}
	if len(gotReq.To) != 2 || gotReq.To[0] != "first@example.com" || gotReq.To[1] != "second@example.com" {
		t.Fatalf("unexpected to list: %#v", gotReq.To)
	}
	if len(gotReq.Cc) != 1 || gotReq.Cc[0] != "copy@example.com" {
		t.Fatalf("unexpected cc list: %#v", gotReq.Cc)
	}
	if gotReq.Subject != "hello" {
		t.Fatalf("unexpected subject: %q", gotReq.Subject)
	}
	if gotReq.Text == nil || *gotReq.Text != "plain body" {
		t.Fatalf("unexpected text body: %#v", gotReq.Text)
	}
	if gotReq.HTML != nil {
		t.Fatalf("expected empty html body, got %#v", gotReq.HTML)
	}
	if stdout.String() != "{\"id\":\"email_123\"}\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunSendReadsBodyFile(t *testing.T) {
	t.Parallel()

	var gotReq SendRequest

	exitCode := run([]string{
		"send",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
		"--subject", "file body",
		"--html-file", "body.html",
	}, runtimeDeps{
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
		getenv: func(string) string {
			return "test-api-key"
		},
		readFile: func(path string) ([]byte, error) {
			if path != "body.html" {
				t.Fatalf("unexpected path: %q", path)
			}
			return []byte("<p>hello</p>"), nil
		},
		newSender: func(string, time.Duration) emailSender {
			return sendFunc(func(_ context.Context, req SendRequest) (SendResponse, error) {
				gotReq = req
				return SendResponse{ID: "email_456"}, nil
			})
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if gotReq.HTML == nil || *gotReq.HTML != "<p>hello</p>" {
		t.Fatalf("unexpected HTML body: %#v", gotReq.HTML)
	}
}

func TestRunSendRejectsMutuallyExclusiveBodySources(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := run([]string{
		"send",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
		"--subject", "hello",
		"--text", "inline",
		"--text-file", "body.txt",
	}, runtimeDeps{
		stdout:   &stdout,
		stderr:   &stderr,
		getenv:   func(string) string { return "test-api-key" },
		readFile: func(string) ([]byte, error) { return nil, nil },
		newSender: func(string, time.Duration) emailSender {
			t.Fatal("sender should not be called")
			return nil
		},
	})

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--text and --text-file are mutually exclusive") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunSendEntersInteractiveModeWhenRequiredFlagsAreMissing(t *testing.T) {
	t.Parallel()

	var gotPromptOpts sendOptions
	var gotReq SendRequest
	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := run([]string{
		"send",
		"--to", "recipient@example.com",
		"--timeout", "30s",
	}, runtimeDeps{
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(string) string {
			return "test-api-key"
		},
		readFile: func(string) ([]byte, error) {
			return nil, errors.New("unexpected file read")
		},
		promptSend: func(opts sendOptions, interrupt <-chan struct{}) (sendOptions, error) {
			if interrupt == nil {
				t.Fatal("expected interrupt channel to be provided")
			}
			gotPromptOpts = opts
			opts.From = "sender@example.com"
			opts.Subject = "interactive subject"
			opts.Text.Inline = singleStringFlag{name: "--text", value: "interactive body", set: true}
			return opts, nil
		},
		newSender: func(string, time.Duration) emailSender {
			return sendFunc(func(_ context.Context, req SendRequest) (SendResponse, error) {
				gotReq = req
				return SendResponse{ID: "email_interactive"}, nil
			})
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
	if gotPromptOpts.From != "" {
		t.Fatalf("expected missing from to be preserved for prompt, got %q", gotPromptOpts.From)
	}
	if len(gotPromptOpts.To.values) != 1 || gotPromptOpts.To.values[0] != "recipient@example.com" {
		t.Fatalf("unexpected to values passed to prompt: %#v", gotPromptOpts.To.values)
	}
	if gotPromptOpts.Subject != "" {
		t.Fatalf("expected missing subject to be preserved for prompt, got %q", gotPromptOpts.Subject)
	}
	if gotPromptOpts.Timeout != 30*time.Second {
		t.Fatalf("expected timeout to be preserved, got %s", gotPromptOpts.Timeout)
	}
	if gotReq.From != "sender@example.com" {
		t.Fatalf("unexpected from: %q", gotReq.From)
	}
	if gotReq.Subject != "interactive subject" {
		t.Fatalf("unexpected subject: %q", gotReq.Subject)
	}
	if gotReq.Text == nil || *gotReq.Text != "interactive body" {
		t.Fatalf("unexpected text body: %#v", gotReq.Text)
	}
	if stdout.String() != "{\"id\":\"email_interactive\"}\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunSendInteractiveModeFailureReturnsRuntimeError(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := run([]string{
		"send",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
	}, runtimeDeps{
		stdout:   &stdout,
		stderr:   &stderr,
		getenv:   func(string) string { return "test-api-key" },
		readFile: func(string) ([]byte, error) { return nil, nil },
		promptSend: func(sendOptions, <-chan struct{}) (sendOptions, error) {
			return sendOptions{}, errors.New("interactive mode is not implemented")
		},
		newSender: func(string, time.Duration) emailSender {
			t.Fatal("sender should not be called")
			return nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "interactive mode is not implemented") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunSendInteractiveCancellationReturnsZero(t *testing.T) {
	t.Parallel()

	exitCode := run([]string{
		"send",
		"--to", "recipient@example.com",
	}, runtimeDeps{
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
		getenv: func(string) string {
			return "test-api-key"
		},
		readFile: func(string) ([]byte, error) {
			return nil, nil
		},
		promptSend: func(sendOptions, <-chan struct{}) (sendOptions, error) {
			return sendOptions{}, errSendCancelled
		},
		newSender: func(string, time.Duration) emailSender {
			t.Fatal("sender should not be called")
			return nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSendInteractiveInterruptReturns130(t *testing.T) {
	t.Parallel()

	exitCode := run([]string{
		"send",
		"--to", "recipient@example.com",
	}, runtimeDeps{
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
		getenv: func(string) string {
			return "test-api-key"
		},
		readFile: func(string) ([]byte, error) {
			return nil, nil
		},
		promptSend: func(sendOptions, <-chan struct{}) (sendOptions, error) {
			return sendOptions{}, errInterrupted
		},
		newSender: func(string, time.Duration) emailSender {
			t.Fatal("sender should not be called")
			return nil
		},
	})

	if exitCode != 130 {
		t.Fatalf("expected exit code 130, got %d", exitCode)
	}
}

func TestRunSendRequiresAPIKey(t *testing.T) {
	t.Parallel()

	var stderr strings.Builder

	exitCode := run([]string{
		"send",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
		"--subject", "hello",
		"--text", "plain",
	}, runtimeDeps{
		stdout:   &strings.Builder{},
		stderr:   &stderr,
		getenv:   func(string) string { return "" },
		readFile: func(string) ([]byte, error) { return nil, nil },
		newSender: func(string, time.Duration) emailSender {
			t.Fatal("sender should not be called")
			return nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), envResendAPIKey+" is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestAddressListFlagRejectsEmptyItem(t *testing.T) {
	t.Parallel()

	var f addressListFlag
	err := f.Set("first@example.com, ")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNeedsInteractiveMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts sendOptions
		want bool
	}{
		{
			name: "complete options",
			opts: sendOptions{
				From:    "sender@example.com",
				To:      addressListFlag{values: []string{"recipient@example.com"}},
				Subject: "hello",
				Text: sourceValue{
					Inline: singleStringFlag{name: "--text", value: "body", set: true},
				},
			},
			want: false,
		},
		{
			name: "missing from",
			opts: sendOptions{
				To:      addressListFlag{values: []string{"recipient@example.com"}},
				Subject: "hello",
				Text: sourceValue{
					Inline: singleStringFlag{name: "--text", value: "body", set: true},
				},
			},
			want: true,
		},
		{
			name: "missing to",
			opts: sendOptions{
				From:    "sender@example.com",
				Subject: "hello",
				Text: sourceValue{
					Inline: singleStringFlag{name: "--text", value: "body", set: true},
				},
			},
			want: true,
		},
		{
			name: "missing subject",
			opts: sendOptions{
				From: "sender@example.com",
				To:   addressListFlag{values: []string{"recipient@example.com"}},
				Text: sourceValue{
					Inline: singleStringFlag{name: "--text", value: "body", set: true},
				},
			},
			want: true,
		},
		{
			name: "missing body",
			opts: sendOptions{
				From:    "sender@example.com",
				To:      addressListFlag{values: []string{"recipient@example.com"}},
				Subject: "hello",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		if got := needsInteractiveMode(tt.opts); got != tt.want {
			t.Fatalf("%s: expected %t, got %t", tt.name, tt.want, got)
		}
	}
}

type sendFunc func(ctx context.Context, req SendRequest) (SendResponse, error)

func (f sendFunc) Send(ctx context.Context, req SendRequest) (SendResponse, error) {
	return f(ctx, req)
}
