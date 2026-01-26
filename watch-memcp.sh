#!/bin/bash
# Watch for Go file changes and restart memcp

cd /home/carli/projekte/memcp

cleanup() {
    echo "Stopping memcp..."
    pkill -f "./memcp" 2>/dev/null
    exit 0
}
trap cleanup SIGINT SIGTERM

echo "Starting memcp watcher..."
make && ./memcp &

while inotifywait -e close_write -r . --include '\.go$' 2>/dev/null; do
    echo ""
    echo "=== Change detected, rebuilding... ==="
    pkill -f "./memcp" 2>/dev/null
    sleep 0.5
    if make; then
        echo "=== Build successful, restarting memcp ==="
        ./memcp &
    else
        echo "=== Build failed ==="
    fi
done
