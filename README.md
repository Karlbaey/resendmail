# resendmail

`resendmail` 是一个用 Go 实现的 CLI，直接调用 Resend HTTP API 发送电子邮件。

友情链接：[LINUX DO](https://linux.do)。

## 想法

发邮件大多要开一个网页端写邮件，或者写个脚本，我嫌麻烦就用 Go 糊了一个命令行版的。

你得先去 [Resend](https://resend.com) 注册个账号，获取 API Key。

## 要求

- Go 1.26+
- 环境变量 `RESEND_API_KEY`

## 手动编译

在仓库根目录执行：

```bash
go build -o resendmail .
```

在 macOS/Linux 上会生成 `./resendmail`，在 Windows 上会生成 `.\resendmail.exe`。

## 用法

```bash
resendmail send \
  --from sender@example.com \
  --to alice@example.com \
  --to bob@example.com \
  --cc copy@example.com \
  --bcc boss@example.com \
  --subject "Hello" \
  --text "Plain text body"
```

也可以从文件读取正文：

```bash
resendmail send \
  --from sender@example.com \
  --to alice@example.com \
  --subject "HTML mail" \
  --html-file body.html
```

### 交互模式

加 `-i`/`--interactive` 进入问答式发送，依次询问发件人、收件人、抄送、密送、主题、正文与附件：

```bash
resendmail send --interactive
# 或
resendmail send -i
```

流程说明：

- 发件人、收件人、主题必填，抄送与密送可直接回车跳过
- 多个地址用英文逗号分隔
- 正文先选格式（纯文本 / HTML），再选输入方式：
  1. **终端输入**：Enter 换行，Ctrl+D 完成
  2. **打开编辑器**：调用 `$EDITOR`；未设置时 Windows 用 `notepad`，其他平台用 `vim`
  3. **选择文件**：输入文件路径读取正文
- 附件可选：文件选择器逐次挑选，选完后在「继续添加附件吗?」选「否」，或直接 Ctrl+C 跳过
- 收集完毕会显示邮件预览（正文超过 50 字截断），确认后才发送
- 发送前在任何一步 Ctrl+C 即取消，不会发出邮件

> 交互模式下**其余所有 flag 均被忽略**，请勿混用；若需要脚本化或指定参数，请用上面的 flag 方式。

### 规则

- `--from`、`--to`、`--subject` 必填
- 至少提供一个正文来源：`--text`、`--text-file`、`--html`、`--html-file`
- `--text` 和 `--text-file` 互斥
- `--html` 和 `--html-file` 互斥
- `--to`、`--cc`、`--bcc` 可重复传入
- 请求超时固定为 `12s`(见 `internal/send/client.go`)

发送成功后输出邮件 ID 与网页查看链接，例如：

```text
Success. Email ID: email_123
Check email on web: https://resend.com/emails/email_123
```

## 开发

默认验证步骤：

```bash
gofmt -w *.go
go test ./...
```
