# resendmail

`resendmail` 是一个使用 Go 标准库实现的 CLI，用来直接调用 Resend HTTP API 发送电子邮件。

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
  --to alice@example.com,bob@example.com \
  --cc copy@example.com \
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

### 规则

- `--from`、`--to`、`--subject` 必填
- 至少提供一个正文来源：`--text`、`--text-file`、`--html`、`--html-file`
- `--text` 和 `--text-file` 互斥
- `--html` 和 `--html-file` 互斥
- `--to` 和 `--cc` 可重复传入，也支持逗号分隔
- 默认请求超时是 `10s`，可通过 `--timeout` 覆盖

发送成功后输出 JSON，例如：

```json
{"id":"email_123"}
```

## 开发

默认验证步骤：

```bash
gofmt -w *.go
go test ./...
```
