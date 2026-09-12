package send

import (
	"fmt"
	"resendmail/internal/constants"
	"strings"

	"github.com/spf13/cobra"
)

func SendEmail(cmd *cobra.Command, args []string) error {
	email, err := constructFlags(cmd)
	if err != nil {
		return err
	}
	if err = email.Validate(); err != nil {
		return err
	}
	client := NewClient(constants.ResendAPIBase)

	result, err := client.SendEmail(cmd.Context(), email)
	if err != nil {
		return err
	}

	fmt.Printf("Success. Email ID: %s\nCheck email on web：https://resend.com/emails/%s", result.ID, result.ID)

	return nil
}

func constructFlags(cmd *cobra.Command) (*EmailPayload, error) {
	subject, err := cmd.Flags().GetString("subject")
	if err != nil {
		return nil, fmt.Errorf("get subject failed: %w", err)
	}

	from, err := cmd.Flags().GetString("from")
	if err != nil {
		return nil, fmt.Errorf("get sender failed: %w", err)
	}

	toParts, err := cmd.Flags().GetString("to")
	if err != nil {
		return nil, fmt.Errorf("get receiver(s) failed: %w", err)
	}
	s := strings.Split(toParts, ",")
	var to []string
	for _, v := range s {
		v = strings.TrimSpace(v)
		if v != "" {
			to = append(to, v)
		}
	}

	text, err := cmd.Flags().GetString("text")
	if err != nil {
		return nil, fmt.Errorf("get text email content failed: %w", err)
	}

	html, err := cmd.Flags().GetString("html")
	if err != nil {
		return nil, fmt.Errorf("get html email content failed: %w", err)
	}

	return &EmailPayload{From: from, To: to, Subject: subject, Text: text, HTML: html}, nil
}
