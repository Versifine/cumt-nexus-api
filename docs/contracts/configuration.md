# 配置说明

配置由 `internal/platform/config` 统一加载。

本地开发从 `.env` 读取；生产环境应使用真实环境变量。

## 规则

- 业务层不直接读取环境变量。
- 缺失必需配置时启动失败。
- 时间配置使用 Go duration 格式，例如 `10s`、`5m`、`24h`。
- `.env.example` 必须覆盖本地启动所需配置。

配置项清单需要通过以下脚本与 `internal/platform/config/load.go` 和 `.env.example` 保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-contract-doc.ps1
```

配置项必需性、默认值和枚举说明需要通过以下脚本与 `internal/platform/config/load.go` 和 `internal/platform/config/validate.go` 保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-semantics-doc.ps1
```

配置加载运行时契约由以下测试覆盖：

```powershell
go test ./internal/platform/config -run TestLoad -v
```

## 配置项

### App

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `APP_NAME` | 是 | 无 | 应用名 |
| `APP_ENV` | 否 | `local` | `local/dev/test/prod` |
| `APP_STARTUP_TIMEOUT` | 否 | `10s` | 启动期初始化超时 |

### PostgreSQL

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `POSTGRES_HOST` | 是 | 无 | PostgreSQL host |
| `POSTGRES_PORT` | 否 | `5432` | PostgreSQL port |
| `POSTGRES_USER` | 是 | 无 | PostgreSQL user |
| `POSTGRES_PASSWORD` | 是 | 无 | PostgreSQL password |
| `POSTGRES_DATABASE` | 是 | 无 | PostgreSQL database |
| `POSTGRES_SSL_MODE` | 否 | `disable` | `disable/require/verify-ca/verify-full` |
| `POSTGRES_MAX_CONNS` | 否 | `25` | 连接池最大连接数 |
| `POSTGRES_MAX_CONN_LIFETIME` | 否 | `5m` | 单连接最大生命周期 |
| `POSTGRES_MAX_CONN_IDLE_TIME` | 否 | `2m` | 空闲连接保留时间 |

### HTTP

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `HTTP_ADDR` | 否 | `:8080` | HTTP 监听地址 |
| `HTTP_READ_TIMEOUT` | 否 | `5s` | 请求读取超时 |
| `HTTP_WRITE_TIMEOUT` | 否 | `10s` | 响应写入超时 |
| `HTTP_SHUTDOWN_TIMEOUT` | 否 | `15s` | 优雅关闭超时 |
| `HTTP_CORS_ALLOWED_ORIGINS` | 否 | 空 | 逗号分隔的允许跨域来源；空表示不启用 CORS，`*` 表示允许任意来源 |

### Log

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `LOG_LEVEL` | 否 | `info` | `debug/info/warn/error` |
| `LOG_FORMAT` | 否 | `json` | `json/text` |

### Auth

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `AUTH_TOKEN_SECRET` | 是 | 无 | JWT 签名密钥 |
| `AUTH_ACCESS_TOKEN_TTL` | 否 | `24h` | access token 有效期 |

### Object Storage

阶段 13 引入。生产/主方案对象存储直接使用 Cloudflare R2，local fallback 只用于本地无凭据验证。

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `OBJECT_STORAGE_PROVIDER` | 否 | `local` | `r2` 或 `local` |
| `OBJECT_STORAGE_ENDPOINT` | `r2` 必需 | 无 | R2 S3 endpoint |
| `OBJECT_STORAGE_REGION` | 否 | `auto` | R2 通常使用 `auto` |
| `OBJECT_STORAGE_BUCKET` | `r2` 必需 | 无 | R2 bucket |
| `OBJECT_STORAGE_ACCESS_KEY_ID` | `r2` 必需 | 无 | R2 access key id |
| `OBJECT_STORAGE_SECRET_ACCESS_KEY` | `r2` 必需 | 无 | R2 secret access key |
| `OBJECT_STORAGE_PUBLIC_BASE_URL` | `r2` 必需 | local 空值时补 `http://localhost:8080/uploads` | 图片访问 base URL |
| `OBJECT_STORAGE_FORCE_PATH_STYLE` | 否 | `true` | R2 S3-compatible client 使用 path-style |
| `OBJECT_STORAGE_LOCAL_ROOT` | 否 | `var/uploads` | local fallback 文件根目录；最终配置不能为空 |

### Upload

阶段 13 引入。

| 变量 | 必需 | 默认 | 说明 |
| --- | --- | --- | --- |
| `UPLOAD_IMAGE_MAX_BYTES` | 否 | `5242880` | 单图片最大字节数 |
| `UPLOAD_IMAGE_MAX_COUNT_PER_POST` | 否 | `9` | 单帖最大图片数 |
| `UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT` | 否 | `1` | 单评论最大图片数 |

## 本地使用

```bash
cp .env.example .env
docker compose up -d postgres
go run ./cmd/migrate up
go run ./cmd/api
```

## 当前不做

当前仍不配置：

- refresh token TTL
- OAuth client
- cookie domain
- Redis
- 外部搜索

这些配置等真实能力接入时再增加。

## 当前校验边界

- `scripts/verify-config-contract-doc.ps1` 只校验环境变量名是否在配置加载代码、`.env.example` 和本文档之间保持一致。
- `scripts/verify-config-semantics-doc.ps1` 校验本文档中的必需性、默认值和枚举说明是否与 `load.go` / `validate.go` 同步。
- `go test ./internal/platform/config -run TestLoad -v` 覆盖 `Load()` 的 local 默认值、R2 配置加载、R2 缺凭据失败和基础解析失败路径。
- 具体数值范围、跨字段约束和错误消息文本仍由 `internal/platform/config` 的测试维护。
- R2 真实凭据不写入 `.env.example`，只保留变量名和空占位。
