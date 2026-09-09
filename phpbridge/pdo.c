//go:build php

/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
#include <php.h>
#include <ext/pdo/php_pdo_driver.h>
#include "bridge.h"
#include "_cgo_export.h"

typedef struct {
	uintptr_t handle;
	zend_string *error;
	int64_t insert_id;
	bool transaction;
} memcp_db;
typedef struct {
	memcp_result *result;
	size_t position;
	bool parameters_required;
	zend_string *error;
} memcp_stmt;

void memcp_result_free(memcp_result *r) {
	if (!r) return;
	free(r->cells); free(r->bytes); free(r->error); free(r);
}

static bool accept_result(pdo_dbh_t *dbh, pdo_stmt_t *stmt, memcp_result *r) {
	memcp_db *db = dbh->driver_data;
	zend_string **error = stmt ? &((memcp_stmt *)stmt->driver_data)->error : &db->error;
	if (*error) { zend_string_release(*error); *error = NULL; }
	if (r->error) {
		*error = zend_string_init(r->error, strlen(r->error), 0);
		memcpy(stmt ? stmt->error_code : dbh->error_code, r->state, 6);
		return false;
	}
	db->insert_id = r->insert_id;
	db->transaction = r->transaction;
	memcpy(stmt ? stmt->error_code : dbh->error_code, PDO_ERR_NONE, 6);
	return true;
}

static void fetch_error(pdo_dbh_t *dbh, pdo_stmt_t *stmt, zval *info) {
	memcp_db *db = dbh->driver_data;
	zend_string *error = stmt && stmt->driver_data ? ((memcp_stmt *)stmt->driver_data)->error : db->error;
	add_next_index_long(info, 0);
	if (error) add_next_index_str(info, zend_string_copy(error));
}

static int close_cursor(pdo_stmt_t *stmt) {
	memcp_stmt *s = stmt->driver_data;
	memcp_result_free(s->result); s->result = NULL; s->position = 0;
	return 1;
}

static int destroy_stmt(pdo_stmt_t *stmt) {
	memcp_stmt *s = stmt->driver_data;
	if (s->error) zend_string_release(s->error);
	close_cursor(stmt); efree(stmt->driver_data); return 1;
}

static int execute_stmt(pdo_stmt_t *stmt) {
	memcp_stmt *s = stmt->driver_data;
	memcp_db *db = stmt->dbh->driver_data;
	close_cursor(stmt);
	if (s->error) { zend_string_release(s->error); s->error = NULL; }
	if (s->parameters_required && (!stmt->bound_params || zend_hash_num_elements(stmt->bound_params) == 0)) {
		pdo_raise_impl_error(stmt->dbh, stmt, "HY093", "Statement requires bound parameters"); return 0;
	}
	memcp_result *r = memcp_query(db->handle, ZSTR_VAL(stmt->active_query_string), ZSTR_LEN(stmt->active_query_string));
	if (!accept_result(stmt->dbh, stmt, r)) { memcp_result_free(r); return 0; }
	s->result = r;
	php_pdo_stmt_set_column_count(stmt, (int)r->columns);
	stmt->row_count = r->affected;
	return 1;
}

static int fetch_row(pdo_stmt_t *stmt, enum pdo_fetch_orientation orientation, zend_long offset) {
	memcp_stmt *s = stmt->driver_data;
	(void)offset;
	if (orientation != PDO_FETCH_ORI_NEXT || !s->result || s->position >= s->result->rows) return 0;
	s->position++; return 1;
}

static int describe_column(pdo_stmt_t *stmt, int column) {
	memcp_stmt *s = stmt->driver_data;
	memcp_cell *c = &s->result->cells[column];
	stmt->columns[column].name = zend_string_init(s->result->bytes ? s->result->bytes + c->offset : "", c->length, 0);
	stmt->columns[column].maxlen = 0;
	stmt->columns[column].precision = 0;
	return 1;
}

static int get_column(pdo_stmt_t *stmt, int column, zval *value, enum pdo_param_type *type) {
	memcp_stmt *s = stmt->driver_data;
	(void)type;
	if (!s->result || s->position == 0) { ZVAL_NULL(value); return 0; }
	memcp_cell *c = &s->result->cells[s->result->columns * s->position + column];
	switch (c->kind) {
	case 1: ZVAL_LONG(value, c->integer); break;
	case 2: ZVAL_DOUBLE(value, c->number); break;
	case 3: ZVAL_STRINGL(value, s->result->bytes ? s->result->bytes + c->offset : "", c->length); break;
	case 4: ZVAL_BOOL(value, c->integer); break;
	default: ZVAL_NULL(value);
	}
	return 1;
}

static const struct pdo_stmt_methods statement_methods = {
	.dtor = destroy_stmt, .executer = execute_stmt, .fetcher = fetch_row,
	.describer = describe_column, .get_col = get_column, .cursor_closer = close_cursor
};

static bool prepare(pdo_dbh_t *dbh, zend_string *sql, pdo_stmt_t *stmt, zval *options) {
	if (pdo_attr_lval(options, PDO_ATTR_CURSOR, PDO_CURSOR_FWDONLY) != PDO_CURSOR_FWDONLY) {
		pdo_raise_impl_error(dbh, stmt, "IM001", "MemCP supports forward-only cursors"); return false;
	}
	stmt->driver_data = ecalloc(1, sizeof(memcp_stmt));
	stmt->methods = &statement_methods;
	/* Ask PDO to discover placeholders, without substituting values. Keep only
	 * whether arguments are required: PDO handles binding and quoting later. */
	stmt->supports_placeholders = PDO_PLACEHOLDER_NAMED;
	stmt->named_rewrite_template = ":memcp%d";
	zend_string *rewritten = NULL;
	int parsed = pdo_parse_params(stmt, sql, &rewritten);
	if (rewritten) zend_string_release(rewritten);
	if (stmt->bound_param_map) {
		((memcp_stmt *)stmt->driver_data)->parameters_required = zend_hash_num_elements(stmt->bound_param_map) != 0;
		zend_hash_destroy(stmt->bound_param_map);
		FREE_HASHTABLE(stmt->bound_param_map);
		stmt->bound_param_map = NULL;
	}
	stmt->named_rewrite_template = NULL;
	stmt->supports_placeholders = PDO_PLACEHOLDER_NONE;
	return parsed >= 0;
}

static zend_long execute(pdo_dbh_t *dbh, const zend_string *sql) {
	memcp_db *db = dbh->driver_data;
	memcp_result *r = memcp_query(db->handle, (char *)ZSTR_VAL(sql), ZSTR_LEN(sql));
	zend_long count = accept_result(dbh, NULL, r) ? r->affected : -1;
	memcp_result_free(r); return count;
}

/* Hex literals preserve arbitrary bytes and cannot be affected by SQL quote
 * escaping modes. PDO itself parses and validates emulated placeholders. */
static zend_string *quote(pdo_dbh_t *dbh, const zend_string *value, enum pdo_param_type type) {
	(void)dbh; (void)type;
	static const char hex[] = "0123456789abcdef";
	zend_string *quoted = zend_string_safe_alloc(2, ZSTR_LEN(value), 2, 0);
	char *p = ZSTR_VAL(quoted); *p++ = '0'; *p++ = 'x';
	for (size_t i = 0; i < ZSTR_LEN(value); i++) {
		unsigned char c = (unsigned char)ZSTR_VAL(value)[i]; *p++ = hex[c >> 4]; *p++ = hex[c & 15];
	}
	*p = '\0'; return quoted;
}

static bool transaction_sql(pdo_dbh_t *dbh, const char *sql) {
	zend_string *s = zend_string_init(sql, strlen(sql), 0);
	zend_long n = execute(dbh, s); zend_string_release(s); return n >= 0;
}
static bool begin(pdo_dbh_t *dbh) { return transaction_sql(dbh, "BEGIN"); }
static bool commit(pdo_dbh_t *dbh) { return transaction_sql(dbh, "COMMIT"); }
static bool rollback_db(pdo_dbh_t *dbh) { return transaction_sql(dbh, "ROLLBACK"); }
static bool in_transaction(pdo_dbh_t *dbh) { return ((memcp_db *)dbh->driver_data)->transaction; }
static zend_string *last_id(pdo_dbh_t *dbh, const zend_string *name) {
	(void)name; return zend_long_to_str(((memcp_db *)dbh->driver_data)->insert_id);
}
static bool set_attribute(pdo_dbh_t *dbh, zend_long attribute, zval *value) {
	if (attribute == PDO_ATTR_EMULATE_PREPARES && zend_is_true(value)) return true;
	pdo_raise_impl_error(dbh, NULL, "IM001", "Unsupported MemCP PDO attribute"); return false;
}
static int get_attribute(pdo_dbh_t *dbh, zend_long attribute, zval *value) {
	(void)dbh;
	switch (attribute) {
	case PDO_ATTR_CLIENT_VERSION: ZVAL_STRING(value, "pdo_memcp/0.1"); return 1;
	case PDO_ATTR_SERVER_VERSION: ZVAL_STRING(value, "MemCP"); return 1;
	case PDO_ATTR_CONNECTION_STATUS: ZVAL_STRING(value, "in-process"); return 1;
	case PDO_ATTR_EMULATE_PREPARES: ZVAL_TRUE(value); return 1;
	default: return 0;
	}
}
static void close_db(pdo_dbh_t *dbh) {
	memcp_db *db = dbh->driver_data;
	if (!db) return;
	if (db->handle) memcp_close(db->handle);
	if (db->error) zend_string_release(db->error);
	efree(db); dbh->driver_data = NULL;
}

static const struct pdo_dbh_methods database_methods = {
	.closer = close_db, .preparer = prepare, .doer = execute, .quoter = quote,
	.begin = begin, .commit = commit, .rollback = rollback_db, .set_attribute = set_attribute,
	.last_id = last_id, .fetch_err = fetch_error, .get_attribute = get_attribute,
	.in_transaction = in_transaction
};

static int connect_db(pdo_dbh_t *dbh, zval *options) {
	(void)options;
	if (dbh->is_persistent) {
		pdo_raise_impl_error(dbh, NULL, "IM001", "Persistent MemCP PDO connections are not supported"); return 0;
	}
	const char prefix[] = "dbname=";
	if (strncmp(dbh->data_source, prefix, sizeof(prefix)-1) != 0 ||
		dbh->data_source_len <= sizeof(prefix)-1 || strlen(dbh->data_source) != dbh->data_source_len || strchr(dbh->data_source, ';')) {
		pdo_raise_impl_error(dbh, NULL, "IM002", "Use memcp:dbname=DATABASE"); return 0;
	}
	dbh->driver_data = ecalloc(1, sizeof(memcp_db));
	dbh->methods = &database_methods;
	memcp_db *db = dbh->driver_data;
	memcp_result *r = memcp_open((char *)dbh->data_source + sizeof(prefix)-1,
		(char *)(dbh->username ? dbh->username : ""), (char *)(dbh->password ? dbh->password : ""));
	if (!accept_result(dbh, NULL, r)) {
		pdo_throw_exception(0, r->error, &dbh->error_code);
		memcp_result_free(r); return 0;
	}
	db->handle = r->handle;
	memcp_result_free(r);
	dbh->max_escaped_char_length = 2;
	return 1;
}

static const pdo_driver_t driver = { PDO_DRIVER_HEADER(memcp), connect_db };
PHP_MINIT_FUNCTION(pdo_memcp) { return php_pdo_register_driver(&driver); }
PHP_MSHUTDOWN_FUNCTION(pdo_memcp) { php_pdo_unregister_driver(&driver); return SUCCESS; }
static const zend_module_dep dependencies[] = { ZEND_MOD_REQUIRED("pdo") ZEND_MOD_END };
static zend_module_entry module = {
	STANDARD_MODULE_HEADER_EX, NULL, dependencies, "pdo_memcp", NULL,
	PHP_MINIT(pdo_memcp), PHP_MSHUTDOWN(pdo_memcp), NULL, NULL, NULL,
	"0.1", STANDARD_MODULE_PROPERTIES
};
void *memcp_module(void) { return &module; }
