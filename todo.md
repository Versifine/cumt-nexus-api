# CUMT-Nexus 项目开发路线图 (TODO List)

> 作为企业级高并发后端服务，所有开发必须遵循：Controller -> Service -> Repository (Dao) 的严格分层架构。

## 阶段一：基础设施搭建 (Infrastructure)

- [X] **项目初始化**: `go mod init` (已完成), 熟悉基本包结构
- [X] **配置中心**: 引入 `spf13/viper`，读取并解析 `config.yaml` 配置文件（如 MySQL, Redis, App 端口号）
- [X] **日志组件**: 引入 `uber-go/zap`，配置异步高性能日志打印与文件切割（用于替代标准库 `log`）
- [X] **数据库连接池**: 引入 `gorm` 和 `mysql` 驱动，初始化 DB 连接并配置最大空闲/活跃连接数
- [X] **Redis 缓存**: 引入 `go-redis/redis`，配置单机/集群模式连接，用于后续的限流和热点数据缓存

## 阶段二：核心框架与中间件 (Core & Middleware)

- [ ] **统一 API 响应格式**: 封装标准化的 JSON 响应体（如 `{ "code": 200, "msg": "success", "data": ... }`）
- [ ] **Gin 引擎初始化**: 搭建 `cmd/server/main.go` 入口，整合配置、日志和路由
- [ ] **全局异常捕获中间件**: 编写 Recovery 中间件，拦截 Panic 并转为 500 JSON 响应，防止程序崩溃
- [ ] **CORS 跨域中间件**: 配置跨域策略，允许前端域名访问
- [ ] **接口文档生成**: 引入 `swaggo/swag`，配置自动生成 Swagger 接口文档

## 阶段三：数据库设计与数据模型 (Database Models)

- [ ] **架构设计文档**: 梳理并完善 `DESIGN.md` 中的实体关系和表结构设计
- [ ] **GORM 模型编写**: 在 `internal/model` 中编写 Go 结构体（User, Post, Comment 等），并配置表名和字段标签
- [ ] **自动迁移工具**: 编写简单的自动建表脚手架脚本或在启动时使用 `AutoMigrate` 生成数据表

## 阶段四：用户认证与鉴权模块 (User & Auth MVP)

- [ ] **密码安全**: 引入 `bcrypt` 包实现密码的单向哈希加密存储
- [ ] **JWT 签发**: 编写 JWT Token 的生成逻辑（包含用户的 UID 和过期时间）
- [ ] **鉴权中间件**: 编写 `JWTAuthMiddleware`，拦截受保护的接口路由，校验 Token 有效性
- [ ] **核心接口开发**:
  - `POST /api/v1/user/register` (用户注册)
  - `POST /api/v1/user/login` (用户登录并返回 Token)
  - `GET /api/v1/user/profile` (获取个人信息 - 需要 Token)

## 阶段五：矿大社区核心业务 (Community Business)

- [ ] **帖子模块**: 发帖、分页查询帖子列表、获取帖子详情
- [ ] **互动模块**: 评论帖子、点赞帖子（考虑使用 Redis 优化高并发点赞）
- [ ] **限流防护**: 编写基于 Redis 的 IP 频率限制中间件，防止接口被恶意刷量
