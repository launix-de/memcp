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
			(if binding (cdr binding) (error "local temporary table does not exist: " name)))
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

(define tsql_select_into_column (lambda (field) (begin
	(define contract (field "sql_datatype"))
	(define type (if (list? contract) (car contract) contract))
	(define dimensions (if (list? contract) (cadr contract) '()))
	(if (or (nil? type) (has? '("any" "NULL" "ROWVERSION" "TIMESTAMP") type))
		(error "SELECT INTO requires an authoritative ordinary SQL result type") true)
	/* IDENTITY inheritance has additional query-shape rules. Reject it until
	that producer contract is represented, rather than silently lose identity. */
	(if (not (equal? (bit_and (field "flags") 16) 0))
		(error "SELECT INTO identity inheritance is not supported") true)
	(list "column" (field "name") type dimensions
		(merge (list (list "null" (not (equal? (bit_and (field "flags") 1) 0)))
			(if (field "sql_collation") (list "collate" (car (field "sql_collation"))) '())))))))

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
	(if (and temporary (find (tsql_temporary_local_bindings scope tx) (lambda (binding)
		(and (equal?? (car binding) name)
			(not (and (> (count binding) 3) (equal? (nth binding 3) "trigger_temporary"))))) nil))
		(error "local temporary table already exists: " name) true)
	(define schema (session "schema"))
	/* Temporary qualifiers never select another database. Persistent targets
	use the ordinary database permission and native declaration owners. */
	(define destination (if temporary schema (tsql_qualifier_database owner schema session)))
	(if temporary true (tsql_require_permission (tsql_session_actor session) destination "CREATE TABLE"))
	(define fields (tsql_describe_statement tx schema (concat definitions query)
		(coalesceNil (session "__tsql_parameter_declarations") '()) session))
	(if (empty_list? fields) (error "SELECT INTO requires result columns") true)
	(define names (map fields (lambda (field) (field "name"))))
	(if (equal? (count names) (count (merge_unique (list (map names toLower))))) true
		(error "SELECT INTO has duplicate result column names"))
	(define columns (map fields tsql_select_into_column))
	(if (not temporary) (tsql_create_relation destination name columns '() false tx session)
		trigger (begin
			/* Completion borrows the caller's still-running mutation. Its native result
			lifetime must therefore outlive the lexical trigger module. */
			(define native (tx_get_or_compute_resource tx (concat "temporary-result-scope/" (scope "id"))
				(lambda () (new_relation_scope (tsql_connection_session session)))))
			(define native_name (concat "." (scope "id") name ":" (uuid)))
			(tsql_create_relation schema native_name columns
				(list "engine" "memory" "ephemeral" true "scope" native) false tx session)
			(scope "bindings" (cons (list name schema native_name) (scope "bindings"))))
		(tsql_create_temporary schema name columns '() false tx session))
	/* The normal INSERT SELECT compiler owns the complete source stream, casts,
	row effects and count. This rebuild quotes only parsed/declared identifiers. */
	(define quoted (lambda (name) (concat "[" (replace name "]" "]]") "]")))
	/* Keep the CTE before the mutation, so the ordinary CTE binder owns its
	definitions for both description and the INSERT SELECT source. */
	(define target (if temporary (quoted name) (concat (quoted destination) ".[dbo]." (quoted name))))
	(concat definitions "INSERT INTO " target " (" (reduce (map names quoted)
		(lambda (text column) (concat text (if (equal? text "") "" ",") column)) "") ") " query))))
(define tsql_run_select_into (lambda (name query definitions owner statement session) (begin
	(define populate (lambda (tx input)
		(tsql_execute_statement tx (session "schema") input session (lambda (_row) true) (lambda args true))))
	(if (or (session "transaction") (session "__tsql_trigger_event"))
		(statement "SELECT INTO" false (lambda (tx)
			(populate tx (tsql_prepare_select_into name query definitions owner tx session))))
		(begin
			(define input (statement "SELECT INTO declaration" false (lambda (tx)
				(tsql_prepare_select_into name query definitions owner tx session))))
			(statement "SELECT INTO" false (lambda (tx) (populate tx input))))))))
