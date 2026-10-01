package prompt

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenEditor 用系统编辑器编辑邮件正文，返回编辑后的最终内容。
// initial 作为起始内容写入临时文件（可为空）；临时文件在函数退出时删除。
func OpenEditor(initial string) (string, error) {
	f, err := os.CreateTemp("", "resendmail-*.txt")
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if initial != "" {
		if _, err := f.WriteString(initial); err != nil {
			f.Close()
			return "", fmt.Errorf("write file failed: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close file failed: %w", err)
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		switch runtime.GOOS {
		case "windows":
			editor = "notepad"
		default:
			editor = "vim"
		}
	}

	cmd := constructEditorCmd(editor, tmp)
	cmd.Stdin = os.Stdin // 把终端交给编辑器
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run editor %q: %w", editor, err)
	}

	body, err := os.ReadFile(tmp)
	if err != nil {
		return "", fmt.Errorf("read edited content: %w", err)
	}
	return string(body), nil
}

func constructEditorCmd(editor, file string) *exec.Cmd {
	if _, err := exec.LookPath(editor); err == nil {
		return exec.Command(editor, file)
	}
	parts := strings.Fields(editor)
	return exec.Command(parts[0], append(parts[1:], file)...)
}
