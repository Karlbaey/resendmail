package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSend(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/emails" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		wantBody := `{"from":"sender@example.com","to":["recipient@example.com"],"subject":"hello","text":"plain body"}`
		if string(body) != wantBody {
			t.Fatalf("unexpected request body: %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}

	resp, err := client.Send(context.Background(), SendRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "hello",
		Text:    stringPointer("plain body"),
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if resp.ID != "email_123" {
		t.Fatalf("unexpected response id: %q", resp.ID)
	}
}

func TestClientSendReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"name":"validation_error","message":"invalid from address","statusCode":422}`)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}

	_, err := client.Send(context.Background(), SendRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "hello",
		Text:    stringPointer("plain body"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 422") || !strings.Contains(err.Error(), "validation_error") || !strings.Contains(err.Error(), "invalid from address") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
