/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Cursor state is frontend control metadata. Result cardinality lives only in
ordinary native scoped MEMORY storage; FETCH borrows one indexed ordinal row.
There is no native reader, requesting session or transaction captured by a cursor. */
(define tsql_cursor_options (lambda (options) (begin
	(if (equal? (count options) (count (merge_unique (list options)))) true (error "duplicate cursor option"))
	(map (list '("LOCAL" "GLOBAL") '("FORWARD_ONLY" "SCROLL") '("STATIC" "DYNAMIC" "FAST_FORWARD" "KEYSET")
		'("READ_ONLY" "SCROLL_LOCKS" "OPTIMISTIC")) (lambda (group)
			(if (> (count (filter group (lambda (option) (has? options option)))) 1) (error "conflicting cursor options") true)))
	(if (find options (lambda (option) (has? '("SCROLL_LOCKS" "OPTIMISTIC" "TYPE_WARNING") option)) nil)
		(error "unsupported cursor option") true)
	(if (and (has? options "FAST_FORWARD") (has? options "SCROLL")) (error "FAST_FORWARD cannot SCROLL") true)
	/* CURSOR_DEFAULT is not configurable in this frontend yet. Its supported
	connection profile is GLOBAL; explicit LOCAL always has lexical lifetime. */
	(list "local" (has? options "LOCAL") "read_only" (has? options "READ_ONLY")
		"kind" (if (has? options "STATIC") "static" (has? options "KEYSET") "keyset"
			(has? options "FAST_FORWARD") "fast_forward" "dynamic")
		"scroll" (if (or (has? options "FORWARD_ONLY") (has? options "FAST_FORWARD")) false
			(or (has? options "SCROLL") (has? options "STATIC") (has? options "KEYSET") (has? options "DYNAMIC")))))))
/* FOR UPDATE is a declaration intent, not native positioned-write authority.
Explicit STATIC/FAST_FORWARD/READ_ONLY conflicts precede implicit conversion.
https://learn.microsoft.com/sql/t-sql/language-elements/declare-cursor-transact-sql */
(define tsql_cursor_update_options (lambda (options clause) (if (nil? clause) options
	(set_assoc (set_assoc options "update" true) "update_columns" (clause "columns")))))
(define tsql_cursor_validate_update_options (lambda (options)
	(if (and (options "update") (or (options "read_only")
		(has? '("static" "fast_forward") (options "kind"))))
		(tsql_raise_error 1048 "FOR UPDATE conflicts with the declared cursor options" 1) true)))
/* Reuse the common binder's source catalog and case/qualifier rules. This scope
contains FROM exports only: a SELECT alias cannot grant an update-column name.
Only declaration-sized references survive; neither this scope nor its query is
retained in a cursor record. */
(define tsql_cursor_update_expression (lambda (expression) (match expression
	((symbol sql_compile_column) _compiler column) (tsql_cursor_update_expression column)
	((symbol sql_type_expr) value _type _collation) (tsql_cursor_update_expression value)
	_ expression)))
(define tsql_cursor_bind_update_column (lambda (sources expression) (begin
	(define entries (map sources (lambda (source)
		(list (source_alias source) source (binding_column_catalog (binding_source_columns source)) true))))
	(match (tsql_cursor_update_expression expression)
		((symbol get_column) alias alias_ic column column_ic) (begin
			(define matches (if (nil? alias) (binding_entries_for_column entries column column_ic)
				(binding_entries_for_alias entries alias alias_ic)))
			(if (empty_list? matches)
				(tsql_raise_error (if (nil? alias) 207 4104) "FOR UPDATE column could not be bound" 1)
				(> (count matches) 1) (tsql_raise_error 209 "ambiguous FOR UPDATE column" 1) true)
			(if (nil? (binding_catalog_column_name (binding_entry_columns (car matches)) column column_ic))
				(tsql_raise_error 207 "FOR UPDATE column does not exist" 1) true)
			(if (nil? alias) (bind_unqualified_column_in_scopes (list entries) column column_ic)
				(bind_qualified_column_in_scopes (list entries) alias alias_ic column column_ic)))
		_ (tsql_raise_error 412 "FOR UPDATE column is derived or constant" 1)))))
(define tsql_cursor_validate_update_column (lambda (sources expression) (begin
	(define reference (tsql_cursor_bind_update_column sources expression))
	(define source (source_for_alias sources nil (nth reference 1) false))
	(define relation (normalize_query_ast (source_relation source)))
	/* A declared base column is a valid OF name, including a computed base
	column. A computed derived export is different: it has no column lineage.
	This proves declaration binding only, not that an UPDATE will be writable. */
	(if (or (string? relation) (local_relation_reference? relation)) true
		(query_block? relation) (tsql_cursor_validate_update_column (qb_sources relation)
			(get_assoc (logical_relation_fields relation) (nth reference 3) nil))
		(tsql_raise_error 412 "FOR UPDATE column is derived or constant" 1)))))
(define tsql_cursor_validate_update (lambda (query options) (if (options "update") (begin
	(if (union_block? query)
		(tsql_raise_error 115 "FOR UPDATE is invalid with set operators" 1) true)
	(map (options "update_columns") (lambda (column)
		(tsql_cursor_validate_update_column (qb_sources query) column)))) true)))
(define tsql_cursor_scope_id (lambda (session) (begin
	(define scope (tsql_identity_scope session))
	(coalesceNil (scope "__tsql_cursor_scope_id") (begin
		(define id (uuid)) (scope "__tsql_cursor_scope_id" id) id)))))
(define tsql_cursor_directory (close_procedure ((lambda (tsql_connection_session tsql_identity_scope)
	(lambda (session global)
		(if global (tsql_connection_session session) (tsql_identity_scope session))))
	tsql_connection_session tsql_identity_scope)))
(define tsql_cursor_key (lambda (global) (if global "__tsql_global_cursors" "__tsql_local_cursors")))
(define tsql_cursor_entries (close_procedure ((lambda (tsql_cursor_directory tsql_cursor_key)
	(lambda (session global)
		(coalesceNil ((tsql_cursor_directory session global) (tsql_cursor_key global)) '())))
	tsql_cursor_directory tsql_cursor_key)))
/* The connection's last-opened status is a bounded scalar pair. A cursor may
retain this dedicated cell, which contains only its UUID and row count; it
never retains the connection, another cursor record or execution authority.
https://learn.microsoft.com/sql/t-sql/functions/cursor-rows-transact-sql */
(define tsql_cursor_rows_owner (close_procedure ((lambda (connection_session)
	(lambda (session)
		((connection_session session) "get_or_compute_scoped" 0 "__tsql_cursor_rows_owner"
			(lambda () (newsession))))) tsql_connection_session)))
(define tsql_cursor_rows (close_procedure ((lambda (status_owner)
	(lambda (session) (begin
		(define last ((status_owner session) "last"))
		(if (nil? last) 0 (cadr last))))) tsql_cursor_rows_owner)))
(define tsql_cursor_publish_rows (lambda (cursor rows)
	((cursor "status_owner") "last" (list (cursor "status_id") rows))))
(define tsql_cursor_retire_rows (lambda (cursor) (begin
	(define owner (cursor "status_owner"))
	(define last (owner "last"))
	(if (nil? last) true
		(equal? (car last) (cursor "status_id"))
		(owner "last" (list (car last) 0)) true))))
/* Names and variables are references to one cursor record. Variables live in
lexical control scope, outside enumerable SQL parameter values and plan inputs.
https://learn.microsoft.com/sql/t-sql/language-elements/deallocate-transact-sql */
(define tsql_cursor_variable_entries (close_procedure ((lambda (tsql_identity_scope)
	(lambda (session)
		(coalesceNil ((tsql_identity_scope session) "__tsql_cursor_variables") '()))) tsql_identity_scope)))
(define tsql_cursor_variable_contract (lambda (session name) (begin
	(define declaration (find (session "__tsql_parameter_declarations")
		(lambda (item) (equal? (car item) name)) nil))
	(if (and declaration (expression_equal? (cadr declaration) '("CURSOR"))) declaration
		(error "cursor variable requires a CURSOR declaration: @" name)))))
(define tsql_cursor_variable_entry (lambda (session name) (begin
	(tsql_cursor_variable_contract session name)
	(coalesceNil (find (tsql_cursor_variable_entries session)
		(lambda (entry) (equal? (car entry) name)) nil)
		(error "cursor variable has not been declared: @" name)))))
(define tsql_cursor_lookup (lambda (session name global) (match name
	'("variable" variable) (begin
		(if global (error "GLOBAL requires a cursor name, not a variable") true)
		(coalesceNil (cadr (tsql_cursor_variable_entry session variable))
			(error "cursor variable has no cursor reference: @" variable)))
	_ (begin
		(define local (if global nil (find (tsql_cursor_entries session false) (lambda (entry) (equal? (car entry) name)) nil)))
		(define entry (coalesceNil local (find (tsql_cursor_entries session true) (lambda (entry) (equal? (car entry) name)) nil)))
		(if entry (cadr entry) (error "cursor does not exist: " name))))))
(define tsql_cursor_retain (lambda (cursor) (begin
	(cursor "references" (+ (cursor "references") 1)) cursor)))
(define tsql_cursor_release (lambda (cursor) (if (nil? cursor) true (begin
	(define references (cursor "references"))
	(if (> references 0) true (error "cursor reference has already been retired"))
	/* Retain the last owner if native retirement fails, so cleanup can retry. */
	(if (equal? references 1) (tsql_cursor_close_record cursor) true)
	(cursor "references" (- references 1)) true))))
(define tsql_cursor_remove (lambda (session name global) (begin
	(define owner (tsql_cursor_directory session global))
	(define entry (find (tsql_cursor_entries session global) (lambda (entry) (equal? (car entry) name)) nil))
	(if entry (tsql_cursor_release (cadr entry)) true)
	(owner (tsql_cursor_key global) (filter (tsql_cursor_entries session global) (lambda (entry) (not (equal? (car entry) name)))))
	true)))
(define tsql_cursor_variable_declare (lambda (session name) (begin
	(tsql_cursor_variable_contract session name)
	(if (find (tsql_cursor_variable_entries session) (lambda (entry) (equal? (car entry) name)) nil)
		(error "duplicate cursor variable declaration: @" name) true)
	((tsql_identity_scope session) "__tsql_cursor_variables"
		(cons (list name nil) (tsql_cursor_variable_entries session))) 0)))
(define tsql_cursor_variable_set (lambda (session name cursor) (begin
	(define previous (tsql_cursor_variable_entry session name))
	/* Retain before release also handles SET @a=@a without closing its rowset. */
	(define retained (if cursor (tsql_cursor_retain cursor) nil))
	(define outcome (try (lambda () (begin (tsql_cursor_release (cadr previous)) true))
		(lambda (failure) (begin (if retained (tsql_cursor_release retained) true) (error failure)))))
	(if outcome (begin
		((tsql_identity_scope session) "__tsql_cursor_variables"
			(cons (list name retained) (filter (tsql_cursor_variable_entries session)
				(lambda (entry) (not (equal? (car entry) name)))))) 0)
		(error "cursor reference retirement did not complete")))))
(define tsql_cursor_reference_syntax (parser (define reference (or
	(parser (define name tsql_variable_name) (list "variable" name))
	(parser (define name tsql_identifier) (toLower name)))) reference
	"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))
(define tsql_cursor_assign_reference (lambda (name text statement session)
	(tsql_statement_call statement session "SET CURSOR REFERENCE" false (lambda (_tx)
		(tsql_cursor_variable_set session name
			(tsql_cursor_lookup session (tsql_cursor_reference_syntax text) false))))))
(define tsql_cursor_close_record (lambda (cursor) (begin
	(define scope (cursor "native"))
	(cursor "closing" true)
	(if scope (close_relation_scope scope) true)
	(cursor "native" nil) (cursor "table" nil) (cursor "open" false) (cursor "closing" false)
	(cursor "fields" nil) (cursor "result_formatter" nil) (cursor "position" 0) (cursor "count" 0)
	(cursor "dynamic_cache" nil) (cursor "dynamic_factory" nil) (cursor "source_contracts" nil)
	(cursor "bookmark" nil) (cursor "started" false) (cursor "eof" false)
	(tsql_cursor_retire_rows cursor)
	true)))
(define tsql_cursor_close_all (lambda (session) (begin
	(map (tsql_cursor_entries session true) (lambda (entry) (tsql_cursor_remove session (car entry) true)))
	true)))
(define tsql_cursor_with_scope (lambda (session callback) (begin
	(define outcome (try (lambda () (list true (callback))) (lambda (failure) (list false failure))))
	(map (tsql_cursor_entries session false) (lambda (entry) (tsql_cursor_remove session (car entry) false)))
	(map (tsql_cursor_variable_entries session) (lambda (entry)
		(tsql_cursor_variable_set session (car entry) nil)))
	((tsql_identity_scope session) "__tsql_cursor_variables" '())
	/* A dynamic transition cursor cannot escape the immutable source lifetime.
	An ordinary GLOBAL static rowset has its normal connection lifetime. */
	(define escaped (filter (tsql_cursor_entries session true) (lambda (entry) (begin
		(define cursor (cadr entry))
		(and (cursor "scoped_source") (equal? (cursor "owner_scope") (tsql_cursor_scope_id session)))))))
	(map escaped (lambda (entry) (tsql_cursor_remove session (car entry) true)))
	(if (not (car outcome)) (error (cadr outcome))
		(not (empty_list? escaped)) (error "scope-bound cursor must be DEALLOCATEd before leaving its declaring scope")
		(cadr outcome)))))

/* Parse/describe reuse the actual SQL parser, declaration binder and policy.
Only this declaration-sized IR proof permits a default DYNAMIC materialization:
a single authenticated immutable transition source, direct columns, fixed order
and no membership predicate, volatile projection, subquery, grouping or limit. */
(define tsql_cursor_planning (lambda (session) (begin
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" (coalesceNil (session "__tsql_parameter_declarations") '()))
	(planning "__tsql_temporary_bindings" (tsql_temporary_bindings session))
	(planning "tsql_import_target" (session "tsql_import_target"))
	(planning "tsql_import_source" (session "tsql_import_source"))
	(planning "__tsql_parameter_analysis" true)
	planning)))
(define tsql_cursor_transition_query? (lambda (query session) (begin
	(if (and (query_block? query) (equal? (count (qb_sources query)) 1)) (begin
		(define source (car (qb_sources query)))
		(define binding (find (tsql_temporary_bindings session) (lambda (binding)
			(and (> (count binding) 3) (equal? (nth binding 3) "transition")
				(equal? (cadr binding) (source_schema source)) (equal? (nth binding 2) (source_relation source)))) nil))
		(define direct (lambda (expression) (match (sql_info_formula (sql_expr_info (qb_sources query) expression))
			((symbol get_column) _alias _alias_ic _column _column_ic) true
			_ false)))
		(and binding (source_is_base_table? source) (not (source_outer? source))
			(or (nil? (source_join_expr source)) (expression_equal? (source_join_expr source) true))
			(expression_equal? (qb_where query) true)
			(empty_list? (coalesceNil (qb_group query) '())) (nil? (qb_having query))
			(empty_list? (coalesceNil (qb_stages query) '())) (nil? (qb_limit query)) (nil? (qb_offset query))
			(reduce (extract_assoc (expand_query_block_fields (qb_sources query) (qb_fields query)) (lambda (_name expression) (direct expression))) (lambda (ok item) (and ok item)) true)
			(reduce (coalesceNil (qb_order query) '()) (lambda (ok item) (and ok (direct (car item)))) true))) false))))

/* These are SQL Server's required implicit conversions, not a fallback for
arbitrary dynamic sources. Consume the SELECT's bound declaration facts only;
never infer uniqueness from current rows or a physical auto-index.
https://learn.microsoft.com/sql/t-sql/language-elements/set-local-variable-transact-sql */
(define tsql_cursor_unkeyed_input? (lambda (query)
	(if (union_block? query) (not (nil? (find (union_branches query) tsql_cursor_unkeyed_input? nil)))
		(query_block? query) (not (nil? (find (qb_sources query) (lambda (source)
			(if (string? (source_relation source))
				(and (not (nil? (source_declarations source))) (empty_list? (source_unique_key_sets source)))
				(tsql_cursor_unkeyed_input? (source_relation source)))) nil)))
		false)))
(define tsql_cursor_snapshot_query? (lambda (query)
	(if (union_block? query) true
		(query_block? query) (or
			(not (empty_list? (qb_group query))) (not (nil? (qb_having query)))
			(query_block_has_local_aggregates? query)
			(not (nil? (find (qb_sources query) (lambda (source)
				(tsql_cursor_snapshot_query? (source_relation source))) nil))))
		false)))
(define tsql_cursor_static_query? (lambda (query options)
	(or (tsql_cursor_snapshot_query? query)
		(and (query_block? query)
			(or (equal? (options "kind") "keyset") (and (options "iso") (options "scroll"))
				(not (empty_list? (qb_order query))))
			(tsql_cursor_unkeyed_input? query)))))
(define tsql_cursor_create_record (lambda (options text tx session) (begin
	(tsql_cursor_validate_update_options options)
	(define planning (tsql_cursor_planning session))
	(define bindings (tsql_temporary_bindings session))
	(define syntax (parse_tsql (session "schema") text
		(tsql_temporary_policy (session "username") bindings) planning tx))
	(define query (match syntax ((symbol quote) query) query _ (error "cursor requires a SELECT result")))
	(tsql_cursor_validate_update query options)
	(define options (if (tsql_cursor_static_query? query options) (set_assoc options "kind" "static") options))
	(define transition (tsql_cursor_transition_query? query session))
	(if (equal? (options "kind") "keyset")
		(error "keyset cursors require native current-row ownership in this profile") true)
	(define dynamic (if (and (equal? (options "kind") "dynamic") (not transition))
		(tsql_cursor_dynamic_recipe query options) nil))
	(if (and (equal? (options "kind") "dynamic") (not transition) (nil? dynamic))
		(error "dynamic cursor requires an admitted native primary-key traversal") true)
	/* Immutable transition sources implicitly convert DYNAMIC to STATIC.
	This effective membership contract also owns empty CURSOR_STATUS, row count
	and scrolling FETCH movement; retain the declaration's scroll flag. */
	(define options (if (and transition (equal? (options "kind") "dynamic"))
		(set_assoc options "kind" "static") options))
	(define options (set_assoc options "effective_read_only"
		(or (options "read_only") (has? '("static" "fast_forward") (options "kind")))))
	(define cursor (newsession))
	(cursor "status_id" (uuid)) (cursor "status_owner" (tsql_cursor_rows_owner session))
	(cursor "schema" (session "schema")) (cursor "text" text) (cursor "options" options)
	(cursor "transition" transition) (cursor "dynamic" (not (nil? dynamic)))
	(cursor "references" 0)
	(cursor "scoped_source" (tsql_temporary_query? query))
	(cursor "owner_scope" (tsql_cursor_scope_id session))
	(cursor "contracts" (filter (coalesceNil (session "__tsql_parameter_declarations") '())
		(lambda (declaration) (not (expression_equal? (cadr declaration) '("CURSOR"))))))
	(cursor "open" false)
	cursor)))
(define tsql_cursor_declare (lambda (name options text statement session)
	(tsql_statement_call statement session "DECLARE CURSOR" false (lambda (tx) (begin
		(define global (not (options "local")))
		(if (find (tsql_cursor_entries session global) (lambda (entry) (equal? (car entry) name)) nil)
			(error "cursor already exists: " name) true)
		(define cursor (tsql_cursor_create_record options text tx session))
		(cursor "global" global) (cursor "name" name)
		((tsql_cursor_directory session global) (tsql_cursor_key global)
			(cons (list name (tsql_cursor_retain cursor)) (tsql_cursor_entries session global))) 0)))))
(define tsql_cursor_assign_query (lambda (name options text statement session)
	(tsql_statement_call statement session "SET CURSOR" false (lambda (tx) (begin
		(tsql_cursor_variable_entry session name)
		(tsql_cursor_variable_set session name (tsql_cursor_create_record options text tx session)))))))
(define tsql_cursor_raw_session (lambda (parent)
	(lambda args (match args
		'() (merge_unique (list '("wire_metadata") (parent)))
		'("wire_metadata") true
		_ (apply parent args)))))
(define tsql_cursor_column (lambda (index field) (begin
	(define contract (field "sql_datatype"))
	(define type (if (list? contract) (car contract) contract))
	(define dimensions (if (list? contract) (cadr contract) '()))
	(if (or (nil? type) (has? '("any" "NULL") type)) (error "cursor result requires an authoritative declared SQL type") true)
	(list "column" (concat "c" index) type dimensions
		(if (field "sql_collation") (list "collate" (car (field "sql_collation"))) '())))))
(define tsql_cursor_open (lambda (name global statement session)
	(tsql_statement_call statement session "OPEN CURSOR" false (lambda (tx) (begin
		(define cursor (tsql_cursor_lookup session name global))
		(if (cursor "closing") (error "cursor retirement must finish before OPEN") true)
		(if (cursor "open") (error "cursor is already open: " name) true)
		(if (and (or (cursor "scoped_source") (not (empty_list? (cursor "contracts"))))
			(not (equal? (cursor "owner_scope") (tsql_cursor_scope_id session))))
			(error "cursor query binding is outside its declaring scope") true)
		(if (cursor "dynamic") (tsql_cursor_dynamic_open cursor session tx) (begin
			(define fields (tsql_describe_statement tx (cursor "schema") (cursor "text") (cursor "contracts") session false))
			(if (empty_list? fields) (error "cursor SELECT must return columns") true)
			(define columns (mapIndex fields tsql_cursor_column))
			(define native (new_relation_scope (tsql_connection_session session)))
			(define native_name (concat ".cursor:" (uuid)))
			(define outcome (try (lambda () (begin
				/* This is independent scratch materialization, not a user mutation or
				second transaction. Source reads retain their actual invocation tx. */
				(createtable (cursor "schema") native_name
					(merge (list (list (list "column" "__ordinal" "BIGINT" '() '("auto_increment" true))
						(list "unique" "PRIMARY" '("__ordinal"))) columns))
					(list "engine" "memory" "ephemeral" true "scope" native) false nil)
				(define handle (table (cursor "schema") false native_name false nil))
				(define names (mapIndex fields (lambda (index _field) (concat "c" index))))
				(tsql_execute_statement tx (cursor "schema") (cursor "text") (tsql_cursor_raw_session session)
					(lambda (row) (begin
						(if (equal? (count row) (* 2 (count fields))) true (error "cursor result width changed during OPEN"))
						(insert handle names (list (map (produceN (count fields)) (lambda (index) (nth row (+ 1 (* 2 index)))))) '() nil false nil nil)
						1)) (lambda args true))
				(define count_rows (scan nil handle '() '() '() (lambda () true) '("__ordinal") (lambda (acc id) (+ acc 1)) 0 +))
				(list true handle count_rows names))) (lambda (failure) (list false failure))))
			(if (not (car outcome)) (begin (close_relation_scope native) (error (cadr outcome))) (begin
				(cursor "native" native) (cursor "table" (cadr outcome)) (cursor "count" (nth outcome 2))
				(cursor "columns" (nth outcome 3)) (cursor "fields" fields) (cursor "position" 0) (cursor "open" true)
				/* A failed OPEN leaves the preceding successful OPEN's status intact.
				There is no asynchronous population profile here. A future genuine
				dynamic operator retains its sentinel even with no qualifying rows. */
				(tsql_cursor_publish_rows cursor (if (equal? ((cursor "options") "kind") "dynamic") -1 (cursor "count")))
				0)))))))))

(define tsql_cursor_fetch_position (lambda (cursor movement amount)
	(match movement
		"next" (+ (cursor "position") 1) "prior" (- (cursor "position") 1)
		"first" 1 "last" (cursor "count")
		"absolute" (if (< amount 0) (+ (cursor "count") amount 1) amount)
		"relative" (+ (cursor "position") amount)
		_ (error "unsupported FETCH direction"))))
/* A result-set FETCH borrows the same single ordinal row as FETCH INTO.
Its formatter is compiled lazily from OPEN's immutable SQL declarations, then
receives the current output/session as arguments. It retains no cursor,
connection or transaction capability and performs no SQL parse or scan.
https://learn.microsoft.com/sql/t-sql/language-elements/fetch-transact-sql */
/* SQL Server's T-SQL FETCH also carries the trailing cursor row status.
It is hidden browse metadata, not an extra visible projection or INTO slot.
The existing generic TDS writer emits COLINFO's hidden bit for this contract.
https://learn.microsoft.com/sql/t-sql/functions/cursor-rows-transact-sql
https://learn.microsoft.com/sql/relational-databases/system-stored-procedures/sp-cursorfetch-transact-sql */
(define tsql_cursor_result_fields (lambda (fields)
	(append fields (merge (tsql_wire_contract "INT" '())
		(list "name" "ROWSTAT" "sql_type" "INT" "sql_datatype" "INT" "flags" 8192 "user_type" 0 "browse" true)))))
(define tsql_cursor_result_formatter (lambda (fields)
	(close_procedure (eval (optimize (list 'lambda (list 'session 'resultrow '__tsql_fetch_row)
		(tsql_result_output_plan (list 'resultrow '__tsql_fetch_row) fields)))))))
(define tsql_cursor_fetch (lambda (name global movement targets statement session resultrow resultfields)
	(tsql_statement_call statement session "FETCH CURSOR" false (lambda (tx) (begin
		(define cursor (tsql_cursor_lookup session name global))
		(tsql_require_database_access (tsql_session_actor session) (cursor "schema"))
		(if (cursor "closing") (error "cursor retirement must finish before FETCH") true)
		(if (cursor "open") true (error "cursor is not open: " name))
		(define fields (cursor "fields"))
		(if targets (if (equal? (count targets) (count fields)) true (error "FETCH variable count differs from cursor projection")) true)
		(define declarations (map (coalesceNil targets '()) (lambda (target)
			(coalesceNil (find (session "__tsql_parameter_declarations") (lambda (item) (equal? (car item) target)) nil)
				(error "undeclared FETCH variable: @" target)))))
		(if (and (not ((cursor "options") "scroll")) (not (equal? (car movement) "next")))
			(error "forward-only cursor supports only FETCH NEXT") true)
		(if (and (equal? ((cursor "options") "kind") "dynamic") (equal? (car movement) "absolute"))
			(error "dynamic cursor does not support FETCH ABSOLUTE") true)
		(if (and (cadr movement) (regexp_test (strtrim (cadr movement)) "^@")) (begin
			(define variable (toLower (substr (strtrim (cadr movement)) 1)))
			(define contract (find (session "__tsql_parameter_declarations") (lambda (entry) (equal? (car entry) variable)) nil))
			(if (and contract (has? '("TINYINT" "SMALLINT" "INT" "INTEGER") (car (tsql_effective_parameter_contract (cadr contract))))) true
				(error "FETCH offset variable requires a declared INT, SMALLINT or TINYINT"))) true)
		(define amount (if (cadr movement) (tsql_scalar_value (session "schema") (cadr movement) session tx "assignment" '("INT")) 0))
		(if (nil? amount) (error "FETCH offset cannot be NULL") true)
		(define position (tsql_cursor_fetch_position cursor (car movement) amount))
		(define position (if (cursor "dynamic") position (max 0 (min (+ (cursor "count") 1) position))))
		(define row (if (cursor "dynamic") (tsql_cursor_dynamic_fetch cursor session tx)
			(and (> position 0) (<= position (cursor "count")))
			(scan nil (cursor "table")
				(scan_access_schema (list (scan_boundary "equal" "__ordinal" 0 0 true true "" false)) '() nil false)
				(list position) '() (lambda () true) (cursor "columns")
				(lambda args (cdr args)) nil (lambda (a b) (coalesceNil a b)) false) nil))
		(define output_fields (if targets fields (tsql_cursor_result_fields fields)))
		(define formatter (if targets nil
			(if (cursor "result_formatter") (cursor "result_formatter") (begin
				(define formatter (tsql_cursor_result_formatter output_fields))
				(cursor "result_formatter" formatter) formatter))))
		(cursor "position" position)
		(if targets true (resultfields (map output_fields (lambda (field) (field "name"))) output_fields))
		(if (nil? row) (begin ((tsql_connection_session session) "__tsql_fetch_status" -1) 0) (begin
			(if (nil? targets) (begin
				((tsql_connection_session session) "__tsql_fetch_status" 0)
				(formatter session resultrow (append (merge (mapIndex fields (lambda (index field)
					(list (field "name") (nth row index))))) "ROWSTAT" 1)))
				(begin
					(define inputs (mapIndex fields (lambda (index field) (begin
						(define datatype (field "sql_datatype"))
						(list (concat "__cursor_value_" index)
							(if (list? datatype) (cons (car datatype) (cadr datatype)) (list datatype)))))))
					(define scoped (tsql_scope_session session inputs nil false))
					(mapIndex row (lambda (index value) (scoped (concat "__tsql_parameter:__cursor_value_" index) value)))
					(define values (mapIndex declarations (lambda (index declaration)
						(tsql_scalar_value (session "schema") (concat "@__cursor_value_" index) scoped tx "assignment" (cadr declaration)))))
					(reduce (produceN (count targets)) (lambda (_ index)
						(session (concat "__tsql_parameter:" (nth targets index)) (nth values index))) nil)
					((tsql_connection_session session) "__tsql_fetch_status" 0)
					true))
			1)))))))
(define tsql_cursor_close (lambda (name global deallocate statement session)
	(tsql_statement_call statement session (if deallocate "DEALLOCATE CURSOR" "CLOSE CURSOR") false (lambda (_tx) (begin
		(define cursor (tsql_cursor_lookup session name global))
		(if (or deallocate (cursor "open")) true (error "cursor is not open: " name))
		(if deallocate (match name
			'("variable" variable) (tsql_cursor_variable_set session variable nil)
			_ (tsql_cursor_remove session name (cursor "global")))
			(tsql_cursor_close_record cursor))
		0)))))

/* Status is a declared scalar invocation input. Normal SQL session binding
hoists it before delegated row work; cursor records never become SQL values.
https://learn.microsoft.com/sql/t-sql/functions/cursor-status-transact-sql */
(define tsql_cursor_status (close_procedure ((lambda (tsql_cursor_variable_entries tsql_cursor_entries)
	(lambda (session kind name) (begin
		(define name (toLower name))
		(define variable (equal? kind "variable"))
		(define entry (if variable
			(find (tsql_cursor_variable_entries session) (lambda (entry)
				(equal? (concat "@" (car entry)) name)) nil)
			(find (tsql_cursor_entries session (equal? kind "global"))
				(lambda (entry) (equal? (car entry) name)) nil)))
		(if (nil? entry) -3 (nil? (cadr entry)) -2 (begin
			(define cursor (cadr entry))
			(if (not (cursor "open")) -1
				(equal? ((cursor "options") "kind") "dynamic") 1
				(equal? (cursor "count") 0) 0 1)))))) tsql_cursor_variable_entries tsql_cursor_entries)))
(define tsql_cursor_status_compiler (lambda (infos) (begin
	(define kind (tryfold (sql_info_formula (car infos))))
	(define name (tryfold (sql_info_formula (cadr infos))))
	(if (and (string? kind) (string? name)) true
		(error "CURSOR_STATUS requires constant source and name strings"))
	(define kind (toLower kind))
	(if (has? '("local" "global" "variable") kind) true (error "invalid CURSOR_STATUS source"))
	(sql_info (list 'session (concat "__tsql_cursor_status:" (json_encode (list kind name)))) "SMALLINT" nil))))
/* CURSOR_STATUS already requires constant kind/name arguments. Resolve that
pure declaration before query planning so the invocation read is visible to
normal session/domain discovery and the scalar-input hoist. A late expression
callback must not introduce a new cursor binding after that boundary. */
(define tsql_cursor_status_expression (lambda (kind name) (begin
	(define info (tsql_cursor_status_compiler (map (list kind name)
		(lambda (expression) (sql_expr_info '() expression)))))
	(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) (sql_info_collation info)))))
/* This adapter can be captured by delegated parameter readers even when SQL
does not use a cursor. Publish its helper chain as lexical values and retain
the caller's actual connection and scope owners across every adapter. */
(define tsql_cursor_scalar_input_key? (close_procedure (lambda (key)
	(if (string? key) (or (equal? key "__tsql_cursor_rows") (strlike key "__tsql_cursor_status:%")) false))))
(define tsql_cursor_input_session (close_procedure ((lambda (tsql_cursor_status tsql_connection_session tsql_identity_scope cursor_rows scalar_input?)
	(lambda (parent)
		(lambda args (match args
			'(key) (if (equal? key "__tsql_connection_session") (tsql_connection_session parent)
				(equal? key "__tsql_scope") (tsql_identity_scope parent)
				(equal? key "__tsql_cursor_rows") (cursor_rows parent)
				(scalar_input? key)
				(begin
					(define input (json_decode (substr key (strlen "__tsql_cursor_status:"))))
					(tsql_cursor_status parent (car input) (cadr input))) (parent key))
			_ (apply parent args))))) tsql_cursor_status tsql_connection_session tsql_identity_scope tsql_cursor_rows tsql_cursor_scalar_input_key?)))
