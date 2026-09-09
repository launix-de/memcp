/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
#ifndef MEMCP_PHP_BRIDGE_H
#define MEMCP_PHP_BRIDGE_H
#include <stdint.h>
#include <stdlib.h>

/* C owns the returned allocation. Offsets refer to one packed byte buffer;
 * no Go pointers survive a call and fetching rows never calls back into Go. */
typedef struct {
	int kind; /* 0 NULL, 1 integer, 2 float, 3 string, 4 boolean */
	int64_t integer;
	double number;
	size_t offset, length;
} memcp_cell;
typedef struct {
	uintptr_t handle;
	int64_t affected, insert_id;
	int transaction;
	size_t columns, rows;
	memcp_cell *cells; /* column names followed by row-major values */
	char *bytes;
	char *error;
	char state[6];
} memcp_result;
void memcp_result_free(memcp_result *result);
void *memcp_module(void);
/* Set before PHP MINIT; zero disables automatic PDO DSN routing. */
void memcp_set_mysql_port(unsigned int port);
#endif
