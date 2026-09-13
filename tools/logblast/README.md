# logblast

A small, dependency-light (libcurl + pthreads only), multi-threaded
synthetic log generator written in C, used specifically for
**performance-overhead testing** — see `backend-go/validator/elastic.go`'s
`ProfileQuery` and the "Performance Overhead Profiling" milestone this
closes.

## Why C, and why a separate tool at all

Every other synthetic-data generator in this repo
(`engine-python/baseline_generator.py`) is Python, on purpose — it needs
precise control over event *content* (specific field values, specific
techniques) for correctness testing, and Python's expressiveness is worth
the overhead there.

Measuring a rule's **query-time overhead** is a different problem: it only
means anything against an index shaped like real production log volume
(tens of millions of documents), and generating that much data with
per-event Python object construction + JSON serialization is too slow to
run in CI. This tool exists purely to generate *volume* cheaply — the
event content is intentionally simple and repetitive, because for this
specific measurement, realistic scale matters far more than content
diversity.

## Build

```bash
sudo apt-get install -y libcurl4-openssl-dev   # if not already present
gcc -O2 -Wall -Wextra -pthread -o logblast logblast.c -lcurl
```

## Usage

```bash
./logblast --es-addr http://localhost:9200 --index wraith-perf-test \
           --events 5000000 --threads 8 --batch-size 2000
```

Output (JSON to stdout, for easy piping into the CI report):

```json
{
  "index": "wraith-perf-test",
  "threads": 8,
  "events_sent": 5000000,
  "http_errors": 0,
  "elapsed_seconds": 21.847,
  "events_per_second": 228915
}
```

Exit code is `0` on success, `2` if any bulk request returned an error or
non-2xx status (check stderr for the first few failures).

## How it fits into the pipeline

Not yet wired into `run_pipeline.py` automatically (that's the natural
next step) — for now, run it manually before profiling a rule:

```bash
# 1. build a realistic-volume index
./tools/logblast/logblast --es-addr http://localhost:9200 \
    --index wraith-baseline-<run-id> --events 5000000 --threads 8

# 2. profile the candidate rule's query against it
cd backend-go && go run . provision --run-id perf-test   # or point at the volume-loaded index directly
```

The natural integration point is a new `--perf-events` flag on
`run_pipeline.py` that shells out to `logblast` before calling
`validator.ProfileQuery`, gating the pipeline on `OverheadPercent` the way
`validate.py` already gates on false-positive rate. Left as an explicit
next step (see `docs/ENTERPRISE.md`) rather than half-wired in, since the
CI runner's available cores/memory determine whether millions-of-events
profiling is practical in a shared GitHub Actions runner vs. a
self-hosted one.

## Verification

Built with `-Wall -Wextra`, zero warnings. End-to-end tested against a
mock Elasticsearch `_bulk` endpoint (a real HTTP server validating that
every document received is well-formed bulk NDJSON with the expected
fields — not just "did the HTTP request succeed"):

| Test | Result |
|---|---|
| 50,000 events, 4 threads, batch size 500 | 100/100 requests received, 50,000/50,000 valid docs, 0 malformed, 0 HTTP errors |
| 2,000,000 events, 8 threads, batch size 2000 | 1,000/1,000 requests received, 2,000,000/2,000,000 valid docs, 0 malformed, 0 HTTP errors, ~229,000 events/sec on a single-vCPU test machine |

One real bug was caught and fixed during this testing: `curl` sends an
`Expect: 100-continue` header for POST bodies over ~1KB by default, and a
naive HTTP server implementation that doesn't explicitly handle that
header will hang indefinitely waiting to send a body the client is
waiting to be told to send. Fixed by explicitly disabling that header
(`Expect:` with no value) — see the comment in `logblast.c` above the
`curl_slist_append(headers, "Expect:")` line.

Not yet tested against a real Elasticsearch instance specifically (the
mock validates bulk *protocol* correctness, not Elasticsearch's actual
indexing behavior under load) — that's a reasonable next step if you're
using this for a real capacity-planning decision rather than relative
before/after overhead comparison.
