# 部署与版本切换规则

本文记录 `cumt-nexus-api` 的最低复杂度部署规则。目标不是一次性做完复杂平台，而是先保证后端可以被重复构建、明确发布、稳定回滚，并且不让本地开发状态影响服务器。

## 核心模型

本地开发和服务器运行是两条线：

```text
本地：持续开发最新代码
服务器：只运行一个明确发布版本
```

服务器不直接跟随开发分支，也不运行“当前最新代码”。服务器只切换到明确的 release 版本。

## 需要管理的版本

发布时至少同时确认这些版本：

| 版本 | 说明 |
|---|---|
| 后端代码版本 | Git tag、commit 或 Docker image tag |
| 数据库版本 | 当前 migration version |
| 前端版本 | 前端适配的后端 API contract |
| 配置版本 | 服务器 `.env`、部署平台 secrets、R2 凭据 |

其中最关键的是：

```text
后端 Docker image tag + 数据库 migration version
```

## 分支和发布约定

默认规则：

```text
开发分支：stage/*、fix/*、spike/*
稳定分支：main
服务器部署：只部署 main 上的 tag
版本命名：v0.1.0、v0.2.0、v0.3.0
Docker image tag：与 Git tag 保持一致
```

不要把服务器部署到某个正在开发的 `stage/*` 分支。开发分支通过验证后，先合并到 `main`，再从 `main` 打 tag 发布。

## 标准发布流程

推荐发布顺序：

```text
1. 本地或 CI 跑完测试、构建和基线校验
2. 合并到 main
3. 从 main 打 Git tag，例如 v0.2.0
4. CI 构建 Docker image，例如 ghcr.io/<owner>/cumt-nexus-api:v0.2.0
5. 推送 Docker image
6. 服务器把 API image 从旧 tag 切到新 tag
7. 备份数据库
8. 执行 migration up
9. 启动或重启 API
10. 检查 /healthz 和关键业务冒烟
```

服务器上的 `.env.production` 可以用镜像变量控制当前运行版本：

```env
API_IMAGE=ghcr.io/<owner>/cumt-nexus-api:v0.2.0
```

部署新版本：

```bash
docker compose pull
docker compose up -d
```

## 回滚规则

代码回滚优先切回旧 Docker image tag：

```env
API_IMAGE=ghcr.io/<owner>/cumt-nexus-api:v0.1.0
```

然后执行：

```bash
docker compose pull
docker compose up -d
```

数据库不默认自动回滚。migration 设计要优先保持向前兼容：

- 优先新增表、字段、索引。
- 避免直接 drop 字段、rename 字段或改变既有字段语义。
- 需要 destructive migration 时必须单独设计发布步骤和回滚方案。
- 发布前先备份数据库。

如果 migration 只是新增表、新增字段或新增索引，通常可以把 API image 回滚到旧版本；如果 migration 删除或重写了旧代码依赖的字段，回滚会变复杂，不能按普通流程处理。

## 已有部署骨架文件

当前仓库已经提供以下部署骨架：

| 文件 | 职责 |
|---|---|
| `Dockerfile` | 构建同一个运行镜像，包含 `api` 和 `migrate` 两个二进制以及 `migrations/` |
| `.dockerignore` | 控制 Docker build context，排除 `.env`、本地缓存、内部 AI 记录等内容 |
| `docker-compose.prod.yml` | 生产形态的 PostgreSQL、migration job 和 API 服务编排 |
| `.env.production.example` | 生产/生产模拟环境变量模板，不包含真实密钥 |
| `.github/workflows/ci.yml` | push / PR 时运行基线校验和 Docker build |
| `.github/workflows/deploy.yml` | tag 或手动触发时构建并推送 GHCR 镜像；可选手动 SSH 部署 |

## 本机模拟生产启动

前提：

```text
Docker Engine 正在运行
Docker Compose plugin 可用
```

先从示例文件生成本地生产模拟配置：

```powershell
Copy-Item .env.production.example .env.production
```

macOS/Linux:

```bash
cp .env.production.example .env.production
```

本机模拟默认使用：

```text
API_IMAGE=cumt-nexus-api:local
OBJECT_STORAGE_PROVIDER=local
POSTGRES_HOST=postgres
GIN_MODE=release
```

`docker-compose.prod.yml` 使用独立 project name `cumt-nexus-api-prod`，避免和本地开发 `compose.yaml` 的 PostgreSQL volume 混用。

构建并启动：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

查看服务：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

检查 API：

```bash
curl http://localhost:8080/healthz
```

查看 migration 版本：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate version
```

停止并保留数据卷：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

停止并删除本机模拟数据卷：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down -v
```

## CI/CD 最小目标

CI 至少覆盖：

```bash
go test ./...
go build -buildvcs=false ./...
```

还应跑快速基线：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip
```

当前 CI 文件是：

```text
.github/workflows/ci.yml
```

它会在 `main`、`stage/**`、`fix/**`、`spike/**` 的 push 以及 pull request 上运行：

```text
1. 启动 PostgreSQL service
2. 安装 Go
3. go mod download
4. verify-current-baseline.ps1 -SkipInternalDocChecks -SkipHttpSmoke -R2Mode Skip
5. docker build -t cumt-nexus-api:ci .
```

`docs/internal/` 是本地内部文档，不提交到 GitHub。CI 必须使用 `-SkipInternalDocChecks`，否则 GitHub checkout 中缺少内部文档会导致合同文档校验失败。本地完整验收仍应不带该参数运行。

CD 最小动作：

```text
1. 构建 Docker image
2. 推送 image 到 registry
3. SSH 到服务器
4. 拉取指定 tag 的 image
5. 执行 migration up
6. 重启 API
7. 检查 /healthz
```

CD 不应把未打 tag 的开发分支直接部署到生产服务器。

当前部署 workflow 是：

```text
.github/workflows/deploy.yml
```

它支持两种触发：

```text
1. push v*.*.* tag：构建并推送 GHCR 镜像
2. workflow_dispatch：手动指定版本，构建并推送 GHCR 镜像；可选择是否 SSH 部署
```

镜像名格式：

```text
ghcr.io/<owner>/<repo>:<version>
```

例如：

```text
ghcr.io/versifine/cumt-nexus-api:v0.2.0
```

手动 SSH 部署需要在 GitHub repository secrets 中配置：

| Secret | 说明 |
|---|---|
| `DEPLOY_HOST` | 服务器地址 |
| `DEPLOY_USER` | SSH 用户 |
| `DEPLOY_SSH_KEY` | 私钥内容 |
| `DEPLOY_PATH` | 服务器上的部署目录，目录内应有 `docker-compose.prod.yml` 和 `.env.production` |

服务器部署目录至少包含：

```text
docker-compose.prod.yml
.env.production
```

其中 `.env.production` 不提交到仓库，只保存在服务器上。

## 配置和密钥

密钥不进代码、不进 README、不进提交信息。

配置存放位置：

| 环境 | 存放位置 |
|---|---|
| 本地开发 | `.env` |
| GitHub Actions | GitHub Secrets |
| 服务器 | 服务器本地 `.env.production` 或部署平台 secret |

Cloudflare R2 使用 S3-compatible 配置，endpoint 使用账号级地址，bucket 单独配置：

```env
OBJECT_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
OBJECT_STORAGE_BUCKET=<bucket-name>
```

不要把 bucket 名拼进 endpoint，不要提交真实 access key 或 secret key。

## 当前阶段建议

当前已经补齐基础部署骨架。接真实服务器前还需要确认：

```text
服务器已安装 Docker Engine 和 Docker Compose plugin
服务器部署目录已放置 docker-compose.prod.yml
服务器部署目录已配置 .env.production
服务器可以访问 GHCR 镜像
域名、HTTPS、反向代理方案已确定
数据库备份策略已确定
```

先在本机用 Docker 模拟生产启动和 migration，再接入真实服务器、域名、HTTPS 和 R2 凭据。
