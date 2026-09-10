<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Fifteen-period nested KPI benchmark

The executable SQL fixture is
[`tests/performance/kpi-period-aggregates.yaml`](../../tests/performance/kpi-period-aggregates.yaml).
It creates 360,000 offers across 36 months and 1,800,000 line items, then measures
12 monthly periods in 2026 and three full years (2024–2026). The query computes
COUNT(offers), SUM(per-offer line totals), and AVG(per-offer line totals).

Run with the binary built from each revision, in that revision's worktree:

```sh
PERF_TEST=1 PERF_NORECALIBRATE=1 PERF_BASELINE_FILE=/tmp/kpi-baseline-fresh.json \
  python3 run_sql_tests.py tests/performance/kpi-period-aggregates.yaml 19512 --log-times
```

Use the same YAML on both revisions and a fresh baseline JSON path per run.
`PERF_NORECALIBRATE=1` prevents the interactive runner from increasing fixture
size after a fast result. CI A/B mode also fixes the shared fixture size. Each run creates its own fixture and checks
all 15 output rows. The cold case has no warmup and one sample; the warm case has
20 warmups and seven samples. The warmup deliberately spans cache admission:
initial direct scans accumulate evidence of reuse before the planner pays for a
shared column. The cold number includes first-use planning and cache creation.
Do not run other benchmarks concurrently.

## Handwritten physical plan

[`handwritten.scm`](handwritten.scm) contains the measured alternative assembled
from EXPLAIN's storage scans. Initialize the fixture and execute its SQL once
first: the handwritten plan references that query's canonical item group cache.
Its setup projects each offer's grouped item total into a computed column. It
then uses three scalar reductions per period, avoiding a per-row aggregate tuple
and a per-offer nested group lookup. The results are triples of offer count,
total, and count of non-NULL offer totals. The last value is AVG's denominator.

For native measurements, compile the expression once with
`(eval (optimize (scheme ...)))` inside a zero-argument lambda, store the lambda
in a session, and time repeated calls through `/scm`. Merely interpreting the
printed EXPLAIN text does not provide the same execution configuration. Printed
compiled access descriptors also require reconstruction as native objects.

Exploratory measurements on the same 2.16-million-row fixture, JIT enabled:

| Physical work | Median / elapsed |
| --- | ---: |
| Original SQL, warm | approximately 1,180 ms |
| Handwritten vector reduction with parent projection | 263 ms |
| Handwritten independent scalar reductions, projection already built | 24.2 ms |
| Build the parent projection from warmed child groups | 1,239 ms |
| Build the parent projection by looking up raw items | 31,944 ms |

These are operator experiments, not the final revision A/B measurement. They
show why preparation cannot be omitted from the cost model.

The cold CPU profile exposed repeated range scans while first reading group
aggregates. Bulk INSERT had supplied aggregate values as ordinary delta slots,
but computed columns read their proxy. Seeding those proxies through `$set`
before rebuilding the group cache reduced the SQL cold run from approximately
76 seconds to 4.3 seconds in development.

## Compiler transformation and costs

The new physical alternative applies to a scalar SUM/COUNT aggregate over a
LEFT JOIN to a prepared group relation, with equality on its complete unique
group key and a filter on the driving relation. AVG already has separate SUM
and non-NULL COUNT descriptors; these remain unchanged. A shared computed
projection retains the LEFT JOIN's NULL result for missing groups. Cache
maintenance handles child changes and deleted groups; parent dates remain the
ordinary range-scan input.

Logical join planning is unchanged. The alternative is introduced only during
physical lowering. Its direct-probe and scan costs use the calibrated work units
owned by `tools/costgen`. EXPLAIN PHYSICAL exposes both alternatives and the full
projection build cost, charged over every driving row. A cold projection is
admitted only after the same canonical projection has accumulated at least that
much actual direct execution work. Each execution also compares scan costs;
tiny inputs keep the direct plan. Preparation is scoped to the current query.

Regression tests additionally cover missing and all-NULL children, empty periods,
updates, first non-NULL child insertion, complete child-group deletion, and moving
an offer between periods.

## Manual revision A/B

Measured with the same JIT binary/toolchain on both revisions, baseline
`767ac7c52` (the correctness fix in PR #877) versus this optimization. Both
runs passed the complete 15-row expectations. Same 360,000 offers, 1,800,000
items, cold warmup 0 / sample 1, warm warmup 20 / samples 7. Latencies are
wall-clock milliseconds reported by the SQL runner; the warm value is its
median.

| Case | Baseline | Optimized | Change | Speedup |
| --- | ---: | ---: | ---: | ---: |
| Cold, including group construction | 71,782.5 ms | 4,500.2 ms | −93.7% | 16.0× |
| Warm, after demonstrated reuse | 1,173.4 ms | 43.4 ms | −96.3% | 27.0× |

Admission is intentionally not immediate. In the corresponding 30-request
development run, requests 2–9 still used direct scans at approximately 1.16 s;
the transition occurred around request 10. The subsequent requests produced
correct results in approximately 39–45 ms. The 20 warmups in both revision
measurements cover that transition explicitly. A one-off query benefits from
the group-fill fix, without speculatively building the parent projection.
