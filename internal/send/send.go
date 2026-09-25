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

	return client.SendEmail(ctx, email)
}
