package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPromptSendInteractiveCollectsMissingFieldsAndConfirms(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder

	opts, err := promptSendInteractive(sendOptions{
		To:      addressListFlag{values: []string{"recipient@example.com"}},
		Timeout: 30 * time.Second,
	}, interactiveDeps{
		stdin:  strings.NewReader("sender@example.com\ncopy@example.com\nhello\n1\n2\nbody.txt\ny\n"),
		stdout: &stdout,
		stderr: &stderr,
		readFile: func(path string) ([]byte, error) {
			if path != "body.txt" {
				t.Fatalf("unexpected read path: %q", path)
			}
			return []byte("This is the email body."), nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("promptSendInteractive returned error: %v", err)
	}

	if opts.From != "sender@example.com" {
		t.Fatalf("unexpected from: %q", opts.From)
	}
	if len(opts.To.values) != 1 || opts.To.values[0] != "recipient@example.com" {
		t.Fatalf("unexpected to values: %#v", opts.To.values)
	}
	if len(opts.Cc.values) != 1 || opts.Cc.values[0] != "copy@example.com" {
		t.Fatalf("unexpected cc values: %#v", opts.Cc.values)
	}
	if opts.Subject != "hello" {
		t.Fatalf("unexpected subject: %q", opts.Subject)
	}
	if !opts.Text.File.set || opts.Text.File.value != "body.txt" {
		t.Fatalf("unexpected text file body: %#v", opts.Text.File)
	}
	if opts.HTML.Inline.set || opts.HTML.File.set {
		t.Fatalf("expected html body to be unset, got %#v", opts.HTML)
	}
	if opts.Timeout != 30*time.Second {
		t.Fatalf("unexpected timeout: %s", opts.Timeout)
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "请输入收件人邮箱") {
		t.Fatalf("did not expect recipient prompt when --to is already provided, got %q", output)
	}
	for _, want := range []string{
		"请输入发件人邮箱：",
		"请输入抄送邮箱（多个地址用逗号分隔，直接回车跳过）：",
		"请输入邮件主题：",
		"请选择正文格式：",
		"请选择输入方式：",
		"请输入文件路径：",
		"=== 邮件信息确认 ===",
		"正文格式: 纯文本",
		"正文预览: This is the email body. (共 23 字符)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestPromptSendInteractiveRepromptsOnInvalidInput(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder

	opts, err := promptSendInteractive(sendOptions{}, interactiveDeps{
		stdin:  strings.NewReader("invalid\nsender@example.com\nrecipient@example.com\n\n\nHello\n3\n1\n3\n2\nbody.txt\ny\n"),
		stdout: &stdout,
		stderr: &stderr,
		readFile: func(path string) ([]byte, error) {
			if path != "body.txt" {
				t.Fatalf("unexpected read path: %q", path)
			}
			return []byte("preview"), nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("promptSendInteractive returned error: %v", err)
	}

	if opts.From != "sender@example.com" {
		t.Fatalf("unexpected from: %q", opts.From)
	}
	if len(opts.To.values) != 1 || opts.To.values[0] != "recipient@example.com" {
		t.Fatalf("unexpected to values: %#v", opts.To.values)
	}
	if opts.Subject != "Hello" {
		t.Fatalf("unexpected subject: %q", opts.Subject)
	}
	if !opts.Text.File.set || opts.Text.File.value != "body.txt" {
		t.Fatalf("unexpected text file body: %#v", opts.Text.File)
	}

	errOutput := stderr.String()
	if strings.Count(errOutput, "error:") < 4 {
		t.Fatalf("expected multiple validation errors, got %q", errOutput)
	}
	for _, want := range []string{
		"--from is not a valid email address",
		"subject is required",
		"请输入选项 (1 或 2)",
	} {
		if !strings.Contains(errOutput, want) {
			t.Fatalf("expected stderr to contain %q, got %q", want, errOutput)
		}
	}
}

func TestPromptSendInteractiveFallsBackToFileWhenEditorFails(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder
	var removedPath string

	opts, err := promptSendInteractive(sendOptions{
		From:    "sender@example.com",
		To:      addressListFlag{values: []string{"recipient@example.com"}},
		Subject: "hello",
	}, interactiveDeps{
		stdin:  strings.NewReader("\n1\n1\ny\nbody.txt\ny\n"),
		stdout: &stdout,
		stderr: &stderr,
		readFile: func(path string) ([]byte, error) {
			if path != "body.txt" {
				t.Fatalf("unexpected read path: %q", path)
			}
			return []byte("body from file"), nil
		},
		createTemp: func() (string, error) {
			return "temp-body.txt", nil
		},
		removeFile: func(path string) error {
			removedPath = path
			return nil
		},
		openEditor: func(path string, interrupt <-chan struct{}) error {
			if path != "temp-body.txt" {
				t.Fatalf("unexpected temp path: %q", path)
			}
			return errors.New("editor unavailable")
		},
	}, nil)
	if err != nil {
		t.Fatalf("promptSendInteractive returned error: %v", err)
	}

	if !opts.Text.File.set || opts.Text.File.value != "body.txt" {
		t.Fatalf("unexpected text file body: %#v", opts.Text.File)
	}
	if removedPath != "temp-body.txt" {
		t.Fatalf("expected temp file to be removed, got %q", removedPath)
	}
	if !strings.Contains(stderr.String(), "editor unavailable") {
		t.Fatalf("expected stderr to mention editor failure, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "是否改为从文件读取？(y/n)") {
		t.Fatalf("expected stdout to contain fallback prompt, got %q", stdout.String())
	}
}

func TestPromptSendInteractiveCancelAtConfirmation(t *testing.T) {
	t.Parallel()

	_, err := promptSendInteractive(sendOptions{
		From:    "sender@example.com",
		To:      addressListFlag{values: []string{"recipient@example.com"}},
		Cc:      addressListFlag{values: []string{"copy@example.com"}},
		Subject: "hello",
		Text: sourceValue{
			Inline: singleStringFlag{name: "--text", value: "plain body", set: true},
		},
	}, interactiveDeps{
		stdin:  strings.NewReader("n\n"),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}, nil)
	if !errors.Is(err, errSendCancelled) {
		t.Fatalf("expected errSendCancelled, got %v", err)
	}
}

func TestPromptSendInteractiveInterrupted(t *testing.T) {
	t.Parallel()

	interrupt := make(chan struct{})
	close(interrupt)

	_, err := promptSendInteractive(sendOptions{
		From:    "sender@example.com",
		To:      addressListFlag{values: []string{"recipient@example.com"}},
		Cc:      addressListFlag{values: []string{"copy@example.com"}},
		Subject: "hello",
		Text: sourceValue{
			Inline: singleStringFlag{name: "--text", value: "plain body", set: true},
		},
	}, interactiveDeps{
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}, interrupt)
	if !errors.Is(err, errInterrupted) {
		t.Fatalf("expected errInterrupted, got %v", err)
	}
}
