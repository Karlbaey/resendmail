# resendmail 需求确认记录

本文档把需求确认阶段的对话结论沉淀为可执行的范围说明，避免后续实现时出现隐含假设。

## 项目目标

从零开始，用 Go 实现一个通过 Resend API 发送电子邮件的 CLI 程序。

## 决策记录

1. CLI 形态

- 结论：第一版使用 `resendmail send`。
- 原因：单一命令的 MVP 最稳，后续再扩展 `config`、`template` 或 `bulk`。

2. 邮件正文格式

- 结论：同时支持 `text` 和 `html`，至少提供一个。
- 原因：兼顾简单脚本场景和 HTML 邮件场景。

3. 收件人模型

- 结论：支持多个 `to` 收件人。
- 约定：`--to` 可重复传入，也可接受逗号分隔。

4. 正文输入来源

- 结论：支持内联内容和文件输入。
- 约定：使用 `--text` / `--html` 传内联内容，使用 `--text-file` / `--html-file` 从文件读取。
- 约束：同一种内容源互斥，例如不能同时传 `--text` 和 `--text-file`。

5. 附件

- 结论：第一版不支持附件。
- 原因：附件会引入 MIME、Base64、体积限制和更多错误处理，不属于 MVP。

6. 抄送与密送

- 结论：支持 `cc`，第一版不支持 `bcc`。

7. 认证与配置

- 结论：API Key 只从环境变量 `RESEND_API_KEY` 读取。
- 原因：最适合 CLI 和脚本，也避免引入本地配置文件策略。

8. 发件人地址

- 结论：每次发送都必须显式传入 `--from`。
- 原因：不同 Resend 账号的已验证域名不同，不能假设默认发件地址可用。

9. 输出格式

- 结论：发送成功时输出结构化 JSON。
- 约束：失败时输出明确错误信息，并返回非 0 退出码。

10. 集成方式

- 结论：第一版直接调用 Resend HTTP API。
- 原因：依赖更少，行为更透明，也更容易排查请求和响应。

11. CLI 框架

- 结论：只使用 Go 标准库，不引入 `cobra` 等框架。

12. 必填字段

- 结论：`--from`、`--to`、`--subject` 必填。
- 约束：至少需要一个 `--to`。

13. 测试边界

- 结论：第一版带单元测试，但不做默认的真实 Resend 集成测试。

14. 本地参数校验

- 结论：使用 Go 标准库 `net/mail` 做基础邮箱格式校验。
- 范围：覆盖 `from`、`to`、`cc`。

15. 超时策略

- 结论：HTTP 请求默认超时为 `10s`，并支持通过 `--timeout` 覆盖。
- 约定：时间格式使用 Go duration，例如 `10s`、`30s`。

16. 重试策略

- 结论：第一版不自动重试。
- 原因：发邮件不是天然幂等，自动重试可能造成重复发送。

17. 命名

- 结论：Go module 名为 `resendmail`，可执行文件名也为 `resendmail`。

18. 文档

- 结论：需要补充 `README.md`。

19. 仓库约束

- 结论：需要补充 `AGENTS.md`。
- 额外约束：所有文本文件一律使用 CRLF 换行。

20. GitHub 发布流程

- 结论：使用 GitHub Actions 在推送 `vX.Y.Z` tag 时自动构建并发布。
- 约定：仅 `vX.Y.Z` 触发正式发布，不支持 `rc` 或其他预发布 tag。
- 约定：发布前必须通过 `gofmt` 检查和 `go test ./...`。
- 约定：发布产物覆盖 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64`。
- 约定：`linux` 和 `darwin` 产物使用 `.tar.gz`，`windows` 使用 `.zip`。
- 约定：归档文件名包含版本、系统和架构，并额外上传 `SHA256SUMS`。
- 约定：GitHub Release 文案使用自动生成的 release notes。

## 当前冻结范围

下面这份清单是后续实现必须遵守的第一版规格：

- Go module：`resendmail`
- 可执行文件：`resendmail`
- 命令形态：`resendmail send`
- 仅使用 Go 标准库
- 直接调用 Resend HTTP API
- API Key 只从环境变量 `RESEND_API_KEY` 读取
- `--from`、`--to`、`--subject` 必填
- `--to` 支持多个值
- 支持 `--cc`
- 第一版不支持 `bcc`
- 支持 `text` 和 `html`
- 至少提供一个正文来源
- 支持内联内容和文件输入
- 同类正文来源互斥
- 第一版不支持附件
- 使用 `net/mail` 做基础邮箱格式校验
- HTTP 默认超时 `10s`
- 支持 `--timeout`
- 第一版不自动重试
- 成功输出 JSON
- 失败返回非 0 并输出明确错误
- 提供单元测试
- 不提供默认真实 Resend 集成测试
- 推送 `vX.Y.Z` tag 时自动创建或更新 GitHub Release
- Release 前执行 `gofmt` 检查和 `go test ./...`
- Release 产物包含 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64`
- `linux`/`darwin` 上传 `.tar.gz`，`windows` 上传 `.zip`
- Release 额外上传 `SHA256SUMS`
- Release notes 使用 GitHub 自动生成内容

## 实现前说明

本文档记录的是需求确认结果，不是 API 设计文档，也不是代码实现说明。进入实现阶段后，若有新增需求或改动，先更新本文档，再更新代码。
