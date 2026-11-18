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

/* Resolve SQL table defaults into column metadata before storage sees DDL.
The planner already uses that metadata with the canonical collate/Less factory.
An explicit column COLLATE wins; non-character types retain their existing
ordering. The inherited value is stored, so ADD COLUMN and restart agree. */
(define sql_column_collation_defaults (lambda (typ attributes collation)
	(if (and (string? collation) (not (equal? collation ""))
		(nil? (get_assoc attributes "collate"))
		(has? '("char" "varchar" "tinytext" "text" "mediumtext" "longtext") (toLower typ)))
		(merge attributes (list "collate" collation)) attributes)))

(define sql_create_table (lambda (schema name columns options ifnotexists tx)
	(createtable schema name
		(map columns (lambda (definition)
			(match definition
				'("column" col typ dimensions attributes)
				(list "column" col typ dimensions
					(sql_column_collation_defaults typ attributes (get_assoc options "collation")))
				_ definition))) options ifnotexists tx)))

(define sql_create_column (lambda (relation name typ dimensions attributes)
	(createcolumn relation name typ dimensions
		(sql_column_collation_defaults typ attributes
			(get_assoc (show relation true) "Collation")))))

/* ADD freezes one value for historical rows. Keep the existing evaluation
path for literal arithmetic and casts; reject volatile or relational defaults
before emitting DDL. Clock defaults use the storage's statement-stable marker. */
(define sql_add_default_constant? (lambda (value)
	(or (plain_literal_expr? value) (logical_sql_null_literal? value)
		(match value
			(cons operator arguments)
			(and (has? (list (quote +) (quote -) (quote *) (quote /)
				(quote mod) (quote div_null) (quote intdiv) (quote simplify)
				(quote concat) (quote sql_concat)
				+ - * / mod div_null intdiv simplify concat sql_concat) operator)
				(reduce arguments (lambda (valid argument)
					(and valid (sql_add_default_constant? argument))) true))
			_ false))))

(define sql_add_column_attributes (lambda (attributes)
	(match attributes
		(cons key (cons value rest)) (begin
			(define tail (sql_add_column_attributes rest))
			/* CAST shares the frontend expression compiler. Validate its emitted
			constant formula with the same whitelist as literal arithmetic. */
			(define value (if (equal? key "default")
				(sql_info_formula (sql_expr_info '() value)) value))
			(if (equal? key "default")
				(if (equal? value (list (quote now)))
					(merge '("default_expression" "CURRENT_TIMESTAMP") tail)
					(if (sql_add_default_constant? value)
						(merge (list key value) tail)
						(error "ADD COLUMN default expression is unsupported")))
				(merge (list key value) tail)))
		_ '())))

/* SQL system-variable defaults and state have one owner. Both @@ reads and
SHOW VARIABLES use this catalog; wire frontends only transport their results.
Keep the 40 MiB packet advertisement used by PDO on the MySQL frontend. */
(define sql_system_variable_defaults (list
	(list "version" "0.9")
	(list "version_comment" runtime_os)
	(list "lower_case_table_names" 0)
	(list "character_set_server" "utf8mb4")
	(list "collation_server" "utf8mb4_general_ci")
	(list "character_set_client" "utf8mb4")
	(list "character_set_results" "utf8mb4")
	(list "collation_connection" "utf8mb4_general_ci")
	(list "collation_database" "utf8mb4_general_ci")
	(list "sql_mode" "")
	(list "unique_checks" 1)
	(list "foreign_key_checks" 1)
	(list "sql_notes" 1)
	(list "time_zone" "UTC")
	(list "system_time_zone" (system_time_zone))
	(list "key_buffer_size" 0)
	(list "max_allowed_packet" 41943040)
	(list "max_connections" 151)
	(list "innodb_buffer_pool_size" 0)))
(define globalvars (newsession))
(reduce sql_system_variable_defaults (lambda (_ entry) (globalvars (car entry) (cadr entry))) nil)

/* Execution shadows session_globalvar with its request-local reader. */
(define session_globalvar (lambda (key) (globalvars key)))
/* Explicit lexical capture keeps this helper portable with a delegated plan.
The native session owner supplies the invocation's immutable read view. */
(define session_globalvar_explicit ((lambda (globals)
	(lambda (session key) (coalesceNil (session key) (globals key)))) globalvars))

/* Build row expressions so mutable values are read when a cached SHOW runs. */
(define sql_system_variable_rows (lambda ()
	(map sql_system_variable_defaults (lambda (entry)
		(list (car entry) (list (quote session_globalvar) (car entry)))))))

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
	(define auto_increment_col (find cols (lambda (col)
		(equal? (col "Extra") "auto_increment"))))
	(define col_defs (map cols (lambda (col)
		(concat "  " (quote_mysql_identifier (col "Field")) " " (col "Type")
			(if (not (col "Null")) " NOT NULL" "")
			(if (not (equal? (col "DefaultExpression") ""))
				(concat " DEFAULT " (col "DefaultExpression"))
				(if (nil? (col "Default"))
					(if (col "Null") " DEFAULT NULL" "")
					(concat " DEFAULT " (if (string? (col "Default"))
						(quote_mysql_string (col "Default")) (col "Default")))))
			(if (equal? (col "Extra") "auto_increment") " AUTO_INCREMENT" "")
			(if (not (equal? (col "Comment") "")) (concat " COMMENT " (quote_mysql_string (col "Comment"))) "")
		)
	)))
	(define uk_defs (map (meta "Unique") (lambda (uk)
		(concat "  " (if (equal? (uk "Id") "PRIMARY") "PRIMARY KEY" (concat "UNIQUE KEY " (quote_mysql_identifier (uk "Id")))) " ("
			(reduce (map (uk "Cols") quote_mysql_identifier) (lambda (a b) (concat a "," b)))
			")")
	)))
	(define fk_defs (map (filter (meta "ForeignKeys") (lambda (fk)
		(equal? (fk "Role") "referencing"))) (lambda (fk)
			(concat "  CONSTRAINT " (quote_mysql_identifier (fk "Id")) " FOREIGN KEY ("
				(reduce (map (fk "LocalColumns") quote_mysql_identifier) (lambda (a b) (concat a "," b)))
				") REFERENCES " (quote_mysql_identifier (fk "OtherTable")) " ("
				(reduce (map (fk "OtherColumns") quote_mysql_identifier) (lambda (a b) (concat a "," b)))
				") ON DELETE " (toUpper (fk "DeleteMode"))
				" ON UPDATE " (toUpper (fk "UpdateMode")))
	)))
	(define all_defs (merge col_defs uk_defs fk_defs))
	(define body (reduce all_defs (lambda (acc item) (concat acc ",\n" item))))
	(concat "CREATE TABLE " (quote_mysql_identifier tbl) " (\n" body
		"\n) ENGINE=" (if (equal? (meta "Engine") "memory") "MEMORY" "InnoDB")
		(if (and (not (nil? auto_increment_col)) (> (meta "AutoIncrement") 1))
			(concat " AUTO_INCREMENT=" (meta "AutoIncrement")) "")
		(if (not (equal? (meta "Collation") "")) (concat " COLLATE=" (meta "Collation")) "")
		(if (not (equal? (meta "Comment") "")) (concat " COMMENT=" (quote_mysql_string (meta "Comment"))) "")
	)
)))

/* SHOW CREATE TRIGGER exposes SQL-language source. SHOW TRIGGERS supplies the
table, timing, and event around that language-neutral persisted source. */
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

(define info_schema_column_default (lambda (col)
	(if (not (equal? (col "DefaultExpression") ""))
		(if (equal? (toUpper (col "DefaultExpression")) "CURRENT_TIMESTAMP")
			"current_timestamp()" (col "DefaultExpression"))
		(if (nil? (col "Default"))
			(if (col "Null") "NULL" NULL)
			(if (string? (col "Default"))
				(quote_mysql_string (col "Default"))
				(concat (col "Default")))))))

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
			"column_default" (info_schema_column_default col)
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

(define info_schema_fk_column_rows (lambda (schema tbl fk local_columns other_columns ordinal)
	(if (empty_list? local_columns)
		(if (empty_list? other_columns) '()
			(error (concat "foreign key " (fk "Id") " has mismatched column lists")))
		(if (empty_list? other_columns)
			(error (concat "foreign key " (fk "Id") " has mismatched column lists"))
			(cons (list "constraint_catalog" "def" "constraint_schema" schema
				"constraint_name" (fk "Id") "table_catalog" "def"
				"table_schema" schema "table_name" tbl
				"column_name" (car local_columns)
				"ordinal_position" ordinal
				"position_in_unique_constraint" ordinal
				"referenced_table_schema" schema
				"referenced_table_name" (fk "OtherTable")
				"referenced_column_name" (car other_columns))
				(info_schema_fk_column_rows schema tbl fk
					(cdr local_columns) (cdr other_columns) (+ ordinal 1)))))))

/* A public constraint name is native key metadata; legacy keys use their ID. */
(define sql_key_constraint_name (lambda (key)
	(if (or (nil? (key "constraint_name")) (equal? (key "constraint_name") ""))
		(key "Id") (key "constraint_name"))))

(define info_schema_unique_column_rows (lambda (schema tbl key columns ordinal)
	(if (empty_list? columns) '()
		(cons (list "constraint_catalog" "def" "constraint_schema" schema
			"constraint_name" (sql_key_constraint_name key) "table_catalog" "def"
			"table_schema" schema "table_name" tbl
			"column_name" (car columns)
			"ordinal_position" ordinal
			"position_in_unique_constraint" NULL
			"referenced_table_schema" NULL
			"referenced_table_name" NULL
			"referenced_column_name" NULL)
			(info_schema_unique_column_rows schema tbl key (cdr columns) (+ ordinal 1))))))

(define info_schema_fk_unique_name (lambda (schema fk) (begin
	(define referenced_info (show schema (fk "OtherTable") true))
	(define referenced_meta (referenced_info "meta"))
	(define referenced_key (find (referenced_meta "Unique") (lambda (key)
		(equal? (key "Cols") (fk "OtherColumns")))))
	(if (nil? referenced_key) NULL (sql_key_constraint_name referenced_key))
)))

(define info_schema_key_column_usage_rows (lambda ()
	(merge (map (show) (lambda (schema)
		(merge (map (show schema) (lambda (tbl) (begin
			(define tblinfo (show schema tbl true))
			(define meta (tblinfo "meta"))
			(define unique_rows (merge (map (meta "Unique") (lambda (key)
				(info_schema_unique_column_rows schema tbl key (key "Cols") 1)))))
			(define foreign_rows (merge (map (filter (meta "ForeignKeys") (lambda (fk)
				(equal? (fk "Role") "referencing"))) (lambda (fk)
					(info_schema_fk_column_rows schema tbl fk
						(fk "LocalColumns") (fk "OtherColumns") 1)))))
			(merge unique_rows foreign_rows)
)))))))))

(define info_schema_referential_constraint_rows (lambda ()
	(merge (map (show) (lambda (schema)
		(merge (map (show schema) (lambda (tbl) (begin
			(define tblinfo (show schema tbl true))
			(define meta (tblinfo "meta"))
			(map (filter (meta "ForeignKeys") (lambda (fk)
				(equal? (fk "Role") "referencing"))) (lambda (fk)
					(list "constraint_catalog" "def" "constraint_schema" schema
						"constraint_name" (fk "Id")
						"unique_constraint_catalog" "def"
						"unique_constraint_schema" schema
						"unique_constraint_name" (info_schema_fk_unique_name schema fk)
						"match_option" "NONE"
						"update_rule" (toUpper (fk "UpdateMode"))
						"delete_rule" (toUpper (fk "DeleteMode"))
						"table_name" tbl
						"referenced_table_name" (fk "OtherTable"))))
)))))))))

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
	(info_schema_key_column_usage_rows)

	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(info_schema_referential_constraint_rows)

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
	(merge '(scanfn '(session "__memcp_tx") '('info_schema_key_column_usage_rows)) rest)
	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(merge '(scanfn '(session "__memcp_tx") '('info_schema_referential_constraint_rows)) rest)
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
		lowered to (session key) before scan/scan_order/scan_batch
		see them. Do not stringify this source and do not add table-name
		fallbacks in Go for it. */
		(define scan-table-source (lambda (table_source) (match table_source
			'(scan-tagged-table base _ _ _ _ _) (scan-table-source base)
			'((symbol scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'((quote scan-tagged-table) base _ _ _ _ _) (scan-table-source base)
			'(materialized-subquery key) (list (quote session) key)
			'((symbol materialized-subquery) key) (list (quote session) key)
			'((quote materialized-subquery) key) (list (quote session) key)
			table_source)))
		(define tbl_resolved (scan-table-source tbl))
		/* materialized subqueries produce list expressions — pass as-is; real tables get (table schema name) */
		(define tbl_arg (if (string? tbl_resolved) (list 'table schema false tbl_resolved false) tbl_resolved))
		(merge (list scanfn '(session "__memcp_tx") tbl_arg) rest))
))))


/* Native catalog relations are exposed as ordinary logical views. Their
filters and join keys survive untangle_query and use the engine's autoindexes.
Only bounded scalar formatting remains in frontend helpers. */
(define sql_catalog_column_type (lambda (typ dimensions labels)
	(concat typ
		(if (and (equal?? typ "enum") (not (nil? labels)))
			(concat "(" (reduce (map labels quote_mysql_string) (lambda (a b) (concat a "," b))) ")")
			(if (or (nil? dimensions) (empty_list? dimensions)) ""
				(concat "(" (reduce (map dimensions string) (lambda (a b) (concat a "," b))) ")"))))))

(define sql_catalog_default (lambda (value expression nullable)
	(info_schema_column_default (list "Default" (if (nil? value) nil (json_decode_scmer value))
		"DefaultExpression" (coalesceNil expression "") "Null" nullable))))

(define sql_catalog_visible_table (lambda (name)
	(not (equal? (substr name 0 1) "."))))

(define sql_catalog_column_key (lambda (rank)
	(if (nil? rank) "" (equal? rank 1) "PRI" "UNI")))

(define sql_catalog_view (lambda (schema relation) (if (equal?? schema "information_schema") (begin
	(define catalog_col (lambda (alias name) (list (quote get_column) alias false name false)))
	(define catalog_source (lambda (alias name) (list alias "system_schema" name false nil)))
	(define catalog_equal_col (lambda (a b name) (list (quote equal?) (catalog_col a name) (catalog_col b name))))
	(define catalog_visible_expr (lambda (alias) (begin
		(define user_expr (list (quote coalesce) (list (quote session) "username") "root"))
		(list (quote or)
			(list (quote inner_select_exists) (make_query_block "system"
				(list (list "catalog_user" "system" "user" false nil)) (list "present" 1)
				(list (quote and) (list (quote equal?) (catalog_col "catalog_user" "username") user_expr)
					(catalog_col "catalog_user" "admin")) nil nil nil nil nil '() '() '()))
			(list (quote inner_select_exists) (make_query_block "system"
				(list (list "catalog_access" "system" "access" false nil)) (list "present" 1)
				(list (quote and) (list (quote equal?) (catalog_col "catalog_access" "username") user_expr)
					(list (quote equal?) (catalog_col "catalog_access" "database") (catalog_col alias "schema_name")))
				nil nil nil nil nil '() '() '()))))))
	(define catalog_view_query (lambda (sources fields where)
		(make_query_block "system_schema" sources fields where nil nil nil nil nil '() '() '())))
	(match (toLower relation)
		"schemata" (catalog_view_query (list (catalog_source "d" "databases"))
			(list "catalog_name" "def" "schema_name" (catalog_col "d" "schema_name")
				"default_character_set_name" "utf8mb4" "default_collation_name" "utf8mb3_general_ci"
				"sql_path" nil "schema_comment" "") (catalog_visible_expr "d"))
		"tables" (catalog_view_query (list (catalog_source "t" "tables"))
			(list "table_catalog" "def" "table_schema" (catalog_col "t" "schema_name")
				"table_name" (catalog_col "t" "name") "table_type" "BASE TABLE"
				"engine" (catalog_col "t" "persistency_mode")
				"table_rows" (list (quote sql_catalog_table_rows) (catalog_col "t" "schema_name") (catalog_col "t" "name"))
				"data_length" (list (quote sql_catalog_table_size) (catalog_col "t" "schema_name") (catalog_col "t" "name"))
				"table_collation" (catalog_col "t" "collation") "table_comment" (catalog_col "t" "comment"))
			(list (quote and) (catalog_visible_expr "t") (list (quote sql_catalog_visible_table) (catalog_col "t" "name"))))
		"columns" (begin
			(define key_rank (list (quote inner_select)
				(catalog_view_query (list (catalog_source "k" "keys") (catalog_source "kc" "key_columns"))
					(list "rank" (list (quote aggregate)
						(list (quote if) (list (quote equal?) (catalog_col "k" "id") "PRIMARY") 1 2) (quote min) nil))
					(list (quote and) (catalog_equal_col "k" "kc" "schema_name") (catalog_equal_col "k" "kc" "table_name")
						(list (quote equal?) (catalog_col "k" "ordinal") (catalog_col "kc" "key_ordinal"))
						(catalog_equal_col "c" "kc" "schema_name") (catalog_equal_col "c" "kc" "table_name")
						(list (quote equal?) (catalog_col "c" "name") (catalog_col "kc" "column_name"))))))
			(catalog_view_query (list (catalog_source "c" "columns"))
				(list "table_catalog" "def" "table_schema" (catalog_col "c" "schema_name")
					"table_name" (catalog_col "c" "table_name") "column_name" (catalog_col "c" "name")
					"ordinal_position" (list (quote +) (catalog_col "c" "ordinal") 1)
					"column_default" (list (quote sql_catalog_default) (catalog_col "c" "default") (catalog_col "c" "default_expression") (catalog_col "c" "allow_null"))
					"is_nullable" (list (quote if) (catalog_col "c" "allow_null") "YES" "NO")
					"data_type" (catalog_col "c" "typ")
					"column_type" (list (quote sql_catalog_column_type) (catalog_col "c" "typ") (catalog_col "c" "typdimensions") (catalog_col "c" "enum_values"))
					"column_key" (list (quote sql_catalog_column_key) key_rank)
					"extra" (list (quote if) (catalog_col "c" "auto_increment") "auto_increment" "")
					"privileges" "select,insert,update,references" "column_comment" (catalog_col "c" "comment")
					"is_generated" "NEVER" "generation_expression" "")
				(list (quote and) (catalog_visible_expr "c")
					(list (quote sql_catalog_visible_table) (catalog_col "c" "table_name"))
					(list (quote not) (catalog_col "c" "is_temp")))))

		"statistics" (catalog_view_query (list (catalog_source "k" "keys") (catalog_source "kc" "key_columns"))
			(list "table_catalog" "def" "table_schema" (catalog_col "k" "schema_name")
				"table_name" (catalog_col "k" "table_name") "non_unique" 0
				"index_schema" (catalog_col "k" "schema_name") "index_name" (catalog_col "k" "id")
				"seq_in_index" (list (quote +) (catalog_col "kc" "ordinal") 1)
				"column_name" (catalog_col "kc" "column_name") "collation" "A"
				"cardinality" nil "sub_part" nil "packed" nil "nullable" "" "index_type" "BTREE" "comment" "" "index_comment" "")
			(list (quote and) (catalog_visible_expr "k")
				(list (quote sql_catalog_visible_table) (catalog_col "k" "table_name"))
				(catalog_equal_col "k" "kc" "schema_name") (catalog_equal_col "k" "kc" "table_name")
				(list (quote equal?) (catalog_col "k" "ordinal") (catalog_col "kc" "key_ordinal"))))
		"key_column_usage" (make_union_block (quote all) (list
			(catalog_view_query (list (catalog_source "k" "keys") (catalog_source "kc" "key_columns"))
				(list "constraint_catalog" "def" "constraint_schema" (catalog_col "k" "schema_name")
					"constraint_name" (list (quote sql_catalog_constraint_name) (catalog_col "k" "constraint_name") (catalog_col "k" "id"))
					"table_catalog" "def" "table_schema" (catalog_col "k" "schema_name") "table_name" (catalog_col "k" "table_name")
					"column_name" (catalog_col "kc" "column_name") "ordinal_position" (list (quote +) (catalog_col "kc" "ordinal") 1)
					"position_in_unique_constraint" nil "referenced_table_schema" nil "referenced_table_name" nil "referenced_column_name" nil)
				(list (quote and) (catalog_visible_expr "k")
					(list (quote sql_catalog_visible_table) (catalog_col "k" "table_name"))
					(catalog_equal_col "k" "kc" "schema_name") (catalog_equal_col "k" "kc" "table_name")
					(list (quote equal?) (catalog_col "k" "ordinal") (catalog_col "kc" "key_ordinal"))))
			(catalog_view_query (list (catalog_source "f" "foreign_keys") (catalog_source "fc" "foreign_key_columns"))
				(list "constraint_catalog" "def" "constraint_schema" (catalog_col "f" "schema_name")
					"constraint_name" (catalog_col "f" "id") "table_catalog" "def"
					"table_schema" (catalog_col "f" "schema_name") "table_name" (catalog_col "f" "tbl1")
					"column_name" (catalog_col "fc" "column1") "ordinal_position" (list (quote +) (catalog_col "fc" "ordinal") 1)
					"position_in_unique_constraint" (list (quote +) (catalog_col "fc" "ordinal") 1)
					"referenced_table_schema" (catalog_col "f" "schema_name") "referenced_table_name" (catalog_col "f" "tbl2")
					"referenced_column_name" (catalog_col "fc" "column2"))
				(list (quote and) (catalog_visible_expr "f")
					(list (quote sql_catalog_visible_table) (catalog_col "f" "tbl1"))
					(list (quote equal?) (catalog_col "f" "table_name") (catalog_col "f" "tbl1"))
					(catalog_equal_col "f" "fc" "schema_name") (catalog_equal_col "f" "fc" "table_name")
					(list (quote equal?) (catalog_col "f" "ordinal") (catalog_col "fc" "foreign_key_ordinal")))))
			nil nil nil '())
		"referential_constraints" (begin
			(define referenced_columns (list (quote inner_select)
				(catalog_view_query (list (catalog_source "fc" "foreign_key_columns"))
					(list "signature" (sql_group_concat_ordered_expr (list (quote sql_catalog_key_part) (catalog_col "fc" "column2"))
						(list (list (catalog_col "fc" "ordinal") <)) "" false))
					(list (quote and) (catalog_equal_col "f" "fc" "schema_name") (catalog_equal_col "f" "fc" "table_name")
						(list (quote equal?) (catalog_col "f" "ordinal") (catalog_col "fc" "foreign_key_ordinal"))))))
			(define key_columns (list (quote inner_select)
				(catalog_view_query (list (catalog_source "kc" "key_columns"))
					(list "signature" (sql_group_concat_ordered_expr (list (quote sql_catalog_key_part) (catalog_col "kc" "column_name"))
						(list (list (catalog_col "kc" "ordinal") <)) "" false))
					(list (quote and) (catalog_equal_col "k" "kc" "schema_name") (catalog_equal_col "k" "kc" "table_name")
						(list (quote equal?) (catalog_col "k" "ordinal") (catalog_col "kc" "key_ordinal"))))))
			(define matching_key (list (quote inner_select)
				(make_query_block "system_schema" (list (catalog_source "k" "keys"))
					(list "constraint_name" (list (quote sql_catalog_constraint_name) (catalog_col "k" "constraint_name") (catalog_col "k" "id")))
					(list (quote and) (catalog_equal_col "f" "k" "schema_name")
						(list (quote equal?) (catalog_col "f" "tbl2") (catalog_col "k" "table_name"))
						(list (quote equal?) key_columns referenced_columns))
					nil nil (list (list (catalog_col "k" "ordinal") <)) 1 nil '() '() '())))
			(catalog_view_query (list (catalog_source "f" "foreign_keys"))
				(list "constraint_catalog" "def" "constraint_schema" (catalog_col "f" "schema_name")
					"constraint_name" (catalog_col "f" "id") "unique_constraint_catalog" "def"
					"unique_constraint_schema" (catalog_col "f" "schema_name") "unique_constraint_name" matching_key
					"match_option" "NONE" "update_rule" (list (quote sql_catalog_fk_action) (catalog_col "f" "updatemode"))
					"delete_rule" (list (quote sql_catalog_fk_action) (catalog_col "f" "deletemode"))
					"table_name" (catalog_col "f" "tbl1") "referenced_table_name" (catalog_col "f" "tbl2"))
				(list (quote and) (catalog_visible_expr "f")
					(list (quote sql_catalog_visible_table) (catalog_col "f" "tbl1"))
					(list (quote equal?) (catalog_col "f" "table_name") (catalog_col "f" "tbl1")))))
		"triggers" (catalog_view_query (list (catalog_source "tr" "triggers"))
			(list "trigger_catalog" "def" "trigger_schema" (catalog_col "tr" "schema_name") "trigger_name" (catalog_col "tr" "name")
				"event_manipulation" (list (quote sql_catalog_trigger_part) (catalog_col "tr" "timing") 1)
				"event_object_catalog" "def" "event_object_schema" (catalog_col "tr" "schema_name") "event_object_table" (catalog_col "tr" "table_name")
				"action_order" 1 "action_condition" nil "action_statement" (catalog_col "tr" "source") "action_orientation" "ROW"
				"action_timing" (list (quote sql_catalog_trigger_part) (catalog_col "tr" "timing") 0)
				"created" nil "sql_mode" "" "definer" "root@localhost" "character_set_client" "utf8mb4"
				"collation_connection" "utf8mb4_general_ci" "database_collation" "utf8mb4_general_ci")
			(list (quote and) (catalog_visible_expr "tr")
				(list (quote sql_catalog_visible_table) (catalog_col "tr" "table_name"))
				(list (quote not) (catalog_col "tr" "hidden")) (list (quote not) (catalog_col "tr" "is_system"))))
		_ nil)) nil)))

(define sql_catalog_query? (lambda (node)
	(match node
		((symbol query-block) _schema sources _fields _where _group _having _order _limit _offset _hidden _stages _facts)
		(or (reduce sources (lambda (found source)
			(or found (and (equal?? (cadr source) "information_schema") (string? (nth source 2))))) false)
			(reduce (cdr node) (lambda (found child) (or found (sql_catalog_query? child))) false))
		(cons head tail) (or (sql_catalog_query? head)
			(reduce tail (lambda (found child) (or found (sql_catalog_query? child))) false))
		_ false)))


(define sql_catalog_constraint_name (lambda (name id)
	(if (or (nil? name) (equal? name "")) id name)))
(define sql_catalog_fk_action (lambda (mode)
	(if (equal? mode 1) "CASCADE" (equal? mode 2) "SET NULL" "RESTRICT")))
(define sql_catalog_trigger_part (lambda (timing part)
	(toUpper (nth (split timing "_") part))))
(define sql_catalog_table_rows (lambda (schema name)
	(get_assoc (table_planner_statistics (table schema false name false)) "row_count")))
(define sql_catalog_table_size (lambda (schema name)
	(get_assoc (show (table schema false name false) "memory") "size_bytes")))

(define sql_catalog_key_part (lambda (name) (concat (strlen name) ":" name)))
