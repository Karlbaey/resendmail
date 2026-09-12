package config

import "os"

var ResendAPIKey string

func init() {
	ResendAPIKey = os.Getenv("RESEND_API_KEY")
}
