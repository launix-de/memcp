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
	'((ignorecase "information_schema") (ignorecase "tables")) '(
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
(define scan_wrapper (lambda (schema tbl filtercols filter mapcols map reduce neutral) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "tables"))
		'((quote scan) schema 
			'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote list) "table_schema" (quote schema) "table_name" (quote tbl) "table_type" "BASE TABLE")))))) 
			filtercols filter mapcols map reduce neutral)
	'((ignorecase "information_schema") (ignorecase "columns"))
		'((quote scan) schema 
			'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote merge) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote map) '((quote show) (quote schema) (quote tbl)) '((quote lambda) '((quote col)) '((quote list) "table_catalog" "def" "table_schema" (quote schema) "table_name" (quote tbl) "column_name" '((quote col) "name") "data_type" '((quote col) "type") "column_type" '((quote concat) '((quote col) "type") '((quote col) "dimensions")))))))))))
			filtercols filter mapcols map reduce neutral)
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
		'(list) /* TODO: list constraints */
	'(schema tbl) /* normal case */
		'((quote scan) schema tbl filtercols filter mapcols map reduce neutral)
)))

