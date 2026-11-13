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
				'("column" col typ dimensions attributes inputs computor)
				(list "column" col typ dimensions
					(sql_column_collation_defaults typ attributes (get_assoc options "collation")) inputs computor)
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

(define info_schema_columns_for_table (lambda (schema table_name tx) (begin
	(define handle (table schema false table_name false tx))
	(define columns (if (nil? handle) '() (show handle)))
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

(define info_schema_fk_unique_name (lambda (schema fk tx) (begin
	(define referenced_meta (show (table schema false (fk "OtherTable") false tx) "constraints" tx))
	(define referenced_key (find (referenced_meta "Unique") (lambda (key)
		(equal? (key "Cols") (fk "OtherColumns")))))
	(if (nil? referenced_key) NULL (sql_key_constraint_name referenced_key))
)))

(define info_schema_key_column_usage_rows (lambda (tx)
	(merge (map (show) (lambda (schema)
		(merge (map (show schema false tx) (lambda (tbl) (begin
			(define meta (show (table schema false tbl false tx) "constraints" tx))
			(define unique_rows (merge (map (meta "Unique") (lambda (key)
				(info_schema_unique_column_rows schema tbl key (key "Cols") 1)))))
			(define foreign_rows (merge (map (filter (meta "ForeignKeys") (lambda (fk)
				(equal? (fk "Role") "referencing"))) (lambda (fk)
					(info_schema_fk_column_rows schema tbl fk
						(fk "LocalColumns") (fk "OtherColumns") 1)))))
			(merge unique_rows foreign_rows)
)))))))))

(define info_schema_referential_constraints_for_table (lambda (schema tbl tx) (begin
	(define handle (table schema false tbl false tx))
	(define meta (if (nil? handle) nil (show handle "constraints" tx)))
	(define foreign (if (nil? meta) '() (meta "ForeignKeys")))
	(map (filter foreign (lambda (fk)
		(equal? (fk "Role") "referencing"))) (lambda (fk)
			(list "constraint_catalog" "def" "constraint_schema" schema
				"constraint_name" (fk "Id")
				"unique_constraint_catalog" "def"
				"unique_constraint_schema" schema
				"unique_constraint_name" (info_schema_fk_unique_name schema fk tx)
				"match_option" "NONE"
				"update_rule" (toUpper (fk "UpdateMode"))
				"delete_rule" (toUpper (fk "DeleteMode"))
				"table_name" tbl
				"referenced_table_name" (fk "OtherTable"))))
)))

/* Constraint catalogs describe actual native declaration publications. No
query rows, synthetic constraints, or automatic access indexes enter them. */
(define info_schema_constraint_rows (lambda (database name tx checks_only) (begin
	(define handle (table database false name false tx))
	(if (nil? handle) '() (begin
		(define constraints (show handle "constraints" tx))
		(define checks (coalesceNil (constraints "Checks") '()))
		(if checks_only (map checks (lambda (check)
			(list "constraint_catalog" "def" "constraint_schema" database
				"constraint_name" (check "Id") "check_clause" (check "Source"))))
			(begin
				(define row (lambda (id type enabled)
					(list "constraint_catalog" "def" "constraint_schema" database "constraint_name" id
						"table_catalog" "def" "table_schema" database "table_name" name
						"constraint_type" type "enforced" (if enabled "YES" "NO"))))
				(merge (list
					(map (constraints "Unique") (lambda (key)
						(row (sql_key_constraint_name key) (if (equal? (key "Id") "PRIMARY") "PRIMARY KEY" "UNIQUE") true)))
					(map (filter (constraints "ForeignKeys") (lambda (key) (equal? (key "Role") "referencing")))
						(lambda (key) (row (key "Id") "FOREIGN KEY" (if (has_assoc? key "Enabled") (key "Enabled") true))))
					(map checks (lambda (check) (row (check "Id") "CHECK" (if (has_assoc? check "Enabled") (check "Enabled") true)))))))))))))

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
	'((ignorecase "information_schema") (ignorecase "check_constraints")) '(
		'("Field" "constraint_catalog") '("Field" "constraint_schema")
		'("Field" "constraint_name") '("Field" "check_clause"))
	'((ignorecase "information_schema") (ignorecase "table_constraints")) '(
		'("Field" "constraint_catalog") '("Field" "constraint_schema") '("Field" "constraint_name")
		'("Field" "table_catalog") '("Field" "table_schema") '("Field" "table_name")
		'("Field" "constraint_type") '("Field" "enforced"))
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
/* Physical name selectors restrict metadata construction, never replace the
scan predicate. A predicate receives only one native name, not an executable
query, session, or relation. Exact selectors use native case-sensitive lookup;
SQL value comparisons instead retain every matching catalog name. */
(define info_schema_name_matches (lambda (selector name)
	(match selector
		nil true
		'("exact" value) (equal? name value)
		'("predicate" predicate) (predicate name)
		_ (error "invalid metadata name selector"))))

(define info_schema_selected_databases (lambda (selection)
	(if (literal_false? selection) '()
		(filter (show) (lambda (db)
			(info_schema_name_matches (get_assoc (coalesceNil selection '()) "schema" nil) db))))))

(define info_schema_selected_tables (lambda (db tx selection)
	(match (get_assoc (coalesceNil selection '()) "table" nil)
		'("exact" name) (if (nil? (table db false name false tx)) '() (list name))
		selector (filter (show db false tx) (lambda (name)
			(info_schema_name_matches selector name))))))

/* Keep one table-level producer for columns and statistics, so name selection
happens before invoking any per-table row builder. Ordinary enumeration retains
its existing explicit transaction visibility. */
(define info_schema_table_rows (lambda (tx selection rows_for_table)
	(merge (map (info_schema_selected_databases selection) (lambda (db)
		(merge (map (info_schema_selected_tables db tx selection) (lambda (name)
			(rows_for_table db name tx)))))))))

(define information_schema_rows (lambda (schema tbl tx selection) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "schemata"))
	(map (info_schema_selected_databases selection) (lambda (db)
		(list
			"catalog_name" "def"
			"schema_name" db
			"default_character_set_name" "utf8mb4"
			"default_collation_name" "utf8mb3_general_ci"
			"sql_path" NULL
			"schema_comment" "")))

	'((ignorecase "information_schema") (ignorecase "tables"))
	/* Native table summaries have a different descriptive contract from SHOW
	a handle's full diagnostics. Preserve the existing summary owner and narrow
	its names before formatting rows; columns/statistics narrow before building. */
	(merge (map (info_schema_selected_databases selection) (lambda (db)
		(map (filter (show db true tx) (lambda (table_info)
			(info_schema_name_matches (get_assoc (coalesceNil selection '()) "table" nil)
				(table_info "name"))))
			(lambda (table_info) (info_schema_table_row db table_info))))))

	'((ignorecase "information_schema") (ignorecase "columns"))
	(info_schema_table_rows tx selection info_schema_columns_for_table)

	'((ignorecase "information_schema") (ignorecase "check_constraints"))
	(info_schema_table_rows tx selection (lambda (database name tx) (info_schema_constraint_rows database name tx true)))
	'((ignorecase "information_schema") (ignorecase "table_constraints"))
	(info_schema_table_rows tx selection (lambda (database name tx) (info_schema_constraint_rows database name tx false)))
	'((ignorecase "information_schema") (ignorecase "key_column_usage"))
	(info_schema_key_column_usage_rows tx)

	'((ignorecase "information_schema") (ignorecase "referential_constraints"))
	(info_schema_table_rows tx selection info_schema_referential_constraints_for_table)

	'((ignorecase "information_schema") (ignorecase "statistics"))
	(info_schema_table_rows tx selection (lambda (db name context)
		(show (table db false name false context) (sql_metadata_identity "statistics"))))

	'((ignorecase "information_schema") (ignorecase "triggers"))
	(info_schema_trigger_rows)

	'((ignorecase "information_schema") (ignorecase "files"))
	(list)

	'((ignorecase "information_schema") (ignorecase "partitions"))
	(list)

	'((ignorecase "information_schema") _)
	(error (concat "INFORMATION_SCHEMA." tbl " is not supported yet"))

	(show (table schema false tbl false tx))
)))

(define scan_wrapper (lambda args (match args (merge '(scanfn schema tbl) rest) (match '(schema tbl)
	'((ignorecase "information_schema") _)
	(merge (list scanfn (list (quote session) "__memcp_tx")
		(list (quote information_schema_rows) schema tbl (list (quote session) "__memcp_tx") nil)) rest)
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
		(define tbl_arg (if (string? tbl_resolved) (list 'table schema false tbl_resolved false (quote tx)) tbl_resolved))
		(merge (list scanfn '(session "__memcp_tx") tbl_arg) rest))
))))
