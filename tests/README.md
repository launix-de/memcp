<!-- Copyright (C) 2026 Carl-Philip Hänsch; SPDX-License-Identifier: GPL-3.0-or-later -->
# SQL Test Taxonomy

SQL integration suites use descriptive lower kebab-case filenames without
sequence numbers. The directory is the stable ownership and selection unit:

- `sql/`: language semantics, compatibility, DDL, DML, security, and triggers
- `planner/`: logical and physical planning for aggregates, joins, ordering,
  set operations, and subqueries
- `execution/`: physical operators and concurrency behavior
- `storage/`: formats, indexes, persistence, and recovery
- `integration/`: imports, anonymized application regressions, and composed
  read-model/query shapes
- `rdf/`: RDF, SPARQL, Turtle, and RDFHP behavior
- `performance/`: explicit latency, scaling, and benchmark suites

Every YAML suite is discovered recursively by `git-pre-commit`. A suite may
opt out with `metadata.ci: false` for manual benchmarks or expensive fixtures.
An expensive performance suite must additionally set `metadata.perf_ci: true`
to remain part of automatic performance A/B while skipping ordinary SQL CI.
Performance discovery requires at least one `threshold_ms` case. Suite-level
`metadata.performance_rows` supplies the fixture size unless a case overrides it.
SCM cases with `threshold_ms` use the same warmup, repetition and A/B gate as SQL;
their assertions must fail explicitly or check the returned JSON result.
Every suite must provide
`metadata.description`.

Place a regression at the layer that owns its root cause, not at the layer
where an application happened to expose it. Use `integration/query-shapes`
only when the composition of otherwise independent SQL features is the subject
of the test. Keep one topic per suite; split a file when setup and assertions
cover independent planner or execution contracts.

Run one suite directly with, for example:

```sh
python3 run_sql_tests.py tests/planner/subqueries/deep-correlation-membership.yaml
```

Run a taxonomy section through the pre-commit selector:

```sh
./git-pre-commit 'tests/planner/subqueries/*.yaml'
```
