# CUMT-Nexus 核心设计与数据字典 (DESIGN)

本文档用于指导后端的实际开发，包含数据库结构设计与核心开发规范。

## 1. 数据库表结构设计 (MySQL)

### 1.1 用户表 (`users`)

记录基本账号信息。

| 字段名            | 数据类型 (MySQL) | Go 结构体类型 | 说明 / 约束                          |
| :---------------- | :--------------- | :------------ | :----------------------------------- |
| `id`            | BIGINT           | `uint`      | 主键，自增                           |
| `username`      | VARCHAR(64)      | `string`    | 用户名，**唯一索引**，用于登录 |
| `password_hash` | VARCHAR(255)     | `string`    | 密码的哈希值（绝对不能存明文！）     |
| `nickname`      | VARCHAR(64)      | `string`    | 社区昵称，默认可以与用户名相同       |
| `avatar_url`    | VARCHAR(255)     | `string`    | 头像的链接地址                       |
| `role`          | TINYINT          | `int`       | 权限角色：0-普通用户, 1-管理员       |
| `status`        | TINYINT          | `int`       | 账号状态：1-正常, 2-封禁             |
| `created_at`    | DATETIME         | `time.Time` | 注册时间                             |
| `updated_at`    | DATETIME         | `time.Time` | 最后修改时间                         |

### 1.2 帖子表 (`posts`)

记录社区内的讨论帖子。

| 字段名         | 数据类型 (MySQL) | Go 结构体类型 | 说明 / 约束                        |
| :------------- | :--------------- | :------------ | :--------------------------------- |
| `id`         | BIGINT           | `uint`      | 主键，自增                         |
| `user_id`    | BIGINT           | `uint`      | 作者 ID，关联 `users.id`         |
| `title`      | VARCHAR(128)     | `string`    | 帖子标题                           |
| `content`    | TEXT             | `string`    | 帖子正文（支持 Markdown 或富文本） |
| `view_count` | INT              | `int`       | 浏览量，默认 0                     |
| `created_at` | DATETIME         | `time.Time` | 发布时间                           |
| `updated_at` | DATETIME         | `time.Time` | 最后修改时间                       |

### 1.3 评论表 (`comments`)

记录帖子下的用户评论。

| 字段名         | 数据类型 (MySQL) | Go 结构体类型      | 说明 / 约束                     |
| :------------- | :--------------- | :----------------- | :------------------------------ |
| `id`         | BIGINT           | `uint`           | 主键，自增                      |
| `post_id`    | BIGINT           | `uint`           | 所属帖子 ID，关联 `posts.id`  |
| `user_id`    | BIGINT           | `uint`           | 评论者 ID，关联 `users.id`    |
| `parent_id`  | BIGINT           | `uint`           | 父评论 ID，顶级评论为 0 或 NULL |
| `content`    | TEXT             | `string`         | 评论内容                        |
| `created_at` | DATETIME         | `time.Time`      | 评论时间                        |
| `updated_at` | DATETIME         | `time.Time`      | 最后修改时间                    |
| `deleted_at` | DATETIME         | `gorm.DeletedAt` | 逻辑删除标志                    |

### 1.4 点赞表 (`likes`)

记录用户对帖子或评论的点赞。

| 字段名          | 数据类型 (MySQL) | Go 结构体类型 | 说明 / 约束                  |
| :-------------- | :--------------- | :------------ | :--------------------------- |
| `id`          | BIGINT           | `uint`      | 主键，自增                   |
| `user_id`     | BIGINT           | `uint`      | 点赞者 ID，关联 `users.id` |
| `target_id`   | BIGINT           | `uint`      | 目标 ID (帖子 ID 或 评论 ID) |
| `target_type` | TINYINT          | `int`       | 目标类型：1-帖子, 2-评论     |
| `created_at`  | DATETIME         | `time.Time` | 点赞时间                     |

*(注意：需要建立联合唯一索引 `idx_user_target` (`user_id`, `target_id`, `target_type`)，防止重复点赞)*

---

## 2. 统一 API 响应规范与全局错误码字典

在前后端分离架构中，为了降低沟通成本、规范前端解析逻辑，本项目所有 RESTful API 的响应体 (Response Body) 必须严格遵循以下 JSON 结构。

### 2.1 基础数据结构

所有的 API 接口，无论是成功还是失败，无论是 GET 请求还是 POST 请求，其最外层必须是包含以下三个字段的 JSON 对象。
**注意：HTTP 状态码我们统一返回 200 OK，真正的业务逻辑成败是由 JSON 体里的 `code` 决定的。**

```json
{
  "code": 0,            // 业务状态码，0 代表完全成功，非 0 代表各种错误
  "msg": "success",     // 提示信息，失败时通常显示给用户看的原因
  "data": null          // 实际承载的业务数据
}
```

#### 典型返回示例：

**请求成功 (返回分页列表)**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 150,   
    "list": [
      { "id": 1, "title": "教二楼自习占座指南" },
      { "id": 2, "title": "南湖一食堂新菜品评测" }
    ]
  }
}
```

**请求失败 (参数错误)**

```json
{
  "code": 10001,
  "msg": "学号必须是纯数字且长度为8位",
  "data": null
}
```

### 2.2 全局业务状态码字典 (`code`)

* `0`: 请求成功 (Success)

**1xxxx: 客户端/参数级别错误**

* `10001`: 参数错误/校验失败 (Invalid Parameter)
* `10002`: 缺少必填参数 (Missing Parameter)

**2xxxx: 用户业务逻辑错误**

* `20001`: 用户不存在 (User Not Found)
* `20002`: 密码错误 (Wrong Password)
* `20003`: 用户名已存在 (Username Already Exists)
* `20004`: 用户被封禁 (User Banned)

**3xxxx: 鉴权与安全级别错误**

* `30001`: 未登录或 Token 无效/过期 (Unauthorized / Invalid Token)
* `30002`: 权限不足，无法访问该资源 (Forbidden)

**4xxxx: 业务资源级别错误**

* `40001`: 请求的资源不存在 (Resource Not Found)

**5xxxx: 服务器底层错误**

* `50000`: 服务器内部错误 (Internal Server Error)

## 3. 核心业务流程图

由于是文字文档，这里用步骤来描述核心业务流：

### 3.1 用户注册与登录

* **注册**：客户端提交 用户名 + 密码 -> Controller 校验格式（必填、长度） -> Service 校验用户名是否重复 -> bcrypt 库加密密码 -> 存入数据库 `users` 表 -> 返回注册成功。
* **登录**：客户端提交 用户名 + 密码 -> Controller 校验格式 -> Service 查数据库获取用户信息 -> bcrypt 对比密码 -> 成功则生成 JWT Token 并下发 -> 客户端保存 Token 并在每次请求头上带上 `Authorization: Bearer <token>`。

### 3.2 发帖与查看

* **发帖**：客户端带 Token 提交标题与内容 -> 中间件校验 Token 获取 user_id -> Service 创建记录存入 `posts` 表 -> 返回发帖成功。
* **查看帖子列表**：客户端无 Token 也可以请求列表，支持分页 (offset, limit) -> 获取帖子集合，需要关联查询作者的基本信息（头像、昵称）。

## 4. 权限与角色控制 (RBAC)

系统目前采用简单的基于角色的访问控制（RBAC）：

* **角色 (Role)**：在 `users` 表中用 `role` 字段表示。
  * `0`: 普通用户 (Normal User) - 可浏览、发帖、评论、点赞。
  * `1`: 管理员 (Admin) - 包含普通用户所有权限，且可以删除违规帖子/评论、封禁用户。
* **实现方案**：
  * 使用 Gin Auth 中间件验证 JWT Token，并将解析后的用户信息（如 ID、Role）放入 `gin.Context` 中。
  * 对于需要管理员权限的接口（如 `/api/v1/admin/*`），额外增加 Admin 中间件，从上下文中提取用户信息判断 `role` 字段。
  * 若 `role != 1`，则拦截并返回错误码 `30002 Forbidden`。

## 5. 第三方依赖与基础设施

* **Web 框架**: Gin (`github.com/gin-gonic/gin`)
* **ORM**: GORM (`gorm.io/gorm`, `gorm.io/driver/mysql`)
* **数据库**: MySQL 8.x
* **缓存/NoSQL (可选)**: Redis (用于缓存热点数据、限制访问频率、存储点赞关系等，后期可引入)
* **JWT**: `github.com/golang-jwt/jwt/v5`
* **密码哈希**: `golang.org/x/crypto/bcrypt`
* **配置管理**: Viper (`github.com/spf13/viper`)
* **日志系统**: Zap (`go.uber.org/zap`) 与 Lumberjack (`gopkg.in/natefinch/lumberjack.v2` 解决日志文件切割)

## 6. 后端开发核心规范

1. **密码安全：** 必须使用 `golang.org/x/crypto/bcrypt` 库对密码进行哈希加密，无论如何不允许在数据库中裸奔明文密码。
2. **逻辑删除：** 对于用户删除帖子，尽量使用 GORM 的 `DeletedAt` 软删除（逻辑删除），而不是真的从数据库 `DELETE` 掉，方便后期数据分析或恢复。
3. **分层原则：**
   * **Controller 层**：只负责解析前端传来的 JSON，验证参数（比如用户名格式对不对），然后调用 Service。
   * **Service 层**：写具体的业务逻辑（比如查数据库看用户名存不存在，对比密码，生成 Token）。
   * **Repository/Dao 层**：专门负责封装对数据库（MySQL/Redis）的访问与增删改查操作（通过 GORM 等），以复用代码和解耦。
   * **Model 层**：定义数据库数据模型（实体类），如 `User`, `Post` 等。
4. **日志记录：** 所有非预期的异常错误、接口请求入参及其重要状态变更（如登录、发帖、权限更改等），都应统一使用 Zap 记录，便于追踪问题。
5. **统一响应：** 所有的返回结果都应封装在一个统一的响应体中（包含 `code`, `msg`, `data` 字段），禁止随意返回不一致的结构。
6. **参数校验：** 建议使用 Gin 自带的 `binding` 与 `validator` 标签进行自动校验，减少在 Controller 里手工编写大量 `if-else` 的繁琐校验代码。
