# Fixed RPS Ladder Report

- Endpoint: `search_all`
- Step duration: `3s` measured, `1s` warmup
- VUs: `5`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `1 req/s` with p95 `189.91 ms`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 1.00 | 1.00 | 3 | 0.00% | 140.11 ms | 189.91 ms | 194.34 ms | 195.45 ms | True |

## Raw Reports

- Target `1 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-173920.md`

