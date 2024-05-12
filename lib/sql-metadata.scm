/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* emulate metadata tables */
(define get_schema (lambda (schema tbl) (match '(schema tbl)
	/* special tables */
	'((ignorecase "information_schema") (ignorecase "schemata")) '(
		'("name" "catalog_name")
		'("name" "schema_name")
		'("name" "default_character_set_name")
		'("name" "default_collation_name")
		'("name" "sql_path")
		'("name" "schema_comment")
	)

	'((ignorecase "information_schema") (ignorecase "tables")) '(
		'("name" "table_catalog")
		'("name" "table_schema")
		'("name" "table_name")
		'("name" "table_type")
	)
	'((ignorecase "information_schema") (ignorecase "columns")) '(
		'("name" "table_catalog")
		'("name" "table_schema")
		'("name" "table_name")
		'("name" "column_name")
		'("name" "ordinal_position")
		'("name" "column_default")
		'("name" "is_nullable")
		'("name" "data_type")
		/* TODO: CHARACTER_MAXIMUM_LENGTH CHARACTER_OCTET_LENGTH NUMERIC_PRECISION NUMERIC_SCALE DATETIME_PRECISION CHARACTER_SET_NAME COLLATION_NAME  */
		'("name" "column_type")
		'("name" "column_key")
		'("name" "extra")
		'("name" "privileges")
		'("name" "column_comment")
		'("name" "is_generated")
		'("name" "generation_expression")
	)
	'((ignorecase "information_schema") (ignorecase "key_column_usage")) '(
		'("name" "constraint_name")
		'("name" "table_schema")
		'("name" "table_name")
		'("name" "column_name")
		'("name" "ordinal_position")
	)
	(show schema tbl) /* otherwise: fetch from metadata */
)))
(define scan_wrapper (lambda args (match args (merge '(scanfn schema tbl) rest) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
		(merge '(scanfn schema 
			'('map '('show) '('lambda '('schema) '('list "catalog_name" "def" "schema_name" 'schema "default_character_set_name" "utf8mb4" "default_collation_name" "utf8mb3_general_ci" "sql_path" NULL "schema_comment" "")))
			) rest)
	'((ignorecase "information_schema") (ignorecase "tables"))
		(merge '(scanfn schema 
			'('merge '('map '('show) '('lambda '('schema) '('map '('show 'schema) '('lambda '('tbl) '('list "table_catalog" "def" "table_schema" 'schema "table_name" 'tbl "table_type" "BASE TABLE")))))) 
			) rest)
	'((ignorecase "information_schema") (ignorecase "columns"))
		(merge '(scanfn schema 
			'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote merge) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote map) '((quote show) (quote schema) (quote tbl)) '((quote lambda) '((quote col)) '((quote list) "table_catalog" "def" "table_schema" (quote schema) "table_name" (quote tbl) "column_name" '((quote col) "name") "data_type" '((quote col) "type") "column_type" '((quote concat) '((quote col) "type") '((quote col) "dimensions")))))))))))
			) rest)
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
		(merge '(scanfn schema tbl) rest) /* TODO: list constraints */
	'(schema tbl) /* normal case */
		(merge '(scanfn schema tbl) rest)
))))

