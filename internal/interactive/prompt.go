package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"resendmail/internal/prompt"
	"resendmail/internal/send"
	"strings"
)

var errCancel = errors.New("user canceled")

// Ask 交互式收集一封邮件的全部要素并返回组装好的 payload。
// 返回 nil payload 表示用户取消，调用方应提示“已取消，未发送”后正常退出。
func Ask(ctx context.Context) (*send.EmailPayload, error) {
	payload, err := run(ctx)
	if errors.Is(err, errCancel) {
		return nil, nil
	}
	return payload, err
}

// run interactive 包中最主要的函数，负责收集用户请求并组装 EmailPayload
func run(ctx context.Context) (*send.EmailPayload, error) {
	if err := prompt.EnsureTTY(); err != nil {
		return nil, err
	}

	from, err := prompt.Input("填写发件人", func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("发件人必填")
		}
		return nil
	})
	if err != nil {
		return nil, cancelOrErr(err)
	}

	toRaw, err := prompt.Input("填写收件人，用英文逗号分隔", func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("收件人必填")
		}
		return nil
	})
	if err != nil {
		return nil, cancelOrErr(err)
	}

	ccRaw, err := prompt.Input("填写抄送（可跳过），用英文逗号分隔", func(string) error { return nil })
	if err != nil {
		return nil, cancelOrErr(err)
	}

	bccRaw, err := prompt.Input("填写密送（可跳过），用英文逗号分隔", func(string) error { return nil })
	if err != nil {
		return nil, cancelOrErr(err)
	}

	subject, err := prompt.Input("主题必填", func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("主题必填")
		}
		return nil
	})
	if err != nil {
		return nil, cancelOrErr(err)
	}

	body, isHTML, err := askBody()
	if err != nil {
		return nil, cancelOrErr(err)
	}

	// 附件可选:「继续添加附件吗?」选「否」或中途 Ctrl+C 即结束选择。
	files, err := prompt.PickFile("选择附件（按下 Ctrl+C 跳过）")
	if err != nil {
		return nil, cancelOrErr(err)
	}

	payload := &send.EmailPayload{
		From:    strings.TrimSpace(from),
		To:      splitAndTrim(toRaw),
		CC:      splitAndTrim(ccRaw),
		BCC:     splitAndTrim(bccRaw),
		Subject: strings.TrimSpace(subject),
	}
	setBody(payload, body, isHTML)
	for _, p := range files {
		a, err := send.LoadLocalAttachment(p)
		if err != nil {
			return nil, fmt.Errorf("load attachment %q: %w", p, err)
		}
		payload.Attachments = append(payload.Attachments, a)
	}

	if err := payload.Validate(); err != nil {
		return nil, err
	}

	ok, err := previewAndConfirm(payload)
	if err != nil {
		return nil, cancelOrErr(err)
	}
	if !ok {
		return nil, errCancel
	}
	return payload, nil
}

// askBody 交互式收集正文,返回正文内容和格式标志（是否为 HTML）。
func askBody() (body string, isHTML bool, err error) {
	format, err := prompt.Select("正文格式？", []string{"纯文本", "HTML"})
	if err != nil {
		return "", false, err
	}

	mode, err := prompt.Select("编辑正文方式？", []string{"终端输入", "打开编辑器", "选择文件"})
	if err != nil {
		return "", false, err
	}

	switch mode {
	case "终端输入":
		body, err = prompt.Text("请输入（Enter 换行，Ctrl+D 完成）\n")
	case "打开编辑器":
		body, err = prompt.OpenEditor("在此处编写邮件正文")
	case "选择文件":
		path, err2 := prompt.Input("输入文件路径……", func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("路径不能为空")
			}
			return nil
		})
		if err2 != nil {
			return "", false, err2
		}
		dat, err3 := os.ReadFile(strings.TrimSpace(path))
		if err3 != nil {
			return "", false, fmt.Errorf("read body file: %w", err3)
		}
		body = string(dat)
	default:
		return "", false, fmt.Errorf("unknown body mode: %s", mode)
	}
	if err != nil {
		return "", false, err
	}
	return body, format == "HTML", nil
}

// setBody 按格式标志把正文放入 EmailPayload 的 Text 或 HTML 字段。
func setBody(payload *send.EmailPayload, body string, isHTML bool) {
	if isHTML {
		payload.HTML = body
		return
	}
	payload.Text = body
}

func previewAndConfirm(em *send.EmailPayload) (bool, error) {
	fmt.Println("──────── 邮件预览 ────────")
	fmt.Printf("发件人：%s\n", em.From)
	fmt.Printf("收件人：%s\n", strings.Join(em.To, ", "))
	if len(em.CC) > 0 {
		fmt.Printf("抄送：%s\n", strings.Join(em.CC, ", "))
	}
	if len(em.BCC) > 0 {
		fmt.Printf("密送：%s\n", strings.Join(em.BCC, ", "))
	}
	fmt.Printf("主题：%s\n", em.Subject)
	bodyFormat := "纯文本"
	if em.HTML != "" {
		bodyFormat = "HTML"
	}
	fmt.Printf("正文格式：%s\n", bodyFormat)
	fmt.Println(previewBody(em))
	if len(em.Attachments) > 0 {
		names := make([]string, 0, len(em.Attachments))
		for _, a := range em.Attachments {
			names = append(names, a.Filename)
		}
		fmt.Printf("附件：%s\n", strings.Join(names, ", "))
	} else {
		fmt.Printf("附件：（无）\n")
	}

	ok, err := prompt.Confirm("确认发送?")
	if err != nil {
		return false, err
	}
	return ok, nil
}

// cancelOrErr 把“用户取消”统一映射为 errCanceled，其余原样返回。
// previewBody 生成正文预览行:正文内容:<前 50 字截断>(共 N 字节)。
// 超过 50 个字时以省略号结尾;换行等控制字符替换为空格,保证预览占一行。
func previewBody(em *send.EmailPayload) string {
	const maxRunes = 50
	const ellipsis = "......"

	body := em.Text
	if em.HTML != "" {
		body = em.HTML
	}

	shown := body
	if runes := []rune(body); len(runes) > maxRunes {
		shown = string(runes[:maxRunes]) + ellipsis
	}
	shown = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return ' '
		}
		return r
	}, shown)

	return fmt.Sprintf("正文内容:%s(共 %d 字节)", shown, len(body))
}

func cancelOrErr(err error) error {
	if prompt.IsAborted(err) {
		return errCancel
	}
	return err
}

// splitAndTrim 把含逗号的字符串分割成邮箱地址，并删除空格
func splitAndTrim(s string) (out []string) {
	for _, part := range strings.Split(s, ",") {
		if cut := strings.TrimSpace(part); cut != "" {
			out = append(out, cut)
		}
	}
	return
}
