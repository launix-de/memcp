/* Copyright (C) 2026 Carl-Philip Haensch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Relational iteration is a scoped group-stage. Its two UNION children are
seed and transition programs, not an ordinary independent bag union. Both are
logical query blocks, bound and decorrelated before any execution code exists.
Only the physical owner emits FOR over a bounded immutable state value. */
(define relational_iteration_parameter (close_procedure (lambda (value) value)))
/* The logical descriptor names this known retaining reducer. Its executable
callable is bound only by emitted code, never used as logical metadata identity. */
(define relational_iteration_last_reduce (close_procedure
	(lambda (previous current) (if (nil? current) previous current))))
(define relational_iteration_stage? (lambda (stage)
	(and (group_stage? stage) (equal? (qassoc_get (gs_facts stage) 'purpose nil) 'relational_iteration))))
(define relational_iteration_input (lambda (stage index)
	(nth (union_branches (gs_input stage)) index)))
/* Invocation reads are explicit algebra operands. In particular a transition
must not retain its first state or clock value inside a maintained group. */
(define relational_iteration_inputs (lambda (node state_symbol) (match node
	((symbol quote) _datum) node
	((symbol now)) (list relational_iteration_parameter node)
	(cons head tail) (cons (relational_iteration_inputs head state_symbol)
		(map tail (lambda (item) (relational_iteration_inputs item state_symbol))))
	_ (if (expression_equal? node state_symbol) (list relational_iteration_parameter node) node))))
(define make_relational_iteration (lambda (seed_query step_query state_symbol outer_sources predicate) (begin
	(if (and (query_block? seed_query) (query_block? step_query) (symbol? state_symbol)) true
		(neumann_fail "relational iteration" "seed and transition must be query blocks with a lexical state symbol"))
	(map (list seed_query step_query) (lambda (query)
		(if (expression_equal? (binding_field_titles (qb_fields query)) '("__fn_state")) true
			(neumann_fail "relational iteration" "program must export exactly one state tuple"))))
	(if (or (query_contains_subquery? predicate) (not (empty_list? (query_expr_session_reads predicate))))
		(neumann_fail "relational iteration" "predicate requires explicit scalar state inputs") true)
	(define seed (relational_iteration_inputs seed_query state_symbol))
	(define step (relational_iteration_inputs step_query state_symbol))
	(define identity (stable_structural_hash (list seed step state_symbol predicate) false))
	(define id (concat "relational-iteration:" identity))
	(define alias (concat "__iteration_" identity))
	(define descriptor (list state_symbol 'relational_iteration_last_reduce nil))
	(define column (aggregate_col_name descriptor))
	(define input (make_union_block 'all (list seed step) '() nil nil
		(list (list 'relational_iteration true))))
	/* Factories can introduce this operator while an enclosing subquery is
	being decorrelated. Publish its real free inputs immediately; the normal
	binder subsequently validates them against the complete lexical scopes. */
	(define domain (merge_unique (list (relational_iteration_outer_refs input outer_sources)
		(query_expr_session_reads seed))))
	(define stage (make_group_stage id input domain domain (list descriptor) nil '() '() nil nil
		(list (list 'purpose 'relational_iteration) (list 'state_symbol state_symbol)
			(list 'fresh_invocation (query_requires_fresh_invocation? input nil))
			(list 'iteration_predicate predicate) (list 'lookup-keys domain)
			(list 'null_semantics 'iteration) (list 'result_max_rows_per_partition 1))))
	(define declarations (list "columns" (list (list "Field" column "Type" "any" "RawType" "any"
		"Dimensions" '() "Null" false "Collation" "")) "constraints" '("Unique" () "ForeignKeys" ())))
	(make_query_block (qb_schema seed_query)
		(list (list alias (qb_schema seed_query) (make_stage_output_relation id) false true declarations))
		(list "__fn_state" (list 'get_column alias false column false)) true nil nil nil nil nil nil (list stage) '()))))

/* Lookup domains are genuine free SQL columns plus explicit scalar invocation
reads. A lexical child source shadows its parent; no alias-prefix heuristics. */
(define relational_iteration_outer_refs (lambda (node sources) (match node
	((symbol quote) _datum) '()
	((symbol query-block) _database locals _fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(begin
		(define visible (filter sources (lambda (source)
			(not (reduce (source_aliases locals) (lambda (shadowed local)
				(or shadowed (equal?? local (source_alias source)))) false)))))
		(relational_iteration_outer_refs (cdr node) visible))
	((symbol get_column) _alias _ic _column _cic)
	(btw2025_expr_outer_column_refs node '() sources)
	(cons head tail) (merge_unique (cons (relational_iteration_outer_refs head sources)
		(map tail (lambda (item) (relational_iteration_outer_refs item sources)))))
	_ '())))
(define bind_relational_iteration_stage (lambda (stage scopes path all_strings) (begin
	(define input (bind_query_names_at (gs_input stage) scopes path all_strings))
	(define sources (map (merge scopes) binding_entry_source))
	(define domain (merge_unique (list (relational_iteration_outer_refs input sources)
		(query_expr_session_reads (relational_iteration_input stage 0)))))
	(make_group_stage (gs_id stage) input domain domain (gs_aggregates stage) nil '() '() nil nil
		(qassoc_set (gs_facts stage) 'lookup-keys domain)))))
(define untangle_relational_iteration_stage (lambda (stage ctx) (begin
	(define local_context (make_uctx ctx (list (list 'defer-subquery-rewrites false))))
	(define branches (map (union_branches (gs_input stage)) (lambda (branch)
		(untangle_query branch local_context))))
	(make_group_stage (gs_id stage)
		(make_union_block 'all branches '() nil nil (union_facts (gs_input stage)))
		(gs_domain stage) (gs_keys stage) (gs_aggregates stage) nil '() '() nil nil (gs_facts stage)))))

/* An unconditional FROM-less projection is exactly one row. Its executable
expression is already bound; relational markers and residual column/domain
reads still belong to the ordinary relational consumer. */
(define relational_iteration_state_projection? (lambda (block)
	(and (query_block? block)
		(expression_equal? (binding_field_titles (qb_fields block)) '("__fn_state"))
		(empty_list? (qb_hidden block)) (empty_list? (qb_stages block))
		(empty_list? (qb_group block)) (empty_list? (qb_order block))
		(literal_true? (coalesceNil (qb_where block) true))
		(literal_true? (coalesceNil (qb_having block) true))
		(nil? (qb_limit block)) (nil? (qb_offset block))
		(not (query_contains_subquery? block)) (not (query_block_has_local_aggregates? block))
		(not (expr_contains_window? (qb_fields block)))
		(empty_list? (expr_probe_stages (qb_fields block))))))
(define relational_iteration_single_state_projection? (lambda (block)
	(and (relational_iteration_state_projection? block) (empty_list? (qb_sources block))
		(not (expr_contains_column_ref? (qb_fields block))))))
/* One syntactic parameter row retains its NULL cells and invocation operands.
All cell expressions are evaluated by the ordinary row-domain owner, before
the projection reads any bound column. No outer SQL read may escape binding. */
(define relational_iteration_parameter_state_projection? (lambda (block)
	(and (relational_iteration_state_projection? block) (single_source? (qb_sources block))
		(begin
			(define src (car (qb_sources block)))
			(define relation (source_relation src))
			(and (parameter_rows_relation? relation) (single_source? (row_domain_data relation))
				(not (source_outer? src)) (literal_true? (coalesceNil (source_join_expr src) true))
				(begin
					(define row (car (row_domain_data relation)))
					(define columns (row_domain_columns relation))
					(and (expression_equal? (binding_field_titles row) columns)
						(not (expr_contains_column_ref? row)) (not (expr_contains_subquery? row))
						(not (expr_has_local_aggregates? row)) (not (expr_contains_window? row))
						(empty_list? (expr_probe_stages row))
						(reduce (extract_assoc (query_expr_alias_set (source_alias src) (qb_fields block) '())
							(lambda (alias _present) alias))
							(lambda (local alias) (and local (equal? alias (source_alias src)))) true)
						(reduce (extract_columns_for_alias src (qb_fields block))
							(lambda (local column) (and local (contains? columns column))) true))))))))
(define lower_relational_iteration_tuple (lambda (value)
	(list (list 'lambda (list '__iteration_packet)
		(list 'if (list 'list? '__iteration_packet) '__iteration_packet
			(list 'error "relational iteration state must be a tuple"))) value)))
(define lower_relational_iteration_parameter_state (lambda (block) (begin
	(define src (car (qb_sources block)))
	(define columns (row_domain_columns (source_relation src)))
	(define row '__iteration_parameter_row)
	(define value (lower_column_expr_for_join (list src) (source_alias src)
		(get_assoc (qb_fields block) "__fn_state")))
	(define project (cons (list 'lambda (map columns (lambda (column)
		(scan_callback_symbol_for_alias (source_alias src) column)))
		(lower_relational_iteration_tuple value))
		(map columns (lambda (column) (list 'get_assoc row column)))))
	(list (list 'lambda (list row) project) (list 'car (source_table_expr src))))))

/* One row is a semantic contract of each program. The existing cluster session
owner transports this bounded result cell; no relation is accumulated in SCM.
The scoped producer admits one writer atomically even if workers finish together.
Proven singleton projections need neither a result cell nor a writer token. */
(define lower_relational_iteration_program (lambda (block)
	(if (relational_iteration_single_state_projection? block)
		(lower_relational_iteration_tuple
			(lower_scalar_marker_expr (get_assoc (qb_fields block) "__fn_state")))
		(relational_iteration_parameter_state_projection? block)
		(lower_relational_iteration_parameter_state block)
		(begin
			(define code (lower_query_block_with_stages block))
			(define collector '__iteration_result)
			(define token '__iteration_row_token)
			(define winner '__iteration_row_winner)
			(define produce (list 'lambda '()
				(list 'list token (list 'get_assoc 'row "__fn_state"))))
			(define receive (list 'lambda (list 'row)
				(list 'begin (list 'define token (list 'uuid))
					(list 'define winner (list collector "get_or_compute_scoped" 0 "row" produce))
					(list 'if (list 'equal? (list 'car winner) token) nil
						(list 'error "relational iteration program produced multiple states")))))
			(define read (list collector "get_or_compute_scoped" 0 "row"
				(list 'lambda '() (list 'error "relational iteration program produced no state"))))
			(define execute (list (list 'lambda (list 'resultrow) code) receive))
			(list (list 'lambda (list collector)
				(list 'begin execute
					(lower_relational_iteration_tuple (list 'cadr read))))
				(list 'newsession))))))
(define lower_relational_iteration_probe (lambda (sources default_alias stage tx_expr) (begin
	(define lookup_keys (qassoc_get (gs_facts stage) 'lookup-keys '()))
	(define parameters (mapIndex lookup_keys (lambda (i _key) (symbol (concat "__iteration_domain_" i)))))
	(define bindings (scalar_query_probe_param_index (gs_keys stage) lookup_keys parameters))
	(define seed_query (rewrite_scalar_query_probe_params bindings (relational_iteration_input stage 0)))
	(define seed (lower_relational_iteration_program seed_query))
	(define state_symbol (qassoc_get (gs_facts stage) 'state_symbol nil))
	(define predicate (qassoc_get (gs_facts stage) 'iteration_predicate nil))
	(define step_query (rewrite_scalar_query_probe_params bindings (relational_iteration_input stage 1)))
	(define step (bind_invocation_scalar_reads (query_expr_session_reads step_query)
		(lower_relational_iteration_program step_query) "__iteration_binding_"))
	/* Seed dependencies bind in the consuming row. FOR owns a scalar tuple;
	its transition is already compiled native relational code, never SQL text. */
	(define program (bind_invocation_scalar_reads (query_expr_session_reads seed_query)
		(list 'car (list 'for (list 'list seed)
			(list 'lambda (list state_symbol) (list 'begin (list 'tx_check tx_expr) predicate))
			(list 'lambda (list state_symbol) (list 'list step)))) "__iteration_seed_binding_"))
	(cons (list 'lambda parameters program)
		(map lookup_keys (lambda (key) (lower_column_expr_for_join sources default_alias key)))))))
