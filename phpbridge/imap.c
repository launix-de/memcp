/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
#define _GNU_SOURCE
#include "_cgo_export.h"
#include "bridge.h"
#include "php.h"

static char *frontend_source;
static zend_function_entry *imap_functions;

ZEND_BEGIN_ARG_INFO_EX(imap_variadic_args, 0, 0, 0)
ZEND_ARG_VARIADIC_INFO(0, args)
ZEND_END_ARG_INFO()
ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(imap_rpc_args, 0, 1, IS_STRING, 0)
ZEND_ARG_TYPE_INFO(0, request, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(imap_shutdown_args, 0, 0, _IS_BOOL, 0)
ZEND_END_ARG_INFO()
static PHP_FUNCTION(memcp_imap_shutting_down) {
	ZEND_PARSE_PARAMETERS_NONE();
	RETURN_BOOL(EG(flags) & EG_FLAGS_IN_SHUTDOWN);
}

static PHP_FUNCTION(memcp_imap_rpc) {
	zend_string *request;
	ZEND_PARSE_PARAMETERS_START(1, 1) Z_PARAM_STR(request) ZEND_PARSE_PARAMETERS_END();
	uintptr_t handle = memcp_php_request();
	if (!handle) {
		zend_throw_error(NULL, "IMAP requires an active PHP request");
		RETURN_THROWS();
	}
	memcp_text result = memcp_imap_call(handle, ZSTR_VAL(request), ZSTR_LEN(request));
	if (result.error)
		zend_throw_error(NULL, "%s", result.error);
	else
		RETVAL_STRINGL(result.data ? result.data : "", result.length);
	free(result.data);
	free(result.error);
}
static PHP_FUNCTION(memcp_imap_dispatch) {
	zval fn, args[2];
	ZVAL_STRING(&fn, "MemCP\\IsolatedIMAP\\dispatch");
	ZVAL_STR_COPY(&args[0], execute_data->func->common.function_name);
	array_init(&args[1]);
	for (uint32_t i = 1; i <= ZEND_NUM_ARGS(); i++) {
		zval copy;
		ZVAL_COPY(&copy, ZEND_CALL_ARG(execute_data, i));
		add_next_index_zval(&args[1], &copy);
	}
	if (ZEND_CALL_INFO(execute_data) & ZEND_CALL_HAS_EXTRA_NAMED_PARAMS) {
		zend_string *key;
		zval *value;
		ZEND_HASH_FOREACH_STR_KEY_VAL(execute_data->extra_named_params, key, value) {
			zval copy;
			ZVAL_COPY(&copy, value);
			zend_hash_update(Z_ARRVAL(args[1]), key, &copy);
		}
		ZEND_HASH_FOREACH_END();
	}
	call_user_function(NULL, NULL, &fn, return_value, 2, args);
	zval_ptr_dtor(&fn);
	zval_ptr_dtor(&args[0]);
	zval_ptr_dtor(&args[1]);
}

static PHP_RINIT_FUNCTION(memcp_imap) {
	return zend_eval_string(frontend_source, NULL, "MemCP isolated IMAP adapter");
}
static zend_module_entry imap_module = {
	STANDARD_MODULE_HEADER, "imap", NULL, NULL,	 NULL,
	PHP_RINIT(memcp_imap),	NULL,	NULL, "0.1", STANDARD_MODULE_PROPERTIES};
void memcp_imap_configure(const char *source, const char *functions) {
	frontend_source = strdup(source);
	size_t count = 1;
	for (const char *p = functions; *p; p++)
		if (*p == '\n')
			count++;
	imap_functions = calloc(count + 3, sizeof(zend_function_entry));
	char *names = strdup(functions), *save = NULL;
	size_t i = 0;
	for (char *name = strtok_r(names, "\n", &save); name; name = strtok_r(NULL, "\n", &save)) {
		imap_functions[i++] = (zend_function_entry){.fname = strdup(name),
													.handler = ZEND_FN(memcp_imap_dispatch),
													.arg_info = imap_variadic_args,
													.num_args = 1};
	}
	free(names);
	imap_functions[i] = (zend_function_entry){.fname = "memcp_imap_rpc",
											  .handler = ZEND_FN(memcp_imap_rpc),
											  .arg_info = imap_rpc_args,
											  .num_args = 1};
	imap_functions[i + 1] = (zend_function_entry){.fname = "memcp_imap_shutting_down",
												  .handler = ZEND_FN(memcp_imap_shutting_down),
												  .arg_info = imap_shutdown_args,
												  .num_args = 0};
	imap_module.functions = imap_functions;
}
int memcp_imap_is_safe(void) {
	zend_function *fn = zend_hash_str_find_ptr(CG(function_table), "imap_open", sizeof("imap_open") - 1);
	return !fn || (fn->type == ZEND_INTERNAL_FUNCTION &&
				   fn->internal_function.handler == ZEND_FN(memcp_imap_dispatch));
}
void *memcp_imap_module(void) { return &imap_module; }
