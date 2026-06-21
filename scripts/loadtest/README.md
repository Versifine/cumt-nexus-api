# Load Test Toolkit

This folder contains a reproducible local load-test workflow for `cumt-nexus-api`.

The goal is to produce a performance baseline that is honest enough for engineering notes or resume evidence:

- fixed environment
- fixed synthetic dataset
- measured HTTP API traffic
- explicit RPS, latency percentiles, and error rate
- JSON and Markdown artifacts under `docs/reports/loadtest/`

## One-Command Local Run

```bash
./scripts/loadtest/run-local-loadtest.sh
```

Default dataset:

| Table | Count |
|---|---:|
| users | 1,000 |
| communities | 50 |
| posts | 20,000 |
| comments | 80,000 |
| post_votes | 120,000 |
| post_saves | 30,000 |
| notifications | 12,000 |
| content_reports | 3,000 |

Default load:

| Setting | Value |
|---|---:|
| API port | 18080 |
| VUs | 50 |
| warmup | 5s |
| measured duration | 60s |

The script uses:

- `compose.yaml` PostgreSQL on localhost `5432`
- database `cumt_nexus_loadtest`
- API on `http://127.0.0.1:18080`
- local object storage root `var/uploads-loadtest`

The seed tool refuses to reset a database whose name does not contain `loadtest`, unless `-unsafe-reset` is explicitly passed to the seed command.

## Custom Run

```bash
./scripts/loadtest/run-local-loadtest.sh \
  --users 2000 \
  --communities 100 \
  --posts 50000 \
  --comments 200000 \
  --vus 100 \
  --duration-seconds 120
```

To isolate the core content path after a known slow endpoint is found:

```bash
./scripts/loadtest/run-local-loadtest.sh \
  --skip-seed \
  --exclude notifications_interactions
```

To run one endpoint at a fixed request issue rate:

```bash
./scripts/loadtest/run-local-loadtest.sh \
  --skip-seed \
  --include search_all \
  --target-rps 20 \
  --vus 80 \
  --duration-seconds 60 \
  --warmup-seconds 10
```

To run a fixed-RPS ladder for one endpoint and summarize the knee point:

```bash
./scripts/loadtest/run-fixed-rps-ladder.sh \
  --skip-seed \
  --endpoint search_all \
  --rps-steps 5,10,15,20,25,30 \
  --step-duration-seconds 60 \
  --warmup-seconds 10 \
  --vus 80
```

## Request Mix

The runner sends a weighted mix across:

- `GET /api/v1/posts?source=all&sort=new`
- `GET /api/v1/posts?source=recommended&sort=hot`
- `GET /api/v1/communities/:slug/posts`
- `GET /api/v1/posts/:id`
- `GET /api/v1/posts/:id/comments?view=tree`
- `GET /api/v1/search`
- `GET /api/v1/notifications`
- `GET /api/v1/admin/mod-queues`
- `PUT /api/v1/posts/:id/vote`

The runner signs local JWTs with the same issuer and secret used by the API process. It does not include login traffic, so bcrypt and login throttling do not dominate the benchmark.

## Manual Steps

If you need to run pieces manually:

```bash
docker compose up -d postgres
go run ./cmd/migrate up
go run ./scripts/loadtest/cmd/seed -reset
go run ./cmd/api
go run ./scripts/loadtest/cmd/runner -base-url http://127.0.0.1:18080 -vus 50 -duration 60s -out-json docs/reports/loadtest/manual.json -out-md docs/reports/loadtest/manual.md
```

Set the same environment variables as `run-local-loadtest.sh` before running manual commands.

## Interpreting Results

Use the generated Markdown for a human-readable report and the JSON for exact aggregate values.

Do not describe these numbers as production capacity. They are local, synthetic, single-node benchmarks intended to expose obvious bottlenecks and provide a repeatable baseline.
