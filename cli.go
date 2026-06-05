package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

const (
	commandSend      = "send"
	envResendAPIKey  = "RESEND_API_KEY"
	defaultTimeout   = 10 * time.Second
	defaultResendURL = "https://api.resend.com"
)

type runtimeDeps struct {
	stdout     io.Writer
	stderr     io.Writer
	getenv     func(string) string
	readFile   func(string) ([]byte, error)
	newSender  func(apiKey string, timeout time.Duration) emailSender
	promptSend func(sendOptions) (sendOptions, error)
}

type emailSender interface {
	Send(ctx context.Context, req SendRequest) (SendResponse, error)
}

type sendOptions struct {
	From    string
	To      addressListFlag
	Cc      addressListFlag
	Subject string
	Text    sourceValue
	HTML    sourceValue
	Timeout time.Duration
}

type sourceValue struct {
	Inline singleStringFlag
	File   singleStringFlag
}

type addressListFlag struct {
	values []string
}

type singleStringFlag struct {
	name  string
	value string
	set   bool
}

func defaultRuntimeDeps(stdout, stderr io.Writer) runtimeDeps {
	return runtimeDeps{
		stdout:   stdout,
		stderr:   stderr,
		getenv:   os.Getenv,
		readFile: os.ReadFile,
		newSender: func(apiKey string, timeout time.Duration) emailSender {
			return &Client{
				BaseURL:    defaultResendURL,
				APIKey:     apiKey,
				HTTPClient: &http.Client{Timeout: timeout},
			}
		},
		promptSend: func(sendOptions) (sendOptions, error) {
			return sendOptions{}, errors.New("interactive mode is not implemented")
		},
	}
}

func run(args []string, deps runtimeDeps) int {
	if len(args) == 0 {
		writeRootUsage(deps.stderr)
		_, _ = fmt.Fprintln(deps.stderr, "error: missing command")
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		writeRootUsage(deps.stdout)
		return 0
	case commandSend:
		return runSend(args[1:], deps)
	default:
		writeRootUsage(deps.stderr)
		_, _ = fmt.Fprintf(deps.stderr, "error: unknown command %q\n", args[0])
		return 2
	}
}

func runSend(args []string, deps runtimeDeps) int {
	opts, helpRequested, err := parseSendArgs(args)
	if err != nil {
		writeSendUsage(deps.stderr)
		_, _ = fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 2
	}
	if helpRequested {
		writeSendUsage(deps.stdout)
		return 0
	}

	usedInteractiveMode := needsInteractiveMode(opts)
	if usedInteractiveMode {
		if deps.promptSend == nil {
			_, _ = fmt.Fprintln(deps.stderr, "error: interactive mode is not available")
			return 1
		}

		opts, err = deps.promptSend(opts)
		if err != nil {
			_, _ = fmt.Fprintf(deps.stderr, "error: %v\n", err)
			return 1
		}
	}

	if err := validateSendOptions(opts); err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "error: %v\n", err)
		if usedInteractiveMode {
			return 1
		}
		return 2
	}

	req, err := buildSendRequest(opts, deps.readFile)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 2
	}

	apiKey := strings.TrimSpace(deps.getenv(envResendAPIKey))
	if apiKey == "" {
		_, _ = fmt.Fprintf(deps.stderr, "error: %s is required\n", envResendAPIKey)
		return 1
	}

	resp, err := deps.newSender(apiKey, opts.Timeout).Send(context.Background(), req)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	enc := json.NewEncoder(deps.stdout)
	if err := enc.Encode(resp); err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "error: encode response: %v\n", err)
		return 1
	}

	return 0
}

func parseSendArgs(args []string) (sendOptions, bool, error) {
	var opts sendOptions
	var helpRequested bool

	fs := flag.NewFlagSet(commandSend, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	opts.Text.Inline = singleStringFlag{name: "--text"}
	opts.Text.File = singleStringFlag{name: "--text-file"}
	opts.HTML.Inline = singleStringFlag{name: "--html"}
	opts.HTML.File = singleStringFlag{name: "--html-file"}

	fs.StringVar(&opts.From, "from", "", "sender email address")
	fs.Var(&opts.To, "to", "recipient email address; repeat or use commas")
	fs.Var(&opts.Cc, "cc", "cc email address; repeat or use commas")
	fs.StringVar(&opts.Subject, "subject", "", "email subject")
	fs.Var(&opts.Text.Inline, "text", "plain text email body")
	fs.Var(&opts.Text.File, "text-file", "path to plain text email body file")
	fs.Var(&opts.HTML.Inline, "html", "HTML email body")
	fs.Var(&opts.HTML.File, "html-file", "path to HTML email body file")
	fs.DurationVar(&opts.Timeout, "timeout", defaultTimeout, "request timeout, e.g. 10s or 30s")
	fs.BoolFunc("help", "show help", func(string) error {
		helpRequested = true
		return nil
	})
	fs.BoolFunc("h", "show help", func(string) error {
		helpRequested = true
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return sendOptions{}, false, err
	}
	if helpRequested {
		return sendOptions{}, true, nil
	}
	if len(fs.Args()) > 0 {
		return sendOptions{}, false, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	if err := validateParsedSendOptions(opts); err != nil {
		return sendOptions{}, false, err
	}

	return opts, false, nil
}

func validateParsedSendOptions(opts sendOptions) error {
	if opts.Timeout <= 0 {
		return errors.New("--timeout must be greater than 0")
	}
	if opts.Text.Inline.set && opts.Text.File.set {
		return errors.New("--text and --text-file are mutually exclusive")
	}
	if opts.HTML.Inline.set && opts.HTML.File.set {
		return errors.New("--html and --html-file are mutually exclusive")
	}

	from := strings.TrimSpace(opts.From)
	if from != "" {
		if err := validateAddress("--from", from); err != nil {
			return err
		}
	}
	if err := validateAddressList("--to", opts.To.values); err != nil {
		return err
	}
	if err := validateAddressList("--cc", opts.Cc.values); err != nil {
		return err
	}

	return nil
}

func validateSendOptions(opts sendOptions) error {
	if err := validateParsedSendOptions(opts); err != nil {
		return err
	}
	if strings.TrimSpace(opts.From) == "" {
		return errors.New("--from is required")
	}
	if len(opts.To.values) == 0 {
		return errors.New("at least one --to is required")
	}
	if strings.TrimSpace(opts.Subject) == "" {
		return errors.New("--subject is required")
	}
	if !hasBodySource(opts) {
		return errors.New("at least one body source is required")
	}

	return nil
}

func needsInteractiveMode(opts sendOptions) bool {
	return strings.TrimSpace(opts.From) == "" || len(opts.To.values) == 0 || strings.TrimSpace(opts.Subject) == "" || !hasBodySource(opts)
}

func hasBodySource(opts sendOptions) bool {
	return opts.Text.Inline.set || opts.Text.File.set || opts.HTML.Inline.set || opts.HTML.File.set
}

func buildSendRequest(opts sendOptions, readFile func(string) ([]byte, error)) (SendRequest, error) {
	text, err := resolveBody("text", opts.Text, readFile)
	if err != nil {
		return SendRequest{}, err
	}
	html, err := resolveBody("html", opts.HTML, readFile)
	if err != nil {
		return SendRequest{}, err
	}

	req := SendRequest{
		From:    strings.TrimSpace(opts.From),
		To:      opts.To.clone(),
		Cc:      opts.Cc.clone(),
		Subject: opts.Subject,
		Text:    text,
		HTML:    html,
	}

	return req, nil
}

func resolveBody(kind string, value sourceValue, readFile func(string) ([]byte, error)) (*string, error) {
	switch {
	case value.Inline.set:
		content := value.Inline.value
		return &content, nil
	case value.File.set:
		content, err := readFile(value.File.value)
		if err != nil {
			return nil, fmt.Errorf("read --%s-file: %w", kind, err)
		}
		body := string(content)
		return &body, nil
	default:
		return nil, nil
	}
}

func validateAddress(flagName, value string) error {
	if _, err := mail.ParseAddress(value); err != nil {
		return fmt.Errorf("%s is not a valid email address: %w", flagName, err)
	}
	return nil
}

func validateAddressList(flagName string, values []string) error {
	for _, value := range values {
		if err := validateAddress(flagName, value); err != nil {
			return err
		}
	}
	return nil
}

func writeRootUsage(w io.Writer) {
	_, _ = io.WriteString(w, "Usage:\n  resendmail send [flags]\n\nRun `resendmail send --help` for command details.\n")
}

func writeSendUsage(w io.Writer) {
	_, _ = io.WriteString(w, "Usage:\n  resendmail send --from <email> --to <email> --subject <text> [flags]\n\nFlags:\n  --from <email>         sender email address\n  --to <email>           recipient email address; repeat or use commas\n  --cc <email>           cc email address; repeat or use commas\n  --subject <text>       email subject\n  --text <body>          plain text email body\n  --text-file <path>     path to plain text email body file\n  --html <body>          HTML email body\n  --html-file <path>     path to HTML email body file\n  --timeout <duration>   request timeout, default 10s\n  --help, -h             show help\n")
}

func (f *addressListFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *addressListFlag) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value cannot be empty")
	}

	for _, part := range strings.Split(value, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			return errors.New("value cannot contain an empty address")
		}
		f.values = append(f.values, item)
	}

	return nil
}

func (f *addressListFlag) clone() []string {
	if len(f.values) == 0 {
		return nil
	}

	values := make([]string, len(f.values))
	copy(values, f.values)
	return values
}

func (f *singleStringFlag) String() string {
	return f.value
}

func (f *singleStringFlag) Set(value string) error {
	if f.set {
		return fmt.Errorf("%s may only be provided once", f.name)
	}
	f.value = value
	f.set = true
	return nil
}
