# Fixed RPS Ladder Report

- Endpoint: `search_all`
- Step duration: `60s` measured, `10s` warmup
- VUs: `80`
- Healthy rule: error rate <= `1.00%`, p95 <= `2000 ms`, actual RPS >= `95%` of target
- Last healthy step: target `20 req/s` with p95 `194.18 ms`
- First failing step: target `25 req/s` with p95 `208.42 ms` and error rate `0.00%`

| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |
|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 5.00 | 4.98 | 299 | 0.00% | 149.74 ms | 179.72 ms | 206.92 ms | 241.96 ms | True |
| 10.00 | 9.97 | 598 | 0.00% | 150.13 ms | 174.85 ms | 196.09 ms | 224.49 ms | True |
| 15.00 | 14.97 | 898 | 0.00% | 152.83 ms | 190.87 ms | 221.76 ms | 281.80 ms | True |
| 20.00 | 19.97 | 1198 | 0.00% | 156.81 ms | 194.18 ms | 231.73 ms | 288.59 ms | True |
| 25.00 | 21.33 | 1280 | 0.00% | 161.08 ms | 208.42 ms | 247.83 ms | 278.22 ms | False |
| 30.00 | 21.33 | 1280 | 0.00% | 178.51 ms | 250.69 ms | 302.05 ms | 340.92 ms | False |
| 35.00 | 21.33 | 1280 | 0.00% | 307.53 ms | 568.34 ms | 729.62 ms | 867.25 ms | False |

## Raw Reports

- Target `5 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-173941.md`
- Target `10 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174057.md`
- Target `15 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174213.md`
- Target `20 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174330.md`
- Target `25 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174447.md`
- Target `30 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174603.md`
- Target `35 req/s`: `D:\Projects\cumt-nexus-api\docs\reports\loadtest\loadtest-20260619-174720.md`

