/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
/* ANSI_NULLS is an execution setting. Stored source modes are portable
language profiles; the normalized expression records preserve their own mode. */
/* Execute-time state is read by expression plans, never captured in source tags.
https://learn.microsoft.com/sql/t-sql/statements/set-ansi-warnings-transact-sql */
(define tsql_ansi_warnings (lambda (session) (coalesceNil (session "__tsql_ansi_warnings") true)))
(define tsql_ansi_warnings_formula (lambda ()
	(list 'coalesceNil (list 'session "__tsql_ansi_warnings") true)))
(define tsql_ansi_nulls (lambda (session) (coalesceNil (session "__tsql_ansi_nulls") true)))
(define tsql_set_ansi_nulls (lambda (session enabled)
	(if (session "__tsql_ansi_nulls_locked") nil (begin (session "__tsql_ansi_nulls" enabled) nil))))
(define tsql_ansi_planning_scope (lambda (parent enabled) (begin
	(define parent (coalesceNil parent (newsession)))
	(lambda args (match args
		'(key) (if (equal? key "__tsql_ansi_nulls") enabled (parent key))
		_ (apply parent args))))))
(define tsql_quoted_identifier (lambda (session)
	(coalesceNil (session "__tsql_quoted_identifier") true)))

/* Source tags persist two independent SQL setting bits. Old source tags
retain ANSI_NULLS ON when reloaded. No invocation session enters the catalog. */
(define tsql_source_profiles (list (list "" true true) (list "-quoted-off" false true)
	(list "-ansi-off" true false) (list "-quoted-off-ansi-off" false false)))
(define tsql_source_language_profile (lambda (base language)
	(find tsql_source_profiles (lambda (profile) (equal? language (concat base (car profile)))) nil)))
(define tsql_source_language_settings (lambda (language) (begin
	(define profile (find (map '("tsql-procedure" "tsql-statement" "tsql-scalar-function" "tsql-relational-scalar-function" "tsql-table-function")
		(lambda (base) (tsql_source_language_profile base language))) (lambda (profile) profile) nil))
	(if profile (cdr profile) (error "unsupported T-SQL source language")))))
(define tsql_source_language (lambda (base session)
	(concat base (if (tsql_quoted_identifier session) "" "-quoted-off")
		(if (tsql_ansi_nulls session) "" "-ansi-off"))))
(define tsql_register_source_profiles (lambda (base compiler register)
	(map tsql_source_profiles (lambda (profile)
		(register (concat base (car profile))
			(close_procedure ((lambda (compiler quoted ansi)
				(lambda (source context) (compiler source (set_assoc context "__tsql_ansi_nulls" ansi) quoted)))
				compiler (cadr profile) (nth profile 2))))))))
