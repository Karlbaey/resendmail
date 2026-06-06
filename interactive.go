package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"
)

var (
	errInterrupted   = errors.New("interrupted")
	errSendCancelled = errors.New("send cancelled")
)

type interactiveDeps struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	readFile   func(string) ([]byte, error)
	createTemp func() (string, error)
	removeFile func(string) error
	openEditor func(string, <-chan struct{}) error
}

type lineReader struct {
	lines chan string
	errs  chan error
}

func promptSendInteractive(opts sendOptions, deps interactiveDeps, interrupt <-chan struct{}) (sendOptions, error) {
	deps = withInteractiveDefaults(deps)
	if err := checkInterrupted(interrupt); err != nil {
		return sendOptions{}, err
	}

	reader := newLineReader(deps.stdin)

	if strings.TrimSpace(opts.From) == "" {
		value, err := promptRequiredAddress(reader, deps, interrupt, "请输入发件人邮箱：", "--from")
		if err != nil {
			return sendOptions{}, err
		}
		opts.From = value
	}

	if len(opts.To.values) == 0 {
		values, err := promptAddressList(reader, deps, interrupt, "请输入收件人邮箱（多个地址用逗号分隔）：", "--to", false)
		if err != nil {
			return sendOptions{}, err
		}
		opts.To.values = values
	}

	if len(opts.Cc.values) == 0 {
		values, err := promptAddressList(reader, deps, interrupt, "请输入抄送邮箱（多个地址用逗号分隔，直接回车跳过）：", "--cc", true)
		if err != nil {
			return sendOptions{}, err
		}
		opts.Cc.values = values
	}

	if strings.TrimSpace(opts.Subject) == "" {
		subject, err := promptSubject(reader, deps, interrupt)
		if err != nil {
			return sendOptions{}, err
		}
		opts.Subject = subject
	}

	if !hasBodySource(opts) {
		bodyOpts, err := promptBodySource(reader, deps, interrupt)
		if err != nil {
			return sendOptions{}, err
		}
		opts.Text = bodyOpts.Text
		opts.HTML = bodyOpts.HTML
	}

	if err := confirmSend(reader, deps, interrupt, opts); err != nil {
		return sendOptions{}, err
	}

	return opts, nil
}

func withInteractiveDefaults(deps interactiveDeps) interactiveDeps {
	if deps.stdin == nil {
		deps.stdin = strings.NewReader("")
	}
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}
	if deps.stderr == nil {
		deps.stderr = io.Discard
	}
	if deps.readFile == nil {
		deps.readFile = os.ReadFile
	}
	if deps.createTemp == nil {
		deps.createTemp = defaultCreateTempFile
	}
	if deps.removeFile == nil {
		deps.removeFile = os.Remove
	}
	if deps.openEditor == nil {
		deps.openEditor = openSystemEditor
	}
	return deps
}

func newLineReader(input io.Reader) *lineReader {
	r := &lineReader{
		lines: make(chan string),
		errs:  make(chan error, 1),
	}

	go func() {
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			r.lines <- scanner.Text()
		}
		r.errs <- scanner.Err()
		close(r.lines)
		close(r.errs)
	}()

	return r
}

func (r *lineReader) readLine(interrupt <-chan struct{}) (string, error) {
	if err := checkInterrupted(interrupt); err != nil {
		return "", err
	}

	select {
	case <-interrupt:
		return "", errInterrupted
	case line, ok := <-r.lines:
		if ok {
			return line, nil
		}
		err, ok := <-r.errs
		if ok && err != nil {
			return "", err
		}
		return "", io.EOF
	}
}

func promptRequiredAddress(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, prompt, flagName string) (string, error) {
	for {
		value, err := promptLine(reader, deps.stdout, interrupt, prompt)
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			writeInteractiveError(deps.stderr, fmt.Errorf("%s is required", strings.TrimPrefix(flagName, "--")))
			continue
		}
		if err := validateAddress(flagName, value); err != nil {
			writeInteractiveError(deps.stderr, err)
			continue
		}
		return value, nil
	}
}

func promptAddressList(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, prompt, flagName string, allowEmpty bool) ([]string, error) {
	for {
		value, err := promptLine(reader, deps.stdout, interrupt, prompt)
		if err != nil {
			return nil, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			if allowEmpty {
				return nil, nil
			}
			writeInteractiveError(deps.stderr, fmt.Errorf("%s is required", strings.TrimPrefix(flagName, "--")))
			continue
		}

		var parsed addressListFlag
		if err := parsed.Set(value); err != nil {
			writeInteractiveError(deps.stderr, err)
			continue
		}
		if err := validateAddressList(flagName, parsed.values); err != nil {
			writeInteractiveError(deps.stderr, err)
			continue
		}

		return parsed.values, nil
	}
}

func promptSubject(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}) (string, error) {
	for {
		value, err := promptLine(reader, deps.stdout, interrupt, "请输入邮件主题：")
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			writeInteractiveError(deps.stderr, errors.New("subject is required"))
			continue
		}
		return value, nil
	}
}

func promptBodySource(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}) (sendOptions, error) {
	format, err := promptChoice(reader, deps, interrupt, "请选择正文格式：\n  1) 纯文本\n  2) HTML\n请输入选项 (1 或 2)：")
	if err != nil {
		return sendOptions{}, err
	}

	method, err := promptChoice(reader, deps, interrupt, "请选择输入方式：\n  1) 使用编辑器输入\n  2) 从文件读取\n请输入选项 (1 或 2)：")
	if err != nil {
		return sendOptions{}, err
	}

	if method == "1" {
		body, err := promptBodyFromEditor(reader, deps, interrupt, format)
		if err != nil {
			return sendOptions{}, err
		}
		return body, nil
	}

	return promptBodyFromFile(reader, deps, interrupt, format)
}

func promptBodyFromEditor(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, format string) (sendOptions, error) {
	for {
		path, err := deps.createTemp()
		if err != nil {
			return sendOptions{}, fmt.Errorf("create temporary file: %w", err)
		}

		body, editorErr := readBodyFromEditor(path, deps, interrupt)
		if editorErr == nil {
			return buildBodyOptions(format, "inline", body), nil
		}

		writeInteractiveError(deps.stderr, editorErr)

		useFile, err := promptYesNo(reader, deps, interrupt, "是否改为从文件读取？(y/n)")
		if err != nil {
			return sendOptions{}, err
		}
		if useFile {
			return promptBodyFromFile(reader, deps, interrupt, format)
		}
	}
}

func readBodyFromEditor(path string, deps interactiveDeps, interrupt <-chan struct{}) (string, error) {
	if err := deps.openEditor(path, interrupt); err != nil {
		if removeErr := deps.removeFile(path); removeErr != nil {
			return "", fmt.Errorf("%w (cleanup failed: %v)", err, removeErr)
		}
		return "", err
	}

	content, err := deps.readFile(path)
	removeErr := deps.removeFile(path)
	if err != nil {
		if removeErr != nil {
			return "", fmt.Errorf("read temporary file: %w (cleanup failed: %v)", err, removeErr)
		}
		return "", fmt.Errorf("read temporary file: %w", err)
	}
	if removeErr != nil {
		return "", fmt.Errorf("remove temporary file: %w", removeErr)
	}
	if len(content) == 0 {
		return "", errors.New("editor returned empty content")
	}
	return string(content), nil
}

func promptBodyFromFile(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, format string) (sendOptions, error) {
	for {
		path, err := promptLine(reader, deps.stdout, interrupt, "请输入文件路径：")
		if err != nil {
			return sendOptions{}, err
		}
		path = strings.TrimSpace(path)
		if path == "" {
			writeInteractiveError(deps.stderr, errors.New("file path is required"))
			continue
		}

		if _, err := deps.readFile(path); err != nil {
			writeInteractiveError(deps.stderr, fmt.Errorf("read file: %w", err))
			continue
		}

		return buildBodyOptions(format, "file", path), nil
	}
}

func buildBodyOptions(format, sourceKind, value string) sendOptions {
	inline := sourceKind == "inline"
	textFlag := singleStringFlag{name: "--text", value: value, set: inline}
	textFileFlag := singleStringFlag{name: "--text-file", value: value, set: !inline}
	htmlFlag := singleStringFlag{name: "--html", value: value, set: inline}
	htmlFileFlag := singleStringFlag{name: "--html-file", value: value, set: !inline}

	if format == "2" {
		return sendOptions{
			HTML: sourceValue{
				Inline: htmlFlag,
				File:   htmlFileFlag,
			},
		}
	}

	return sendOptions{
		Text: sourceValue{
			Inline: textFlag,
			File:   textFileFlag,
		},
	}
}

func confirmSend(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, opts sendOptions) error {
	formatLabel, preview, err := buildBodyPreview(opts, deps.readFile)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(deps.stdout, "=== 邮件信息确认 ===")
	_, _ = fmt.Fprintf(deps.stdout, "发件人: %s\n", strings.TrimSpace(opts.From))
	_, _ = fmt.Fprintf(deps.stdout, "收件人: %s\n", strings.Join(opts.To.values, ", "))
	_, _ = fmt.Fprintf(deps.stdout, "抄送: %s\n", strings.Join(opts.Cc.values, ", "))
	_, _ = fmt.Fprintf(deps.stdout, "主题: %s\n", opts.Subject)
	_, _ = fmt.Fprintf(deps.stdout, "正文格式: %s\n", formatLabel)
	_, _ = fmt.Fprintf(deps.stdout, "正文预览: %s\n\n", preview)

	confirmed, err := promptYesNo(reader, deps, interrupt, "确认发送？(y/n):")
	if err != nil {
		return err
	}
	if !confirmed {
		return errSendCancelled
	}
	return nil
}

func buildBodyPreview(opts sendOptions, readFile func(string) ([]byte, error)) (string, string, error) {
	label := bodyFormatLabel(opts)
	body, err := loadPreviewBody(opts, readFile)
	if err != nil {
		return "", "", err
	}

	count := utf8.RuneCountInString(body)
	preview := firstNRunes(body, 100)
	if count > 100 {
		preview += "..."
	}
	return label, fmt.Sprintf("%s (共 %d 字符)", preview, count), nil
}

func bodyFormatLabel(opts sendOptions) string {
	hasText := opts.Text.Inline.set || opts.Text.File.set
	hasHTML := opts.HTML.Inline.set || opts.HTML.File.set
	switch {
	case hasText && hasHTML:
		return "纯文本 + HTML"
	case hasHTML:
		return "HTML"
	default:
		return "纯文本"
	}
}

func loadPreviewBody(opts sendOptions, readFile func(string) ([]byte, error)) (string, error) {
	switch {
	case opts.Text.Inline.set:
		return opts.Text.Inline.value, nil
	case opts.Text.File.set:
		content, err := readFile(opts.Text.File.value)
		if err != nil {
			return "", fmt.Errorf("read --text-file: %w", err)
		}
		return string(content), nil
	case opts.HTML.Inline.set:
		return opts.HTML.Inline.value, nil
	case opts.HTML.File.set:
		content, err := readFile(opts.HTML.File.value)
		if err != nil {
			return "", fmt.Errorf("read --html-file: %w", err)
		}
		return string(content), nil
	default:
		return "", errors.New("at least one body source is required")
	}
}

func firstNRunes(value string, n int) string {
	if n <= 0 {
		return ""
	}

	count := 0
	for i := range value {
		if count == n {
			return value[:i]
		}
		count++
	}
	return value
}

func promptChoice(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, prompt string) (string, error) {
	for {
		value, err := promptLine(reader, deps.stdout, interrupt, prompt)
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "1" || value == "2" {
			return value, nil
		}
		writeInteractiveError(deps.stderr, errors.New("请输入选项 (1 或 2)"))
	}
}

func promptYesNo(reader *lineReader, deps interactiveDeps, interrupt <-chan struct{}, prompt string) (bool, error) {
	for {
		value, err := promptLine(reader, deps.stdout, interrupt, prompt+" ")
		if err != nil {
			return false, err
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(value) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			writeInteractiveError(deps.stderr, errors.New("请输入 y 或 n"))
		}
	}
}

func promptLine(reader *lineReader, stdout io.Writer, interrupt <-chan struct{}, prompt string) (string, error) {
	if _, err := io.WriteString(stdout, prompt); err != nil {
		return "", err
	}
	line, err := reader.readLine(interrupt)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", errors.New("unexpected end of input")
		}
		return "", err
	}
	return line, nil
}

func writeInteractiveError(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
}

func checkInterrupted(interrupt <-chan struct{}) error {
	if interrupt == nil {
		return nil
	}
	select {
	case <-interrupt:
		return errInterrupted
	default:
		return nil
	}
}

func defaultCreateTempFile() (string, error) {
	file, err := os.CreateTemp("", "resendmail-*.tmp")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func openSystemEditor(path string, interrupt <-chan struct{}) error {
	editor := "vi"
	if runtime.GOOS == "windows" {
		editor = "notepad.exe"
	} else if visual := strings.TrimSpace(os.Getenv("VISUAL")); visual != "" {
		editor = visual
	} else if configured := strings.TrimSpace(os.Getenv("EDITOR")); configured != "" {
		editor = configured
	}

	cmd := exec.Command(editor, path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open editor: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if interrupt == nil {
		if err := <-done; err != nil {
			return fmt.Errorf("wait for editor: %w", err)
		}
		return nil
	}

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("wait for editor: %w", err)
		}
		return nil
	case <-interrupt:
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return errInterrupted
	}
}
