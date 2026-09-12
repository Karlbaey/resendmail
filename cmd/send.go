package cmd

import (
	"fmt"
	"os"
	"resendmail/internal/send"
	"strings"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send emails",
	Long: `Send email through Resend API. Easily decide options
using flags or interactive prompts. More info check Resend API Docs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := send.Send(cmd.Context(), os.Getenv("RESEND_API_KEY"), constructFlags())
		if err != nil {
			return err
		}
		fmt.Printf("Success. Email ID: %s\nCheck email on web: https://resend.com/emails/%s", result.ID, result.ID)
		return nil
	},
}

var sendFlags struct {
	subject string
	from    string
	to      string
	text    string
	html    string
}

func init() {
	sf := &sendFlags
	sendCmd.Flags().StringVarP(&sf.subject, "subject", "s", "", "发件主题（必填）")
	sendCmd.Flags().StringVarP(&sf.from, "from", "f", "", "发件人（必填）")
	sendCmd.Flags().StringVarP(&sf.to, "to", "t", "", "收件人（必填）")
	sendCmd.Flags().StringVar(&sf.text, "text", "", "纯文本邮件内容（与 HTML 二选一）")
	sendCmd.Flags().StringVar(&sf.html, "html", "", "富文本邮件内容（与 Text 二选一）")

	_ = sendCmd.MarkFlagRequired("subject")
	_ = sendCmd.MarkFlagRequired("from")
	_ = sendCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(sendCmd)
}

func constructFlags() *send.EmailPayload {
	return &send.EmailPayload{
		Subject: sendFlags.subject,
		From:    sendFlags.from,
		To:      splitCSV(sendFlags.to),
		HTML:    sendFlags.html,
		Text:    sendFlags.text,
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
