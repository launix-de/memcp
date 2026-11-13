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
	(if (find options (lambda (option) (has? '("KEYSET" "SCROLL_LOCKS" "OPTIMISTIC" "TYPE_WARNING") option)) nil)
		(error "unsupported cursor option") true)
	(if (and (has? options "FAST_FORWARD") (has? options "SCROLL")) (error "FAST_FORWARD cannot SCROLL") true)
	/* CURSOR_DEFAULT is not configurable in this frontend yet. Its supported
	connection profile is GLOBAL; explicit LOCAL always has lexical lifetime. */
	(list "local" (has? options "LOCAL")
		"kind" (if (has? options "STATIC") "static" (has? options "FAST_FORWARD") "fast_forward" "dynamic")
		"scroll" (if (or (has? options "FORWARD_ONLY") (has? options "FAST_FORWARD")) false
			(or (has? options "SCROLL") (has? options "STATIC") (has? options "DYNAMIC")))))))
(define tsql_cursor_scope_id (lambda (session) (begin
	(define scope (tsql_identity_scope session))
	(coalesceNil (scope "__tsql_cursor_scope_id") (begin
		(define id (uuid)) (scope "__tsql_cursor_scope_id" id) id)))))
(define tsql_cursor_directory (lambda (session global)
	(if global (tsql_connection_session session) (tsql_identity_scope session))))
(define tsql_cursor_key (lambda (global) (if global "__tsql_global_cursors" "__tsql_local_cursors")))
(define tsql_cursor_entries (lambda (session global)
	(coalesceNil ((tsql_cursor_directory session global) (tsql_cursor_key global)) '())))
/* Names and variables are references to one cursor record. Variables live in
lexical control scope, outside enumerable SQL parameter values and plan inputs.
https://learn.microsoft.com/sql/t-sql/language-elements/deallocate-transact-sql */
(define tsql_cursor_variable_entries (lambda (session)
	(coalesceNil ((tsql_identity_scope session) "__tsql_cursor_variables") '())))
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
(define tsql_cursor_reference_syntax (parser (or
	(parser (define name tsql_variable_name) (list "variable" name))
	(parser (define name tsql_identifier) (toLower name)))
	"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))
(define tsql_cursor_assign_reference (lambda (name text statement session)
	(statement "SET CURSOR REFERENCE" false (lambda (_tx)
		(tsql_cursor_variable_set session name
			(tsql_cursor_lookup session (tsql_cursor_reference_syntax text) false))))))
(define tsql_cursor_close_record (lambda (cursor) (begin
	(define scope (cursor "native"))
	(cursor "closing" true)
	(if scope (close_relation_scope scope) true)
	(cursor "native" nil) (cursor "table" nil) (cursor "open" false) (cursor "closing" false)
	(cursor "fields" nil) (cursor "position" 0) (cursor "count" 0)
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
(define tsql_cursor_create_record (lambda (options text tx session) (begin
	(define planning (tsql_cursor_planning session))
	(define bindings (tsql_temporary_bindings session))
	(define syntax (parse_tsql (session "schema") text
		(tsql_temporary_policy (session "username") bindings) planning tx))
	(define query (match syntax ((symbol quote) query) query _ (error "cursor requires a SELECT result")))
	(define transition (tsql_cursor_transition_query? query session))
	(if (and (equal? (options "kind") "dynamic") (not transition))
		(error "dynamic cursors require an immutable direct transition source in this profile") true)
	(define cursor (newsession))
	(cursor "schema" (session "schema")) (cursor "text" text) (cursor "options" options)
	(cursor "transition" transition)
	(cursor "references" 0)
	(cursor "scoped_source" (tsql_temporary_query? query))
	(cursor "owner_scope" (tsql_cursor_scope_id session))
	(cursor "contracts" (filter (coalesceNil (session "__tsql_parameter_declarations") '())
		(lambda (declaration) (not (expression_equal? (cadr declaration) '("CURSOR"))))))
	(cursor "open" false)
	cursor)))
(define tsql_cursor_declare (lambda (name options text statement session)
	(statement "DECLARE CURSOR" false (lambda (tx) (begin
		(define global (not (options "local")))
		(if (find (tsql_cursor_entries session global) (lambda (entry) (equal? (car entry) name)) nil)
			(error "cursor already exists: " name) true)
		(define cursor (tsql_cursor_create_record options text tx session))
		(cursor "global" global) (cursor "name" name)
		((tsql_cursor_directory session global) (tsql_cursor_key global)
			(cons (list name (tsql_cursor_retain cursor)) (tsql_cursor_entries session global))) 0)))))
(define tsql_cursor_assign_query (lambda (name options text statement session)
	(statement "SET CURSOR" false (lambda (tx) (begin
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
	(statement "OPEN CURSOR" false (lambda (tx) (begin
		(define cursor (tsql_cursor_lookup session name global))
		(if (cursor "closing") (error "cursor retirement must finish before OPEN") true)
		(if (cursor "open") (error "cursor is already open: " name) true)
		(if (and (or (cursor "scoped_source") (not (empty_list? (cursor "contracts"))))
			(not (equal? (cursor "owner_scope") (tsql_cursor_scope_id session))))
			(error "cursor query binding is outside its declaring scope") true)
		(define fields (tsql_describe_statement tx (cursor "schema") (cursor "text") (cursor "contracts") session))
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
			0)))))))

(define tsql_cursor_fetch_position (lambda (cursor movement amount)
	(match movement
		"next" (+ (cursor "position") 1) "prior" (- (cursor "position") 1)
		"first" 1 "last" (cursor "count")
		"absolute" (if (< amount 0) (+ (cursor "count") amount 1) amount)
		"relative" (+ (cursor "position") amount)
		_ (error "unsupported FETCH direction"))))
(define tsql_cursor_fetch (lambda (name global movement targets statement session)
	(statement "FETCH CURSOR" false (lambda (tx) (begin
		(define cursor (tsql_cursor_lookup session name global))
		(tsql_require_database_access (tsql_session_actor session) (cursor "schema"))
		(if (cursor "closing") (error "cursor retirement must finish before FETCH") true)
		(if (cursor "open") true (error "cursor is not open: " name))
		(if (nil? targets) (error "FETCH without INTO is not supported") true)
		(define fields (cursor "fields"))
		(if (equal? (count targets) (count fields)) true (error "FETCH variable count differs from cursor projection"))
		(define declarations (map targets (lambda (target)
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
		(define position (max 0 (min (+ (cursor "count") 1) position)))
		(define row (if (and (> position 0) (<= position (cursor "count")))
			(scan nil (cursor "table")
				(scan_access_schema (list (scan_boundary "equal" "__ordinal" 0 0 true true "" false)) '() nil false)
				(list position) '() (lambda () true) (cursor "columns")
				(lambda args (cdr args)) nil (lambda (a b) (coalesceNil a b)) false) nil))
		(cursor "position" position)
		(if (nil? row) (begin ((tsql_connection_session session) "__tsql_fetch_status" -1) 0) (begin
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
			1)))))))
(define tsql_cursor_close (lambda (name global deallocate statement session)
	(statement (if deallocate "DEALLOCATE CURSOR" "CLOSE CURSOR") false (lambda (_tx) (begin
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
(define tsql_cursor_status (lambda (session kind name) (begin
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
			(equal? (cursor "count") 0) 0 1))))))
(define tsql_cursor_status_compiler (lambda (infos) (begin
	(define kind (tryfold (sql_info_formula (car infos))))
	(define name (tryfold (sql_info_formula (cadr infos))))
	(if (and (string? kind) (string? name)) true
		(error "CURSOR_STATUS requires constant source and name strings"))
	(define kind (toLower kind))
	(if (has? '("local" "global" "variable") kind) true (error "invalid CURSOR_STATUS source"))
	(sql_info (list 'session (concat "__tsql_cursor_status:" (json_encode (list kind name)))) "SMALLINT" nil))))
(define tsql_cursor_input_session (lambda (parent)
	(lambda args (match args
		'(key) (if (if (string? key) (strlike key "__tsql_cursor_status:%") false)
			(begin
				(define input (json_decode (substr key (strlen "__tsql_cursor_status:"))))
				(tsql_cursor_status parent (car input) (cadr input))) (parent key))
		_ (apply parent args)))))
