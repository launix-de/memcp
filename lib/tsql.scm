/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-parser.scm")
(import "tsql-transaction-options.scm")
/* Session encoding is announced before the client serializes ANSI parameters. */
(define tsql_session_collation (hex2bin "0904d00034"))
(define tsql_queryplan_cache (newcachemap))
(define tsql_statement_text strtrim)
/* A regular server setting supplies new connection defaults. Session SET never
changes this startup profile; modules inherit and restore their caller options. */
(define tsql_login_language (car (tsql_language_profile (arg "tds-language" "us_english"))))
(define tsql_initialize_date_context (lambda (session) (begin
	(if (nil? (session "__tsql_language")) (tsql_set_language session tsql_login_language) true)
	session)))

/* Database selection is execution state. Resolve and authorize before changing
it; compilation and failed selections never publish a new connection context. */
(define tsql_use_database (lambda (session name) (begin
	(define database (database_name name true))
	(if (nil? database) (error "database does not exist: " name) true)
	(tsql_require_database_access (tsql_session_actor session) database)
	(session "schema" database)
	0)))

/* Database access probes use the authenticated invocation identity and
the native account and database membership. Keep the probe at execution
time so cached statements observe grants and revocations. */
(define tsql_has_dbaccess (close_procedure ((lambda (permission)
	(lambda (name username)
		(if (nil? name) nil (begin
			(define database (database_name name true))
			(if (nil? database) nil (if (permission username database "CONNECT") 1 0)))))) tsql_permission?)))

/* Compatibility database identifiers are durable frontend data, not a
second database directory. Membership always comes from native (show).
A unique native key serializes concurrent first use across cluster peers;
ordinary reads reuse the assigned identity without catalog writes. */
(eval (parse_sql "system" "CREATE TABLE IF NOT EXISTS tsql_database_ids (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) COLLATE utf8mb4_bin NOT NULL UNIQUE) ENGINE=SAFE AUTO_INCREMENT=5"
	(lambda (_schema _table _write) true)))
(eval (parse_sql "system" "CREATE TABLE IF NOT EXISTS tsql_database_options (database VARCHAR(255) COLLATE utf8mb4_bin PRIMARY KEY, trustworthy BOOLEAN NOT NULL DEFAULT FALSE) ENGINE=SAFE"
	(lambda (_schema _table _write) true)))
(tsql_initialize_object_ids)
(define tsql_database_id (lambda (name) (begin
	(define identities (table "system" false "tsql_database_ids" false nil))
	(define lookup (lambda () (scan_lookup nil identities
		(list 372734710317056 (scan_boundary "equal" "name" 0 0 true true "" false) "id") (list name))))
	(define existing (lookup))
	(if existing
		(if (and (> existing 0) (<= existing 2147483647)) existing
			(error "database identity exceeds the T-SQL INT domain"))
		(begin
			/* master has its conventional identity when present; no fictitious
			tempdb, model or msdb rows are introduced into the native directory. */
			(if (equal? name "master")
				(insert identities '("id" "name") (list (list 1 name)) '() (lambda () true))
				(insert identities '("name") (list (list name)) '() (lambda () true)))
			(define identity (lookup))
			(if (and identity (> identity 0) (<= identity 2147483647)) identity
				(error "database identity exceeds the T-SQL INT domain")))))))
/* DB_ID uses the same durable identities as sys.databases, after the actual
invocation's database visibility check. NULL is distinct from an omitted name.
SQL Server 2022 describes SMALLINT, despite the documentation's INT wording:
https://dbfiddle.uk/aOmrn4iH . No native database or host identity is exposed. */
(define tsql_db_id_value (close_procedure ((lambda (identifier visible raise_error)
	(lambda (name actor) (if (nil? name) nil (begin
		(define database (database_name name true))
		(if (if database (visible actor database false) false) (begin
			(define id (identifier database))
			(if (> id 32767) (raise_error 8115 "database identity exceeds the T-SQL SMALLINT domain" 1) id))
			nil))))) tsql_database_id tsql_metadata_visible? tsql_raise_error)))
/* Native server-role profile: admin accounts are sysadmin; every account is
public. The other fixed roles have no native members. Unknown roles/logins
and membership hidden from another nonadministrator yield SQL NULL.
Read account state at execution time, never at query compilation. */
(define tsql_is_srvrolemember (lambda (role login caller)
	(if (or (nil? role) (nil? login) (nil? caller)) nil (begin
		(define normalized (toLower role))
		(if (not (has? '("sysadmin" "public" "serveradmin" "securityadmin" "processadmin" "setupadmin" "bulkadmin" "diskadmin" "dbcreator") normalized)) nil (begin
			(define users (table "system" false "user" false nil))
			(define boundary (scan_boundary "equal" "username" 0 0 true true "" false))
			(define caller_admin (scan_lookup nil users (list 372734710317056 boundary "admin") (list caller)))
			(if (not (or (equal? caller login) caller_admin)) nil
				(nil? (scan_lookup nil users (list 372734710317056 boundary "username") (list login))) nil
				(equal? normalized "public") 1
				(equal? normalized "sysadmin")
				(if (scan_lookup nil users (list 372734710317056 boundary "admin") (list login)) 1 0)
				0)))))))

/* Native database membership projects to dbo ownership and public membership.
Other fixed roles are not separate native memberships. Read the current actor
and database at invocation time so cached queries cannot retain revoked rights. */
(define tsql_is_member (close_procedure ((lambda (permission)
	(lambda (role database actor)
		(if (nil? role) nil (begin
			(define name (toLower role))
			(if (not (has? '("db_owner" "public" "db_accessadmin" "db_securityadmin"
				"db_ddladmin" "db_backupoperator" "db_datareader" "db_datawriter"
				"db_denydatareader" "db_denydatawriter") name)) nil
				(not (permission actor database "DATABASE ACCESS")) 0
				(has? '("db_owner" "public") name) 1
				0))))) tsql_permission?)))

/* Alias declarations are ordinary, durable frontend data, like SQL views.
The native column keeps the qualified alias spelling in its existing RawType.
Only this frontend interprets the declaration; storage has no alias registry. */
(if (has? (show "system") "tsql_types") true
	(eval (parse_sql "system" "CREATE TABLE tsql_types (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, `database` VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, base_type VARCHAR(255) NOT NULL, dimensions TEXT NOT NULL, nullable BOOLEAN NOT NULL, UNIQUE KEY alias_name (`database`,name)) ENGINE=SAFE"
		(lambda (_schema _table _write) true))))

/* Alias DDL and database lifecycle share the common SQL DDL gate. Reads and
DML never acquire it; the database must still exist once ownership is held. */
(define tsql_type_ddl (lambda (database tx session operation) (begin
	/* Catalog-row writes and native column publication do not yet share a
	transactional DDL contract. Refuse explicit-transaction DDL before either one
	changes, matching the frontend's source-procedure DDL restriction. */
	(if (session "transaction") (error "alias DDL inside explicit transactions is not supported") true)
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
	(scan nil (table "system" false "tsql_types" false nil)
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
		/* Cache entries retain their cache owner and its factory. This factory
		needs only its key; do not retain the compiler session through its frame. */
		(lambda () (newcachemap (close_procedure (lambda (key)
			(match (json_decode key) '(database name) (tsql_read_type_alias database name)))))))))
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
		(define column (set_assoc column "IsRowGuidCol" (tsql_row_guid_column? column)))
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
		(if (nil? alias)
			(sql_info (sql_info_formula info)
				(if (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "BINARY" "VARBINARY" "FLOAT" "TIME" "DATETIME2" "DATETIMEOFFSET") (sql_info_type info))
					(list (sql_info_type info) (coalesceNil (declaration "Dimensions") '())) (sql_info_datatype info))
				(sql_info_collation info)) (begin
				(if (alias "id") (tsql_guard_type_alias (declaration "RawType") (alias "id") planning_session) true)
				(sql_info (sql_info_formula info)
					(if (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "BINARY" "VARBINARY" "FLOAT" "TIME" "DATETIME2" "DATETIMEOFFSET") (alias "base_type"))
						(list (alias "base_type") (alias "dimensions")) (alias "base_type"))
					(if (sql_text_type? (alias "base_type"))
						(list (coalesce (declaration "Collation") "utf8mb4_general_ci") 2) nil)))))))))
(define tsql_compile_column (tsql_column_compiler nil))
/* Logical records contain executable operands alongside declaration/fact
containers. All T-SQL expression walkers use this boundary rather than treating
source metadata as an expression list. A source is visited only in its typed
owner slot; arbitrary six-element expressions are never inferred as sources. */
(define tsql_map_source_expressions (lambda (source visit)
	(source_with_join_expr (source_with_relation source (visit (source_relation source)))
		(visit (source_join_expr source)))))
(define tsql_map_executable_children (lambda (node visit) (match node
	((symbol quote) _datum) node
	((symbol literal-rows) _columns _rows) node
	((symbol local-relation) _name _columns) node
	((symbol stage-output) _id) node
	((symbol sql_type_expr) expression type collation)
	(list 'sql_type_expr (visit expression) type collation)
	((symbol sql_compile_column) compiler expression)
	(list 'sql_compile_column compiler (visit expression))
	((symbol query-block) database sources fields condition grouping having ordering limit offset hidden stages facts)
	(make_query_block database (map sources (lambda (source) (tsql_map_source_expressions source visit)))
		(map_assoc fields (lambda (_title expression) (visit expression)))
		(visit condition) (visit grouping) (visit having)
		(map ordering (lambda (item) (list (visit (car item)) (cadr item))))
		(visit limit) (visit offset) (map_assoc hidden (lambda (_title expression) (visit expression)))
		(map stages visit) facts)
	((symbol union-block) mode branches ordering limit offset facts)
	(make_union_block mode (map branches visit)
		(map ordering (lambda (item) (list (visit (car item)) (cadr item))))
		(visit limit) (visit offset) facts)
	((symbol group-stage) id input domain keys aggregates having output ordering limit offset facts)
	(make_group_stage id
		(if (or (query_block? input) (union_block? input)) (visit input)
			(tsql_map_source_expressions input visit))
		(visit domain) (visit keys) (visit aggregates) (visit having) (visit output)
		(map ordering (lambda (item) (list (visit (car item)) (cadr item))))
		(visit limit) (visit offset)
		(if (nil? (qassoc_get facts 'iteration_predicate nil)) facts
			(qassoc_set facts 'iteration_predicate (visit (qassoc_get facts 'iteration_predicate nil)))))
	((symbol window-stage) id source column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit facts)
	(make_window_stage id (tsql_map_source_expressions source visit) column sortcols sortdirs partitioncount
		mapcols (visit mapfn) (visit reducefn) (visit reduceinit) facts)
	((symbol orc-stage) id source column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit)
	(make_orc_stage id (tsql_map_source_expressions source visit) column sortcols sortdirs partitioncount
		mapcols (visit mapfn) (visit reducefn) (visit reduceinit))
	(cons head tail) (cons (visit head) (map tail visit))
	_ node)))
(define tsql_bind_column_compilers (lambda (node compiler) (match node
	((symbol sql_compile_column) _previous expression) (list (quote sql_compile_column) compiler expression)
	_ (tsql_map_executable_children node (lambda (child) (tsql_bind_column_compilers child compiler))))))
/* OFF plans use the same pure declared computation with explicit base inputs.
Native readers load computed columns before projection, so an invocation IF
around a cached get_column cannot own an arithmetic-error boundary. Bind names
first and specialize this finite profile at the frontend cache boundary. */
(define tsql_expand_computed_columns (lambda (node outer planning tx) (begin
	(define visit (lambda (expression sources)
		(tsql_expand_computed_columns expression sources planning tx)))
	(match node
		((symbol quote) _value) node
		((symbol query-block) database sources fields condition group having order limit offset hidden stages facts) (begin
			(define scope (merge (list sources outer)))
			(make_query_block database
				(map sources (lambda (source) (tsql_map_source_expressions source
					(lambda (expression) (visit expression scope)))))
				(visit fields scope) (visit condition scope) (visit group scope) (visit having scope)
				(map order (lambda (item) (list (visit (car item) scope) (cadr item))))
				(visit limit scope) (visit offset scope) (visit hidden scope) (visit stages scope) facts))
		((symbol get_column) alias alias_ic column column_ic) (begin
			(define source (source_for_alias outer (if (empty_list? outer) nil (source_alias (car outer))) alias alias_ic))
			(define declaration (if (and source (source_is_base_table? source))
				(source_column_declaration source column column_ic) nil))
			(if (not (and declaration (declaration "Computed"))) node (begin
				(define handle (table (source_schema source) false (source_storage_relation source) false tx))
				(define recipe (show handle "computation" (sql_column_storage_name declaration) false tx))
				(if (nil? recipe) node (begin
					/* Declarations are immutable until replacement of the relation.
					The native identity is an O(1) scalar guard on this borrowed recipe. */
					(planner_record_guard_condition (list 'equal?
						(list 'show (source_native_table_expr source 'tx) "identity" 'tx)
						(show handle "identity" tx)) planning)
					(define inputs (map (car recipe) (lambda (name) (begin
						(define input (find (source_columns source) (lambda (column)
							(equal? (sql_column_storage_name column) name)) nil))
						(if input true (error "computed physical input declaration disappeared: " name))
						(visit (list 'get_column (source_alias source) false (input "Field") false) outer)))))
					(define info ((tsql_column_compiler planning) (sql_expr_info outer node) declaration))
					(list 'sql_type_expr (cons (tsql_computed_off_function source declaration recipe) inputs)
						(sql_info_datatype info) (sql_info_collation info)))))))
		_ (tsql_map_executable_children node (lambda (child) (visit child outer)))))))
/* Frontend options own the profile, including standalone compilation defaults.
The optional planning handle remains only a catalog/guard context. */
(define tsql_bind_computed_profile (lambda (query planning tx enabled)
	(if enabled query
		(tsql_expand_computed_columns (bind_query_names query '()) '() planning tx))))
/* Persisted native definitions are strict pure computations. The OFF query
profile supplies its error boundary at the ordinary relational expression. */
(define tsql_computed_policy_formula (lambda (formula enabled) (match formula
	((symbol quote) _value) formula
	(list head _previous ((symbol lambda) '() body))
	(if (or (equal? head 'tsql_arithmetic_value) (expression_equal? head tsql_arithmetic_value))
		(if enabled (tsql_computed_policy_formula body true)
			(list head false (list 'lambda '() (tsql_computed_policy_formula body false))))
		(cons (tsql_computed_policy_formula head enabled)
			(map (cdr formula) (lambda (child) (tsql_computed_policy_formula child enabled)))))
	(cons head arguments) (cons (tsql_computed_policy_formula head enabled)
		(map arguments (lambda (child) (tsql_computed_policy_formula child enabled))))
	_ formula)))
(define tsql_build_dml_plan (lambda (name alias query planning_session tx enabled) (begin
	(define query (tsql_bind_column_compilers query (tsql_column_compiler planning_session)))
	(define query (make_query_block (qb_schema query) (qb_sources query) (qb_fields query) (qb_where query)
		(qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query)
		(qb_hidden query) (qb_stages query) (qassoc_set (qb_facts query) 'dml_count_matched true)))
	(define expanded (if enabled query
		(tsql_bind_computed_profile (sql_bind_catalog_declarations query tx planning_session) planning_session tx enabled)))
	(define formula (build_dml_plan name alias expanded planning_session tx))
	(if (local_relation_reference? name)
		(tsql_independent_dml_expr (local_relation_symbol name)
			(physical_dml_descriptor expanded (qb_schema expanded) name) formula)
		formula))))
(define create_sql_type_alias (lambda (database name base dimensions nullable tx session)
	(tsql_type_ddl database tx session (lambda () (begin
		/* Until colliding user aliases have a bound precedence contract, reject
		the ambiguity explicitly instead of shadowing the built-in declaration. */
		(if (equal?? name "sysname") (error "user alias colliding with built-in type is not supported") true)
		(define native_type (toUpper base))
		(if (tsql_builtin_type_declaration native_type) (error "alias base must be a native type") true)
		(define native_dimensions (tsql_decl_dimensions native_type dimensions))
		(if (has? '("TIMESTAMP" "ROWVERSION") native_type) (error "generated versions cannot be alias base types") true)
		(if (tsql_find_type_alias database name nil) (error "alias type already exists: " name) true)
		(insert (table "system" false "tsql_types" false tx) '("database" "name" "base_type" "dimensions" "nullable")
			(list (list database name native_type (json_encode native_dimensions) nullable)))
		true)))))
(define drop_sql_type_alias (lambda (database name ifexists tx session)
	(tsql_type_ddl database tx session (lambda () (begin
		(define alias (tsql_find_type_alias database name nil))
		(if (nil? alias) (if ifexists false (error "alias type does not exist: " name)) (begin
			(define qualified (tsql_qualified_type database (alias "name")))
			/* Enumerating declarations is DROP TYPE work, never query costing or
			row execution. The database gate excludes concurrent alias consumers. */
			(define dependent (find (show database) (lambda (relation)
				(find (show database relation) (lambda (column) (equal?? (column "RawType") qualified)))) nil))
			(if dependent (error "alias type is used by table: " dependent) true)
			(scan nil (table "system" false "tsql_types" false tx)
				(list 369436175368192 (scan_boundary "equal" "id" 0 0 true true "" false))
				(list (alias "id")) '() (lambda () true) '("$update")
				(lambda (count remove) (begin (remove) (+ count 1))) 0 +)
			true)))))))

(define tsql_native_types '("VARCHAR" "CHAR" "NVARCHAR" "NCHAR" "BINARY" "VARBINARY"
	"DECIMAL" "NUMERIC" "FLOAT" "TIME" "DATETIME2" "DATETIMEOFFSET" "INT" "INTEGER" "BIGINT"
	"SMALLINT" "TINYINT" "BIT" "REAL" "MONEY" "SMALLMONEY" "DATE" "DATETIME" "SMALLDATETIME"
	"ROWVERSION" "TIMESTAMP" "UNIQUEIDENTIFIER" "TEXT" "NTEXT" "IMAGE" "XML"))
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
			(define alias (tsql_find_type_alias database name planning_session))
			(if alias true (tsql_raise_error 2715 (concat "Cannot find data type " name ".") 1))
			(merge (list "type" (tsql_qualified_type database (alias "name"))) alias))))))
/* Only already-folded DEFAULT syntax reaches this probe. The existing text
assignment compiler owns encoding, units and ignored trailing spaces. A strict
truncation proves that the actual writing session must choose the policy;
no DDL session is sampled or retained. Other conversion errors remain errors. */
(define tsql_default_policy_sensitive? (lambda (value declaration)
	(begin
		(define type (declaration "RawType"))
		(define dimensions (coalesceNil (declaration "Dimensions") '()))
		(if (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
			(begin
				(define collation (declaration "Collation"))
				(define input (sql_expr_info '() (if collation
					(list 'sql_compile_expr (list 'tsql_collation_compiler collation) value) value)))
				(define strict ((tsql_assignment_text_compiler type dimensions true) (list input)))
				(define formula (sql_info_formula strict))
				/* A destination conversion can introduce context even when its input
				was folded, for example DATETIME's language-dependent text format. */
				(if (tsql_default_dynamic? formula) true
					(if (or (empty_list? dimensions) (>= (car dimensions) 0))
						(try (lambda () (begin (eval formula) false))
							(lambda (failure) (match (failure_value failure)
								'("tsql-error" 8152 _message _state) true
								_ (error failure)))) false))) false))))
/* CREATE and ALTER share one constant DEFAULT destination owner. The source
has already been proved constant; its pure declared conversion needs no session.
Deferred SQL source remains bound by the later actual insertion session. */
(define tsql_default_destination_value (lambda (value type dimensions)
	(eval (tsql_computed_policy_formula
		(sql_type_formula '() (tsql_implicit_cast_expr value (tsql_addition_type (toUpper type)) dimensions)) true))))
(define tsql_prepare_column (lambda (database definition planning_session) (match definition
	'((symbol list) "column" name type dimensions attributes) (begin
		(define contract (tsql_resolve_schema_type database type (cdr dimensions) planning_session))
		(if (contract "id") (tsql_guard_type_alias (contract "type") (contract "id") planning_session) true)
		(define base (coalesceNil (contract "base_type") (contract "type")))
		(define native_dimensions (contract "dimensions"))
		(define options (tsql_column_collation_defaults base (cdr attributes)))
		(if (options "rowguidcol")
			(if (and (equal? base "UNIQUEIDENTIFIER") (nil? (contract "id"))) true
				(error "ROWGUIDCOL requires the native UNIQUEIDENTIFIER declaration")) true)
		(if (and (has? '("ROWVERSION" "TIMESTAMP") base)
			(or (has_assoc? options "default") (has_assoc? options "default_expression")))
			(error "generated version column cannot have a DEFAULT declaration") true)
		(define null_options (if (or (nil? (contract "base_type")) (has_assoc? options "null")) options
			(append options "null" (contract "nullable"))))
		/* Effective alias nullability is known here. NOT NULL ADD initializes
		historical rows even when the spelling omitted an explicit NOT NULL. */
		(define historical_options (if (and (has_assoc? null_options "fill_existing")
			(has_assoc? null_options "null") (not (null_options "null")))
			(set_assoc null_options "fill_existing" true) null_options))
		(if (and (contract "id") (null_options "auto_increment"))
			(error "IDENTITY alias declarations are not yet supported") true)
		(define identity_options (tsql_identity_attributes base native_dimensions historical_options))
		/* Context-free defaults use the native value. Temporal text retains its
		SQL recipe for conversion under the later writing session. */
		(define default_options (if (if planning_session (planning_session "__tsql_compile_only") false)
			identity_options (tsql_prepare_default identity_options base
				(lambda (value) (tsql_default_policy_sensitive? value (list
					"RawType" base "Dimensions" native_dimensions "Collation" (identity_options "collate"))))
				(lambda (value) (tsql_default_destination_value value base native_dimensions)))))
		(define marked_options (if (default_options "rowguidcol")
			(append (merge (extract_assoc default_options (lambda (key value)
				(if (equal? key "rowguidcol") '() (list key value)))))
				"properties" (list (quote list) "tsql.rowguidcol" true)) default_options))
		(define published_options (if (contract "id") (append marked_options "__tsql_alias_id" (contract "id")) marked_options))
		(list (quote list) "column" name (tsql_storage_type (contract "type"))
			(cons (quote list) native_dimensions) (cons (quote list) published_options)))
	_ definition)))
/* A stored row computation consumes declaration facts before its table exists.
Only ordinary local columns enter this compile scope; no native handles or
request state can become part of the persisted function. */
(define tsql_computed_source (lambda (database definitions) (list
	"__computed" database "__declaration" false nil
	(list "columns" (merge (map definitions (lambda (definition) (match definition
		'((symbol list) "column" name type dimensions (cons (symbol list) attributes))
		(list (list "Field" name "RawType" type "Dimensions" (cdr dimensions)
			"Null" (coalesceNil (attributes "null") true)
			"Collation" (coalesceNil (attributes "collate") "utf8mb4")))
		_ '())))) "constraints" '("Unique" () "ForeignKeys" ())))))
(define tsql_computed_formula_safe? (lambda (formula) (match formula
	/* Only the compiler's literal ASCII-space trim is admitted here;
	regexp callbacks and dynamic patterns retain their existing rejection. */
	((symbol regexp_replace) value " +$" "") (tsql_computed_formula_safe? value)
	((symbol equal_collate) left right (string? _collation))
	(and (tsql_computed_formula_safe? left) (tsql_computed_formula_safe? right))
	((symbol get_column) _alias _alias_ic _name _name_ic) true
	((symbol quote) value) (begin (sql_declaration_data value) true)
	(list head enabled ((symbol lambda) '() body))
	(and (or (identical? enabled true) (identical? enabled false)) (or (equal? head 'tsql_arithmetic_value) (expression_equal? head tsql_arithmetic_value))
		(tsql_computed_formula_safe? body))
	/* The trusted expression compiler selected this collation from declarations.
	It is an immutable comparison function, never a SQL/session input. */
	((symbol sql_compare) left right _comparison _operation)
	(and (tsql_computed_formula_safe? left) (tsql_computed_formula_safe? right))
	/* Character equality is selected by the SQL declaration compiler and is
	pure over its two values and declared collation. Admit that exact closed
	helper alongside the existing numeric/cast heads, never arbitrary closures. */
	(cons head arguments) (if (identical? head sql_compare)
		(match arguments
			(list left right _comparison _operation)
			(and (tsql_computed_formula_safe? left) (tsql_computed_formula_safe? right))
			_ false)
		(and
			(has? (list 'if 'and 'or 'not 'equal? 'equal?? '< '<= '> '>=
				'coalesceNil 'nil? '+ '- '* '/ 'div_null 'intdiv 'mod 'sql_not
				'tsql_cast_integer 'tsql_cast_float 'tsql_cast_decimal 'tsql_cast_text
				'tsql_cast_fixed_text 'tsql_numeric_input 'sql_decimal_value 'tsql_padded_text_equal
				'tsql_checked_integer_operation tsql_checked_integer_operation
				'tsql_integer_divide 'tsql_fractional_divide tsql_integer_divide tsql_fractional_divide
				+ - * / div_null intdiv mod coalesceNil nil? equal? equal?? < <= > >= not sql_not
				tsql_cast_integer tsql_cast_float tsql_cast_decimal tsql_cast_text
				tsql_cast_fixed_text tsql_numeric_input sql_decimal_value tsql_padded_text_equal) head)
			(reduce arguments (lambda (safe arg) (and safe (tsql_computed_formula_safe? arg))) true)))
	_ (or (nil? formula) (string? formula) (number? formula)
		(identical? formula true) (identical? formula false)))))
/* Frontend helpers are Scheme globals, not declared runtime builtins. Bind
those values explicitly before closing a persisted function so its meaning is
independent of receiver-side global names. */
(define tsql_computed_bindings (lambda () (list
	(list 'sql_compare sql_compare) (list 'sql_decimal_value sql_decimal_value)
	(list 'tsql_padded_text_equal tsql_padded_text_equal)
	(list 'tsql_cast_integer tsql_cast_integer) (list 'tsql_cast_float tsql_cast_float)
	(list 'tsql_cast_decimal tsql_cast_decimal) (list 'tsql_cast_text tsql_cast_text)
	(list 'tsql_cast_fixed_text tsql_cast_fixed_text) (list 'tsql_numeric_input tsql_numeric_input)
	(list 'tsql_checked_integer_operation tsql_checked_integer_operation)
	(list 'tsql_arithmetic_value tsql_arithmetic_value)
	(list 'tsql_integer_divide tsql_integer_divide) (list 'tsql_fractional_divide tsql_fractional_divide))))
(define tsql_computed_function (lambda (parameters body) (begin
	(define bindings (tsql_computed_bindings))
	(list 'close_procedure (cons
		(list 'lambda (map bindings car) (list 'lambda parameters body))
		(map bindings cadr))))))
/* Persist frontend expression data, never an optimized procedure's numbered
locals or capture layout. Only this exact pure helper registry is restored. */
(define tsql_computed_recipe_expr (lambda (formula) (begin
	(define binding (find (tsql_computed_bindings)
		(lambda (entry) (expression_equal? formula (cadr entry))) nil))
	(if binding (car binding) (match formula
		((symbol quote) _value) formula
		(cons head arguments) (cons (tsql_computed_recipe_expr head)
			(map arguments tsql_computed_recipe_expr))
		_ (if (or (nil? formula) (string? formula) (number? formula) (symbol? formula)
			(identical? formula true) (identical? formula false)) formula
			(scheme (serialize formula))))))))
(define tsql_computed_recipe_comparisons? (lambda (formula) (match formula
	((symbol quote) _value) true
	((symbol get_column) alias alias_ic name name_ic)
	(and (equal? alias "__computed") (identical? alias_ic false)
		(string? name) (identical? name_ic false))
	((symbol sql_compare) left right comparison operation)
	(and (has? '("equal?" "equal??" "<" ">" "<=" ">=") operation)
		(tsql_computed_recipe_comparisons? left) (tsql_computed_recipe_comparisons? right)
		(or (has? '(equal? equal?? < <= > >=) comparison)
			(match comparison
				((symbol collate) name reverse) (and (string? name)
					(or (identical? reverse true) (identical? reverse false)))
				_ false)))
	(cons _head arguments) (reduce arguments (lambda (safe child)
		(and safe (tsql_computed_recipe_comparisons? child))) true)
	_ true)))
/* Older native computations have no frontend expression profile. Their pure
ON definition remains readable; OFF cannot reconstruct lost SQL operation
boundaries from an optimized callable and therefore reports that absence. */
/* The Scheme reader produces expression symbols for reserved scalar literals.
Restore only those exact data encodings before applying the pure recipe checks. */
(define tsql_computed_recipe_data (lambda (expression) (match expression
	(symbol true) true
	(symbol false) false
	(symbol nil) nil
	(cons head arguments) (cons (tsql_computed_recipe_data head)
		(map arguments tsql_computed_recipe_data))
	_ expression)))
/* Stored row recipes carry physical dependency names. Build their lexical
input declaration from those same coordinates, rather than rebinding an old
SQL spelling after a rename. Ordinary SQL sources retain their visible names. */
(define tsql_computed_input_source (lambda (source)
	(list "__computed" (source_schema source) (source_relation source) false nil
		(set_assoc (source_declarations source) "columns"
			(map (source_columns source) (lambda (column)
				(set_assoc column "Field" (sql_column_storage_name column))))))))
(define tsql_computed_off_function (lambda (source declaration recipe) (begin
	(define properties (coalesceNil (declaration "Properties") '()))
	(define encoded (properties "tsql.computed.off"))
	(if (string? encoded) true (error "computed declaration has no ANSI_WARNINGS OFF expression profile"))
	(define formula (match (tsql_computed_recipe_data (expression_syntax (scheme encoded)))
		(list "tsql-computed-off-v1" body) body
		_ (error "invalid computed expression profile")))
	(if (and (tsql_computed_formula_safe? formula) (tsql_computed_recipe_comparisons? formula)) true
		(error "computed expression profile is not a supported pure scalar"))
	(define input_source (tsql_computed_input_source source))
	(define inputs (extract_columns_for_alias input_source formula))
	(if (expression_equal? inputs (car recipe)) true
		(error "computed expression profile input declaration changed"))
	(eval (tsql_computed_function
		(map inputs (lambda (name) (scan_callback_symbol_for_alias "__computed" name)))
		(lower_column_expr_for_alias input_source formula))))))
(define tsql_compile_computed_declarations (lambda (database definitions planning_session) (begin
	(define source (tsql_computed_source database definitions))
	(define names (merge (map definitions (lambda (definition) (match definition
		'((symbol list) "column" name _type _dims _attrs) (list name)
		'((symbol list) "computed" name _expr _persisted) (list name)
		_ '())))))
	(if (equal? (count names) (count (merge_unique (list (map names toLower)))))
		true (error "duplicate column name"))
	(map definitions (lambda (definition) (match definition
		'((symbol list) "computed" name expression persisted) (begin
			/* PERSISTED requires a distinct durable-value contract; lazy caching
			must not silently claim that its values were persisted as base data. */
			(if persisted (error "PERSISTED computed columns are not supported") true)
			(define bound (bind_query_names (make_query_block database (list source)
				(list "value" (tsql_bind_column_compilers expression (tsql_column_compiler planning_session)))
				true nil nil nil nil nil nil nil nil) '()))
			(define source (car (qb_sources bound)))
			(define info (sql_expr_info (list source) (cadr (qb_fields bound))))
			(define raw_formula (sql_storage_column_references (source_columns source)
				(sql_type_lower_annotations (list source) (sql_info_formula info))))
			(define source (tsql_computed_input_source source))
			(define formula (tsql_computed_policy_formula raw_formula true))
			(define off_formula (tsql_computed_recipe_expr (tsql_computed_policy_formula raw_formula false)))
			(if (and (tsql_computed_formula_safe? formula) (tsql_computed_formula_safe? off_formula)
				(tsql_computed_recipe_comparisons? off_formula)) true
				(error "computed column requires a deterministic local scalar expression"))
			(define inputs (extract_columns_for_alias source formula))
			(map inputs (lambda (column)
				(if (source_column_declaration source column false) true
					(error "computed input column does not exist: " column))))
			(if (has? '("any" "NULL") (sql_info_type info))
				(error "computed column has no resolved SQL type") true)
			(define attributes (append (if (sql_info_collation info)
				(list "null" true "collate" (sql_collation_name info)) '("null" true))
				"properties" (list 'list "tsql.computed.off"
					(serialize (list "tsql-computed-off-v1" off_formula)))))
			(list 'list "column" name (tsql_storage_type (sql_info_type info))
				(cons 'list (sql_info_dimensions info)) (cons 'list attributes)
				(cons 'list inputs)
				(tsql_computed_function
					(map inputs (lambda (column) (scan_callback_symbol_for_alias (source_alias source) column)))
					(lower_column_expr_for_alias source formula))))
		_ definition))))))
(define tsql_prepare_computed_columns (lambda (database definitions planning_session)
	(if (find definitions (lambda (definition) (match definition
		'((symbol list) "computed" _name _expression _persisted) true _ false)))
		(tsql_compile_computed_declarations database definitions planning_session) definitions)))
(define tsql_publish_alias_attributes (lambda (type attributes) (begin
	(if (has_assoc? attributes "__tsql_alias_id")
		(if (equal? ((tsql_alias_declaration type nil) "id") (attributes "__tsql_alias_id")) true
			(error "alias declaration changed before DDL execution")) true)
	(merge (extract_assoc attributes (lambda (key value)
		(if (equal? key "__tsql_alias_id") '() (list key value))))))))
(define tsql_rebuild_relation (lambda (database name tx) (begin
	(define handle (table database false name false tx))
	(if (nil? handle) (tsql_raise_error 4902 (concat "table does not exist: " name) 1) true)
	/* Rebuild uses the native parallel storage operator. Completion and I/O
	failure belong to that owner; no textual status is interpreted as success. */
	(rebuild handle true false tx true)
	0)))
/* These scalar generators belong to the SQL default declaration. Native ADD
receives the actual callable plus inert future DEFAULT data, never SQL syntax
or a pre-sampled UUID. The native batch owns statement Undo before its FIRST
column, including ordinary siblings of a generated column. */
(define tsql_add_column_initializer (lambda (attributes)
	(match (toUpper (coalesceNil (attributes "default_expression") ""))
		"NEWID" (lambda () (uuid_v4))
		"UUID_V4" (lambda () (uuid_v4))
		"UUIDV4" (lambda () (uuid_v4))
		"GEN_RANDOM_UUID" (lambda () (uuid_v4))
		"UUID_GENERATE_V4" (lambda () (uuid_v4))
		"UUID_V7" (lambda () (uuid_v7))
		"UUIDV7" (lambda () (uuid_v7))
		_ nil)))
(define tsql_add_columns (lambda (database name columns tx session) (begin
	(define marked (tsql_validate_row_guid_definitions columns))
	/* Resolve all declaration/default profiles before any column publication. */
	(if (find columns (lambda (column) (begin
		(define options (nth column 4))
		(and (options "default_deferred") (has_assoc? options "null") (not (options "null"))
			(nil? (tsql_add_column_initializer options))))))
		(error "ALTER ADD NOT NULL with a deferred DEFAULT requires historical initialization") true)
	(if (find columns (lambda (column) (begin
		(define options (nth column 4))
		(and (tsql_row_guid_definition? column) (nil? (tsql_add_column_initializer options))
			(or (identical? (options "null") false) (has_assoc? options "default")
				(has_assoc? options "default_expression"))))))
		(error "ROWGUIDCOL in ALTER ADD requires its atomic initialized declaration") true)
	(define publish (lambda () (begin
		(define published_columns (map columns (lambda (column) (match column
			'("column" column_name type dimensions attributes)
			(list column_name type dimensions (tsql_publish_alias_attributes type attributes))))))
		(define initialized (find published_columns (lambda (column)
			(not (nil? (tsql_add_column_initializer (nth column 3))))) nil))
		/* A nullable marker without a DEFAULT uses the same native NULL history
		and exclusive declaration claim; ROWGUIDCOL never invents a generator. */
		(if (or initialized marked) (begin
			(define handle (table database false name false tx))
			(define collation (get_assoc (show handle true) "Collation"))
			(createcolumns handle (map published_columns (lambda (column) (match column
				'(column_name type dimensions attributes) (begin
					(define attributes (sql_add_column_attributes attributes))
					(define attributes (if (if (attributes "properties") ((attributes "properties") "tsql.rowguidcol") false)
						(append attributes "exclusive_properties" '("tsql.rowguidcol")) attributes))
					(list column_name type dimensions
						(sql_column_collation_defaults type (append attributes "name_ignorecase" true) collation)
						(tsql_add_column_initializer attributes)))))) tx))
			(reduce published_columns (lambda (_ column) (match column '(column_name type dimensions attributes)
				(if (sql_create_column (table database false name false tx) column_name type dimensions
					(merge (sql_add_column_attributes attributes) '("name_ignorecase" true)) tx) true
					(error "column already exists: " column_name)))) true)))))
	(if (find columns (lambda (column) (has_assoc? (nth column 4) "__tsql_alias_id")))
		(tsql_type_ddl database tx session publish) (publish)))))


/* Transport tuples describe bytes and unscaled coefficients only. Their SQL
meaning is chosen here from the request's declaration, not in the Go bridge. */
/* The transport exposes integer day/clock fields, never a SQL runtime type.
Validate their declaration-specific units once per parameter. Generic calendar
parts preserve nanoseconds; SQL scale and legacy tick units stay in this frontend.
Offset-bearing declarations remain unsupported until their zone contract is modeled. */
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
	(define seconds (if short (* ticks 60) (intdiv ticks factor)))
	(define nanos (if short 0 (intdiv (+ (* (mod ticks factor) 1000000000) (intdiv factor 2)) factor)))
	(define date (tsql_calendar_shift_seconds
		(date_add (parse_date (if (or legacy (equal? kind 41)) "1900-01-01" "0001-01-01")) days "DAY") seconds))
	(define date (date_from_parts (unix_timestamp date) nanos 0))
	/* TIME retains its existing clock-text carrier, now including the declared
	fraction; other contracts use the normal native calendar scalar. */
	(if (equal? kind 41) (tsql_temporal_clock_text date (wire "scale")) date))))
(define tds_money_small? (lambda (wire) (match (wire "size")
	4 true 8 false _ (error "invalid currency parameter byte width"))))
(define tds_parameter_value (lambda (parameter) (begin
	(define value (parameter "value"))
	(define wire (parameter "type"))
	(if (nil? value) nil
		(equal? (wire "kind") 36) (begin
			(if (equal? (wire "size") 16) true (error "invalid GUID parameter width"))
			(uuid_from_bin value))
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
	(define date (if (and (string? value) (equal? kind 41))
		(tsql_calendar_text value "ymd" "us_english" false) (parse_date value)))
	(if date true (error "invalid temporal wire result"))
	(define parts (date_parts date))
	(define seconds (car parts))
	(define day (intdiv (floor (/ seconds 86400)) 1))
	(define clock (- seconds (* day 86400)))
	(define legacy (equal? kind 111))
	(define short (and legacy (equal? (descriptor "size") 4)))
	(if (equal? kind 41) true
		(and (>= seconds (if legacy (if short -2208988800 -6847804800) -62135596800))
			(<= seconds (if short 3453321540 253402300799))) true
		(error "calendar wire result overflow"))
	(if (and short (or (not (equal? (mod clock 60) 0)) (not (equal? (cadr parts) 0))))
		(error "small datetime wire result requires whole minutes") true)
	(define factor (if (equal? kind 40) 1 legacy 300 (nth '(1 10 100 1000 10000 100000 1000000 10000000) (descriptor "scale"))))
	(define ticks (if (equal? kind 40) 0 short (intdiv clock 60)
		(+ (* clock factor) (intdiv (+ (* (cadr parts) factor) 500000000) 1000000000))))
	(define carry (if (or short (equal? kind 40)) 0 (intdiv ticks (* 86400 factor))))
	(if (and (not (equal? kind 41)) (> (+ day carry) 2932896))
		(error "calendar wire rounding overflow") true)
	(list "days" (if (equal? kind 41) 0 (+ day carry (if legacy 25567 719162)))
		"ticks" (if (or short (equal? kind 40)) ticks (mod ticks (* 86400 factor)))
		"offset" 0))))
(define tds_result_value (lambda (descriptor value) (if (nil? value) nil
	(match (descriptor "kind")
		36 (list "bytes" (uuid_to_bin value))
		40 (tds_temporal_result_value descriptor value)
		41 (tds_temporal_result_value descriptor value)
		42 (tds_temporal_result_value descriptor value)
		104 (tsql_cast_bit value)
		111 (tds_temporal_result_value descriptor value)
		165 (list "bytes" value)
		173 (list "bytes" value)
		110 (tsql_money_coefficient value (equal? (descriptor "size") 4))
		106 (tds_result_value (set_assoc descriptor "kind" 108) value)
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

/* Both RPC forms use the descriptor as their SQL source declaration. Stored
procedure formals still own the destination contract and default/output binding. */
(define tds_parameter_source_type (lambda (wire) (begin
	(if (not (has? '(31 36 38 40 41 42 43 104 106 108 109 110 111 165 167 173 175 231 239) (wire "kind")))
		(error "unsupported T-SQL parameter wire type: " (wire "kind")) true)
	(match (wire "kind")
		36 "UNIQUEIDENTIFIER"
		38 "BIGINT" 104 "BIT" 106 "DECIMAL" 108 "NUMERIC" 109 "FLOAT"
		40 "DATE" 41 "TIME" 42 "DATETIME2" 43 "DATETIMEOFFSET"
		111 (if (equal? (wire "size") 4) "SMALLDATETIME" "DATETIME")
		110 (if (tds_money_small? wire) "SMALLMONEY" "MONEY")
		165 "VARBINARY" 173 "BINARY" 167 "VARCHAR" 175 "CHAR" 231 "NVARCHAR" 239 "NCHAR" _ "NULL"))))
(define tds_parameter_assignment (lambda (session parameter declaration) (begin
	(define contract (tsql_effective_parameter_contract declaration))
	(define wire (parameter "type"))
	(define source_type (tds_parameter_source_type wire))

	(if (and (not (nil? (parameter "value"))) (tsql_binary_type? source_type)
		(not (tsql_binary_type? (car contract))))
		(error "binary parameter conversion is not implemented") true)
	(if (and (not (nil? (parameter "value"))) (tsql_temporal_type? source_type)
		(not (or (tsql_temporal_type? (car contract)) (sql_text_type? (car contract)))))
		(error "temporal parameter conversion is not implemented") true)
	/* Generated implicit casts may read SQL option values. The real calling
	scope remains a lexical invocation input to eval, as in prepared binding. */
	(define converted (if (nil? (parameter "value")) nil
		(eval (sql_type_lower_annotations '()
			(tsql_implicit_cast_expr (list (quote sql_type_expr) (list (quote quote) (tds_parameter_value parameter)) source_type nil)
				(car contract) (cdr contract))))))
	converted)))

/* Bind at the request boundary, not in storage or row loops. NULL is a supplied
value; the wire descriptor and SQL declaration own the type independently. */
(define tds_parameter_name (lambda (name positional)
	(if (equal? name "") positional
		(toLower (if (equal? (substr name 0 1) "@") (substr name 1) name)))))
(define tds_parameter_bindings (lambda (declarations parameters) (begin
	(if (not (equal? (count declarations) (count parameters)))
		(error "T-SQL parameter count differs from declarations") true)
	(define names (map declarations car))
	(if (not (equal? (count names) (count (merge_unique (list names)))))
		(error "duplicate T-SQL parameter declaration") true)
	(define named (merge (map (produceN (count parameters)) (lambda (i) (begin
		(define parameter (nth parameters i))
		(define name (tds_parameter_name (parameter "name") (car (nth declarations i))))
		(if (has? names name) true (error "unknown T-SQL parameter: @" name))
		(define declaration (find declarations (lambda (entry) (equal? (car entry) name)) nil))
		(if (equal? (parameter "flags") 0) true
			(equal? (parameter "flags") 1)
			(if (if (> (count declaration) 2) (nth declaration 2) false) true
				(error "parameter is not declared OUTPUT: @" name))
			(error "unsupported RPC parameter flags"))
		(list name parameter))))))
	(if (not (equal? (count parameters) (count (merge_unique (list (extract_assoc named (lambda (name _value) name)))))))
		(error "duplicate T-SQL parameter value") true)
	named)))
(define tds_bind_parameters (lambda (session declarations parameters) (begin
	(define named (tds_parameter_bindings declarations parameters))
	(define values (map declarations (lambda (declaration) (begin
		(define name (car declaration))
		(define parameter (named name))
		(define converted (tds_parameter_assignment session parameter (cadr declaration)))
		(list (concat "__tsql_parameter:" name) converted)))))
	/* All declarations and values are validated before installing this request. */
	(map values (lambda (item) (session (car item) (cadr item))))
	(session "__tsql_parameter_declarations" declarations)
	true)))

/* Description binds names and expression contracts without physical planning
or execution. A single mutation without OUTPUT has an empty result contract;
its ordinary grammar still validates declarations and input expressions. */
(define tsql_describe_statement (lambda (tx schema text declarations session) (begin
	(tsql_validate_login_identity session)
	(tsql_require_database_access (tsql_session_actor session) schema)
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" declarations)
	(planning "__tsql_local_relations" (coalesceNil (session "__tsql_local_relations") '()))
	(planning "__tsql_describe" true)
	(planning "__tsql_quoted_identifier" (tsql_quoted_identifier session))
	(planning "__tsql_ansi_nulls" (tsql_ansi_nulls session))
	(planning "username" (session "username"))
	(planning "__tsql_principal_id" (session "__tsql_principal_id"))
	(planning "tsql_import_target" (session "tsql_import_target"))
	(planning "tsql_import_source" (session "tsql_import_source"))
	(define bindings (tsql_temporary_bindings session))
	(planning "__tsql_temporary_bindings" bindings)
	(match (parse_tsql schema (tsql_statement_text text)
		(tsql_temporary_policy (session "username") bindings) planning tx)
		((symbol quote) fields) fields
		_ (error "unsupported T-SQL describe shape")))))
(define tds_describe (lambda (schema text declarations session) (begin
	(define tx (if (session "transaction") ((tsql_connection_session session) "__memcp_tx") nil))
	/* The trusted diagnostic API keeps its historical default; wire sessions
	always retain the authenticated explicit account. */
	(if (nil? (session "username")) (session "username" "root") true)
	(tsql_describe_statement tx schema text (tds_bind_declarations schema declarations) session))))

/* INSERT EXEC borrows one native statement transaction across its procedure.
DDL and transaction boundaries cannot join that statement's atomic write set.
Classify with the SQL lexer, so leading comments never bypass this guard. */
(define tsql_insert_exec_leaf_kind (parser '((* ";")
	(define keyword (regex "[A-Za-z]+" false false))
	(regex "(?s).*" false false)) (toUpper keyword)
	"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))
/* A separate frontend entrypoint keeps dialect decisions out of existing SQL
execution paths. The cache, transaction owner and compiled evaluator are shared. */
/* SQL transaction meanings stay in the frontend. The native TDS adapter only
consumes explicit transport edges and current database names. */
/* READ COMMITTED uses native ACID row staging for implicit statements and
explicit BEGIN alike. SET changes the scoped SQL option, never the transaction
owner: BEGIN would finalize/reset pending work and is not an isolation setter. */
(define tsql_set_transaction_isolation (lambda (session level tx) (begin
	(if (has? '("READ COMMITTED" "SERIALIZABLE") level) true
		(error "unsupported T-SQL isolation level: " level))
	(if (and (equal? level "SERIALIZABLE") ((tsql_connection_session session) "transaction"))
		(tx_serializable tx) true)
	(session "__tsql_isolation_level" level)
	true)))
/* SQL nesting changes @@TRANCOUNT, not the native transaction. Only the
outermost BEGIN installs its name; only its final COMMIT decides storage.
Names and savepoint capabilities belong to the real connection, including
when a procedure or dynamic batch executes the control statement.
https://learn.microsoft.com/sql/t-sql/language-elements/begin-transaction-transact-sql
https://learn.microsoft.com/sql/t-sql/language-elements/save-transaction-transact-sql */
(define tsql_transaction_count (lambda (session) (begin
	(define connection (tsql_connection_session session))
	(if (connection "transaction")
		(coalesceNil (connection "__tsql_transaction_count") 1) 0))))
(define tsql_transaction_clear (lambda (connection) (begin
	(connection "__tsql_transaction_count" nil)
	(connection "__tsql_transaction_name" nil)
	true)))
/* Capabilities stay in the existing native transaction resource scope, not
enumerable session/planning bindings. Decision cleanup releases this metadata;
the producer captures neither a connection nor a transaction. */
(define tsql_transaction_savepoint_state (lambda (tx)
	(tx_get_or_compute_resource tx "tsql-savepoints" (lambda () (newsession)))))
/* A name/admission error has no storage effects. Report it after the existing
native statement completes, so CATCH does not abort an earlier transaction.
Native failures themselves are never converted to this SQL diagnostic. */
(define tsql_transaction_error (lambda (session number message) (begin
	(define failure (list "tsql-error" number message 1))
	(define diagnostic (session "__tsql_statement_error"))
	(if diagnostic (begin (diagnostic failure) false) (error failure)))))
(define tsql_begin_transaction (lambda (session tx name) (begin
	(define connection (tsql_connection_session session))
	(define depth (tsql_transaction_count connection))
	(if (> depth 0) (begin
		(tx_check tx)
		(connection "__tsql_transaction_count" (+ depth 1)))
		(begin
			(tx_begin_acid connection tx)
			(if (equal? (session "__tsql_isolation_level") "SERIALIZABLE") (tx_serializable tx) true)
			(connection "__tsql_transaction_count" 1)
			(connection "__tsql_transaction_name" (if (nil? name) nil (car name)))))
	(connection "__tsql_transaction_end" 0)
	true)))
(define tsql_commit_transaction (lambda (session name) (begin
	/* COMMIT's optional name is parsed and typed but does not select a
	transaction. An inner COMMIT cannot publish the outer transaction.
	https://learn.microsoft.com/sql/t-sql/language-elements/commit-transaction-transact-sql */
	(define connection (tsql_connection_session session))
	(define depth (tsql_transaction_count connection))
	(if (and (> depth 0) (equal? (tx_state (connection "__memcp_tx")) "rollback-only"))
		(tsql_transaction_error session 3930 "The current transaction cannot be committed; roll back the transaction.")
		(equal? depth 0)
		(tsql_transaction_error session 3902 "The COMMIT TRANSACTION request has no corresponding BEGIN TRANSACTION.")
		(> depth 1) (begin
			(tx_check (connection "__memcp_tx"))
			(connection "__tsql_transaction_count" (- depth 1))
			true)
		(begin
			(tx_commit connection)
			(tsql_transaction_clear connection)
			(connection "__tsql_transaction_end" 2)
			true)))))
(define tsql_save_transaction (lambda (session tx name) (begin
	(define connection (tsql_connection_session session))
	(if (equal? (tx_state tx) "rollback-only")
		(tsql_transaction_error session 3930 "The current transaction cannot create a savepoint; roll back the transaction.")
		(equal? (tsql_transaction_count connection) 0)
		(tsql_transaction_error session 628 "Cannot issue SAVE TRANSACTION when there is no active transaction.")
		(nil? (car name))
		(tsql_transaction_error session 3977 "The savepoint name cannot be NULL.")
		(begin
			(define state (tsql_transaction_savepoint_state tx))
			(define boundary (tx_savepoint tx))
			/* Most recent first: duplicate names select the latest SAVE.
			This list holds control capabilities, never relational rows. */
			(state "history"
				(cons (list (car name) boundary)
					(coalesceNil (state "history") '())))
			true)))))
(define tsql_savepoint_tail (lambda (savepoints name)
	(match savepoints
		(cons point rest) (if (expression_equal? (car point) name) savepoints
			(tsql_savepoint_tail rest name))
		_ '())))
(define tsql_rollback_transaction (lambda (session name) (begin
	(define connection (tsql_connection_session session))
	(if (equal? (tsql_transaction_count connection) 0)
		(tsql_transaction_error session 3903 "The ROLLBACK TRANSACTION request has no corresponding BEGIN TRANSACTION.")
		(if (nil? name) false (nil? (car name)))
		(tsql_transaction_error session 3977 "The savepoint name cannot be NULL.")
		(if (nil? name) true (expression_equal? (car name) (connection "__tsql_transaction_name")))
		(begin
			(tx_rollback connection)
			(tsql_transaction_clear connection)
			(connection "__tsql_transaction_end" 3)
			true)
		(begin
			(if (equal? (tx_state (connection "__memcp_tx")) "rollback-only")
				(tsql_raise_error 3931 "The current transaction cannot roll back to a savepoint; roll back the transaction." 1) true)
			(define state (tsql_transaction_savepoint_state (connection "__memcp_tx")))
			(define remaining (tsql_savepoint_tail
				(coalesceNil (state "history") '()) (car name)))
			(if (empty_list? remaining)
				(tsql_transaction_error session 6401
					(concat "Cannot roll back " (car name) ". No transaction or savepoint of that name was found."))
				(begin
					(tx_rollback_to_savepoint (connection "__memcp_tx") (cadr (car remaining)))
					/* The selected boundary remains usable; later boundaries no
					longer denote retained native history. Count is unchanged. */
					(state "history" remaining)
					true)))))))
(define tds_session_completion (lambda (session) (begin
	(define active (not (nil? (session "transaction"))))
	(define previous (coalesceNil (session "__tds_wire_transaction") false))
	(define action (if (and active (not previous)) 1
		(and previous (not active)) (if (equal? (session "__tsql_transaction_end") 3) 3 2) 0))
	(session "__tds_wire_transaction" active)
	(list "database" (session "schema") "transaction" action))))

/* SQL parameters are immutable inputs to one statement. Bind only their actual
reads after planning: logical domain keys and SQL types remain planner-owned,
while delegated callbacks capture scalar values rather than a lexical scope's
forwarding procedure and its connection capabilities. Session writes and all
other execution bindings retain their existing native owners. */
(define tsql_parameter_input_expr (lambda (expr bindings) (match expr
	((symbol quote) _value) expr
	((symbol session) key) (if (has_assoc? bindings key) (bindings key) expr)
	(cons head tail) (cons (tsql_parameter_input_expr head bindings)
		(map tail (lambda (part) (tsql_parameter_input_expr part bindings))))
	_ expr)))
(define tsql_bind_parameter_inputs (lambda (formula declarations) (begin
	(define names (merge (map declarations (lambda (declaration)
		(list (concat "__tsql_parameter:" (car declaration)) true)))))
	(define reads (filter (query_expr_session_reads formula) (lambda (read)
		(and (string? (cadr read)) (or (has_assoc? names (cadr read))
			/* Adapter-produced cursor metadata is plain SQL input too. Bind it
			at invocation alongside the declared scalar parameters. */
			(tsql_cursor_scalar_input_key? (cadr read)))))))
	(if (empty_list? reads) formula (begin
		(define symbols (map (produceN (count reads)) (lambda (index)
			(symbol (concat "__tsql_parameter_value_" index)))))
		(define bindings (merge (map (produceN (count reads)) (lambda (index)
			(list (cadr (nth reads index)) (nth symbols index))))))
		(cons (list (quote lambda) symbols (tsql_parameter_input_expr formula bindings)) reads))))))

/* Pending commit publication retains its original native owner. Diagnostic
access cannot turn irreversible uncertainty into a committable SQL state or
admit another statement through the aborted-transaction recovery capability. */
(define tsql_bind_transaction_state (lambda (session tx) (begin
	(define state (tx_state tx))
	(if (equal? state "deciding") (error "transaction commit outcome is unknown; retry COMMIT")
		(session "__tsql_native_transaction_state" state)))))
(define tsql_execute_statement (lambda (tx schema text session resultrow resultfields) (begin
	(define session (tsql_cursor_input_session session))
	/* COMMIT/ROLLBACK retain their parsed recovery owner, including COMMIT
	retry after an uncertain durable decision. Ordinary leaves validate below. */
	(session "__tsql_native_transaction_state" (tx_state tx))
	(if (if (session "__tsql_insert_exec")
		(has? '("CREATE" "ALTER" "DROP" "TRUNCATE" "BACKUP" "RESTORE" "USE" "BEGIN" "COMMIT" "ROLLBACK")
			(tsql_insert_exec_leaf_kind text)) false)
		(error "DDL and transaction control are unsupported inside INSERT EXEC") true)
	/* Expose the active scope's scalar value through the ordinary session binding.
	The common planner can then discover and bind this dependency like parameters;
	an opaque scope-reader call would hide it from shared cache/domain analysis. */
	(session "__tsql_scope_identity" ((tsql_identity_scope session) "__tsql_scope_identity"))
	(define input (tsql_statement_text text))
	(define username (coalesce (session "username") "root"))
	(define declarations (coalesceNil (session "__tsql_parameter_declarations") (list)))
	(define local_relations (coalesceNil (session "__tsql_local_relations") '()))
	/* Cache identity includes the complete immutable declaration contract. Values
	remain invocation inputs; differently typed calls cannot reuse wrong metadata. */
	(define import_target (session "tsql_import_target"))
	(define import_source (session "tsql_import_source"))
	(define quoted (tsql_quoted_identifier session))
	(define ansi (tsql_ansi_nulls session))
	(define warnings (tsql_ansi_warnings session))
	(define arithabort (tsql_arithabort session))
	(define concat_null (tsql_concat_null_yields_null session))
	(define rowcount_active (> (coalesceNil (session "__tsql_rowcount_limit") 0) 0))
	(define temporary_scope (tsql_temporary_scope session))
	(define temporary_bindings (tsql_temporary_visible temporary_scope tx))
	(session "__tsql_temporary_bindings" temporary_bindings)
	(define cache_scope (json_encode (list username (session "__tsql_principal_id") declarations local_relations import_target import_source temporary_bindings quoted ansi warnings arithabort concat_null rowcount_active)))
	/* The cached dispatcher retains this parser for future misses. Close its
	declaration/import/name and lexical snapshots here, before embedding it as a literal;
	its creation frame also owns the current session, transaction and outputs. */
	(define parser (close_procedure (lambda (database text policy planning tx) (begin
		(planning "__tsql_parameter_declarations" declarations)
		(planning "__tsql_local_relations" local_relations)
		(planning "tsql_import_target" import_target)
		(planning "tsql_import_source" import_source)
		(planning "__tsql_quoted_identifier" quoted)
		(planning "__tsql_ansi_nulls" ansi)
		(planning "__tsql_ansi_warnings" warnings)
		(planning "__tsql_arithabort" arithabort)
		(planning "__tsql_concat_null_yields_null" concat_null)
		(planning "__tsql_rowcount_active" rowcount_active)
		(planning "__tsql_temporary_bindings" temporary_bindings)
		(planning "__tsql_database_access_requirements" '())
		(define formula (parse_tsql database text policy planning tx))
		(tsql_bind_table_variable_inputs
			(tsql_bind_parameter_inputs (tsql_permission_formula formula planning) declarations) local_relations)))))
	(define setting (tsql_set_options_probe input))
	(if (if setting (has? (cadr setting) "FMTONLY") false)
		(tsql_set_options session (cadr setting) (nth setting 2))
		(session "__tsql_fmtonly") (begin
			/* Describe before physical planning: no scan, DML preparation or
			volatile expression can execute in metadata-only mode. */
			(define fields (tsql_describe_statement tx schema input declarations session))
			(if (empty_list? fields) true
				(resultfields (map fields (lambda (field) (field "name"))) fields))
			0)
		(begin
			(define formula (tsql_compile_phase session (lambda ()
				(cached_parse (temporary_scope "cache") "tsql" (list parser (list "exact_guards" true)) schema input
					(tsql_temporary_policy username temporary_bindings) cache_scope session false tx))))
			(sql_execute_formula session tx formula resultrow resultfields))))))

/* Syntax recipes retain only source and declaration data. Query compilation
and all row execution stay in the existing shared SQL compiler and planner. */
(define tsql_script_cache (newcachemap))

/* The output callback belongs to one request. It is never enumerated into a
planner session or stored in a reusable plan. Nested SQL scopes inherit it. */
(define tsql_message_session (close_procedure ((lambda (tsql_connection_session)
	(lambda (parent message)
		(lambda args (match args
			'() (filter (parent) (lambda (key) (not (equal? key "__tsql_message"))))
			'(key) (if (equal? key "__tsql_message") message
				(equal? key "__tsql_connection_session") (tsql_connection_session parent) (parent key))
			'(key value) (if (equal? key "__tsql_message")
				(error "request message callback is immutable") (parent key value))
			_ (apply parent args))))) tsql_connection_session)))
(define tsql_emit_message (lambda (session number severity state text) (begin
	(define message (list "number" number "state" state "severity" severity
		"text" text "server" "MemCP" "procedure" "" "line" 1))
	(define deliver (session "__tsql_message"))
	(if deliver (deliver message) (print text))
	0)))
(define tsql_run_print (lambda (text statement session) (begin
	(define value (tsql_statement_call statement session "PRINT expression" false (lambda (tx)
		(tsql_scalar_value (session "schema") text session tx "print" nil))))
	(if (nil? value) 0 (tsql_emit_message session 0 0 1 value)))))
(define tsql_scalar_cache (newcachemap))
(define tsql_batch_contracts (lambda (commands inherited) (reduce commands (lambda (contracts command)
	(match command
		'("declare" declarations) (reduce declarations (lambda (contracts declaration) (begin
			(define name (car declaration))
			(if (find contracts (lambda (item) (equal? (car item) name)) nil)
				(error "duplicate variable declaration: @" name) true)
			(append contracts (list name (cadr declaration))))) contracts)
		'("declare-table" name _columns) (begin
			(if (find contracts (lambda (item) (equal? (car item) name)) nil)
				(error "duplicate variable declaration: @" name) true)
			(append contracts (list name '("TABLE"))))
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
	(local "__tsql_local_relations" '())
	(local "__tsql_ansi_nulls_locked" false)
	(local "__tsql_table_variables" (newsession))
	(local "__tsql_connection_session" (tsql_connection_session parent))
	(if module_settings (local "__tsql_temporary_scope"
		(tsql_temporary_new_scope (tsql_temporary_scope parent) (tsql_connection_session parent))) true)
	/* Existing runtime SET options inherit once per module invocation. Writes
	stay local even for OFF/nil, so return and error unwinding restore the caller
	without a shared-session save/restore race. Ordinary batches delegate these
	options, retaining their connection state across GO and subsequent requests. */
	(define module_keys (if module_settings
		'("__tsql_ansi_warnings" "__tsql_arithabort" "__tsql_xact_abort" "__tsql_concat_null_yields_null" "__tsql_identity_insert" "__tsql_datefirst" "__tsql_dateformat" "__tsql_language" "__tsql_nocount" "__tsql_fmtonly" "__tsql_temporary_scope" "__tsql_quoted_identifier" "__tsql_ansi_nulls" "__tsql_isolation_level" "__tsql_rowcount_limit" "__tsql_noexec") '()))
	(map (filter module_keys (lambda (key) (not (equal? key "__tsql_temporary_scope"))))
		(lambda (key) (local key (parent key))))
	/* Error-reporting procedures inherit the enclosing immutable CATCH context.
	A nested CATCH still installs and restores only this invocation's fields. */
	(define error_context (parent "__tsql_error_context"))
	(if error_context (tsql_install_error_context local error_context) true)
	/* Parameters are lexical batch/procedure locals; account, transaction and
	connection state remain owned by the original session. Nested invocations
	do not overwrite a caller's local variables or identity owner. */
	(lambda args (match args
		'() (merge_unique (list (filter (local) (lambda (key)
			(not (has? '("__tsql_scope" "__tsql_connection_session" "__tsql_table_variables") key)))) (filter (parent) (lambda (key)
				(not (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
					(or (has? '("__tsql_scope" "__tsql_connection_session" "__tsql_local_relations" "__tsql_table_variables" "__tsql_ansi_nulls_locked") key) (strlike key "__tsql_error_%"))
					(and owner_schema (equal? key "schema"))
					(has? module_keys key)))))))
		'(key) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session" "__tsql_local_relations" "__tsql_table_variables" "__tsql_ansi_nulls_locked") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))
			(has? module_keys key)) (local key) (parent key))
		'(key value) (if (or (strlike key "__tsql_parameter:%") (equal? key "__tsql_parameter_declarations")
			(or (has? '("__tsql_scope" "__tsql_connection_session" "__tsql_local_relations" "__tsql_table_variables" "__tsql_ansi_nulls_locked") key) (strlike key "__tsql_error_%"))
			(and owner_schema (equal? key "schema"))
			(has? module_keys key)) (local key value) (parent key value))
		_ (apply parent args))))))
/* Table variables are lexical native relations. Only immutable SQL declarations
are visible to planning; native scopes/handles stay in the non-enumerated frame
controller. Its lifetime is the batch/module, independent of caller decisions. */
/* Declaration publication is lexical metadata. Native allocation happens only
when an executing command asks for the handle; compile-only DECLARE can thus
bind later columns without constructing a relation or a lifetime capability. */
(define tsql_table_variable_declaration (lambda (database name columns compile_only) (begin
	(define planning (if compile_only (newsession) nil))
	(if planning (planning "__tsql_compile_only" true) true)
	/* Keep the bound type/default declaration. Native allocation must not
	reparse SQL or run a skipped DEFAULT conversion when the table is read.
	Deferred defaults already have their actual INSERT conversion owner. */
	(append (tsql_relational_table_declaration database name columns planning) database))))
(define tsql_register_table_variable (lambda (name columns session) (begin
	(define declarations (coalesceNil (session "__tsql_local_relations") '()))
	(define previous (find declarations (lambda (declaration) (equal? (car declaration) name)) nil))
	(if previous previous (begin
		(define declaration (tsql_table_variable_declaration (session "schema") name columns (session "__tsql_noexec")))
		(session "__tsql_local_relations" (append declarations declaration)) declaration)))))
(define tsql_materialize_table_variable (lambda (session declaration tx) (begin
	(define owner (session "__tsql_table_variables"))
	(define scope (owner "get_or_compute_scoped" "table-variables" "native-lifetime" (lambda ()
		(new_relation_scope (tsql_connection_session session) nil "independent"))))
	(owner "scope" scope)
	(define physical_name (concat ".table-variable:" (uuid)))
	(define database (nth declaration 4))
	(createtable database physical_name (nth declaration 2)
		(list "engine" "memory" "ephemeral" true "scope" scope) false tx)
	(define handle (table database false physical_name false tx))
	(owner (concat "table:" (car declaration)) handle)
	handle)))
(define tsql_table_variable_handle (lambda (session name tx) (begin
	(define owner (session "__tsql_table_variables"))
	(define handle (if owner (owner (concat "table:" name)) nil))
	(if handle handle (begin
		(define declaration (find (session "__tsql_local_relations")
			(lambda (declaration) (equal? (car declaration) name)) nil))
		(if declaration (tsql_materialize_table_variable session declaration tx)
			(error "undeclared local table variable: @" name)))))))
(define tsql_bind_table_variable_inputs (lambda (formula declarations)
	(if (empty_list? declarations) formula (begin
		/* Use the compiler's lexical owner, including quote and local shadowing.
		Unknown syntax callees conservatively retain candidates: this is not a
		proof that every retained binding will execute or require allocation. */
		(define referenced (expression_referenced_bindings formula (map declarations cadr)))
		(define declarations (filter declarations (lambda (declaration)
			(has? referenced (cadr declaration)))))
		(if (empty_list? declarations) formula
			(cons (list 'lambda (map declarations cadr) formula)
				(map declarations (lambda (declaration)
					(list 'tsql_table_variable_handle 'session (car declaration) 'tx)))))))))
(define tsql_with_table_variables (lambda (session body) (begin
	(define outcome (try (lambda () (list true (body))) (lambda (failure) (list false failure))))
	(define owner (session "__tsql_table_variables"))
	(define scope (if owner (owner "scope") nil))
	/* The whole producer has returned before retirement. Never register this
	capability as a transaction resource: ROLLBACK does not end SQL lifetime. */
	(if scope (close_relation_scope scope) true)
	(if (car outcome) (cadr outcome) (error (cadr outcome))))))
(define tsql_run_declare_table (lambda (name columns statement session)
	(tsql_statement_call statement session "DECLARE table variable" false (lambda (tx) (begin
		(tsql_require_database_access (tsql_session_actor session) (session "schema"))
		(tsql_register_table_variable name columns session)
		(if (session "__tsql_fmtonly") true (tsql_table_variable_handle session name tx))
		(tsql_no_count_completion))))))
/* Script recipes retain their validated source. Native INSERT parses/plans once
per SQL statement and consumes the same streaming sink as an ordinary target. */
(define tsql_run_insert_local (lambda (name columns query statement session resultrow resultfields) (begin
	(define text (concat "INSERT INTO @" name
		(if columns (concat "(" (reduce (map columns (lambda (name)
			(concat "[" (replace name "]" "]]") "]")))
			(lambda (text column) (concat text (if (equal? text "") "" ",") column)) "") ")") "") " " query))
	(tsql_statement_call statement session text false (lambda (tx)
		(tsql_execute_statement tx (session "schema") text session resultrow resultfields))))))
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

/* The native recovery flag is a capability, not a read-only SQL guess. Only
these parsed diagnostic values and their Boolean composition can cross an
aborted transaction's statement boundary. Unknown functions, variables,
relations and assignment targets retain ordinary native admission.
https://learn.microsoft.com/sql/t-sql/language-elements/try-catch-transact-sql */
(define tsql_recovery_value? (lambda (expression) (match expression
	((symbol sql_type_expr) value _type _collation) (tsql_recovery_value? value)
	((symbol session) key) (has? '("transaction" "__tsql_transaction_count"
		"__tsql_native_transaction_state" "__tsql_error_number" "__tsql_error_message"
		"__tsql_error_state" "__tsql_error_severity") key)
	((symbol sql_compile_expr) ((symbol tsql_comparison_compiler) operation) left right)
	(and (has? '("eq" "ne" "lt" "gt" "le" "ge") operation)
		(tsql_recovery_value? left) (tsql_recovery_value? right))
	(cons head arguments)
	(and (symbol? head) (has? '(if coalesceNil equal? not and or sql_and sql_or sql_not nil?) head)
		(reduce arguments (lambda (valid value) (and valid (tsql_recovery_value? value))) true))
	_ (or (number? expression) (string? expression) (nil? expression)
		(expression_equal? expression true) (expression_equal? expression false)))))
(define tsql_recovery_required? (lambda (session)
	(and (or (session "__tsql_error_context") (session "transaction")) (begin
		(define state (tx_state ((tsql_connection_session session) "__memcp_tx")))
		(and (not (equal? state "deciding"))
			(or (session "__tsql_error_context") (equal? state "aborted")))))))
(define tsql_recovery_syntax (lambda (schema text session expression) (begin
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" (session "__tsql_parameter_declarations"))
	(planning "__tsql_quoted_identifier" (tsql_quoted_identifier session))
	(planning "__tsql_ansi_nulls" (tsql_ansi_nulls session))
	(if expression (begin (planning "__tsql_expression_only" true)
		(planning "__tsql_expression_syntax" true)) (planning "__tsql_logical_query" true))
	(parse_tsql schema text (lambda args (error "recovery diagnostics require a relation-free expression")) planning nil))))
(define tsql_recovery_expression? (lambda (schema text session)
	(and (tsql_recovery_required? session) (try
		(lambda () (tsql_recovery_value? (tsql_recovery_syntax schema text session true)))
		(lambda (_failure) false)))))
(define tsql_recovery_select? (lambda (schema text session)
	(and (tsql_recovery_required? session) (try
		(lambda () (match (tsql_recovery_syntax schema text session false)
			((symbol query-block) _database sources fields condition grouping having ordering limit offset hidden stages facts)
			(and (empty_list? sources) (empty_list? (coalesceNil grouping '())) (nil? having)
				(empty_list? (coalesceNil ordering '())) (nil? limit) (nil? offset)
				(empty_list? hidden) (empty_list? stages)
				(not (qassoc_get facts 'tsql_select_assignments nil))
				(tsql_recovery_value? condition)
				(reduce (produceN (/ (count fields) 2)) (lambda (valid index)
					(and valid (tsql_recovery_value? (nth fields (+ (* index 2) 1))))) true))
			_ false)) (lambda (_failure) false)))))
(import "tsql-scalar-functions.scm")
(import "tsql-relational-scalar-functions.scm")
(import "tsql-table-functions.scm")

/* A durable view contains names and argument syntax, not the function snapshot
which happened to exist at CREATE VIEW. Bind executable nodes only; declarations
and query facts remain data. The caller's normal source guards own warm reuse. */
(define tsql_bind_view_function_references (lambda (node policy planning tx) (match node
	((symbol tsql_view_xml_nodes) database document path alias column)
	(tsql_xml_nodes_query database
		(tsql_bind_view_function_references document policy planning tx) path alias column)
	((symbol tsql_view_table_call) database name arguments alias)
	(tsql_table_function_call database name
		(map arguments (lambda (argument) (tsql_bind_view_function_references argument policy planning tx)))
		alias policy planning tx)
	((symbol tsql_view_scalar_call) database name arguments)
	(begin
		(define arguments (map arguments (lambda (argument)
			(tsql_bind_view_function_references argument policy planning tx))))
		(if (planning "__sql_view_declaration_analysis")
			(tsql_scalar_function_declaration_call database name arguments policy planning tx)
			(tsql_scalar_function_call database name arguments policy planning tx false)))
	_ (tsql_map_executable_children node (lambda (child)
		(tsql_bind_view_function_references child policy planning tx))))))
(define tsql_expand_view_functions (lambda (query policy planning tx) (begin
	(define planning (if (nil? planning) (newsession) planning))
	(define query (sql_bind_catalog_declarations query tx planning))
	(if (planning "__tsql_compile_only")
		(if (empty_list? (tsql_compile_relation_dependencies query tx))
			(tsql_bind_column_compilers (tsql_compile_function_references query policy planning tx)
				(tsql_column_compiler planning)) query)
		(begin
			(define bound (tsql_bind_column_compilers
				(tsql_bind_view_function_references query policy planning tx) (tsql_column_compiler planning)))
			(if (planning "__sql_view_declaration_analysis") bound
				(tsql_bind_column_compilers (tsql_expand_relational_scalar_calls bound '() policy planning tx)
					(tsql_column_compiler planning))))))))
(register_sql_view_expansion "tsql"
	(close_procedure (lambda (query policy planning tx) (tsql_expand_view_functions query policy planning tx))))
(define tsql_scalar_formula (lambda (schema text contracts mode target ansi tx) (begin
	(define key (json_encode (list schema text contracts mode target ansi)))
	/* Shared scalar recipes compile independently of a requesting transaction.
	The cache argument controls waiter cancellation, not factory parameters. */
	(tsql_scalar_cache key tx (lambda () (begin
		(define planning (newsession))
		(planning "__tsql_parameter_declarations" contracts)
		(planning "__tsql_expression_only" true)
		(planning "__tsql_ansi_nulls" ansi)
		(define info (parse_tsql schema text nil planning nil))
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
		/* PRINT's output width follows the declared SQL string family, never
		the Scheme carrier. The ordinary cast compiler owns conversion/truncation. */
		(define expr (if (equal? mode "print")
			(sql_type_formula '() (tsql_cast_expr
				(list (quote sql_type_expr) expr (sql_info_datatype info) (sql_info_collation info))
				(if (has? '("NVARCHAR" "NCHAR" "NTEXT") (toUpper (sql_info_type info))) "NVARCHAR" "VARCHAR")
				(list (if (has? '("NVARCHAR" "NCHAR" "NTEXT") (toUpper (sql_info_type info))) 4000 8000)))) expr))
		(define conversion_target (if (nil? target) nil (tsql_effective_parameter_contract target)))
		(define expr (if (has? '("assignment" "message-text" "message-integer") mode)
			(sql_type_formula '() (tsql_implicit_cast_expr (list (quote sql_type_expr) expr (sql_info_datatype info) (sql_info_collation info))
				(car conversion_target) (cdr conversion_target))) expr))
		(sql_queryplan_compile_formula nil (sql_queryplan_bind_tx_calls
			(sql_queryplan_bind_execution_session (sql_type_lower_annotations '() expr))))))))))
(define tsql_scalar_value (lambda (schema text session tx mode target) (begin
	(define session (tsql_cursor_input_session session))
	(tsql_bind_transaction_state session tx)
	/* Scalar control expressions use the same scope dependency as SELECT leaves.
	Restore the caller owner before shared compilation after a child procedure. */
	(session "__tsql_scope_identity" ((tsql_identity_scope session) "__tsql_scope_identity"))
	(if (equal? mode "predicate") (begin
		/* A control predicate is a one-column SQL query. EXISTS, IN and scalar
		subqueries pass through the common logical binder and decorrelator once,
		before execution; there is no recursive row-time subquery evaluator. */
		(define answer (newsession))
		(tsql_execute_statement tx schema
			(concat "SELECT CASE WHEN (" text "\n) THEN CAST(1 AS BIT) ELSE CAST(0 AS BIT) END AS __tsql_predicate")
			session (lambda (row) (begin
				(if (answer "seen") (error "control predicate produced multiple rows") true)
				(answer "seen" true)
				(answer "value" (row "__tsql_predicate")))) (lambda args true))
		(if (answer "seen") (answer "value") (error "control predicate produced no row")))
		((tsql_scalar_formula schema text (session "__tsql_parameter_declarations") mode target (tsql_ansi_nulls session) tx)
			session tx (lambda (_row) (error "scalar expression emitted rows")) (lambda args true))))))

/* DECLARE and SET already own the destination variable's complete SQL
contract. Normalize their RHS to the ordinary SELECT-assignment owner: it
binds the declared implicit conversion/collation before decorrelation and
physical planning. The outer SELECT has one scalar projection; nested query
cardinality, empty-input NULLs and source visibility remain ordinary SQL.
The current session/transaction also retain local tables, temporary names,
authority and guarded plan reuse. Shared scalar recipes remain relation-free. */
(define tsql_assign_variable_query (lambda (schema name text session tx)
	(tsql_execute_statement tx schema (concat "SELECT @" name " = (" text "\n)") session
		(lambda (_row) (error "variable assignment emitted a result row"))
		(lambda args (error "variable assignment emitted result metadata")))))
/* Alias names remain source data until this actual DECLARE. The existing
schema resolver supplies SQL base types/dimensions, never value tags. Bind only
the current declarations: an earlier variable keeps its own established type.
The invocation's normal declaration cache profile then guards every query;
neither source-recipe caches nor unrelated future DECLAREs sample this alias. */
(define tsql_bind_local_parameter_aliases (lambda (schema declarations session) (begin
	(define aliases (filter declarations (lambda (declaration) (begin
		(define type (car (cadr declaration)))
		(and (equal? (substr type 0 1) "[") (nil? (tsql_builtin_type_declaration type)))))))
	(if (empty_list? aliases) true (begin
		(tsql_require_database_access (tsql_session_actor session) schema)
		(define resolved (map aliases (lambda (declaration) (begin
			(define contract (tsql_resolve_schema_type schema (car (cadr declaration)) '() nil))
			(list (car declaration) (cons (contract "base_type") (contract "dimensions")))))))
		(session "__tsql_parameter_declarations"
			(map (session "__tsql_parameter_declarations") (lambda (declaration)
				(coalesceNil (find resolved (lambda (item) (equal? (car item) (car declaration))) nil) declaration))))
		true)))))

/* SELECT assignment consumes the ordinary planner's result stream. Retain only
one final scalar tuple, then publish it to the invoking batch/procedure scope.
No user result set is exposed and a zero-row query leaves its locals unchanged.
RHS session dependencies are bound by the shared compiler before execution;
unordered assignment reads statement inputs. An explicit ordered reducer may
instead consume one bounded previous-value tuple for each emitted row. */
(define tsql_run_select_assignment (lambda (session targets inputs tx run row_reducer) (begin
	/* The result sink owns local publication; source callbacks receive only
	this invocation's SQL input values. A lexical scope's forwarding session
	also owns temporary relations and connection capabilities, which are not
	row inputs and must not become delegated closure captures. */
	(define read_values (newsession))
	/* Authority and transaction visibility are execution bindings even when
	metadata helpers consume them indirectly. Preserve their native owners;
	neither belongs to logical SQL inputs or the lexical scope forwarding map. */
	(define actor (tsql_session_actor session))
	(read_values "username" (car actor))
	(read_values "__tsql_principal_id" (cadr actor))
	(read_values "schema" (session "schema"))
	(read_values "__memcp_tx" tx)
	(map inputs (lambda (key) (read_values key (session key))))
	/* SQL values stay in the scalar snapshot, while opaque native namespaces
	retain the request's existing owner. They are not enumerable SQL inputs. */
	(define namespace_owner (tsql_connection_session session))
	(define read_inputs (lambda args (match args
		'("get_or_compute_scoped" scope key producer)
		(namespace_owner "get_or_compute_scoped" scope key producer)
		'("get_or_compute_scoped" scope key state producer)
		(namespace_owner "get_or_compute_scoped" scope key state producer)
		_ (apply read_values args))))
	(define output (newsession))
	(define fold (if row_reducer (row_reducer read_inputs tx (lambda (_row) true) (lambda args true)) nil))
	(if fold (output "last" (merge (mapIndex targets (lambda (index name)
		(list (tsql_select_assignment_slot index) (session (concat "__tsql_parameter:" name))))))) true)
	(define result (run (lambda (row) (begin
		(output "last" (if fold (fold (output "last") row) row))
		(output "seen" true) 1)) read_inputs))
	(define row (output "last"))
	/* Publish each variable once after the stream completes. Repeated targets
	use their final slot here; this chooses one permitted outcome, not a SQL
	promise about expression evaluation order. The mapping is syntax-sized. */
	(if (output "seen") (extract_assoc (tsql_select_assignment_positions targets) (lambda (target index)
		(session (concat "__tsql_parameter:" target) (row (tsql_select_assignment_slot index))))) true)
	(if (number? result) result (error "SELECT assignment requires its completed native reduction")))))
/* Labels name source continuations, not SQL relations. Lower structured control
once per entry into a batch or TRY/CATCH region. The resulting source-sized graph
contains no plans, rows, transaction handles or invocation sessions. Ordinary SQL
leaves still execute through the existing statement owner and command dispatcher. */
/* One source graph retains two continuations. Execution preserves its chosen
branch and loop backedges; compilation visits each lexical node once without
running predicates or transfers. Both continuations share the actual commands,
labels, declaration scope and existing statement dispatcher. */
(define tsql_flow_bind (lambda (state id command next compile_next loop)
	(list id (set_assoc (cadr state) (string id) (list command next loop compile_next))
		(nth state 2) (nth state 3) id)))
(define tsql_flow_emit (lambda (state command next compile_next loop)
	(tsql_flow_bind (list nil (cadr state) (nth state 2) (+ (nth state 3) 1) nil)
		(nth state 3) command next compile_next loop)))
(define tsql_flow_commands (lambda (commands next compile_next loop state)
	(reduce (reverse commands) (lambda (state command)
		(tsql_flow_command command (car state) (nth state 4) loop state))
		(list next (cadr state) (nth state 2) (nth state 3) compile_next))))
(define tsql_flow_command (lambda (command next compile_next loop state) (match command
	'("block" body) (tsql_flow_commands body next compile_next loop state)
	'("label" name) (begin
		/* Retaining the label node keeps distinct continuations even when a
		label ends an IF arm or an empty ordinary block. */
		(define node (tsql_flow_emit state command next compile_next loop))
		(list (car node) (cadr node) (set_assoc (nth node 2) name (car node))
			(nth node 3) (nth node 4)))
	'("if" condition yes no) (begin
		(define no_flow (if no (tsql_flow_command no next compile_next loop state)
			(list next (cadr state) (nth state 2) (nth state 3) compile_next)))
		(define yes_flow (tsql_flow_command yes next (nth no_flow 4) loop no_flow))
		(tsql_flow_emit yes_flow (list "branch" condition (car yes_flow) (car no_flow) false)
			nil (nth yes_flow 4) loop))
	'("while" condition body) (begin
		(define header (nth state 3))
		(define reserved (list nil (cadr state) (nth state 2) (+ header 1) nil))
		(define body_flow (tsql_flow_command body header compile_next (list next header) reserved))
		(define header_flow (tsql_flow_bind body_flow header
			(list "branch" condition (car body_flow) next true) nil (nth body_flow 4) loop))
		(tsql_flow_emit header_flow (list "reset") header header loop))
	_ (tsql_flow_emit state command next compile_next loop))))
(define tsql_run_commands (lambda (commands statement session resultrow resultfields)
	(if ((tsql_identity_scope session) "__tsql_transfer") 0
		(empty_list? commands) 0
		/* Ordinary one-statement queries need neither a control graph nor another
		source-cache key. Keep their existing direct statement boundary. */
		(and (not (session "__tsql_noexec")) (equal? (count commands) 1) (has? '("statement" "recovery") (car (car commands))))
		(tsql_run_command (car commands) statement session resultrow resultfields)
		(tsql_run_flow commands statement session resultrow resultfields))))
(define tsql_run_flow (lambda (commands statement session resultrow resultfields) (begin
	/* Reuse the existing bounded source cache. The key and value describe only
	syntax; compilation never captures this call's SQL or native execution state. */
	(define flow (tsql_script_cache (json_encode (list "control-flow" commands)) nil
		(lambda (_tx) (tsql_flow_commands commands nil nil nil (list nil (list) (list) 0 nil)))))
	(tsql_run_bound_flow flow (car flow) statement session resultrow resultfields))))
(define tsql_run_bound_flow (lambda (flow start statement session resultrow resultfields) (begin
	(define nodes (cadr flow))
	(define labels (nth flow 2))
	(define scope (tsql_identity_scope session))
	(cadr (for (list start 0)
		(lambda (pc _result) (not (nil? pc)))
		(lambda (pc _previous) (begin
			(define node (nodes (string pc)))
			(define command (car node))
			(define loop (nth node 2))
			(define compiling (session "__tsql_noexec"))
			(if compiling (tx_check ((tsql_connection_session session) "__memcp_tx")) true)
			(define step (if compiling
				(list (cadr node) (match command
					'("set-options" options enabled) (begin
						/* While ON only this mode's OFF transition executes. Other
						options in a mixed OFF setter resume their ordinary owner. */
						(if (and (has? options "NOEXEC") (not enabled))
							(tsql_run_command command statement session resultrow resultfields)
							(begin (tsql_noexec_compile_command command session) 0)))
					'("try" body recovery) (tsql_run_try body recovery statement session resultrow resultfields)
					_ (begin (tsql_noexec_compile_command command session) 0)))
				(match command
					'("label" _name) (list (cadr node) _previous)
					'("branch" condition yes no preserve_result) (begin
						(define answer (tsql_with_error_status session (lambda ()
							(tsql_statement_call statement session condition (tsql_recovery_expression? (session "schema") condition session) (lambda (tx)
								(tsql_scalar_value (session "schema") condition session tx "predicate" nil))))))
						(session "__tsql_rowcount" 0)
						(list (if answer yes no) (if preserve_result _previous 0)))
					'("reset") (list (cadr node) 0)
					'("trigger-rollback") (list nil (tsql_trigger_rollback_at flow (cadr node) statement session))
					_ (list (cadr node) (tsql_run_command command statement session resultrow resultfields)))))
			(define next (if (session "__tsql_noexec") (nth node 3) (match (scope "__tsql_transfer")
				'("goto" label) (if (has_assoc? labels label)
					(begin (scope "__tsql_transfer" nil) (labels label)) nil)
				"break" (if loop (begin (scope "__tsql_transfer" nil) (car loop)) nil)
				"continue" (if loop (begin (scope "__tsql_transfer" nil) (cadr loop)) nil)
				nil (car step)
				_ nil)))
			(list next (cadr step)))))))))
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
/* Diagnostics inspect the language value only after the native statement has
finished its decision. THROW retains the raw cause, including completion facts. */
(define tsql_diagnostic_failure_value (lambda (cause)
	(match (failure_value cause)
		'("tsql-compile-error" _scope original) (tsql_diagnostic_failure_value original)
		'("tsql-user-error" original) (tsql_diagnostic_failure_value original)
		'("cardinality-error" message) (list "tsql-error" 512 message 1 16)
		'("constraint-violation" "check" _name message) (list "tsql-error" 547 message 1 16)
		'("constraint-violation" "unique-declaration" _name message) (list "tsql-error" 1505 message 1 16)
		'("constraint-violation" "not-null-declaration" _name message) (list "tsql-error" 515 message 1 16)
		'("column-declaration-conflict" "name" table column)
		(list "tsql-error" 2705 (concat "Column names in each table must be unique. Column name '" column "' in table '" table "' is specified more than once.") 1 16)
		'("column-property-conflict" table _column "tsql.rowguidcol" true _owner)
		(list "tsql-error" 4925 (concat "ALTER TABLE ALTER COLUMN ADD ROWGUIDCOL failed because a column already exists in table '" table "' with ROWGUIDCOL property.") 1 16)
		value (if (transaction_serialization_failure? cause)
			/* Optimistic validation uses SQL Server's in-memory conflict
			diagnostic; the native owner has already required full rollback.
			https://learn.microsoft.com/sql/relational-databases/errors-events/mssqlserver-41325-database-engine-error */
			(list "tsql-error" 41325 "The current transaction failed to commit due to a serializable validation failure." 1 17)
			(transaction_rollback_only_failure? cause)
			(list "tsql-error" 3930 "The transaction is uncommittable; only reads and full rollback are allowed." 1)
			value))))
(define tsql_error_context (lambda (cause)
	(match (tsql_diagnostic_failure_value cause)
		'("tsql-error" number message state severity)
		(list "error" cause "number" number "message" message "state" state "severity" severity)
		'("tsql-error" number message state)
		(list "error" cause "number" number "message" message "state" state "severity" 16)
		_ (begin
			(define diagnostic (failure_diagnostic cause))
			/* Preserve the actual backend code. SQL Server number mappings
			require their own declared-category proof, never a message match. */
			(list "error" cause "number" (if diagnostic (diagnostic "code") nil)
				"message" (if diagnostic (diagnostic "message") (string cause)) "state" nil)))))
/* @@ERROR is the last executed statement's status, not the lexical CATCH
context used by ERROR_NUMBER(). A one-slot immutable value distinguishes the
initial absent status (0) from an opaque failure with no numeric diagnostic;
the latter must never be reported as a successful zero. The raw failure still
propagates through its existing completion/handler owner.
https://learn.microsoft.com/sql/t-sql/functions/error-transact-sql */
(define tsql_publish_error_status (lambda (session failure)
	(session "__tsql_last_error" (list (if (nil? failure) 0
		(tsql_error_value (tsql_error_context failure) "number"))))))
(define tsql_with_error_status (lambda (session body)
	(try (lambda () ((lambda (result) (begin
		/* A borrowed trigger/module leaf can return with a diagnostic pending
		at its actual outer native owner; do not turn that into a success. */
		(define diagnostic (session "__tsql_statement_error"))
		(tsql_publish_error_status session (if diagnostic (diagnostic) nil))
		result)) (body)))
		(lambda (failure) (begin
			(tsql_publish_error_status session failure)
			(error failure))))))
(define tsql_run_try (lambda (body recovery statement session resultrow resultfields)
	(if (session "__tsql_noexec")
		(tsql_noexec_compiled_try body recovery statement session resultrow resultfields)
		(begin
			/* This TRY was actually entered. Its native error/recovery owner is
			unchanged; a compiled CATCH has no fabricated runtime error context. */
			(tsql_publish_error_status session nil)
			(session "__tsql_rowcount" 0)
			(define outcome (try
				(lambda () (list false (tsql_run_commands body statement (tsql_try_session session) resultrow resultfields)))
				(lambda (err) (if (or (transaction_abort_transfer? err)
					(tsql_noexec_same_level_failure? err session)) (error err) (begin
						(tsql_statement_call statement session "catch boundary" true (lambda (_tx) true))
						(session "__tsql_rowcount" 0)
						(define previous (session "__tsql_error_context"))
						(tsql_install_error_context session (tsql_error_context err))
						(try (lambda ()
							((lambda (result) (begin
								(tsql_install_error_context session previous)
								(list true result)))
								(tsql_run_commands recovery statement session resultrow resultfields)))
							(lambda (cause) (begin (tsql_install_error_context session previous) (error cause)))))))))
			(if ((tsql_identity_scope session) "__tsql_transfer") true (session "__tsql_rowcount" 0))
			(if (and (not (car outcome)) (session "__tsql_noexec"))
				(tsql_noexec_compiled_region recovery statement session resultrow resultfields) (cadr outcome))))))
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
/* Bind and format once through the ordinary typed scalar owner. Statement
trigger execution can choose its error transfer without rebinding arguments. */
(define tsql_bind_raiserror (lambda (arguments statement session) (begin
	(if (and (>= (count arguments) 3) (<= (count arguments) 23)) true
		(error "RAISERROR requires message, severity, state and at most twenty substitutions"))
	(tsql_statement_call statement session "error message arguments" false (lambda (tx) (begin
		(define schema (session "schema"))
		(define text (tsql_scalar_value schema (car arguments) session tx "dynamic" nil))
		(define severity (tsql_scalar_value schema (cadr arguments) session tx "assignment" '("INT")))
		(define state (tsql_scalar_value schema (nth arguments 2) session tx "assignment" '("INT")))
		/* Fatal severities require a separate connection-termination contract. */
		(if (and (not (nil? severity)) (>= severity 0) (<= severity 18)) true
			(error "RAISERROR currently supports severity zero through eighteen"))
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
		(list "tsql-error" 50000 (car formatted) (if (< state 0) 1 state) severity)))))))
(define tsql_run_raiserror (lambda (arguments statement session) (begin
	(define raised (tsql_bind_raiserror arguments statement session))
	/* Raising after scalar binding keeps a frontend message from marking a
	successful native transaction statement as failed. RAISERROR is not DML. */
	(if (<= (nth raised 4) 10)
		(tsql_emit_message session (nth raised 1) (if (equal? (nth raised 4) 10) 0 (nth raised 4))
			(nth raised 3) (nth raised 2))
		(error (list "tsql-user-error" raised))))))
/* Resolve once at statement admission. Session-local relations can share this
binding boundary without changing the procedure executor or native row sink. */
(define tsql_insert_exec_target (lambda (owner name session tx) (begin
	(match name
		'("table-variable" variable) (begin
			/* Reuse the declaration binding used by ordinary local INSERT. The
			third field is a lexical name, not a retained native capability. */
			(define source (tsql_relational_table_source (session "schema") variable nil session))
			(list (source_schema source) (source_relation source) variable))
		_ (begin
			(define database (tsql_qualifier_database owner (session "schema") session))
			(tsql_resolve_relation database name session tx))))))
/* Result descriptors contain the SQL compiler's declaration, independently of
row carriers. Compile each result set's assignment vector once; only its plain
formula reaches the existing native INSERT. No relational list is retained. */
(define tsql_insert_exec_sink (lambda (names descriptors declarations columns identity target session tx database target_name) (begin
	(define policy (tsql_policy (tsql_session_actor session)))
	(define planning (newsession))
	(if (and (equal? (count names) (count declarations))
		(equal? (count descriptors) (count declarations))) true
		(error "INSERT EXEC result column count does not match target"))
	(define values (cons (quote list) (map (produceN (count declarations)) (lambda (index) (begin
		(define field (nth descriptors index))
		(define type (coalesceNil (field "sql_datatype") (field "sql_type")))
		(if type true (error "INSERT EXEC source has no declared SQL type"))
		(sql_type_formula '() (tsql_assignment_expr
			(list (quote sql_type_expr) (list (quote nth) (quote row) (+ 1 (* 2 index)))
				type (field "sql_collation")) (nth declarations index))))))))
	(define formula (tsql_native_insert_expr (quote target) identity columns
		(list (quote list) values) 1 false (tsql_insert_deferred_columns database target_name columns policy planning tx)
		(tsql_write_declarations database target_name planning tx)))
	(define compiled (eval (list (quote lambda)
		(list (quote row) (quote session) (quote tx) (quote target))
		(sql_type_lower_annotations '() formula))))
	(lambda (row) (compiled row session tx target)))))
/* A bulk command retains only its authorized relation and bounded declaration
vector. The later wire request rechecks authority and generation. Neither a
transaction, prepared sink nor a callback survives the command boundary. */
(define tsql_bulk_check_triggers (lambda (target options tx) (begin
	/* The native transaction retains the complete declaration set, including
	hidden/private programs, before any source row or default can execute. */
	(define declarations (retain_table_trigger_declarations target tx))
	/* The initial durable CHECK profile has no unchecked/untrusted public
	bulk state. Reject the option before consuming rows instead of ignoring it. */
	(if (and (not (has? options "CHECK_CONSTRAINTS"))
		(not (empty_list? (coalesceNil ((show target "constraints" tx) "Checks") '()))))
		(error "unchecked bulk into a CHECK table is not supported") true)
	/* The current TDS Bulk frontdoor has no diagnostic sink. It cannot attest
	statement-trigger error/control completion, with or without FIRE_TRIGGERS.
	Reject before consuming the first row; do not change ordinary bulk targets
	or pretend that the SQL Server default runs statement INSERT triggers. */
	(if (find declarations (lambda (trigger)
		(and (equal? (trigger "timing") "after_statement") (has? (trigger "events") "insert"))) nil)
		(error "bulk statement INSERT triggers require a supported diagnostic completion sink") true)
	(if (or (has? options "FIRE_TRIGGERS")
		(nil? (find declarations (lambda (trigger)
			(or (has? '("before_insert" "after_insert") (trigger "timing"))
				(and (equal? (trigger "timing") "after_statement") (has? (trigger "events") "insert")))) nil))) true
		(error "bulk import into a table with INSERT triggers requires FIRE_TRIGGERS")))))
(define tsql_prepare_bulk (lambda (database name columns options session tx) (begin
	(if (session "wire_metadata") true (error "INSERT BULK requires a TDS connection"))
	(if (session "__tsql_bulk_target") (error "a bulk stream is already pending") true)
	(define declarations (tsql_insert_declarations database name (map columns car) nil tx))
	(define contracts (map columns (lambda (column) (begin
		(define type (cadr column))
		(if (has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "BIT" "BOOLEAN" "BOOL"
			"FLOAT" "REAL" "DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY" "UNIQUEIDENTIFIER"
			"DATE" "TIME" "DATETIME" "SMALLDATETIME" "DATETIME2" "VARCHAR" "CHAR" "NVARCHAR" "NCHAR"
			"BINARY" "VARBINARY") type) true (error "unsupported bulk source SQL type: " type))
		(list type (nth column 2) (nth column 3))))))
	(define target (table database false name false tx))
	(session "__tsql_bulk_target" (list "database" database "name" name "target" target
		"columns" (map declarations (lambda (column) (column "Field"))) "contracts" contracts
		"declarations" (map declarations tsql_declaration_contract) "options" options))
	0)))
(define tsql_bulk_wire_matches? (lambda (wire contract) (begin
	(define expected (tsql_wire_contract (car contract) (cadr contract)))
	/* DECIMALN and NUMERICN carry the same coefficient declaration. Clients
	may normalize these SQL aliases and use any codec-validated coefficient
	buffer. Precision and scale remain the exact SQL source declaration. */
	(define kind_matches (or (equal? (wire "kind") (expected "kind"))
		(and (has? '(106 108) (wire "kind")) (has? '(106 108) (expected "kind")))))
	/* DATE and modern clock tokens transmit their kind/scale, with no size
	field in COLMETADATA. Their payload widths are validated by the codec;
	compare only declaration fields that actually exist on this wire shape. */
	/* MONEYN transmits only its byte width. Its precision and scale are
	implicit SQL properties, unlike DECIMALN's explicit declaration fields. */
	(define keys (if (equal? (wire "kind") 110) '("size")
		(has? '(40 41 42 43 106 108) (wire "kind")) '("scale" "precision")
		'("size" "scale" "precision")))
	(define numeric (has? '(106 108) (wire "kind")))
	(define size_matches (if numeric
		(and (has? '(5 9 13 17) (wire "size"))
			(>= (wire "size") (tsql_declared_byte_length
				(list "RawType" (car contract) "Dimensions" (cadr contract))))) true))
	(and kind_matches size_matches (reduce keys (lambda (valid key)
		(and valid (equal? (coalesceNil (wire key) 0) (coalesceNil (expected key) 0)))) true)))))
/* Compile once from the SQL declaration and wire schema, then pass only plain
values into the existing native INSERT. Every row is consumed immediately;
the storage engine owns rollback, partitioning, constraints and durability. */
(define tsql_bulk_sink (lambda (pending wires declarations target session tx) (begin
	(define policy (tsql_policy (tsql_session_actor session)))
	(define planning (newsession))
	(define columns (pending "columns"))
	(define contracts (pending "contracts"))
	(if (equal? (count wires) (count columns)) true (error "bulk column count differs from INSERT BULK"))
	(define values (cons (quote list) (map (produceN (count columns)) (lambda (index) (begin
		(define wire (nth wires index))
		(define contract (nth contracts index))
		(define declaration (nth declarations index))
		(if (and (equal?? (wire "name") (nth columns index)) (tsql_bulk_wire_matches? wire contract))
			true (error "bulk wire declaration differs from INSERT BULK for " (nth columns index)
				": expected " (list (car contract) (cadr contract)) ", received TDS kind/size/precision/scale "
				(list (wire "kind") (wire "size") (wire "precision") (wire "scale"))))
		(define expression (list (quote sql_type_expr)
			(list (quote tds_parameter_value) (list (quote list) "type" (list (quote quote) wire)
				"value" (list (quote nth) (quote row) index)))
			(if (empty_list? (cadr contract)) (car contract) (list (car contract) (cadr contract)))
			(if (nth contract 2) (list (nth contract 2) 0) nil)))
		(define assigned (tsql_assignment_expr expression declaration))
		(define has_default (or (not (nil? (declaration "Default")))
			(not (equal? (coalesceNil (declaration "DefaultExpression") "") ""))))
		(sql_type_formula '() (if (and has_default (not (has? (pending "options") "KEEP_NULLS")))
			(list (quote if) (list (quote nil?) (list (quote nth) (quote row) index))
				(tsql_assignment_expr (tsql_default_assignment_expr declaration (pending "database") policy planning tx) declaration) assigned)
			assigned)))))))
	(define identity (tsql_insert_identity (pending "database") (pending "name") nil tx))
	(define formula (tsql_native_insert_expr (quote target) identity columns (list (quote list) values) 1 false
		(tsql_insert_deferred_columns (pending "database") (pending "name") columns policy planning tx)
		(tsql_write_declarations (pending "database") (pending "name") planning tx)))
	(define compiled (eval (list (quote lambda) (list (quote row) (quote session) (quote tx) (quote target))
		(sql_type_lower_annotations '() formula))))
	(lambda (row) (compiled row session tx target)))))
(define tsql_run_bulk_value (lambda (session payload state sequence) (begin
	(define pending (session "__tsql_bulk_target"))
	(session "__tsql_bulk_target" nil)
	(if pending true (error "bulk data has no preceding INSERT BULK command"))
	(with_query_request (tsql_connection_session session) state sequence "bulk import" "acid" (lambda (statement)
		(tsql_statement_call statement session "bulk import" false (lambda (tx) (begin
			(define database (pending "database"))
			(define name (pending "name"))
			((tsql_policy (tsql_session_actor session)) database name true)
			(tsql_require_permission (tsql_session_actor session) database
				(if (sql_private_relation? name) "CONNECT" "INSERT"))
			(define target (table database false name false tx))
			(if (and target (identical? target (pending "target"))) true (error "bulk target was dropped or replaced"))
			(define declarations (tsql_insert_declarations database name (pending "columns") nil tx))
			(if (expression_equal? (map declarations tsql_declaration_contract) (pending "declarations")) true
				(error "bulk target declarations changed"))
			(tsql_bulk_check_triggers target (pending "options") tx)
			(with_table_mutation (list target) (list "insert" '()) tx session (lambda (tx) ((payload "consume") (tsql_bulk_sink pending (payload "columns") declarations target session tx))))))))))))
(define tsql_run_bulk (lambda (session payload state sequence)
	(tsql_with_error_status session (lambda ()
		(tsql_with_statement_diagnostic session (lambda (session)
			(tsql_run_bulk_value session payload state sequence)))))))
/* Only declared SQL diagnostics become producer outcomes. Arbitrary native
failures, terminal transfers and cancellation retain their original unwinding
owner. Native integrity-constraint diagnostics are admitted separately at the
actual independent producer boundary. */
(define tsql_independent_mutation_producer_outcome (lambda (body)
	(try (lambda () (list true (body))) (lambda (failure)
		(match (failure_value failure)
			'("tsql-error" number _text _state)
			(if (or (>= number 50000) (has? '(8152 2628) number)) (list false failure) (error failure))
			'("tsql-error" number _text _state severity)
			(if (and (<= severity 18) (or (>= number 50000) (has? '(8152 2628) number)))
				(list false failure) (error failure))
			_ (error failure))))))
/* A native data exception is not by itself proof of a committable SQL
transaction. This initial profile covers integrity failures and bounded-text
truncation. Conversion failures such as 245/8114 additionally require the
separate doomed-but-readable transaction contract; never pretend it is writable.
SQL Server 2022 reference: https://dbfiddle.uk/qgw7v_EK */
(define tsql_independent_mutation_committable? (lambda (outcome) (begin
	(define diagnostic (nth outcome 2))
	(if (nil? diagnostic) true
		(or (equal? (diagnostic "sqlstate") "22001")
			(equal? (substr (diagnostic "sqlstate") 0 2) "23"))))))
/* The public statement returns normally only for this frontend's proven
committable profile. Its original diagnostic is raised after native completion;
autocommit and unproved dispositions retain ordinary failure unwinding. */
(define tsql_independent_mutation_result (lambda (outcome session explicit)
	(if (car outcome) (cadr outcome)
		(and explicit (tsql_independent_mutation_committable? outcome)) (begin
			(define diagnostic (session "__tsql_statement_error"))
			(if diagnostic (begin (diagnostic (tsql_profile_failure session (cadr outcome))) 0)
				(error "independent mutation failure requires its statement diagnostic owner")))
		(error (cadr outcome)))))
(define tsql_run_independent_dml (lambda (target descriptor tx session body) (begin
	/* A source module inside INSERT EXEC has a borrowed statement executor.
	Its own CATCH must see a local failure immediately; only the enclosing
	capture may defer a diagnostic past the actual public request boundary. */
	(define explicit (if (session "__tsql_insert_exec") false (not (nil? (session "transaction")))))
	(define outcome (with_independent_mutation_outcome (list target) descriptor tx session
		(lambda (sink _source) (tsql_independent_mutation_producer_outcome (lambda () (body sink))))))
	(tsql_independent_mutation_result outcome session explicit))))
(define tsql_run_insert_exec (lambda (owner name explicit_columns invocation statement session) (begin
	(if (session "__tsql_insert_exec") (error "INSERT EXEC cannot be nested") true)
	(if (has? '("execute" "execute-dynamic") (car invocation)) true
		(error "INSERT EXEC requires a procedure or dynamic invocation without a return-status assignment"))
	(define explicit (not (nil? (session "transaction"))))
	(tsql_statement_call statement session "INSERT EXEC" false (lambda (tx) (begin
		(define resolved (tsql_insert_exec_target owner name session tx))
		(define database (car resolved))
		(define target_name (cadr resolved))
		(tsql_require_permission (tsql_session_actor session) database
			(if (local_relation_reference? target_name) "CONNECT"
				(sql_private_relation? target_name) "CONNECT" "INSERT"))
		(define columns (coalesceNil explicit_columns (tsql_insert_target_columns database target_name tx)))
		(define declarations (tsql_insert_declarations database target_name columns nil tx))
		(define columns (map declarations (lambda (column) (column "Field"))))
		(define identity (tsql_insert_identity database target_name nil tx))
		(define target (if (local_relation_reference? target_name)
			(tsql_table_variable_handle session (nth resolved 2) tx)
			(table database false target_name false tx)))
		(define independent (local_relation_reference? target_name))
		(define producer (lambda (tx source_tx) (begin
			/* A table variable's independent data decision belongs to its sink.
			The EXEC source uses its own admitted scope of the caller authority;
			it may have its own local variables or ordinary public side effects.
			Persistent destinations keep their existing joined statement owner. */
			(if identity (tsql_check_identity_insert session target (not (nil? explicit_columns))
				(has? columns (identity "Field"))) true)
			(define state (newsession))
			(state "count" 0)
			(define count_lock (mutex))
			(define scoped (lambda args (match args
				'("__tsql_insert_exec") true
				'("wire_metadata") true
				_ (apply session args))))
			/* Source SQL leaves borrow this statement's transaction; they do
			not reenter the request owner or commit independently. */
			(define borrowed (lambda (_text _recovery body _failure_policy) (begin (tx_check source_tx) (body source_tx))))
			(tsql_run_command invocation borrowed scoped
				(lambda (row) (try (lambda () (begin
					(define sink (state "sink"))
					(if sink true (error "INSERT EXEC received a row before its SQL descriptor"))
					(define inserted (sink row))
					(count_lock (lambda () (state "count" (+ (state "count") inserted))))
					inserted)) (lambda (failure) (begin (state "failure" failure) (error failure)))))
				(lambda (names descriptors) (try (lambda () (state "sink"
					(tsql_insert_exec_sink names descriptors declarations columns identity target session tx database target_name)))
					(lambda (failure) (begin (state "failure" failure) (error failure))))))
			/* A source CATCH cannot turn a failed target admission into a partial
			success. Let the native owner roll back the whole statement. */
			(if (state "failure") (error (state "failure")) true)
			(if identity
				(if (has? columns (identity "Field"))
					(tsql_unresolved_identity session "explicit streaming IDENTITY insertion")
					(> (state "count") 0) (tsql_unresolved_identity session "streaming INSERT EXEC")
					(session "__tsql_identity" nil))
				(session "__tsql_identity" nil))
			(state "count"))))
		(define outcome (if independent
			(with_independent_mutation_outcome (list target) (list "insert" '()) tx session
				(lambda (tx source_tx) (tsql_independent_mutation_producer_outcome (lambda () (producer tx source_tx)))))
			(list true (with_table_mutation (list target) (list "insert" '()) tx session
				(lambda (tx) (producer tx tx))) nil)))
		(tsql_independent_mutation_result outcome session explicit)))))))
/* Procedure parameters are opaque language-owned declaration data. Legacy
source definitions retain their original bare parameter list. Option-bearing
definitions carry a versioned envelope; creator identity is provenance, never
a fabricated schema/module owner or a server administration grant. */
(define tsql_procedure_declaration? (lambda (declaration)
	(if (and (list? declaration) (not (empty_list? declaration))
		(equal? (car declaration) "__tsql_procedure_declaration"))
		(if (expression_equal? (cadr declaration) 1) true
			(error "unsupported procedure declaration version: " (cadr declaration))) false)))
(define tsql_procedure_parameters (close_procedure ((lambda (envelope?)
	(lambda (declaration) (if (envelope? declaration) (declaration "parameters") declaration)))
	tsql_procedure_declaration?)))
(define tsql_procedure_execution (close_procedure ((lambda (envelope?)
	(lambda (declaration) (if (envelope? declaration) (declaration "execute_as") (list "kind" "CALLER"))))
	tsql_procedure_declaration?)))
(define tsql_require_procedure_context (close_procedure ((lambda (execution)
	(lambda (declaration) (begin
		(define context (execution declaration))
		(if (equal? (context "kind") "CALLER") true
			(error "database execution context is not supported: " (context "kind"))))))
	tsql_procedure_execution)))
(define tsql_procedure_declare_context (lambda (declaration database session)
	(if (not (tsql_procedure_declaration? declaration)) declaration (begin
		(define creator (tsql_session_actor session))
		(define execution (declaration "execute_as"))
		(define kind (execution "kind"))
		(define principal (if (equal? kind "CALLER") nil (equal? kind "OWNER") -2
			(equal? kind "SELF") (tsql_catalog_actor_field database creator "database_id")
			(equal? kind "USER") (begin
				(define name (execution "name"))
				(if (equal?? name "dbo") 1 (begin
					(define actor (tsql_actor_value name nil))
					(tsql_require_database_access actor database)
					(tsql_catalog_actor_field database actor "database_id"))))
			(error "invalid procedure execution context")))
		(set_assoc (set_assoc (set_assoc declaration "creator" creator) "database" database)
			"execute_as" (set_assoc execution "principal_id" principal))))))
(define tsql_define_procedure (lambda (owner name parameters source replace must_exist statement session)
	(tsql_statement_call statement session "procedure definition" false (lambda (tx) (begin
		/* Stored procedure source is not yet normalized for import aliases or
		checked dependencies. Reject before catalog publication; ordinary
		procedure creation still uses its existing source compiler. */
		(if (session "tsql_import_target")
			(error "retargeted procedure definitions are not supported") true)
		(define owner (tsql_qualifier_database owner (session "schema") session))
		(tsql_require_permission (tsql_session_actor session) owner "CREATE PROCEDURE")
		(if replace (begin
			(tsql_require_permission (tsql_session_actor session) owner "ALTER")
			(if (tsql_scalar_function_reserved? owner name tx)
				(error "a scalar function already owns this object name: " name) true)) true)
		(createprocedure owner (toLower name) (tsql_procedure_language session) source
			(tsql_procedure_declare_context parameters owner session) replace tx must_exist)
		0)))))
/* SQL Server row-count state belongs to statement completion, independently
of protocol DONE_COUNT and NOCOUNT. Internal predicates, parameter binding and
describe calls are not SQL statement completions. EXEC itself preserves state;
actual statements in its called batch/module publish through this same owner.
https://learn.microsoft.com/sql/t-sql/functions/rowcount-transact-sql */
(define tsql_rowcount_command_mode (lambda (command) (match command
	'("declare" declarations) (if (find declarations (lambda (declaration)
		(not (nil? (nth declaration 2)))) nil) "one" "preserve")
	(cons kind arguments) (if
		(has? '("assign" "return" "cursor-declare" "cursor-assign-query") kind) "one"
		(has? '("cursor-fetch" "select-into" "insert-local" "insert-execute") kind) "rows"
		(has? '("set-options" "print" "raiserror" "trigger-raiserror" "cursor-open" "cursor-close" "cursor-deallocate" "trigger-rollback") kind) "zero"
		(has? '("statement" "statement-source" "recovery" "insert-local-source") kind) (begin
			(define keyword (tsql_insert_exec_leaf_kind (car arguments)))
			(if (has? '("SELECT" "WITH" "INSERT" "UPDATE" "DELETE") keyword) "rows"
				(has? '("EXEC" "EXECUTE") keyword) "preserve"
				(nil? keyword) "preserve" "zero"))
		(has? '("create-scalar-function" "alter-scalar-function" "create-table-function" "alter-table-function" "drop-function" "create-procedure" "alter-procedure" "drop-procedure"
			"create-trigger" "alter-trigger" "drop-trigger") kind) "zero"
		"preserve")
	_ "preserve")))

/* Source commands already carry their declaration kind. Do not infer it from
native results: both module DDL and a zero-row DML may return numeric zero. */
(define tsql_run_command (lambda (command statement session resultrow resultfields) (begin
	(define run_command (lambda () (begin
		/* The SELECT profile must not silently execute writes/cursor snapshots
		without their separate native target/rowset limit contract. */
		(if (and (> (coalesceNil (session "__tsql_rowcount_limit") 0) 0)
			(has? '("select-into" "insert-local" "insert-local-source" "insert-execute" "cursor-open") (car command)))
			(error "ROWCOUNT requires the native target or cursor-rowset limit contract") true)
		(define mode (tsql_rowcount_command_mode command))
		/* RAISERROR can transfer before returning an ordinary completion. Its SQL
		count reset precedes the diagnostic; native failure ownership is untouched. */
		(if (has? '("raiserror" "trigger-raiserror") (car command)) (session "__tsql_rowcount" 0) true)
		(define run (lambda (session)
			(tsql_run_command_value command statement session resultrow resultfields)))
		/* Control nodes and EXEC are not native statement owners. Their inner SQL
		leaves deliver to their nearest CATCH; INSERT EXEC instead owns one complete
		native producer and its nested module leaves inherit that same error cell. */
		(define result (if (has? '("statement" "statement-source" "recovery" "select-into" "insert-execute" "insert-local" "insert-local-source" "declare-table") (car command))
			(tsql_with_statement_diagnostic session run) (run session)))
		/* Trigger-invoked modules cannot CATCH in this native completion profile.
		Unwind only their own existing SQL flow after a child reported an error;
		throwing here would run native statement Undo instead of caller CATCH. */
		(define diagnostic (session "__tsql_statement_error"))
		(if (if (session "__tsql_trigger_event") (if diagnostic (diagnostic) false) false)
			((tsql_identity_scope session) "__tsql_transfer" "return") true)
		/* Publish only the completed statement's native result. SELECT has an
		explicit final-consumer reduction, independent of the caller's sink value;
		DML, INSERT EXEC, local insert, SELECT INTO and FETCH retain their owners. */
		(if (if diagnostic (diagnostic) false) true
			(equal? mode "one") (session "__tsql_rowcount" 1)
			(equal? mode "zero") (session "__tsql_rowcount" 0)
			(equal? mode "rows") (session "__tsql_rowcount"
				(if (number? result) result
					(error "statement result requires a completed native reduction"))) true)
		(if (has? '("create-scalar-function" "alter-scalar-function" "create-table-function" "alter-table-function" "drop-function" "create-procedure" "alter-procedure" "drop-procedure"
			"create-trigger" "alter-trigger" "drop-trigger") (car command))
			(tsql_no_count_completion) result))))
	/* Structural containers have no extra completion after their real leaves.
	EXEC is an executed statement too; its successful completion clears status. */
	(if (has? '("block" "while" "try") (car command)) (run_command)
		(tsql_with_error_status session run_command)))))
(define tsql_run_command_value (lambda (command statement session resultrow resultfields) (begin
	(define schema (session "schema"))
	(if (and (session "__tsql_insert_exec") (has? '("create-procedure" "alter-procedure" "drop-procedure") (car command)))
		(error "procedure DDL is unsupported inside INSERT EXEC") true)
	(match command
		'("set-options" options enabled) (tsql_statement_call statement session "SET options" false (lambda (_tx)
			(tsql_set_options session options enabled)))
		'("select-into" name query definitions owner) (tsql_run_select_into name query definitions owner statement session)
		'("declare-table" name columns) (tsql_run_declare_table name columns statement session)
		'("insert-local" name columns query) (tsql_run_insert_local name columns query statement session resultrow resultfields)
		'("insert-local-source" text) (tsql_statement_call statement session text false (lambda (tx)
			(tsql_execute_statement tx schema text session resultrow resultfields)))
		'("statement" text) (tsql_statement_call statement session text (tsql_recovery_select? schema text session) (lambda (tx) (begin
			(tsql_bind_transaction_state session tx)
			(tsql_execute_statement tx schema text session resultrow resultfields))))
		'("statement-source" text quoted) (tsql_statement_call statement session text (tsql_recovery_select? schema text (tsql_parse_source_session session quoted)) (lambda (tx) (begin
			(tsql_bind_transaction_state session tx)
			(tsql_execute_statement tx schema text (tsql_parse_source_session session quoted) resultrow resultfields))))
		'("recovery" text) (tsql_statement_call statement session text true (lambda (tx)
			(tsql_execute_statement tx schema text session resultrow resultfields)))
		'("trigger-rollback") (tsql_statement_call statement session "trigger ROLLBACK" true (lambda (tx)
			(request_transaction_abort tx)))
		'("trigger-raiserror" arguments returning) (tsql_run_trigger_raiserror arguments statement session returning)
		'("block" commands) (tsql_run_commands commands statement session resultrow resultfields)
		'("while" _condition _body) (tsql_run_commands (list command) statement session resultrow resultfields)
		'("try" body recovery) (tsql_run_try body recovery statement session resultrow resultfields)
		'("break") (begin ((tsql_identity_scope session) "__tsql_transfer" "break") 0)
		'("continue") (begin ((tsql_identity_scope session) "__tsql_transfer" "continue") 0)
		'("goto" name) (tsql_statement_call statement session "GOTO" false (lambda (_tx) (begin
			((tsql_identity_scope session) "__tsql_transfer" (list "goto" name)) 0)))
		'("return" text) (begin
			(define status (if (nil? text) 0
				(tsql_statement_call statement session "return status" false (lambda (tx)
					(coalesceNil (tsql_scalar_value schema text session tx "assignment" '("INT") ) 0)))))
			(define scope (tsql_identity_scope session))
			(scope "__tsql_return_status" status)
			(scope "__tsql_transfer" "return")
			status)
		'("print" text) (tsql_run_print text statement session)
		'("raiserror" arguments) (tsql_run_raiserror arguments statement session)
		'("throw" arguments) (if (nil? arguments)
			/* Rethrowing is a new THROW statement for XACT_ABORT, even when the
			original diagnostic came from RAISERROR. The original native failure
			carrier remains intact; only this leaf's language policy is distinct. */
			(tsql_statement_call statement (tsql_execution_fact_session session "__tsql_failure_origin" "runtime")
				"THROW" true (lambda (_tx) (begin
					(define context (session "__tsql_error_context"))
					(if (nil? context) (error "THROW requires an active CATCH") (error (context "error"))))))
			(tsql_statement_call statement session "throw" false (lambda (tx) (begin
				(define number (tsql_scalar_value schema (car arguments) session tx "assignment" '("INT")))
				(define message (tsql_scalar_value schema (cadr arguments) session tx "assignment" '("NVARCHAR" -1)))
				(define state (tsql_scalar_value schema (nth arguments 2) session tx "assignment" '("INT")))
				(if (or (nil? number) (< number 50000) (nil? state) (< state 0) (> state 255) (nil? message))
					(error "invalid THROW arguments") (tsql_raise_error number message state))))))
		'("if" _condition _yes _no) (tsql_run_commands (list command) statement session resultrow resultfields)
		'("cursor-declare" name options text) (tsql_cursor_declare name options text statement session)
		'("cursor-assign-query" name options text) (tsql_cursor_assign_query name options text statement session)
		'("cursor-open" name global) (tsql_cursor_open name global statement session)
		'("cursor-fetch" name global movement targets) (tsql_cursor_fetch name global movement targets statement session resultrow resultfields)
		'("cursor-close" name global) (tsql_cursor_close name global false statement session)
		'("cursor-deallocate" name global) (tsql_cursor_close name global true statement session)
		'("declare" declarations) (begin
			(tsql_bind_local_parameter_aliases schema declarations session)
			(reduce declarations (lambda (_result declaration) (begin
				(define name (car declaration))
				(if (expression_equal? (cadr declaration) '("CURSOR"))
					(begin
						(if (nil? (nth declaration 2)) nil
							(tsql_statement_call statement session "variable initializer" false (lambda (tx)
								(tsql_scalar_value schema (nth declaration 2) session tx "assignment" (cadr declaration)))))
						(tsql_cursor_variable_declare session name))
					(nil? (nth declaration 2)) (session (concat "__tsql_parameter:" name) nil)
					(tsql_statement_call statement session "variable initializer" false (lambda (tx)
						(tsql_assign_variable_query schema name (nth declaration 2) session tx)))))) nil)
			0)
		'("assign" name operation text) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) name)) nil))
			(if declaration true (error "undeclared variable: @" name))
			(if (expression_equal? (cadr declaration) '("CURSOR"))
				(if operation (error "cursor assignment cannot use a compound operator")
					(tsql_cursor_assign_reference name text statement session)) (begin
					/* Compound syntax uses the existing query compiler and assignment cast,
					including BIT destinations. Operand admission belongs to that compiler.
					The parsed variable identifier is syntax, never caller-provided SQL text. */
					(define text (if operation (concat "@" name " " operation " (" text ")") text))
					(tsql_statement_call statement session text false (lambda (tx)
						(tsql_assign_variable_query schema name text session tx)))
					0)))
		'("create-scalar-function" name parameters return_type source)
		(tsql_define_scalar_function name parameters return_type source false false statement session)
		'("alter-scalar-function" name parameters return_type source)
		(tsql_define_scalar_function name parameters return_type source true true statement session)
		'("create-inline-table-function" name parameters source replace)
		(tsql_define_inline_table_function name parameters source replace false statement session)
		'("alter-inline-table-function" name parameters source)
		(tsql_define_inline_table_function name parameters source true true statement session)
		'("create-table-function" name parameters return_name columns source)
		(tsql_define_table_function name parameters return_name columns source false false statement session)
		'("alter-table-function" name parameters return_name columns source)
		(tsql_define_table_function name parameters return_name columns source true true statement session)
		'("drop-function" names ignore)
		(tsql_drop_scalar_functions names ignore statement session)
		'("create-procedure" owner name parameters source replace)
		(tsql_define_procedure owner name parameters source replace false statement session)
		'("alter-procedure" owner name parameters source)
		(tsql_define_procedure owner name parameters source true true statement session)
		'("create-trigger" owner name target_owner target events source timing replace)
		(tsql_define_statement_trigger owner name target_owner target events source timing replace false statement session)
		'("alter-trigger" owner name target_owner target events source timing)
		(tsql_define_statement_trigger owner name target_owner target events source timing true true statement session)
		'("drop-trigger" owner name ignore) (tsql_statement_call statement session "statement trigger deletion" false (lambda (tx) (begin
			(if (or (session "__tsql_insert_exec") (session "__tsql_trigger_event"))
				(error "trigger DDL is unsupported inside native mutation execution") true)
			(define database (tsql_qualifier_database owner schema session))
			(tsql_require_permission (tsql_session_actor session) database "ALTER")
			(dropstatementtrigger database (toLower name) ignore tx)
			0)))
		'("drop-procedure" owner name ignore) (tsql_statement_call statement session "procedure deletion" false (lambda (tx) (begin
			(define owner (tsql_qualifier_database owner schema session))
			(tsql_require_permission (tsql_session_actor session) owner "ALTER")
			(define definition (find (showprocedures owner (toLower name) tx) (lambda (_definition) true) nil))
			(if (and (not (nil? definition))
				(has? '("FN" "TF" "IF") (tsql_source_module_kind (definition "language"))))
				(error "object is not a procedure: " name) true)
			(dropprocedure owner (if definition (definition "name") name) ignore tx)
			0)))
		'("insert-execute" owner name columns invocation)
		(tsql_run_insert_exec owner name columns invocation statement session)
		'("execute" owner name arguments) (tsql_invoke_procedure owner name arguments statement session resultrow resultfields)
		'("execute-status" owner name arguments target) (begin
			(define declaration (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) target)) nil))
			(if declaration true (error "undeclared return status variable: @" target))
			(define status (tsql_invoke_procedure owner name arguments statement session resultrow resultfields))
			(define contract (tsql_effective_parameter_contract (cadr declaration)))
			/* The procedure frontend declares its return status INT. Conversion
			uses that contract, never the dynamic carrier of the returned value. */
			(define value (eval (sql_type_lower_annotations '() (tsql_implicit_cast_expr
				(list (quote sql_type_expr) (list (quote quote) status) "INT" nil) (car contract) (cdr contract)))))
			(session (concat "__tsql_parameter:" target) value)
			0)
		'("execute-dynamic" expression) (begin
			(define text (tsql_statement_call statement session "dynamic SQL text" false (lambda (tx)
				(tsql_scalar_value schema expression session tx "dynamic" nil))))
			(tsql_execute_dynamic schema text '() '() statement session resultrow resultfields
				(lambda (result _status _scoped) result)))
		_ (error "unsupported batch command")))))
(define tsql_execute_command_batch (lambda (schema commands declarations supplied statement caller resultrow resultfields module_settings quoted finish) (begin
	(define scoped (tsql_scope_session caller (tsql_batch_contracts commands declarations)
		(if module_settings schema nil) module_settings))
	(scoped "__tsql_quoted_identifier" quoted)
	(map supplied (lambda (item) (scoped (concat "__tsql_parameter:" (car item)) (cadr item))))
	/* Completion and the declared INT return status are different contracts.
	Evaluate the SQL commands before reading their lexical scope's status. */
	(define run (lambda ()
		(finish (tsql_run_commands commands statement scoped resultrow resultfields)
			(coalesceNil ((tsql_identity_scope scoped) "__tsql_return_status") 0) scoped)))
	(define cursors (lambda () (tsql_with_table_variables scoped (lambda () (tsql_cursor_with_scope scoped run)))))
	(if module_settings (tsql_temporary_with_module scoped cursors) (cursors)))))
(define tsql_execute_batches (lambda (schema text session statement resultrow resultfields) (begin
	(session "schema" schema)
	/* A root connection is already its own owner. Only child scopes retain an
	explicit connection link; storing self here would make a captured parent
	view cyclic when a batch parameter is delegated to a cluster peer. */
	(define inherited (coalesceNil (session "__tsql_parameter_declarations") '()))
	(define supplied (map inherited (lambda (declaration)
		(list (car declaration) (session (concat "__tsql_parameter:" (car declaration)))))))
	(define batches (tsql_script_cache (json_encode (list "batch-texts" schema text)) nil
		(close_procedure (lambda (_tx) (tsql_script_batches text)))))
	(reduce batches (lambda (_result batch) (reduce (produceN (batch "repeat")) (lambda (_result _iteration) (begin
		(define parsed (tsql_cached_batch_context schema (batch "text") (tsql_quoted_identifier session) false (tsql_ansi_nulls session)))
		(define commands (cadr (parsed "syntax")))
		/* Parsing saves the batch's last option even when its SET sits in an
		unexecuted branch. Nested batches instead install it in their own scope. */
		(session "__tsql_quoted_identifier" (parsed "quoted_identifier"))
		(tsql_execute_command_batch schema commands inherited supplied statement session resultrow resultfields false
			(parsed "quoted_identifier")
			(lambda (result _status _scoped) result)))) 0)) 0))))
/* Frontend scopes borrow names; the native request still owns the original
connection session and its reusable transaction record. */
(define tsql_execute_request (lambda (schema text session state sequence resultrow resultfields)
	(tsql_finish_request session (lambda () (tsql_trigger_abort_boundary session (lambda ()
		(with_query_request (tsql_connection_session session) state sequence text "acid" (lambda (statement)
			(begin (tsql_validate_login_identity session)
				(tsql_initialize_date_context session)
				(tsql_initialize_parse_context session)
				(tsql_require_database_access (tsql_session_actor session) schema)
				(tsql_execute_batches schema text session statement resultrow resultfields))))))))))
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
	(tsql_validate_login_identity session)
	((tsql_policy (tsql_session_actor session)) schema true false)
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
(define tsql_run_procedure_recipe (lambda (commands contracts schema quoted ansi statement caller values resultrow resultfields finish) (begin
	/* Procedure calls inside a trigger retain their own RETURN status, but
	must not conceal DDL or unsupported transaction effects in opaque bodies. */
	(define commands (if (caller "__tsql_trigger_event")
		(tsql_prepare_trigger_body schema commands true) commands))
	(define session (tsql_scope_session caller contracts schema true))
	(session "__tsql_quoted_identifier" quoted)
	(session "__tsql_ansi_nulls" ansi)
	(session "__tsql_ansi_nulls_locked" true)
	(tsql_temporary_with_module session (lambda () (tsql_with_table_variables session (lambda () (tsql_cursor_with_scope session (lambda () (begin
		(session "schema" schema)
		(map values (lambda (item)
			(session (concat "__tsql_parameter:" (car item)) (cadr item))))
		(tsql_run_commands commands statement session resultrow resultfields)
		(finish (coalesceNil ((tsql_identity_scope session) "__tsql_return_status") 0) session)))))))))))
(define tsql_procedure_compile (lambda (source context quoted) (begin
	(define ansi (coalesceNil (context "__tsql_ansi_nulls") true))
	(define schema (context "schema"))
	(define declaration (context "parameters"))
	(define parameters (tsql_unique_parameters (tsql_procedure_parameters declaration)))
	(define commands (cadr ((tsql_parse_batch_context schema source quoted true ansi) "syntax")))
	(define contracts (tsql_batch_contracts commands (map parameters (lambda (parameter)
		(list (car parameter) (cadr parameter))))))
	/* Retain declaration recipes only; invocation sessions and native temporary
	owners are explicit call inputs and never become persisted procedure code. */
	(eval (list (quote lambda)
		(list (quote statement) (quote caller) (quote values) (quote resultrow) (quote resultfields) (quote finish))
		(list (quote !begin) (list tsql_require_procedure_context (list (quote quote) declaration))
			(list (quote tsql_run_procedure_recipe) (list (quote quote) commands)
				(list (quote quote) contracts) schema quoted ansi (quote statement) (quote caller) (quote values)
				(quote resultrow) (quote resultfields) (quote finish))))))))

(tsql_register_source_profiles "tsql-procedure" tsql_procedure_compile registerprocedurelanguage)
(define tsql_execute_dynamic (lambda (schema text declarations values statement caller resultrow resultfields finish) (begin
	/* Text and bindings are scalar request inputs. Compilation and all relational
	execution use the ordinary frontend and planner, with the same statement owner
	and a new lexical scope. No caller local leaks into a nested dynamic batch. */
	(if (nil? text) (error "dynamic SQL text cannot be NULL") true)
	/* GO separates submitted scripts, but is not a statement in executed text. */
	(define parsed (tsql_cached_batch_context schema text (tsql_quoted_identifier caller) false (tsql_ansi_nulls caller)))
	(define syntax (parsed "syntax"))
	(define commands (if (caller "__tsql_trigger_event")
		(tsql_prepare_trigger_body schema (cadr syntax) true) (cadr syntax)))
	(tsql_execute_command_batch schema commands declarations values statement caller resultrow resultfields true
		(parsed "quoted_identifier") finish))))
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
	/* Explicit arguments read caller locals. Defaults referring to formals read
	only the already converted values of preceding declarations in this scope. */
	(define defaults (tsql_scope_session caller declarations nil false))
	(define values (tsql_statement_call statement caller "procedure parameter binding" false (lambda (tx) (reverse (reduce declarations (lambda (values declaration) (begin
		(define name (car declaration))
		(define supplied (find named (lambda (item) (equal? (car item) name)) nil))
		(define expression (match (if supplied (cadr supplied) '("default"))
			'("output" target) (concat "@" target)
			'("default") (if (nil? (nth declaration 2)) (error "missing procedure argument: @" name) (nth declaration 2))
			text text))
		(define value (match expression
			'("wire-parameter" parameter) (tds_parameter_assignment caller parameter (cadr declaration))
			'("parameter-reference" reference)
			(tsql_scalar_value (caller "schema") (concat "@" reference) defaults tx "assignment" (cadr declaration))
			_ (tsql_scalar_value (caller "schema") expression caller tx "assignment" (cadr declaration))))
		(defaults (concat "__tsql_parameter:" name) value)
		(cons (list name value) values))) '())))))
	(list values outputs))))
(define tsql_copy_outputs (lambda (outputs scoped statement caller) (begin
	/* Convert all outgoing scalars using the callee's declared source contract
	and the caller's declared target, then publish the caller bindings. */
	(if (empty_list? outputs) true (begin
		(define values (tsql_statement_call statement caller "procedure output binding" false (lambda (tx) (map outputs (lambda (output)
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
	(define binding (tsql_statement_call statement caller "dynamic declarations" false (lambda (tx) (begin
		(define text (tsql_scalar_value (caller "schema") (cadr stmt) caller tx "dynamic" nil))
		(define contracts (if params (tds_bind_declarations schema
			(tsql_scalar_value (caller "schema") (cadr params) caller tx "dynamic" nil)) '()))
		(list text (map contracts (lambda (contract)
			(list (car contract) (cadr contract) nil (if (> (count contract) 2) (nth contract 2) false)))))))))
	(define bound (tsql_bind_call supplied (cadr binding) statement caller))
	(tsql_execute_dynamic schema (car binding) (map (cadr binding) (lambda (item) (list (car item) (cadr item))))
		(car bound) statement caller resultrow resultfields
		(lambda (_result status scoped) (begin (tsql_copy_outputs (cadr bound) scoped statement caller) status))))))
/* Virtual filesystem calls share ordinary typed argument/output binding. The
namespace module cannot perform host I/O, even for an administrator. */
(define tsql_execute_fileexist (lambda (arguments statement caller resultrow resultfields) (begin
	(define declarations (list
		(list "filename" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(4000)) nil false)
		(list "file_exists" (tsql_parameter_declaration (caller "schema") "INT" '()) "NULL" true)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(if (and (> (count arguments) 1) (empty_list? (cadr binding)))
		(error "xp_fileexist second argument requires OUTPUT") true)
	(define status (tsql_filesystem_stat (cadr (find (car binding) (lambda (item) (equal? (car item) "filename")) nil))))
	(if (empty_list? (cadr binding)) (begin
		(define names '("File Exists" "File is a Directory" "Parent Directory Exists"))
		(resultfields names (map names (lambda (name)
			(list "name" name "kind" 38 "size" 4 "flags" 1 "sql_type" "INT"))))
		(resultrow (list "File Exists" (status "file") "File is a Directory" (status "directory")
			"Parent Directory Exists" (status "parent"))))
		(begin
			(define scoped (tsql_scope_session caller (map declarations (lambda (item) (list (car item) (cadr item)))) (caller "schema") false))
			(scoped "__tsql_parameter:file_exists" (status "file"))
			(tsql_copy_outputs (cadr binding) scoped statement caller)))
	0)))
/* xp_cmdshell is a legacy command spelling for explicit virtual operations.
There is no process execution path. Unimplemented engine writes fail in the
shared dispatcher; directory results contain no fabricated free-space line. */
(define tsql_execute_virtual_command (lambda (arguments statement caller resultrow resultfields) (begin
	((tsql_policy (tsql_session_actor caller)) "system" true true)
	(define declarations (list
		(list "command_string" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(4000)) nil false)
		(list "no_output" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(20)) "NULL" false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define values (car binding))
	(define option (cadr (find values (lambda (item) (equal? (car item) "no_output")) nil)))
	(if (or (nil? option) (equal?? option "NO_OUTPUT")) true (error "unsupported virtual command option"))
	(define result (tsql_filesystem_execute (cadr (find values (lambda (item) (equal? (car item) "command_string")) nil)) '()))
	(if (nil? option) (begin
		(resultfields '("output") (list (list "name" "output" "kind" 231 "size" 510
			"collation" (hex2bin "0904d00034") "flags" 1 "sql_type" "NVARCHAR")))
		(match (result "operation")
			"list" (map (result "result") (lambda (entry)
				(resultrow (list "output" (concat "<DIR> " (entry "name"))))))
			_ (error "virtual operation output is not implemented"))) true)
	0)))
/* Compatibility metadata is a bounded virtual profile, not host introspection.
System procedure arguments use the ordinary declaration and OUTPUT contracts. */
(define tsql_server_info_columns (list
	(list "name" "Index" "kind" 38 "size" 4 "flags" 1 "sql_type" "INT" "sql_datatype" "INT")
	(list "name" "Name" "kind" 231 "size" 64 "flags" 1 "sql_type" "NVARCHAR" "sql_datatype" (list "NVARCHAR" '(32)) "collation" (hex2bin "0904d00034"))
	(list "name" "Internal_Value" "kind" 38 "size" 4 "flags" 1 "sql_type" "INT" "sql_datatype" "INT")
	(list "name" "Character_Value" "kind" 231 "size" 1024 "flags" 1 "sql_type" "NVARCHAR" "sql_datatype" (list "NVARCHAR" '(512)) "collation" (hex2bin "0904d00034"))))
(define tsql_execute_msver (lambda (arguments statement caller resultrow resultfields) (begin
	(define declarations (list (list "optname" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(128)) "NULL" false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define option (cadr (car (car binding))))
	(if (and (not (empty_list? arguments)) (nil? option)) (error "virtual server information option cannot be NULL") true)
	(define rows (tsql_server_info_rows option))
	(resultfields '("Index" "Name" "Internal_Value" "Character_Value") tsql_server_info_columns)
	(map rows (lambda (row) (resultrow (list "Index" (nth row 0) "Name" (nth row 1)
		"Internal_Value" (nth row 2) "Character_Value" (nth row 3)))))
	1)))
(define tsql_execute_virtual_registry (lambda (arguments statement caller resultrow resultfields) (begin
	(define declarations (list
		(list "rootkey" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(128)) nil false)
		(list "key" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(512)) nil false)
		(list "value_name" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(128)) nil false)
		(list "value" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(512)) "NULL" true)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(if (and (> (count arguments) 3) (empty_list? (cadr binding)))
		(error "virtual registry fourth argument requires OUTPUT") true)
	(define values (car binding))
	(define read (lambda (name) (cadr (find values (lambda (item) (equal? (car item) name)) nil))))
	(define value (tsql_virtual_registry_read (read "rootkey") (read "key") (read "value_name")))
	(if (empty_list? (cadr binding)) (begin
		(resultfields '("Value" "Data") (list
			(list "name" "Value" "kind" 231 "size" 256 "flags" 1 "sql_type" "NVARCHAR" "collation" (hex2bin "0904d00034"))
			(list "name" "Data" "kind" 231 "size" 1024 "flags" 1 "sql_type" "NVARCHAR" "collation" (hex2bin "0904d00034"))))
		(resultrow (list "Value" (read "value_name") "Data" value)))
		(begin
			(define scoped (tsql_scope_session caller (map declarations (lambda (item) (list (car item) (cadr item)))) (caller "schema") false))
			(scoped "__tsql_parameter:value" value)
			(tsql_copy_outputs (cadr binding) scoped statement caller)))
	0)))
/* Linux profile: write the real MemCP server log, without a client message
handler or a Windows Event Log emulation. The return/status contract is the
SQL Server logging procedure's 0/1; malformed number/severity never becomes a
THROW or a fabricated @@ERROR. Native administration is checked before binding.
https://learn.microsoft.com/sql/relational-databases/system-stored-procedures/xp-logevent-transact-sql
SQL Server 2022 reference: https://dbfiddle.uk/TuUw4stC */
(define tsql_execute_logevent (lambda (arguments statement caller) (begin
	(tsql_require_permission (tsql_session_actor caller) (caller "schema") "NATIVE ADMIN")
	(define declarations (list
		(list "error_number" (tsql_parameter_declaration (caller "schema") "INT" '()) nil false)
		(list "message" (tsql_parameter_declaration (caller "schema") "VARCHAR" '(2048)) nil false)
		(list "severity" (tsql_parameter_declaration (caller "schema") "VARCHAR" '(128)) "'INFORMATIONAL'" false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define values (car binding))
	(define number (cadr (nth values 0)))
	(define message (cadr (nth values 1)))
	(define severity (cadr (nth values 2)))
	(if (or (nil? number) (nil? message) (nil? severity)) 1
		(or (<= number 50000) (not (has? '("INFORMATIONAL" "WARNING" "ERROR") (toUpper severity)))) 1
		(begin (print (concat "[xp_logevent " (toUpper severity) " " number "] " message)) 0)))))

/* The existing logical view binder validates the complete stored declaration.
Only current-database views are admitted, as required by sp_refreshview.
The system command owns no data query or physical materialization.
https://learn.microsoft.com/sql/relational-databases/system-stored-procedures/sp-refreshview-transact-sql */
(define tsql_execute_refreshview (lambda (schema arguments statement caller) (begin
	(define declarations (list
		(list "viewname" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(776)) nil false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define name (cadr (car (car binding))))
	(if (nil? name) (error "view name cannot be NULL") true)
	(define parts (tsql_object_name_parts name))
	(define database (database_name schema true))
	(define target (database_name (tsql_qualifier_database (car parts) database caller) true))
	(if (and (equal? target database) (equal?? (cadr parts) "dbo")) true
		(error "sp_refreshview requires a current-database view"))
	(tsql_require_permission (tsql_session_actor caller) database "ALTER")
	(tsql_statement_call statement caller "refresh view" false (lambda (tx)
		(refresh_sql_view tx database (nth parts 2)
			(tsql_policy (tsql_session_actor caller)) caller))))))

/* SQL Server changes an object's address, not stored SQL source text. Native
column bindings retain data, keys and computed dependencies; the original
statement owner supplies the transaction, including earlier data changes.
https://learn.microsoft.com/sql/relational-databases/system-stored-procedures/sp-rename-transact-sql */
(define tsql_rename_address (lambda (schema text kind caller) (begin
	(define parts (if (nil? text) nil (tsql_name_parts text)))
	(define column (equal? kind "COLUMN"))
	(define minimum (if column 2 1))
	(define maximum (if column 4 3))
	(if (or (nil? parts) (< (count parts) minimum) (> (count parts) maximum)
		(find parts (lambda (part) (or (nil? part) (equal? part "")
			(not (tsql_sysname? part)) (sql_private_relation? part))) nil))
		(tsql_raise_error 15248 "The object name is invalid or ambiguous for the requested object type." 1) true)
	(define reversed (reverse parts))
	(define namespace (if (>= (count parts) (+ minimum 1)) (nth reversed minimum) "dbo"))
	(define qualifier (if (equal? (count parts) maximum) (car parts) nil))
	(define database (database_name (caller "schema") true))
	(define invoked_database (database_name schema true))
	(define addressed_database (database_name (tsql_qualifier_database qualifier database caller) true))
	(if (and (equal? invoked_database database) (equal? addressed_database database)) true
		(tsql_raise_error 15250 "The database name must be the current database." 1))
	(if (equal?? namespace "dbo") true
		(tsql_raise_error 15248 "The object is not in the requested schema." 1))
	(list database (nth reversed (- minimum 1)) (if column (car reversed) nil)))))
(define tsql_execute_rename (lambda (schema arguments statement caller) (begin
	(define declarations (list
		(list "objname" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(776)) nil false)
		(list "newname" (tsql_parameter_declaration (caller "schema") "NVARCHAR" '(128)) nil false)
		(list "objtype" (tsql_parameter_declaration (caller "schema") "VARCHAR" '(13)) "NULL" false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define values (car binding))
	(define name (cadr (nth values 0)))
	(define replacement (cadr (nth values 1)))
	(define kind (toUpper (sql_rtrim (coalesceNil (cadr (nth values 2)) "OBJECT"))))
	(if (has? '("COLUMN" "OBJECT") kind) true
		(error "sp_rename object type is not supported: " kind))
	(define address (tsql_rename_address schema name kind caller))
	/* @newname is one literal SYSNAME, not a multipart address. Dots and
	brackets remain part of that name, unlike the parsed @objname argument.
	SQL Server 2022 reference: https://dbfiddle.uk/NdvHBCtS */
	(if (or (nil? replacement) (equal? replacement "") (sql_private_relation? replacement))
		(tsql_raise_error 15248 "The new name must be a nonempty public identifier." 1) true)
	(define database (car address))
	(tsql_require_permission (tsql_session_actor caller) database "ALTER")
	(tsql_statement_call statement caller "rename object" false (lambda (tx) (begin
		(define relation (tsql_resolve_relation database (cadr address) caller tx))
		(define handle (tsql_native_relation_handle (car relation) (cadr relation) tx))
		(if (nil? handle) (tsql_raise_error 15248 "The rename target does not exist or is ambiguous." 1) true)
		(if (equal? kind "COLUMN") (begin
			(define declaration (show handle "column" (nth address 2) true tx))
			(if (nil? declaration) (tsql_raise_error 15248 "The column to rename does not exist." 1) true)
			(if (find (show handle "columns" tx) (lambda (column)
				(and (equal?? (column "Field") replacement)
					(not (equal? (column "Field") (declaration "Field"))))) nil)
				(tsql_raise_error 15335 "The new column name is already in use." 1) true)
			(renamecolumn handle (declaration "Field") replacement true
				(coalesceNil (declaration "StorageField") (declaration "Field"))
				(list (declaration "RawType") (coalesceNil (declaration "Dimensions") '())
					(declaration "Null") (coalesceNil (declaration "Collation") "")) tx))
			(begin
				(if (or (tsql_temporary_name? (cadr address)) (tsql_temporary_name? replacement))
					(error "temporary table rename requires temporary namespace rebinding") true)
				(define occupied (tsql_native_relation_handle database replacement tx))
				(if (and occupied (not (identical? occupied handle)))
					(tsql_raise_error 15335 "The new object name is already in use." 1) true)
				(if (tsql_view_name database replacement tx)
					(tsql_raise_error 15335 "The new object name is already in use." 1) true)
				(renametable handle replacement true (show handle "identity" tx) tx)))
		0))))))

(define tsql_invoke_procedure (lambda (schema name arguments statement caller resultrow resultfields) (begin
	(define system_call (tsql_system_procedure_declaration name))
	(define name (if system_call (car system_call) name))
	(define metadata_call (if system_call (equal? (nth system_call 2) "metadata") false))
	/* sys is a virtual owner only for supported system calls. Other owners
	use the same import binding as ordinary SQL, before any invocation runs. */
	(define schema (if (and (equal?? schema "sys")
		(not (nil? system_call))) (caller "schema")
		(tsql_qualifier_database schema (caller "schema") caller)))
	(if (and (caller "__tsql_fmtonly")
		(if system_call (equal? (nth system_call 2) "command") false))
		(error "FMTONLY does not execute system commands") true)
	(if (equal? (toLower name) "sp_executesql")
		(tsql_execute_parameterized (if (equal?? schema "sys") (caller "schema") schema)
			arguments statement caller resultrow resultfields)
		(equal? name "sp_refreshview")
		(tsql_execute_refreshview schema arguments statement caller)
		(equal? name "sp_rename")
		(tsql_execute_rename schema arguments statement caller)
		(equal? (toLower name) "sp_changedbowner")
		(tsql_change_database_owner arguments statement caller)
		(equal? (toLower name) "xp_logevent")
		(tsql_execute_logevent arguments statement caller)
		(equal? (toLower name) "xp_msver")
		(tsql_execute_msver arguments statement caller resultrow resultfields)
		(has? '("xp_regread" "xp_instance_regread") (toLower name))
		(tsql_execute_virtual_registry arguments statement caller resultrow resultfields)
		(equal? (toLower name) "xp_cmdshell")
		(tsql_execute_virtual_command arguments statement caller resultrow resultfields)
		(equal? (toLower name) "xp_fileexist")
		(tsql_execute_fileexist arguments statement caller resultrow resultfields)
		metadata_call
		(tsql_statement_call statement caller "metadata procedure" false (lambda (tx) (begin
			(define values (map arguments (lambda (argument) (begin
				(match (cadr argument) (cons _head _tail) (error "metadata parameters are input-only") _ true)
				(list (car argument) (tsql_scalar_value (caller "schema") (cadr argument) caller tx "value" nil))))))
			(tsql_execute_metadata (if (equal?? schema "sys") (caller "schema") schema)
				name values (tsql_policy (tsql_session_actor caller))
				(if (caller "__tsql_fmtonly") (lambda (_row) true) resultrow)
				resultfields (tsql_session_actor caller) tx))))
		(begin
			(tsql_require_permission (tsql_session_actor caller) schema "EXECUTE")
			/* Retain one immutable definition and callable across argument binding. */
			(define snapshot (tsql_statement_call statement caller "procedure lookup" false (lambda (tx)
				(procedure schema (toLower name) true tx))))
			(define definition (snapshot "declaration"))
			(if (tsql_procedure_language? (definition "language")) true
				(error "procedure language cannot execute in this frontend"))
			/* Reject unsupported impersonation before default/argument binding or
			any output callback can execute. The compiled callable also owns this
			admission for direct native invocations of the stored definition. */
			(tsql_require_procedure_context (definition "parameters"))
			(define binding (tsql_bind_call arguments (tsql_procedure_parameters (definition "parameters")) statement caller))
			((snapshot "callable") statement caller (car binding) resultrow resultfields
				(lambda (status scoped) (begin (tsql_copy_outputs (cadr binding) scoped statement caller) status))))))))

(define tsql_http_query (lambda (req res schema) (begin
	(define account (sql_authenticate_account (req "username") (req "password")))
	(if account
		(begin
			(define text ((req "body")))
			(try (lambda () (time (begin
				((res "header") "Content-Type" "text/event-stream; charset=utf-8")
				(define session (sql_bind_http_session req account))
				(sql_http_query_parameters req session)
				/* Authentication and the route own identity and database selection;
				query options may supply values, never replace either authority. */
				(session "username" (account "username"))
				(session "__tsql_principal_id" (account "principal_id"))
				(session "schema" schema)
				(session "tsql_import_target" nil)
				(session "tsql_import_source" nil)
				/* One scope owner per submitted batch, never per native insert row. */
				(session "__tsql_scope" (newsession))
				(define output (newsession))
				/* Capture the declared response callbacks, never the response
				dictionary with its request-local native HTTP writer. */
				(define write_row (res "jsonl"))
				(define write_message (res "print"))
				(define request_session (tsql_message_session session (lambda (message)
					(write_message ": " (json_encode (list "info" message)) "\n\n"))))
				(define result (tsql_execute_request schema text request_session (req "__session_state") (req "__query_seq")
					(lambda (values) (begin (output "rows" true) (write_row values)))
					(lambda args (output "fields" true))))
				/* An empty result set still has a declared field contract. Its
				procedure return status must not become an affected-row record. */
				(if (and (not (or (output "rows") (output "fields")))
					(not (tsql_completion_count_absent? result))
					(not (session "__tsql_nocount")) (number? result))
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
(define tsql_metadata_table_names (lambda (schema pattern patterns policy tx) (begin
	(if policy (policy schema true false) true)
	(filter (coalesceNil (show schema false tx) '()) (lambda (name)
		(and (not (internal_column_name? name)) (tsql_metadata_match name pattern patterns)))))))
/* View definitions stay in the existing system catalog. This transient answer
joins object names only; native key procedures keep enumerating native tables. */
(define tsql_metadata_relations (lambda (schema pattern patterns policy tx) (begin
	(define native (map (tsql_metadata_table_names schema pattern patterns policy tx)
		(lambda (name) (list name "TABLE"))))
	(define views (if (sql_views_may_exist)
		(scan tx (table "system" false "views" false tx)
			(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false))
			(list schema) '("name") (lambda (name) (tsql_metadata_match name pattern patterns))
			'("name") (lambda (acc name) (cons (list name "VIEW") acc))
			'() (lambda (a b) (merge (list a b)))) '()))
	(sort (merge (list native views)) (lambda (left right)
		(if (equal? (cadr left) (cadr right)) (< (car left) (car right)) (< (cadr left) (cadr right))))))))
/* Describe a logical view projection, never its rows. Native declarations can
supply direct projection nullability and dimensions, but not a view default,
identity, key or comment. Unknown expression widths must not become width one. */
(define tsql_metadata_view_columns (lambda (schema name policy planning_session tx) (begin
	(define query (tsql_describe_query
		(make_query_block schema (list (list name schema name false nil nil))
			(list "*" (list (quote get_column) nil false "*" false))
			nil nil nil nil nil nil '() '() '()) policy planning_session tx))
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
(define tsql_metadata_constraints (lambda (schema name tx) (show (table schema false name false tx) "constraints")))
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
	(define type (toLower (tsql_metadata_base_type (column "RawType"))))
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
	(define type (toLower (tsql_metadata_base_type (column "RawType"))))
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
		(expression_equal? head tsql_cast_bit)
		(reduce (list tsql_cast_text tsql_cast_fixed_text tsql_cast_binary tsql_cast_decimal)
			(lambda (matched helper) (or matched (expression_equal? head helper))) false))))
/* The compiler's arithmetic-error boundary preserves the cast declaration.
Inspect its exact syntax without evaluating either policy or the cast thunk. */
(define tsql_parameter_cast_formula (lambda (formula) (match formula
	'(head _enabled ((symbol lambda) '() body))
	(if (identical? head tsql_arithmetic_value) body formula)
	_ formula)))
(define tsql_parameter_analysis_join (lambda (left right)
	(list (merge (list (car left) (car right))) (merge (list (cadr left) (cadr right))))))
(define tsql_parameter_candidates (lambda (node sources declarations) (match node
	((symbol quote) _datum) (list '() '())
	((symbol tsql_unbound_parameter) name spelling) (list (list (list name spelling)) '())
	(cons _head _tail) (begin
		(define scope (if (query_block? node) (merge (list (qb_sources node) sources)) sources))
		(define own (match node
			((symbol sql_type_expr) formula type _collation) (match (tsql_parameter_cast_formula formula)
				(cons head (cons operand _arguments)) (begin
					(define parameter (tsql_parameter_probe operand))
					(if (and parameter (tsql_parameter_cast_head? head))
						(list (list (car parameter) (cadr parameter) (sql_datatype_name type)
							(match type '(_name dimensions) dimensions
								_ (coalesceNil (tsql_result_dimensions scope formula declarations) '())))) '()))
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
(define tsql_describe_parameters (lambda (schema text parameter_text policy actor tx) (begin
	(define declarations (tds_bind_declarations schema (coalesceNil parameter_text "")))
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" declarations)
	(planning "__tsql_parameter_analysis" true)
	(if actor (begin (planning "username" (car actor)) (planning "__tsql_principal_id" (cadr actor))) true)
	(planning "__tsql_describe" true)
	(define query (match (parse_tsql schema text policy planning tx)
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
	(if actor (begin (validation "username" (car actor)) (validation "__tsql_principal_id" (cadr actor))) true)
	(validation "__tsql_parameter_declarations" (merge (list declarations
		(map specifications (lambda (entry) (list (car entry) (cons (nth entry 2) (nth entry 3))))))))
	(define checked (parse_tsql schema text policy validation tx))
	(if (list? checked)
		(map (produceN (count specifications)) (lambda (i) (tsql_parameter_description (nth specifications i) (+ i 1))))
		(error "invalid inferred parameter statement")))))

/* The ordinal row contract matches the ordinary typed SELECT description.
Only declared character columns carry a collation; native table/file layout
and request-local objects never become metadata values. */
(define tsql_tablecollations (lambda (schema object policy actor tx) (begin
	(if (nil? object) (error "table collation metadata requires an object name") true)
	(define parts (tsql_object_name_parts object))
	(define database (coalesceNil (car parts) schema))
	(if (and (equal?? database schema) (equal?? (cadr parts) "dbo")) true
		(error "table collation qualifier is not the current database and owner"))
	(if actor (tsql_require_permission actor schema "SELECT") true)
	(if policy (policy schema (nth parts 2) false) true)
	(define handle (tsql_native_relation_handle schema (nth parts 2) tx))
	(if handle true (error "table collation object does not exist"))
	(define columns (filter (show handle) (lambda (column) (not (internal_column_name? (column "Field"))))))
	(define rows (mapIndex columns (lambda (i raw_column) (begin
		(define column (tsql_effective_column raw_column nil))
		(define type (tsql_metadata_base_type (column "RawType")))
		(define dimensions (column "Dimensions"))
		(define character (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") type))
		(define collation (column "Collation"))
		(define info (sql_info nil (if (empty_list? dimensions) type (list type dimensions))
			(if (equal? collation "") nil (list collation 2))))
		(list "colid" (+ i 1) "name" (column "Field")
			"tds_collation" (if character ((tsql_wire_info_contract info dimensions) "collation") nil)
			"collation" (if character (if (equal? collation "") "SQL_Latin1_General_CP1_CI_AS" collation) nil))))))
	(define descriptors (list
		(merge (list "name" "colid" "sql_type" "INT" "sql_datatype" "INT" "flags" 0) (tsql_wire_contract "INT" '()))
		(merge (list "name" "name" "sql_type" "NVARCHAR" "sql_datatype" (list "NVARCHAR" '(128)) "flags" 0) (tsql_wire_contract "NVARCHAR" '(128)))
		(merge (list "name" "tds_collation" "sql_type" "VARBINARY" "sql_datatype" (list "VARBINARY" '(5)) "flags" 1) (tsql_wire_contract "VARBINARY" '(5)))
		(merge (list "name" "collation" "sql_type" "NVARCHAR" "sql_datatype" (list "NVARCHAR" '(128)) "flags" 1) (tsql_wire_contract "NVARCHAR" '(128)))))
	(list '("colid" "name" "tds_collation" "collation") '("colid") rows descriptors))))

(define tsql_metadata_result (lambda (schema procedure arguments policy actor tx) (begin
	(define procedure (toLower procedure))
	(define procedure (if (and (>= (strlen procedure) 4) (equal? (substr procedure 0 4) "sys.")) (substr procedure 4) procedure))
	(define names (match procedure
		"sp_describe_undeclared_parameters" '("tsql" "params")
		"sp_datatype_info" '("data_type" "odbcver")
		"sp_datatype_info_100" '("data_type" "odbcver")
		"sp_tables" '("table_name" "table_owner" "table_qualifier" "table_type" "fusepattern")
		"sp_columns" '("table_name" "table_owner" "table_qualifier" "column_name" "odbcver")
		"sp_tablecollations" '("object")
		"sp_tablecollations_100" '("object")
		"sp_pkeys" '("table_name" "table_owner" "table_qualifier")
		"sp_statistics" '("table_name" "table_owner" "table_qualifier" "index_name" "is_unique" "accuracy")
		"sp_fkeys" '("pktable_name" "pktable_owner" "pktable_qualifier" "fktable_name" "fktable_owner" "fktable_qualifier")
		"sp_special_columns" '("table_name" "table_owner" "qualifier" "col_type" "scope" "nullable" "odbcver")
		_ (error "unsupported metadata procedure: " procedure)))
	(define options (tsql_metadata_arguments arguments names))
	(if (and actor (not (has? '("sp_datatype_info" "sp_datatype_info_100" "sp_describe_undeclared_parameters") procedure)))
		(tsql_require_permission actor schema "SELECT") true)

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
	(if actor (begin
		(planning_session "username" (car actor))
		(planning_session "__tsql_principal_id" (cadr actor))) true)
	/* View declarations expand source relations through the same SELECT
	admission as execution; CONNECT to a source does not reveal its columns. */
	(define declaration_policy (if actor (tsql_metadata_policy actor) policy))
	(define result (match procedure
		"sp_describe_undeclared_parameters" (begin
			(if (nil? (options "tsql")) (error "parameter description requires statement text") true)
			(define rows (tsql_describe_parameters schema (options "tsql") (options "params") (if actor (tsql_metadata_policy actor) policy) actor tx))
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
		"sp_tablecollations" (tsql_tablecollations schema (options "object") policy actor tx)
		"sp_tablecollations_100" (tsql_tablecollations schema (options "object") policy actor tx)
		"sp_datatype_info" (tsql_metadata_type_info options version)
		"sp_datatype_info_100" (tsql_metadata_type_info options version)
		"sp_tables" (begin
			(define filter_type (options "table_type"))
			(define types (if (nil? filter_type) '("TABLE" "VIEW")
				(map (split filter_type ",") (lambda (value) (toUpper (replace (strtrim value) "'" ""))))))
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(map (filter (tsql_metadata_relations schema (options "table_name")
					(not (equal? (coalesceNil (options "fusepattern") 1) 0)) policy tx)
					(lambda (relation) (has? types (cadr relation))))
					(lambda (relation) (list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo"
						"TABLE_NAME" (car relation) "TABLE_TYPE" (cadr relation) "REMARKS" nil))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "TABLE_TYPE" "REMARKS") '() rows))
		"sp_columns" (begin
			(if (nil? (options "table_name")) (error "sp_columns requires a table name") true)
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(merge (map (tsql_metadata_relations schema (options "table_name") true policy tx) (lambda (relation) (begin
					(define name (car relation))
					(define columns (map (if (equal? (cadr relation) "VIEW")
						(tsql_metadata_view_columns schema name declaration_policy planning_session tx)
						(filter (show (table schema false name false tx)) (lambda (column) (not (internal_column_name? (column "Field"))))))
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
			(define rows (if (equal?? owner "dbo") (merge (map (tsql_metadata_table_names schema (options "table_name") false policy tx) (lambda (name) (begin
				(define columns (map (show (table schema false name false tx))
					(lambda (column) (tsql_effective_column column planning_session))))
				(define keys ((tsql_metadata_constraints schema name tx) "Unique"))
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
				(merge (map (tsql_metadata_table_names schema (options "table_name") false policy tx) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name tx) "Unique"))
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
				(merge (map (tsql_metadata_table_names schema (options "table_name") false policy tx) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name tx) "Unique"))
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
				(merge (map (tsql_metadata_table_names schema child false policy tx) (lambda (name) (begin
					(define keys ((tsql_metadata_constraints schema name tx) "ForeignKeys"))
					(merge (map (filter keys (lambda (key) (and (equal? (key "Role") "referencing")
						(tsql_metadata_match (key "OtherTable") parent false)))) (lambda (key) (begin
							(define referenced (table schema false (key "OtherTable") false tx))
							(define candidate (if referenced (find ((show referenced "constraints") "Unique")
								(lambda (unique) (equal? (unique "Cols") (key "OtherColumns")))) nil))
							(tsql_metadata_foreign_columns (key "LocalColumns") (key "OtherColumns") (lambda (column other ordinal)
								(list "PKTABLE_QUALIFIER" schema "PKTABLE_OWNER" "dbo" "PKTABLE_NAME" (key "OtherTable") "PKCOLUMN_NAME" other
									"FKTABLE_QUALIFIER" schema "FKTABLE_OWNER" "dbo" "FKTABLE_NAME" name "FKCOLUMN_NAME" column "KEY_SEQ" ordinal
									"UPDATE_RULE" (tsql_metadata_fk_rule (key "UpdateMode")) "DELETE_RULE" (tsql_metadata_fk_rule (key "DeleteMode"))
									"FK_NAME" (key "Id") "PK_NAME" (if candidate (tsql_metadata_key_name candidate) nil) "DEFERRABILITY" 7))))))))))) '()))
			(list '("PKTABLE_QUALIFIER" "PKTABLE_OWNER" "PKTABLE_NAME" "PKCOLUMN_NAME" "FKTABLE_QUALIFIER" "FKTABLE_OWNER" "FKTABLE_NAME" "FKCOLUMN_NAME" "KEY_SEQ" "UPDATE_RULE" "DELETE_RULE" "FK_NAME" "PK_NAME" "DEFERRABILITY")
				'("KEY_SEQ" "UPDATE_RULE" "DELETE_RULE" "DEFERRABILITY") rows))))
	(list "columns" (if (> (count result) 3) (nth result 3)
		(tsql_metadata_descriptors (car result) (cadr result))) "rows" (nth result 2)))))
(define tsql_execute_metadata (lambda (schema procedure arguments policy resultrow resultfields actor tx) (begin
	(define result (tsql_metadata_result schema procedure arguments policy actor tx))
	(resultfields (map (result "columns") (lambda (column) (column "name"))) (result "columns"))
	(map (result "rows") resultrow)
	0)))

/* Named RPC identifiers and values remain data. The metadata transport event
also carries ordinary stored procedure names; only actual metadata declarations
use the bounded catalog adapter. Prefix resolution matches the normal EXEC
owner: dbo is relative, two-part native qualifiers retain their existing mapping. */
(define tds_metadata_procedure? (lambda (name) (begin
	(define lower (toLower name))
	(define bare (if (and (>= (strlen lower) 4) (equal? (substr lower 0 4) "sys.")) (substr lower 4) lower))
	(define declaration (tsql_system_procedure_declaration bare))
	(if declaration (equal? (nth declaration 2) "metadata") false))))
(define tds_rpc_address (lambda (name) (begin
	(define parts (tsql_object_name_parts name))
	(define database (car parts))
	(define owner (cadr parts))
	(if (and database (not (equal?? owner "dbo")))
		(error "unsupported qualified owner: " owner) true)
	(list (if database database (equal?? owner "dbo") 'tsql_default_owner owner) (nth parts 2)))))
/* A by-reference wire parameter is the caller's scalar variable declaration.
The stored procedure keeps its own independent formal declaration. Copy-in/out
therefore use the same implicit casts as SQL EXEC with differently typed locals. */
(define tds_parameter_contract (lambda (wire) (begin
	(define type (tds_parameter_source_type wire))
	(define dimensions (match (wire "kind")
		38 (begin
			(define integer (match (wire "size") 1 "TINYINT" 2 "SMALLINT" 4 "INT" 8 "BIGINT"
				_ (error "invalid integer wire parameter width")))
			(list integer))
		109 (list (match (wire "size") 4 "REAL" 8 "FLOAT" _ (error "invalid floating wire parameter width")))
		106 (list "DECIMAL" (wire "precision") (wire "scale"))
		108 (list "NUMERIC" (wire "precision") (wire "scale"))
		41 (list "TIME" (wire "scale"))
		42 (list "DATETIME2" (wire "scale"))
		43 (error "offset-bearing RPC OUTPUT is not implemented")
		31 (error "an RPC OUTPUT parameter requires a declared wire type")
		165 (list "VARBINARY" (if (equal? (wire "size") 65535) -1 (wire "size")))
		173 (list "BINARY" (wire "size"))
		167 (list "VARCHAR" (if (equal? (wire "size") 65535) -1 (wire "size")))
		175 (list "CHAR" (wire "size"))
		231 (list "NVARCHAR" (if (equal? (wire "size") 65535) -1 (intdiv (wire "size") 2)))
		239 (list "NCHAR" (intdiv (wire "size") 2))
		_ (list type)))
	/* Reuse the normal frontend dimension validation, including MAX/scale. */
	(tsql_parameter_declaration nil (car dimensions) (cdr dimensions)))))
(define tds_rpc_output_slots (lambda (parameters)
	(map (filter parameters (lambda (parameter) (equal? (parameter "flags") 1)))
		(lambda (parameter) (begin
			(define ordinal (parameter "ordinal"))
			(if (and (int? ordinal) (>= ordinal 0) (<= ordinal 65535)) true
				(error "RPC OUTPUT requires its original wire ordinal"))
			(define name (concat "__tds_rpc_output_" ordinal))
			(list "slot" name "parameter" parameter
				"declaration" (list name (tds_parameter_contract (parameter "type")))))))))
(define tds_rpc_output_scope (lambda (parent slots) (begin
	(define scope (tsql_scope_session parent (map slots (lambda (slot) (slot "declaration"))) nil false))
	(map slots (lambda (slot)
		(scope (concat "__tsql_parameter:" (slot "slot"))
			(tds_parameter_assignment scope (slot "parameter") (cadr (slot "declaration"))))))
	scope)))
(define tds_rpc_arguments (lambda (parameters slots)
	(map parameters (lambda (parameter) (begin
		(if (has? '(0 1) (parameter "flags")) true (error "unsupported RPC parameter flags"))
		(define name (tds_parameter_name (parameter "name") ""))
		(define expression (if (equal? (parameter "flags") 0) (list "wire-parameter" parameter)
			(begin
				(define slot (find slots (lambda (slot)
					(equal? ((slot "parameter") "ordinal") (parameter "ordinal"))) nil))
				(list "output" (slot "slot")))))
		(list name expression))))))
(define tds_rpc_return_values (lambda (slots caller)
	/* Transport owns token ordering; retain original incoming parameter order. */
	(map slots
		(lambda (slot) (begin
			(define parameter (slot "parameter"))
			(define descriptor (parameter "type"))
			(list "ordinal" (parameter "ordinal") "name" (parameter "name") "status" 1
				"user_type" 0 "flags" 0 "type" descriptor
				"value" (tds_result_value descriptor (caller (concat "__tsql_parameter:" (slot "slot"))))))))))
(define tds_result_callbacks (lambda (session output row fields) (list
	(lambda (values) (row (mapIndex (output "descriptors") (lambda (index descriptor)
		(tds_result_value descriptor (nth values (+ 1 (* 2 index))))))))
	(lambda (_names descriptors) (begin
		(define completion (if (output "descriptors")
			(merge (tds_session_completion session) (list "count_rows" true "more" true "no_count" (coalesceNil (session "__tsql_nocount") false))) nil))
		(output "descriptors" descriptors)
		(if completion (fields descriptors completion) (fields descriptors)))))))
(define tds_execute_procedure (lambda (session payload row fields state sequence) (begin
	(if (payload "is_rpc") true (error "named procedure execution requires an RPC request"))
	(session "__tsql_bulk_target" nil)
	(define address (tds_rpc_address (payload "procedure")))
	(define slots (tds_rpc_output_slots (payload "parameters")))
	(define output (newsession))
	(define callbacks (tds_result_callbacks session output row fields))
	(define returned (newsession))
	(define status (with_query_request (tsql_connection_session session) state sequence "procedure RPC" "acid" (lambda (statement) (begin
		(tsql_require_database_access (tsql_session_actor session) (session "schema"))
		(define caller (if (empty_list? slots) session (tds_rpc_output_scope session slots)))
		(define command (list "execute" (car address) (cadr address) (tds_rpc_arguments (payload "parameters") slots)))
		/* Do not put invocation values into the shared syntax/control-flow cache.
		The existing command owner performs ordinary EXEC status/diagnostics;
		its module callable receives fresh bound values on every invocation. */
		(define result (if (caller "__tsql_noexec") (begin (tsql_noexec_compile_command command caller) 0)
			(tsql_run_command command statement caller (car callbacks) (cadr callbacks))))
		(if (payload "is_rpc") (returned "values" (tds_rpc_return_values slots caller)) true)
		result))))
	/* Return status and requested scalar outputs are final RPC completion data. */
	(merge (tds_session_completion session)
		(list "rows" 0 "count_rows" (not (nil? (output "descriptors"))))
		(if (payload "is_rpc") (list "return_status" (intdiv status 1) "return_values" (coalesceNil (returned "values") '())) '())
		(if (session "__tsql_nocount") (list "no_count" true) '())))))

/* Prepared RPC controls are stripped by transport, but parameter ordinals are
not rebased. Dynamic SQL owns its own declared input scope and module finish;
the protocol's caller frame holds only requested by-reference scalar slots. */
(define tds_execute_parameterized_outputs (lambda (schema payload declarations parameters session statement resultrow resultfields returned) (begin
	(if (payload "is_rpc") true (error "RPC OUTPUT values require an RPC request"))
	(define named (tds_parameter_bindings declarations parameters))
	(define slots (tds_rpc_output_slots parameters))
	(define caller (tds_rpc_output_scope session slots))
	(define input (tsql_scope_session session declarations nil false))
	(tds_bind_parameters input declarations parameters)
	(define outputs (map slots (lambda (slot) (begin
		(define parameter (slot "parameter"))
		(define name (car (find (extract_assoc named (lambda (name value) (list name value)))
			(lambda (entry) (equal? ((cadr entry) "ordinal") (parameter "ordinal"))) nil)))
		(list name (slot "slot") (cadr (slot "declaration")))))))
	(tsql_execute_dynamic schema (payload "text") declarations
		(map declarations (lambda (declaration) (list (car declaration) (input (concat "__tsql_parameter:" (car declaration))))))
		statement caller resultrow resultfields
		(lambda (result status scoped) (begin
			(tsql_copy_outputs outputs scoped statement caller)
			(returned "values" (tds_rpc_return_values slots caller))
			(returned "status" status)
			result))))))

/* Network callbacks use the same accounts, transaction owner and SQL execution
path as the other frontends. Wire actions are interpreted here, never in scm/. */
(define tds_frontend_value (lambda (event original_session payload row fields message state sequence) (begin
	(define session (if (and original_session (or (equal? event "execute")
		(and (equal? event "metadata") (not (tds_metadata_procedure? (payload "procedure"))))))
		(tsql_message_session original_session message) original_session))
	(if (and session (not (has? '("login" "close") event))) (tsql_validate_login_identity session) true)
	(tsql_trigger_abort_boundary session (lambda () (match event
		"login" (begin
			(define user (payload "username"))
			(define database (database_name (payload "database") true))
			(define authenticated_id (tsql_authenticate user (payload "password")))
			(if (and database authenticated_id (equal? (tsql_has_dbaccess database (list user authenticated_id)) 1))
				(begin
					((tsql_policy (list user authenticated_id)) database true false)
					(define session (newsession))
					(session "username" user)
					(session "__tsql_principal_id" authenticated_id)
					(session "__tsql_client_hostname" (payload "hostname"))
					(session "__tsql_client_appname" (payload "appname"))
					(session "schema" database)
					(session "syntax" "tsql")
					(session "wire_metadata" true)
					(tsql_initialize_parse_context (tsql_initialize_date_context session))) nil))
		"state" (tds_session_completion session)
		"close" (begin (tx_rollback session) (tsql_cursor_close_all session) (tsql_temporary_close session))
		"reset" (begin
			(tx_rollback session)
			(tsql_cursor_close_all session)
			(tsql_temporary_close session)
			(define replacement (newsession))
			(replacement "username" (session "username"))
			(replacement "__tsql_principal_id" (session "__tsql_principal_id"))
			(replacement "__tsql_client_hostname" (session "__tsql_client_hostname"))
			(replacement "__tsql_client_appname" (session "__tsql_client_appname"))
			(replacement "schema" (session "schema"))
			(replacement "syntax" "tsql")
			(replacement "wire_metadata" true)
			(tsql_initialize_parse_context (tsql_initialize_date_context replacement)))
		"transaction" (begin
			(session "__tsql_bulk_target" nil)
			(if (and (equal? (payload "action") 1) (not (has? '(0 2) (payload "isolation"))))
				(error "unsupported TDS transaction isolation") true)
			(define sql (match (payload "action") 1 "BEGIN TRANSACTION" 2 "COMMIT" 3 "ROLLBACK" _ (error "unknown transaction action")))
			(tsql_execute_request (session "schema") sql session state sequence (lambda (_row) true) (lambda args true))
			(session "__tds_wire_transaction" (equal? (payload "action") 1))
			(list "rows" 0 "transaction" (payload "action")))
		"execute" (begin
			(session "__tsql_bulk_target" nil)
			(define output (newsession))
			(define returned (newsession))
			(define callbacks (tds_result_callbacks session output row fields))
			(define result (with_query_request (tsql_connection_session session) state sequence (payload "text") "acid" (lambda (statement) (begin
				(tsql_require_database_access (tsql_session_actor session) (database_name (payload "database") true))
				(define declarations (tds_bind_declarations (database_name (payload "database") true) (payload "declarations")))
				(if (find (payload "parameters") (lambda (parameter) (equal? (parameter "flags") 1)) nil)
					(tds_execute_parameterized_outputs (database_name (payload "database") true) payload declarations (payload "parameters")
						session statement (car callbacks) (cadr callbacks) returned)
					(begin
						(tds_bind_parameters session declarations (payload "parameters"))
						(tsql_execute_batches (database_name (payload "database") true) (payload "text") session statement
							(car callbacks) (cadr callbacks))))))))
			/* NOCOUNT changes protocol completion only; native DML still computes
			its affected-row count and result rows follow the shared planner. */
			(merge (tds_session_completion session)
				(list "rows" (if (number? result) result 0) "count_rows" (not (nil? (output "descriptors"))))
				(if (and (payload "is_rpc") (returned "values"))
					(list "return_status" (intdiv (returned "status") 1) "return_values" (returned "values")) '())
				(if (or (tsql_completion_count_absent? result) (session "__tsql_nocount"))
					(list "no_count" true) '())))
		"bulk" (begin
			(define count (tsql_run_bulk session payload state sequence))
			(merge (tds_session_completion session) (list "rows" count)))
		"describe" (tds_describe (database_name (payload "database") true) (payload "text") (payload "declarations") session)
		"metadata" (if (not (tds_metadata_procedure? (payload "procedure")))
			(tds_execute_procedure session payload row fields state sequence)
			(begin
				(tsql_require_database_access (tsql_session_actor session) (session "schema"))
				(define arguments (map (payload "parameters") (lambda (parameter) (begin
					(if (equal? (parameter "flags") 0) true (error "metadata output parameters are unsupported"))
					(if (has? '(31 38 104 167 175 231 239) ((parameter "type") "kind")) true
						(error "unsupported metadata argument wire type"))
					(list (parameter "name") (parameter "value"))))))
				(define result (with_query_request (tsql_connection_session session) state sequence "metadata procedure" "acid" (lambda (statement)
					(tsql_statement_call statement session "metadata procedure" false (lambda (tx)
						(tsql_metadata_result (session "schema") (payload "procedure") arguments
							(tsql_policy (tsql_session_actor session)) (tsql_session_actor session) tx))))))
				(fields (result "columns"))
				(map (result "rows") (lambda (values)
					(row (map (result "columns") (lambda (column) (begin
						(define value (values (column "name")))
						(if (nil? value) nil (equal? (column "kind") 231) (string value) (intdiv value 1))))))))
				(list "rows" (count (result "rows")))))
		_ (error "unknown TDS frontend callback")))))))
(define tds_frontend (lambda (event session payload row fields message state sequence)
	(if (and session (has? '("execute" "bulk" "metadata") event))
		(tsql_trigger_abort_boundary session (lambda () (tsql_finish_request session (lambda ()
			(tds_frontend_value event session payload row fields message state sequence)))))
		(tds_frontend_value event session payload row fields message state sequence))))
(import "tsql-checks.scm")
(import "tsql-cursors.scm")
(import "tsql-dynamic-cursors.scm")
(import "tsql-statement-triggers.scm")
(import "tsql-statement-trigger-runtime.scm")
(import "tsql-trigger-continuation.scm")
(import "tsql-noexec.scm")
/* The operator supplies files only at startup. The transport receives PEM
values, and no certificate/key path enters frontend sessions or SQL metadata. */
(define tds_tls_cli_config (lambda (policy certificate_path private_key_path load_file) (begin
	(if (or (not (string? policy)) (not (has? '("default" "optional" "required") policy)))
		(error "unknown TDS TLS policy"))
	(if (and (nil? certificate_path) (nil? private_key_path))
		(if (equal? policy "default") nil
			(error "TDS TLS policy requires certificate and private key files"))
		(begin
			(if (or (nil? certificate_path) (nil? private_key_path))
				(error "TDS TLS requires both certificate and private key files"))
			(if (or (not (string? certificate_path)) (not (string? private_key_path))
				(equal? certificate_path "") (equal? private_key_path ""))
				(error "TDS TLS certificate and key paths must be nonempty strings"))
			(list "policy" policy "certificate" (load_file certificate_path)
				"private_key" (load_file private_key_path)))))))
(if (arg "tds-port" nil) (begin
	(define port (arg "tds-port" nil))
	(define tls_options (tds_tls_cli_config (arg "tds-tls-policy" "default")
		(arg "tds-tls-cert" nil) (arg "tds-tls-key" nil)
		(lambda (filename) (readfile (if (equal? (substr filename 0 1) "/")
			filename (concat __CWD__ "/" filename))))))
	(tds port tds_frontend (arg "tds-database" "memcp-tests") tsql_session_collation tls_options)
	(service_registry "T-SQL protocol" (list port "" "TDS"))
	(print "TDS listener on port " port)))
