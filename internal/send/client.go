package send

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	apiKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  http.DefaultClient,
		apiKey:  apiKey,
	}
}

func (c *Client) SendEmail(ctx context.Context, payload *EmailPayload) (*EmailResponseSuccess, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emails", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "Resendmail/1.0.0-alpha.1")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response data failed: %w", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < 300 {
		success := &EmailResponseSuccess{}
		if err := json.Unmarshal(result, success); err != nil {
			return nil, fmt.Errorf("parse success json failed: %w", err)
		}
		return success, nil
	}
	failed := &EmailResponseError{}
	if err := json.Unmarshal(result, failed); err != nil {
		return nil, fmt.Errorf("resend API returned HTTP %d with invalid error body: %w", resp.StatusCode, err)
	}
	return nil, failed
}
