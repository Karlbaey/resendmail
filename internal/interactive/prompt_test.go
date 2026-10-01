package interactive

import (
	"strings"
	"testing"

	"resendmail/internal/send"
)

func TestPreviewBody(t *testing.T) {
	tests := []struct {
		name string
		em   *send.EmailPayload
		want string
	}{
		{
			name: "short text body is shown fully",
			em:   &send.EmailPayload{Text: "你好世界"},
			want: "正文内容:你好世界(共 12 字节)",
		},
		{
			name: "long text body is truncated to 50 runes",
			em:   &send.EmailPayload{Text: strings.Repeat("a", 60)},
			want: "正文内容:" + strings.Repeat("a", 50) + "......(共 60 字节)",
		},
		{
			name: "multibyte runes truncate by runes not bytes",
			em:   &send.EmailPayload{Text: strings.Repeat("好", 60)},
			want: "正文内容:" + strings.Repeat("好", 50) + "......(共 180 字节)",
		},
		{
			name: "html body takes precedence over text",
			em:   &send.EmailPayload{Text: "text版", HTML: "html版"},
			want: "正文内容:html版(共 7 字节)",
		},
		{
			name: "newlines are replaced to keep preview on one line",
			em:   &send.EmailPayload{Text: "第一行\n第二行"},
			want: "正文内容:第一行 第二行(共 19 字节)",
		},
		{
			name: "empty body",
			em:   &send.EmailPayload{},
			want: "正文内容:(共 0 字节)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := previewBody(tt.em); got != tt.want {
				t.Errorf("previewBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreviewBodyExactBoundary(t *testing.T) {
	// 恰好 50 个字符不截断、无省略号;51 个字符才截断。
	body50 := strings.Repeat("x", 50)
	if got := previewBody(&send.EmailPayload{Text: body50}); strings.Contains(got, "......") {
		t.Errorf("50-char body should not be truncated: %q", got)
	}
	body51 := strings.Repeat("x", 51)
	got := previewBody(&send.EmailPayload{Text: body51})
	if !strings.Contains(got, "......") {
		t.Errorf("51-char body should be truncated: %q", got)
	}
}
