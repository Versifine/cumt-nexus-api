# CUMT-Nexus API 

> 中国矿业大学校园社区枢纽 - 核心后端服务 (Backend Service)

CUMT-Nexus 是一个致力于连接矿大同学、信息与服务的现代化校园社区平台。本项目为 Nexus 架构的后端 API 服务，采用高并发、轻量级的 Go 语言构建，提供纯净的 RESTful 接口支持。

## 核心技术栈

本项目坚持“稳定、成熟、工程化”的选型原则：

* **核心语言:** Go 1.21+
* **Web 框架:** [Gin](https://github.com/gin-gonic/gin) (高性能路由与中间件)
* **关系型数据库:** MySQL 8.0
* **ORM 框架:** [GORM](https://gorm.io/gorm) (数据持久化)
* **高性能缓存:** [Redis](https://github.com/redis/go-redis) (热点数据与限流)
* **身份鉴权:** JWT (JSON Web Token)
* **基础工程组件:** * 配置管理: [Viper](https://github.com/spf13/viper)
  * 日志追踪: [Zap](https://go.uber.org/zap)
  * 接口文档: [Swaggo](https://github.com/swaggo/swag) (代码即文档)

## 标准工程目录 (Standard Go Layout)

```text
cumt-nexus-api/
├── cmd/
│   └── server/       # 程序的唯一入口，main.go 所在位置
├── internal/         # 私有应用与库代码（不对外暴露）
│   ├── config/       # Viper 配置文件解析与结构体
│   ├── controller/   # API 路由处理函数 (解析入参、调用 service、返回 JSON)
│   ├── middleware/   # Gin 中间件 (CORS 跨域、JWT 鉴权、日志拦截)
│   ├── model/        # 数据库模型 (GORM structs)
│   ├── repository/   # 数据库访问层 (Dao)，封装所有 GORM 增删改查
│   ├── router/       # 集中化 API 路由注册
│   └── service/      # 核心业务逻辑实现
├── pkg/              # 可以被外部项目引用的公共工具包 (如密码加密、响应封装)
├── docs/             # Swaggo 自动生成的 API 文档存放处
├── .env.example      # 环境变量配置示例 (不包含敏感密码)
├── go.mod            # Go 依赖管理
└── README.md         # 项目总览
```
