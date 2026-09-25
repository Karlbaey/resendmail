package send

import (
	"errors"
	"fmt"
)

type EmailPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
}

type EmailResponseSuccess struct {
	ID string `json:"id"`
}

type EmailResponseError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Name       string `json:"name"`
}

func (e *EmailPayload) Validate() error {
	switch {
	case e.From == "":
		return errors.New("sender is required")
	case len(e.To) == 0:
		return errors.New("receiver(s) is required")
	case e.Subject == "":
		return errors.New("subject is required")
	case e.Text == "" && e.HTML == "":
		return errors.New("text and html cannot both be empty: provide exactly one of them")
	case e.Text != "" && e.HTML != "":
		return errors.New("text and html are mutually exclusive: provide exactly one of them")
	}
	return nil
}

func (e *EmailResponseError) Error() string {
	return fmt.Sprintf("Resend API error [%s] (HTTP %d): %s", e.Name, e.StatusCode, e.Message)
}
