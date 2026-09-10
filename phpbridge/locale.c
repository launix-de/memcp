/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
#define _GNU_SOURCE
#include "Zend/zend_smart_str.h"
#include "_cgo_export.h"
#include "bridge.h"
#include "ext/standard/php_string.h"
#include "ext/standard/basic_functions.h"
#include "php.h"
#include <langinfo.h>
#include <locale.h>

/* libc's domain bindings and setlocale are process-global. Keep PHP's public
 * API, but give each request a POSIX thread locale and its own domain map. */
ZEND_BEGIN_MODULE_GLOBALS(memcp_locale)
locale_t locale;
locale_t previous;
HashTable directories;
HashTable codesets;
zend_string *domain;
uintptr_t request;
bool active;
ZEND_END_MODULE_GLOBALS(memcp_locale)
ZEND_DECLARE_MODULE_GLOBALS(memcp_locale)
#define LG(v) ZEND_MODULE_GLOBALS_ACCESSOR(memcp_locale, v)
static locale_t initial_locale;
static const int categories[] = {LC_CTYPE, LC_NUMERIC, LC_TIME, LC_COLLATE, LC_MONETARY, LC_MESSAGES};
static const char *category_names[] = {"LC_CTYPE",	 "LC_NUMERIC",	"LC_TIME",
									   "LC_COLLATE", "LC_MONETARY", "LC_MESSAGES"};

static zend_string *locale_name(int category) {
	if (category != LC_ALL)
		return zend_string_init(nl_langinfo_l(_NL_LOCALE_NAME(category), LG(locale)),
								strlen(nl_langinfo_l(_NL_LOCALE_NAME(category), LG(locale))), 0);
	const char *first = nl_langinfo_l(_NL_LOCALE_NAME(LC_CTYPE), LG(locale));
	bool same = true;
	for (size_t i = 1; i < sizeof(categories) / sizeof(categories[0]); i++)
		if (strcmp(first, nl_langinfo_l(_NL_LOCALE_NAME(categories[i]), LG(locale))))
			same = false;
	if (same)
		return zend_string_init(first, strlen(first), 0);
	smart_str text = {0};
	for (size_t i = 0; i < sizeof(categories) / sizeof(categories[0]); i++) {
		if (i)
			smart_str_appendc(&text, ';');
		smart_str_appends(&text, category_names[i]);
		smart_str_appendc(&text, '=');
		smart_str_appends(&text, nl_langinfo_l(_NL_LOCALE_NAME(categories[i]), LG(locale)));
	}
	smart_str_0(&text);
	return text.s;
}
static zend_string *request_env(const char *name) {
	zval fn, key, value;
	ZVAL_STRING(&fn, "getenv");
	ZVAL_STRING(&key, name);
	ZVAL_UNDEF(&value);
	call_user_function(NULL, NULL, &fn, &value, 1, &key);
	zend_string *result =
		Z_TYPE(value) == IS_STRING ? zend_string_copy(Z_STR(value)) : zend_string_init("", 0, 0);
	zval_ptr_dtor(&fn);
	zval_ptr_dtor(&key);
	zval_ptr_dtor(&value);
	return result;
}
static locale_t environment_locale(int category, locale_t copy) {
	zend_string *all = request_env("LC_ALL"), *lang = request_env("LANG");
	for (size_t i = 0; i < sizeof(categories) / sizeof(categories[0]); i++) {
		if (category != LC_ALL && category != categories[i])
			continue;
		zend_string *specific = request_env(category_names[i]);
		const char *name = ZSTR_LEN(all)		? ZSTR_VAL(all)
						   : ZSTR_LEN(specific) ? ZSTR_VAL(specific)
						   : ZSTR_LEN(lang)		? ZSTR_VAL(lang)
												: "C";
		locale_t next = EG(exception) ? (locale_t)0 : newlocale(1 << categories[i], name, copy);
		zend_string_release(specific);
		if (!next) {
			freelocale(copy);
			copy = 0;
			break;
		}
		copy = next;
	}
	zend_string_release(all);
	zend_string_release(lang);
	return copy;
}
static zend_string *try_locale(int category, zval *value) {
	zend_string *name = zval_get_string(value);
	if (EG(exception)) {
		zend_string_release(name);
		return NULL;
	}
	if (memchr(ZSTR_VAL(name), 0, ZSTR_LEN(name))) {
		zend_string_release(name);
		zend_value_error("Locale must not contain null bytes");
		return NULL;
	}
	if (zend_string_equals_literal(name, "0")) {
		zend_string_release(name);
		return locale_name(category);
	}
	int mask = category == LC_ALL ? LC_ALL_MASK : (1 << category);
	locale_t copy = duplocale(LG(locale));
	bool from_environment = ZSTR_LEN(name) == 0;
	locale_t next =
		copy ? (from_environment ? environment_locale(category, copy) : newlocale(mask, ZSTR_VAL(name), copy))
			 : (locale_t)0;
	zend_string_release(name);
	if (!next) {
		if (copy && !from_environment)
			freelocale(copy);
		return NULL;
	}
	uselocale(next);
	freelocale(LG(locale));
	LG(locale) = next;
	zend_update_current_locale();
	if (category == LC_ALL || category == LC_CTYPE) {
		if (BG(ctype_string))
			zend_string_release(BG(ctype_string));
		BG(ctype_string) = locale_name(LC_CTYPE);
	}
	return locale_name(category);
}
static PHP_FUNCTION(memcp_setlocale) {
	zend_long category;
	zval *args;
	uint32_t count;
	ZEND_PARSE_PARAMETERS_START(2, -1)
	Z_PARAM_LONG(category) Z_PARAM_VARIADIC('+', args, count) ZEND_PARSE_PARAMETERS_END();
	bool valid = category == LC_ALL;
	for (size_t i = 0; i < sizeof(categories) / sizeof(categories[0]); i++)
		if (category == categories[i])
			valid = true;
	if (!valid) {
		zend_argument_value_error(1, "must be a valid locale category");
		RETURN_THROWS();
	}
	for (uint32_t i = 0; i < count; i++) {
		if (Z_TYPE(args[i]) == IS_ARRAY) {
			zval *item;
			ZEND_HASH_FOREACH_VAL(Z_ARRVAL(args[i]), item) {
				zend_string *name = try_locale((int)category, item);
				if (EG(exception))
					RETURN_THROWS();
				if (name)
					RETURN_STR(name);
			}
			ZEND_HASH_FOREACH_END();
		} else {
			zend_string *name = try_locale((int)category, &args[i]);
			if (EG(exception))
				RETURN_THROWS();
			if (name)
				RETURN_STR(name);
		}
	}
	RETURN_FALSE;
}
static PHP_FUNCTION(memcp_textdomain) {
	zend_string *name = NULL;
	ZEND_PARSE_PARAMETERS_START(0, 1) Z_PARAM_OPTIONAL Z_PARAM_STR_OR_NULL(name) ZEND_PARSE_PARAMETERS_END();
	if (name) {
		if (!ZSTR_LEN(name) || ZSTR_LEN(name) > 1024 || zend_string_equals_literal(name, "0") ||
			memchr(ZSTR_VAL(name), 0, ZSTR_LEN(name))) {
			zend_argument_value_error(1, "must be a nonzero domain of 1 to 1024 bytes without null bytes");
			RETURN_THROWS();
		}
		zend_string_release(LG(domain));
		LG(domain) = ZSTR_LEN(name) ? zend_string_copy(name) : zend_string_init("messages", 8, 0);
	}
	RETURN_STR_COPY(LG(domain));
}
static PHP_FUNCTION(memcp_bindtextdomain) {
	zend_string *domain, *directory = NULL;
	ZEND_PARSE_PARAMETERS_START(1, 2)
	Z_PARAM_STR(domain) Z_PARAM_OPTIONAL Z_PARAM_STR_OR_NULL(directory) ZEND_PARSE_PARAMETERS_END();
	if (!ZSTR_LEN(domain) || ZSTR_LEN(domain) > 1024 || memchr(ZSTR_VAL(domain), 0, ZSTR_LEN(domain))) {
		zend_argument_value_error(1, "must be a non-empty domain without null bytes");
		RETURN_THROWS();
	}
	if (directory) {
		if (memchr(ZSTR_VAL(directory), 0, ZSTR_LEN(directory))) {
			zend_argument_value_error(2, "must not contain null bytes");
			RETURN_THROWS();
		}
		char path[MAXPATHLEN];
		if (ZSTR_LEN(directory) && !zend_string_equals_literal(directory, "0")) {
			if (!VCWD_REALPATH(ZSTR_VAL(directory), path))
				RETURN_FALSE;
		} else if (!VCWD_GETCWD(path, MAXPATHLEN))
			RETURN_FALSE;
		zval value;
		ZVAL_STRING(&value, path);
		zend_hash_update(&LG(directories), domain, &value);
	}
	zval *value = zend_hash_find(&LG(directories), domain);
	if (value)
		RETURN_COPY(value);
	RETURN_STRING("/usr/share/locale");
}
static PHP_FUNCTION(memcp_bind_codeset) {
	zend_string *domain, *codeset = NULL;
	ZEND_PARSE_PARAMETERS_START(1, 2)
	Z_PARAM_STR(domain) Z_PARAM_OPTIONAL Z_PARAM_STR_OR_NULL(codeset) ZEND_PARSE_PARAMETERS_END();
	if (!ZSTR_LEN(domain) || ZSTR_LEN(domain) > 1024) {
		zend_argument_value_error(1, "must not be empty");
		RETURN_THROWS();
	}
	if (codeset && memchr(ZSTR_VAL(codeset), 0, ZSTR_LEN(codeset))) {
		zend_argument_value_error(2, "must not contain null bytes");
		RETURN_THROWS();
	}
	if (codeset) {
		zval value;
		ZVAL_STR_COPY(&value, codeset);
		zend_hash_update(&LG(codesets), domain, &value);
	}
	zval *value = zend_hash_find(&LG(codesets), domain);
	if (value)
		RETURN_COPY(value);
	RETURN_FALSE;
}
static PHP_FUNCTION(memcp_gettext_dispatch) {
	const char *function = ZSTR_VAL(execute_data->func->common.function_name);
	bool domain_arg = function[0] == 'd';
	bool plural = strstr(function, "ngettext") != NULL;
	bool category_arg = domain_arg && function[1] == 'c';
	zend_string *domain = LG(domain), *message, *other = NULL;
	zend_long n = 1, category = LC_MESSAGES;
	zend_result parsed;
	if (plural && category_arg)
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "SSSll", &domain, &message, &other, &n, &category);
	else if (plural && domain_arg)
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "SSSl", &domain, &message, &other, &n);
	else if (plural)
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "SSl", &message, &other, &n);
	else if (category_arg)
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "SSl", &domain, &message, &category);
	else if (domain_arg)
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "SS", &domain, &message);
	else
		parsed = zend_parse_parameters(ZEND_NUM_ARGS(), "S", &message);
	if (parsed == FAILURE)
		RETURN_THROWS();
	if (!ZSTR_LEN(domain) || ZSTR_LEN(domain) > 1024 || ZSTR_LEN(message) > 4096 ||
		(other && ZSTR_LEN(other) > 4096)) {
		zend_value_error("Gettext requires a domain of 1 to 1024 bytes and messages up to 4096 bytes");
		RETURN_THROWS();
	}
	bool valid = false;
	for (size_t i = 0; i < sizeof(categories) / sizeof(categories[0]); i++)
		if (category == categories[i])
			valid = true;
	if (!valid) {
		zend_value_error("Invalid gettext locale category");
		RETURN_THROWS();
	}
	zval *directory = zend_hash_find(&LG(directories), domain),
		 *codeset = zend_hash_find(&LG(codesets), domain);
	zend_string *language = request_env("LANGUAGE"), *locale = locale_name((int)category);
	if (!EG(exception)) {
		if (!other)
			other = message;
		memcp_text result =
			memcp_translate(LG(request), directory ? Z_STRVAL_P(directory) : "/usr/share/locale",
							ZSTR_VAL(domain), ZSTR_VAL(locale), ZSTR_VAL(language),
							codeset ? Z_STRVAL_P(codeset) : (char *)nl_langinfo_l(CODESET, LG(locale)),
							ZSTR_VAL(message), ZSTR_LEN(message), ZSTR_VAL(other), ZSTR_LEN(other),
							(long long)n, plural, (char *)category_names[category]);
		if (result.error)
			zend_throw_error(NULL, "%s", result.error);
		else
			RETVAL_STRINGL(result.data ? result.data : "", result.length);
		free(result.data);
		free(result.error);
	}
	zend_string_release(locale);
	zend_string_release(language);
}
static struct {
	const char *name;
	zif_handler replacement, original;
} locale_hooks[] = {{"setlocale", ZEND_FN(memcp_setlocale), NULL},
					{"textdomain", ZEND_FN(memcp_textdomain), NULL},
					{"bindtextdomain", ZEND_FN(memcp_bindtextdomain), NULL},
					{"bind_textdomain_codeset", ZEND_FN(memcp_bind_codeset), NULL},
					{"gettext", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"_", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"dgettext", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"dcgettext", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"ngettext", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"dngettext", ZEND_FN(memcp_gettext_dispatch), NULL},
					{"dcngettext", ZEND_FN(memcp_gettext_dispatch), NULL}};
PHP_MINIT_FUNCTION(memcp_locale) {
	if (!memcp_imap_is_safe()) {
		php_error_docref(NULL, E_CORE_WARNING,
						 "Native IMAP is unsafe in ZTS: use PHPIMAPBinary for the isolated adapter");
		return FAILURE;
	}
	initial_locale = duplocale(LC_GLOBAL_LOCALE);
	if (!initial_locale)
		return FAILURE;
	for (size_t i = 0; i < sizeof(locale_hooks) / sizeof(locale_hooks[0]); i++) {
		zend_function *fn =
			zend_hash_str_find_ptr(CG(function_table), locale_hooks[i].name, strlen(locale_hooks[i].name));
		if (fn && fn->type == ZEND_INTERNAL_FUNCTION) {
			locale_hooks[i].original = fn->internal_function.handler;
			fn->internal_function.handler = locale_hooks[i].replacement;
		}
	}
	return SUCCESS;
}
PHP_MSHUTDOWN_FUNCTION(memcp_locale) {
	for (size_t i = 0; i < sizeof(locale_hooks) / sizeof(locale_hooks[0]); i++) {
		zend_function *fn =
			zend_hash_str_find_ptr(CG(function_table), locale_hooks[i].name, strlen(locale_hooks[i].name));
		if (fn && fn->type == ZEND_INTERNAL_FUNCTION &&
			fn->internal_function.handler == locale_hooks[i].replacement)
			fn->internal_function.handler = locale_hooks[i].original;
	}
	freelocale(initial_locale);
	return SUCCESS;
}
PHP_RINIT_FUNCTION(memcp_locale) {
	LG(locale) = duplocale(initial_locale);
	if (!LG(locale))
		return FAILURE;
	LG(previous) = uselocale(LG(locale));
	zend_update_current_locale();
	zend_hash_init(&LG(directories), 4, NULL, ZVAL_PTR_DTOR, 0);
	zend_hash_init(&LG(codesets), 4, NULL, ZVAL_PTR_DTOR, 0);
	LG(domain) = zend_string_init("messages", 8, 0);
	LG(request) = memcp_text_request_start();
	LG(active) = true;
	return SUCCESS;
}
static zend_result locale_post_deactivate(void) {
	if (LG(active)) {
		zend_hash_destroy(&LG(directories));
		zend_hash_destroy(&LG(codesets));
		zend_string_release(LG(domain));
		memcp_text_request_end(LG(request));
		if (BG(ctype_string)) {
			zend_string_release(BG(ctype_string));
			BG(ctype_string) = NULL;
		}
		uselocale(LG(previous));
		zend_update_current_locale();
		freelocale(LG(locale));
		LG(locale) = 0;
		LG(active) = false;
	}
	return SUCCESS;
}
static zend_module_entry locale_module = {STANDARD_MODULE_HEADER,
										  "memcp_locale",
										  NULL,
										  PHP_MINIT(memcp_locale),
										  PHP_MSHUTDOWN(memcp_locale),
										  PHP_RINIT(memcp_locale),
										  NULL,
										  NULL,
										  "0.1",
										  PHP_MODULE_GLOBALS(memcp_locale),
										  NULL,
										  NULL,
										  locale_post_deactivate,
										  STANDARD_MODULE_PROPERTIES_EX};
void *memcp_locale_module(void) { return &locale_module; }

uintptr_t memcp_php_request(void) { return LG(active) ? LG(request) : 0; }
