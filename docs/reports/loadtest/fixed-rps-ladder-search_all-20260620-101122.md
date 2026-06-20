# Fixed RPS Ladder Report

- Endpoint: `search_all`
- Step duration: `60s` measured, `10s` warmup
- VUs: `200`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `50 req/s` with p95 `707.79 ms`
- First failing step: target `55 req/s` with p95 `1972.23 ms` and error rate `0.00%`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 45.00 | 43.47 | 2608 | 0.00% | 317.85 ms | 579.62 ms | 743.35 ms | 972.50 ms | True |
| 50.00 | 48.00 | 2880 | 0.00% | 313.45 ms | 707.79 ms | 934.53 ms | 2,139.20 ms | True |
| 55.00 | 46.22 | 2773 | 0.00% | 830.56 ms | 1,972.23 ms | 2,301.65 ms | 3,141.77 ms | False |
| 60.00 | 46.42 | 2785 | 0.00% | 1,695.17 ms | 3,649.62 ms | 4,077.93 ms | 4,941.45 ms | False |

## Raw Reports

- Target `45 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-100602.md`
- Target `50 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-100725.md`
- Target `55 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-100846.md`
- Target `60 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260620-101008.md`

