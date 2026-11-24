/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* A temporary relation is an ordinary native MEMORY table. Only SQL name
binding and lexical lifetime live here. The engine owns rows, constraints,
transactions and portable runtime schema; ephemeral declarations are not durable.
Lexical declarations bind SQL names to opaque native schema slots. Resolved
immutable compile snapshots contain logical name, native database and physical
generation name, never scope capabilities or a frontend undo journal. */
(define tsql_temporary_name? (lambda (name)
	(and (string? name) (regexp_test name "^#"))))
(define tsql_temporary_check_name (lambda (name)
	(if (regexp_test name "^##") (error "global temporary tables are not supported") true)))
/* Lifetime control is local to this scope's hidden state. Its public bindings
remain ordinary session values that can be projected to a worker. A read or
cleanup must reuse the original capability, even after failed retirement. */
(define tsql_temporary_native_scope (lambda (scope initialize)
	(scope "get_or_compute_scoped" "temporary-relations" "native-lifetime"
		(coalesceNil initialize (lambda () (error "temporary scope has no native lifetime"))))))
(define tsql_temporary_new_scope (lambda (parent session) (begin
	(define scope (newsession))
	(scope "id" (uuid))
	(scope "bindings" '())
	(scope "parent" parent)
	(tsql_temporary_native_scope scope (lambda ()
		(new_relation_scope session (if parent (tsql_temporary_native_scope parent nil) nil) "transactional")))
	(scope "cache" (newcachemap))
	scope)))
(define tsql_temporary_scope (lambda (session) (begin
	(define scope (session "__tsql_temporary_scope"))
	(if scope scope (begin
		(define owner (tsql_connection_session session))
		(define scope (owner "get_or_compute_scoped" "connection" "temporary-relations"
			(lambda () (tsql_temporary_new_scope nil owner))))
		(owner "__tsql_temporary_scope" scope)
		scope)))))
/* A lexical declaration owns one opaque native slot. Its current physical
identity is resolved by the native schema decision, including rollback of a
DROP/recreate chain. The frontend never keeps an undo journal or table values. */
(define tsql_temporary_local_bindings (lambda (scope tx)
	(filter (map (scope "bindings") (lambda (entry)
		(if (and (> (count entry) 3) (equal? (nth entry 3) "scoped_schema"))
			(begin
				(define relation (relation_scope_binding (tsql_temporary_native_scope scope nil) (nth entry 2) tx))
				(if relation (cons (car entry) relation) nil))
			entry))) (lambda (entry) (not (nil? entry))))))
(define tsql_temporary_visible (lambda (scope tx) (begin
	(define local (tsql_temporary_local_bindings scope tx))
	(define inherited (if (scope "parent") (tsql_temporary_visible (scope "parent") tx) '()))
	(merge (list local (filter inherited (lambda (entry)
		(not (find local (lambda (own) (equal?? (car own) (car entry))) nil)))))))))
(define tsql_temporary_bindings (lambda (session)
	(if (session "__tsql_temporary_planning")
		(coalesceNil (session "__tsql_temporary_bindings") '())
		(tsql_temporary_visible (tsql_temporary_scope session) (session "__memcp_tx")))))
(define tsql_temporary_binding (lambda (name session) (begin
	(tsql_temporary_check_name name)
	(find (tsql_temporary_bindings session) (lambda (entry) (equal?? (car entry) name)) nil))))
(define tsql_transition_binding (lambda (database name session)
	(if (and (string? name) (has? '("inserted" "deleted") (toLower name))) (begin
		/* Ordinary relation lookup must not create a temporary scope just to
		probe these two reserved completion names. Planning uses only its already
		snapshotted bindings; execution borrows an existing lexical scope. */
		(define scope (session "__tsql_temporary_scope"))
		(define bindings (if (session "__tsql_temporary_planning")
			(coalesceNil (session "__tsql_temporary_bindings") '())
			(if scope (tsql_temporary_visible scope (session "__memcp_tx")) '())))
		(find bindings (lambda (entry)
			(and (> (count entry) 3) (equal? (nth entry 3) "transition")
				(equal? (cadr entry) database) (equal?? (car entry) name))) nil)) nil)))
(define tsql_resolve_relation (lambda (database name session tx) (begin
	(if (sql_private_relation? name) (error "private SQL relation names cannot be referenced directly") true)
	(define transition (tsql_transition_binding database name session))
	(if transition (list (cadr transition) (nth transition 2))
		(tsql_temporary_name? name) (begin
			(define binding (tsql_temporary_binding name session))
			(if binding (cdr binding)
				(session "__tsql_compile_only") (list database name)
				(error "local temporary table does not exist: " name)))
		(list database (tsql_relation_name database name tx))))))
(define tsql_temporary_database (lambda (database name session tx)
	(if (or (tsql_temporary_name? name) (tsql_transition_binding database name session))
		(car (tsql_resolve_relation database name session tx)) database)))
(define tsql_temporary_policy (lambda (username bindings) (begin
	(define account (tsql_policy username))
	(lambda (database name write)
		(if (sql_private_relation? name)
			(begin
				(define binding (find bindings (lambda (binding)
					(and (equal? database (cadr binding)) (equal? name (nth binding 2)))) nil))
				(if (nil? binding) (error "access denied to private SQL relation")
					(and write (> (count binding) 3) (equal? (nth binding 3) "transition"))
					(error "transition relations are read only")
					(and write (> (count binding) 3) (equal? (nth binding 3) "trigger_temporary"))
					(error "temporary writes inside statement triggers are not supported")
					(account database true write)))
			(account database name write))))))
(define tsql_create_temporary (lambda (database name columns options ifnotexists tx session) (begin
	(tsql_temporary_check_name name)
	(define scope (tsql_temporary_scope session))
	(define local (scope "bindings"))
	(define existing (find local (lambda (entry) (equal?? (car entry) name)) nil))
	(define scoped (if existing (and (> (count existing) 3) (equal? (nth existing 3) "scoped_schema")) false))
	(define native (tsql_temporary_native_scope scope nil))
	(define present (if scoped (relation_scope_binding native (nth existing 2) tx) existing))
	(if present (if ifnotexists false (error "local temporary table already exists: " name)) (begin
		(define key (if scoped (nth existing 2) (uuid)))
		(define native_name (concat "." (scope "id") name ":" (uuid)))
		(if (find columns (lambda (column) (equal? (car column) "foreign")) nil)
			(error "temporary foreign keys are not supported") true)
		(tsql_create_relation database native_name columns
			(merge options (list "engine" "memory" "ephemeral" true "scope" native "scope_key" key)) false tx session)
		(if scoped true (scope "bindings" (cons (list name database key "scoped_schema") local)))
		true)))))
(define tsql_drop_temporary (lambda (name ifexists session tx) (begin
	(define trigger (not (nil? (session "__tsql_trigger_event"))))
	(define scope (tsql_temporary_scope session))
	(define binding (tsql_temporary_binding name session))
	(if binding (begin
		(if trigger
			/* Completion results retain the caller's native mutation lifetime. */
			(if (and (has? (scope "bindings") binding) (equal? (count binding) 3))
				(scope "bindings" (filter (scope "bindings") (lambda (entry) (not (equal? entry binding)))))
				(error "cannot drop an inherited temporary relation inside a statement trigger"))
			(begin
				(define drop (lambda (scope) (begin
					(define own (find (scope "bindings") (lambda (entry)
						(and (equal?? (car entry) name) (> (count entry) 3) (equal? (nth entry 3) "scoped_schema"))) nil))
					(define native (tsql_temporary_native_scope scope nil))
					(define visible (if own (relation_scope_binding native (nth own 2) tx) nil))
					(if visible (drop_relation_scope_binding native (nth own 2) false tx)
						(scope "parent") (drop (scope "parent"))
						(error "local temporary table does not exist: " name)))))
				(drop scope)))
		true)
		ifexists false (error "local temporary table does not exist: " name)))))
(define tsql_temporary_cleanup_scope (lambda (scope) (begin
	/* Cleanup owns only this lexical scope, never an inherited caller table.
	The caller rolls back connection-owned transactions before session teardown. */
	(close_relation_scope (tsql_temporary_native_scope scope nil))
	(scope "bindings" '())
	(scope "cache" (newcachemap))
	true)))
(define tsql_temporary_close (lambda (session) (begin
	(define scope ((tsql_connection_session session) "__tsql_temporary_scope"))
	(if scope (tsql_temporary_cleanup_scope scope) true))))
(define tsql_temporary_with_module (lambda (session callback) (begin
	/* Keep the original exception intact, and run teardown exactly once outside
	the catch. A cleanup failure is an engine failure and must remain visible. */
	(define outcome (try (lambda () (list true (callback))) (lambda (failure) (list false failure))))
	(tsql_temporary_cleanup_scope (tsql_temporary_scope session))
	(if (car outcome) (cadr outcome) (error (cadr outcome))))))
(define tsql_temporary_query? (lambda (node) (match node
	((symbol query-block) _database sources fields condition grouping having ordering _limit _offset hidden stages _facts)
	(or (find sources (lambda (source)
		(if (source_is_base_table? source) (sql_private_relation? (source_relation source))
			(tsql_temporary_query? (source_relation source)))) nil)
		(tsql_temporary_query? (list fields condition grouping having ordering hidden stages)))
	((symbol quote) _value) false
	(cons _head _tail) (not (nil? (find node tsql_temporary_query? nil)))
	_ false)))

/* Preserve the SQL projection's original shape before inlining or join
optimization. Ordinary result flags alone cannot attest identity inheritance.
These records contain declaration data only, never a native handle or rows. */
(define tsql_select_into_identity_origin? (lambda (origin)
	(if (nil? origin) false (equal? ((cadr origin) "Extra") "auto_increment"))))
(define tsql_select_into_origin_key (lambda (origin)
	(list (source_schema (car origin)) (source_relation (car origin))
		(sql_column_storage_name (cadr origin)))))
/* FALSE proves an ordinary expression/declaration, TRUE traces an identity,
and NULL means provenance is unavailable. In particular a view's ordinary
result flags cannot prove that a direct export lacks an underlying identity. */
(define tsql_select_into_identity_status (lambda (sources facts info tx) (match (sql_info_formula info)
	((symbol get_column) alias alias_ic name name_ic) (begin
		(define source (source_for_alias sources (if (empty_list? sources) nil (source_alias (car sources))) alias alias_ic))
		(if (nil? source) nil
			(source_is_base_table? source)
			(if (tsql_native_relation_handle (source_schema source) (source_relation source) tx)
				(tsql_select_into_identity_origin? (tsql_result_origin sources facts info false)) nil)
			(match (source_relation source)
				((symbol query-block) _schema inner_sources fields _where _group _having _order _limit _offset _hidden _stages inner_facts)
				(begin
					(define expression ((expand_query_block_fields inner_sources fields) name))
					(define scope (tsql_result_derived_scope sources source inner_sources))
					(tsql_select_into_identity_status scope inner_facts (sql_expr_info scope expression) tx))
				((symbol union-block) _mode _branches _order _limit _offset _facts) false
				_ nil)))
	_ false)))
(define tsql_select_into_provenance (lambda (query expanded tx) (match query
	((symbol union-block) _mode _branches _order _limit _offset _facts)
	(begin
		/* UNION disqualifies identity inheritance. Its fully expanded first
		branch owns result provenance; evaluating the original view's delayed
		UNION conversion would attempt to type its unresolved ANY input. */
		(define branch (car (union_branches expanded)))
		(map (tsql_result_entries branch true) (lambda (entry) (begin
			(define status (tsql_select_into_identity_status (qb_sources branch) (qb_facts branch) (nth entry 2) tx))
			(if (identical? status false) nil
				(list "mode" (if status "discard" "unknown")))))))
	((symbol query-block) _schema sources _fields _where _group _having _order _limit _offset _hidden _stages facts)
	(begin
		(define entries (tsql_result_entries query false))
		(define expanded_entries (tsql_result_entries expanded true))
		(define origins (map entries (lambda (entry) (nth entry 3))))
		(define one_handle (if (equal? (count sources) 1)
			(if (source_is_base_table? (car sources))
				(tsql_native_relation_handle (source_schema (car sources)) (source_relation (car sources)) tx) nil) nil))
		/* A view wildcard has no native declaration before expansion. Its
		expanded entries own the complete output order and can positively prove
		an ordinary nullable export without inventing identity provenance. */
		(define aligned (equal? (count entries) (count expanded_entries)))
		(mapIndex expanded_entries (lambda (index expanded_entry) (begin
			(define origin (if aligned (nth origins index) nil))
			(if (tsql_select_into_identity_origin? origin) (begin
				(define key (tsql_select_into_origin_key origin))
				(define unique (equal? 1 (count (filter origins (lambda (other)
					(and (tsql_select_into_identity_origin? other) (expression_equal? key (tsql_select_into_origin_key other))))))))
				(list "mode" (if (or (> (count sources) 1) (not unique)) "discard" one_handle "inherit" "unknown")
					"source" key "source_uuid" (if one_handle (show one_handle "identity" tx) nil)))
				(begin
					(define status (tsql_select_into_identity_status (qb_sources expanded) (qb_facts expanded)
						(nth expanded_entry 2) tx))
					(if (identical? status false) nil
						(list "mode" (if (and status (> (count sources) 1)) "discard" "unknown")))))))))
	_ (error "SELECT INTO requires an authoritative logical projection"))))

(define tsql_select_into_identity_source (lambda (origins tx) (begin
	(if (find origins (lambda (origin) (if origin (equal? (origin "mode") "unknown") false)) nil)
		(error "SELECT INTO identity provenance requires a direct local base-table export") true)
	(define candidates (filter origins (lambda (origin) (if origin (equal? (origin "mode") "inherit") false))))
	(if (> (count candidates) 1) (error "SELECT INTO cannot inherit multiple identity columns") true)
	(define inherited (if (empty_list? candidates) nil (car candidates)))
	(if (nil? inherited) nil (begin
		(define source (inherited "source"))
		(define handle (tsql_native_relation_handle (car source) (cadr source) tx))
		(if (if handle (equal? (show handle "identity" tx) (inherited "source_uuid")) false) true
			(error "SELECT INTO source generation changed during description"))
		(define column (find (show handle "columns" tx) (lambda (column)
			(equal? (sql_column_storage_name column) (nth source 2))) nil))
		(if (if column (equal? (column "Extra") "auto_increment") false) true
			(error "SELECT INTO lost its original identity declaration"))
		(define state (table_id_state handle tx))
		(if (and (state "start") (equal? (state "table_uuid") (show handle "identity" tx))) true
			(error "SELECT INTO identity seed provenance is unavailable"))
		/* Existing native identities have increment one. Other increments are
		not admitted by the declaration grammar and are never guessed here. */
		(define seed (tsql_parse_integer (state "start")))
		(if (and (>= seed 1) (<= seed (tsql_identity_maximum (column "RawType") (column "Dimensions")))) true
			(error "SELECT INTO requires a supported positive identity seed"))
		(list "handle" handle "source" source "seed" seed))))))

(define tsql_select_into_column (lambda (field origin) (begin
	(define contract (field "sql_datatype"))
	(define type (if (list? contract) (car contract) contract))
	(define dimensions (if (list? contract) (cadr contract) '()))
	/* An empty constant expression can have width zero, while an ordinary
	character column must retain the minimum legal declaration width. */
	(define dimensions (if (and (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
		(equal? dimensions '(0))) '(1) dimensions))
	(if (or (nil? type) (has? '("any" "NULL" "ROWVERSION" "TIMESTAMP") type))
		(error "SELECT INTO requires an authoritative ordinary SQL result type") true)
	(if (and (nil? origin) (not (equal? (bit_and (field "flags") 16) 0)))
		(error "SELECT INTO identity projection provenance is unavailable") true)
	(define inherited (if origin (equal? (origin "mode") "inherit") false))
	(list "column" (field "name") type dimensions
		(merge (list (list "null" (if origin false (not (equal? (bit_and (field "flags") 1) 0))))
			(if inherited '("auto_increment" true) '())
			(if (field "sql_collation") (list "collate" (car (field "sql_collation"))) '())))))))

/* This immutable invocation wrapper is not installed on the connection and
enumerates only its parent. The exact copy capability cannot enter a cache key,
session projection or another statement's public IDENTITY_INSERT setting. */
(define tsql_select_into_population_session (lambda (parent copy)
	(lambda args (match args
		'() (parent)
		'(key) (if (equal? key "__tsql_select_into_copy") copy (parent key))
		'(key value) (if (equal? key "__tsql_select_into_copy")
			(error "SELECT INTO copy target is immutable") (parent key value))
		_ (apply parent args)))))

/* SQL Server SELECT INTO semantics:
https://learn.microsoft.com/en-us/sql/t-sql/queries/select-into-clause-transact-sql
SELECT INTO has two ordinary autocommit decisions: failed population
leaves the empty declaration. An explicit transaction and a trigger completion
instead retain their existing actual transaction across both operations. */
(define tsql_prepare_select_into (lambda (name query definitions owner tx session) (begin
	(define trigger (not (nil? (session "__tsql_trigger_event"))))
	(define temporary (tsql_temporary_name? name))
	(define scope (if temporary (tsql_temporary_scope session) nil))
	(if temporary (tsql_temporary_check_name name)
		(sql_private_relation? name) (error "private SQL relation names cannot be created directly") true)
	(if (and trigger (not temporary)) (error "persistent SELECT INTO is unsupported inside a statement trigger") true)
	(if (and (session "__tsql_insert_exec") (not temporary))
		(error "persistent SELECT INTO is unsupported inside INSERT EXEC") true)
	(if (and temporary (find (tsql_temporary_local_bindings scope tx) (lambda (binding)
		(and (equal?? (car binding) name)
			(not (and (> (count binding) 3) (equal? (nth binding 3) "trigger_temporary"))))) nil))
		(error "local temporary table already exists: " name) true)
	(define schema (session "schema"))
	/* Temporary qualifiers never select another database. Persistent targets
	use the ordinary database permission and native declaration owners. */
	(define destination (if temporary schema (tsql_qualifier_database owner schema session)))
	(if temporary true (tsql_require_permission (tsql_session_actor session) destination "CREATE TABLE"))
	(define description (tsql_describe_statement tx schema (concat definitions query)
		(coalesceNil (session "__tsql_parameter_declarations") '()) session true))
	(define fields (car description))
	(define origins (cadr description))
	(if (empty_list? fields) (error "SELECT INTO requires result columns") true)
	(define names (map fields (lambda (field) (field "name"))))
	(if (equal? (count names) (count (merge_unique (list (map names toLower))))) true
		(error "SELECT INTO has duplicate result column names"))
	(define identity_source (tsql_select_into_identity_source origins tx))
	(define columns (mapIndex fields (lambda (index field) (tsql_select_into_column field (nth origins index)))))
	(define options (if identity_source (list "auto_increment" (identity_source "seed")) '()))
	(if (not temporary) (tsql_create_relation destination name columns options false tx session)
		trigger (begin
			/* Completion borrows the caller's still-running mutation. Its native result
			lifetime must therefore outlive the lexical trigger module. */
			(define native (tx_get_or_compute_resource tx (concat "temporary-result-scope/" (scope "id"))
				(lambda () (new_relation_scope (tsql_connection_session session)))))
			(define native_name (concat "." (scope "id") name ":" (uuid)))
			(tsql_create_relation schema native_name columns
				(merge options (list "engine" "memory" "ephemeral" true "scope" native)) false tx session)
			(scope "bindings" (cons (list name schema native_name) (scope "bindings"))))
		(tsql_create_temporary schema name columns options false tx session))
	(define resolved (if temporary (tsql_resolve_relation schema name session tx) (list destination name)))
	(define handle (tsql_native_relation_handle (car resolved) (cadr resolved) tx))
	(define identity (if identity_source (find (show handle "columns" tx)
		(lambda (column) (equal? (column "Extra") "auto_increment")) nil) nil))
	/* The normal INSERT SELECT compiler owns the complete source stream, casts,
	row effects and count. This rebuild quotes only parsed/declared identifiers. */
	(define quoted (lambda (name) (concat "[" (replace name "]" "]]") "]")))
	/* Keep the CTE before the mutation, so the ordinary CTE binder owns its
	definitions for both description and the INSERT SELECT source. */
	(define target (if temporary (quoted name) (concat (quoted destination) ".[dbo]." (quoted name))))
	(list (concat definitions "INSERT INTO " target " (" (reduce (map names quoted)
		(lambda (text column) (concat text (if (equal? text "") "" ",") column)) "") ") " query)
		(if identity (list handle (sql_column_storage_name identity) (identity_source "seed")) nil) identity_source))))
(define tsql_run_select_into (lambda (name query definitions owner statement session) (begin
	(define populate (lambda (tx input) (begin
		(define identity_input (nth input 2))
		(if identity_input (begin
			(define binding (identity_input "source"))
			(if (identical? (identity_input "handle") (tsql_native_relation_handle (car binding) (cadr binding) tx)) true
				(error "SELECT INTO source generation changed before population"))) true)
		(tsql_execute_statement tx (session "schema") (car input)
			(tsql_select_into_population_session session (cadr input)) (lambda (_row) true) (lambda args true)))))
	(if (or (session "transaction") (session "__tsql_trigger_event"))
		(tsql_statement_call statement session "SELECT INTO" false (lambda (tx)
			(populate tx (tsql_prepare_select_into name query definitions owner tx session))))
		(begin
			(define input (tsql_statement_call statement session "SELECT INTO declaration" false (lambda (tx)
				(tsql_prepare_select_into name query definitions owner tx session))))
			(tsql_statement_call statement session "SELECT INTO" false (lambda (tx) (populate tx input))))))))
