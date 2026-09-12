package send

import (
	"context"
	"resendmail/internal/constants"
)

func Send(ctx context.Context, apiKey string, email *EmailPayload) (*EmailResponseSuccess, error) {
	err := email.Validate()
	if err != nil {
		return nil, err
	}
	client := NewClient(constants.ResendAPIBase, apiKey)

	result, err := client.SendEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return result, nil
}
