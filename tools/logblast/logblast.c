/*
 * logblast.c — high-throughput synthetic log generator for WRAITH's
 * performance-overhead testing stage.
 *
 * WHY THIS EXISTS: backend-go/validator/elastic.go's ProfileQuery measures
 * how much CPU/latency overhead a candidate detection rule adds to the
 * SIEM's indexers. That measurement is only meaningful against a
 * realistically-sized index — a few thousand synthetic events (what
 * engine-python/baseline_generator.py produces via Faker, fine for
 * correctness/false-positive testing) is nowhere near production log
 * volume, and Python's per-event object construction + JSON encoding
 * overhead makes generating tens of millions of events impractically
 * slow for a CI pipeline.
 *
 * This tool exists specifically to close that gap: a small, dependency-
 * light (only libcurl + pthreads, both ubiquitous), multi-threaded NDJSON
 * bulk-indexer that can push hundreds of thousands of events per second
 * on a single machine, so a rule's query-time overhead can be measured
 * against an index shaped like a real production SIEM before it ships.
 *
 * This is NOT used for correctness testing (that stays in Python/Go,
 * where the event content needs to be precisely controlled — see
 * baseline_generator.py and attack_simulator.py). It exists purely to
 * generate REALISTIC VOLUME cheaply.
 *
 * Build:
 *   gcc -O2 -pthread -o logblast logblast.c -lcurl
 *
 * Usage:
 *   ./logblast --es-addr http://localhost:9200 --index wraith-perf-test \
 *              --events 5000000 --threads 8 --batch-size 2000
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#include <pthread.h>
#include <curl/curl.h>

/* ---- fast, deterministic-per-seed PRNG (xorshift64*) ----
 * Not cryptographic — we want speed and a repeatable-per-thread stream,
 * not security, for synthetic log field selection. */
typedef struct { uint64_t state; } rng_t;

static void rng_seed(rng_t *r, uint64_t seed) {
    r->state = seed ? seed : 0x9E3779B97F4A7C15ULL;
}

static uint64_t rng_next(rng_t *r) {
    uint64_t x = r->state;
    x ^= x >> 12;
    x ^= x << 25;
    x ^= x >> 27;
    r->state = x;
    return x * 0x2545F4914F6CDD1DULL;
}

static int rng_index(rng_t *r, int n) {
    return (int)(rng_next(r) % (uint64_t)n);
}

/* ---- synthetic field pools (benign enterprise telemetry, same spirit
 * as engine-python/baseline_generator.py, kept intentionally small and
 * hand-written rather than pulled from a data file — this tool has no
 * dependencies beyond libcurl on purpose) ---- */
static const char *PROCESS_NAMES[] = {
    "chrome.exe", "outlook.exe", "teams.exe", "excel.exe", "winword.exe",
    "explorer.exe", "svchost.exe", "code.exe", "slack.exe", "onedrive.exe",
};
#define N_PROCESS_NAMES (sizeof(PROCESS_NAMES) / sizeof(PROCESS_NAMES[0]))

static const char *EVENT_TYPES[] = {
    "process_creation", "network_connection", "authentication", "file_modification",
};
#define N_EVENT_TYPES (sizeof(EVENT_TYPES) / sizeof(EVENT_TYPES[0]))

#define N_SYNTHETIC_USERS 200
#define N_SYNTHETIC_HOSTS 300

typedef struct {
    int thread_id;
    const char *es_addr;
    const char *index_name;
    long events_to_send;
    int batch_size;
    long events_sent;      /* output */
    long http_errors;      /* output */
    double elapsed_seconds;/* output */
} worker_args_t;

struct curl_response_buf {
    char *data;
    size_t len;
};

static size_t discard_response(void *ptr, size_t size, size_t nmemb, void *userdata) {
    (void)ptr; (void)userdata;
    return size * nmemb; /* we don't care about response bodies, just status */
}

/* Appends one bulk-format action+doc pair (two NDJSON lines) to buf.
 * Returns the number of bytes written, or -1 if it wouldn't fit. */
static int append_bulk_doc(char *buf, size_t remaining, const char *index_name,
                            rng_t *rng, long seq, uint64_t epoch_ms_base) {
    const char *proc = PROCESS_NAMES[rng_index(rng, (int)N_PROCESS_NAMES)];
    const char *etype = EVENT_TYPES[rng_index(rng, (int)N_EVENT_TYPES)];
    int user_n = rng_index(rng, N_SYNTHETIC_USERS);
    int host_n = rng_index(rng, N_SYNTHETIC_HOSTS);
    uint64_t ts_ms = epoch_ms_base + (uint64_t)(rng_next(rng) % 86400000ULL);

    int written = snprintf(buf, remaining,
        "{\"index\":{\"_index\":\"%s\"}}\n"
        "{\"@timestamp\":%llu,\"event_type\":\"%s\",\"process_name\":\"%s\","
        "\"user\":\"synthetic-user-%d\",\"host\":\"WKS-PERF-%d\","
        "\"wraith_synthetic\":true,\"wraith_label\":\"perf-volume\",\"seq\":%ld}\n",
        index_name, (unsigned long long)ts_ms, etype, proc, user_n, host_n, seq);

    if (written < 0 || (size_t)written >= remaining) return -1;
    return written;
}

static void *worker_main(void *arg) {
    worker_args_t *w = (worker_args_t *)arg;
    rng_t rng;
    rng_seed(&rng, (uint64_t)(w->thread_id * 0xA24BAED4963EE407ULL + 1));

    CURL *curl = curl_easy_init();
    if (!curl) {
        fprintf(stderr, "[thread %d] curl_easy_init failed\n", w->thread_id);
        return NULL;
    }

    char url[512];
    snprintf(url, sizeof(url), "%s/_bulk", w->es_addr);

    struct curl_slist *headers = NULL;
    headers = curl_slist_append(headers, "Content-Type: application/x-ndjson");
    /* Disable the "Expect: 100-continue" header curl auto-adds for bodies
     * over ~1KB. Real Elasticsearch handles it fine, but it's an
     * unnecessary extra round-trip either way, and some HTTP server
     * implementations (including anything not handling 100-continue
     * explicitly) will hang waiting for a "100 Continue" that never
     * comes — safer and faster to just not ask for one. */
    headers = curl_slist_append(headers, "Expect:");

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, discard_response);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10L);

    size_t buf_cap = (size_t)w->batch_size * 320 + 4096; /* generous per-doc estimate */
    char *buf = malloc(buf_cap);
    if (!buf) {
        fprintf(stderr, "[thread %d] out of memory allocating send buffer\n", w->thread_id);
        curl_slist_free_all(headers);
        curl_easy_cleanup(curl);
        return NULL;
    }

    uint64_t epoch_ms_base = (uint64_t)time(NULL) * 1000ULL - 86400000ULL; /* ~24h window, matches baseline_generator.py's spirit */

    struct timespec t0, t1;
    clock_gettime(CLOCK_MONOTONIC, &t0);

    long sent = 0;
    long errors = 0;
    while (sent < w->events_to_send) {
        size_t offset = 0;
        int in_batch = 0;
        int batch_target = w->batch_size;
        if (w->events_to_send - sent < batch_target) {
            batch_target = (int)(w->events_to_send - sent);
        }

        for (int i = 0; i < batch_target; i++) {
            int n = append_bulk_doc(buf + offset, buf_cap - offset, w->index_name, &rng, sent + i, epoch_ms_base);
            if (n < 0) break; /* buffer full, flush what we have */
            offset += (size_t)n;
            in_batch++;
        }
        if (in_batch == 0) break;

        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, buf);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, (long)offset);

        CURLcode res = curl_easy_perform(curl);
        long http_code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        if (res != CURLE_OK || http_code >= 300) {
            errors++;
            if (errors <= 3) {
                fprintf(stderr, "[thread %d] bulk request failed: %s (HTTP %ld)\n",
                        w->thread_id, curl_easy_strerror(res), http_code);
            }
        }

        sent += in_batch;
    }

    clock_gettime(CLOCK_MONOTONIC, &t1);
    w->events_sent = sent;
    w->http_errors = errors;
    w->elapsed_seconds = (t1.tv_sec - t0.tv_sec) + (t1.tv_nsec - t0.tv_nsec) / 1e9;

    free(buf);
    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);
    return NULL;
}

static void print_usage(const char *prog) {
    fprintf(stderr,
        "Usage: %s --es-addr <url> [--index <name>] [--events <n>] "
        "[--threads <n>] [--batch-size <n>]\n"
        "  --es-addr     Elasticsearch base URL (required), e.g. http://localhost:9200\n"
        "  --index       target index name (default: wraith-perf-test)\n"
        "  --events      total synthetic events to send (default: 1000000)\n"
        "  --threads     concurrent sender threads (default: 4)\n"
        "  --batch-size  events per _bulk request per thread (default: 1000)\n",
        prog);
}

int main(int argc, char **argv) {
    const char *es_addr = NULL;
    const char *index_name = "wraith-perf-test";
    long total_events = 1000000;
    int n_threads = 4;
    int batch_size = 1000;

    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--es-addr") == 0 && i + 1 < argc) es_addr = argv[++i];
        else if (strcmp(argv[i], "--index") == 0 && i + 1 < argc) index_name = argv[++i];
        else if (strcmp(argv[i], "--events") == 0 && i + 1 < argc) total_events = atol(argv[++i]);
        else if (strcmp(argv[i], "--threads") == 0 && i + 1 < argc) n_threads = atoi(argv[++i]);
        else if (strcmp(argv[i], "--batch-size") == 0 && i + 1 < argc) batch_size = atoi(argv[++i]);
        else if (strcmp(argv[i], "--help") == 0) { print_usage(argv[0]); return 0; }
    }

    if (!es_addr) {
        fprintf(stderr, "error: --es-addr is required\n\n");
        print_usage(argv[0]);
        return 1;
    }
    if (n_threads < 1) n_threads = 1;
    if (batch_size < 1) batch_size = 1;

    curl_global_init(CURL_GLOBAL_DEFAULT);

    pthread_t *tids = malloc(sizeof(pthread_t) * (size_t)n_threads);
    worker_args_t *args = calloc((size_t)n_threads, sizeof(worker_args_t));
    if (!tids || !args) {
        fprintf(stderr, "error: out of memory\n");
        return 1;
    }

    long base_events = total_events / n_threads;
    long remainder = total_events % n_threads;

    for (int i = 0; i < n_threads; i++) {
        args[i].thread_id = i;
        args[i].es_addr = es_addr;
        args[i].index_name = index_name;
        args[i].events_to_send = base_events + (i < remainder ? 1 : 0);
        args[i].batch_size = batch_size;
        pthread_create(&tids[i], NULL, worker_main, &args[i]);
    }

    long total_sent = 0, total_errors = 0;
    double max_elapsed = 0.0;
    for (int i = 0; i < n_threads; i++) {
        pthread_join(tids[i], NULL);
        total_sent += args[i].events_sent;
        total_errors += args[i].http_errors;
        if (args[i].elapsed_seconds > max_elapsed) max_elapsed = args[i].elapsed_seconds;
    }

    double eps = max_elapsed > 0 ? (double)total_sent / max_elapsed : 0.0;

    printf("{\n");
    printf("  \"index\": \"%s\",\n", index_name);
    printf("  \"threads\": %d,\n", n_threads);
    printf("  \"events_sent\": %ld,\n", total_sent);
    printf("  \"http_errors\": %ld,\n", total_errors);
    printf("  \"elapsed_seconds\": %.3f,\n", max_elapsed);
    printf("  \"events_per_second\": %.0f\n", eps);
    printf("}\n");

    free(tids);
    free(args);
    curl_global_cleanup();

    return total_errors > 0 ? 2 : 0;
}
