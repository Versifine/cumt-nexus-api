# CUMT Nexus API 压力测试报告

测试日期：2026-06-19  
测试目的：为简历和后续性能优化建立一份可复现的本机压力测试基线，而不是证明线上生产容量。

## 结论摘要

本轮压测先建立可复现基线，再完成通知列表和搜索两轮修复。结论很明确：

1. 原始内容读写核心链路在本机 50 VU 下可稳定运行，错误率 0%，整体约 335.92 RPS，p95 为 541.59 ms。
2. 初始完整混合场景暴露两个主要瓶颈：通知列表 100% 15s 超时，`search_all` p95 约 8.57s 且有少量 15s 超时。
3. 通知列表修复后，`notifications_interactions` 在 1.2 万通知数据下 p95 降到约 145 ms，错误率 0%。
4. 搜索修复后，完整混合场景观察到约 206.43 req/s 的请求完成速率、错误率 0%、整体 p95 916.47 ms；以搜索改动前最近一次完整混合场景为基线，`search_all` p95 从 8669.39 ms 降到 1367.71 ms。
5. 这组数据适合写成“建立压测基线，定位并修复通知/搜索慢查询瓶颈”，不适合包装成“高并发系统”。

## 测试环境

| 项目 | 值 |
|---|---|
| OS | Microsoft Windows 11 家庭版 中文版 |
| CPU | AMD Ryzen 7 7840H, 8 cores / 16 logical processors |
| Memory | 约 31.2 GiB，总测试前空闲约 11.3 GiB |
| Go | go1.25.4 windows/amd64 |
| Docker | Docker 29.2.0, Compose v5.0.2 |
| PostgreSQL | PostgreSQL 16.13, Docker `postgres:16` |
| API | `go run ./cmd/api`, `APP_ENV=test`, `POSTGRES_MAX_CONNS=50` |
| API URL | `http://127.0.0.1:18080` |
| DB | `cumt_nexus_loadtest`, database size about 91 MB |
| Background note | Existing prod-like stack on `:8080` was left running; load-test API used separate `:18080` and separate DB. |

## 数据集

由 `scripts/loadtest/cmd/seed` 直接写入 `cumt_nexus_loadtest`。

| 数据 | 数量 |
|---|---:|
| users | 1,000 |
| communities | 50 |
| posts | 20,000 |
| comments | 80,000 |
| post_votes | 120,000 |
| post_saves | 30,000 |
| notifications | 12,000 |
| content_reports | 3,000 |

Seed summary: `docs/reports/loadtest/seed-summary.json`

## 场景结果

### 场景 A：完整混合场景

包含内容读、viewer-aware 读、搜索、通知、管理审核队列和投票写入。

Raw report: `docs/reports/loadtest/loadtest-20260619-154446.md`

| VUs | Duration | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 50 | 60s | 1,762 | 29.37 | 7.09% | 147.60 ms | 15000.31 ms | 15001.54 ms |

主要问题：

- `notifications_interactions`: 123/123 全部超时，p50 约 15s。
- `search_all`: p50 6156.56 ms，p95 9205.01 ms，2 次超时。
- 其他内容核心接口均为 0 错误。

### 场景 B：排除通知列表

用于确认通知列表是否单独拖垮整体结果。

Raw report: `docs/reports/loadtest/loadtest-20260619-155002.md`

| VUs | Duration | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 50 | 60s | 3,893 | 64.88 | 0.03% | 86.88 ms | 5695.21 ms | 7991.83 ms |

主要问题：

- 排除通知后吞吐提升，但 `search_all` 仍然明显偏慢。
- `search_all`: p50 5487.05 ms，p95 8566.44 ms，1 次超时。

### 场景 C：内容核心场景

排除 `notifications_interactions` 和 `search_all`，保留帖子流、社区帖子、帖子详情、评论树、投票写入和管理审核队列。

Raw report: `docs/reports/loadtest/loadtest-20260619-155154.md`

| VUs | Duration | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 50 | 60s | 20,155 | 335.92 | 0.00% | 93.33 ms | 541.59 ms | 816.25 ms |

核心接口拆分：

| Endpoint | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|
| `post_detail` | 3,680 | 61.33 | 0.00% | 59.06 ms | 114.51 ms | 141.95 ms |
| `comment_tree` | 3,418 | 56.97 | 0.00% | 65.58 ms | 123.36 ms | 154.08 ms |
| `community_posts` | 3,543 | 59.05 | 0.00% | 79.90 ms | 139.50 ms | 175.82 ms |
| `posts_feed_new` | 4,434 | 73.90 | 0.00% | 138.67 ms | 240.74 ms | 298.67 ms |
| `post_vote_write` | 1,236 | 20.60 | 0.00% | 114.46 ms | 208.06 ms | 264.60 ms |
| `admin_mod_queue` | 1,484 | 24.73 | 0.00% | 95.51 ms | 167.77 ms | 212.34 ms |
| `posts_feed_hot` | 2,360 | 39.33 | 0.00% | 510.77 ms | 878.08 ms | 1042.30 ms |

## 瓶颈分析

### 通知列表

`notifications_interactions` 在当前数据集下全部 15s 超时。对仓储 SQL 做 `EXPLAIN` 后可见：

- 计划成本约 `11554537`。
- 对 `comments` 做 8 万行 Seq Scan。
- 对 `posts` 做 2 万行 Seq Scan + Materialize。
- `notifications.source_type/source_id` 与 `comments.id::text`、`posts.id::text` 的 join 形态不利于索引。
- `status=all` 场景没有直接匹配当前 `recipient + type + created_at` 读取路径的索引。

本轮采用的优化方向：

- 给通知列表增加面向 `recipient_id + type/category + created_at DESC` 的索引。
- 避免在列表页对完整候选集做上下文 join，先在 `notifications` 内完成过滤、排序和分页。
- 分离 `post` 和 `comment` source 的 join，避免 OR join 造成高成本计划。
- 仅给当前页补充 actor、comment、post、community 上下文。

### 搜索

`search_all` 在当前数据集下 p95 达到 8.57s。当前搜索 SQL 同时做 full-text、前缀/包含匹配和排序，且会对帖子标题、正文、社区名等字段做组合评分。

初始测试中搜索 p95 约在 8.6s 到 9.2s 之间波动。本文的搜索修复前后对比统一使用搜索改动前最近一次完整混合场景作为基线，即 `8669.39 ms -> 1367.71 ms`。

本轮采用的优化方向：

- 将搜索文档落成 `tsvector` 生成列或实体列，避免查询时重复计算。
- 为 stored `tsvector` 搜索文档增加 GIN 索引。
- 搜索 SQL 直接使用生成列做全文匹配和评分，减少查询时逐行构造文档的开销。
- 对安全的单 token ASCII 查询增加 prefix tsquery，改善 `notification` 与 `notifications` 这类词形差异。

仍可继续优化：

- 对中文/子串搜索引入 `pg_trgm` 索引，进一步降低 contains fallback 成本。
- 对 `scope=all` 做更明确的分区候选集裁剪，避免一次请求同时重查 posts/communities/users。

### Feed Hot

`posts_feed_hot` 在核心场景下 p95 为 878.08 ms，明显慢于 `posts_feed_new`。原因符合预期：hot 排序需要聚合投票、评论和收藏统计。

后续优化方向：

- 对热门分数做短 TTL 缓存或物化统计。
- 将帖子计数、分数等热字段做增量维护，避免每次列表读取做聚合。
- 区分首页热榜和社区内新帖流，不让所有列表都走复杂 ranking。

## 复现命令

首次完整压测：

```bash
./scripts/loadtest/run-local-loadtest.sh \
  --users 1000 \
  --communities 50 \
  --posts 20000 \
  --comments 80000 \
  --post-votes 120000 \
  --post-saves 30000 \
  --notifications 12000 \
  --reports 3000 \
  --vus 50 \
  --duration-seconds 60 \
  --warmup-seconds 5
```

内容核心场景：

```bash
./scripts/loadtest/run-local-loadtest.sh \
  --skip-seed \
  --users 1000 \
  --communities 50 \
  --posts 20000 \
  --comments 80000 \
  --post-votes 120000 \
  --post-saves 30000 \
  --notifications 12000 \
  --reports 3000 \
  --vus 50 \
  --duration-seconds 60 \
  --warmup-seconds 5 \
  --exclude notifications_interactions,search_all
```

## 简历可用表述

建议写：

```text
为 Go 社区平台建立可复现的本机压测基线，基于 1k 用户、2 万帖子、8 万评论的合成数据集，对内容流、搜索、通知及互动写入等 API 进行 50 VU 混合负载测试。
```

```text
根据执行计划重构通知列表分页与关联查询，将其从 100% 触发 15 秒超时优化至 p95 145 ms；通过 stored tsvector 与 GIN 索引将全局搜索 p95 从 8.67 秒降至 1.37 秒，最终混合场景错误率为 0%、整体 p95 约 916 ms。
```

不要写：

```text
支持高并发 / 生产级高可用 / 万级 QPS
```

本轮压测是本机单节点、合成数据、短时压力测试，只能证明项目已经具备性能基线和瓶颈定位流程。

## 修复后复测：通知列表瓶颈

修复日期：2026-06-19  
修复范围：`GET /api/v1/notifications?category=interactions&status=all`

本次修复没有扩大业务语义，只改通知列表的读取路径：

- 列表查询先在 `notifications` 内按 `recipient_id/category/status/created_at/id` 完成排序分页，再只给当前页补充 actor、comment、post、community 上下文。
- 拆掉原列表查询中的 `posts` OR join，改成 `direct_post` 与 `comment_post` 两条路径。
- 新增 `notifications_recipient_created_idx` 和 `notifications_recipient_type_created_idx`，覆盖 `status=all` 与 category/type 过滤下的分页读取。
- 顺手修复压测脚本在 `-Exclude` 为空时可能吞掉 `-out-json/-out-md` 的参数传递问题。

Raw report: `docs/reports/loadtest/loadtest-20260619-161102.md`

| 场景 | VUs | Duration | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 修复后完整混合场景 | 50 | 60s | 4,697 | 78.28 | 0.04% | 80.83 ms | 5291.73 ms | 7878.35 ms |

通知接口修复前后对比：

| Endpoint | 修复前 | 修复后 |
|---|---:|---:|
| `notifications_interactions` error rate | 100.00% | 0.00% |
| `notifications_interactions` p50 | about 15000 ms | 51.90 ms |
| `notifications_interactions` p95 | about 15000 ms | 145.08 ms |
| `notifications_interactions` max | about 15000 ms | 276.65 ms |

整体混合场景对比：

| 指标 | 修复前完整混合场景 | 修复后完整混合场景 |
|---|---:|---:|
| Requests | 1,762 | 4,697 |
| RPS | 29.37 | 78.28 |
| Error Rate | 7.09% | 0.04% |
| Overall p95 | 15000.31 ms | 5291.73 ms |

复测结论：

- 通知列表瓶颈已经消除：`notifications_interactions` 在 50 VU、1.2 万通知数据下 p95 降到 145.08 ms，0 错误。
- 该阶段完整混合场景仍不是“核心链路 335 RPS”的水平，因为 `search_all` 仍然是当时主瓶颈：p50 5626.86 ms、p95 8669.39 ms、2 次 15s 超时。

## 修复后复测：搜索瓶颈

修复日期：2026-06-19  
修复范围：`GET /api/v1/search?scope=all`

本次修复保持搜索接口和响应结构不变，改的是搜索仓储的数据访问路径：

- 在 `posts` 和 `communities` 上新增 stored `search_document` 生成列，提前落地 `tsvector` 文档。
- 为两个生成列新增 GIN 索引，避免每次搜索逐行重算 `to_tsvector`。
- 搜索 SQL 改为直接使用 `search_document` 做全文匹配和 `ts_rank_cd` 排序。
- 对安全的单 token ASCII 查询增加 prefix tsquery，例如 `notification` 可命中 `notifications`。

Raw report: `docs/reports/loadtest/loadtest-20260619-162130.md`

| 场景 | VUs | Duration | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 搜索修复后完整混合场景 | 50 | 60s | 12,386 | 206.43 | 0.00% | 122.14 ms | 916.47 ms | 1253.05 ms |

这里的 206.43 RPS 表示本轮相同混合场景下观察到的请求完成速率，不代表系统最大容量。

本轮复测用于证明同一脚本、同一数据规模下的优化闭环。由于后续复测发生在前序测试之后，PostgreSQL 与操作系统缓存可能已被预热，因此这些数字不应被解释为冷启动容量结论。若要进一步增强对比可信度，建议使用固定数据库快照，关闭其他本地 stack，分别重启 PostgreSQL 和 API，统一预热 30s 后，对优化前/优化后各跑 3 次并取中位数。

最终 endpoint 明细：

| Endpoint | Requests | RPS | Error Rate | p50 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|
| `notifications_interactions` | 878 | 14.63 | 0.00% | 63.74 ms | 148.89 ms | 193.30 ms |
| `search_all` | 1,245 | 20.75 | 0.00% | 771.11 ms | 1367.71 ms | 1616.74 ms |
| `posts_feed_hot` | 1,287 | 21.45 | 0.00% | 644.94 ms | 1104.01 ms | 1337.26 ms |
| `posts_feed_new` | 2,301 | 38.35 | 0.00% | 179.83 ms | 320.56 ms | 411.55 ms |
| `post_detail` | 1,826 | 30.43 | 0.00% | 62.67 ms | 146.14 ms | 213.63 ms |
| `comment_tree` | 1,781 | 29.68 | 0.00% | 68.78 ms | 153.95 ms | 213.43 ms |
| `community_posts` | 1,714 | 28.57 | 0.00% | 101.15 ms | 195.42 ms | 251.53 ms |
| `post_vote_write` | 626 | 10.43 | 0.00% | 126.60 ms | 277.13 ms | 416.93 ms |
| `admin_mod_queue` | 728 | 12.13 | 0.00% | 116.31 ms | 212.90 ms | 256.13 ms |

搜索接口修复前后对比：

| Endpoint | 修复前 | 修复后 |
|---|---:|---:|
| `search_all` error rate | 0.48% | 0.00% |
| `search_all` p50 | 5626.86 ms | 771.11 ms |
| `search_all` p95 | 8669.39 ms | 1367.71 ms |
| `search_all` max | 15000.41 ms | 1853.21 ms |

完整混合场景最终对比：

| 指标 | 初始完整混合场景 | 通知修复后 | 搜索修复后 |
|---|---:|---:|---:|
| Requests | 1,762 | 4,697 | 12,386 |
| RPS | 29.37 | 78.28 | 206.43 |
| Error Rate | 7.09% | 0.04% | 0.00% |
| Overall p95 | 15000.31 ms | 5291.73 ms | 916.47 ms |

最终复测结论：

- 通知超时问题已解决，搜索慢查询得到显著优化并消除超时；完整混合场景 50 VU 下无 4xx/5xx/客户端超时错误。
- 按 p95 看，当前剩余主要慢路径为 `search_all` 和 `posts_feed_hot`，分别约 1367.71 ms 和 1104.01 ms；前者仍有进一步优化空间。
- 排除搜索类重查询后，内容核心链路中最慢接口为 `posts_feed_hot`；这是热度排序聚合类查询，属于下一阶段独立优化目标。
