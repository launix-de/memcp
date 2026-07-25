# AP0 Status: Measurement Baseline

Date: 2026-07-25

Measurement script:

- `todos/topdown-cleanup/measure-omnestum.py`

Local result directories:

- plan-only baseline: `/tmp/omnestum-measure-ap0-plan`
- small runtime baseline: `/tmp/omnestum-measure-ap0-runtime-small`

Measured against:

- API: `http://127.0.0.1:4526`
- database: `omnestum`
- session variable setup:
  - `SET @fop_user := 105`
  - `SET @fop_time := UNIX_TIMESTAMP()`

Plan baseline:

| query | EXPLAIN time | EXPLAIN IR time | IR bytes | SELECTs | EXISTS | legacy fallback | materialized |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `t_ID_t_jahr_canView_-trace` | 2.759s | 2.756s | 132699 | 154 | 131 | 125 | 36 |
| `t_ID_t_standort_canView_` | 2.792s | 2.761s | 133870 | 154 | 131 | 125 | 36 |
| `t_ID_t_uploaded_at_` | 2.625s | 2.651s | 124655 | 154 | 131 | 125 | 32 |
| `t_` | 3.931s | 3.894s | 218770 | 154 | 131 | 125 | 72 |
| `nosorter` | 3.945s | 3.853s | 218751 | 154 | 131 | 125 | 72 |
| `idonly` | cache-hot | cache-hot | 123952 | 154 | 131 | 125 | 32 |

Small runtime baseline:

| query | samples | median |
| --- | ---: | ---: |
| `idonly` | 1 | 3.164s |
| `t_ID_t_uploaded_at_` | 1 | 3.192s |

Initial conclusion:

- The Omnestum dataview query shape is dominated by planner structure, not only
  by missing text indexes.
- Even ID-only / narrow projection still contains 154 SELECTs and 131 EXISTS in
  the SQL source.
- Current IR still shows 125 legacy fallback markers and many session
  materialization helpers.
- The next planner work must reduce derived-table wrapping, correlated ACL
  subqueries, and legacy fallback paths before specialized fulltext indexes can
  deliver the final speedup.
