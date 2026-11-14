/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* ROWGUIDCOL is a declaration marker, not a generator or a constraint.
https://learn.microsoft.com/en-us/sql/t-sql/statements/create-table-transact-sql
Only this frontend interprets the namespaced scalar property. The existing
native declaration owns its publication, persistence and metadata lifetime.
Marked ALTER ADD, $ROWGUID and COLUMNPROPERTY require their own atomic
declaration validation or logical binding/catalog contracts and remain
explicitly unsupported by the current grammar. */
(define tsql_row_guid_column? (lambda (column)
	(if (nil? column) false (begin
		(define properties (column "Properties"))
		(if properties (if (properties "tsql.rowguidcol") true false) false)))))
(define tsql_row_guid_definition? (lambda (definition) (match definition
	'("column" _name _type _dimensions attributes)
	(begin (define properties (attributes "properties"))
		(if properties (if (properties "tsql.rowguidcol") true false) false))
	_ false)))
(define tsql_validate_row_guid_definitions (lambda (definitions) (begin
	(define marked (filter definitions tsql_row_guid_definition?))
	(if (> (count marked) 1) (error "only one ROWGUIDCOL column is allowed") true)
	(map marked (lambda (definition)
		(if (equal? (toUpper (nth definition 2)) "UNIQUEIDENTIFIER") true
			(error "ROWGUIDCOL requires the native UNIQUEIDENTIFIER declaration"))))
	(not (empty_list? marked)))))
