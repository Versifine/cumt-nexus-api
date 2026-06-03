# Contract Documents

本目录存放可提交、可被 CI 校验、可供前后端和部署协作引用的项目合同文档。

这些文档不是内部过程记录，也不包含真实密钥。它们是当前后端行为的轻量合同源，在没有 OpenAPI 生成流程之前，用脚本和源码保持同步。

| 文档 | 职责 |
|---|---|
| `http-api-contract.md` | HTTP 路由、认证边界和查询参数清单 |
| `http-api-schema.md` | 请求/成功响应 schema、接口 schema 映射和请求必填字段 |
| `http-error-handling.md` | 业务错误码、HTTP 状态码和统一错误响应形状 |
| `configuration.md` | 环境变量、默认值、枚举和 R2/local 条件必需语义 |
| `migrations.md` | migration 文件命名、配对、连续编号和清单 |

校验入口：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip
```

`docs/internal/` 只放内部架构推演、工作流和本地协作记录；不能作为 CI 或前端协作依赖的唯一合同源。
