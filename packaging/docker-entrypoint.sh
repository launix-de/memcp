#!/bin/sh
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

set -eu

DATA_DIR=${MEMCP_DATA_DIR:-/data}
PASSWORD_FILE=${MEMCP_ROOT_PASSWORD_FILE:-/run/secrets/memcp_root_password}

if [ ! -d "$DATA_DIR/system" ]; then
	if [ -r "$PASSWORD_FILE" ]; then
		set -- --root-password-file="$PASSWORD_FILE" "$@"
	elif [ -n "${ROOT_PASSWORD:-}" ]; then
		root_password=$ROOT_PASSWORD
		set -- --root-password="$root_password" "$@"
	else
		printf '%s\n' \
			'memcp: a fresh data directory requires MEMCP_ROOT_PASSWORD_FILE or ROOT_PASSWORD' >&2
		exit 1
	fi
fi

exec /app/memcp --no-repl -data "$DATA_DIR" "$@"
