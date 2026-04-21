# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端项目。

当前仓库处于初始化阶段，当前已明确的基础方向如下：

- 技术栈以 Go 为主
- 架构采用模块化单体
- 主数据库使用 PostgreSQL

## 当前状态

目前仓库重点在于：

- 初始化项目骨架
- 整理公开文档
- 建立基础工程约定

## 近期目标

第一阶段聚焦项目底座，包括：

- Go 项目骨架
- 配置系统
- PostgreSQL 连接
- migration 机制
- HTTP 服务最小启动链路
- 统一错误响应与基础日志

## 目录说明

当前关键目录约定如下：

- `cmd/`：程序入口与依赖组装
- `internal/`：业务模块与平台基础设施
- `docs/public/`：对外公开的说明文档
- `docs/internal/`：内部设计与规划文档
- `pkg/`：无业务语义的公共工具包

## 本地开发

以下命令默认都在仓库根目录执行。

### 前置依赖

- Go
- Docker
- Docker Compose

### 环境变量

本地开发使用 `.env` 文件，生产环境仍以真实环境变量为准。

先复制示例配置：

```bash
cp .env.example .env
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

当前最小必需配置见 [`.env.example`](./.env.example)，包括：

- `APP_NAME`
- `POSTGRES_*`
- `HTTP_*`
- `LOG_*`

### 启动 PostgreSQL

项目根目录提供了 [compose.yaml](./compose.yaml)，当前只用于启动本地 PostgreSQL：

```bash
docker compose up -d postgres
```

查看状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f postgres
```

### 执行 migration

当前 migration 入口为 `cmd/migrate`，SQL 文件位于 `migrations/` 目录。

执行升级：

```bash
go run ./cmd/migrate up
```

查看当前版本：

```bash
go run ./cmd/migrate version
```

执行回滚：

```bash
go run ./cmd/migrate down
```

### 验证数据库连通性

当前 `cmd/api` 还不是完整 HTTP 服务入口，现阶段主要用于：

- 加载配置
- 初始化 PostgreSQL 连接池
- 执行数据库连通性检查

执行：

```bash
go run ./cmd/api
```

如果配置和数据库都正常，当前会输出数据库连接成功信息。

### 当前限制

- 当前阶段尚未完成 Gin HTTP 服务骨架
- `/healthz` 还未接入
- `cmd/api` 目前不会常驻启动服务，只做启动期基础设施检查

## 测试说明

当前可执行的基础检查：

```bash
go test ./...
```

## License

本项目当前采用 [MIT License](./LICENSE)。
