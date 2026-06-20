# Fixed RPS Ladder Report

- Endpoint: `search_all`
- Step duration: `60s` measured, `10s` warmup
- VUs: `100`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `40 req/s` with p95 `828.28 ms`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 5.00 | 4.98 | 299 | 0.00% | 217.89 ms | 337.56 ms | 492.02 ms | 684.47 ms | True |
| 10.00 | 9.97 | 598 | 0.00% | 240.43 ms | 336.45 ms | 429.77 ms | 486.84 ms | True |
| 15.00 | 14.97 | 898 | 0.00% | 186.74 ms | 264.76 ms | 334.53 ms | 432.39 ms | True |
| 20.00 | 19.98 | 1199 | 0.00% | 207.38 ms | 322.18 ms | 401.70 ms | 516.23 ms | True |
| 25.00 | 24.92 | 1495 | 0.00% | 236.13 ms | 393.94 ms | 502.76 ms | 656.22 ms | True |
| 30.00 | 29.90 | 1794 | 0.00% | 264.92 ms | 469.22 ms | 624.77 ms | 767.01 ms | True |
| 35.00 | 34.78 | 2087 | 0.00% | 313.78 ms | 663.99 ms | 933.49 ms | 1,617.78 ms | True |
| 40.00 | 39.38 | 2363 | 0.00% | 402.09 ms | 828.28 ms | 1,181.19 ms | 2,700.16 ms | True |

## Raw Reports

- Target `5 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174931.md`
- Target `10 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175052.md`
- Target `15 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175213.md`
- Target `20 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175332.md`
- Target `25 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175450.md`
- Target `30 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175608.md`
- Target `35 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175727.md`
- Target `40 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-175845.md`

