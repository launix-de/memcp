/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-parser.scm")
(define tsql_queryplan_cache (newcachemap))
(define tsql_statement_text strtrim)

/* Alias declarations are ordinary, durable frontend data, like SQL views.
The native column keeps the qualified alias spelling in its existing RawType.
Only this frontend interprets the declaration; storage has no alias registry. */
(if (has? (show "system") "tsql_types") true
	(eval (parse_sql "system" "CREATE TABLE tsql_types (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, `database` VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, base_type VARCHAR(255) NOT NULL, dimensions TEXT NOT NULL, nullable BOOLEAN NOT NULL, UNIQUE KEY alias_name (`database`,name)) ENGINE=SAFE"
		(lambda (_schema _table _write) true))))

/* These gates coordinate alias schema DDL only. Reads and DML never acquire
them. CREATE/DROP TYPE and publishing columns which use aliases share one gate
per database, preventing a DROP between dependency checking and publication.
The stable scope retains gates, not requests, transactions or executable plans. */
(define tsql_type_ddl_gates (newsession))
(define tsql_type_ddl_scope (newsession))
(define tsql_type_ddl (lambda (database tx operation)
	((tsql_type_ddl_gates "get_or_compute_scoped" tsql_type_ddl_scope database (lambda () (mutex))) tx operation)))
(define tsql_qualified_type (lambda (database name)
	(concat "[" (replace database "]" "]]") "].[dbo].[" (replace name "]" "]]") "]")))
(define tsql_alias_name_parts (parser '(
	(define database tsql_identifier) "." (define owner tsql_identifier) "." (define name tsql_identifier)
) (if (equal?? owner "dbo") (list database name) (error "unsupported alias owner: " owner))))
(define tsql_find_type_alias (lambda (database name)
	(scan nil (table "system" "tsql_types")
		(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false))
		(list database) '("name") (lambda (candidate) (equal?? candidate name))
		'("id" "name" "base_type" "dimensions" "nullable")
		(lambda (_ id canonical base dimensions nullable)
			(list "id" id "name" canonical "base_type" base "dimensions" (json_decode dimensions) "nullable" nullable))
		nil (lambda (a b) (if (nil? a) b (nil? b) a (error "ambiguous alias type: " name))))))
(define tsql_alias_declaration (lambda (type)
	(if (equal? (substr type 0 1) "[") (begin
		(define parts (tsql_alias_name_parts type))
		(coalesceNil (tsql_find_type_alias (car parts) (cadr parts))
			(error "alias type no longer exists: " type))) nil)))
(define tsql_effective_column (lambda (column)
	(if (nil? column) nil (begin
		(define alias (tsql_alias_declaration (column "RawType")))
		(if (nil? alias) column
			(set_assoc (set_assoc (set_assoc (set_assoc column "DeclaredType" (column "RawType"))
				"RawType" (alias "base_type")) "Type" (alias "base_type")) "AliasId" (alias "id")))))))
(define tsql_compile_column (lambda (info declaration)
	(if (nil? declaration) info (begin
		(define alias (tsql_alias_declaration (declaration "RawType")))
		(if (nil? alias) info
			(sql_info (sql_info_formula info) (alias "base_type")
				(if (sql_text_type? (alias "base_type"))
					(list (coalesce (declaration "Collation") "utf8mb4_general_ci") 2) nil)))))))
(define create_sql_type_alias (lambda (database name base dimensions nullable)
	(tsql_type_ddl database nil (lambda () (begin
		(define native_type (toUpper base))
		(define native_dimensions (tsql_decl_dimensions native_type dimensions))
		(if (has? '("TIMESTAMP" "ROWVERSION") native_type) (error "generated versions cannot be alias base types") true)
		(if (tsql_find_type_alias database name) (error "alias type already exists: " name) true)
		(insert (table "system" "tsql_types") '("database" "name" "base_type" "dimensions" "nullable")
			(list (list database name native_type (json_encode native_dimensions) nullable)))
		true)))))
(define drop_sql_type_alias (lambda (database name ifexists)
	(tsql_type_ddl database nil (lambda () (begin
		(define alias (tsql_find_type_alias database name))
		(if (nil? alias) (if ifexists false (error "alias type does not exist: " name)) (begin
			(define qualified (tsql_qualified_type database (alias "name")))
			/* Enumerating declarations is DROP TYPE work, never query costing or
			row execution. The database gate excludes concurrent alias consumers. */
			(define dependent (find (show database) (lambda (relation)
				(find (show database relation) (lambda (column) (equal?? (column "RawType") qualified)))) nil))
			(if dependent (error "alias type is used by table: " dependent) true)
			(scan nil (table "system" "tsql_types")
				(list 369436175368192 (scan_boundary "equal" "id" 0 0 true true "" false))
				(list (alias "id")) '() (lambda () true) '("$update")
				(lambda (count remove) (begin (remove) (+ count 1))) 0 +)
			true)))))))

(define tsql_native_types '("VARCHAR" "CHAR" "NVARCHAR" "NCHAR" "BINARY" "VARBINARY"
	"DECIMAL" "NUMERIC" "FLOAT" "TIME" "DATETIME2" "DATETIMEOFFSET" "INT" "INTEGER" "BIGINT"
	"SMALLINT" "TINYINT" "BIT" "REAL" "MONEY" "SMALLMONEY" "DATE" "DATETIME" "SMALLDATETIME"
	"ROWVERSION" "TIMESTAMP" "UNIQUEIDENTIFIER" "TEXT" "NTEXT" "IMAGE"))
(define tsql_schema_type_parts (parser (define parts (+ tsql_identifier ".")) parts))
(define tsql_resolve_schema_type (lambda (database type dimensions) (begin
	(define parts (tsql_schema_type_parts type))
	(define native (match parts
		'(name) (if (has? tsql_native_types (toUpper name)) (toUpper name) nil)
		'(owner name) (if (equal?? owner "sys") (toUpper name) nil)
		_ nil))
	(if native (list "type" native "dimensions" (tsql_decl_dimensions native dimensions) "nullable" true)
		(begin
			(define name (match parts
				'(name) name
				'(owner name) (if (equal?? owner "dbo") name (error "unsupported alias owner: " owner))
				'(catalog owner name) (if (and (equal? catalog database) (equal?? owner "dbo")) name
					(error "alias must belong to the target database"))
				_ (error "invalid alias type name")))
			(if (empty_list? dimensions) true (error "alias dimensions cannot be overridden"))
			(define alias (coalesceNil (tsql_find_type_alias database name) (error "unknown type: " name)))
			(merge (list "type" (tsql_qualified_type database (alias "name"))) alias))))))
(define tsql_prepare_column (lambda (database definition) (match definition
	'((symbol list) "column" name type dimensions attributes) (begin
		(define contract (tsql_resolve_schema_type database type (cdr dimensions)))
		(define base (coalesceNil (contract "base_type") (contract "type")))
		(define native_dimensions (contract "dimensions"))
		(define options (cdr attributes))
		(define null_options (if (or (nil? (contract "id")) (has_assoc? options "null")) options
			(append options "null" (contract "nullable"))))
		(if (and (contract "id") (null_options "auto_increment"))
			(error "IDENTITY alias declarations are not yet supported") true)
		(define identity_options (tsql_identity_attributes base native_dimensions null_options))
		/* Frozen defaults use the same frontend conversion as explicit writes.
		The existing native default field remains their only persisted value. */
		(define default_options (if (and (has_assoc? identity_options "default")
			(or (contract "id") (has? '("NVARCHAR" "NCHAR") base)))
			(set_assoc identity_options "default" (eval (sql_type_formula '()
				(tsql_cast_expr (list (quote quote) (identity_options "default")) base native_dimensions)))) identity_options))
		(define published_options (if (contract "id") (append default_options "__tsql_alias_id" (contract "id")) default_options))
		(list (quote list) "column" name (tsql_storage_type (contract "type"))
			(cons (quote list) native_dimensions) (cons (quote list) published_options)))
	_ definition)))
(define tsql_publish_alias_attributes (lambda (type attributes) (begin
	(if (has_assoc? attributes "__tsql_alias_id")
		(if (equal? ((tsql_alias_declaration type) "id") (attributes "__tsql_alias_id")) true
			(error "alias declaration changed before DDL execution")) true)
	(merge (extract_assoc attributes (lambda (key value)
		(if (equal? key "__tsql_alias_id") '() (list key value))))))))
(define tsql_add_columns (lambda (database name columns tx) (begin
	(define publish (lambda () (begin
		/* Validate every alias identity before changing any column. */
		(define published_columns (map columns (lambda (column) (match column
			'("column" column_name type dimensions attributes)
			(list column_name type dimensions (tsql_publish_alias_attributes type attributes))))))
		(reduce published_columns (lambda (_ column) (match column '(column_name type dimensions attributes)
			(if (sql_create_column (table database name) column_name type dimensions
				(merge (sql_add_column_attributes attributes) '("name_ignorecase" true))) true
				(error "column already exists: " column_name)))) true))))
	(if (find columns (lambda (column) (has_assoc? (nth column 4) "__tsql_alias_id")))
		(tsql_type_ddl database tx publish) (publish)))))

/* Transport tuples describe bytes and unscaled coefficients only. Their SQL
meaning is chosen here from the request's declaration, not in the Go bridge. */
(define tds_parameter_value (lambda (parameter) (begin
	(define value (parameter "value"))
	(define wire (parameter "type"))
	(if (nil? value) nil
		(has? '(106 108) (wire "kind")) (begin
			(define coefficient (value "coefficient"))
			(define scale (wire "scale"))
			(if (and (string? coefficient) (regexp_test coefficient "^-?(?:0|[1-9][0-9]*)$")
				(>= scale 0) (<= scale (wire "precision")) (<= (wire "precision") 38)) true
				(error "invalid numeric parameter payload"))
			(if (equal? scale 0) (tsql_parse_integer coefficient) (begin
				(define negative (equal? (substr coefficient 0 1) "-"))
				(define digits (if negative (substr coefficient 1) coefficient))
				(define padded (concat (string_repeat "0" (max 0 (+ 1 (- scale (strlen digits))))) digits))
				(define point (- (strlen padded) scale))
				(tsql_numeric_input (concat (if negative "-" "") (substr padded 0 point) "." (substr padded point))))))
		value))))
(define tds_result_value (lambda (descriptor value) (if (nil? value) nil
	(match (descriptor "kind")
		165 (list "bytes" value)
		108 (begin
			(define scale (descriptor "scale"))
			/* A zero-scale SQL contract keeps native integer IDs exact. Scaled
			coefficients currently use the existing native numeric precision. */
			(define coefficient (if (equal? scale 0) value
				(round (* value (simplify (concat "1e" scale))))))
			(list "coefficient" (string (tsql_cast_integer coefficient
				(tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807")))))
		231 (string value)
		109 (list "floating" value)
		38 (intdiv value 1)
		_ value))))

/* Bind at the request boundary, not in storage or row loops. NULL is a supplied
value; the wire descriptor and SQL declaration own the type independently. */
(define tds_bind_parameters (lambda (session declarations parameters) (begin
	(if (not (equal? (count declarations) (count parameters)))
		(error "T-SQL parameter count differs from declarations") true)
	(define names (map declarations car))
	(if (not (equal? (count names) (count (merge_unique (list names)))))
		(error "duplicate T-SQL parameter declaration") true)
	(define named (merge (map (produceN (count parameters)) (lambda (i) (begin
		(define parameter (nth parameters i))
		(define name (parameter "name"))
		(define name (if (equal? name "") (car (nth declarations i))
			(toLower (if (equal? (substr name 0 1) "@") (substr name 1) name))))
		(if (or (not (has? names name)) (not (equal? (parameter "flags") 0)))
			(error "unknown or unsupported output parameter: @" name) true)
		(list name parameter))))))
	(if (not (equal? (count parameters) (count (merge_unique (list (extract_assoc named (lambda (name _value) name)))))))
		(error "duplicate T-SQL parameter value") true)
	(define values (map declarations (lambda (declaration) (begin
		(define name (car declaration))
		(define contract (cadr declaration))
		(define parameter (named name))
		(define wire (parameter "type"))
		(if (not (has? '(31 38 104 106 108 109 165 167 173 175 231 239) (wire "kind")))
			(error "unsupported T-SQL parameter wire type: " (wire "kind")) true)
		/* A supplied NULL needs no numeric representation or conversion. Its SQL
		contract still comes exclusively from the declaration. */
		(define source_type (match (wire "kind")
			38 "BIGINT" 104 "BIT" 106 "DECIMAL" 108 "NUMERIC" 109 "FLOAT"
			165 "VARBINARY" 173 "BINARY" 167 "VARCHAR" 175 "CHAR" 231 "NVARCHAR" 239 "NCHAR" _ "NULL"))
		(if (and (not (nil? (parameter "value"))) (tsql_binary_type? source_type)
			(not (tsql_binary_type? (car contract))))
			(error "binary parameter conversion is not implemented") true)
		(define converted (if (nil? (parameter "value")) nil
			(eval (sql_type_lower_annotations
				(tsql_cast_expr (list (quote sql_type_expr) (list (quote quote) (tds_parameter_value parameter)) source_type nil)
					(car contract) (cdr contract))))))
		(list (concat "__tsql_parameter:" name) converted)))))
	/* All declarations and values are validated before installing this request. */
	(map values (lambda (item) (session (car item) (cadr item))))
	(session "__tsql_parameter_declarations" declarations)
	true)))

/* Description parses and resolves SQL contracts without running the compiled
statement. SELECT bypasses physical planning; mutations are refused here until
the parser exposes a pure metadata path for their result contract. */
(define tds_describe (lambda (schema text declarations session) (begin
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" (tds_bind_declarations schema declarations))
	(planning "__tsql_describe" true)
	(match (parse_tsql schema (tsql_statement_text text) (sql_policy (coalesceNil (session "username") "root")) planning nil)
		((symbol quote) fields) fields
		_ (error "unsupported T-SQL describe shape")))))

/* A separate frontend entrypoint keeps dialect decisions out of existing SQL
execution paths. The cache, transaction owner and compiled evaluator are shared. */
(define tsql_execute_statement (lambda (tx schema text session resultrow resultfields) (begin
	/* Expose the active scope's scalar value through the ordinary session binding.
	The common planner can then discover and bind this dependency like parameters;
	an opaque scope-reader call would hide it from shared cache/domain analysis. */
	(session "__tsql_scope_identity" ((tsql_identity_scope session) "__tsql_scope_identity"))
	(define input (tsql_statement_text text))
	(define username (coalesce (session "username") "root"))
	(define declarations (coalesceNil (session "__tsql_parameter_declarations") (list)))
	/* Cache identity includes the complete immutable declaration contract. Values
	remain invocation inputs; differently typed calls cannot reuse wrong metadata. */
	(define cache_scope (json_encode (list username declarations)))
	(define parser (lambda (database text policy planning tx) (begin
		(planning "__tsql_parameter_declarations" declarations)
		(parse_tsql database text policy planning tx))))
	(define formula (cached_parse tsql_queryplan_cache (list parser (list "exact_guards" true)) schema input
		(list (quote sql-policy-for) username) cache_scope session false tx))
	(sql_execute_formula session tx formula resultrow resultfields))))

/* Syntax recipes retain only source and declaration data. Query compilation
and all row execution stay in the existing shared SQL compiler and planner. */
(define tsql_script_cache (newcachemap))
(define tsql_scalar_cache (newcachemap))
(define tsql_batch_contracts (lambda (commands inherited) (reduce commands (lambda (contracts command)
	(match command
		'("declare" declarations) (reduce declarations (lambda (contracts declaration) (begin
			(define name (car declaration))
			(if (find contracts (lambda (item) (equal? (car item) name)) nil)
				(error "duplicate variable declaration: @" name) true)
			(append contracts (list name (cadr declaration))))) contracts)
		'("block" commands) (tsql_batch_contracts commands contracts)
		'("while" _condition body) (tsql_batch_contracts (list body) contracts)
		'("try" body recovery) (tsql_batch_contracts (merge (list body recovery)) contracts)
		'("if" _condition yes no) (tsql_batch_contracts (if no (list yes no) (list yes)) contracts)
		_ contracts)) inherited)))
(define tsql_scope_session (lambda (parent contracts owner_schema) (begin
	(define local (newsession))
	(if owner_schema (local "schema" owner_schema) true)
	(local "__tsql_parameter_declarations" contracts)
	(local "__tsql_scope" (newsession))
	(local "__tsql_connection_session" (tsql_connection_session parent))
	/* Error-reporting procedures inherit the enclosing immutable CATCH context.
	A nested CATCH still installs and restores only this invocation's fields. */
	(define error_context (parent "__tsql_error_context"))
	(if error_context (tsql_install_error_context local error_context) true)
	/* Parameters are lexical batch/procedure locals; account, transaction and
	connection state remain owned by the original session. Nested invocations
	do not overwrite a caller's local variables or identity owner. */
	(lambda args (match args
		'() (merge_unique (list (filter (local) (lambda (key)
			(not (has? '("__tsql_scope" "__tsql_connection_session") key)))) (filter (parent) (lambda (key)
				(not (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
					(or (has? '("__tsql_scope" "__tsql_connection_session") key) (strlike key "__tsql_error_%"))
					(and owner_schema (equal? key "schema"))))))))
		'(key) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))) (local key) (parent key))
		'(key value) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))) (local key value) (parent key value))
		_ (apply parent args))))))
(define tsql_scalar_expression_allowed? (lambda (expr) (match expr
	((symbol quote) _value) true
	(cons head tail) (and (not (and (symbol? head)
		(has? '(inner_select inner_select_exists inner_select_in get_column aggregate window) head)))
		(reduce tail (lambda (allowed child) (and allowed (tsql_scalar_expression_allowed? child))) true))
	_ true)))
(define tsql_scalar_formula (lambda (schema text contracts mode target tx) (begin
	(define key (json_encode (list schema text contracts mode target)))
	(tsql_scalar_cache "get_or_compute" key tx (lambda (tx) (begin
		(define planning (newsession))
		(planning "__tsql_parameter_declarations" contracts)
		(planning "__tsql_expression_only" true)
		(define info (parse_tsql schema text nil planning tx))
		(define expr (sql_info_formula info))
		(if (tsql_scalar_expression_allowed? expr) true
			(error "batch scalar expression requires a relational plan"))
		(if (and (equal? mode "predicate") (not (equal? (toUpper (sql_info_type info)) "BOOLEAN")))
			(error "IF requires a Boolean predicate") true)
		(if (and (equal? mode "dynamic")
			(not (has? tsql_addition_text_types (toUpper (sql_info_type info)))))
			(error "dynamic SQL requires a declared text expression") true)
		(if (and (equal? mode "message-text")
			(not (has? (append tsql_addition_text_types "NULL") (toUpper (sql_info_type info)))))
			(error "message substitution requires a declared text argument") true)
		(if (and (equal? mode "message-integer")
			(not (has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "NULL") (toUpper (sql_info_type info)))))
			(error "message substitution requires a declared integer argument") true)
		(define expr (if (has? '("assignment" "message-text" "message-integer") mode)
			(sql_type_formula '() (tsql_cast_expr (list (quote sql_type_expr) expr (sql_info_type info) (sql_info_collation info))
				(car target) (cdr target))) expr))
		(sql_queryplan_compile_formula tx (sql_queryplan_bind_tx_calls
			(sql_queryplan_bind_execution_session (sql_type_lower_annotations expr))))))))))
(define tsql_scalar_value (lambda (schema text session tx mode target) (begin
	/* Scalar control expressions use the same scope dependency as SELECT leaves.
	Restore the caller owner before shared compilation after a child procedure. */
	(session "__tsql_scope_identity" ((tsql_identity_scope session) "__tsql_scope_identity"))
	((tsql_scalar_formula schema text (session "__tsql_parameter_declarations") mode target tx)
		session tx (lambda (_row) (error "scalar expression emitted rows")) (lambda args true)))))
(define tsql_run_commands (lambda (commands statement session resultrow resultfields)
	(reduce commands (lambda (_result command)
		(if ((tsql_identity_scope session) "__tsql_transfer") _result
			(tsql_run_command command statement session resultrow resultfields))) 0)))
/* Procedural iterations have data dependencies; independent SQL work still uses
the shared parallel planner. The existing sequential loop carries only frontend
control state. Every predicate/leaf gets the normal cancellable statement scope. */
(define tsql_run_loop (lambda (condition body statement session resultrow resultfields) (begin
	(define scope (tsql_identity_scope session))
	(cadr (for (list false 0)
		(lambda (stopped _result) (if stopped false
			(statement condition false (lambda (tx)
				(tsql_scalar_value (session "schema") condition session tx "predicate" nil)))))
		(lambda (_stopped _result) (begin
			(define result (tsql_run_command body statement session resultrow resultfields))
			(match (scope "__tsql_transfer")
				"break" (begin (scope "__tsql_transfer" nil) (list true result))
				"continue" (begin (scope "__tsql_transfer" nil) (list false result))
				nil (list false result)
				_ (list true result)))))))))
(define tsql_error_value (lambda (context name)
	(if (nil? context) nil (context name))))
(define tsql_install_error_context (lambda (session context) (begin
	(session "__tsql_error_context" context)
	/* Expose scalar inputs through the ordinary execution-session boundary. The
	shared planner binds their current values, rather than discovering an opaque
	error-record reader in the SQL expression. Updates occur only at CATCH edges. */
	(session "__tsql_error_number" (tsql_error_value context "number"))
	(session "__tsql_error_message" (tsql_error_value context "message"))
	(session "__tsql_error_state" (tsql_error_value context "state"))
	(session "__tsql_error_severity" (tsql_error_value context "severity")))))
(define tsql_run_try (lambda (body recovery statement session resultrow resultfields)
	(try (lambda () (tsql_run_commands body statement session resultrow resultfields))
		(lambda (err) (begin
			/* The request owner checks cancellation before entering CATCH. Its
			recovery capability admits this boundary, never an aborted-tx write. */
			(statement "catch boundary" true (lambda (_tx) true))
			(define previous (session "__tsql_error_context"))
			(define context (match err
				'("tsql-error" number message state severity)
				(list "error" err "number" number "message" message "state" state "severity" severity)
				'("tsql-error" number message state)
				(list "error" err "number" number "message" message "state" state "severity" 16)
				_ (list "error" err "number" nil "message" (string err) "state" nil)))
			(tsql_install_error_context session context)
			/* Consume the recovery result before restoring the enclosing context.
			The eager argument boundary makes this dependency explicit instead of
			letting a single-use local be substituted after the session update. */
			(try (lambda ()
				((lambda (result) (begin
					(tsql_install_error_context session previous)
					result))
					(tsql_run_commands recovery statement session resultrow resultfields)))
				(lambda (cause) (begin (tsql_install_error_context session previous) (error cause)))))))))
/* Message formatting is frontend syntax over a bounded argument list. The
ordinary expression compiler owns each substitution's declared SQL contract. */
(define tsql_message_parts (parser (define parts (* (or
	(parser (define text (regex "[^%]+" false false)) (list "text" text))
	/* Format markers are case-sensitive prefixes. SQL word terminals would
	incorrectly reject alphabetic text immediately following a conversion. */
	(parser (define marker (regex "%%|%I64d|%[di]|%s" false false))
		(match marker "%%" '("text" "%") "%I64d" '("BIGINT")
			"%s" '("NVARCHAR" -1) _ '("INT")))) empty)) parts ""))
(define tsql_run_raiserror (lambda (arguments statement session) (begin
	(if (and (>= (count arguments) 3) (<= (count arguments) 23)) true
		(error "RAISERROR requires message, severity, state and at most twenty substitutions"))
	(define raised (statement "error message arguments" false (lambda (tx) (begin
		(define schema (session "schema"))
		(define text (tsql_scalar_value schema (car arguments) session tx "dynamic" nil))
		(define severity (tsql_scalar_value schema (cadr arguments) session tx "assignment" '("INT")))
		(define state (tsql_scalar_value schema (nth arguments 2) session tx "assignment" '("INT")))
		/* Informational and fatal delivery need a transport message callback;
		do not silently turn them into catchable statement errors. */
		(if (and (not (nil? severity)) (>= severity 11) (<= severity 18)) true
			(error "RAISERROR currently supports catchable severity eleven through eighteen"))
		(if (or (nil? state) (> state 255)) (error "invalid RAISERROR state") true)
		(if (nil? text) (error "RAISERROR message cannot be NULL") true)
		(define formatted (reduce (tsql_message_parts text) (lambda (result part)
			(match part
				'("text" text) (list (concat (car result) text) (cadr result))
				_ (begin
					(define index (cadr result))
					(if (< index (count arguments)) true (error "missing RAISERROR substitution"))
					(define value (tsql_scalar_value schema (nth arguments index) session tx
						(if (equal? (car part) "NVARCHAR") "message-text" "message-integer") part))
					(list (concat (car result) (coalesceNil value "(null)")) (+ index 1))))) (list "" 3)))
		(list "tsql-error" 50000 (car formatted) (if (< state 0) 1 state) severity)))))
	/* Raising after scalar binding keeps a frontend message from marking a
	successful native transaction statement as failed. RAISERROR is not DML. */
	(error raised))))
(define tsql_run_command (lambda (command statement session resultrow resultfields) (begin
	(define schema (session "schema"))
	(match command
		'("statement" text) (statement text false (lambda (tx)
			(tsql_execute_statement tx schema text session resultrow resultfields)))
		'("recovery" text) (statement text true (lambda (tx)
			(tsql_execute_statement tx schema text session resultrow resultfields)))
		'("block" commands) (tsql_run_commands commands statement session resultrow resultfields)
		'("while" condition body) (tsql_run_loop condition body statement session resultrow resultfields)
		'("try" body recovery) (tsql_run_try body recovery statement session resultrow resultfields)
		'("break") (begin ((tsql_identity_scope session) "__tsql_transfer" "break") 0)
		'("continue") (begin ((tsql_identity_scope session) "__tsql_transfer" "continue") 0)
		'("return" text) (begin
			(define status (if (nil? text) 0
				(statement "return status" false (lambda (tx)
					(coalesceNil (tsql_scalar_value schema text session tx "assignment" '("INT") ) 0)))))
			(define scope (tsql_identity_scope session))
			(scope "__tsql_return_status" status)
			(scope "__tsql_transfer" "return")
			status)
		'("raiserror" arguments) (tsql_run_raiserror arguments statement session)
		'("throw" arguments) (statement "throw" false (lambda (tx)
			(if (nil? arguments)
				(begin (define context (session "__tsql_error_context"))
					(if (nil? context) (error "THROW requires an active CATCH") (error (context "error"))))
				(begin
					(define number (tsql_scalar_value schema (car arguments) session tx "assignment" '("INT")))
					(define message (tsql_scalar_value schema (cadr arguments) session tx "assignment" '("NVARCHAR" -1)))
					(define state (tsql_scalar_value schema (nth arguments 2) session tx "assignment" '("INT")))
					(if (or (nil? number) (< number 50000) (nil? state) (< state 0) (> state 255) (nil? message))
						(error "invalid THROW arguments") (tsql_raise_error number message state))))))
		'("if" condition yes no) (begin
			(define selected (statement condition false (lambda (tx)
				(tsql_scalar_value schema condition session tx "predicate" nil))))
			(if selected (tsql_run_command yes statement session resultrow resultfields)
				no (tsql_run_command no statement session resultrow resultfields) 0))
		'("declare" declarations) (begin
			(reduce declarations (lambda (_result declaration) (begin
				(define name (car declaration))
				(define value (if (not (nil? (nth declaration 2)))
					(statement "variable initializer" false (lambda (tx)
						(tsql_scalar_value schema (nth declaration 2) session tx "assignment" (cadr declaration)))) nil))
				(session (concat "__tsql_parameter:" name) value))) nil)
			0)
		'("assign" name text) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) name)) nil))
			(if declaration true (error "undeclared variable: @" name))
			(define value (statement text false (lambda (tx)
				(tsql_scalar_value schema text session tx "assignment" (cadr declaration)))))
			(session (concat "__tsql_parameter:" name) value)
			0)
		'("create-procedure" owner name parameters source replace) (statement "procedure definition" false (lambda (tx) (begin
			(if (session "transaction") (error "procedure DDL inside explicit transactions is not supported") true)
			((sql_policy (session "username")) owner name true)
			(createprocedure owner (toLower name) "tsql-procedure" source parameters replace)
			0)))
		'("drop-procedure" owner name ignore) (statement "procedure deletion" false (lambda (_tx) (begin
			(if (session "transaction") (error "procedure DDL inside explicit transactions is not supported") true)
			((sql_policy (session "username")) owner name true)
			(define definition (find (showprocedures owner (toLower name)) (lambda (_definition) true) nil))
			(dropprocedure owner (if definition (definition "name") name) ignore)
			0)))
		'("execute" owner name arguments) (tsql_invoke_procedure owner name arguments statement session resultrow resultfields)
		'("execute-status" owner name arguments target) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) target)) nil))
			(if declaration true (error "undeclared return status variable: @" target))
			(define status (tsql_invoke_procedure owner name arguments statement session resultrow resultfields))
			(define contract (cadr declaration))
			/* The procedure frontend declares its return status INT. Conversion
			uses that contract, never the dynamic carrier of the returned value. */
			(define value (eval (sql_type_lower_annotations (tsql_cast_expr
				(list (quote sql_type_expr) (list (quote quote) status) "INT" nil) (car contract) (cdr contract)))))
			(session (concat "__tsql_parameter:" target) value)
			0)
		'("execute-dynamic" expression) (begin
			(define text (statement "dynamic SQL text" false (lambda (tx)
				(tsql_scalar_value schema expression session tx "dynamic" nil))))
			(tsql_execute_dynamic schema text '() '() statement session resultrow resultfields
				(lambda (result _status _scoped) result)))
		_ (error "unsupported batch command")))))
(define tsql_execute_command_batch (lambda (schema commands declarations supplied statement caller resultrow resultfields finish) (begin
	(define scoped (tsql_scope_session caller (tsql_batch_contracts commands declarations) schema))
	(map supplied (lambda (item) (scoped (concat "__tsql_parameter:" (car item)) (cadr item))))
	/* Completion and the declared INT return status are different contracts.
	Evaluate the SQL commands before reading their lexical scope's status. */
	(finish (tsql_run_commands commands statement scoped resultrow resultfields)
		(coalesceNil ((tsql_identity_scope scoped) "__tsql_return_status") 0) scoped))))
(define tsql_execute_batches (lambda (schema text session statement resultrow resultfields) (begin
	(session "schema" schema)
	(session "__tsql_connection_session" (tsql_connection_session session))
	(define inherited (coalesceNil (session "__tsql_parameter_declarations") '()))
	(define supplied (map inherited (lambda (declaration)
		(list (car declaration) (session (concat "__tsql_parameter:" (car declaration)))))))
	(define batches (tsql_script_cache "get_or_compute" (json_encode (list schema text)) nil
		(lambda (_tx) (map (tsql_script_batches text) (lambda (batch)
			(list "syntax" (parse_tsql_batch schema (batch "text")) "repeat" (batch "repeat")))))))
	(reduce batches (lambda (_result batch) (reduce (produceN (batch "repeat")) (lambda (_result _iteration) (begin
		(define commands (cadr (batch "syntax")))
		(tsql_execute_command_batch schema commands inherited supplied statement session resultrow resultfields
			(lambda (result _status _scoped) result)))) 0)) 0))))
(define tsql_execute_request (lambda (schema text session state sequence resultrow resultfields)
	(with_query_request session state sequence text (lambda (statement)
		(tsql_execute_batches schema text session statement resultrow resultfields)))))
/* Console imports accept the same textual script as REST and the connector.
The caller supplies its ordinary authenticated session; file I/O stays in load. */
(define load_tsql (lambda (schema stream session)
	(tsql_execute_request schema (load stream) session nil 0 print (lambda args true))))
/* Source compilers capture definitions only. Execution receives the caller's
statement owner, account, transaction session, arguments and output callbacks. */
(define tsql_procedure_compile (lambda (source context) (begin
	(define schema (context "schema"))
	(define parameters (tsql_unique_parameters (context "parameters")))
	(define commands (cadr (parse_tsql_batch schema source)))
	(define contracts (tsql_batch_contracts commands (map parameters (lambda (parameter)
		(list (car parameter) (cadr parameter))))))
	/* This compiler emits a closed lambda explicitly. Contract lists and command
	recipes are quoted data; no creator environment or privilege is captured. */
	(eval (list (quote lambda)
		(list (quote statement) (quote caller) (quote values) (quote resultrow) (quote resultfields) (quote finish))
		(list (quote begin)
			(list (quote define) (quote session)
				(list (quote tsql_scope_session) (quote caller) (list (quote quote) contracts) schema))
			(list (quote session) "schema" schema)
			(list (quote map) (quote values) (list (quote lambda) (list (quote item))
				(list (quote session) (list (quote concat) "__tsql_parameter:" (list (quote car) (quote item)))
					(list (quote cadr) (quote item)))))
			(list (quote tsql_run_commands) (list (quote quote) commands) (quote statement) (quote session)
				(quote resultrow) (quote resultfields))
			(list (quote finish) (list (quote coalesceNil)
				(list (list (quote tsql_identity_scope) (quote session)) "__tsql_return_status") 0)
				(quote session))))))))

(registerprocedurelanguage "tsql-procedure" tsql_procedure_compile)
(define tsql_execute_dynamic (lambda (schema text declarations values statement caller resultrow resultfields finish) (begin
	/* Text and bindings are scalar request inputs. Compilation and all relational
	execution use the ordinary frontend and planner, with the same statement owner
	and a new lexical scope. No caller local leaks into a nested dynamic batch. */
	(if (nil? text) (error "dynamic SQL text cannot be NULL") true)
	/* GO separates submitted scripts, but is not a statement in executed text. */
	(define syntax (tsql_script_cache "get_or_compute" (json_encode (list schema text "dynamic")) nil
		(lambda (_tx) (parse_tsql_batch schema text))))
	(tsql_execute_command_batch schema (cadr syntax) declarations values statement caller resultrow resultfields finish))))
/* Bind call syntax against formal and caller declarations before execution.
Only parameter-sized scalar tuples are collected; relational SQL stays native. */
(define tsql_check_call_order (lambda (arguments)
	(reduce arguments (lambda (named argument)
		(if (equal? (car argument) "")
			(if named (error "positional argument follows a named argument") false) true)) false)))
(define tsql_bind_call (lambda (arguments declarations statement caller) (begin
	(tsql_unique_parameters declarations)
	(tsql_check_call_order arguments)
	(define named (map (produceN (count arguments)) (lambda (i) (begin
		(define supplied (nth arguments i))
		(define name (car supplied))
		(if (and (equal? name "") (>= i (count declarations))) (error "too many procedure arguments") true)
		(list (if (equal? name "") (car (nth declarations i)) name) (cadr supplied))))))
	(define names (map named car))
	(if (equal? (count names) (count (merge_unique (list names)))) true (error "duplicate procedure argument"))
	(map names (lambda (name) (if (find declarations (lambda (declaration) (equal? (car declaration) name)) nil)
		true (error "unknown procedure argument: @" name))))
	(define outputs (filter (map declarations (lambda (declaration) (begin
		(define supplied (find named (lambda (item) (equal? (car item) (car declaration))) nil))
		(match (if supplied (cadr supplied) nil)
			'("output" target) (begin
				(if (nth declaration 3) true (error "parameter is not declared OUTPUT: @" (car declaration)))
				(define contract (find (caller "__tsql_parameter_declarations") (lambda (item) (equal? (car item) target)) nil))
				(if contract (list (car declaration) target (cadr contract))
					(error "undeclared output variable: @" target)))
			_ nil)))) (lambda (item) (not (nil? item)))))
	(define values (statement "procedure parameter binding" false (lambda (tx) (map declarations (lambda (declaration) (begin
		(define name (car declaration))
		(define supplied (find named (lambda (item) (equal? (car item) name)) nil))
		(define expression (match (if supplied (cadr supplied) '("default"))
			'("output" target) (concat "@" target)
			'("default") (if (nil? (nth declaration 2)) (error "missing procedure argument: @" name) (nth declaration 2))
			text text))
		(list name (tsql_scalar_value (caller "schema") expression caller tx "assignment" (cadr declaration)))))))))
	(list values outputs))))
(define tsql_copy_outputs (lambda (outputs scoped statement caller) (begin
	/* Convert all outgoing scalars using the callee's declared source contract
	and the caller's declared target, then publish the caller bindings. */
	(if (empty_list? outputs) true (begin
		(define values (statement "procedure output binding" false (lambda (tx) (map outputs (lambda (output)
			(list (cadr output) (tsql_scalar_value (scoped "schema") (concat "@" (car output))
				scoped tx "assignment" (nth output 2))))))))
		(reduce values (lambda (_result item)
			(caller (concat "__tsql_parameter:" (car item)) (cadr item))) nil))))))
(define tsql_execute_parameterized (lambda (schema arguments statement caller resultrow resultfields) (begin
	(tsql_check_call_order arguments)
	(define named (map (produceN (count arguments)) (lambda (i) (begin
		(define arg (nth arguments i))
		(define name (car arg))
		(list (if (equal? name "") (if (< i 2) (nth '("stmt" "params") i) "") name) (cadr arg))))))
	(define stmt (find named (lambda (arg) (equal? (car arg) "stmt")) nil))
	(define params (find named (lambda (arg) (equal? (car arg) "params")) nil))
	(if (nil? stmt) (error "dynamic SQL requires @stmt") true)
	(if (> (count (filter named (lambda (arg) (has? '("stmt" "params") (car arg)))))
		(+ 1 (if params 1 0))) (error "duplicate dynamic SQL text or declarations") true)
	(define supplied (filter named (lambda (arg) (not (has? '("stmt" "params") (car arg))))))
	(map (filter named (lambda (arg) (has? '("stmt" "params") (car arg)))) (lambda (arg)
		(match (cadr arg) (cons _head _tail) (error "dynamic text and declarations are input-only") _ true)))
	(define binding (statement "dynamic declarations" false (lambda (tx) (begin
		(define text (tsql_scalar_value (caller "schema") (cadr stmt) caller tx "dynamic" nil))
		(define contracts (if params (tds_bind_declarations schema
			(tsql_scalar_value (caller "schema") (cadr params) caller tx "dynamic" nil)) '()))
		(list text (map contracts (lambda (contract)
			(list (car contract) (cadr contract) nil (if (> (count contract) 2) (nth contract 2) false)))))))))
	(define bound (tsql_bind_call supplied (cadr binding) statement caller))
	(tsql_execute_dynamic schema (car binding) (map (cadr binding) (lambda (item) (list (car item) (cadr item))))
		(car bound) statement caller resultrow resultfields
		(lambda (_result status scoped) (begin (tsql_copy_outputs (cadr bound) scoped statement caller) status))))))
(define tsql_invoke_procedure (lambda (schema name arguments statement caller resultrow resultfields)
	(if (equal? (toLower name) "sp_executesql")
		(tsql_execute_parameterized (if (equal?? schema "sys") (caller "schema") schema)
			arguments statement caller resultrow resultfields)
		(has? '("sp_tables" "sp_columns" "sp_pkeys" "sp_statistics" "sp_fkeys" "sp_datatype_info" "sp_datatype_info_100" "sp_special_columns") (toLower name))
		(statement "metadata procedure" false (lambda (tx) (begin
			(define values (map arguments (lambda (argument) (begin
				(match (cadr argument) (cons _head _tail) (error "metadata parameters are input-only") _ true)
				(list (car argument) (tsql_scalar_value (caller "schema") (cadr argument) caller tx "value" nil))))))
			(tsql_execute_metadata (if (equal?? schema "sys") (caller "schema") schema)
				name values (sql_policy (caller "username")) resultrow resultfields))))
		(begin
			((sql_policy (caller "username")) schema name false)
			/* Retain one immutable definition and callable across argument binding. */
			(define snapshot (statement "procedure lookup" false (lambda (_tx)
				(procedure schema (toLower name) true))))
			(define definition (snapshot "declaration"))
			(if (equal? (definition "language") "tsql-procedure") true
				(error "procedure language cannot execute in this frontend"))
			(define binding (tsql_bind_call arguments (definition "parameters") statement caller))
			((snapshot "callable") statement caller (car binding) resultrow resultfields
				(lambda (status scoped) (begin (tsql_copy_outputs (cadr binding) scoped statement caller) status)))))))

(define tsql_http_query (lambda (req res schema) (begin
	(define stored (mysql_auth (req "username")))
	(if (and stored (equal? stored (password (req "password"))))
		(begin
			(define text ((req "body")))
			(try (lambda () (time (begin
				((res "header") "Content-Type" "text/event-stream; charset=utf-8")
				(define session (req "__session"))
				(extract_assoc (req "query") (lambda (key value) (session key value)))
				/* Authentication and the route own identity and database selection;
				query options may supply values, never replace either authority. */
				(session "username" (req "username"))
				(session "schema" schema)
				/* One scope owner per submitted batch, never per native insert row. */
				(session "__tsql_scope" (newsession))
				(define output (newsession))
				(define result (tsql_execute_request schema text session (req "__session_state") (req "__query_seq")
					(lambda (values) (begin (output "rows" true) ((res "jsonl") values)))
					(lambda args true)))
				(if (and (not (output "rows")) (number? result))
					((res "jsonl") (list "affected_rows" result)) true)
			) text)) (lambda (err) (begin
					(error_log (concat err) schema (req "username") text)
					((res "header") "Content-Type" "text/plain")
					((res "status") 500)
					((res "print") "SQL Error: " err)))))
		(begin
			((res "header") "Content-Type" "text/plain")
			((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
			((res "status") 401)
			((res "print") "Unauthorized"))))))
(define http_handler (begin
	(define previous http_handler)
	(lambda (req res) (match (req "path")
		(regex "^/tsql/([^/]+)$" _ schema) (tsql_http_query req res schema)
		_ (previous req res)))))
(service_registry "T-SQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/tsql/[database]" "POST, NDJSON"))

/* Driver catalogs adapt existing declarations and constraints. They never
inspect user rows, enumerate automatic access indexes, or retain a second catalog.
Descriptor types below are frontend contracts, including for empty results. */
(define tsql_metadata_arguments (lambda (arguments names) (begin
	(define bind (lambda (remaining index values) (match remaining
		(cons argument rest) (begin
			(define supplied (toLower (car argument)))
			(define supplied (if (and (> (strlen supplied) 0) (equal? (substr supplied 0 1) "@")) (substr supplied 1) supplied))
			(define name (if (equal? supplied "")
				(if (< index (count names)) (nth names index) (error "too many metadata arguments")) supplied))
			(if (has? names name) true (error "unknown metadata argument: " name))
			(if (has? (extract_assoc values (lambda (key _value) key)) name)
				(error "duplicate metadata argument") true)
			(bind rest (+ index 1) (append values name (cadr argument))))
		_ values)))
	(bind arguments 0 '()))))
(define tsql_metadata_match (lambda (name pattern patterns)
	(if (nil? pattern) true
		patterns (begin
			/* ODBC's literal wildcard spellings use the existing LIKE engine. */
			(define escaped (replace (replace pattern "[_]" "\\_") "[%]" "\\%"))
			(if (regexp_test escaped "\\[") (error "unsupported metadata bracket pattern")
				(strlike name escaped "utf8mb4_general_ci")))
		(equal?? name pattern))))
(define tsql_metadata_table_names (lambda (schema pattern patterns policy) (begin
	(if policy (policy schema true false) true)
	(filter (coalesceNil (show schema) '()) (lambda (name)
		(and (not (internal_column_name? name)) (tsql_metadata_match name pattern patterns)))))))
/* Enumerate declaration columns directly, preserving their ordinal. */
(define tsql_metadata_key_columns (lambda (columns make_row) (begin
	(define walk (lambda (remaining ordinal) (match remaining
		(cons column rest) (cons (make_row column ordinal) (walk rest (+ ordinal 1)))
		_ '())))
	(walk columns 1))))
(define tsql_metadata_foreign_columns (lambda (local remote make_row) (begin
	(define walk (lambda (left right ordinal) (match left
		(cons column rest) (match right
			(cons other others) (cons (make_row column other ordinal) (walk rest others (+ ordinal 1)))
			_ (error "foreign key declaration column count differs"))
		'() (if (empty_list? right) '() (error "foreign key declaration column count differs"))
		_ (error "invalid foreign key column declaration"))))
	(walk local remote 1))))
(define tsql_metadata_constraints (lambda (schema name) (show (table schema name) "constraints")))
(define tsql_metadata_key_name (lambda (key)
	(if (equal? (key "constraint_name") "") (key "Id") (key "constraint_name"))))
(define tsql_metadata_fk_rule (lambda (mode) (match mode "cascade" 0 "set null" 2 "set default" 3 _ 1)))
(define tsql_metadata_descriptors (lambda (names integers)
	(map names (lambda (name)
		(if (has? integers name) (list "name" name "kind" 38 "size" 4 "flags" 1)
			(list "name" name "kind" 231 "size" 8000 "flags" 1))))))
/* (type name, ODBC code, precision, byte length, scale, radix, datetime subtype).
Only declarations decide these values: an empty or all-NULL table is identical. */
(define tsql_metadata_type (lambda (column version) (begin
	(define dimensions (column "Dimensions"))
	(define length (if (empty_list? dimensions) 1 (car dimensions)))
	(define type (toLower (column "RawType")))
	(match type
		"bigint" '("bigint" -5 19 8 0 10 nil)
		"int" '("int" 4 10 4 0 10 nil)
		"integer" '("int" 4 10 4 0 10 nil)
		"smallint" '("smallint" 5 5 2 0 10 nil)
		"tinyint" '("tinyint" -6 3 1 0 10 nil)
		"boolean" '("bit" -7 1 1 0 nil nil)
		"bit" '("bit" -7 1 1 0 nil nil)
		"real" '("real" 7 24 4 nil 2 nil)
		"float" '("float" 6 53 8 nil 2 nil)
		"double" '("float" 6 53 8 nil 2 nil)
		"money" '("money" 3 19 8 4 10 nil)
		"smallmoney" '("smallmoney" 3 10 4 4 10 nil)
		"uniqueidentifier" '("uniqueidentifier" -11 36 16 nil nil nil)
		"date" (list "date" (if (equal? version 2) 9 91) 10 6 nil nil 1)
		"datetime" (list "datetime" (if (equal? version 2) 11 93) 23 16 3 nil 3)
		"smalldatetime" (list "smalldatetime" (if (equal? version 2) 11 93) 16 16 0 nil 3)
		"rowversion" '("rowversion" -2 8 8 nil nil nil)
		"timestamp" (begin
			(define scale (if (empty_list? dimensions) 6 (car dimensions)))
			(list "datetime2" (if (equal? version 2) 11 93) (if (> scale 0) (+ 20 scale) 19) 16 scale nil 3))
		_ (if (has? '("decimal" "numeric") type)
			(begin
				(define precision (if (empty_list? dimensions) 18 (car dimensions)))
				(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
				(list type (if (equal? type "decimal") 3 2) precision
					(if (<= precision 9) 5 (<= precision 19) 9 (<= precision 28) 13 17) scale 10 nil))
			(has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary") type)
			(list type (match type "varchar" 12 "char" 1 "nvarchar" -9 "nchar" -8 "binary" -2 "varbinary" -3)
				length (if (and (> length 0) (has? '("nvarchar" "nchar") type)) (* length 2) length) nil nil nil)
			(error "unsupported metadata declaration type: " type))))))
(define tsql_metadata_column_row (lambda (schema name column ordinal version) (begin
	(define contract (tsql_metadata_type column version))
	(define subtype (nth contract 6))
	(define type (car contract))
	(list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name
		"COLUMN_NAME" (column "Field") "DATA_TYPE" (nth contract 1) "TYPE_NAME" type
		"PRECISION" (nth contract 2) "LENGTH" (nth contract 3) "SCALE" (nth contract 4)
		"RADIX" (nth contract 5) "NULLABLE" (if (column "Null") 1 0)
		"REMARKS" (if (equal? (column "Comment") "") nil (column "Comment"))
		"COLUMN_DEF" (if (equal? (column "DefaultExpression") "") (column "Default") (column "DefaultExpression"))
		"SQL_DATA_TYPE" (if subtype 9 (nth contract 1)) "SQL_DATETIME_SUB" subtype
		"CHAR_OCTET_LENGTH" (if (has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary") type) (nth contract 3) nil)
		"ORDINAL_POSITION" ordinal "IS_NULLABLE" (if (column "Null") "YES" "NO")))))
/* Type info is a frontend declaration catalog, not a registry of runtime tags. */
(define tsql_metadata_type_info (lambda (options version) (begin
	(define definitions (list (list "bigint" '()) (list "int" '()) (list "smallint" '()) (list "tinyint" '())
		(list "bit" '()) (list "real" '()) (list "float" '()) (list "decimal" '(38 0)) (list "numeric" '(38 0))
		(list "varchar" '(8000)) (list "char" '(8000)) (list "date" '())))
	(define rows (map definitions (lambda (definition) (begin
		(define contract (tsql_metadata_type (list "RawType" (car definition) "Dimensions" (cadr definition)) version))
		(define type (car contract))
		(define character (has? '("varchar" "char") type))
		(define decimal (has? '("decimal" "numeric") type))
		(list "TYPE_NAME" type "DATA_TYPE" (nth contract 1) "PRECISION" (nth contract 2)
			"LITERAL_PREFIX" (if (or character (equal? type "date")) "'" nil)
			"LITERAL_SUFFIX" (if (or character (equal? type "date")) "'" nil)
			"CREATE_PARAMS" (if character "length" decimal "precision,scale" nil)
			"NULLABLE" 1 "CASE_SENSITIVE" (if character 1 0) "SEARCHABLE" 3
			"UNSIGNED_ATTRIBUTE" (if (equal? type "tinyint") 1 (has? '("bigint" "int" "smallint" "decimal" "numeric" "real" "float") type) 0 nil)
			"MONEY" 0 "AUTO_INCREMENT" (if (has? '("bigint" "int" "smallint" "tinyint") type) 1 0)
			"LOCAL_TYPE_NAME" type "MINIMUM_SCALE" (nth contract 4) "MAXIMUM_SCALE" (if decimal 38 (nth contract 4))
			"SQL_DATA_TYPE" (if (nth contract 6) 9 (nth contract 1)) "SQL_DATETIME_SUB" (nth contract 6)
			"NUM_PREC_RADIX" (nth contract 5) "INTERVAL_PRECISION" nil)))))
	(list '("TYPE_NAME" "DATA_TYPE" "PRECISION" "LITERAL_PREFIX" "LITERAL_SUFFIX" "CREATE_PARAMS" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "LOCAL_TYPE_NAME" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		'("DATA_TYPE" "PRECISION" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		(filter rows (lambda (row) (or (nil? (options "data_type")) (equal? (options "data_type") 0) (equal? (row "DATA_TYPE") (options "data_type")))))))))
(define tsql_metadata_result (lambda (schema procedure arguments policy) (begin
	(define procedure (toLower procedure))
	(define procedure (if (and (>= (strlen procedure) 4) (equal? (substr procedure 0 4) "sys.")) (substr procedure 4) procedure))
	(define names (match procedure
		"sp_datatype_info" '("data_type" "odbcver")
		"sp_datatype_info_100" '("data_type" "odbcver")
		"sp_tables" '("table_name" "table_owner" "table_qualifier" "table_type" "fusepattern")
		"sp_columns" '("table_name" "table_owner" "table_qualifier" "column_name" "odbcver")
		"sp_pkeys" '("table_name" "table_owner" "table_qualifier")
		"sp_statistics" '("table_name" "table_owner" "table_qualifier" "index_name" "is_unique" "accuracy")
		"sp_fkeys" '("pktable_name" "pktable_owner" "pktable_qualifier" "fktable_name" "fktable_owner" "fktable_qualifier")
		"sp_special_columns" '("table_name" "table_owner" "qualifier" "col_type" "scope" "nullable" "odbcver")
		_ (error "unsupported metadata procedure: " procedure)))
	(define options (tsql_metadata_arguments arguments names))
	/* Never silently substitute another database. Policy executes before catalog
	access, and every table access below belongs to this authenticated database. */
	(map '("table_qualifier" "pktable_qualifier" "fktable_qualifier" "qualifier") (lambda (key)
		(if (or (nil? (options key)) (equal?? schema (options key))) true
			(error "metadata qualifier is not the current database"))))
	(if policy (policy schema true false) true)
	(define owner (coalesceNil (options "table_owner") "dbo"))
	(define version (coalesceNil (options "odbcver") 2))
	(if (has? '(2 3) version) true (error "unsupported ODBC version"))
	(define result (match procedure
		"sp_datatype_info" (tsql_metadata_type_info options version)
		"sp_datatype_info_100" (tsql_metadata_type_info options version)
		"sp_tables" (begin
			(define filter_type (options "table_type"))
			(define types (if (nil? filter_type) '("TABLE")
				(map (split filter_type ",") (lambda (value) (toUpper (replace (strtrim value) "'" ""))))))
			(define rows (if (and (tsql_metadata_match "dbo" owner true) (has? types "TABLE"))
				(map (tsql_metadata_table_names schema (options "table_name") (not (equal? (coalesceNil (options "fusepattern") 1) 0)) policy)
					(lambda (name) (list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name "TABLE_TYPE" "TABLE" "REMARKS" nil))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "TABLE_TYPE" "REMARKS") '() rows))
		"sp_columns" (begin
			(if (nil? (options "table_name")) (error "sp_columns requires a table name") true)
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(merge (map (tsql_metadata_table_names schema (options "table_name") true policy) (lambda (name) (begin
					(define columns (filter (show (table schema name)) (lambda (column) (not (internal_column_name? (column "Field"))))))
					(map (filter (produceN (count columns)) (lambda (i)
						(tsql_metadata_match ((nth columns i) "Field") (options "column_name") true)))
						(lambda (i) (tsql_metadata_column_row schema name (nth columns i) (+ i 1) version))))))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "COLUMN_NAME" "DATA_TYPE" "TYPE_NAME" "PRECISION" "LENGTH" "SCALE" "RADIX" "NULLABLE" "REMARKS" "COLUMN_DEF" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "CHAR_OCTET_LENGTH" "ORDINAL_POSITION" "IS_NULLABLE")
				'("DATA_TYPE" "PRECISION" "LENGTH" "SCALE" "RADIX" "NULLABLE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "CHAR_OCTET_LENGTH" "ORDINAL_POSITION") rows))
		"sp_special_columns" (begin
			(if (options "table_name") true (error "sp_special_columns requires a table name"))
			(define kind (toUpper (coalesceNil (options "col_type") "R")))
			(define scope (toUpper (coalesceNil (options "scope") "T")))
			(define nullable (toUpper (coalesceNil (options "nullable") "U")))
			(if (and (has? '("R" "V") kind) (has? '("C" "T" "S") scope) (has? '("O" "U") nullable)) true
				(error "unsupported special-column option"))
			(define rows (if (equal?? owner "dbo") (merge (map (tsql_metadata_table_names schema (options "table_name") false policy) (lambda (name) (begin
				(define columns (show (table schema name)))
				(define keys ((tsql_metadata_constraints schema name) "Unique"))
				(define primary (find keys (lambda (key) (equal? (key "Id") "PRIMARY"))))
				(define candidate (coalesceNil primary (find keys (lambda (key)
					(reduce (key "Cols") (lambda (valid col) (and valid (not ((find columns (lambda (column) (equal? (column "Field") col))) "Null")))) true)))))
				(define names (if (equal? kind "V")
					(map (filter columns (lambda (column) (equal?? (column "RawType") "ROWVERSION"))) (lambda (column) (column "Field")))
					(if candidate (candidate "Cols") '())))
				(map (filter names (lambda (col) (or (equal? nullable "U")
					(not ((find columns (lambda (column) (equal? (column "Field") col))) "Null"))))) (lambda (col) (begin
						(define column (find columns (lambda (column) (equal? (column "Field") col))))
						(define contract (tsql_metadata_type column version))
						(list "SCOPE" (match scope "C" 0 "T" 1 "S" 2) "COLUMN_NAME" col "DATA_TYPE" (nth contract 1)
							"TYPE_NAME" (car contract) "PRECISION" (nth contract 2) "LENGTH" (nth contract 3) "SCALE" (nth contract 4) "PSEUDO_COLUMN" 1)))))))) '()))
			(list '("SCOPE" "COLUMN_NAME" "DATA_TYPE" "TYPE_NAME" "PRECISION" "LENGTH" "SCALE" "PSEUDO_COLUMN")
				'("SCOPE" "DATA_TYPE" "PRECISION" "LENGTH" "SCALE" "PSEUDO_COLUMN") rows))
		"sp_pkeys" (begin
			(if (nil? (options "table_name")) (error "sp_pkeys requires a table name") true)
			(define rows (if (equal?? owner "dbo")
				(merge (map (tsql_metadata_table_names schema (options "table_name") false policy) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name) "Unique"))
					(merge (map (filter keys (lambda (key) (equal? (key "Id") "PRIMARY"))) (lambda (key)
						(tsql_metadata_key_columns (key "Cols") (lambda (column ordinal)
							(list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name "COLUMN_NAME" column
								"KEY_SEQ" ordinal "PK_NAME" (tsql_metadata_key_name key))))))))))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "COLUMN_NAME" "KEY_SEQ" "PK_NAME") '("KEY_SEQ") rows))
		"sp_statistics" (begin
			(if (nil? (options "table_name")) (error "sp_statistics requires a table name") true)
			(if (has? '(nil "Y" "N") (options "is_unique")) true (error "unsupported unique filter"))
			(if (has? '(nil "Q") (options "accuracy")) true (error "unsupported statistics accuracy"))
			(define rows (if (equal?? owner "dbo")
				(merge (map (tsql_metadata_table_names schema (options "table_name") false policy) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name) "Unique"))
					(merge (map (filter keys (lambda (key) (tsql_metadata_match (tsql_metadata_key_name key) (options "index_name") true))) (lambda (key)
						(tsql_metadata_key_columns (key "Cols") (lambda (column ordinal)
							/* ODBC OTHER describes the logical key, not a physical index
							technology. Counts, pages and physical ordering are unknown. */
							(list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name "NON_UNIQUE" 0 "INDEX_QUALIFIER" schema
								"INDEX_NAME" (tsql_metadata_key_name key) "TYPE" 3 "SEQ_IN_INDEX" ordinal "COLUMN_NAME" column
								"COLLATION" nil "CARDINALITY" nil "PAGES" nil "FILTER_CONDITION" nil)))))))))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "NON_UNIQUE" "INDEX_QUALIFIER" "INDEX_NAME" "TYPE" "SEQ_IN_INDEX" "COLUMN_NAME" "COLLATION" "CARDINALITY" "PAGES" "FILTER_CONDITION")
				'("NON_UNIQUE" "TYPE" "SEQ_IN_INDEX" "CARDINALITY" "PAGES") rows))
		"sp_fkeys" (begin
			(define parent (options "pktable_name"))
			(define child (options "fktable_name"))
			(if (or parent child) true (error "sp_fkeys requires a table filter"))
			(define rows (if (and (equal?? (coalesceNil (options "pktable_owner") "dbo") "dbo")
				(equal?? (coalesceNil (options "fktable_owner") "dbo") "dbo"))
				(merge (map (tsql_metadata_table_names schema child false policy) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name) "ForeignKeys"))
					(merge (map (filter keys (lambda (key) (and (equal? (key "Role") "referencing")
						(tsql_metadata_match (key "OtherTable") parent false)))) (lambda (key) (begin
							(define referenced (table schema (key "OtherTable")))
							(define candidate (if referenced (find ((show referenced "constraints") "Unique")
								(lambda (unique) (equal? (unique "Cols") (key "OtherColumns")))) nil))
							(tsql_metadata_foreign_columns (key "LocalColumns") (key "OtherColumns") (lambda (column other ordinal)
								(list "PKTABLE_QUALIFIER" schema "PKTABLE_OWNER" "dbo" "PKTABLE_NAME" (key "OtherTable") "PKCOLUMN_NAME" other
									"FKTABLE_QUALIFIER" schema "FKTABLE_OWNER" "dbo" "FKTABLE_NAME" name "FKCOLUMN_NAME" column "KEY_SEQ" ordinal
									"UPDATE_RULE" (tsql_metadata_fk_rule (key "UpdateMode")) "DELETE_RULE" (tsql_metadata_fk_rule (key "DeleteMode"))
									"FK_NAME" (key "Id") "PK_NAME" (if candidate (tsql_metadata_key_name candidate) nil) "DEFERRABILITY" 7))))))))))) '()))
			(list '("PKTABLE_QUALIFIER" "PKTABLE_OWNER" "PKTABLE_NAME" "PKCOLUMN_NAME" "FKTABLE_QUALIFIER" "FKTABLE_OWNER" "FKTABLE_NAME" "FKCOLUMN_NAME" "KEY_SEQ" "UPDATE_RULE" "DELETE_RULE" "FK_NAME" "PK_NAME" "DEFERRABILITY")
				'("KEY_SEQ" "UPDATE_RULE" "DELETE_RULE" "DEFERRABILITY") rows))))
	(list "columns" (tsql_metadata_descriptors (car result) (cadr result)) "rows" (nth result 2)))))
(define tsql_execute_metadata (lambda (schema procedure arguments policy resultrow resultfields) (begin
	(define result (tsql_metadata_result schema procedure arguments policy))
	(resultfields (map (result "columns") (lambda (column) (column "name"))) (result "columns"))
	(map (result "rows") resultrow)
	0)))

/* Network callbacks use the same accounts, transaction owner and SQL execution
path as the other frontends. Wire actions are interpreted here, never in scm/. */
(define tds_frontend (lambda (event session payload row fields state sequence) (match event
	"login" (begin
		(define user (payload "username"))
		(define database (payload "database"))
		(define stored (mysql_auth user))
		(if (and stored (equal? stored (password (payload "password"))) (mysql_schema user database))
			(begin
				((sql_policy user) database true false)
				(define session (newsession))
				(session "username" user)
				(session "schema" database)
				(session "syntax" "tsql")
				(session "wire_metadata" true)
				session) nil))
	"close" (tx_rollback session)
	"reset" (begin
		(tx_rollback session)
		(define replacement (newsession))
		(replacement "username" (session "username"))
		(replacement "schema" (session "schema"))
		(replacement "syntax" "tsql")
		(replacement "wire_metadata" true)
		replacement)
	"transaction" (begin
		(if (and (equal? (payload "action") 1) (not (has? '(0 2) (payload "isolation"))))
			(error "unsupported TDS transaction isolation") true)
		(define sql (match (payload "action") 1 "BEGIN TRANSACTION" 2 "COMMIT" 3 "ROLLBACK" _ (error "unknown transaction action")))
		(tsql_execute_request (session "schema") sql session state sequence (lambda (_row) true) (lambda args true))
		(list "rows" 0 "transaction" (payload "action")))
	"execute" (begin
		(define output (newsession))
		(define result (with_query_request session state sequence (payload "text") (lambda (statement) (begin
			(tds_bind_parameters session (tds_bind_declarations (payload "database") (payload "declarations"))
				(payload "parameters"))
			(tsql_execute_batches (payload "database") (payload "text") session statement
				(lambda (values) (row (map (output "descriptors") (lambda (descriptor) (begin
					(define value (values (descriptor "name")))
					(tds_result_value descriptor value))))))
				(lambda (_names descriptors) (begin
					(output "descriptors" descriptors)
					(fields descriptors))))))))
		/* NOCOUNT changes protocol completion only; native DML still computes
		its affected-row count and result rows follow the shared planner. */
		(if (session "__tsql_nocount")
			(list "rows" (if (number? result) result 0) "no_count" true)
			(if (number? result) (list "rows" result) nil)))
	"describe" (tds_describe (payload "database") (payload "text") (payload "declarations") session)
	"metadata" (begin
		(define arguments (map (payload "parameters") (lambda (parameter) (begin
			(if (equal? (parameter "flags") 0) true (error "metadata output parameters are unsupported"))
			(if (has? '(31 38 104 167 175 231 239) ((parameter "type") "kind")) true
				(error "unsupported metadata argument wire type"))
			(list (parameter "name") (parameter "value"))))))
		(define result (tsql_metadata_result (session "schema") (payload "procedure") arguments (sql_policy (session "username"))))
		(fields (result "columns"))
		(map (result "rows") (lambda (values)
			(row (map (result "columns") (lambda (column) (begin
				(define value (values (column "name")))
				(if (nil? value) nil (equal? (column "kind") 231) (string value) (intdiv value 1))))))))
		(list "rows" (count (result "rows"))))
	_ (error "unknown TDS frontend callback"))))
(if (arg "tds-port" nil) (begin
	(define port (arg "tds-port" nil))
	(tds port tds_frontend (arg "tds-database" "memcp-tests"))
	(service_registry "T-SQL protocol" (list port "" "TDS"))
	(print "TDS listener on port " port)))
