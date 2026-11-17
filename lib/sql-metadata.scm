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

/* INFORMATION_SCHEMA is a set of built-in views over the relational catalog
system_schema (see storage/catalog.go). They expand like user views before
untangle_query, so filters, joins and ordering on catalog queries are planned
like on any other table. The views run with definer rights: users who may read
INFORMATION_SCHEMA need no access to system_schema itself. */

/* Runtime values that are not declarations: statistics come from the RAM table
behind the transient system_schema.tables.handle column. */
(sql_builtins "MEMCP_TABLE_STATISTIC" (lambda (handle key)
	(if (nil? handle) nil ((table_statistics handle) key))))
(sql_builtins "MEMCP_COLUMN_TYPE" (lambda (typ dimensions enum_values)
	(column_type_display typ dimensions enum_values)))
/* COLUMN_DEFAULT of a catalog row; default_json is the stored Scheme value. */
(sql_builtins "MEMCP_COLUMN_DEFAULT" (lambda (default_json default_expression allow_null) (begin
	(define value (if (nil? default_json) nil (json_decode_scmer default_json)))
	(if (and (not (nil? default_expression)) (not (equal? default_expression "")))
		(if (equal? (toUpper default_expression) "CURRENT_TIMESTAMP")
			"current_timestamp()" default_expression)
		(if (nil? value)
			(if allow_null "NULL" NULL)
			(if (string? value)
				(quote_mysql_string value)
				(concat value)))))))

(define information_schema_view_sql (lambda (name) (match (toLower name)
	"schemata" "SELECT 'def' AS catalog_name, d.schema_name AS schema_name,
		'utf8mb4' AS default_character_set_name, 'utf8mb3_general_ci' AS default_collation_name,
		NULL AS sql_path, '' AS schema_comment
		FROM system_schema.databases d"
	"tables" "SELECT 'def' AS table_catalog, t.schema_name AS table_schema, t.name AS table_name,
		'BASE TABLE' AS table_type,
		CASE WHEN t.persistency_mode IN ('logged', 'sloppy', 'memory') THEN t.persistency_mode ELSE 'safe' END AS engine,
		MEMCP_TABLE_STATISTIC(t.handle, 'row_count') AS table_rows,
		MEMCP_TABLE_STATISTIC(t.handle, 'size_bytes') AS data_length,
		t.`collation` AS table_collation, t.`comment` AS table_comment
		FROM system_schema.tables t WHERE t.name NOT LIKE '.%'"
	"columns" "SELECT 'def' AS table_catalog, c.schema_name AS table_schema, c.table_name AS table_name,
		c.name AS column_name, c.ordinal + 1 AS ordinal_position,
		MEMCP_COLUMN_DEFAULT(c.`default`, c.default_expression, c.allow_null) AS column_default,
		CASE WHEN c.allow_null THEN 'YES' ELSE 'NO' END AS is_nullable,
		c.typ AS data_type,
		MEMCP_COLUMN_TYPE(c.typ, c.typdimensions, c.enum_values) AS column_type,
		CASE WHEN k.key_rank = 2 THEN 'PRI' WHEN k.key_rank = 1 THEN 'UNI' ELSE '' END AS column_key,
		CASE WHEN c.auto_increment THEN 'auto_increment' ELSE '' END AS extra,
		'select,insert,update,references' AS privileges,
		c.`comment` AS column_comment, 'NEVER' AS is_generated, '' AS generation_expression
		FROM system_schema.columns c
		LEFT JOIN (SELECT kc.schema_name AS schema_name, kc.table_name AS table_name, kc.column_name AS column_name,
				MAX(CASE WHEN ky.id = 'PRIMARY' THEN 2 ELSE 1 END) AS key_rank
			FROM system_schema.key_columns kc
			JOIN system_schema.`keys` ky ON ky.schema_name = kc.schema_name AND ky.table_name = kc.table_name AND ky.ordinal = kc.key_ordinal
			GROUP BY kc.schema_name, kc.table_name, kc.column_name) k
			ON k.schema_name = c.schema_name AND k.table_name = c.table_name AND k.column_name = c.name
		WHERE c.table_name NOT LIKE '.%'"
	"key_column_usage" "SELECT 'def' AS constraint_catalog, ky.schema_name AS constraint_schema,
			CASE WHEN ky.constraint_name IS NULL OR ky.constraint_name = '' THEN ky.id ELSE ky.constraint_name END AS constraint_name,
			'def' AS table_catalog, ky.schema_name AS table_schema, ky.table_name AS table_name,
			kc.column_name AS column_name, kc.ordinal + 1 AS ordinal_position,
			NULL AS position_in_unique_constraint, NULL AS referenced_table_schema,
			NULL AS referenced_table_name, NULL AS referenced_column_name
		FROM system_schema.`keys` ky
		JOIN system_schema.key_columns kc ON kc.schema_name = ky.schema_name AND kc.table_name = ky.table_name AND kc.key_ordinal = ky.ordinal
		WHERE ky.table_name NOT LIKE '.%'
		UNION ALL
		SELECT 'def' AS constraint_catalog, f.schema_name AS constraint_schema, f.id AS constraint_name,
			'def' AS table_catalog, f.schema_name AS table_schema, f.table_name AS table_name,
			fc.column1 AS column_name, fc.ordinal + 1 AS ordinal_position, fc.ordinal + 1 AS position_in_unique_constraint,
			f.schema_name AS referenced_table_schema, f.tbl2 AS referenced_table_name, fc.column2 AS referenced_column_name
		FROM system_schema.foreign_keys f
		JOIN system_schema.foreign_key_columns fc ON fc.schema_name = f.schema_name AND fc.table_name = f.table_name AND fc.foreign_key_ordinal = f.ordinal
		WHERE f.table_name = f.tbl1 AND f.table_name NOT LIKE '.%'"
	/* unique_constraint_name names the first key of the referenced table whose
	column list equals the foreign key's referenced columns. */
	"referential_constraints" "SELECT 'def' AS constraint_catalog, f.schema_name AS constraint_schema, f.id AS constraint_name,
			'def' AS unique_constraint_catalog, f.schema_name AS unique_constraint_schema,
			CASE WHEN uk.constraint_name IS NULL OR uk.constraint_name = '' THEN uk.id ELSE uk.constraint_name END AS unique_constraint_name,
			'NONE' AS match_option,
			CASE f.updatemode WHEN 1 THEN 'CASCADE' WHEN 2 THEN 'SET NULL' ELSE 'RESTRICT' END AS update_rule,
			CASE f.deletemode WHEN 1 THEN 'CASCADE' WHEN 2 THEN 'SET NULL' ELSE 'RESTRICT' END AS delete_rule,
			f.table_name AS table_name, f.tbl2 AS referenced_table_name
		FROM system_schema.foreign_keys f
		LEFT JOIN (SELECT m.schema_name AS schema_name, m.table_name AS table_name, m.foreign_key_ordinal AS foreign_key_ordinal,
				m.key_table AS key_table, MIN(m.key_ordinal) AS key_ordinal
			FROM (SELECT fc.schema_name AS schema_name, fc.table_name AS table_name, fc.foreign_key_ordinal AS foreign_key_ordinal,
					f2.tbl2 AS key_table, kc.key_ordinal AS key_ordinal, COUNT(*) AS matched
				FROM system_schema.foreign_key_columns fc
				JOIN system_schema.foreign_keys f2 ON f2.schema_name = fc.schema_name AND f2.table_name = fc.table_name AND f2.ordinal = fc.foreign_key_ordinal
				JOIN system_schema.key_columns kc ON kc.schema_name = fc.schema_name AND kc.table_name = f2.tbl2 AND kc.ordinal = fc.ordinal AND kc.column_name = fc.column2
				GROUP BY fc.schema_name, fc.table_name, fc.foreign_key_ordinal, f2.tbl2, kc.key_ordinal) m
			JOIN (SELECT fc.schema_name AS schema_name, fc.table_name AS table_name, fc.foreign_key_ordinal AS foreign_key_ordinal, COUNT(*) AS width
				FROM system_schema.foreign_key_columns fc GROUP BY fc.schema_name, fc.table_name, fc.foreign_key_ordinal) fw
				ON fw.schema_name = m.schema_name AND fw.table_name = m.table_name AND fw.foreign_key_ordinal = m.foreign_key_ordinal AND fw.width = m.matched
			JOIN (SELECT kc.schema_name AS schema_name, kc.table_name AS table_name, kc.key_ordinal AS key_ordinal, COUNT(*) AS width
				FROM system_schema.key_columns kc GROUP BY kc.schema_name, kc.table_name, kc.key_ordinal) kw
				ON kw.schema_name = m.schema_name AND kw.table_name = m.key_table AND kw.key_ordinal = m.key_ordinal AND kw.width = m.matched
			GROUP BY m.schema_name, m.table_name, m.foreign_key_ordinal, m.key_table) km
			ON km.schema_name = f.schema_name AND km.table_name = f.table_name AND km.foreign_key_ordinal = f.ordinal
		LEFT JOIN system_schema.`keys` uk ON uk.schema_name = km.schema_name AND uk.table_name = km.key_table AND uk.ordinal = km.key_ordinal
		WHERE f.table_name = f.tbl1 AND f.table_name NOT LIKE '.%'"
	"statistics" "SELECT 'def' AS table_catalog, ky.schema_name AS table_schema, ky.table_name AS table_name,
			0 AS non_unique, ky.schema_name AS index_schema, ky.id AS index_name,
			kc.ordinal + 1 AS seq_in_index, kc.column_name AS column_name, 'A' AS `collation`,
			NULL AS cardinality, NULL AS sub_part, NULL AS packed, '' AS nullable,
			'BTREE' AS index_type, '' AS `comment`, '' AS index_comment
		FROM system_schema.`keys` ky
		JOIN system_schema.key_columns kc ON kc.schema_name = ky.schema_name AND kc.table_name = ky.table_name AND kc.key_ordinal = ky.ordinal
		WHERE ky.table_name NOT LIKE '.%'"
	"triggers" "SELECT 'def' AS trigger_catalog, tr.schema_name AS trigger_schema, tr.name AS trigger_name,
			CASE WHEN tr.timing IN ('before_update', 'after_update') THEN 'UPDATE'
				WHEN tr.timing IN ('before_delete', 'after_delete') THEN 'DELETE' ELSE 'INSERT' END AS event_manipulation,
			'def' AS event_object_catalog, tr.schema_name AS event_object_schema, tr.table_name AS event_object_table,
			1 AS action_order, NULL AS action_condition,
			CASE WHEN tr.language = 'sql' THEN tr.source ELSE '' END AS action_statement,
			'ROW' AS action_orientation,
			CASE WHEN tr.timing IN ('after_insert', 'after_update', 'after_delete') THEN 'AFTER' ELSE 'BEFORE' END AS action_timing,
			NULL AS created, '' AS sql_mode, '' AS definer, 'utf8mb4' AS character_set_client,
			'utf8mb4_general_ci' AS collation_connection, 'utf8mb4_general_ci' AS database_collation
		FROM system_schema.triggers tr
		WHERE NOT COALESCE(tr.is_system, FALSE) AND NOT COALESCE(tr.hidden, FALSE)"
	/* MemCP has no tablespaces, undo logs or MySQL partitions */
	"files" "SELECT NULL AS file_name, NULL AS file_type, NULL AS tablespace_name, NULL AS logfile_group_name,
			NULL AS total_extents, NULL AS extent_size, NULL AS initial_size, NULL AS engine, NULL AS extra
		FROM system_schema.databases d WHERE FALSE"
	"partitions" "SELECT NULL AS table_schema, NULL AS table_name, NULL AS partition_name, NULL AS tablespace_name
		FROM system_schema.databases d WHERE FALSE"
	nil)))

/* Parsed once per process: the IR of a built-in view never changes. */
(define information_schema_view_ir (newsession))
(define information_schema_view (lambda (name) (begin
	(define key (toLower name))
	(define cached (information_schema_view_ir key))
	(if (nil? cached)
		(begin
			(define sql (information_schema_view_sql key))
			(if (nil? sql) (error (concat "INFORMATION_SCHEMA." name " is not supported yet")) true)
			(define parsed (parse_sql "information_schema"
				(concat "CREATE VIEW information_schema." key " AS " sql) nil (newsession) nil))
			(define ir (cadr (nth parsed 6)))
			(information_schema_view_ir key ir)
			ir)
		cached))))

/* The view's query blocks belong to the database of the query that reads
them: the planner places helper tables (keytables, group caches) there.
Catalog tables are fully qualified, so only that placement changes. A query
whose own database is INFORMATION_SCHEMA uses "system". */
(define information_schema_rehome (lambda (node schema)
	(match node
		(cons head _) (begin
			(define mapped (map node (lambda (item) (information_schema_rehome item schema))))
			(if (and (equal? head (quote query-block)) (equal?? (cadr mapped) "information_schema"))
				(cons head (cons schema (cdr (cdr mapped))))
				mapped))
		_ node)))

/* Column catalog of an INFORMATION_SCHEMA relation, derived from its view. */
(define information_schema_column_catalog (lambda (schema tbl) (begin
	(define block_fields (lambda (block) (match block
		((symbol query-block) _ _ fields _ _ _ _ _ _ _ _ _) fields
		((symbol union-block) _ branches _ _ _ _) (block_fields (car branches))
		_ (error "INFORMATION_SCHEMA view has no query block"))))
	(define fields (block_fields (information_schema_view tbl)))
	(map (produceN (/ (count fields) 2)) (lambda (i) (begin
		(define name (nth fields (* i 2)))
		(if (has? '("table_rows" "data_length" "action_order") name)
			(list "Field" name "Type" "bigint")
			(list "Field" name))))))))
