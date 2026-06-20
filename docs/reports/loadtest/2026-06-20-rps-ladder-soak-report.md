# 固定 RPS 阶梯与 30 分钟持续压测报告

日期：2026-06-20  
环境：Windows 本机，Docker PostgreSQL，Go API 本地二进制，固定合成数据集

## 测试口径

- 数据集：1,000 用户、50 社区、20,000 帖子、80,000 评论。
- 阶梯测试：单 endpoint 固定请求发出速率，每档预热 10s、统计 60s。
- 健康判定：错误率 <= 1%，p95 <= 2,000 ms，实际完成 RPS >= 目标 RPS 的 95%。
- 持续测试：完整混合场景固定 150 target req/s，预热 30s，统计 1,800s。
- 说明：这里的 RPS 是本机同脚本下的观察值，不代表线上最大容量。

## Search 固定 RPS 阶梯

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 5 | 4.98 | 299 | 0.00% | 217.89 ms | 337.56 ms | 492.02 ms | 563.11 ms | true |
| 10 | 9.97 | 598 | 0.00% | 240.43 ms | 336.45 ms | 429.77 ms | 569.33 ms | true |
| 20 | 19.98 | 1,199 | 0.00% | 207.38 ms | 322.18 ms | 401.70 ms | 507.14 ms | true |
| 30 | 29.90 | 1,794 | 0.00% | 264.92 ms | 469.22 ms | 624.77 ms | 827.61 ms | true |
| 40 | 39.38 | 2,363 | 0.00% | 402.09 ms | 828.28 ms | 1,181.19 ms | 2,700.16 ms | true |
| 45 | 43.47 | 2,608 | 0.00% | 317.85 ms | 579.62 ms | 743.35 ms | 972.50 ms | true |
| 50 | 48.00 | 2,880 | 0.00% | 313.45 ms | 707.79 ms | 934.53 ms | 2,139.20 ms | true |
| 55 | 46.22 | 2,773 | 0.00% | 830.56 ms | 1,972.23 ms | 2,301.65 ms | 3,141.77 ms | false |
| 60 | 46.42 | 2,785 | 0.00% | 1,695.17 ms | 3,649.62 ms | 4,077.93 ms | 4,941.45 ms | false |

结论：`search_all` 的本机单接口拐点落在 50-55 target req/s 区间。50 RPS 仍满足健康规则；55 RPS 开始无法跟上目标发压速率，且 p95 接近 2s；60 RPS 明显排队。

## Hot Feed 固定 RPS 阶梯

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 5 | 4.98 | 299 | 0.00% | 131.62 ms | 184.44 ms | 225.30 ms | 292.51 ms | true |
| 10 | 9.98 | 599 | 0.00% | 148.26 ms | 216.93 ms | 265.08 ms | 331.33 ms | true |
| 20 | 19.95 | 1,197 | 0.00% | 203.05 ms | 296.05 ms | 373.25 ms | 572.48 ms | true |
| 30 | 29.77 | 1,786 | 0.00% | 246.49 ms | 425.67 ms | 573.13 ms | 1,000.94 ms | true |
| 40 | 39.50 | 2,370 | 0.00% | 292.51 ms | 529.77 ms | 691.52 ms | 1,018.46 ms | true |
| 50 | 49.87 | 2,992 | 0.00% | 125.61 ms | 184.35 ms | 226.11 ms | 347.04 ms | true |
| 60 | 59.80 | 3,588 | 0.00% | 144.08 ms | 212.78 ms | 269.85 ms | 447.83 ms | true |
| 70 | 68.93 | 4,136 | 0.00% | 163.40 ms | 279.74 ms | 359.17 ms | 515.17 ms | true |
| 80 | 75.27 | 4,516 | 0.00% | 216.16 ms | 439.63 ms | 567.56 ms | 930.94 ms | false |
| 100 | 79.15 | 4,749 | 0.00% | 481.91 ms | 1,373.57 ms | 1,995.79 ms | 3,359.46 ms | false |

结论：`posts_feed_hot` 的本机单接口拐点落在 70-80 target req/s 区间。70 RPS 仍满足健康规则；80 RPS 开始无法达到目标发压速率；100 RPS 出现明显排队，p95 上升到 1.37s。

## 30 分钟混合持续测试

混合脚本权重总和为 100，其中 `search_all` 和 `posts_feed_hot` 各占 10%。150 target req/s 的混合持续测试约等价于搜索和 Hot Feed 各 15 target req/s，低于各自单接口拐点。

| Metric | Value |
|---|---:|
| Target RPS | 150.00 |
| Actual RPS | 142.81 |
| Requests | 257,062 |
| Failures | 0 |
| Error Rate | 0.00% |
| p50 | 51.90 ms |
| p95 | 344.93 ms |
| p99 | 542.83 ms |
| Max | 2,331.89 ms |
| Duration | 1,800.30 s |

| Endpoint | Requests | Actual RPS | Error Rate | p50 | p95 | p99 | Max |
|---|---:|---:|---:|---:|---:|---:|---:|
| admin_mod_queue | 15,413 | 8.56 | 0.00% | 39.23 ms | 113.01 ms | 192.59 ms | 659.90 ms |
| comment_tree | 35,967 | 19.98 | 0.00% | 23.17 ms | 109.27 ms | 184.51 ms | 972.70 ms |
| community_posts | 35,947 | 19.97 | 0.00% | 37.70 ms | 146.41 ms | 247.93 ms | 1,033.86 ms |
| notifications_interactions | 20,491 | 11.38 | 0.00% | 22.84 ms | 79.47 ms | 137.35 ms | 648.28 ms |
| post_detail | 38,653 | 21.47 | 0.00% | 22.11 ms | 102.55 ms | 176.10 ms | 962.33 ms |
| post_vote_write | 12,832 | 7.13 | 0.00% | 48.12 ms | 220.93 ms | 341.71 ms | 1,000.30 ms |
| posts_feed_hot | 25,884 | 14.38 | 0.00% | 169.25 ms | 358.91 ms | 549.93 ms | 2,012.95 ms |
| posts_feed_new | 46,211 | 25.67 | 0.00% | 68.88 ms | 178.70 ms | 292.74 ms | 1,220.37 ms |
| search_all | 25,664 | 14.26 | 0.00% | 322.46 ms | 622.01 ms | 930.86 ms | 2,331.89 ms |

结论：150 target req/s 的完整混合场景在 30 分钟内保持 0% 错误率，实际完成速率为目标的 95.21%，整体 p95 为 344.93 ms。剩余最重路径仍是 `search_all`，其持续测试 p95 为 622.01 ms。

## 可用于简历的严谨表述

在本机合成数据集上为 Go 社区平台建立性能验证闭环，使用固定 RPS 阶梯测试定位 `search_all` 与 `posts_feed_hot` 的单接口拐点分别约为 50-55 req/s、70-80 req/s；在 150 target req/s 的 30 分钟混合持续压测中完成 25.7 万请求，错误率 0%，整体 p95 约 345 ms。

## 原始报告

- Search 低阶梯：`docs/reports/loadtest/fixed-rps-ladder-search_all-20260619-175959.md`
- Search 边界阶梯：`docs/reports/loadtest/fixed-rps-ladder-search_all-20260620-101122.md`
- Hot Feed 低阶梯：`docs/reports/loadtest/fixed-rps-ladder-posts_feed_hot-20260619-181103.md`
- Hot Feed 高阶梯：`docs/reports/loadtest/fixed-rps-ladder-posts_feed_hot-20260620-095643.md`
- 30 分钟持续测试：`docs/reports/loadtest/loadtest-20260620-111911.md`
