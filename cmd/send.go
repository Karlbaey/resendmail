package cmd

import (
	"fmt"
	"os"
	"resendmail/internal/send"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send emails",
	Long: `Send email through Resend API. Easily decide options
using flags. More info check Resend API Docs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := constructFlags()
		if err != nil {
			return err
		}
		result, err := send.Send(cmd.Context(), os.Getenv("RESEND_API_KEY"), email)
		if err != nil {
			return err
		}
		fmt.Printf("Success. Email ID: %s\nCheck email on web: https://resend.com/emails/%s\n", result.ID, result.ID)
		return nil
	},
}

var sendFlags struct {
	subject  string
	from     string
	to       []string
	text     string
	html     string
	textFile string
	htmlFile string
	attach   []string
	bcc      []string
	cc       []string
}

func init() {
	sf := &sendFlags
	sendCmd.Flags().StringVarP(&sf.subject, "subject", "s", "", "发件主题（必填）")
	sendCmd.Flags().StringVarP(&sf.from, "from", "f", "", "发件人（必填）")
	sendCmd.Flags().StringArrayVarP(&sf.to, "to", "t", []string{}, "收件人（必填，可多次使用 flag）")
	sendCmd.Flags().StringVar(&sf.text, "text", "", "纯文本邮件内容（与 HTML 二选一）")
	sendCmd.Flags().StringVar(&sf.html, "html", "", "富文本邮件内容（与 Text 二选一）")
	sendCmd.Flags().StringVar(&sf.textFile, "text-file", "", "纯文本邮件内容（从文件读取）")
	sendCmd.Flags().StringVar(&sf.htmlFile, "html-file", "", "富文本邮件内容（从文件读取）")
	sendCmd.Flags().StringArrayVarP(&sf.attach, "attach", "a", []string{}, "附件（填文件路径，可多次使用 flag）")
	sendCmd.Flags().StringArrayVar(&sf.bcc, "bcc", []string{}, "密送（可多次使用 flag）")
	sendCmd.Flags().StringArrayVar(&sf.cc, "cc", []string{}, "抄送（可多次使用 flag）")

	_ = sendCmd.MarkFlagRequired("subject")
	_ = sendCmd.MarkFlagRequired("from")
	_ = sendCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(sendCmd)
}

func constructFlags() (*send.EmailPayload, error) {
	if sendFlags.htmlFile != "" && sendFlags.html != "" {
		return nil, fmt.Errorf("--html and --html-file are mutually exclusive")
	}
	if sendFlags.textFile != "" && sendFlags.text != "" {
		return nil, fmt.Errorf("--text and --text-file are mutually exclusive")
	}
	html, err := loadContent(sendFlags.html, sendFlags.htmlFile)
	if err != nil {
		return nil, err
	}
	text, err := loadContent(sendFlags.text, sendFlags.textFile)
	if err != nil {
		return nil, err
	}
	var attach []send.Attachment
	for _, path := range sendFlags.attach {
		dat, err := send.LoadLocalAttachment(path)
		if err != nil {
			return nil, err
		}
		attach = append(attach, dat)
	}

	return &send.EmailPayload{
		Subject:     sendFlags.subject,
		From:        sendFlags.from,
		To:          sendFlags.to,
		HTML:        html,
		Text:        text,
		Attachments: attach,
		BCC:         sendFlags.bcc,
		CC:          sendFlags.cc,
	}, nil
}

func loadContent(inline, path string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if path == "" {
		return "", nil
	}
	dat, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read content file %q: %w", path, err)
	}
	return string(dat), nil
}
