//go:build php

/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
#include <php.h>
#if PHP_VERSION_ID < 80500
#error "MemCP PHP quotas require PHP 8.5 or newer (max_memory_limit)"
#endif
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

/* Transfer retained results into Zend's request heap only after Go has returned.
 * A PHP allocation bailout must never unwind through an active Go stack. The
 * temporary C buffer is freed on OOM too, and an unclaimed connection is closed. */
static memcp_result *request_result(memcp_result *r) {
	const size_t cells_size = r->cells_length * sizeof(memcp_cell);
	const size_t error_size = r->error ? strlen(r->error) + 1 : 0;
	memcp_result *owned = NULL;
	zend_try {
		owned = emalloc(sizeof(*r) + cells_size + r->bytes_length + error_size);
	} zend_catch {
		if (r->handle) memcp_close(r->handle);
		memcp_result_free(r);
		zend_bailout();
	} zend_end_try();
	*owned = *r;
	char *buffer = (char *)(owned + 1);
	owned->cells = cells_size ? (memcp_cell *)buffer : NULL;
	if (cells_size) memcpy(buffer, r->cells, cells_size);
	buffer += cells_size;
	owned->bytes = r->bytes_length ? buffer : NULL;
	if (r->bytes_length) memcpy(buffer, r->bytes, r->bytes_length);
	buffer += r->bytes_length;
	owned->error = error_size ? buffer : NULL;
	if (error_size) memcpy(buffer, r->error, error_size);
	memcp_result_free(r);
	return owned;
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
	if (s->result) efree(s->result); s->result = NULL; s->position = 0;
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
	memcp_result *r = request_result(memcp_query(db->handle, ZSTR_VAL(stmt->active_query_string), ZSTR_LEN(stmt->active_query_string)));
	if (!accept_result(stmt->dbh, stmt, r)) { efree(r); return 0; }
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
	memcp_result *r = request_result(memcp_query(db->handle, (char *)ZSTR_VAL(sql), ZSTR_LEN(sql)));
	zend_long count = accept_result(dbh, NULL, r) ? r->affected : -1;
	efree(r); return count;
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
	memcp_result *r = request_result(memcp_open((char *)dbh->data_source + sizeof(prefix)-1,
		(char *)(dbh->username ? dbh->username : ""), (char *)(dbh->password ? dbh->password : "")));
	if (!accept_result(dbh, NULL, r)) {
		pdo_throw_exception(0, r->error, &dbh->error_code);
		efree(r); return 0;
	}
	db->handle = r->handle;
	efree(r);
	dbh->max_escaped_char_length = 2;
	return 1;
}

static const pdo_driver_t driver = { PDO_DRIVER_HEADER(memcp), connect_db };

/* Constructor dispatch is installed at MINIT, before request threads start.
 * The native MySQL driver's registry entry and methods are never modified. */
static unsigned int local_mysql_port;
static zif_handler original_construct, original_connect;
void memcp_set_mysql_port(unsigned int port) { local_mysql_port = port; }

static zend_string *local_dsn(zend_execute_data *execute_data) {
	if (!local_mysql_port || ZEND_CALL_NUM_ARGS(execute_data) < 1 ||
		zend_get_called_scope(execute_data) != php_pdo_get_dbh_ce()) return NULL;
	zval *arg = ZEND_CALL_ARG(execute_data, 1);
	if (Z_TYPE_P(arg) != IS_STRING || Z_STRLEN_P(arg) < 6 ||
		memcmp(Z_STRVAL_P(arg), "mysql:", 6) || strlen(Z_STRVAL_P(arg)) != Z_STRLEN_P(arg)) return NULL;
	/* Keep options whose semantics require native MySQL on the wire path. */
	if (ZEND_CALL_NUM_ARGS(execute_data) >= 4) {
		zval *options = ZEND_CALL_ARG(execute_data, 4);
		if (Z_TYPE_P(options) != IS_NULL && Z_TYPE_P(options) != IS_ARRAY) return NULL;
		if (Z_TYPE_P(options) == IS_ARRAY) {
			zend_ulong key; zend_string *name; zval *value;
			ZEND_HASH_FOREACH_KEY_VAL(Z_ARRVAL_P(options), key, name, value) {
				if (name) return NULL;
				switch (key) {
				case PDO_ATTR_ERRMODE: case PDO_ATTR_CASE: case PDO_ATTR_ORACLE_NULLS:
				case PDO_ATTR_DEFAULT_FETCH_MODE: case PDO_ATTR_STRINGIFY_FETCHES:
					break; /* handled by PDO itself */
				case PDO_ATTR_EMULATE_PREPARES:
					if (!zend_is_true(value)) return NULL;
					break;
				default: return NULL;
				}
			} ZEND_HASH_FOREACH_END();
		}
	}
	/* An exact, unambiguous subset only; native PDO parses every other DSN.
	 * Require explicit host/port/database; never capture Unix socket DSNs. */
	const char *host = NULL, *database = NULL;
	size_t host_len = 0, database_len = 0;
	unsigned int port = 0, seen = 0;
	const char *p = Z_STRVAL_P(arg)+6, *end = Z_STRVAL_P(arg)+Z_STRLEN_P(arg);
	while (p < end) {
		const char *stop = memchr(p, ';', (size_t)(end-p));
		if (!stop) stop = end;
		const char *equal = memchr(p, '=', (size_t)(stop-p));
		if (!equal || equal+1 == stop) return NULL;
		const char *value = equal+1;
		size_t length = (size_t)(stop-value), key_len = (size_t)(equal-p);
		unsigned int bit;
		if (key_len == 4 && !memcmp(p, "host", 4)) {
			bit = 1; host = value; host_len = length;
		} else if (key_len == 4 && !memcmp(p, "port", 4)) {
			bit = 2;
			if (length > 5) return NULL;
			for (const char *digit = value; digit < stop; digit++) {
				if (*digit < '0' || *digit > '9') return NULL;
				port = port*10+(unsigned int)(*digit-'0');
			}
		} else if (key_len == 6 && !memcmp(p, "dbname", 6)) {
			bit = 4; database = value; database_len = length;
		} else if (key_len == 7 && !memcmp(p, "charset", 7)) {
			bit = 8;
			if (!((length == 7 && !memcmp(value, "utf8mb4", 7)) ||
				(length == 4 && !memcmp(value, "utf8", 4)))) return NULL;
		} else return NULL;
		if (seen & bit) return NULL;
		seen |= bit;
		p = stop == end ? end : stop+1;
	}
	if ((seen & 7) != 7 || port != local_mysql_port ||
		!((host_len == 9 && !memcmp(host, "localhost", 9)) ||
		  (host_len == 9 && !memcmp(host, "127.0.0.1", 9)))) return NULL;
	zend_string *dsn = zend_string_alloc(sizeof("memcp:dbname=")-1+database_len, 0);
	memcpy(ZSTR_VAL(dsn), "memcp:dbname=", sizeof("memcp:dbname=")-1);
	memcpy(ZSTR_VAL(dsn)+sizeof("memcp:dbname=")-1, database, database_len);
	ZSTR_VAL(dsn)[ZSTR_LEN(dsn)] = '\0';
	return dsn;
}

static void routed_construct(zif_handler original, INTERNAL_FUNCTION_PARAMETERS) {
	zend_string *dsn = local_dsn(execute_data);
	if (!dsn) { original(INTERNAL_FUNCTION_PARAM_PASSTHRU); return; }
	zval *arg = ZEND_CALL_ARG(execute_data, 1), saved;
	ZVAL_COPY_VALUE(&saved, arg);
	ZVAL_STR(arg, dsn);
	/* Preserve the original argument and PDO's own exceptions/SQLSTATE. */
	zend_try {
		original(INTERNAL_FUNCTION_PARAM_PASSTHRU);
	} zend_catch {
		zval_ptr_dtor(arg); ZVAL_COPY_VALUE(arg, &saved);
		zend_bailout();
	} zend_end_try();
	zval_ptr_dtor(arg); ZVAL_COPY_VALUE(arg, &saved);
}
static void routed_pdo_construct(INTERNAL_FUNCTION_PARAMETERS) {
	routed_construct(original_construct, INTERNAL_FUNCTION_PARAM_PASSTHRU);
}
static void routed_pdo_connect(INTERNAL_FUNCTION_PARAMETERS) {
	routed_construct(original_connect, INTERNAL_FUNCTION_PARAM_PASSTHRU);
}
PHP_MINIT_FUNCTION(pdo_memcp) {
	if (php_pdo_register_driver(&driver) != SUCCESS) return FAILURE;
	zend_class_entry *ce = php_pdo_get_dbh_ce();
	original_construct = ce->constructor->internal_function.handler;
	ce->constructor->internal_function.handler = routed_pdo_construct;
	zend_function *connect = zend_hash_str_find_ptr(&ce->function_table, "connect", sizeof("connect")-1);
	if (connect) {
		original_connect = connect->internal_function.handler;
		connect->internal_function.handler = routed_pdo_connect;
	}
	return SUCCESS;
}
PHP_MSHUTDOWN_FUNCTION(pdo_memcp) {
	zend_class_entry *ce = php_pdo_get_dbh_ce();
	if (ce->constructor->internal_function.handler == routed_pdo_construct)
		ce->constructor->internal_function.handler = original_construct;
	zend_function *connect = zend_hash_str_find_ptr(&ce->function_table, "connect", sizeof("connect")-1);
	if (connect && connect->internal_function.handler == routed_pdo_connect)
		connect->internal_function.handler = original_connect;
	php_pdo_unregister_driver(&driver); return SUCCESS;
}
static const zend_module_dep dependencies[] = { ZEND_MOD_REQUIRED("pdo") ZEND_MOD_END };
static zend_module_entry module = {
	STANDARD_MODULE_HEADER_EX, NULL, dependencies, "pdo_memcp", NULL,
	PHP_MINIT(pdo_memcp), PHP_MSHUTDOWN(pdo_memcp), NULL, NULL, NULL,
	"0.1", STANDARD_MODULE_PROPERTIES
};
void *memcp_module(void) { return &module; }
