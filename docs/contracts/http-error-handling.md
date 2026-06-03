# HTTP 错误响应约定

本文记录错误语义和 HTTP 响应之间的边界。

错误码、HTTP 状态码和错误响应形状需要通过以下脚本与 `internal/apperr`、`internal/platform/httpserver` 保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1
```

该脚本校验错误码集合、HTTP 状态码映射和 `{"error":{"code":"...","message":"..."}}` 形状；它不校验每个业务场景的错误消息全文。

## 分层

项目错误分两层：

- `internal/apperr`：项目级错误语义，不依赖 HTTP。
- `internal/platform/httpserver`：把项目错误映射成 HTTP 状态码和 JSON 响应。

domain/usecase 可以返回 `apperr`，但不能决定 HTTP 状态码。

## 响应结构

错误响应固定为：

```json
{
  "error": {
    "code": "not_found",
    "message": "resource not found"
  }
}
```

未知系统错误统一返回：

```json
{
  "error": {
    "code": "internal",
    "message": "internal server error"
  }
}
```

不要把数据库错误、panic 内容、密钥、SQL 或内部堆栈返回给客户端。

## 错误码

| code | HTTP | 语义 |
| --- | ---: | --- |
| `invalid_argument` | 400 | 请求格式或输入不合法 |
| `unauthenticated` | 401 | 未登录或认证信息无效 |
| `forbidden` | 403 | 已认证但当前状态不允许 |
| `not_found` | 404 | 资源不存在 |
| `conflict` | 409 | 唯一约束、状态冲突或重复操作 |
| `internal` | 500 | 未预期系统错误 |

## handler 规则

handler 遇到错误时只做：

```go
_ = c.Error(err)
c.Abort()
return
```

成功时才写业务响应。

handler 不手写错误 JSON，不猜 HTTP 状态码。

## 错误包装

包装错误必须用 `%w`：

```go
return fmt.Errorf("create user: %w", err)
```

不要用 `%v` 包装需要保留语义的错误。HTTP 层依赖 `errors.As` 识别 `apperr`。

## 中间件

基础 router 注册顺序：

```text
RecoveryMiddleware
RequestIDMiddleware
RequestLoggerMiddleware
ErrorMiddleware
```

`RecoveryMiddleware` 负责兜住 panic。

`ErrorMiddleware` 负责读取 Gin context 上的错误并输出统一响应。

## 常见语义边界

- repository 可以返回 `not_found`。
- 登录 usecase 必须把“用户不存在”和“密码错误”统一成 `unauthenticated`。
- 数据库唯一约束冲突应映射成 `conflict`。
- disabled 用户登录应返回 `forbidden`。
- token 缺失、格式错误、过期或签名错误应返回 `unauthenticated`。

## 测试要求

至少覆盖：

- `apperr` 到 HTTP 状态码的映射。
- 未知错误不会泄露原始错误文本。
- panic 返回统一 `internal`。
- 业务 handler 错误路径走中间件。
- `scripts/verify-http-error-contract-doc.ps1` 校验文档表格与代码映射同步。
