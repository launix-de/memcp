/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* SQL owns RESEED's fresh/used distinction and declared integer bounds.
The native allocator owns epoch selection, reservations and original-T recovery.
https://learn.microsoft.com/sql/t-sql/database-console-commands/dbcc-checkident-transact-sql */
(define tsql_checkident_minimum (lambda (type dimensions) (match (toUpper type)
	"TINYINT" 0
	"SMALLINT" -32768
	"INT" -2147483648
	"INTEGER" -2147483648
	"BIGINT" (tsql_parse_integer "-9223372036854775808")
	"DECIMAL" (- 0 (tsql_identity_decimal_maximum dimensions))
	"NUMERIC" (- 0 (tsql_identity_decimal_maximum dimensions))
	_ (error "unsupported IDENTITY declaration"))))

(define tsql_checkident (lambda (command session tx) (begin
	(define parts (tsql_object_name_parts (car command)))
	(if (equal?? (cadr parts) "dbo") true (error "unsupported DBCC CHECKIDENT schema"))
	(define database (tsql_qualifier_database (car parts) (session "schema") session))
	(tsql_require_permission (tsql_session_actor session) database "ALTER")
	(define relation (tsql_resolve_relation database (nth parts 2) session tx))
	(define handle (tsql_native_relation_handle (car relation) (cadr relation) tx))
	(if (nil? handle) (error "DBCC CHECKIDENT requires an existing native table") true)
	(define column (find (show handle "columns" tx)
		(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(if (nil? column) (error "DBCC CHECKIDENT requires an IDENTITY column") true)
	(define physical (coalesceNil (column "StorageField") (column "Field")))
	(define state (table_id_state handle tx))
	(if (equal? (show handle "identity" tx) (state "table_uuid")) true
		(error "DBCC CHECKIDENT allocator generation is unavailable"))
	(define current (coalesceNil (state "current")
		(if (nil? (state "issued")) nil (not (state "issued")) (state "start")
			(state "high_allocated") (state "high") nil)))
	(if (or (nil? current) (nil? (state "issued")))
		(error "DBCC CHECKIDENT requires authoritative allocator provenance") true)
	(define explicit (nth command 2))
	/* Explicit reseeds need no row scan. Automatic correction and NORESEED
	read the identity column using the existing parallel snapshot reduction. */
	(define maximum (if explicit nil (scan tx handle '() '() '() (lambda () true)
		(list physical) (lambda (acc value) (if (nil? value) acc (nil? acc) value (max acc value)))
		nil (lambda (left right) (if (nil? left) right (nil? right) left (max left right))))))
	(define target (if explicit
		(tsql_scalar_value database explicit session tx "assignment" '("BIGINT"))
		(and (not (equal? (cadr command) "NORESEED")) (not (nil? maximum))
			(< (tsql_parse_integer current) maximum)) maximum nil))
	(if explicit (if (nil? target) (error "DBCC CHECKIDENT reseed value cannot be NULL") true) true)
	(if (nil? target) true (begin
		(define type (column "RawType"))
		(define dimensions (coalesceNil (column "Dimensions") '()))
		(if (or (< target (tsql_checkident_minimum type dimensions))
			(> target (tsql_identity_maximum type dimensions)))
			(tsql_raise_error 8115 "Arithmetic overflow converting the reseed value to the IDENTITY type." 1) true)
		(define next (if (not (state "issued")) target
			(equal? target (tsql_parse_integer "9223372036854775807")) nil (+ target 1)))
		(reset_table_ids handle physical (state "token") (concat target)
			(if (nil? next) nil (concat next)) tx)))
	(if (nth command 3) 0 (tsql_emit_message session 0 0 1
		(concat "Checking identity information: current identity value '" current "'"
			(if explicit "." (concat ", current column value '" (if (nil? maximum) "NULL" maximum) "'.")))))
	0)))
