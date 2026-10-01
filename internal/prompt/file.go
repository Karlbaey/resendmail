package prompt

import (
	"charm.land/huh/v2"
)

// PickFile 循环选择多个本地文件，返回选中的文件路径列表（去重）。
//
// huh 的 FilePicker 只能单选，所以这里用「选一个 → 确认是否继续」实现多选。
// 附件是可选的,因此结束选择有两条明确路径：
//   - 在「继续添加附件吗？」时选择「否」，表示选完了，函数返回已收集的文件；
//   - 任何时候按 Ctrl+C（huh 的全局中止键，经 IsAborted 捕获），
//     同样视为「选完了」，返回已收集的文件。
//
// 注意：huh v2 的 FilePicker 中 Esc 只是「退出选择模式/返回上级目录」，
// 并不会结束选择，所以不要依赖 Esc 来终止附件挑选。
func PickFile(title string) ([]string, error) {
	var files []string
	for {
		picked, err := pickOne(title)
		if err != nil {
			if IsAborted(err) {
				return files, nil // Ctrl+C 视为选完,返回已收集的文件
			}
			return nil, err
		}
		files = appendUnique(files, picked)

		more, err := Confirm("继续添加附件吗?")
		if err != nil {
			if IsAborted(err) {
				return files, nil // 确认环节 Ctrl+C 同样视为选完
			}
			return nil, err
		}
		if !more {
			return files, nil // 明确表示选完了
		}
	}
}

// pickOne 弹出一次文件选择器，返回用户选中的单个文件路径。
// 选择器内：Enter 选中文件，Enter/→ 进入目录，Esc/← 返回上级目录或退出选择模式。
func pickOne(title string) (string, error) {
	var file string
	fp := huh.NewFilePicker().
		Title(title).
		DirAllowed(false).
		Picking(true).
		Value(&file).
		WithTheme(theme)
	if err := fp.Run(); err != nil {
		return "", normalizeErr(err)
	}
	return file, nil
}

// appendUnique 把 picked 追加到 files，空路径或已存在的路径会被忽略。
func appendUnique(files []string, picked string) []string {
	if picked == "" || contains(files, picked) {
		return files
	}
	return append(files, picked)
}

func contains(array []string, s string) bool {
	for _, x := range array {
		if x == s {
			return true
		}
	}
	return false
}
