#!/usr/bin/env bash
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
# Run from a workspace containing base/ and candidate/ (both built).
#
# UPGRADE CONTRACT -- DO NOT WEAKEN TO MAKE A FORMAT CHANGE PASS:
# 1. The old version fills the fixture, executes the public queries repeatedly
#    (cold/warm behaviour), and records the reference results outside data-dir.
# 2. The new version opens that SAME persisted data and executes the SAME queries.
# 3. Compare every result and the complete public-data snapshot with the old
#    reference BEFORE any candidate-side test DML. Mutation tests follow later.
# Cache names, internal format versions and layouts are BLACK BOXES to this test.
# Never prepare the candidate data by deleting caches, rebuilding tables, changing
# source data, patching engine code or rewriting the old reference. Such actions
# can hide incompatible persisted caches. Migration/reconstruction must be handled
# by the engine through its normal upgrade/read path, not by this test harness.
# A crash, query error or result mismatch MUST fail CI. Do not skip predecessors,
# queries or comparisons, reduce cache-exercising repetitions, or accept changed
# expectations merely to make a failing upgrade green: fix the compatibility bug.
set -euo pipefail
DATA_DIR="$GITHUB_WORKSPACE/upgrade-test-data"
ORACLES="$GITHUB_WORKSPACE/upgrade-oracles"
LOGS="$GITHUB_WORKSPACE/upgrade-logs"
VALIDATOR="$GITHUB_WORKSPACE/candidate/tests/storage/persistence/upgrade-validate.sh"
HELPERS="$GITHUB_WORKSPACE/candidate/tests/storage/persistence/upgrade-helpers.py"
API_PORT=${UPGRADE_API_PORT:-14521}
MYSQL_PORT=${UPGRADE_MYSQL_PORT:-14522}
MEMCP_PID=
mkdir -p "$DATA_DIR" "$ORACLES" "$LOGS"
# Newer clients preserve leading fixture comments. Predecessor SQL parsers may
# not support those comments; strip them on the client without changing SQL.
mysql_args=(timeout --foreground --kill-after=5s 30 mysql --skip-comments --connect-timeout=5 -h 127.0.0.1 -P "$MYSQL_PORT" -u root -padmin)

cleanup() {
  if [ -n "$MEMCP_PID" ]; then
    kill -TERM "$MEMCP_PID" 2>/dev/null || true
    wait "$MEMCP_PID" || true
  fi
}
trap cleanup EXIT

start_server() {
  local tree="$1" phase="$2"
  echo "upgrade phase: $phase"
  (cd "$tree" && exec ./memcp -data "$DATA_DIR" \
    --api-port="$API_PORT" --mysql-port="$MYSQL_PORT" --no-repl lib/main.scm) \
    > "$LOGS/$phase.log" 2>&1 &
  MEMCP_PID=$!
  for attempt in $(seq 1 60); do
    kill -0 "$MEMCP_PID" || { cat "$LOGS/$phase.log"; return 1; }
    if "${mysql_args[@]}" -e 'SELECT 1' >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  cat "$LOGS/$phase.log"
  echo "$phase never became ready" >&2
  return 1
}

stop_server() {
  "${mysql_args[@]}" -D memcp-tests -e 'SHUTDOWN'
  if kill -0 "$MEMCP_PID" 2>/dev/null; then
    kill -TERM "$MEMCP_PID"
  fi
  wait "$MEMCP_PID"
  MEMCP_PID=
}

# Only the old binary supplies the original reference. It is stored
# outside data-dir and never replaced by a candidate's observed data.
start_server base base-fill
"${mysql_args[@]}" -e 'CREATE DATABASE IF NOT EXISTS `memcp-tests`'
"${mysql_args[@]}" -D memcp-tests < candidate/tests/storage/persistence/upgrade-fill.sql
curl --max-time 30 -fsS "http://127.0.0.1:$API_PORT/scm" -u root:admin -d '
  (begin
    /* OLD-only ABI adaptation uses the native declaration signature returned
    by HELP. Its numeric bounds are emitted by Declaration.MinParams/MaxParams,
    not inferred from descriptive prose or a release name. Accept only known
    interfaces, supply every declared position, and fail on unknown signatures
    or a missing fixture. No failed call is caught or treated as success. */
    (define fixture_table_arity (match (help "table")
      (regex "(?m)^Allowed nø of parameters: ([0-9]+)-([0-9]+)$" _ minimum maximum)
      (begin
        (define arity (simplify maximum))
        (if (and (has? (list 2 4 5) arity) (<= (simplify minimum) arity)) arity
          (error "unsupported declared predecessor table signature")))
      _ (error "missing predecessor table declaration signature")))
    (define fixture_table (lambda (database name) (begin
      (define args (match fixture_table_arity
        5 (list database false name false nil)
        4 (list database false name false)
        2 (list database name)
        _ (error "unsupported predecessor table interface")))
      (define handle (apply table args))
      (if handle handle (error "missing owned upgrade fixture relation")))))
    (createcolumn (fixture_table "memcp-tests" "up_compute") "doubled" "INT" (quote ()) (quote ())
      (quote ("val")) (lambda (val) (* val 2)))
    (insert (fixture_table "memcp-tests" "up_group_events")
      (quote ("id" "tenant_id" "happened_at" "amount"))
      (map (produceN 6000) (lambda (i)
        (list (+ i 1) (+ 1 (mod i 2)) i (+ 1 (mod i 17))))))
    (insert (fixture_table "memcp-tests" "up_group_windows")
      (quote ("id" "tenant_id" "range_to"))
      (list (list 1 2 1000) (list 2 2 1500)
        (list 3 1 3000) (list 4 1 3000)))
    (insert (fixture_table "memcp-tests" "up_helper_file") (quote ("id" "stamp"))
      (map (produceN 4000) (lambda (i) (list (+ i 1) (+ i 1)))))
    (insert (fixture_table "memcp-tests" "up_helper_driver") (quote ("id" "file_id"))
      (map (produceN 4000) (lambda (i) (list (+ i 1) (+ i 1)))))
    (rebuild)
    true)
'
python3 "$HELPERS" record "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" snapshot "$ORACLES/before.json"
stop_server

# Distinguish old-writer/restart defects from candidate reader defects.
start_server base base-restart
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/before.json"
# Only OLD manufactures obsolete derived callbacks, after its unchanged public
# oracle. No candidate-side data preparation or schema-file edits are allowed.
curl --max-time 30 -fsS "http://127.0.0.1:$API_PORT/scm" -u root:admin -d '
  (begin
    (define fixture_table_arity (match (help "table")
      (regex "(?m)^Allowed nø of parameters: ([0-9]+)-([0-9]+)$" _ minimum maximum)
      (begin
        (define arity (simplify maximum))
        (if (and (has? (list 2 4 5) arity) (<= (simplify minimum) arity)) arity
          (error "unsupported declared predecessor table signature")))
      _ (error "missing predecessor table declaration signature")))
    (define fixture_table (lambda (database name) (begin
      (define args (match fixture_table_arity
        5 (list database false name false nil)
        4 (list database false name false)
        2 (list database name)
        _ (error "unsupported predecessor table interface")))
      (define handle (apply table args))
      (if handle handle (error "missing owned upgrade fixture relation")))))
    /* The public native cleanup endpoint attests the separate system-trigger
    owner. Zero-argument HELP lists declarations without invoking an absent function;
    the owner, when declared, must expose its exact schema/name interface. */
    (define fixture_native_fk_owner (match (help)
      (regex "(?m)^  remove_system_trigger: " _)
      (match (help "remove_system_trigger")
        (regex "(?m)^Allowed nø of parameters: 2-2$" _) true
        _ (error "unsupported predecessor system-trigger owner signature"))
      _ false))
    /* Historical SHOW encoded uint8 timing as a one-character string;
    the native declaration owner publishes the canonical timing names. */
    (define fixture_fk_timings (if fixture_native_fk_owner
      (list "before_insert" "before_update" "before_delete" "after_update")
      (json_decode "[\"\\u0000\",\"\\u0002\",\"\\u0004\",\"\\u0003\"]")))
    /* Historical producers used either one BEFORE priority of -100 or +100;
    CASCADE AFTER UPDATE has always used -100. Require a coherent native set. */
    (define fixture_fk_before_priority (if fixture_native_fk_owner 100
      (begin
        (define header (find
          (get_assoc (show "memcp-tests" "up_fk_abi_child" true) "triggers")
          (lambda (header) (equal? (get_assoc header "name") "__fk_up_fk_abi_ref_child_insert")) nil))
        (match (get_assoc header "priority")
          -100 -100
          100 100
          _ (error "unsupported predecessor foreign callback priority")))))
    (define fixture_fk_headers (lambda (system) (begin
      (define endpoints (list
        (list "up_fk_abi_child" (list
          (list "__fk_up_fk_abi_ref_child_insert" (nth fixture_fk_timings 0) fixture_fk_before_priority)
          (list "__fk_up_fk_abi_ref_child_update" (nth fixture_fk_timings 1) fixture_fk_before_priority)))
        (list "up_fk_abi_parent" (list
          (list "__fk_up_fk_abi_ref_parent_delete" (nth fixture_fk_timings 2) fixture_fk_before_priority)
          (list "__fk_up_fk_abi_ref_parent_update" (nth fixture_fk_timings 3) -100)))))
      (map endpoints (lambda (endpoint) (begin
        (define headers (get_assoc (show "memcp-tests" (nth endpoint 0) true) "triggers"))
        (define foreign_headers (filter headers (lambda (header)
          (match (get_assoc header "name")
            (regex "^__fk_up_fk_abi_ref_" _) true
            _ false))))
        (if (not (equal? (count foreign_headers) 2))
          (error "unexpected predecessor foreign callback endpoint declarations"))
        (map (nth endpoint 1) (lambda (role) (begin
          (define matches (filter foreign_headers (lambda (header)
            (equal? (get_assoc header "name") (nth role 0)))))
          (if (not (equal? (count matches) 1))
            (error "missing or duplicate predecessor foreign callback"))
          (define header (car matches))
          (if (not (and
            (equal? (get_assoc header "timing") (nth role 1))
            (equal? (get_assoc header "system") system)
            (equal? (get_assoc header "hidden") (not system))
            (equal? (get_assoc header "priority") (if system (nth role 2) 0))))
            (error "inconsistent predecessor foreign callback producer"))
          true))))))
      true)))
    /* Every writer first proves the real SQL FOREIGN declaration producer.
    Only writers with the historical public replacement interface manufacture
    the four obsolete persisted procedures; native owners keep their own code. */
    (match (help "createtrigger")
      (regex "(?m)^Allowed nø of parameters: 7-(7|8)$" _ maximum) true
      _ (error "unsupported predecessor public trigger producer signature"))
    (fixture_fk_headers true)
    (if (not fixture_native_fk_owner) (begin
    /* Legacy FK code is produced only by OLD, without weakening candidate
    system ownership or rewriting the persisted reader input. */
    (createtrigger
        (fixture_table "memcp-tests" "up_fk_abi_child")
        "__fk_up_fk_abi_ref_child_insert"
        "before_insert"
        ""
        ""
        (lambda (OLD NEW session tx)
          (begin
            (__fk_check_ref "memcp-tests" "up_fk_abi_parent" (quote ("id")) (list (get_assoc NEW "parent_id")) "up_fk_abi_ref")
            NEW))
        false)
      (createtrigger
        (fixture_table "memcp-tests" "up_fk_abi_child")
        "__fk_up_fk_abi_ref_child_update"
        "before_update"
        ""
        ""
        (lambda (OLD NEW session tx)
          (begin
            (__fk_check_ref "memcp-tests" "up_fk_abi_parent" (quote ("id")) (list (get_assoc NEW "parent_id")) "up_fk_abi_ref")
            NEW))
        false)
      (createtrigger
        (fixture_table "memcp-tests" "up_fk_abi_parent")
        "__fk_up_fk_abi_ref_parent_delete"
        "before_delete"
        ""
        ""
        (lambda (OLD NEW session tx)
          (__fk_on_parent_delete "memcp-tests" "up_fk_abi_child" (quote ("parent_id")) (list (get_assoc OLD "id")) "up_fk_abi_ref" "CASCADE"))
        false)
      (createtrigger
        (fixture_table "memcp-tests" "up_fk_abi_parent")
        "__fk_up_fk_abi_ref_parent_update"
        "after_update"
        ""
        ""
        (lambda (OLD NEW session tx)
          (__fk_on_parent_update "memcp-tests" "up_fk_abi_child" (quote ("parent_id")) (list (get_assoc OLD "id")) (list (get_assoc NEW "id")) "up_fk_abi_ref" "CASCADE"))
        false)
      (fixture_fk_headers false)))
    true)
'
stop_server

# Read the predecessor's untouched persisted data before any candidate DML.
# Compare only public results; cache names, versions and layouts are a black box.
start_server candidate candidate-upgrade
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/before.json"
stop_server

# Use a separate cold process to retain coverage of mutations before first read.
start_server candidate candidate-cold-dml
python3 "$HELPERS" cold-mutate "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
bash "$VALIDATOR" "$MYSQL_PORT" zero-policy-checks
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/before.json"
python3 "$HELPERS" mutate "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
bash "$VALIDATOR" "$MYSQL_PORT" mutate "$ORACLES/before.json" "$ORACLES/after-dml.json"
# New writer-only identity fixtures follow every mandatory legacy comparison.
curl --max-time 30 -fsS "http://127.0.0.1:$API_PORT/scm" -u root:admin \
  --data-binary @"$GITHUB_WORKSPACE/candidate/tests/storage/persistence/upgrade-identity-add-fill.scm"

stop_server

# The post-DML oracle is derived from the old reference plus specified
# mutations. Reopening must preserve that entire expected state.
start_server candidate candidate-restart
python3 "$HELPERS" cold-mutate "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json" --alias-variants
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/after-dml.json"
# New writer-only identity fixtures follow every mandatory legacy comparison.
curl --max-time 30 -fsS "http://127.0.0.1:$API_PORT/scm" -u root:admin \
  --data-binary @"$GITHUB_WORKSPACE/candidate/tests/storage/persistence/upgrade-identity-add-verify.scm"

stop_server
