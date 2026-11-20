/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Compilation receipts contain syntax and declarations, never executable
plans or a failed native operation disguised as success. Deferred relation
binding is a distinct state and cannot become a physical input.
https://learn.microsoft.com/sql/t-sql/statements/set-noexec-transact-sql */
(define tsql_command_binding (lambda (kind value dependencies)
	(list 'tsql-command-binding kind (if (empty_list? dependencies) 'bound 'deferred) value dependencies)))
(define tsql_command_binding_dependencies (lambda (binding) (nth binding 4)))
(define tsql_compile_require_parameters (lambda (node) (match node
	((symbol quote) _literal) true
	((symbol tsql_unbound_parameter) _name spelling)
	(tsql_raise_error 137 (concat "Must declare the scalar variable @" spelling) 2)
	(cons head tail) (begin (tsql_compile_require_parameters head)
		(map tail tsql_compile_require_parameters) true)
	_ true)))
/* Resolve all relation identities before column binding. SQL Server defers
that entire statement phase when an object is absent, including columns of
other, existing inputs. Variable declarations are checked before this step. */
(define tsql_compile_relation_dependencies (lambda (node tx) (match node
	((symbol quote) _literal) '()
	((symbol tsql_view_scalar_call) database name arguments)
	(merge_unique (list
		(if (tsql_scalar_function_definition database name tx) '() (list (list 'scalar-function database name)))
		(tsql_compile_relation_dependencies arguments tx)))
	((symbol tsql_view_table_call) database name arguments _alias)
	(merge_unique (list
		(if (tsql_scalar_function_definition database name tx) '() (list (list 'table-function database name)))
		(tsql_compile_relation_dependencies arguments tx)))
	((symbol query-block) _database sources fields condition grouping having ordering limit offset hidden stages _facts)
	(merge_unique (list
		(merge (map sources (lambda (source) (begin
			(define relation (source_relation source))
			(if (and (string? relation)
				(not (equal?? (source_schema source) "information_schema"))
				(nil? (table (source_schema source) false relation false tx))
				(nil? (sql_find_view tx (source_schema source) relation)))
				(list (list 'relation (source_schema source) relation))
				(tsql_compile_relation_dependencies relation tx))))))
		(merge (map (list (source_join_exprs sources) fields condition grouping having ordering limit offset hidden stages)
			(lambda (part) (tsql_compile_relation_dependencies part tx))))))
	(cons head tail) (merge_unique (list (tsql_compile_relation_dependencies head tx)
		(merge (map tail (lambda (part) (tsql_compile_relation_dependencies part tx))))))
	_ '())))
/* The generic binder reports semantic name failures. SQL error numbering is
selected only here, independently of rendered text or native error carriers. */
(define tsql_compile_with_binding_diagnostics (lambda (body)
	(try body (lambda (failure) (match failure
		'("sql-binding-error" kind message)
		(tsql_raise_error (match kind "missing-column" 207 "ambiguous-column" 209 _ 4104) message 1)
		_ (error failure))))))
(define tsql_compile_query_binding (lambda (query policy planning tx) (begin
	(tsql_compile_require_parameters query)
	(define dependencies (tsql_compile_relation_dependencies query tx))
	(if (not (empty_list? dependencies)) (tsql_command_binding 'query query dependencies)
		(begin
			/* View expansion is the ordinary logical definition owner. It creates
			no storage relation. Recheck its sources before invoking column/type
			binding; a dropped dependency remains a deferred object. */
			(define expanded (sql_expand_views (tsql_compile_function_references query policy planning tx) policy planning tx))
			(define dependencies (tsql_compile_relation_dependencies expanded tx))
			(if (not (empty_list? dependencies)) (tsql_command_binding 'query expanded dependencies)
				(tsql_compile_with_binding_diagnostics (lambda () (begin
					(define bound (tsql_describe_query expanded policy planning tx))
					/* Annotation-free projections still need declared assignment
					contracts. The shared output owner publishes their descriptors;
					type-annotation lowering alone may legitimately leave them alone. */
					(define typed (sql_type_lower_annotations '() (sql_annotate_query_outputs bound false)))
					(tsql_command_binding 'query typed '()))))))))))
(define tsql_compile_defer (lambda (kind syntax nodes tx) (begin
	(tsql_compile_require_parameters syntax)
	(define dependencies (tsql_compile_relation_dependencies nodes tx))
	(if (empty_list? dependencies) nil (tsql_command_binding kind syntax dependencies)))))
(define tsql_compile_projection_info (lambda (query index) (match query
	((symbol query-block) _db _sources _fields _condition _group _having _order _limit _offset _hidden _stages facts)
	(cadr (nth (extract_assoc (qassoc_get facts 'result-types '()) (lambda (name info) (list name info))) index))
	((symbol union-block) _mode branches _order _limit _offset _facts)
	(tsql_compile_projection_info (car branches) index)
	_ (error "compiled projection requires a logical relation"))))

/* A compile receipt binds module headers, not executable procedure snapshots.
The local relation is a lexical declaration carrier only. A receipt has no
physical lowering path and cannot allocate or scan that relation. */
(define tsql_compile_function_references (lambda (node policy planning tx) (match node
	((symbol quote) _literal) node
	((symbol tsql_view_xml_nodes) database document path alias column)
	(tsql_xml_nodes_query database (tsql_compile_function_references document policy planning tx) path alias column)
	((symbol tsql_view_scalar_call) database name arguments)
	(tsql_scalar_function_declaration_call database name
		(map arguments (lambda (argument) (tsql_compile_function_references argument policy planning tx)))
		policy planning tx)
	((symbol tsql_view_table_call) database name arguments alias) (begin
		(define definition (tsql_scalar_function_definition database name tx))
		(if (tsql_table_function_language? (definition "language")) true (error "object is not a table-valued function: " name))
		(define signature (definition "parameters"))
		(define parameters (car signature))
		(if (equal? (count parameters) (count arguments)) true (error "wrong number of table function arguments"))
		(define resolved (tsql_scalar_function_resolve_arguments arguments parameters))
		/* Argument expressions still belong to this query scope. The typed
		contract is represented in hidden projections, so the ordinary binder
		checks every column without evaluating an invocation or its defaults. */
		(define hidden (merge (mapIndex resolved (lambda (index argument)
			(list (concat "__compile_argument_" index)
				(list 'sql_compile_expr
					(list 'tsql_compile_function_argument (list 'quote (cadr (nth parameters index))))
					(tsql_compile_function_references argument policy planning tx)))))))
		(define columns (nth signature 3))
		(define source (list alias database
			(make_local_relation_reference (symbol (concat "__compile_function_" name)) columns)
			false nil (list "columns" columns "constraints" '("Unique" () "ForeignKeys" ()))))
		(make_query_block database (list source)
			(merge (map columns (lambda (column)
				(list (column "Field") (list 'get_column alias false (column "Field") false)))))
			true nil nil nil nil nil hidden nil nil))
	(cons head tail) (cons (tsql_compile_function_references head policy planning tx)
		(map tail (lambda (child) (tsql_compile_function_references child policy planning tx))))
	_ node)))
(define tsql_compile_function_argument (lambda (declaration) (lambda (infos)
	(tsql_scalar_function_convert (car infos) (tsql_scalar_function_declaration declaration)))))

/* CREATE/ALTER declarations bind types and computed expressions before any
native catalog publication. The lexical input below has only declarations;
there is no table handle or row producer to execute. */
(define tsql_compile_table_columns (lambda (database name definitions policy planning tx) (begin
	(define columns (source_columns (tsql_computed_source database definitions)))
	(define source (list "__computed" database
		(make_local_relation_reference (symbol (concat "__compile_table_" name)) columns)
		false nil (list "columns" columns "constraints" '("Unique" () "ForeignKeys" ()))))
	(define bindings (map definitions (lambda (definition) (match definition
		((symbol list) "computed" title expression _persisted)
		(tsql_compile_query_binding (make_query_block database (list source)
			(list title expression) true nil nil nil nil nil nil nil nil) policy planning tx)
		_ (tsql_command_binding 'column definition '())))))
	(tsql_command_binding 'table bindings
		(merge_unique (map bindings tsql_command_binding_dependencies))))))
/* Structured commands retain every child's unresolved-name state. */
(define tsql_compile_composite (lambda (kind bindings)
	(tsql_command_binding kind bindings (merge_unique (map bindings tsql_command_binding_dependencies)))))

/* DEFAULT execution, generation and NOT NULL checks are not compilation.
Retain its actual declaration as a symbolic typed reference in the receipt;
never evaluate a generator, source expression or fabricated SQL NULL result. */
(define tsql_compile_default_declaration (lambda (declaration) (begin
	(define type (toUpper (declaration "RawType")))
	(define dimensions (coalesceNil (declaration "Dimensions") '()))
	(define collation (declaration "Collation"))
	(list 'sql_type_expr (list 'sql_default (list 'quote declaration))
		(if (empty_list? dimensions) type (list type dimensions))
		(if (and (sql_text_type? type) (string? collation) (not (equal? collation "")))
			(list collation 2) nil)))))

/* Schema-bound declarations do not inherit ordinary statement DNR. A view
must acquire every source export before its declaration can be accepted.
SQL Server 2022: https://dbfiddle.uk/Bb60YqUT */
(define tsql_compile_view_binding (lambda (query policy planning tx) (begin
	(define binding (tsql_compile_query_binding query policy planning tx))
	(define dependencies (tsql_command_binding_dependencies binding))
	(if (empty_list? dependencies) binding (begin
		(define dependency (car dependencies))
		(tsql_raise_error (if (equal? (car dependency) 'scalar-function) 4121 208)
			(concat "Invalid object name: " (nth dependency 2)) 1))))))
