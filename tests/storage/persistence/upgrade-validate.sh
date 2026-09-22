#!/usr/bin/env bash
# Copyright (C) 2026  Carl-Philip Haensch
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Upgrade-compatibility CI runner, phase 2 (validate). Run against the NEW
# (candidate) binary+library, pointed at the same -data directory that
# upgrade-fill.sql (run under the OLD/base binary) just wrote and cleanly
# shut down. Every value checked here must be exactly what upgrade-fill.sql
# wrote -- any mismatch means the candidate change lost or corrupted data
# written by an older version. Also exercises DML on the upgraded, disk-
# loaded storage to prove the new binary can mutate data the old binary
# compressed, not just read it.
#
# Usage: upgrade-validate.sh <mysql-port> snapshot|compare <snapshot.json>
#        upgrade-validate.sh <mysql-port> mutate <old.json> <post-dml.json>
#        upgrade-validate.sh <mysql-port> checks  # legacy focused DML checks
#        upgrade-validate.sh <mysql-port> group-checks
#        upgrade-validate.sh <mysql-port> zero-policy-checks
set -uo pipefail

PORT="${1:?usage: upgrade-validate.sh <mysql-port>}"
# Snapshots live outside the database directory and are written only by OLD.
# Full comparison is read-only; mutate derives its oracle from the OLD snapshot,
# never from candidate output. The workflow compares it again after a restart.
MODE="${2:-checks}"
if [ "$MODE" != checks ] && [ "$MODE" != group-checks ]; then
  python3 - "$0" "$PORT" "$MODE" "${3:-}" "${4:-}" <<'PYTHON'
import base64
import contextlib
import io
import json
import pathlib
import re
import subprocess
import sys

script, port, mode, snapshot_path, post_path = sys.argv[1:]
client = ["mysql", "-h", "127.0.0.1", "-P", port, "-u", "root", "-padmin",
          "--batch", "--raw", "memcp-tests"]


def query(sql, include_headers=False):
    options = ["--column-names"] if include_headers else ["--skip-column-names"]
    result = subprocess.run(client + options + ["-e", sql], capture_output=True, check=True, timeout=120)
    # Numeric values travel directly through RowWriter.Float64 ->
    # strconv.AppendFloat(value, 'g', -1, 64): shortest exact float64 roundtrip,
    # including signed zero. Never CAST/ROUND, parse to float, or use tolerance.
    # Every selected string is Base64 and numeric output is ASCII. Raw mode
    # therefore cannot turn embedded tabs/newlines/NULs into record boundaries.
    return [line.split(b"\t") for line in result.stdout.splitlines()]


def ident(name):
    if not re.fullmatch(r"[A-Za-z_][A-Za-z_0-9]*", name):
        raise ValueError("unexpected fixture identifier: " + repr(name))
    return "`" + name + "`"


def encoding(sql_type):
    kind = sql_type.lower().split("(")[0].split()[0]
    if kind in {"int", "integer", "bigint", "smallint", "tinyint", "mediumint"}:
        return "integer"
    if kind in {"float", "double", "real", "decimal", "numeric"}:
        return "number"
    if kind in {"json", "bson"}:
        return "json-base64"
    if kind in {"varchar", "char", "text", "longtext", "mediumtext", "tinytext",
                "blob", "longblob", "mediumblob", "tinyblob", "binary", "varbinary", "enum"}:
        return "bytes-base64"
    raise ValueError("unhandled fixture column type: " + sql_type)


def capture():
    names = sorted(row[0].decode("ascii") for row in query("SHOW TABLES")
                   if row[0].startswith(b"up_"))
    if not names:
        raise ValueError("no up_* fixture tables found")
    tables = {}
    for name in names:
        # Only schema fields are persistent promises. SHOW COLUMNS also exposes
        # planner statistics that legitimately change after compression/restart.
        metadata = query("SHOW COLUMNS FROM " + ident(name), include_headers=True)
        if not metadata:
            raise ValueError("missing column metadata for " + name)
        header = [field.decode("ascii") for field in metadata[0]]
        stable_fields = ("Field", "Type", "Collation", "RawType", "Dimensions", "Null",
                         "Key", "Default", "DefaultExpression", "Extra", "Privileges", "Comment")
        if len(set(header)) != len(header) or not set(stable_fields).issubset(header):
            raise ValueError("missing or duplicate schema metadata fields for " + name)
        columns = []
        expressions = []
        for fields in metadata[1:]:
            if len(fields) != len(header):
                raise ValueError("unexpected column metadata width for " + name)
            schema = dict(zip(header, fields))
            column = schema["Field"].decode("ascii")
            # Planner-owned temporary projections are cache artifacts attached
            # to durable tables. They are validated by group-checks and the
            # cache ABI, not part of the old-writer application-data oracle.
            if column.startswith("."):
                continue
            sql_type = schema["Type"].decode("ascii")
            codec = encoding(sql_type)
            columns.append({"name": column, "encoding": codec,
                            "metadata": {key: base64.b64encode(schema[key]).decode("ascii")
                                         for key in stable_fields}})
            value = ident(column)
            expressions.append(value + " IS NULL")
            if codec == "json-base64":
                value = "TO_BASE64(VECTOR_TO_STRING(" + value + "))"
            elif codec == "bytes-base64":
                value = "TO_BASE64(" + value + ")"
            expressions.append(value)
        if not columns:
            raise ValueError("fixture table has no columns: " + name)
        rows = []
        for fields in query("SELECT " + ", ".join(expressions) + " FROM " + ident(name)):
            if len(fields) != 2 * len(columns):
                raise ValueError("unexpected export row width in " + name)
            row = []
            for i, column in enumerate(columns):
                null, value = fields[2*i:2*i+2]
                if null not in (b"0", b"1"):
                    raise ValueError("invalid NULL marker in " + name)
                if null == b"1":
                    row.append(None)
                else:
                    token = value.decode("ascii")
                    if column["encoding"].endswith("base64"):
                        base64.b64decode(token, validate=True)
                    row.append(token)
            rows.append(row)
        # Compare a multiset of complete records: row order is not persistent,
        # while duplicate multiplicity, every column and every byte matter.
        tables[name] = {"columns": columns, "rows": sorted(rows, key=canonical)}
    return {"format": "memcp-upgrade-values-v1", "tables": tables}


def canonical(value):
    return json.dumps(value, ensure_ascii=True, sort_keys=True, separators=(",", ":"))


def comparison_values(snapshot):
    # Keep the saved old-writer oracle byte-for-byte intact. SQL DECIMAL and
    # NUMERIC have one zero; only their exact wire token -0 compares as 0.
    # FLOAT/DOUBLE, nonzero tokens, NULLs and schema metadata remain exact.
    tables = {}
    for name, table in snapshot["tables"].items():
        decimal_columns = []
        for column in table["columns"]:
            sql_type = base64.b64decode(column["metadata"]["Type"], validate=True).decode("ascii")
            kind = sql_type.lower().split("(")[0].split()[0]
            decimal_columns.append(kind in {"decimal", "numeric"})
        if any(len(row) != len(decimal_columns) for row in table["rows"]):
            raise ValueError("unexpected snapshot row width in " + name)
        rows = [["0" if decimal and value == "-0" else value
                 for decimal, value in zip(decimal_columns, row)]
                for row in table["rows"]]
        tables[name] = {"columns": table["columns"], "rows": sorted(rows, key=canonical)}
    return {**snapshot, "tables": tables}


def write_new(path, value):
    # Exclusive creation prevents accidentally overwriting the OLD oracle.
    with open(path, "x", encoding="ascii") as out:
        out.write(canonical(value) + "\n")


def compare(expected):
    actual = comparison_values(capture())
    expected = comparison_values(expected)
    if actual != expected:
        for name in sorted(set(actual["tables"]) | set(expected["tables"])):
            if actual["tables"].get(name) != expected["tables"].get(name):
                print("MISMATCH: complete fixture table " + name, file=sys.stderr)
                old_table = expected["tables"].get(name)
                new_table = actual["tables"].get(name)
                if old_table and new_table and old_table["columns"] == new_table["columns"]:
                    old_rows, new_rows = old_table["rows"], new_table["rows"]
                    print(f"  row counts: expected {len(old_rows)}, actual {len(new_rows)}", file=sys.stderr)
                    for row_index, (old_row, new_row) in enumerate(zip(old_rows, new_rows)):
                        if old_row != new_row:
                            for column, old_value, new_value in zip(old_table["columns"], old_row, new_row):
                                if old_value != new_value:
                                    print(f"  sorted row {row_index}, column {column['name']}: "
                                          f"expected {old_value!r}, actual {new_value!r}", file=sys.stderr)
                            break
        raise ValueError("full fixture snapshot differs (no tolerances)")
    count = sum(len(t["rows"]) for t in actual["tables"].values())
    print(f"upgrade snapshot: all {len(actual['tables'])} tables / {count} rows match exactly")


def check_zero_policy():
    name = "up_oracle_zero_policy"
    # This runs only against the candidate, outside the saved old-writer
    # fixture. Never replace or derive an original oracle from candidate data.
    query("CREATE TABLE " + ident(name) +
          " (id INT, d DECIMAL(10,2), n NUMERIC(10,2), f FLOAT, b DOUBLE, v DECIMAL(10,2))")
    try:
        query("INSERT INTO " + ident(name) + " VALUES (1,-0,-0,-0,-0,1.25),(2,NULL,NULL,NULL,NULL,NULL)")
        expected = capture()
        unchanged = canonical(expected)
        malformed = json.loads(unchanged)
        malformed["tables"][name]["rows"][0].append("extra")
        try:
            comparison_values(malformed)
        except ValueError:
            print("upgrade zero policy: correctly rejected malformed row width")
        else:
            raise ValueError("zero-policy comparison truncated malformed row")
        zero_row = next(row for row in expected["tables"][name]["rows"] if row[0] == "1")
        if zero_row[1:5] != ["-0"] * 4:
            raise ValueError("zero-policy fixture did not preserve signed zero")
        # SQL equality makes UPDATE -0 to +0 a no-op. Go through nonzero
        # values to guarantee that the requested zero sign is really stored.
        query("UPDATE " + ident(name) + " SET d=1,n=1 WHERE id=1")
        query("UPDATE " + ident(name) + " SET d=0,n=0 WHERE id=1")
        compare(expected)
        checks = (("f=0", "f=-0"), ("b=0", "b=-0"),
                  ("v=1.26", "v=1.25"), ("d=NULL", "d=0"),
                  ("d=0.01", "d=0"))
        for change, restore in checks:
            column = change.split("=")[0]
            query("UPDATE " + ident(name) + " SET " + column + "=999 WHERE id=1")
            query("UPDATE " + ident(name) + " SET " + change + " WHERE id=1")
            try:
                with contextlib.redirect_stderr(io.StringIO()):
                    compare(expected)
            except ValueError:
                print("upgrade zero policy: correctly rejected " + change)
            else:
                raise ValueError("zero-policy comparison accepted " + change)
            query("UPDATE " + ident(name) + " SET " + column + "=999 WHERE id=1")
            query("UPDATE " + ident(name) + " SET " + restore + " WHERE id=1")
        compare(expected)
        if canonical(expected) != unchanged:
            raise ValueError("zero-policy comparison mutated the original oracle")
    finally:
        query("DROP TABLE " + ident(name))


def changed_oracle(expected):
    def table(name):
        return expected["tables"][name]

    def row_by_id(name, wanted):
        target = table(name)
        id_column = [c["name"] for c in target["columns"]].index("id")
        matches = [r for r in target["rows"] if r[id_column] == str(wanted)]
        if len(matches) != 1:
            raise ValueError("DML oracle needs exactly one id in " + name)
        return matches[0]

    def update(name, wanted, column, value):
        idx = [c["name"] for c in table(name)["columns"]].index(column)
        row_by_id(name, wanted)[idx] = value

    update("up_float", 0, "val", "3.14159")
    const = table("up_const")
    new_row = list(row_by_id("up_const", 0))
    new_row[[c["name"] for c in const["columns"]].index("id")] = "50"
    const["rows"].append(new_row)
    enum = table("up_enum")
    enum["rows"].remove(row_by_id("up_enum", 99))
    update("up_compute", 1, "val", "100")
    update("up_compute", 1, "doubled", "200")
    for target in expected["tables"].values():
        target["rows"].sort(key=canonical)
    return expected


try:
    if mode == "zero-policy-checks":
        check_zero_policy()
    elif not snapshot_path:
        raise ValueError("snapshot path required")
    elif mode == "snapshot":
        write_new(snapshot_path, capture())
    elif mode in {"compare", "mutate"}:
        expected = json.loads(pathlib.Path(snapshot_path).read_text(encoding="ascii"))
        if expected.get("format") != "memcp-upgrade-values-v1":
            raise ValueError("unknown snapshot format")
        compare(expected)
        if mode == "mutate":
            if not post_path:
                raise ValueError("mutate requires a separate post-DML snapshot path")
            expected = changed_oracle(expected)
            subprocess.run(["bash", script, port, "checks"], check=True)
            compare(expected)
            write_new(post_path, expected)
    else:
        raise ValueError("mode must be snapshot, compare, mutate, checks, or zero-policy-checks")
except (ValueError, OSError, subprocess.SubprocessError) as error:
    print("upgrade snapshot FAILED: " + str(error), file=sys.stderr)
    if isinstance(error, subprocess.CalledProcessError) and error.stderr:
        sys.stderr.buffer.write(error.stderr)
    sys.exit(1)
PYTHON
  exit "$?"
fi

MYSQL_BASE=(mysql -h 127.0.0.1 -P "$PORT" -u root -padmin -N -B memcp-tests)

CHECKS=0
FAILURES=0

# check DESC QUERY EXPECTED
# EXPECTED is compared verbatim against mysql's -N -B (tab-separated, no
# headers) output. NULL prints as the literal string "NULL" in this mode.
check() {
  local desc="$1" query="$2" expected="$3" actual stderr_out status=0
  CHECKS=$((CHECKS + 1))
  # stdout and stderr must stay separate: the real MySQL client (unlike
  # MariaDB's) prints "[Warning] Using a password on the command line
  # interface can be insecure." to stderr on every invocation, which would
  # corrupt every comparison if merged into the compared value.
  actual=$("${MYSQL_BASE[@]}" -e "$query" 2>/tmp/upgrade-validate-stderr.$$) || status=$?
  stderr_out=$(cat /tmp/upgrade-validate-stderr.$$ 2>/dev/null)
  rm -f /tmp/upgrade-validate-stderr.$$
  if [ "$status" -ne 0 ] || [ "$actual" != "$expected" ]; then
    echo "MISMATCH: $desc"
    echo "  query:    $query"
    echo "  mysql exit status: $status"
    printf '  expected: %q\n' "$expected"
    printf '  actual:   %q\n' "$actual"
    if [ -n "$stderr_out" ]; then
      printf '  stderr:   %q\n' "$stderr_out"
    fi
    FAILURES=$((FAILURES + 1))
  fi
}

exec_sql() {
  check "DML statement succeeds" "$1" ""
}

check_contains() {
  local desc="$1" query="$2" needle="$3" actual stderr_out status=0
  CHECKS=$((CHECKS + 1))
  actual=$("${MYSQL_BASE[@]}" -e "$query" 2>/tmp/upgrade-validate-stderr.$$) || status=$?
  stderr_out=$(cat /tmp/upgrade-validate-stderr.$$ 2>/dev/null)
  rm -f /tmp/upgrade-validate-stderr.$$
  if [ "$status" -ne 0 ] || [[ "$actual" != *"$needle"* ]]; then
    echo "MISMATCH: $desc"
    echo "  query:    $query"
    echo "  mysql exit status: $status"
    printf '  expected output containing: %q\n' "$needle"
    printf '  actual:   %q\n' "$actual"
    if [ -n "$stderr_out" ]; then
      printf '  stderr:   %q\n' "$stderr_out"
    fi
    FAILURES=$((FAILURES + 1))
  fi
}

if [ "$MODE" = group-checks ]; then
  check "Group cache survives upgrade" \
    "SELECT tenant_id, COUNT(*) FROM up_group_events GROUP BY tenant_id ORDER BY tenant_id" \
    "$(printf '1\t3000\n2\t3000')"
  range_query="SELECT w.id, (SELECT e.id FROM up_group_events e WHERE e.tenant_id = w.tenant_id AND e.happened_at <= w.range_to ORDER BY e.happened_at DESC LIMIT 1) FROM up_group_windows w ORDER BY w.id"
  check "Range group cache survives upgrade" "$range_query" \
    "$(printf '1\t1000\n2\t1500\n3\t3001\n4\t3001')"
  check_contains "Range query still selects the range cache operator" \
    "EXPLAIN PHYSICAL $range_query" "range_group_cache"
  echo "upgrade group-cache validation: $((CHECKS - FAILURES))/$CHECKS checks passed"
  if [ "$FAILURES" -ne 0 ]; then
    exit 1
  fi
  exit 0
fi

# ==========================================================================
# 1. StorageFloat
# ==========================================================================
check "Float: NULL survives"          "SELECT val FROM up_float WHERE id = 30"        "NULL"
check "Float: count total"            "SELECT COUNT(*) FROM up_float"                 "32"
check "Float: original first value"   "SELECT val = 1.1557281258737144 FROM up_float WHERE id = 0" "1"
check "Float: original middle value"  "SELECT val = 2.8369278733601684 FROM up_float WHERE id = 15" "1"
check "Float: original last value"    "SELECT val = 2.9714025949704714 FROM up_float WHERE id = 29" "1"
check "Float: original small value"   "SELECT val = 0.000000012345678912345678 FROM up_float WHERE id = 31" "1"

# ==========================================================================
# 2. StorageString with dictionary
# ==========================================================================
check "Dict: id=0 Berlin"       "SELECT city FROM up_string_dict WHERE id = 0"      "Berlin"
check "Dict: id=3 Cologne"      "SELECT city FROM up_string_dict WHERE id = 3"      "Cologne"
check "Dict: count Berlin"      "SELECT COUNT(*) FROM up_string_dict WHERE city = 'Berlin'" "40"
check "Dict: NULL survives"     "SELECT city FROM up_string_dict WHERE id = 200"    "NULL"
check "Dict: count total"       "SELECT COUNT(*) FROM up_string_dict"               "201"

# ==========================================================================
# 3. StorageString without dictionary
# ==========================================================================
check "Nodict: id=0"       "SELECT description FROM up_string_nodict WHERE id = 0"   "item_0_description_unique_0"
check "Nodict: id=99"      "SELECT description FROM up_string_nodict WHERE id = 99"  "item_99_description_unique_693"
check "Nodict: id=149"     "SELECT description FROM up_string_nodict WHERE id = 149" "item_149_description_unique_1043"
check "Nodict: NULL survives" "SELECT description FROM up_string_nodict WHERE id = 150" "NULL"
check "Nodict: count total" "SELECT COUNT(*) FROM up_string_nodict"                  "151"

# ==========================================================================
# 4. StorageSparse
# ==========================================================================
check "Sparse: non-null id=0"    "SELECT rare_val FROM up_sparse WHERE id = 0"   "0"
check "Sparse: non-null id=20"   "SELECT rare_val FROM up_sparse WHERE id = 20"  "200"
check "Sparse: non-null id=180"  "SELECT rare_val FROM up_sparse WHERE id = 180" "1800"
check "Sparse: NULL id=1"        "SELECT rare_val FROM up_sparse WHERE id = 1"   "NULL"
check "Sparse: count non-null"   "SELECT COUNT(*) FROM up_sparse WHERE rare_val IS NOT NULL" "10"
check "Sparse: count NULL"       "SELECT COUNT(*) FROM up_sparse WHERE rare_val IS NULL"     "190"

# ==========================================================================
# 5. StorageSeq
# ==========================================================================
check "Seq: id=0 (100)"          "SELECT seq_val FROM up_seq WHERE id = 0"   "100"
check "Seq: id=99 (298)"         "SELECT seq_val FROM up_seq WHERE id = 99"  "298"
check "Seq: run2 id=100 (500)"   "SELECT seq_val FROM up_seq WHERE id = 100" "500"
check "Seq: run2 id=149 (647)"   "SELECT seq_val FROM up_seq WHERE id = 149" "647"
check "Seq: count total"         "SELECT COUNT(*) FROM up_seq"               "150"
check "Seq: SUM first run"       "SELECT SUM(seq_val) FROM up_seq WHERE id < 100" "19900"

# ==========================================================================
# 6. StorageInt with NULLs and wide range
# ==========================================================================
check "IntNull: positive"        "SELECT val FROM up_int_null WHERE id = 1" "100"
check "IntNull: NULL"            "SELECT val FROM up_int_null WHERE id = 2" "NULL"
check "IntNull: negative"        "SELECT val FROM up_int_null WHERE id = 3" "-50"
check "IntNull: count NULLs"     "SELECT COUNT(*) FROM up_int_null WHERE val IS NULL" "5"
check "IntWide: max positive"    "SELECT val FROM up_int_wide WHERE id = 6" "2147483647"
check "IntWide: max negative"    "SELECT val FROM up_int_wide WHERE id = 7" "-2147483648"

# ==========================================================================
# 7. StorageDecimal
# ==========================================================================
check "DecNeg: positive"       "SELECT amount FROM up_decimal_neg WHERE id = 1" "12.5"
check "DecNeg: negative"       "SELECT amount FROM up_decimal_neg WHERE id = 2" "-12.5"
check "DecNeg: small positive" "SELECT amount FROM up_decimal_neg WHERE id = 3" "0.01"
check "DecNeg: NULL"           "SELECT amount FROM up_decimal_neg WHERE id = 5" "NULL"
check "DecNeg: large positive" "SELECT amount FROM up_decimal_neg WHERE id = 6" "99999.99"
check "DecNeg: large negative" "SELECT amount FROM up_decimal_neg WHERE id = 7" "-99999.99"

# ==========================================================================
# 8. StorageConst
# ==========================================================================
check "Const: count total"       "SELECT COUNT(*) FROM up_const"                          "50"
check "Const: all rows active"   "SELECT COUNT(*) FROM up_const WHERE status = 'active'"  "50"
check "Const: spot check id=25"  "SELECT status FROM up_const WHERE id = 25"              "active"

# ==========================================================================
# 9. StorageEnum
# ==========================================================================
check "Enum: count total"    "SELECT COUNT(*) FROM up_enum"                     "100"
check "Enum: distribution A" "SELECT COUNT(*) FROM up_enum WHERE grade = 'A'"   "90"
check "Enum: distribution B" "SELECT COUNT(*) FROM up_enum WHERE grade = 'B'"   "5"
check "Enum: distribution C" "SELECT COUNT(*) FROM up_enum WHERE grade = 'C'"   "3"
check "Enum: distribution D" "SELECT COUNT(*) FROM up_enum WHERE grade = 'D'"   "2"
check "Enum: boundary id=97/98" "SELECT id, grade FROM up_enum WHERE id IN (97, 98) ORDER BY id" "$(printf '97\tC\n98\tD')"

# ==========================================================================
# 10. StorageSCMER (JSON/BSON)
# ==========================================================================
check "JSON: count total"        "SELECT COUNT(*) FROM up_json" "5"
check "JSON: scalar field"       "SELECT JSON_VALUE(payload, '\$.rank' RETURNING UNSIGNED) FROM up_json WHERE id = 3" "3"
check "JSON: nested array elem"  "SELECT JSON_VALUE(payload, '\$.tags[1]') FROM up_json WHERE id = 3" "e"
check "JSON: empty array survives" "SELECT JSON_VALUE(payload, '\$.tags[0]') FROM up_json WHERE id = 2" "NULL"
check "JSON: aggregate over tenant" "SELECT COUNT(*) FROM up_json WHERE JSON_VALUE(payload, '\$.tenant' RETURNING UNSIGNED) = 0" "2"

# ==========================================================================
# 11. OverlayBlob
# ==========================================================================
check "Blob: length of large value"     "SELECT LENGTH(content) FROM up_blob WHERE id = 1" "3012"
check "Blob: complete large value"      "SELECT content = CONCAT('alpha-', REPEAT('x', 3000), '-alpha') FROM up_blob WHERE id = 1" "1"
check "Blob: content markers both ends" "SELECT LEFT(content, 6), RIGHT(content, 6) FROM up_blob WHERE id = 3" "$(printf 'charli\tharlie')"
check "Blob: four large rows distinct"  "SELECT COUNT(DISTINCT content) FROM up_blob WHERE id <= 4" "4"
check "Blob: short value alongside"     "SELECT content FROM up_blob WHERE id = 5" "short"
check "Blob: count total"               "SELECT COUNT(*) FROM up_blob" "5"

# ==========================================================================
# 12. StorageComputeProxy
# ==========================================================================
check "Compute: values for every row" "SELECT id, val, doubled FROM up_compute ORDER BY id" "$(printf '1\t5\t10\n2\t10\t20\n3\t-3\t-6\n4\t0\t0\n5\t42\t84')"

# ==========================================================================
# DML on the upgraded, disk-loaded storage: proves the new binary can not
# only read but also mutate data compressed by the old binary.
# ==========================================================================
exec_sql "UPDATE up_float SET val = 3.14159 WHERE id = 0"
check "DML: float UPDATE visible" "SELECT val FROM up_float WHERE id = 0" "3.14159"

exec_sql "INSERT INTO up_const VALUES (50, 'active')"
check "DML: const INSERT visible" "SELECT COUNT(*) FROM up_const WHERE status = 'active'" "51"

exec_sql "DELETE FROM up_enum WHERE id = 99"
check "DML: enum DELETE visible" "SELECT COUNT(*) FROM up_enum" "99"

exec_sql "UPDATE up_compute SET val = 100 WHERE id = 1"
check "DML: compute base column UPDATE visible" "SELECT val FROM up_compute WHERE id = 1" "100"
check "DML: computed column UPDATE visible" "SELECT doubled FROM up_compute WHERE id = 1" "200"

echo "upgrade-validate: $((CHECKS - FAILURES))/$CHECKS checks passed"
if [ "$FAILURES" -ne 0 ]; then
  echo "upgrade-validate: $FAILURES check(s) FAILED -- candidate lost or corrupted data written by the base binary"
  exit 1
fi
exit 0
