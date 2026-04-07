/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

/* format_create_table: build a CREATE TABLE statement from show metadata */
(define format_create_table (lambda (schema tbl) (begin
	(define tblinfo (show schema tbl true))
	(define cols (filter (tblinfo "columns") (lambda (col) (not (col "IsTemp")))))
	(define meta (tblinfo "meta"))
	(define col_defs (map cols (lambda (col)
		(concat "  `" (col "Field") "` " (col "Type")
			(if (not (col "Null")) " NOT NULL" "")
			(if (equal? (col "Extra") "auto_increment") " AUTO_INCREMENT" "")
			(if (not (equal? (col "Comment") "")) (concat " COMMENT '" (col "Comment") "'") "")
		)
	)))
	(define uk_defs (map (meta "Unique") (lambda (uk)
		(concat "  UNIQUE KEY `" (uk "Id") "` ("
			(reduce (map (uk "Cols") (lambda (c) (concat "`" c "`"))) (lambda (a b) (concat a "," b)))
			")")
	)))
	(define all_defs (merge col_defs uk_defs))
	(define body (reduce all_defs (lambda (acc item) (concat acc ",\n" item))))
	(concat "CREATE TABLE `" tbl "` (\n" body
		"\n) ENGINE=" (meta "Engine")
		(if (not (equal? (meta "Collation") "")) (concat " COLLATE=" (meta "Collation")) "")
		(if (not (equal? (meta "Comment") "")) (concat " COMMENT='" (meta "Comment") "'") "")
	)
)))

/* build one INFORMATION_SCHEMA.TABLES row for (schema, tbl) */
(define info_schema_table_row (lambda (schema tbl) (begin
	(define tblinfo (show schema tbl true))
	(define meta (tblinfo "meta"))
	(define shards (tblinfo "shards"))
	(list
		"table_catalog" "def"
		"table_schema" schema
		"table_name" tbl
		"table_type" "BASE TABLE"
		"engine" (meta "Engine")
		"table_rows" (reduce shards (lambda (acc s) (+ acc (+ (s "main_count") (s "delta")) (- 0 (s "deletions")))) 0)
		"data_length" (reduce shards (lambda (acc s) (+ acc (s "size_bytes"))) 0)
		"table_collation" (meta "Collation")
		"table_comment" (meta "Comment")
	)
)))

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
		'("Field" "engine")
		'("Field" "table_rows" "Type" "bigint")
		'("Field" "data_length" "Type" "bigint")
		'("Field" "table_collation")
		'("Field" "table_comment")
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

	'((ignorecase "information_schema") (ignorecase "statistics")) '(
		'("Field" "table_catalog")
		'("Field" "table_schema")
		'("Field" "table_name")
		'("Field" "non_unique")
		'("Field" "index_schema")
		'("Field" "index_name")
		'("Field" "seq_in_index")
		'("Field" "column_name")
		'("Field" "collation")
		'("Field" "cardinality")
		'("Field" "sub_part")
		'("Field" "packed")
		'("Field" "nullable")
		'("Field" "index_type")
		'("Field" "comment")
		'("Field" "index_comment")
	)

	/* Unknown INFORMATION_SCHEMA table → clear SCM-side error */
	'((ignorecase "information_schema") _)
	(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))
	(show schema tbl) /* otherwise: fetch from metadata */
)))
(define scan_wrapper (lambda args (match args (merge '(scanfn schema tbl) rest) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(merge '(scanfn '(session "__memcp_tx") schema 
		'('map '('show) '('lambda '('schema) '('list "catalog_name" "def" "schema_name" 'schema "default_character_set_name" "utf8mb4" "default_collation_name" "utf8mb3_general_ci" "sql_path" NULL "schema_comment" "")))
	) rest)
	'((ignorecase "information_schema") (ignorecase "tables"))
	(merge '(scanfn '(session "__memcp_tx") schema
		'('merge '('map '('show) '('lambda '('schema) '('map '('show 'schema) '('lambda '('tbl) '('info_schema_table_row 'schema 'tbl))))))
	) rest)
	'((ignorecase "information_schema") (ignorecase "columns"))
	(merge '(scanfn '(session "__memcp_tx") schema 
		'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote merge) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote map) '((quote show) (quote schema) (quote tbl)) '((quote lambda) '((quote col)) '((quote list) "table_catalog" "def" "table_schema" (quote schema) "table_name" (quote tbl) "column_name" '((quote col) "Field") "data_type" '((quote col) "RawType") "column_type" '((quote concat) '((quote col) "Type") '((quote col) "Dimensions")))))))))))
	) rest)
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(merge '(scanfn '(session "__memcp_tx") schema '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(merge '(scanfn '(session "__memcp_tx") schema '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "statistics"))
	(merge '(scanfn '(session "__memcp_tx") schema '('merge '('map '('show) '('lambda '('schema) '('merge '('map '('show 'schema) '('lambda '('tbl) '('show 'schema 'tbl "statistics")))))))) rest)
	'((ignorecase "information_schema") (ignorecase "files"))
	(merge '(scanfn '(session "__memcp_tx") schema '(list)) rest) /* empty: MemCP has no tablespaces/undo logs */
	'((ignorecase "information_schema") (ignorecase "partitions"))
	(merge '(scanfn '(session "__memcp_tx") schema '(list)) rest) /* empty: no MySQL partitions */
	'(schema tbl) /* normal case */
	(begin
		(define scan-table-source (lambda (table_source) (match table_source
			'(materialized-subquery key) (list (list (quote context) "session") key)
			'((symbol materialized-subquery) key) (list (list (quote context) "session") key)
			'((quote materialized-subquery) key) (list (list (quote context) "session") key)
			table_source)))
		(merge (list scanfn '(session "__memcp_tx") schema (scan-table-source tbl)) rest))
))))
