package cmd

import (
	"resendmail/internal/send"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send emails",
	Long: `Send email through Resend API. Easily decide options
	using flags or interactive prompts. More info check `,
	RunE: send.SendEmail,
}

func init() {
	var subject, from, to, text, html string

	sendCmd.Flags().StringVarP(&subject, "subject", "s", "", "发件主题（必填）")
	sendCmd.Flags().StringVarP(&from, "from", "f", "", "发件人（必填）")
	sendCmd.Flags().StringVarP(&to, "to", "t", "", "收件人（必填）")
	sendCmd.Flags().StringVar(&text, "text", "", "纯文本邮件内容（与 HTML 二选一）")
	sendCmd.Flags().StringVar(&html, "html", "", "富文本邮件内容（与 Text 二选一）")

	_ = sendCmd.MarkFlagRequired("subject")
	_ = sendCmd.MarkFlagRequired("from")
	_ = sendCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(sendCmd)
}
