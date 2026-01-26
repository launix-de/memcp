#!/bin/bash
#
# Copyright (C) 2023, 2024  Carl-Philip Hänsch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#
# Watch memcp executable and lib/ folder; restart on changes.

cd "$(dirname "$0")" || exit 1

MEMCP_PID=""

stop_memcp() {
    if [ -n "$MEMCP_PID" ]; then
        echo "Stopping memcp..."
        kill "$MEMCP_PID" 2>/dev/null
        wait "$MEMCP_PID" 2>/dev/null
        MEMCP_PID=""
    fi
}

cleanup() {
    stop_memcp
}
trap cleanup EXIT INT TERM

start_memcp() {
    tail -f /dev/null | ./memcp &
    MEMCP_PID=$!
    echo "Started memcp with PID $MEMCP_PID"
}

restart_memcp() {
    stop_memcp
    start_memcp
}

echo "Starting memcp watcher..."
start_memcp

# Keep watching forever - restart when executable memcp changes or anything in lib/ changes.
# go build does: write temp file, then rename -> moved_to event on directory.
if ! command -v inotifywait >/dev/null 2>&1; then
    echo "inotifywait not found; please install inotify-tools" >&2
    exit 1
fi

BASE_DIR="$(pwd)"
MEMCP_PATH="${BASE_DIR}/memcp"
LIB_DIR="${BASE_DIR}/lib"

while true; do
    while IFS='|' read -r changed_path changed_events; do
        if [ "${changed_path}" = "${MEMCP_PATH}" ]; then
            case "${changed_events}" in
                *CLOSE_WRITE*|*MOVED_TO*|*ATTRIB*)
                    echo ""
                    echo "=== Change detected (${changed_events} ${changed_path}), restarting memcp ==="
                    restart_memcp
                    ;;
            esac
        elif [ "${changed_path#${LIB_DIR}/}" != "${changed_path}" ]; then
            echo ""
            echo "=== Change detected (${changed_events} ${changed_path}), restarting memcp ==="
            restart_memcp
        fi
    done < <(
        inotifywait -m -q \
            -e close_write -e moved_to -e moved_from -e create -e delete -e attrib \
            --format '%w%f|%e' \
            "${BASE_DIR}" "${LIB_DIR}"
    )

    # inotifywait terminated unexpectedly; re-start the watcher loop.
done
