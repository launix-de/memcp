/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
/* ANSI_NULLS is an execution setting. Stored source modes are portable
language profiles; the normalized expression records preserve their own mode. */
/* Execute-time state is read by expression plans, never captured in source tags.
https://learn.microsoft.com/sql/t-sql/statements/set-ansi-warnings-transact-sql */
(define tsql_ansi_warnings (lambda (session) (coalesceNil (session "__tsql_ansi_warnings") true)))
(define tsql_ansi_warnings_formula (lambda ()
	(list 'coalesceNil (list 'session "__tsql_ansi_warnings") true)))
(define tsql_arithabort (lambda (session) (coalesceNil (session "__tsql_arithabort") false)))
(define tsql_arithabort_formula (lambda ()
	(list 'coalesceNil (list 'session "__tsql_arithabort") false)))
(define tsql_arithmetic_enabled (lambda (session)
	(or (tsql_ansi_warnings session) (tsql_arithabort session))))
(define tsql_arithmetic_enabled_formula (lambda ()
	(list 'or (tsql_ansi_warnings_formula) (tsql_arithabort_formula))))
(define tsql_concat_null_yields_null (lambda (session)
	(coalesceNil (session "__tsql_concat_null_yields_null") true)))
(define tsql_concat_null_formula (lambda ()
	(list 'coalesceNil (list 'session "__tsql_concat_null_yields_null") true)))
(define tsql_ansi_nulls (lambda (session) (coalesceNil (session "__tsql_ansi_nulls") true)))
(define tsql_set_ansi_nulls (lambda (session enabled)
	(if (session "__tsql_ansi_nulls_locked") nil (begin (session "__tsql_ansi_nulls" enabled) nil))))
/* One admitted option vocabulary serves statement parsing and lexical SET
recognition. Adding another option also requires its actual semantic owner.
https://learn.microsoft.com/sql/t-sql/statements/set-statements-transact-sql */
(define tsql_set_option_profiles (list
	(list "ANSI_NULLS" (list true false)) (list "ANSI_PADDING" (list true false))
	(list "ANSI_WARNINGS" (list true false)) (list "CONCAT_NULL_YIELDS_NULL" (list true false))
	(list "ARITHABORT" (list true false))
	(list "XACT_ABORT" (list true false))
	(list "FMTONLY" (list true false)) (list "NOCOUNT" (list true false))
	(list "NOEXEC" (list true false))
	(list "NUMERIC_ROUNDABORT" (list false)) (list "QUOTED_IDENTIFIER" (list true false))))
(define tsql_set_option_names (map tsql_set_option_profiles car))
(define tsql_validate_set_options (lambda (options enabled) (begin
	(map options (lambda (option) (begin
		(define profile (find tsql_set_option_profiles (lambda (profile) (equal? (car profile) option)) nil))
		(if (nil? profile) (error "unsupported SET option: " option)
			(has? (cadr profile) enabled) true
			(error "unsupported SET option profile: " option (if enabled " ON" " OFF"))))))
	true)))
(define tsql_set_options (lambda (session options enabled) (begin
	/* Validate the complete list before changing any runtime member. */
	(tsql_validate_set_options options enabled)
	(map options (lambda (option) (match option
		"ANSI_NULLS" (tsql_set_ansi_nulls session enabled)
		"ANSI_WARNINGS" (session "__tsql_ansi_warnings" enabled)
		"ARITHABORT" (session "__tsql_arithabort" enabled)
		"XACT_ABORT" (session "__tsql_xact_abort" enabled)
		"CONCAT_NULL_YIELDS_NULL" (session "__tsql_concat_null_yields_null" enabled)
		"FMTONLY" (session "__tsql_fmtonly" enabled)
		"NOCOUNT" (session "__tsql_nocount" enabled)
		/* Declared DECIMAL conversion already rounds at its destination scale.
		Only that existing OFF profile is admitted; ON needs a real precision-
		loss diagnostic and its ARITHABORT-dependent disposition, not a flag.
		https://learn.microsoft.com/sql/t-sql/statements/set-numeric-roundabort-transact-sql */
		"NUMERIC_ROUNDABORT" nil
		"NOEXEC" (session "__tsql_noexec" enabled)
		/* QUOTED_IDENTIFIER has already acted in the lexical owner. Modern
		SQL Server always retains ANSI_PADDING ON, including after OFF. */
		"QUOTED_IDENTIFIER" nil "ANSI_PADDING" nil
		_ (error "unsupported SET option: " option))))
	/* SET resets SQL row-count state but has no affected-data-row count. */
	(tsql_no_count_completion))))
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
(define tsql_source_language_profile (close_procedure ((lambda (profiles)
	(lambda (base language)
		(find profiles (lambda (profile) (equal? language (concat base (car profile)))) nil)))
	tsql_source_profiles)))
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
