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

type sendFunc func(ctx context.Context, req SendRequest) (SendResponse, error)

func (f sendFunc) Send(ctx context.Context, req SendRequest) (SendResponse, error) {
	return f(ctx, req)
}
