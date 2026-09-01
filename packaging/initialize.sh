#!/bin/sh
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

set -eu

CONFIG=${MEMCP_CONFIG:-/etc/memcp/memcp.conf}
BINARY=${MEMCP_BINARY:-/usr/bin/memcp}
CREDENTIAL=${MEMCP_INITIAL_PASSWORD_FILE:-/etc/memcp/initial-root-password}
RUN_USER=${MEMCP_RUN_USER-memcp}

data_directory() {
	data=/var/lib/memcp
	while IFS= read -r line; do
		case "$line" in
			-data\ *) data=${line#-data } ;;
		esac
	done < "$CONFIG"
	case "$data" in
		/*) printf '%s\n' "$data" ;;
		*) printf 'memcp: data directory must be absolute: %s\n' "$data" >&2; return 1 ;;
	esac
}

generate_password() {
	umask 077
	credential_dir=${CREDENTIAL%/*}
	[ "$credential_dir" = "$CREDENTIAL" ] && credential_dir=.
	mkdir -p "$credential_dir"
	if [ ! -s "$CREDENTIAL" ]; then
		password=$(od -An -N24 -tx1 /dev/urandom | tr -d ' \n')
		[ "${#password}" -eq 48 ] || {
			printf 'memcp: could not generate the initial password\n' >&2
			return 1
		}
		printf '%s\n' "$password" > "$CREDENTIAL"
	fi
	chmod 600 "$CREDENTIAL"

	IFS= read -r password < "$CREDENTIAL"
	[ -n "$password" ] || {
		printf 'memcp: initial password file is empty: %s\n' "$CREDENTIAL" >&2
		return 1
	}
}

run_server() {
	if [ -n "$RUN_USER" ] && [ "$(id -u)" -eq 0 ]; then
		runuser -u "$RUN_USER" -- "$@"
	else
		"$@"
	fi
}

DATA=$(data_directory)
FRESH=true
if [ -d "$DATA/system" ]; then
	FRESH=false
else
	generate_password
fi
mkdir -p "$DATA"
if [ -n "$RUN_USER" ] && [ "$(id -u)" -eq 0 ]; then
	chown "$RUN_USER:$RUN_USER" "$DATA"
	if [ "$FRESH" = true ]; then
		# MemCP resolves --root-password-file itself. Grant its service group read
		# access only for the short bootstrap process, then restore root-only mode.
		chown "root:$RUN_USER" "$CREDENTIAL"
		chmod 640 "$CREDENTIAL"
	fi
fi

LOG=$(mktemp /tmp/memcp-package-init.XXXXXX)
chmod 600 "$LOG"
cleanup() {
	if [ "$FRESH" = true ] && [ -n "$RUN_USER" ] && [ "$(id -u)" -eq 0 ]; then
		chown root:root "$CREDENTIAL" 2>/dev/null || true
		chmod 600 "$CREDENTIAL" 2>/dev/null || true
	fi
	rm -f "$LOG"
}
trap cleanup EXIT HUP INT TERM

if [ "$FRESH" = true ]; then
	set -- --root-password-file="$CREDENTIAL"
else
	set --
fi
if ! run_server "$BINARY" -data "$DATA" "$@" --initialize \
	--disable-api --disable-mysql --mysql-socket= >"$LOG" 2>&1; then
	printf 'memcp: database initialization failed\n' >&2
	sed -n '1,120p' "$LOG" >&2
	exit 1
fi

[ -d "$DATA/system" ] || {
	printf 'memcp: initialization did not persist the system database\n' >&2
	exit 1
}

if [ "$FRESH" = true ]; then
	printf 'memcp: initial root password written to %s (mode 0600)\n' "$CREDENTIAL"
else
	printf 'memcp: existing database validated; credentials were not changed\n'
fi
