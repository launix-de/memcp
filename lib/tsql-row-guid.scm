/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* ROWGUIDCOL is a declaration marker, not a generator or a constraint.
https://learn.microsoft.com/en-us/sql/t-sql/statements/create-table-transact-sql
Only this frontend interprets the namespaced scalar property. The existing
native declaration owns its publication, persistence and metadata lifetime.
$ROWGUID and COLUMNPROPERTY require their own logical binding/catalog
contracts and remain explicitly unsupported by the current grammar. */
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

/* DROP changes only the declaration marker, retaining values, DEFAULT and
other namespaced properties. The native owner removes the named property
from its actual immutable predecessor under the original transaction.
https://learn.microsoft.com/sql/t-sql/statements/alter-table-transact-sql */
(define tsql_drop_row_guid (lambda (database name column tx) (begin
	(define handle (table database false name false tx))
	(define declaration (if handle (show handle "column" column true tx) nil))
	(if (nil? declaration)
		(tsql_raise_error 4924 (concat "ALTER COLUMN failed because column '" column "' does not exist in table '" name "'.") 1) true)
	(define missing (lambda ()
		(tsql_raise_error 4926 (concat "ALTER COLUMN DROP ROWGUIDCOL failed because column '" column "' has no ROWGUIDCOL property.") 1)))
	(if (tsql_row_guid_column? declaration) true (missing))
	(if (altercolumn handle (declaration "Field") "drop_property" "tsql.rowguidcol" tx) true (missing)))))

/* Claim publication is atomic in the native declaration owner. The frontend
owns only direct-type eligibility and the meaning of this namespaced marker;
there is no preliminary table-wide scan whose result could race publication. */
(define tsql_add_row_guid (lambda (database name column tx) (begin
	(define handle (table database false name false tx))
	(define declaration (if handle (show handle "column" column true tx) nil))
	(if (nil? declaration)
		(tsql_raise_error 4924 (concat "ALTER COLUMN failed because column '" column "' does not exist in table '" name "'.") 1) true)
	(if (equal? (toUpper (declaration "RawType")) "UNIQUEIDENTIFIER") true
		(error "ROWGUIDCOL requires the native UNIQUEIDENTIFIER declaration"))
	(altercolumn handle (declaration "Field") "claim_property"
		(list "tsql.rowguidcol" true
			(list (declaration "RawType") (coalesceNil (declaration "Dimensions") '())
				(declaration "Null") (coalesceNil (declaration "Collation") ""))) tx))))
