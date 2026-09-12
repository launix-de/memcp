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
# Usage: upgrade-validate.sh <mysql-port>
set -uo pipefail

PORT="${1:?usage: upgrade-validate.sh <mysql-port>}"
MYSQL_BASE=(mysql -h 127.0.0.1 -P "$PORT" -u root -padmin -N -B memcp-tests)

CHECKS=0
FAILURES=0

# check DESC QUERY EXPECTED
# EXPECTED is compared verbatim against mysql's -N -B (tab-separated, no
# headers) output. NULL prints as the literal string "NULL" in this mode.
check() {
  local desc="$1" query="$2" expected="$3" actual stderr_out
  CHECKS=$((CHECKS + 1))
  # stdout and stderr must stay separate: the real MySQL client (unlike
  # MariaDB's) prints "[Warning] Using a password on the command line
  # interface can be insecure." to stderr on every invocation, which would
  # corrupt every comparison if merged into the compared value.
  actual=$("${MYSQL_BASE[@]}" -e "$query" 2>/tmp/upgrade-validate-stderr.$$)
  stderr_out=$(cat /tmp/upgrade-validate-stderr.$$ 2>/dev/null)
  rm -f /tmp/upgrade-validate-stderr.$$
  if [ "$actual" != "$expected" ]; then
    echo "MISMATCH: $desc"
    echo "  query:    $query"
    printf '  expected: %q\n' "$expected"
    printf '  actual:   %q\n' "$actual"
    if [ -n "$stderr_out" ]; then
      printf '  stderr:   %q\n' "$stderr_out"
    fi
    FAILURES=$((FAILURES + 1))
  fi
}

exec_sql() {
  "${MYSQL_BASE[@]}" -e "$1"
}

# ==========================================================================
# 1. StorageFloat
# ==========================================================================
check "Float: NULL survives"          "SELECT val FROM up_float WHERE id = 30"        "NULL"
check "Float: count total"            "SELECT COUNT(*) FROM up_float"                 "31"
check "Float: id=0 is non-null"       "SELECT val IS NOT NULL FROM up_float WHERE id = 0" "1"

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

echo "upgrade-validate: $((CHECKS - FAILURES))/$CHECKS checks passed"
if [ "$FAILURES" -ne 0 ]; then
  echo "upgrade-validate: $FAILURES check(s) FAILED -- candidate lost or corrupted data written by the base binary"
  exit 1
fi
exit 0
