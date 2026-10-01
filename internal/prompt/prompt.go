package prompt

import (
	"errors"
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
)

var ErrAborted = errors.New("prompt aborted")

func IsAborted(err error) bool { return errors.Is(err, ErrAborted) }

func Input(title string, validate func(string) error) (string, error) {
	var v string
	err := huh.NewInput().
		Title(title).
		Validate(validate).
		Value(&v).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", normalizeErr(err)
	}
	return v, nil
}

// Confirm 确认选择，选择后即为真。
func Confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title(title).
		Value(&ok).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, normalizeErr(err)
	}
	return ok, nil
}

// Select 单选列表。options 里放选项文本，返回选中的那一项。
func Select(title string, options []string) (string, error) {
	var out string
	opts := make([]huh.Option[string], 0, len(options))
	for _, o := range options {
		opts = append(opts, huh.NewOption(o, o))
	}

	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&out).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", normalizeErr(err)
	}
	return out, nil
}

// Text 在终端编辑文本，可用 Enter 换行，Ctrl+D（Windows）退出
func Text(title string) (string, error) {
	var v string
	err := huh.NewText().
		Title(title).
		Value(&v).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", normalizeErr(err)
	}
	return v, nil
}

func EnsureTTY() error {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return errors.New("not a terminal")
	}
	return nil
}

func normalizeErr(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrAborted
	}
	return fmt.Errorf("prompt failed: %w", err)
}
