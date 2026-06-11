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

/* INFORMATION_SCHEMA queries from MySQL clients use canonical upper-case
column names, while existing MemCP tests and some generated SQL use lower-case
unquoted names. Until the top-down pipeline canonicalizes these projections
reliably, expose both row keys to keep metadata lookups non-fatal. */
(define info_schema_dual_row (lambda (upper_row)
	(merge upper_row
		(reduce_assoc upper_row (lambda (acc key value)
			(merge acc (list (toLower key) value)))
			'()))))

(define info_schema_dual_columns (lambda (cols)
	(merge
		(map cols (lambda (col) (list "Field" col)))
		(map cols (lambda (col) (list "Field" (toLower col)))))))

/* build one INFORMATION_SCHEMA.TABLES row for (schema, tbl) */
(define info_schema_table_row (lambda (schema tbl) (begin
	(define tblinfo (show schema tbl true))
	(define meta (tblinfo "meta"))
	(define shards (tblinfo "shards"))
	(info_schema_dual_row (list
		"TABLE_CATALOG" "def"
		"TABLE_SCHEMA" schema
		"TABLE_NAME" tbl
		"TABLE_TYPE" "BASE TABLE"
		"ENGINE" (meta "Engine")
		"TABLE_ROWS" (reduce shards (lambda (acc s) (+ acc (+ (s "main_count") (s "delta")) (- 0 (s "deletions")))) 0)
		"DATA_LENGTH" (reduce shards (lambda (acc s) (+ acc (s "size_bytes"))) 0)
		"TABLE_COLLATION" (meta "Collation")
		"TABLE_COMMENT" (meta "Comment")
	))
)))

/* emulate metadata tables */
(define get_schema (lambda (schema tbl) (match '(schema tbl)
	/* special tables */
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(info_schema_dual_columns '("CATALOG_NAME" "SCHEMA_NAME" "DEFAULT_CHARACTER_SET_NAME" "DEFAULT_COLLATION_NAME" "SQL_PATH" "SCHEMA_COMMENT"))

	'((ignorecase "information_schema") (ignorecase "tables"))
	(info_schema_dual_columns '("TABLE_CATALOG" "TABLE_SCHEMA" "TABLE_NAME" "TABLE_TYPE" "ENGINE" "TABLE_ROWS" "DATA_LENGTH" "TABLE_COLLATION" "TABLE_COMMENT"))
	'((ignorecase "information_schema") (ignorecase "columns"))
	/* TODO: CHARACTER_MAXIMUM_LENGTH CHARACTER_OCTET_LENGTH NUMERIC_PRECISION NUMERIC_SCALE DATETIME_PRECISION CHARACTER_SET_NAME COLLATION_NAME  */
	(info_schema_dual_columns '("TABLE_CATALOG" "TABLE_SCHEMA" "TABLE_NAME" "COLUMN_NAME" "ORDINAL_POSITION" "COLUMN_DEFAULT" "IS_NULLABLE" "DATA_TYPE" "COLUMN_TYPE" "COLUMN_KEY" "EXTRA" "PRIVILEGES" "COLUMN_COMMENT" "IS_GENERATED" "GENERATION_EXPRESSION"))
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(info_schema_dual_columns '("CONSTRAINT_CATALOG" "CONSTRAINT_SCHEMA" "CONSTRAINT_NAME" "TABLE_CATALOG" "TABLE_SCHEMA" "TABLE_NAME" "COLUMN_NAME" "ORDINAL_POSITION" "POSITION_IN_UNIQUE_CONSTRAINT" "REFERENCED_TABLE_SCHEMA" "REFERENCED_TABLE_NAME" "REFERENCED_COLUMN_NAME"))
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(info_schema_dual_columns '("CONSTRAINT_CATALOG" "CONSTRAINT_SCHEMA" "CONSTRAINT_NAME" "UNIQUE_CONSTRAINT_CATALOG" "UNIQUE_CONSTRAINT_SCHEMA" "UNIQUE_CONSTRAINT_NAME" "MATCH_OPTION" "UPDATE_RULE" "DELETE_RULE" "TABLE_NAME" "REFERENCED_TABLE_NAME"))

	/* Minimal compatibility for mysqldump probes */
	'((ignorecase "information_schema") (ignorecase "files"))
	(info_schema_dual_columns '("FILE_NAME" "FILE_TYPE" "TABLESPACE_NAME" "LOGFILE_GROUP_NAME" "TOTAL_EXTENTS" "INITIAL_SIZE" "ENGINE" "EXTRA"))
	'((ignorecase "information_schema") (ignorecase "partitions"))
	(info_schema_dual_columns '("TABLE_SCHEMA" "TABLE_NAME" "PARTITION_NAME" "TABLESPACE_NAME"))

	'((ignorecase "information_schema") (ignorecase "statistics"))
	(info_schema_dual_columns '("TABLE_CATALOG" "TABLE_SCHEMA" "TABLE_NAME" "NON_UNIQUE" "INDEX_SCHEMA" "INDEX_NAME" "SEQ_IN_INDEX" "COLUMN_NAME" "COLLATION" "CARDINALITY" "SUB_PART" "PACKED" "NULLABLE" "INDEX_TYPE" "COMMENT" "INDEX_COMMENT"))

	/* Unknown INFORMATION_SCHEMA table → clear SCM-side error */
	'((ignorecase "information_schema") _)
	(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))
	(show schema tbl) /* otherwise: fetch from metadata */
)))
(define scan_wrapper (lambda args (match args (merge '(scanfn schema tbl) rest) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(merge '(scanfn '(session "__memcp_tx")
		'('map '('show) '('lambda '('schema) '('info_schema_dual_row '('list "CATALOG_NAME" "def" "SCHEMA_NAME" 'schema "DEFAULT_CHARACTER_SET_NAME" "utf8mb4" "DEFAULT_COLLATION_NAME" "utf8mb3_general_ci" "SQL_PATH" NULL "SCHEMA_COMMENT" ""))))
	) rest)
	'((ignorecase "information_schema") (ignorecase "tables"))
	(list 'begin
		/* Materialize the table list at runtime but BEFORE the scan starts, so
		info_schema_table_row's (show schema tbl true) calls do not execute inside
		a scan callback where locks are held (which deadlocks). */
		'('define '__info_tables_data '('merge '('map '('show) '('lambda '('s) '('map '('show 's) '('lambda '('t) '('info_schema_table_row 's 't)))))))
		(merge '(scanfn '(session "__memcp_tx") '__info_tables_data) rest))
	'((ignorecase "information_schema") (ignorecase "columns"))
	(merge '(scanfn '(session "__memcp_tx")
		'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote merge) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote map) '((quote show) (quote schema) (quote tbl)) '((quote lambda) '((quote col)) '((quote info_schema_dual_row) '((quote list) "TABLE_CATALOG" "def" "TABLE_SCHEMA" (quote schema) "TABLE_NAME" (quote tbl) "COLUMN_NAME" '((quote col) "Field") "DATA_TYPE" '((quote col) "RawType") "COLUMN_TYPE" '((quote concat) '((quote col) "Type") '((quote col) "Dimensions"))))))))))))
	) rest)
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "statistics"))
	(merge '(scanfn '(session "__memcp_tx") '('merge '('map '('show) '('lambda '('schema) '('merge '('map '('show 'schema) '('lambda '('tbl) '('show 'schema 'tbl "statistics")))))))) rest)
	'((ignorecase "information_schema") (ignorecase "files"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* empty: MemCP has no tablespaces/undo logs */
	'((ignorecase "information_schema") (ignorecase "partitions"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* empty: no MySQL partitions */
	'(schema tbl) /* normal case */
	(begin
		/* scan helpers receive a runtime source as their table argument.
		Materialized subqueries are stored in the session and therefore must be
		lowered to ((context "session") key) before scan/scan_order/scan_batch
		see them. Do not stringify this source and do not add table-name
		fallbacks in Go for it. */
		(define scan-table-source (lambda (table_source) (match table_source
			'(unnest_helper_table helper_schema base _)
			(scan-table-source (if (string? base) (list 'table helper_schema base) base))
			'((symbol unnest_helper_table) helper_schema base _)
			(scan-table-source (if (string? base) (list 'table helper_schema base) base))
			'((quote unnest_helper_table) helper_schema base _)
			(scan-table-source (if (string? base) (list 'table helper_schema base) base))
			'(scan-tagged-table base _ _ _ _ _) (scan-table-source base)
			'(scan-tagged-table base _ _ _ _ _ _) (scan-table-source base)
			'((symbol scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'((symbol scan-tagged-table) base _ _ _ _ _ _) (scan-table-source base)
			'((quote scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'((quote scan-tagged-table) base _ _ _ _ _ _) (scan-table-source base)
			'(materialized-subquery key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'((symbol materialized-subquery) key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'((quote materialized-subquery) key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'(materialized-subquery-source key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'((symbol materialized-subquery-source) key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'((quote materialized-subquery-source) key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'((context "session") key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'(((symbol context) "session") key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			'(((quote context) "session") key) (list (list (quote context) "session")
				(if (string? key) key (list (quote quote) key)))
			table_source)))
		(define tbl_resolved (scan-table-source tbl))
		/* materialized subqueries produce list expressions — pass as-is; real tables get (table schema name) */
		(define tbl_arg (if (string? tbl_resolved) (list 'table schema tbl_resolved) tbl_resolved))
		(merge (list scanfn '(session "__memcp_tx") tbl_arg) rest))
))))
