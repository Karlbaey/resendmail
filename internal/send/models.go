package send

import (
	"errors"
	"fmt"
)

type EmailPayload struct {
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text,omitempty"`
	HTML        string       `json:"html,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	CC          []string     `json:"cc,omitempty"`
	BCC         []string     `json:"bcc,omitempty"`
}

type Attachment struct {
	Content     string `json:"content,omitempty"`
	Path        string `json:"path,omitempty"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
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
	return e.validateAttachments()
}

func (e *EmailPayload) validateAttachments() error {
	size := 0
	for i := range e.Attachments {
		a := &e.Attachments[i]
		switch {
		case a.Filename == "":
			return fmt.Errorf("attachment number %d does not exist", i)
		case a.Content == "" && a.Path == "":
			return fmt.Errorf("attachment number %d: content and path cannot both be empty: provide exactly one of them", i)
		case a.Content != "" && a.Path != "":
			return fmt.Errorf("attachment number %d: content and path are mutually exclusive: provide exactly one of them", i)
		}
		size += len(a.Content)
	}
	if size >= 40*1024*1024 {
		return errors.New("size of attachments is over the limit 40MB")
	}
	return nil
}

func (e *EmailResponseError) Error() string {
	return fmt.Sprintf("Resend API error [%s] (HTTP %d): %s", e.Name, e.StatusCode, e.Message)
}
