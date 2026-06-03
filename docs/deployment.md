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

在接真实服务器前，先补齐部署骨架：

```text
Dockerfile
docker-compose.prod.yml
.env.production.example
.github/workflows/ci.yml
.github/workflows/deploy.yml
```

先在本机用 Docker 模拟生产启动和 migration，再接入真实服务器、域名、HTTPS 和 R2 凭据。
