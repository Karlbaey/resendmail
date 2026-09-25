package send

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendEmailSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/emails" {
			t.Errorf("expected path /emails, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("User-Agent header not set")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var payload EmailPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("request body is not valid JSON: %v (body=%s)", err, body)
			return
		}
		if payload.From != "sender@example.com" {
			t.Errorf("From = %q, want sender@example.com", payload.From)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"abc123"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.SendEmail(context.Background(), &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err != nil {
		t.Fatalf("SendEmail() unexpected error: %v", err)
	}
	if result.ID != "abc123" {
		t.Fatalf("result.ID = %q, want abc123", result.ID)
	}
}

func TestSendEmailAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"statusCode":401,"message":"invalid api key","name":"unauthorized_error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	_, err := client.SendEmail(context.Background(), &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected error, got nil")
	}

	var apiErr *EmailResponseError
	if !errors.As(err, &apiErr) {
		t.Fatalf("SendEmail() error type = %T, want *EmailResponseError", err)
	}
	if apiErr.StatusCode != 401 || apiErr.Message != "invalid api key" || apiErr.Name != "unauthorized_error" {
		t.Fatalf("unexpected API error: %+v", apiErr)
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("error %q does not include HTTP status", err.Error())
	}
}

func TestSendEmailInvalidErrorBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.SendEmail(context.Background(), &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid error body") {
		t.Fatalf("error %q does not mention invalid error body", err.Error())
	}
}

func TestSendEmailInvalidSuccessBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.SendEmail(context.Background(), &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse success json failed") {
		t.Fatalf("error %q does not mention parse success json", err.Error())
	}
}

func TestSendEmailServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"statusCode":500,"message":"boom","name":"internal_error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.SendEmail(context.Background(), &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected error, got nil")
	}
	var apiErr *EmailResponseError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *EmailResponseError", err)
	}
	if apiErr.StatusCode != 500 {
		t.Fatalf("apiErr.StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestSendEmailContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	client := NewClient(server.URL, "key")
	_, err := client.SendEmail(ctx, &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected error for cancelled context, got nil")
	}
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	c := NewClient("https://api.example.com", "key")
	if c.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want https://api.example.com", c.baseURL)
	}
	if c.apiKey != "key" {
		t.Errorf("apiKey = %q, want key", c.apiKey)
	}
	if c.client == nil {
		t.Error("client should default to http.DefaultClient")
	}
}

func TestSendEmailTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the response longer than the client's context deadline so the
		// client gives up first. The handler returns after the sleep, which
		// lets httptest.Server.Close shut down cleanly.
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.SendEmail(ctx, &EmailPayload{
		From:    "sender@example.com",
		To:      []string{"receiver@example.com"},
		Subject: "Hello",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("SendEmail() expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendEmail() error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("SendEmail() took %v, expected it to respect context timeout", elapsed)
	}
}