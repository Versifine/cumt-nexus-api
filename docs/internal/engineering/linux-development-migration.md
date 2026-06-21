# Linux 开发环境迁移备忘录

本文用于把 `cumt-nexus-api` 从当前 Windows 本地开发环境迁移到 Linux 开发环境。目标是先建立可重复的开发闭环，再处理脚本跨平台化和历史本地数据迁移。

## 迁移原则

- 优先 fresh clone 到 Linux 文件系统，例如 `~/dev/cumt-nexus-api`，不要直接把 Windows 工作目录整体搬过去。
- 代码、提交历史和可提交文档以 Git 为准；ignored 的本地协作文件和环境变量单独复制。
- 本地开发和生产模拟分开跑：`compose.yaml` 用于本地 PostgreSQL，`docker-compose.prod.yml` 用于生产形态模拟。
- 迁移第一天先跑最小闭环，不急着迁移旧 Docker volume 或压测数据。

## 迁移前检查

在 Windows 旧环境先确认当前状态：

```bash
git status --short
git branch --show-current
git log --oneline -5
```

如果存在未提交改动，先决定是提交、stash，还是打包备份。不要在 Linux 新环境里靠手工复制半截改动恢复状态。

这些文件通常不会随 Git 自动迁移，需要按需复制：

```text
AGENTS.md
.ai/
docs/internal/
.env
.env.production
```

其中 `.env` 和 `.env.production` 可能含密钥，只能通过本机安全渠道复制，不要提交。

## Linux 依赖

需要安装：

```text
git
go 1.25.4
docker engine
docker compose plugin
curl
jq
postgresql-client
```

`go.mod` 和 `Dockerfile` 当前都使用 Go `1.25.4`。如果发行版仓库没有这个版本，使用官方 tarball、asdf 或 mise 管理 Go 版本，不要降级 Go。

建议设置：

```bash
git config --global core.autocrlf input
git config --global core.eol lf
```

## 首次启动

```bash
git clone <repo-url> ~/dev/cumt-nexus-api
cd ~/dev/cumt-nexus-api

cp .env.example .env
docker compose up -d postgres

go run ./cmd/migrate up
go run ./cmd/api
```

另开终端验证：

```bash
curl http://localhost:8080/healthz
go test ./...
go build -buildvcs=false ./...
./scripts/verify-current-baseline.sh --skip-http-smoke --r2-mode Skip
```

`verify-current-baseline.sh` 依赖 PostgreSQL 已启动。第一次迁移先使用 `--skip-http-smoke --r2-mode Skip`，确认代码、migration、合同文档和构建链路可跑通。

## 生产形态模拟

生产形态模拟使用独立 compose project，不与本地开发数据库 volume 混用：

```bash
cp .env.production.example .env.production

docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
curl http://localhost:8080/healthz
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate version
```

停止但保留数据：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

停止并删除模拟数据：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down -v
```

## 配置要点

本地开发默认使用：

```text
POSTGRES_HOST=localhost
POSTGRES_USER=postgres
POSTGRES_DATABASE=cumt_nexus
OBJECT_STORAGE_PROVIDER=local
OBJECT_STORAGE_PUBLIC_BASE_URL=http://localhost:8080/uploads
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

生产模拟的 `docker-compose.prod.yml` 会把 `POSTGRES_HOST` 覆盖为 `postgres`，数据库用户来自 `.env.production`。

如果前端也迁移到 Linux 并继续跑 Next dev server，默认 CORS 仍覆盖 `localhost:3000`。如果前端运行在另一台机器、WSL bridge、局域网 IP 或反向代理后面，必须把浏览器实际 origin 加到 `HTTP_CORS_ALLOWED_ORIGINS`。

## 运行时排查顺序

不要只看 Docker `healthy`。本项目排运行时问题时按这个顺序确认：

```bash
curl http://localhost:8080/healthz
curl -i http://localhost:8080/api/v1/posts
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker inspect <api-container-name>
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate version
```

重点区分三件事：

- 当前源码是否正确。
- 当前运行的容器镜像是否来自最新源码。
- 当前数据库 migration 是否已经到目标版本。

## 脚本现状

当前仓库脚本入口统一为 Bash/Python，不再依赖 PowerShell：

```bash
./scripts/verify-current-baseline.sh --skip-http-smoke --r2-mode Skip
./scripts/smoke-stage-13-content-system.sh --skip-migration
./scripts/smoke-stage-14-content-lifecycle.sh --skip-migration
./scripts/smoke-stage-15-r2-upload.sh --skip-migration --skip-when-missing-credentials
```

压测入口也使用 Bash：

```bash
./scripts/loadtest/run-local-loadtest.sh
./scripts/loadtest/run-fixed-rps-ladder.sh --endpoint search_all --skip-seed
```

## 旧数据迁移

如果只是继续开发，建议 fresh DB：

```bash
docker compose up -d postgres
go run ./cmd/migrate up
```

如果确实要迁移旧数据库，使用 dump/restore，不要直接复制 Docker volume：

```bash
pg_dump -h localhost -U postgres -d cumt_nexus -Fc -f cumt_nexus.dump
pg_restore -h localhost -U postgres -d cumt_nexus --clean --if-exists cumt_nexus.dump
```

迁移后检查：

```bash
go run ./cmd/migrate version
curl http://localhost:8080/healthz
```

## 首日验收标准

迁移第一天只要求完成这些结果：

- Linux 新目录可以正常 `go test ./...`。
- Linux 新目录可以正常 `go build -buildvcs=false ./...`。
- `compose.yaml` 的 PostgreSQL 可以启动。
- `go run ./cmd/migrate up` 可以把数据库迁到最新版本。
- `go run ./cmd/api` 后 `/healthz` 返回 `{"status":"ok"}`。
- 快速 baseline 通过或只剩已知的跨平台 smoke 脚本问题。

这些通过后，再继续处理脚本跨平台化、旧数据迁移、R2 凭据和生产模拟。
