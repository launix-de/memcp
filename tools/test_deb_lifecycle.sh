#!/bin/sh
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

set -eu

DEB=${1:?usage: test_deb_lifecycle.sh PACKAGE.deb}
CREDENTIAL=/etc/memcp/initial-root-password
SOCKET=/run/memcp/memcp.sock

wait_for_mysql() {
	n=0
	while [ "$n" -lt 60 ]; do
		if mysqladmin --protocol=socket --socket="$SOCKET" \
			-uroot -p"$PASSWORD" ping >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
		n=$((n + 1))
	done
	printf 'memcp package service did not become ready\n' >&2
	systemctl status memcp.service --no-pager >&2 || true
	journalctl -u memcp.service --no-pager -n 100 >&2 || true
	return 1
}

integrity_signature() {
	mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" \
		-N -B memcp-tests -e \
		'SELECT COUNT(*), SUM(amount), GROUP_CONCAT(label ORDER BY id) FROM package_integrity'
}

assert_integrity() {
	actual=$(integrity_signature)
	expected=$(printf '3\t60\talpha,beta,gamma')
	[ "$actual" = "$expected" ] || {
		printf 'unexpected package lifecycle signature: %s\n' "$actual" >&2
		return 1
	}
}

sudo dpkg -i "$DEB"
PASSWORD=$(sudo sed -n '1p' "$CREDENTIAL")
[ -n "$PASSWORD" ]
wait_for_mysql

mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" -e \
	'CREATE DATABASE IF NOT EXISTS `memcp-tests`'
mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" memcp-tests -e \
	'DROP TABLE IF EXISTS package_integrity'
mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" memcp-tests -e \
	'CREATE TABLE package_integrity (id INT, amount INT, label TEXT) ENGINE=SAFE'
columns=$(mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" \
	-N -B memcp-tests -e 'SHOW COLUMNS FROM package_integrity' | cut -f1 | tr '\n' ',')
[ "$columns" = 'id,amount,label,' ] || {
	printf 'unexpected package lifecycle columns: %s\n' "$columns" >&2
	exit 1
}
mysql --protocol=socket --socket="$SOCKET" -uroot -p"$PASSWORD" memcp-tests -e \
	'INSERT INTO package_integrity VALUES (1,10,"alpha"),(2,20,"beta"),(3,30,"gamma")'
assert_integrity

# Upgrade/reinstall must restart gracefully without changing data or credentials.
sudo dpkg -i "$DEB"
wait_for_mysql
assert_integrity

# Removal followed by reinstall must retain the database.
sudo dpkg --remove memcp
test -d /var/lib/memcp/system
test -s "$CREDENTIAL"
sudo dpkg -i "$DEB"
wait_for_mysql
assert_integrity

# Even explicit package purge must not be an implicit database deletion command.
sudo dpkg --purge memcp
test -d /var/lib/memcp/system
test -s "$CREDENTIAL"
