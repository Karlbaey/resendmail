# AGENTS

本仓库的协作约束如下：

* 所有文本文件一律使用 CRLF 换行。

* MVP 的 Go module 名为 `resendmail`。

* MVP 的可执行文件名为 `resendmail`。

* 第一版 CLI 命令形态固定为 `resendmail send`。

* 第一版仅使用 Go 标准库，不引入 CLI 框架，也不引入 Resend SDK。

* 第一版直接调用 Resend HTTP API。

* 需求范围发生变化时，必须同步更新 [docs/requirements.md](docs/requirements.md)。

* 默认验证步骤是 `gofmt` 和 `go test ./...`。
