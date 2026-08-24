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

(define sql_metadata_identity (lambda (x) x))

(define quote_mysql_identifier (lambda (id)
	(concat "`" (replace id "`" "``") "`")))

(define quote_mysql_string (lambda (value)
	(concat "'" (replace (replace value "\\" "\\\\") "'" "''") "'")))

(define format_create_database (lambda (schema if_not_exists)
	(concat "CREATE DATABASE " (if if_not_exists "IF NOT EXISTS " "")
		(quote_mysql_identifier schema)
		" CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")))

/* format_create_table: build a CREATE TABLE statement from show metadata */
(define format_create_table (lambda (schema tbl) (begin
	(define tblinfo (show schema tbl true))
	(define cols (filter (tblinfo "columns") (lambda (col) (not (col "IsTemp")))))
	(define meta (tblinfo "meta"))
	(define col_defs (map cols (lambda (col)
		(concat "  `" (col "Field") "` " (col "Type")
			(if (not (col "Null")) " NOT NULL" "")
			(if (nil? (col "Default"))
				(if (col "Null") " DEFAULT NULL" "")
				(concat " DEFAULT " (if (string? (col "Default"))
					(quote_mysql_string (col "Default")) (col "Default"))))
			(if (equal? (col "Extra") "auto_increment") " AUTO_INCREMENT" "")
			(if (not (equal? (col "Comment") "")) (concat " COMMENT " (quote_mysql_string (col "Comment"))) "")
		)
	)))
	(define uk_defs (map (meta "Unique") (lambda (uk)
		(concat "  " (if (equal? (uk "Id") "PRIMARY") "PRIMARY KEY" (concat "UNIQUE KEY " (quote_mysql_identifier (uk "Id")))) " ("
			(reduce (map (uk "Cols") (lambda (c) (concat "`" c "`"))) (lambda (a b) (concat a "," b)))
			")")
	)))
	(define all_defs (merge col_defs uk_defs))
	(define body (reduce all_defs (lambda (acc item) (concat acc ",\n" item))))
	(concat "CREATE TABLE `" tbl "` (\n" body
		"\n) ENGINE=" (meta "Engine")
		(if (not (equal? (meta "Collation") "")) (concat " COLLATE=" (meta "Collation")) "")
		(if (not (equal? (meta "Comment") "")) (concat " COMMENT=" (quote_mysql_string (meta "Comment"))) "")
	)
)))

/* SHOW CREATE TRIGGER must expose executable SQL. SourceSQL contains the
original body, while SHOW TRIGGERS supplies the table, timing, and event. */
(define format_create_trigger (lambda (schema trigger_name) (begin
	(define tr (find (show_triggers schema) (lambda (row)
		(equal? (row "Trigger") trigger_name))))
	(if (nil? tr)
		(error (concat "trigger " trigger_name " does not exist"))
		(list
			"Trigger" trigger_name
			"sql_mode" ""
			"SQL Original Statement" (concat
				"CREATE TRIGGER " (quote_mysql_identifier trigger_name) " "
				(tr "Timing") " " (tr "Event") " ON "
				(quote_mysql_identifier (tr "Table")) " FOR EACH ROW "
				(tr "Statement"))
			"character_set_client" "utf8mb4"
			"collation_connection" "utf8mb4_general_ci"
			"Database Collation" "utf8mb4_general_ci"
			"Created" NULL))
)))

(define info_schema_columns_for_table (lambda (schema table_name) (begin
	(define columns (show schema table_name))
	(map (produceN (count columns)) (lambda (ordinal) (begin
		(define col (nth columns ordinal))
		(list
			"table_catalog" "def"
			"table_schema" schema
			"table_name" table_name
			"column_name" (col "Field")
			"ordinal_position" (+ ordinal 1)
			"column_default" (col "Default")
			"is_nullable" (if (col "Null") "YES" "NO")
			"data_type" (col "RawType")
			"column_type" (col "Type")
			"column_key" (col "Key")
			"extra" (col "Extra")
			"privileges" (col "Privileges")
			"column_comment" (col "Comment")
			"is_generated" "NEVER"
			"generation_expression" "")
	)))
)))

(define info_schema_trigger_rows (lambda ()
	(merge (map (show) (lambda (schema)
		(map (show_triggers schema) (lambda (tr)
			(list
				"trigger_catalog" "def"
				"trigger_schema" schema
				"trigger_name" (tr "Trigger")
				"event_manipulation" (tr "Event")
				"event_object_catalog" "def"
				"event_object_schema" schema
				"event_object_table" (tr "Table")
				"action_order" 1
				"action_condition" NULL
				"action_statement" (tr "Statement")
				"action_orientation" "ROW"
				"action_timing" (tr "Timing")
				"created" (tr "Created")
				"sql_mode" (tr "sql_mode")
				"definer" (tr "Definer")
				"character_set_client" (tr "character_set_client")
				"collation_connection" (tr "collation_connection")
				"database_collation" (tr "Database Collation")))))))))

/* build one INFORMATION_SCHEMA.TABLES row from the catalog snapshot returned
by (show schema true).  Keeping table discovery and metadata collection in one
storage call avoids a DROP racing the second per-table lookup. */
(define info_schema_table_row (lambda (schema tblinfo) (begin
	(list
		"table_catalog" "def"
		"table_schema" schema
		"table_name" (tblinfo "name")
		"table_type" "BASE TABLE"
		"engine" (tblinfo "engine")
		"table_rows" (tblinfo "row_count")
		"data_length" (tblinfo "size_bytes")
		"table_collation" (tblinfo "collation")
		"table_comment" (tblinfo "comment")
	)
)))

/* Column catalog for virtual INFORMATION_SCHEMA relations. Storage-backed
relations are deliberately resolved by the single public get_schema dispatcher. */
(define information_schema_column_catalog (lambda (schema tbl) (match '(schema tbl)
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
		'("Field" "extent_size")
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
	'((ignorecase "information_schema") (ignorecase "triggers")) '(
		'("Field" "trigger_catalog")
		'("Field" "trigger_schema")
		'("Field" "trigger_name")
		'("Field" "event_manipulation")
		'("Field" "event_object_catalog")
		'("Field" "event_object_schema")
		'("Field" "event_object_table")
		'("Field" "action_order" "Type" "bigint")
		'("Field" "action_condition")
		'("Field" "action_statement")
		'("Field" "action_orientation")
		'("Field" "action_timing")
		'("Field" "created")
		'("Field" "sql_mode")
		'("Field" "definer")
		'("Field" "character_set_client")
		'("Field" "collation_connection")
		'("Field" "database_collation")
	)

	/* Unknown INFORMATION_SCHEMA table → clear SCM-side error */
	'((ignorecase "information_schema") _)
	(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))
)))

/* runtime row source for virtual INFORMATION_SCHEMA tables */
(define information_schema_rows (lambda (schema tbl) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(map (show) (lambda (db)
		(list
			"catalog_name" "def"
			"schema_name" db
			"default_character_set_name" "utf8mb4"
			"default_collation_name" "utf8mb3_general_ci"
			"sql_path" NULL
			"schema_comment" "")))

	'((ignorecase "information_schema") (ignorecase "tables"))
	(merge (map (show) (lambda (db)
		(map (show db true) (lambda (table_info)
			(info_schema_table_row db table_info))))))

	'((ignorecase "information_schema") (ignorecase "columns"))
	(merge (map (show) (lambda (db)
		(merge (map (show db) (lambda (table_name)
			(info_schema_columns_for_table db table_name)))))))

	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(list)

	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(list)

	'((ignorecase "information_schema") (ignorecase "statistics"))
	(merge (map (show) (lambda (db)
		(merge (map (show db) (lambda (table_name)
			(show db table_name (sql_metadata_identity "statistics"))))))))

	'((ignorecase "information_schema") (ignorecase "triggers"))
	(info_schema_trigger_rows)

	'((ignorecase "information_schema") (ignorecase "files"))
	(list)

	'((ignorecase "information_schema") (ignorecase "partitions"))
	(list)

	'((ignorecase "information_schema") _)
	(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))

	(show schema tbl)
)))

(define scan_wrapper (lambda args (match args (merge '(scanfn schema tbl) rest) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(merge '(scanfn '(session "__memcp_tx")
		'('map '('show) '('lambda '('schema) '('list "catalog_name" "def" "schema_name" 'schema "default_character_set_name" "utf8mb4" "default_collation_name" "utf8mb3_general_ci" "sql_path" NULL "schema_comment" "")))
	) rest)
	'((ignorecase "information_schema") (ignorecase "tables"))
	(list 'begin
		/* TODO(planner-scalability): expose catalog metadata as a physical
		relation instead of constructing a cardinality-dependent SCM list. */
		/* Materialize a complete catalog snapshot at runtime but BEFORE the scan
		starts.  The single (show schema true) call also keeps concurrent DROP from
		invalidating names between table discovery and metadata collection. */
		'('define '__info_tables_data '('merge '('map '('show) '('lambda '('s) '('map '('show 's true) '('lambda '('t) '('info_schema_table_row 's 't)))))))
		(merge '(scanfn '(session "__memcp_tx") '__info_tables_data) rest))
	'((ignorecase "information_schema") (ignorecase "columns"))
	(merge '(scanfn '(session "__memcp_tx")
		'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote merge) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote info_schema_columns_for_table) (quote schema) (quote tbl))))))))
	) rest)
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(merge '(scanfn '(session "__memcp_tx") '(list)) rest) /* TODO: list constraints */
	'((ignorecase "information_schema") (ignorecase "statistics"))
	(merge '(scanfn '(session "__memcp_tx") '('merge '('map '('show) '('lambda '('schema) '('merge '('map '('show 'schema) '('lambda '('tbl) '('show 'schema 'tbl "statistics")))))))) rest)
	'((ignorecase "information_schema") (ignorecase "triggers"))
	(merge '(scanfn '(session "__memcp_tx") '('info_schema_trigger_rows)) rest)
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
			'(scan-tagged-table base _ _ _ _ _) (scan-table-source base)
			'((symbol scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'((quote scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'(materialized-subquery key) (list (list (quote context) "session") key)
			'((symbol materialized-subquery) key) (list (list (quote context) "session") key)
			'((quote materialized-subquery) key) (list (list (quote context) "session") key)
			table_source)))
		(define tbl_resolved (scan-table-source tbl))
		/* materialized subqueries produce list expressions — pass as-is; real tables get (table schema name) */
		(define tbl_arg (if (string? tbl_resolved) (list 'table schema tbl_resolved) tbl_resolved))
		(merge (list scanfn '(session "__memcp_tx") tbl_arg) rest))
))))
