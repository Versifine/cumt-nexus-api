# Fixed RPS Ladder Report

- Endpoint: `posts_feed_hot`
- Step duration: `60s` measured, `10s` warmup
- VUs: `100`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `40 req/s` with p95 `529.77 ms`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 5.00 | 4.98 | 299 | 0.00% | 131.62 ms | 184.44 ms | 225.30 ms | 245.52 ms | True |
| 10.00 | 9.98 | 599 | 0.00% | 148.26 ms | 216.93 ms | 265.08 ms | 376.56 ms | True |
| 15.00 | 15.00 | 900 | 0.00% | 154.60 ms | 238.55 ms | 300.48 ms | 376.82 ms | True |
| 20.00 | 19.95 | 1197 | 0.00% | 203.05 ms | 296.05 ms | 373.25 ms | 432.86 ms | True |
| 25.00 | 24.87 | 1492 | 0.00% | 247.43 ms | 414.52 ms | 608.75 ms | 947.04 ms | True |
| 30.00 | 29.77 | 1786 | 0.00% | 246.49 ms | 425.67 ms | 573.13 ms | 843.22 ms | True |
| 35.00 | 34.67 | 2080 | 0.00% | 250.38 ms | 506.54 ms | 719.88 ms | 1,315.77 ms | True |
| 40.00 | 39.50 | 2370 | 0.00% | 292.51 ms | 529.77 ms | 691.52 ms | 871.59 ms | True |

## Raw Reports

- Target `5 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180031.md`
- Target `10 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180150.md`
- Target `15 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180309.md`
- Target `20 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180428.md`
- Target `25 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180548.md`
- Target `30 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180709.md`
- Target `35 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180828.md`
- Target `40 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-180950.md`

