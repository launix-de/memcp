#!/usr/bin/env bash
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
# Run from a workspace containing base/ and candidate/ (both built).
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
mysql_args=(timeout --foreground --kill-after=5s 30 mysql --connect-timeout=5 -h 127.0.0.1 -P "$MYSQL_PORT" -u root -padmin)

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
    (createcolumn (table "memcp-tests" "up_compute") "doubled" "INT" (quote ()) (quote ())
      (quote ("val")) (lambda (val) (* val 2)))
    (insert (table "memcp-tests" "up_group_events")
      (quote ("id" "tenant_id" "happened_at" "amount"))
      (map (produceN 6000) (lambda (i)
        (list (+ i 1) (+ 1 (mod i 2)) i (+ 1 (mod i 17))))))
    (insert (table "memcp-tests" "up_group_windows")
      (quote ("id" "tenant_id" "range_to"))
      (list (list 1 2 1000) (list 2 2 1500)
        (list 3 1 3000) (list 4 1 3000)))
    (insert (table "memcp-tests" "up_helper_file") (quote ("id" "stamp"))
      (map (produceN 4000) (lambda (i) (list (+ i 1) (+ i 1)))))
    (insert (table "memcp-tests" "up_helper_driver") (quote ("id" "file_id"))
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
stop_server

start_server candidate candidate-upgrade
python3 "$HELPERS" cold-mutate "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" zero-policy-checks
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/before.json"
python3 "$HELPERS" mutate "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" mutate "$ORACLES/before.json" "$ORACLES/after-dml.json"
stop_server

# The post-DML oracle is derived from the old reference plus specified
# mutations. Reopening must preserve that entire expected state.
start_server candidate candidate-restart
python3 "$HELPERS" cold-mutate "$MYSQL_PORT" "$ORACLES/queries.json"
python3 "$HELPERS" check "$MYSQL_PORT" "$ORACLES/queries.json"
bash "$VALIDATOR" "$MYSQL_PORT" compare "$ORACLES/after-dml.json"
stop_server
