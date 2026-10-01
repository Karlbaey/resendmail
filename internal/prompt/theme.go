package prompt

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// 本包所有交互组件共用同一个 huh 主题。
//
// 在 Catppuccin(Latte 浅色 / Mocha 深色,随终端自动切换)基础上定制:
//   - 界面主要文字 → 浅蓝色(浅色终端用中等饱和蓝保证对比度,深色终端用亮浅蓝);
//   - 必填项校验失败的错误提示 → 红色。
//
// 想整体换成其他内置主题(如 ThemeDracula / ThemeBase16 / ThemeBase)
// 或自定义配色时,在程序启动或第一次交互前调用 SetTheme 即可。
var theme huh.Theme = huh.ThemeFunc(defaultTheme)

// defaultTheme 以 Catppuccin 为基座,只覆盖文字与错误提示的配色。
func defaultTheme(isDark bool) *huh.Styles {
	lightDark := lipgloss.LightDark(isDark)

	// 主打色:浅蓝色。浅色(Latte)背景用中等饱和蓝,深色(Mocha)背景用亮浅蓝。
	lightBlue := lightDark(lipgloss.Color("#4C8EFF"), lipgloss.Color("#8AD7FF"))
	// 错误提示:红色。
	red := lightDark(lipgloss.Color("#D64550"), lipgloss.Color("#FF7A85"))

	st := huh.ThemeCatppuccin(isDark)

	// 主要文字 → 浅蓝色
	st.Focused.Base = st.Focused.Base.BorderForeground(lightBlue)
	st.Focused.Title = st.Focused.Title.Foreground(lightBlue).Bold(true)
	st.Focused.NoteTitle = st.Focused.NoteTitle.Foreground(lightBlue)
	st.Focused.Directory = st.Focused.Directory.Foreground(lightBlue)
	st.Focused.File = st.Focused.File.Foreground(lightBlue)
	st.Focused.Option = st.Focused.Option.Foreground(lightBlue)
	st.Focused.SelectedOption = st.Focused.SelectedOption.Foreground(lightBlue)
	st.Focused.SelectedPrefix = st.Focused.SelectedPrefix.Foreground(lightBlue)
	st.Focused.UnselectedPrefix = st.Focused.UnselectedPrefix.Foreground(lightBlue)
	st.Focused.TextInput.Text = st.Focused.TextInput.Text.Foreground(lightBlue)
	st.Focused.TextInput.Prompt = st.Focused.TextInput.Prompt.Foreground(lightBlue)
	st.Focused.TextInput.Cursor = st.Focused.TextInput.Cursor.Foreground(lightBlue)

	// 必填错误提示 → 红色
	st.Focused.ErrorIndicator = st.Focused.ErrorIndicator.Foreground(red)
	st.Focused.ErrorMessage = st.Focused.ErrorMessage.Foreground(red)

	// 失焦状态复用聚焦配色,仅保留隐藏边框与空白指示器
	blurredBase := st.Blurred.Base
	st.Blurred = st.Focused
	st.Blurred.Base = blurredBase
	st.Blurred.NextIndicator = lipgloss.NewStyle()
	st.Blurred.PrevIndicator = lipgloss.NewStyle()

	st.Group.Title = st.Focused.Title
	st.Group.Description = st.Focused.Description
	return st
}

// SetTheme 替换 prompt 包内所有交互组件的主题。
// 传入 nil 会恢复默认主题(浅蓝文字 + 红色错误提示)。
func SetTheme(t huh.Theme) {
	if t == nil {
		theme = huh.ThemeFunc(defaultTheme)
		return
	}
	theme = t
}
