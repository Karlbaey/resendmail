package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient httpDoer
}

type SendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc,omitempty"`
	Subject string   `json:"subject"`
	Text    *string  `json:"text,omitempty"`
	HTML    *string  `json:"html,omitempty"`
}

type SendResponse struct {
	ID string `json:"id"`
}

type resendAPIError struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

func (c *Client) Send(ctx context.Context, req SendRequest) (SendResponse, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = defaultResendURL
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return SendResponse{}, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/emails", bytes.NewReader(body))
	if err != nil {
		return SendResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "resendmail")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return SendResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return SendResponse{}, decodeAPIError(resp)
	}

	var out SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return SendResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		return SendResponse{}, errors.New("resend response missing id")
	}

	return out, nil
}

func decodeAPIError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("resend API returned HTTP %d and the error body could not be read: %w", resp.StatusCode, err)
	}

	var apiErr resendAPIError
	if err := json.Unmarshal(body, &apiErr); err == nil {
		messageParts := []string{fmt.Sprintf("resend API returned HTTP %d", resp.StatusCode)}
		if apiErr.Name != "" {
			messageParts = append(messageParts, apiErr.Name)
		}
		if apiErr.Message != "" {
			messageParts = append(messageParts, apiErr.Message)
		}
		if apiErr.StatusCode != 0 && apiErr.StatusCode != resp.StatusCode {
			messageParts = append(messageParts, fmt.Sprintf("statusCode=%d", apiErr.StatusCode))
		}
		return errors.New(strings.Join(messageParts, ": "))
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("resend API returned HTTP %d", resp.StatusCode)
	}

	return fmt.Errorf("resend API returned HTTP %d: %s", resp.StatusCode, message)
}
