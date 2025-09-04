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
		'("Field" "catalog_name")
		'("Field" "schema_name")
		'("Field" "default_character_set_name")
		'("Field" "default_collation_name")
		'("Field" "sql_path")
		'("Field" "schema_comment")
	)

	'((ignorecase "information_schema") (ignorecase "tables")) '(
		'("Field" "table_catalog")
		'("Field" "table_schema")
		'("Field" "table_name")
		'("Field" "table_type")
	)
	'((ignorecase "information_schema") (ignorecase "columns")) '(
		'("Field" "table_catalog")
		'("Field" "table_schema")
		'("Field" "table_name")
		'("Field" "column_name")
		'("Field" "ordinal_position")
		'("Field" "column_default")
		'("Field" "is_nullable")
		'("Field" "data_type")
		/* TODO: CHARACTER_MAXIMUM_LENGTH CHARACTER_OCTET_LENGTH NUMERIC_PRECISION NUMERIC_SCALE DATETIME_PRECISION CHARACTER_SET_NAME COLLATION_NAME  */
		'("Field" "column_type")
		'("Field" "column_key")
		'("Field" "extra")
		'("Field" "privileges")
		'("Field" "column_comment")
		'("Field" "is_generated")
		'("Field" "generation_expression")
	)
	'((ignorecase "information_schema") (ignorecase "key_column_usage")) '(
		'("Field" "constraint_catalog")
		'("Field" "constraint_schema")
		'("Field" "constraint_name")
		'("Field" "table_catalog")
		'("Field" "table_schema")
		'("Field" "table_name")
		'("Field" "column_name")
		'("Field" "ordinal_position")
		'("Field" "position_in_unique_constraint")
		'("Field" "referenced_table_schema")
		'("Field" "referenced_table_name")
		'("Field" "referenced_column_name")
	)
	'((ignorecase "information_schema") (ignorecase "referential_constraints")) '(
		'("Field" "constraint_catalog")
		'("Field" "constraint_schema")
		'("Field" "constraint_name")
		'("Field" "unique_constraint_catalog")
		'("Field" "unique_constraint_schema")
		'("Field" "unique_constraint_name")
		'("Field" "match_option")
		'("Field" "update_rule")
		'("Field" "delete_rule")
		'("Field" "table_name")
		'("Field" "referenced_table_name")
	)

	/* Minimal compatibility for mysqldump probes */
	'((ignorecase "information_schema") (ignorecase "files")) '(
		'("Field" "file_name")
		'("Field" "file_type")
		'("Field" "tablespace_name")
		'("Field" "logfile_group_name")
		'("Field" "total_extents")
		'("Field" "initial_size")
		'("Field" "engine")
		'("Field" "extra")
	)
	'((ignorecase "information_schema") (ignorecase "partitions")) '(
		'("Field" "table_schema")
		'("Field" "table_name")
		'("Field" "partition_name")
		'("Field" "tablespace_name")
	)

	/* Unknown INFORMATION_SCHEMA table → clear SCM-side error */
	'((ignorecase "information_schema") _)
		(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))
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
		(merge '(scanfn schema '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
		(merge '(scanfn schema '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "files"))
		(merge '(scanfn schema '(list)) rest) /* empty: MemCP has no tablespaces/undo logs */
	'((ignorecase "information_schema") (ignorecase "partitions"))
		(merge '(scanfn schema '(list)) rest) /* empty: no MySQL partitions */
	'(schema tbl) /* normal case */
		(merge '(scanfn schema tbl) rest)
))))
