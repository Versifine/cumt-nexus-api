# Fixed RPS Ladder Report

- Endpoint: `posts_feed_hot`
- Step duration: `60s` measured, `10s` warmup
- VUs: `200`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `70 req/s` with p95 `279.74 ms`
- First failing step: target `80 req/s` with p95 `439.63 ms` and error rate `0.00%`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 50.00 | 49.87 | 2992 | 0.00% | 125.61 ms | 184.35 ms | 226.11 ms | 347.04 ms | True |
| 60.00 | 59.80 | 3588 | 0.00% | 144.08 ms | 212.78 ms | 269.85 ms | 447.83 ms | True |
| 70.00 | 68.93 | 4136 | 0.00% | 163.41 ms | 279.74 ms | 359.17 ms | 515.17 ms | True |
| 80.00 | 75.27 | 4516 | 0.00% | 216.16 ms | 439.63 ms | 567.56 ms | 930.94 ms | False |
| 100.00 | 79.15 | 4749 | 0.00% | 481.91 ms | 1,373.57 ms | 1,995.79 ms | 3,359.46 ms | False |

## Raw Reports

- Target `50 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-095010.md`
- Target `60 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-095129.md`
- Target `70 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-095250.md`
- Target `80 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-095411.md`
- Target `100 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-095531.md`

