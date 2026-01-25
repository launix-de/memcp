# Performance Tests

MemCP includes optional performance tests that measure execution time for key operations and ensure they stay below configurable thresholds.

## Running Performance Tests

Performance tests are **skipped by default** to keep regular CI runs fast. To run them:

```bash
PERF_TEST=1 make test
```

Or run a specific performance test file:

```bash
PERF_TEST=1 python3 run_sql_tests.py tests/90_perf_basic.yaml
```

## Configuration

### TIME_SCALE

For slower systems (e.g., Raspberry Pi), you can scale the thresholds:

```bash
TIME_SCALE=2.0 PERF_TEST=1 make test
```

This multiplies all `threshold_ms` values by 2.0, allowing tests to pass on slower hardware.

## Writing Performance Tests

Performance tests use the same YAML format as regular tests, with additional fields:

### threshold_ms

Specifies the maximum allowed execution time in milliseconds:

```yaml
- name: "Perf: aggregation on 100k rows"
  sql: "SELECT SUM(value) FROM large_table"
  threshold_ms: 500
  expect:
    rows: 1
```

### generate_data

Generates test data before running the query:

```yaml
- name: "Perf: bulk insert 100k rows"
  generate_data:
    table: perf_data
    rows: 100000
    columns:
      - name: id
        type: sequential    # 0, 1, 2, ...
      - name: value
        type: int           # random integer 0-1000000
      - name: price
        type: float         # random float 0-1000000
      - name: name
        type: string        # "str_<random>"
      - name: active
        type: bool          # random true/false
  sql: "SELECT COUNT(*) FROM perf_data"
  threshold_ms: 5000
```

### warmup

By default, a warmup query runs before the timed execution. Disable with:

```yaml
- name: "Perf: cold start query"
  sql: "SELECT * FROM table LIMIT 10"
  threshold_ms: 100
  warmup: false
```

## Test Output

Performance tests show timing information:

```
✅ Perf: aggregation SUM/COUNT on 100k rows (22.3ms / 500ms)
```

Failed tests show how much the threshold was exceeded:

```
❌ Perf: slow query
    Reason: Too slow: 650.2ms > 500ms
```

## Best Practices

1. **Keep tests fast** - Use small but representative datasets (100k rows, not millions)
2. **One test per topic** - Don't create many similar performance tests
3. **Set realistic thresholds** - Account for CI variance (use ~2x expected time)
4. **Use warmup** - First query often includes cache initialization
5. **Test key operations** - Focus on: bulk insert, aggregation, filtered scans, joins, ORDER BY + LIMIT

## Example Test File

See `tests/90_perf_basic.yaml` for a complete example covering:

- Bulk insert (100k rows)
- Aggregation (SUM, COUNT, AVG, MIN, MAX)
- Filtered scans (WHERE clause)
- ORDER BY with LIMIT
- Self-joins
