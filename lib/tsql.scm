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

/* Alias DDL and database lifecycle share the common SQL DDL gate. Reads and
DML never acquire it; the database must still exist once ownership is held. */
(define tsql_type_ddl (lambda (database tx operation) (begin
	/* Catalog-row writes and native column publication do not yet share a
	transactional DDL contract. Refuse transaction-local DDL before either one
	changes, matching the frontend's source-procedure DDL restriction. */
	(if (tx_requires_query_local_cache tx) (error "alias DDL inside explicit transactions is not supported") true)
	(sql_database_ddl database tx (lambda () (begin
		/* This catalog enumeration is DDL-only, never a planner lookup. */
		(if (nil? (show database)) (error "database does not exist: " database) true)
		(operation)))))))
(define tsql_qualified_type (lambda (database name)
	(concat "[" (replace database "]" "]]") "].[dbo].[" (replace name "]" "]]") "]")))
(define tsql_alias_name_parts (parser '(
	(define database tsql_identifier) "." (define owner tsql_identifier) "." (define name tsql_identifier)
) (if (equal?? owner "dbo") (list database name) (error "unsupported alias owner: " owner))))
(define tsql_read_type_alias (lambda (database name)
	(scan nil (table "system" false "tsql_types" false)
		(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false))
		(list database) '("name") (lambda (candidate) (equal?? candidate name))
		'("id" "name" "base_type" "dimensions" "nullable")
		(lambda (_ id canonical base dimensions nullable)
			(list "id" id "name" canonical "base_type" base "dimensions" (json_decode dimensions) "nullable" nullable))
		nil (lambda (a b) (if (nil? a) b (nil? b) a (error "ambiguous alias type: " name))))))
/* Alias memoization belongs to one compiler session. Scoped values are not
copied into a later compilation, connection session, or executable plan. DDL
and execution-time declaration checks read the authoritative table directly.
Keys retain their spelling: Unicode lowercasing is not the same equivalence
relation as the case-insensitive declaration lookup. */
(define tsql_type_alias_cache (lambda (planning_session)
	(planning_session "get_or_compute_scoped" planning_session "__tsql_type_alias_cache"
		(lambda () (newcachemap (lambda (key)
			(match (json_decode key) '(database name) (tsql_read_type_alias database name))))))))
(define tsql_find_type_alias (lambda (database name planning_session)
	(if (nil? planning_session) (tsql_read_type_alias database name)
		((tsql_type_alias_cache planning_session) (json_encode (list database name))))))
(define tsql_alias_declaration (lambda (type planning_session) (begin
	(define builtin (tsql_builtin_type_declaration type))
	(if builtin builtin
		(equal? (substr type 0 1) "[") (begin
			(define parts (tsql_alias_name_parts type))
			(coalesceNil (tsql_find_type_alias (car parts) (cadr parts) planning_session)
				(error "alias type no longer exists: " type))) nil))))
(define tsql_effective_column (lambda (column planning_session)
	(if (nil? column) nil (begin
		(define alias (tsql_alias_declaration (column "RawType") planning_session))
		(if (nil? alias) column
			(set_assoc (set_assoc (set_assoc (set_assoc column "DeclaredType" (column "RawType"))
				"RawType" (alias "base_type")) "Type" (alias "base_type")) "AliasId" (alias "id")))))))
(define tsql_type_alias_matches? (lambda (type identity) (begin
	(define parts (tsql_alias_name_parts type))
	(define alias (tsql_find_type_alias (car parts) (cadr parts) nil))
	(if (nil? alias) false (equal? (alias "id") identity)))))
(define tsql_guard_type_alias (lambda (type identity planning_session)
	(planner_record_guard_condition (list (quote tsql_type_alias_matches?) type identity) planning_session)))
/* Context is attached only to a consuming compilation, after stored views
have expanded. Persisted parser definitions retain the context-free compiler;
the common binder consumes the temporary callback before planning. */
(define tsql_column_compiler (lambda (planning_session) (lambda (info declaration)
	(if (nil? declaration) info (begin
		(define alias (tsql_alias_declaration (declaration "RawType") planning_session))
		(if (nil? alias) info (begin
			(if (alias "id") (tsql_guard_type_alias (declaration "RawType") (alias "id") planning_session) true)
			(sql_info (sql_info_formula info) (alias "base_type")
				(if (sql_text_type? (alias "base_type"))
					(list (coalesce (declaration "Collation") "utf8mb4_general_ci") 2) nil)))))))))
(define tsql_compile_column (tsql_column_compiler nil))
(define tsql_bind_column_compilers (lambda (node compiler) (match node
	((symbol quote) _datum) node
	((symbol sql_compile_column) _previous expression) (list (quote sql_compile_column) compiler expression)
	(cons _head _tail) (map node (lambda (item) (tsql_bind_column_compilers item compiler)))
	_ node)))
(define tsql_build_dml_plan (lambda (schema name alias sources columns condition order limit offset planning_session tx) (begin
	(define compiler (tsql_column_compiler planning_session))
	(build_dml_plan schema name alias
		(tsql_bind_column_compilers sources compiler)
		(tsql_bind_column_compilers columns compiler)
		(tsql_bind_column_compilers condition compiler)
		(tsql_bind_column_compilers order compiler) limit offset planning_session tx))))
(define create_sql_type_alias (lambda (database name base dimensions nullable tx)
	(tsql_type_ddl database tx (lambda () (begin
		/* Until colliding user aliases have a bound precedence contract, reject
		the ambiguity explicitly instead of shadowing the built-in declaration. */
		(if (equal?? name "sysname") (error "user alias colliding with built-in type is not supported") true)
		(define native_type (toUpper base))
		(if (tsql_builtin_type_declaration native_type) (error "alias base must be a native type") true)
		(define native_dimensions (tsql_decl_dimensions native_type dimensions))
		(if (has? '("TIMESTAMP" "ROWVERSION") native_type) (error "generated versions cannot be alias base types") true)
		(if (tsql_find_type_alias database name nil) (error "alias type already exists: " name) true)
		(insert (table "system" false "tsql_types" false) '("database" "name" "base_type" "dimensions" "nullable")
			(list (list database name native_type (json_encode native_dimensions) nullable)))
		true)))))
(define drop_sql_type_alias (lambda (database name ifexists tx)
	(tsql_type_ddl database tx (lambda () (begin
		(define alias (tsql_find_type_alias database name nil))
		(if (nil? alias) (if ifexists false (error "alias type does not exist: " name)) (begin
			(define qualified (tsql_qualified_type database (alias "name")))
			/* Enumerating declarations is DROP TYPE work, never query costing or
			row execution. The database gate excludes concurrent alias consumers. */
			(define dependent (find (show database) (lambda (relation)
				(find (show database relation) (lambda (column) (equal?? (column "RawType") qualified)))) nil))
			(if dependent (error "alias type is used by table: " dependent) true)
			(scan nil (table "system" false "tsql_types" false)
				(list 369436175368192 (scan_boundary "equal" "id" 0 0 true true "" false))
				(list (alias "id")) '() (lambda () true) '("$update")
				(lambda (count remove) (begin (remove) (+ count 1))) 0 +)
			true)))))))

(define tsql_native_types '("VARCHAR" "CHAR" "NVARCHAR" "NCHAR" "BINARY" "VARBINARY"
	"DECIMAL" "NUMERIC" "FLOAT" "TIME" "DATETIME2" "DATETIMEOFFSET" "INT" "INTEGER" "BIGINT"
	"SMALLINT" "TINYINT" "BIT" "REAL" "MONEY" "SMALLMONEY" "DATE" "DATETIME" "SMALLDATETIME"
	"ROWVERSION" "TIMESTAMP" "UNIQUEIDENTIFIER" "TEXT" "NTEXT" "IMAGE"))
(define tsql_schema_type_parts (parser (define parts (+ tsql_identifier ".")) parts))
(define tsql_resolve_schema_type (lambda (database type dimensions planning_session) (begin
	(define parts (tsql_schema_type_parts type))
	(define builtin (match parts
		'(name) (tsql_builtin_type_declaration name)
		'(owner name) (if (equal?? owner "sys") (tsql_builtin_type_declaration name) nil)
		_ nil))
	(define native (match parts
		'(name) (if (has? tsql_native_types (toUpper name)) (toUpper name) nil)
		'(owner name) (if (equal?? owner "sys") (toUpper name) nil)
		_ nil))
	(if builtin (begin
		(tsql_decl_dimensions "SYSNAME" dimensions)
		(if (and (equal? (count parts) 1) (tsql_find_type_alias database (car parts) planning_session))
			(error "user alias colliding with built-in type is not supported") true)
		(merge (list "type" (if (equal? (count parts) 1) "SYSNAME" "[sys].[SYSNAME]")) builtin))
		native (list "type" native "dimensions" (tsql_decl_dimensions native dimensions) "nullable" true)
		(begin
			(define name (match parts
				'(name) name
				'(owner name) (if (equal?? owner "dbo") name (error "unsupported alias owner: " owner))
				'(catalog owner name) (if (and (equal? catalog database) (equal?? owner "dbo")) name
					(error "alias must belong to the target database"))
				_ (error "invalid alias type name")))
			(if (empty_list? dimensions) true (error "alias dimensions cannot be overridden"))
			(define alias (coalesceNil (tsql_find_type_alias database name planning_session) (error "unknown type: " name)))
			(merge (list "type" (tsql_qualified_type database (alias "name"))) alias))))))
(define tsql_prepare_column (lambda (database definition planning_session) (match definition
	'((symbol list) "column" name type dimensions attributes) (begin
		(define contract (tsql_resolve_schema_type database type (cdr dimensions) planning_session))
		(if (contract "id") (tsql_guard_type_alias (contract "type") (contract "id") planning_session) true)
		(define base (coalesceNil (contract "base_type") (contract "type")))
		(define native_dimensions (contract "dimensions"))
		(define options (cdr attributes))
		(if (and (has? '("ROWVERSION" "TIMESTAMP") base)
			(or (has_assoc? options "default") (has_assoc? options "default_expression")))
			(error "generated version column cannot have a DEFAULT declaration") true)
		(define null_options (if (or (nil? (contract "base_type")) (has_assoc? options "null")) options
			(append options "null" (contract "nullable"))))
		(if (and (contract "id") (null_options "auto_increment"))
			(error "IDENTITY alias declarations are not yet supported") true)
		(define identity_options (tsql_identity_attributes base native_dimensions null_options))
		/* Frozen defaults use the same frontend conversion as explicit writes.
		The existing native default field remains their only persisted value. */
		(define default_options (if (has_assoc? identity_options "default")
			(set_assoc identity_options "default" (eval (sql_type_formula '()
				(tsql_cast_expr (identity_options "default") base native_dimensions)))) identity_options))
		(define published_options (if (contract "id") (append default_options "__tsql_alias_id" (contract "id")) default_options))
		(list (quote list) "column" name (tsql_storage_type (contract "type"))
			(cons (quote list) native_dimensions) (cons (quote list) published_options)))
	_ definition)))
(define tsql_publish_alias_attributes (lambda (type attributes) (begin
	(if (has_assoc? attributes "__tsql_alias_id")
		(if (equal? ((tsql_alias_declaration type nil) "id") (attributes "__tsql_alias_id")) true
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
			(if (sql_create_column (table database false name false) column_name type dimensions
				(merge (sql_add_column_attributes attributes) '("name_ignorecase" true))) true
				(error "column already exists: " column_name)))) true))))
	(if (find columns (lambda (column) (has_assoc? (nth column 4) "__tsql_alias_id")))
		(tsql_type_ddl database tx publish) (publish)))))

/* Transport tuples describe bytes and unscaled coefficients only. Their SQL
meaning is chosen here from the request's declaration, not in the Go bridge. */
/* The transport exposes integer day/clock fields, never a SQL runtime type.
Validate their declaration-specific units once per parameter. The current native
calendar carrier keeps whole seconds; refuse fractional loss and offset loss
until those contracts can be represented by the existing date primitives. */
(define tds_temporal_parameter_value (lambda (wire value) (begin
	(define kind (wire "kind"))
	(define days (value "days"))
	(define ticks (value "ticks"))
	(define offset (value "offset"))
	(if (and (int? days) (int? ticks) (int? offset)) true
		(error "invalid temporal parameter fields"))
	(define legacy (equal? kind 111))
	(define short (and legacy (equal? (wire "size") 4)))
	(if (and legacy (not (has? '(4 8) (wire "size"))))
		(error "invalid legacy temporal parameter size") true)
	(define factor (if (has? '(41 42 43) kind) (begin
		(define scale (wire "scale"))
		(if (and (int? scale) (>= scale 0) (<= scale 7)) true
			(error "invalid temporal parameter scale"))
		(tsql_parse_integer (concat "1" (string_repeat "0" scale))))
		legacy (if short 1 300) 1))
	(if (and (equal? kind 41) (not (equal? days 0)))
		(error "invalid time parameter day") true)
	(if (and (>= days (if legacy (if short 0 -53690) 0))
		(<= days (if legacy (if short 65535 2958463) 3652058))) true
		(error "temporal parameter calendar overflow"))
	(if (and (>= ticks 0) (< ticks (if short 1440 (* 86400 factor)))
		(or (not (equal? kind 40)) (equal? ticks 0))) true
		(error "temporal parameter clock overflow"))
	(if (and (equal? kind 43) (or (< offset -840) (> offset 840)))
		(error "temporal parameter offset overflow") true)
	(if (equal? kind 43)
		(error "offset temporal parameter preservation is not implemented")
		(not (equal? offset 0)) (error "unexpected temporal parameter offset") true)
	(if (equal? (mod ticks factor) 0) true
		(error "fractional temporal parameter precision is not implemented"))
	(define seconds (if short (* ticks 60) (intdiv ticks factor)))
	(define date (tsql_calendar_shift_seconds
		(date_add (parse_date (if (or legacy (equal? kind 41)) "1900-01-01" "0001-01-01")) days "DAY") seconds))
	/* TIME uses the frontend's existing clock-text carrier at the boundary;
	other temporal contracts retain the normal native Unix calendar value. */
	(if (equal? kind 41) (format_date date "%H:%i:%s") date))))
(define tds_money_small? (lambda (wire) (match (wire "size")
	4 true 8 false _ (error "invalid currency parameter byte width"))))
(define tds_parameter_value (lambda (parameter) (begin
	(define value (parameter "value"))
	(define wire (parameter "type"))
	(if (nil? value) nil
		(equal? (wire "kind") 110) (begin
			(define small (tds_money_small? wire))
			/* This checks the codec's integer payload contract. The descriptor,
			not the carrier, selected the fixed-scale SQL declaration above. */
			(if (and (int? value) (or (not small) (and (>= value -2147483648) (<= value 2147483647)))) true
				(error "invalid currency parameter coefficient"))
			(define number (/ value 10000))
			(if (equal? (tsql_money_coefficient number small) value) number
				(error "currency parameter exceeds native numeric precision")))
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
		(has? '(40 41 42 43 111) (wire "kind")) (tds_temporal_parameter_value wire value)
		value))))
/* Convert only the external result edge to the existing neutral transport
payload. The shared plan and stored columns retain their normal Unix values.
Floor is required before 1970: truncating a negative partial day changes both
its date and clock. SQL declarations select the descriptor before any rows. */
(define tds_temporal_result_value (lambda (descriptor value) (begin
	(define kind (descriptor "kind"))
	/* A string is an existing text representation of this declared SQL type,
	not a source for choosing its type. Check before the generic Unix adapter
	can discard a fraction; native Unix/date values need no text round trip.
	DATE intentionally discards its clock under the declared DATE contract. */
	(define textual (string? value))
	(define checked (if (and textual (not (equal? kind 40)))
		(tsql_calendar_cast_text value) value))
	(define seconds (unix_timestamp (if (and textual (equal? kind 41))
		(tsql_calendar_text checked) checked)))
	(define day (intdiv (floor (/ seconds 86400)) 1))
	(define clock (- seconds (* day 86400)))
	(define legacy (equal? kind 111))
	(define short (and legacy (equal? (descriptor "size") 4)))
	(if (equal? kind 41) true
		(and (>= seconds (if legacy (if short -2208988800 -6847804800) -62135596800))
			(<= seconds (if short 3453321540 253402300799))) true
		(error "calendar wire result overflow"))
	(if (and short (not (equal? (mod clock 60) 0)))
		(error "small datetime wire result requires whole minutes") true)
	(list "days" (if (equal? kind 41) 0 (+ day (if legacy 25567 719162)))
		"ticks" (if (equal? kind 40) 0 short (intdiv clock 60)
			(* clock (if legacy 300 (nth '(1 10 100 1000 10000 100000 1000000 10000000) (descriptor "scale")))))
		"offset" 0))))
(define tds_result_value (lambda (descriptor value) (if (nil? value) nil
	(match (descriptor "kind")
		40 (tds_temporal_result_value descriptor value)
		41 (tds_temporal_result_value descriptor value)
		42 (tds_temporal_result_value descriptor value)
		111 (tds_temporal_result_value descriptor value)
		165 (list "bytes" value)
		110 (tsql_money_coefficient value (equal? (descriptor "size") 4))
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
		(define contract (tsql_effective_parameter_contract (cadr declaration)))
		(define parameter (named name))
		(define wire (parameter "type"))
		(if (not (has? '(31 38 40 41 42 43 104 106 108 109 110 111 165 167 173 175 231 239) (wire "kind")))
			(error "unsupported T-SQL parameter wire type: " (wire "kind")) true)
		/* A supplied NULL needs no numeric representation or conversion. Its SQL
		contract still comes exclusively from the declaration. */
		(define source_type (match (wire "kind")
			38 "BIGINT" 104 "BIT" 106 "DECIMAL" 108 "NUMERIC" 109 "FLOAT"
			40 "DATE" 41 "TIME" 42 "DATETIME2" 43 "DATETIMEOFFSET"
			111 (if (equal? (wire "size") 4) "SMALLDATETIME" "DATETIME")
			110 (if (tds_money_small? wire) "SMALLMONEY" "MONEY")
			165 "VARBINARY" 173 "BINARY" 167 "VARCHAR" 175 "CHAR" 231 "NVARCHAR" 239 "NCHAR" _ "NULL"))
		(if (and (not (nil? (parameter "value"))) (tsql_binary_type? source_type)
			(not (tsql_binary_type? (car contract))))
			(error "binary parameter conversion is not implemented") true)
		(if (and (not (nil? (parameter "value"))) (tsql_temporal_type? source_type)
			(not (or (tsql_temporal_type? (car contract)) (sql_text_type? (car contract)))))
			(error "temporal parameter conversion is not implemented") true)
		(define converted (if (nil? (parameter "value")) nil
			(eval (sql_type_lower_annotations
				(tsql_cast_expr (list (quote sql_type_expr) (list (quote quote) (tds_parameter_value parameter)) source_type nil)
					(car contract) (cdr contract))))))
		(list (concat "__tsql_parameter:" name) converted)))))
	/* All declarations and values are validated before installing this request. */
	(map values (lambda (item) (session (car item) (cadr item))))
	(session "__tsql_parameter_declarations" declarations)
	true)))

/* Description binds names and expression contracts without physical planning
or execution. A single mutation without OUTPUT has an empty result contract;
its ordinary grammar still validates declarations and input expressions. */
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
	(define import_target (session "tsql_import_target"))
	(define import_source (session "tsql_import_source"))
	(define cache_scope (json_encode (list username declarations import_target import_source)))
	(define parser (lambda (database text policy planning tx) (begin
		(planning "__tsql_parameter_declarations" declarations)
		(planning "tsql_import_target" import_target)
		(planning "tsql_import_source" import_source)
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
(define tsql_scope_session (lambda (parent contracts owner_schema module_settings) (begin
	(define local (newsession))
	(if owner_schema (local "schema" owner_schema) true)
	(local "__tsql_parameter_declarations" contracts)
	(local "__tsql_scope" (newsession))
	(local "__tsql_connection_session" (tsql_connection_session parent))
	/* Existing runtime SET options inherit once per module invocation. Writes
	stay local even for OFF/nil, so return and error unwinding restore the caller
	without a shared-session save/restore race. Ordinary batches delegate these
	options, retaining their connection state across GO and subsequent requests. */
	(define module_keys (if module_settings
		'("__tsql_identity_insert" "__tsql_datefirst" "__tsql_nocount") '()))
	(map module_keys (lambda (key) (local key (parent key))))
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
					(and owner_schema (equal? key "schema"))
					(has? module_keys key)))))))
		'(key) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))
			(has? module_keys key)) (local key) (parent key))
		'(key value) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))
			(has? module_keys key)) (local key value) (parent key value))
		_ (apply parent args))))))
(define tsql_scalar_expression_allowed? (lambda (expr) (match expr
	((symbol quote) _value) true
	/* Parser source locations can wrap a symbol. Match the semantic head,
	not its raw carrier tag, before admitting a standalone scalar formula. */
	(cons (symbol inner_select) _arguments) false
	(cons (symbol inner_select_exists) _arguments) false
	(cons (symbol inner_select_in) _arguments) false
	(cons (symbol get_column) _arguments) false
	(cons (symbol aggregate) _arguments) false
	(cons (symbol window) _arguments) false
	(cons _head tail)
	(reduce tail (lambda (allowed child) (and allowed (tsql_scalar_expression_allowed? child))) true)
	_ true)))
(define tsql_scalar_formula (lambda (schema text contracts mode target tx) (begin
	(define key (json_encode (list schema text contracts mode target)))
	(tsql_scalar_cache key tx (lambda (tx) (begin
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
		(define conversion_target (if (nil? target) nil (tsql_effective_parameter_contract target)))
		(define expr (if (has? '("assignment" "message-text" "message-integer") mode)
			(sql_type_formula '() (tsql_cast_expr (list (quote sql_type_expr) expr (sql_info_type info) (sql_info_collation info))
				(car conversion_target) (cdr conversion_target))) expr))
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
	(parser "%%" '("text" "%"))
	(parser '("%" (define flags (regex "[-+0# ]*" false false))
		(? (define width (regex "(?:\\*|[1-9][0-9]*)" false false)))
		(? "." (define precision (regex "(?:\\*|[0-9]*)" false false)))
		(define marker (regex "(?:I64d|[dis])" false false)))
		(list "format" (match marker "I64d" '("BIGINT") "s" '("NVARCHAR" -1) _ '("INT"))
			flags width precision))) empty)) parts ""))
(define tsql_message_argument (lambda (arguments index schema session tx contract) (begin
	(if (< index (count arguments)) true (error "missing RAISERROR substitution"))
	(tsql_scalar_value schema (nth arguments index) session tx
		(if (equal? (car contract) "NVARCHAR") "message-text" "message-integer") contract))))
(define tsql_message_bound (lambda (token arguments index schema session tx) (begin
	(define value (if (nil? token) nil (equal? token "*")
		(tsql_message_argument arguments index schema session tx '("INT"))
		(equal? token "") 0 (tsql_parse_integer token)))
	(if (and (not (nil? token)) (nil? value)) (error "invalid RAISERROR format bound") true)
	(list value (+ index (if (equal? token "*") 1 0))))))
(define tsql_message_field (lambda (value contract flags raw_width raw_precision) (begin
	(define requested_width (coalesceNil raw_width 0))
	/* Exact truncation also charges hidden substitution storage. Until that
	contract is modeled, refuse oversized fields before allocating their padding. */
	(if (or (> requested_width 2047) (< requested_width -2047))
		(error "RAISERROR field width above 2047 is not implemented") true)
	(define left (or (< requested_width 0) (regexp_test flags "-")))
	(define width (if (< requested_width 0) (- 0 requested_width) requested_width))
	(define precision (if (and (not (nil? raw_precision)) (< raw_precision 0)) nil raw_precision))
	(define text (coalesceNil value "(null)"))
	(define field (if (or (nil? value) (equal? (car contract) "NVARCHAR")) (begin
		(define prefix (if (nil? precision) text (utf16_prefix text precision)))
		(if (and (not (nil? precision)) (< (utf16_length prefix) (min (utf16_length text) precision)))
			(error "RAISERROR precision splitting a supplementary pair is not implemented") true)
		(list "" prefix))
		(begin
			(if (and (not (nil? precision)) (> precision 2047))
				(error "RAISERROR integer precision above 2047 is not implemented") true)
			(define negative (< value 0))
			/* Remove the printed sign instead of negating the signed minimum. */
			(define raw_digits (if negative (substr (string value) 1) (string value)))
			(define digits (if (and (equal? precision 0) (equal? value 0)) "" raw_digits))
			(define sign (if negative "-" (regexp_test flags "\\+") "+" (regexp_test flags " ") " " ""))
			(list sign (concat (string_repeat "0" (max 0 (- (coalesceNil precision 1) (strlen digits)))) digits)))))
	(define sign (car field))
	(define body (cadr field))
	(define units (+ (strlen sign) (utf16_length body)))
	(if (> units 2047) (error "RAISERROR formatted field above 2047 is not implemented") true)
	(define padding (max 0 (- width units)))
	(define zero (and (not left) (nil? precision) (not (nil? value))
		(not (equal? (car contract) "NVARCHAR")) (regexp_test flags "0")))
	(if left (concat sign body (string_repeat " " padding))
		zero (concat sign (string_repeat "0" padding) body)
		(concat (string_repeat " " padding) sign body)))))
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
				'("format" contract flags width precision) (begin
					(define width_value (tsql_message_bound width arguments (cadr result) schema session tx))
					(define precision_value (tsql_message_bound precision arguments (cadr width_value) schema session tx))
					(define index (cadr precision_value))
					(define value (tsql_message_argument arguments index schema session tx contract))
					(list (concat (car result) (tsql_message_field value contract flags
						(car width_value) (car precision_value))) (+ index 1))))) (list "" 3)))
		(if (> (utf16_length (car formatted)) 2047)
			(error "RAISERROR message truncation above 2047 is not implemented") true)
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
		'("assign" name operation text) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) name)) nil))
			(if declaration true (error "undeclared variable: @" name))
			(if operation (tsql_bit_assignment_type (car (cadr declaration))) true)
			/* Compound syntax uses the existing scalar compiler and assignment cast.
			The parsed variable identifier is syntax, never caller-provided SQL text. */
			(define text (if operation (concat "@" name " " operation " (" text ")") text))
			(define value (statement text false (lambda (tx)
				(tsql_scalar_value schema text session tx "assignment" (cadr declaration)))))
			(session (concat "__tsql_parameter:" name) value)
			0)
		'("create-procedure" owner name parameters source replace) (statement "procedure definition" false (lambda (tx) (begin
			/* Stored procedure source is not yet normalized for import aliases or
			checked dependencies. Reject before catalog publication; ordinary
			procedure creation still uses its existing source compiler. */
			(if (session "tsql_import_target")
				(error "retargeted procedure definitions are not supported") true)
			(if (session "transaction") (error "procedure DDL inside explicit transactions is not supported") true)
			(define owner (tsql_qualifier_database owner schema session))
			((sql_policy (session "username")) owner name true)
			(createprocedure owner (toLower name) "tsql-procedure" source parameters replace)
			0)))
		'("drop-procedure" owner name ignore) (statement "procedure deletion" false (lambda (_tx) (begin
			(if (session "transaction") (error "procedure DDL inside explicit transactions is not supported") true)
			(define owner (tsql_qualifier_database owner schema session))
			((sql_policy (session "username")) owner name true)
			(define definition (find (showprocedures owner (toLower name)) (lambda (_definition) true) nil))
			(dropprocedure owner (if definition (definition "name") name) ignore)
			0)))
		'("execute" owner name arguments) (tsql_invoke_procedure owner name arguments statement session resultrow resultfields)
		'("execute-status" owner name arguments target) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) target)) nil))
			(if declaration true (error "undeclared return status variable: @" target))
			(define status (tsql_invoke_procedure owner name arguments statement session resultrow resultfields))
			(define contract (tsql_effective_parameter_contract (cadr declaration)))
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
(define tsql_execute_command_batch (lambda (schema commands declarations supplied statement caller resultrow resultfields module_settings finish) (begin
	(define scoped (tsql_scope_session caller (tsql_batch_contracts commands declarations) schema module_settings))
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
	(define batches (tsql_script_cache (json_encode (list schema text)) nil
		(lambda (_tx) (map (tsql_script_batches text) (lambda (batch)
			(list "syntax" (parse_tsql_batch schema (batch "text")) "repeat" (batch "repeat")))))))
	(reduce batches (lambda (_result batch) (reduce (produceN (batch "repeat")) (lambda (_result _iteration) (begin
		(define commands (cadr (batch "syntax")))
		(tsql_execute_command_batch schema commands inherited supplied statement session resultrow resultfields false
			(lambda (result _status _scoped) result)))) 0)) 0))))
/* Frontend scopes borrow names; the native request still owns the original
connection session and its reusable transaction record. */
(define tsql_execute_request (lambda (schema text session state sequence resultrow resultfields)
	(with_query_request (tsql_connection_session session) state sequence text (lambda (statement)
		(tsql_execute_batches schema text session statement resultrow resultfields)))))
/* Console imports accept the same textual script as REST and the connector.
The caller supplies its ordinary authenticated session; file I/O stays in load. */
/* The two import names belong to one invocation. Other state, notably the
account and existing transaction, remains on the ordinary caller session. */
(define tsql_import_scope (lambda (schema parent) (begin
	(define names (newsession))
	(names "tsql_import_target" schema)
	(names "__tsql_connection_session" (tsql_connection_session parent))
	(define own? (lambda (key) (has? '("tsql_import_target" "tsql_import_source" "__tsql_connection_session") key)))
	(lambda args (match args
		'() (merge_unique (list (names) (filter (parent) (lambda (key) (not (own? key))))))
		'(key) (if (own? key) (names key) (parent key))
		'(key value) (if (own? key) (names key value) (parent key value))
		_ (apply parent args))))))
/* USE declares one source alias during import; it never selects that database.
Literal contents are untouched and all following statements keep the target. */
(define tsql_import_database (lambda (session name) (begin
	(define source (session "tsql_import_source"))
	(if (equal? name (session "tsql_import_target")) 0
		(nil? source) (begin (session "tsql_import_source" name) 0)
		(equal? source name) 0
		(error "import cannot select a second source database: " name)))))
(define load_tsql (lambda (schema stream session) (begin
	((sql_policy (session "username")) schema true false)
	/* An import borrows an existing caller transaction. Otherwise it owns only
	explicit transactions opened by this script, and must release them on failure
	or unfinished input. Completed autocommit statements retain their commits. */
	(define caller_transaction (session "transaction"))
	(define outcome (try (lambda ()
		(list true (tsql_execute_request schema (load stream nil nil "bom") (tsql_import_scope schema session) nil 0 print (lambda args true))))
		(lambda (failure) (list false failure))))
	(define unfinished (and (nil? caller_transaction) (session "transaction")))
	/* Cleanup runs once, outside the request's catch: a persistence failure
	during rollback must propagate rather than retry the same native cleanup. */
	(if unfinished (tx_rollback session) true)
	(if (not (car outcome)) (error (cadr outcome))
		unfinished (error "unfinished import transaction") (cadr outcome)))))
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
				(list (quote tsql_scope_session) (quote caller) (list (quote quote) contracts) schema true))
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
	(define syntax (tsql_script_cache (json_encode (list schema text "dynamic")) nil
		(lambda (_tx) (parse_tsql_batch schema text))))
	(tsql_execute_command_batch schema (cadr syntax) declarations values statement caller resultrow resultfields true finish))))
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
(define tsql_invoke_procedure (lambda (schema name arguments statement caller resultrow resultfields) (begin
	(define metadata_call (has? '("sp_describe_undeclared_parameters" "sp_tables" "sp_columns" "sp_pkeys" "sp_statistics" "sp_fkeys" "sp_datatype_info" "sp_datatype_info_100" "sp_special_columns") (toLower name)))
	/* sys is a virtual owner only for supported system calls. Other owners
	use the same import binding as ordinary SQL, before any invocation runs. */
	(define schema (if (and (equal?? schema "sys")
		(or metadata_call (equal? (toLower name) "sp_executesql"))) (caller "schema")
		(tsql_qualifier_database schema (caller "schema") caller)))
	(if (equal? (toLower name) "sp_executesql")
		(tsql_execute_parameterized (if (equal?? schema "sys") (caller "schema") schema)
			arguments statement caller resultrow resultfields)
		metadata_call
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
				(lambda (status scoped) (begin (tsql_copy_outputs (cadr binding) scoped statement caller) status))))))))

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
				(session "tsql_import_target" nil)
				(session "tsql_import_source" nil)
				/* One scope owner per submitted batch, never per native insert row. */
				(session "__tsql_scope" (newsession))
				(define output (newsession))
				(define result (tsql_execute_request schema text session (req "__session_state") (req "__query_seq")
					(lambda (values) (begin (output "rows" true) ((res "jsonl") values)))
					(lambda args (output "fields" true))))
				/* An empty result set still has a declared field contract. Its
				procedure return status must not become an affected-row record. */
				(if (and (not (or (output "rows") (output "fields"))) (number? result))
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
/* View definitions stay in the existing system catalog. This transient answer
joins object names only; native key procedures keep enumerating native tables. */
(define tsql_metadata_relations (lambda (schema pattern patterns policy) (begin
	(define native (map (tsql_metadata_table_names schema pattern patterns policy)
		(lambda (name) (list name "TABLE"))))
	(define views (if (> (sql_view_catalog_state "count") 0)
		(scan nil (table "system" false "views" false)
			(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false))
			(list schema) '("name") (lambda (name) (tsql_metadata_match name pattern patterns))
			'("name") (lambda (acc name) (cons (list name "VIEW") acc))
			'() (lambda (a b) (merge (list a b)))) '()))
	(sort (merge (list native views)) (lambda (left right)
		(if (equal? (cadr left) (cadr right)) (< (car left) (car right)) (< (cadr left) (cadr right))))))))
/* Describe a logical view projection, never its rows. Native declarations can
supply direct projection nullability and dimensions, but not a view default,
identity, key or comment. Unknown expression widths must not become width one. */
(define tsql_metadata_view_columns (lambda (schema name policy planning_session) (begin
	(define query (tsql_describe_query
		(make_query_block schema (list (list name schema name false nil))
			(list "*" (list (quote get_column) nil false "*" false))
			nil nil nil nil nil nil '() '() '()) policy planning_session))
	(define sources (qb_sources query))
	(map (tsql_result_entries query true) (lambda (entry) (begin
		(define expression (nth entry 1))
		(define info (nth entry 2))
		(define origin (nth entry 3))
		(define declaration (if origin (tsql_effective_column (cadr origin) planning_session) nil))
		(define type (sql_info_type info))
		(define dimensions (coalesceNil (tsql_result_dimensions sources expression '())
			(tsql_result_dimensions sources (sql_info_formula info) '()) '()))
		(if (and (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "BINARY" "VARBINARY" "DECIMAL" "NUMERIC") type)
			(or (empty_list? dimensions)
				(not (reduce dimensions (lambda (known dimension) (and known (not (nil? dimension)))) true))))
			(error "view expression declaration dimensions are unavailable") true)
		(list "Field" (car entry) "RawType" type "Dimensions" dimensions
			"DeclaredType" (if declaration (declaration "DeclaredType") nil)
			"Null" (if origin (or (source_outer? (car origin)) ((cadr origin) "Null")) true)
			"Comment" "" "DefaultExpression" "" "Default" nil)))))))
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
(define tsql_metadata_constraints (lambda (schema name) (show (table schema false name false) "constraints")))
(define tsql_metadata_key_name (lambda (key)
	(if (equal? (key "constraint_name") "") (key "Id") (key "constraint_name"))))
(define tsql_metadata_fk_rule (lambda (mode) (match mode "cascade" 0 "set null" 2 "set default" 3 _ 1)))
(define tsql_metadata_descriptors (lambda (names integers)
	(map names (lambda (name)
		(if (has? integers name) (list "name" name "kind" 38 "size" 4 "flags" 1)
			(list "name" name "kind" 231 "size" 8000 "flags" 1))))))
/* (type name, ODBC code, precision, transfer length, scale, radix, datetime subtype).
Transfer length describes the driver's default C representation, not compressed
or physical column storage. Only declarations decide these values. */
(define tsql_metadata_type (lambda (column version) (begin
	(define dimensions (column "Dimensions"))
	(define width (if (empty_list? dimensions) 1 (car dimensions)))
	/* The declaration's MAX sentinel is not an ODBC length. */
	(define length (if (equal? width -1) 0 width))
	(define type (toLower (column "RawType")))
	(define contract (match type
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
		"money" '("money" 3 19 21 4 10 nil)
		"smallmoney" '("smallmoney" 3 10 12 4 10 nil)
		"uniqueidentifier" '("uniqueidentifier" -11 36 16 nil nil nil)
		"text" '("text" -1 2147483647 2147483647 nil nil nil)
		"ntext" '("ntext" -10 1073741823 2147483646 nil nil nil)
		"image" '("image" -4 2147483647 2147483647 nil nil nil)
		"date" (list "date" (if (equal? version 2) 9 91) 10 6 0 nil 1)
		"datetime" (list "datetime" (if (equal? version 2) 11 93) 23 16 3 nil 3)
		"smalldatetime" (list "smalldatetime" (if (equal? version 2) 11 93) 16 16 0 nil 3)
		"rowversion" '("rowversion" -2 8 8 nil nil nil)
		"timestamp" (begin
			(define scale (if (empty_list? dimensions) 6 (car dimensions)))
			(list "datetime2" (if (equal? version 2) 11 93) (if (> scale 0) (+ 20 scale) 19) 16 scale nil 3))
		_ (if (has? '("time" "datetime2" "datetimeoffset") type)
			(begin
				(define scale (if (empty_list? dimensions) 7 (car dimensions)))
				(define code (match type "time" -154 "datetimeoffset" -155 _ (if (equal? version 2) 11 93)))
				(define precision (+ (match type "time" 8 "datetimeoffset" 26 _ 19) (if (> scale 0) (+ scale 1) 0)))
				(list type code precision (match type "time" 10 "datetimeoffset" 20 _ 16) scale nil
					(if (equal? type "datetime2") 3 nil)))
			(has? '("decimal" "numeric") type)
			(begin
				(define precision (if (empty_list? dimensions) 18 (car dimensions)))
				(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
				(list type (if (equal? type "decimal") 3 2) precision (+ precision 2) scale 10 nil))
			(has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary") type)
			(list type (match type "varchar" 12 "char" 1 "nvarchar" -9 "nchar" -8 "binary" -2 "varbinary" -3)
				length (if (and (> length 0) (has? '("nvarchar" "nchar") type)) (* length 2) length) nil nil nil)
			(error "unsupported metadata declaration type: " type))))
	(define declared (column "DeclaredType"))
	(if (nil? declared) contract
		(cons (if (tsql_builtin_type_declaration declared) "sysname" (cadr (tsql_alias_name_parts declared))) (cdr contract))))))
(define tsql_metadata_column_row (lambda (schema name column ordinal version) (begin
	(define contract (tsql_metadata_type column version))
	(define subtype (nth contract 6))
	(define type (toLower (column "RawType")))
	(list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name
		"COLUMN_NAME" (column "Field") "DATA_TYPE" (nth contract 1) "TYPE_NAME" (car contract)
		"PRECISION" (nth contract 2) "LENGTH" (nth contract 3) "SCALE" (nth contract 4)
		"RADIX" (nth contract 5) "NULLABLE" (if (column "Null") 1 0)
		"REMARKS" (if (equal? (column "Comment") "") nil (column "Comment"))
		"COLUMN_DEF" (if (equal? (column "DefaultExpression") "") (column "Default") (column "DefaultExpression"))
		"SQL_DATA_TYPE" (if subtype 9 (nth contract 1)) "SQL_DATETIME_SUB" subtype
		"CHAR_OCTET_LENGTH" (if (has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary" "text" "ntext" "image") type) (nth contract 3) nil)
		"ORDINAL_POSITION" ordinal "IS_NULLABLE" (if (column "Null") "YES" "NO")))))
/* Type info is a frontend declaration catalog, not a registry of runtime tags. */
(define tsql_metadata_type_info (lambda (options version) (begin
	(define definitions (list (list "bigint" '()) (list "int" '()) (list "smallint" '()) (list "tinyint" '())
		(list "bit" '()) (list "real" '()) (list "float" '()) (list "decimal" '(38 0)) (list "numeric" '(38 0))
		(list "money" '()) (list "smallmoney" '()) (list "uniqueidentifier" '())
		(list "varchar" '(8000)) (list "char" '(8000)) (list "nvarchar" '(4000)) (list "nchar" '(4000))
		(list "binary" '(8000)) (list "varbinary" '(8000))
		(list "text" '()) (list "ntext" '()) (list "image" '())
		(list "date" '()) (list "datetime" '()) (list "smalldatetime" '())
		(list "datetime2" '(7)) (list "time" '(7)) (list "datetimeoffset" '(7))))
	(define rows (map definitions (lambda (definition) (begin
		(define contract (tsql_metadata_type (list "RawType" (car definition) "Dimensions" (cadr definition)) version))
		(define type (car contract))
		(define character (has? '("varchar" "char" "nvarchar" "nchar" "text" "ntext") type))
		(define binary (has? '("binary" "varbinary" "image") type))
		(define long_value (has? '("text" "ntext" "image") type))
		(define calendar (has? '("date" "datetime" "smalldatetime" "datetime2" "time" "datetimeoffset") type))
		(define scaled_calendar (has? '("datetime2" "time" "datetimeoffset") type))
		(define decimal (has? '("decimal" "numeric") type))
		(define money (has? '("money" "smallmoney") type))
		(define numeric (or decimal money (has? '("bigint" "int" "smallint" "tinyint" "real" "float") type)))
		(define quoted (or character calendar (equal? type "uniqueidentifier")))
		(list "TYPE_NAME" type "DATA_TYPE" (nth contract 1) "PRECISION" (nth contract 2)
			"LITERAL_PREFIX" (if quoted "'" binary "0x" nil)
			"LITERAL_SUFFIX" (if quoted "'" nil)
			"CREATE_PARAMS" (if (and (or character binary) (not long_value)) "length" decimal "precision,scale" scaled_calendar "scale" nil)
			"NULLABLE" 1 "CASE_SENSITIVE" (if character 1 0)
			/* ODBC predicate capabilities: none=0, LIKE-only=1, full=3. */
			"SEARCHABLE" (match type "image" 0 "text" 1 "ntext" 1 _ 3)
			"UNSIGNED_ATTRIBUTE" (if (equal? type "tinyint") 1 numeric 0 nil)
			"MONEY" (if money 1 0) "AUTO_INCREMENT" (if (has? '("bigint" "int" "smallint" "tinyint") type) 1 numeric 0 nil)
			"LOCAL_TYPE_NAME" type "MINIMUM_SCALE" (if scaled_calendar 0 (nth contract 4))
			"MAXIMUM_SCALE" (if decimal 38 scaled_calendar 7 (nth contract 4))
			"SQL_DATA_TYPE" (if (nth contract 6) 9 (nth contract 1)) "SQL_DATETIME_SUB" (nth contract 6)
			"NUM_PREC_RADIX" (nth contract 5) "INTERVAL_PRECISION" nil)))))
	(list '("TYPE_NAME" "DATA_TYPE" "PRECISION" "LITERAL_PREFIX" "LITERAL_SUFFIX" "CREATE_PARAMS" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "LOCAL_TYPE_NAME" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		'("DATA_TYPE" "PRECISION" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		(sort (filter rows (lambda (row) (or (nil? (options "data_type")) (equal? (options "data_type") 0) (equal? (row "DATA_TYPE") (options "data_type")))))
			(lambda (left right) (< (left "DATA_TYPE") (right "DATA_TYPE"))))))))
/* Undeclared parameters are parsed by the ordinary grammar, then inferred
from its bound declaration contracts. Probe markers are analysis-only data:
no supplied value, physical plan, cache binding or marker is executed. */
(define tsql_parameter_probe (lambda (expression) (match expression
	((symbol sql_type_expr) value _type _collation) (tsql_parameter_probe value)
	((symbol tsql_unbound_parameter) name spelling) (list name spelling)
	_ nil)))
(define tsql_parameter_cast_head? (lambda (head)
	(or (has? (list (quote tsql_cast_text) (quote tsql_cast_fixed_text) (quote tsql_cast_binary)
		(quote tsql_cast_decimal) (quote tsql_cast_integer) (quote tsql_cast_float)
		(quote tsql_cast_bit)) head)
		(expression_equal? head tsql_cast_integer) (expression_equal? head tsql_cast_float)
		(expression_equal? head tsql_cast_bit))))
(define tsql_parameter_analysis_join (lambda (left right)
	(list (merge (list (car left) (car right))) (merge (list (cadr left) (cadr right))))))
(define tsql_parameter_candidates (lambda (node sources declarations) (match node
	((symbol quote) _datum) (list '() '())
	((symbol tsql_unbound_parameter) name spelling) (list (list (list name spelling)) '())
	(cons _head _tail) (begin
		(define scope (if (query_block? node) (merge (list (qb_sources node) sources)) sources))
		(define own (match node
			((symbol sql_type_expr) formula type _collation) (match formula
				(cons head (cons operand _arguments)) (begin
					(define parameter (tsql_parameter_probe operand))
					(if (and parameter (tsql_parameter_cast_head? head))
						(list (list (car parameter) (cadr parameter) type
							(coalesceNil (tsql_result_dimensions scope formula declarations) '()))) '()))
				_ '())
			'(head left right) (if (has? (list (quote equal?) (quote equal??)) head) (begin
				(define parameter (coalesceNil (tsql_parameter_probe left) (tsql_parameter_probe right)))
				(define known (if (tsql_parameter_probe left) right left))
				(define info (sql_expr_info scope known))
				(if (and parameter (not (has? '("any" "NULL") (sql_info_type info))))
					(list (list (car parameter) (cadr parameter) (sql_info_type info)
						(coalesceNil (tsql_result_dimensions scope known declarations) '()))) '())) '())
			_ '()))
		(reduce (map node (lambda (item) (tsql_parameter_candidates item scope declarations)))
			tsql_parameter_analysis_join (list '() own)))
	_ (list '() '()))))
(define tsql_parameter_type_name (lambda (type dimensions)
	(concat (toLower type) (if (empty_list? dimensions) ""
		(concat "(" (reduce dimensions (lambda (text value)
			(concat text (if (equal? text "") "" ",") (if (equal? value -1) "max" (string value)))) "") ")")))))
/* Input parameter metadata follows the declared protocol type and width.
The result text adapter deliberately serves a different output contract.
FLOAT(p) casts do not yet retain their precision here; temporal and alias
parameter contracts also need their binder before they can be advertised. */
(define tsql_parameter_transport (lambda (column) (begin
	(define contract (tsql_effective_parameter_contract (cons (column "RawType") (column "Dimensions"))))
	(define column (set_assoc (set_assoc column "RawType" (car contract)) "Dimensions" (cdr contract)))
	(define type (column "RawType"))
	(define ids (match type
		"BIGINT" '(127 38) "INT" '(56 38) "INTEGER" '(56 38)
		"SMALLINT" '(52 38) "TINYINT" '(48 38) "BIT" '(104 104)
		"REAL" '(59 109)
		"DECIMAL" '(106 106) "NUMERIC" '(108 108)
		"MONEY" '(60 110) "SMALLMONEY" '(122 110)
		"CHAR" '(175 175) "VARCHAR" '(167 167)
		"NCHAR" '(239 239) "NVARCHAR" '(231 231)
		"BINARY" '(173 173) "VARBINARY" '(165 165)
		_ (error "unsupported parameter transport contract: " type)))
	(define length (tsql_declared_byte_length column))
	(list "system_type" (car ids) "kind" (cadr ids)
		"size" (if (equal? length -1) 65535 length)))))
(define tsql_parameter_description (lambda (specification ordinal) (match specification
	'(name spelling type dimensions) (begin
		(define dimensions (tsql_decl_dimensions type dimensions))
		(define column (list "RawType" type "Dimensions" dimensions))
		(define contract (tsql_metadata_type column 3))
		(define wire (tsql_parameter_transport column))
		(list "parameter_ordinal" ordinal "name" (concat "@" spelling)
			"suggested_system_type_id" (wire "system_type")
			"suggested_system_type_name" (tsql_parameter_type_name type dimensions)
			"suggested_max_length" (tsql_declared_byte_length column)
			"suggested_precision" (if (or (sql_numeric_type? type) (has? '("MONEY" "SMALLMONEY") type)) (nth contract 2) 0)
			"suggested_scale" (coalesceNil (nth contract 4) 0)
			"suggested_user_type_id" nil "suggested_user_type_database" nil
			"suggested_user_type_schema" nil "suggested_user_type_name" nil
			"suggested_assembly_qualified_type_name" nil "suggested_xml_collection_id" nil
			"suggested_xml_collection_database" nil "suggested_xml_collection_schema" nil
			"suggested_xml_collection_name" nil "suggested_is_xml_document" 0
			"suggested_is_case_sensitive" 0 "suggested_is_fixed_length_clr_type" 0
			"suggested_is_input" 1 "suggested_is_output" 0 "formal_parameter_name" nil
			"suggested_tds_type_id" (wire "kind") "suggested_tds_length" (wire "size"))))))
(define tsql_describe_parameters (lambda (schema text parameter_text policy) (begin
	(define declarations (tds_bind_declarations schema (coalesceNil parameter_text "")))
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" declarations)
	(planning "__tsql_parameter_analysis" true)
	(planning "__tsql_describe" true)
	(define query (match (parse_tsql schema text policy planning nil)
		((symbol quote) query) query _ (error "parameter description requires a single SELECT")))
	(define analysis (tsql_parameter_candidates query '() declarations))
	(define unknowns (reduce (car analysis) (lambda (entries parameter)
		(if (find entries (lambda (entry) (equal? (car entry) (car parameter)))) entries
			(append entries parameter))) '()))
	(define specifications (map unknowns (lambda (parameter) (begin
		(define candidates (filter (cadr analysis) (lambda (entry) (equal? (car entry) (car parameter)))))
		(if (empty_list? candidates) (error "cannot infer parameter: @" (cadr parameter)) true)
		(define first (car candidates))
		(if (reduce candidates (lambda (same entry)
			(and same (equal? (nth first 2) (nth entry 2))
				(expression_equal? (nth first 3) (nth entry 3)))) true) true
			(error "conflicting parameter declarations: @" (cadr parameter)))
		(list (car parameter) (cadr parameter) (nth first 2) (nth first 3))))))
	/* Re-run the ordinary pure description with the inferred declarations.
	No probe can reach the expression compiler used for execution. */
	(define validation (newsession))
	(validation "__tsql_describe" true)
	(validation "__tsql_parameter_declarations" (merge (list declarations
		(map specifications (lambda (entry) (list (car entry) (cons (nth entry 2) (nth entry 3))))))))
	(define checked (parse_tsql schema text policy validation nil))
	(if (list? checked)
		(map (produceN (count specifications)) (lambda (i) (tsql_parameter_description (nth specifications i) (+ i 1))))
		(error "invalid inferred parameter statement")))))

(define tsql_metadata_result (lambda (schema procedure arguments policy) (begin
	(define procedure (toLower procedure))
	(define procedure (if (and (>= (strlen procedure) 4) (equal? (substr procedure 0 4) "sys.")) (substr procedure 4) procedure))
	(define names (match procedure
		"sp_describe_undeclared_parameters" '("tsql" "params")
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
	/* One metadata request owns its alias factory; neither connection sessions
	nor later requests retain resolved or negatively cached declarations. */
	(define planning_session (newsession))
	(define result (match procedure
		"sp_describe_undeclared_parameters" (begin
			(if (nil? (options "tsql")) (error "parameter description requires statement text") true)
			(define rows (tsql_describe_parameters schema (options "tsql") (options "params") policy))
			(list '("parameter_ordinal" "name" "suggested_system_type_id" "suggested_system_type_name"
				"suggested_max_length" "suggested_precision" "suggested_scale" "suggested_user_type_id"
				"suggested_user_type_database" "suggested_user_type_schema" "suggested_user_type_name"
				"suggested_assembly_qualified_type_name" "suggested_xml_collection_id" "suggested_xml_collection_database"
				"suggested_xml_collection_schema" "suggested_xml_collection_name" "suggested_is_xml_document"
				"suggested_is_case_sensitive" "suggested_is_fixed_length_clr_type" "suggested_is_input" "suggested_is_output"
				"formal_parameter_name" "suggested_tds_type_id" "suggested_tds_length")
				'("parameter_ordinal" "suggested_system_type_id" "suggested_max_length" "suggested_precision" "suggested_scale"
					"suggested_user_type_id" "suggested_xml_collection_id" "suggested_is_xml_document" "suggested_is_case_sensitive"
					"suggested_is_fixed_length_clr_type" "suggested_is_input" "suggested_is_output" "suggested_tds_type_id" "suggested_tds_length") rows))
		"sp_datatype_info" (tsql_metadata_type_info options version)
		"sp_datatype_info_100" (tsql_metadata_type_info options version)
		"sp_tables" (begin
			(define filter_type (options "table_type"))
			(define types (if (nil? filter_type) '("TABLE" "VIEW")
				(map (split filter_type ",") (lambda (value) (toUpper (replace (strtrim value) "'" ""))))))
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(map (filter (tsql_metadata_relations schema (options "table_name")
					(not (equal? (coalesceNil (options "fusepattern") 1) 0)) policy)
					(lambda (relation) (has? types (cadr relation))))
					(lambda (relation) (list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo"
						"TABLE_NAME" (car relation) "TABLE_TYPE" (cadr relation) "REMARKS" nil))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "TABLE_TYPE" "REMARKS") '() rows))
		"sp_columns" (begin
			(if (nil? (options "table_name")) (error "sp_columns requires a table name") true)
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(merge (map (tsql_metadata_relations schema (options "table_name") true policy) (lambda (relation) (begin
					(define name (car relation))
					(define columns (map (if (equal? (cadr relation) "VIEW")
						(tsql_metadata_view_columns schema name policy planning_session)
						(filter (show (table schema false name false)) (lambda (column) (not (internal_column_name? (column "Field"))))))
						(lambda (column) (tsql_effective_column column planning_session))))
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
				(define columns (map (show (table schema false name false))
					(lambda (column) (tsql_effective_column column planning_session))))
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
							(define referenced (table schema false (key "OtherTable") false))
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
