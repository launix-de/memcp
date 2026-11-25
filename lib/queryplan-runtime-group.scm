/* Copyright (C) 2026 Carl-Philip Haensch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Volatile domains cannot own maintained caches. Their group and DISTINCT
keys live in ordinary native scoped MEMORY relations for one execution. Only
bounded row/descriptor tuples cross the Scheme callback boundary. */
/* Generated relation inputs use the same scoped MEMORY owner and native
indexes as other execution-local carriers. The source producer has already
computed their immutable positional cells with its actual transaction; no SQL
datatype is inferred here. Filling the source borrows the actual caller's
native mutation owner, so a failed consuming statement restores its private
rows together with its public effects. The native resource retains the scope
until that caller's decision; a nil-transaction invocation closes it directly. */
(define runtime_generated_source (lambda (database names collations rows session tx scopes) (begin
	(tx_check tx)
	(if (equal? (count names) (count collations)) true
		(error "generated source names and collations must have the same length"))
	(define name (concat "." (uuid) "#generated-source"))
	(define scope (if (nil? tx) (new_relation_scope session)
		(tx_get_or_compute_resource tx (concat "generated-source/" name)
			(lambda () (new_relation_scope session)))))
	(try (lambda () (begin
		(createtable database name
			(mapIndex names (lambda (index column)
				(list "column" column "any" '()
					(if (string? (nth collations index))
						(list "collate" (nth collations index)) '()))))
			(list "engine" "memory" "ephemeral" true "scope" scope) false tx)
		(define source (table database false name false tx))
		(if (empty_list? rows) true
			(with_table_mutation (list source) (list "insert" names) tx session
				(lambda (sink) (insert source names rows '() (lambda () true) false nil sink))))
		(tx_check tx)
		(if (nil? tx) (scopes name scope) true)
		source))
		(lambda (failure) (begin
			(if (nil? tx) (close_relation_scope scope) true)
			(error failure)))))))

/* Native any-valued columns retain the generic runtime's coercing equality.
A typed null coordinate makes SQL group identity injective for NULL versus
zero/false/empty values without changing that native equality contract. */
(define runtime_group_null_column (lambda (name) (concat "__group_null:" name)))
/* Every caller populates a fresh native scope-owned GROUP/DISTINCT or UNION
carrier. Source/key/operand expressions retain their actual query bindings;
only the private scratch row mutation is outside the user transaction Undo. */
(define runtime_group_insert_expr (lambda (target names values collision_columns collision null_keys)
	(begin
		(define parameters (mapIndex names (lambda (index _name)
			(symbol (concat "__group_cell_" index)))))
		(define row_values (if (empty_list? null_keys) values parameters))
		(define null_values (map null_keys (lambda (key)
			(begin
				(define position (find (produceN (count names)) (lambda (index)
					(equal? key (nth names index))) nil))
				(if (nil? position) (neumann_fail "build_queryplan" "native group null key is outside its row declaration")
					(list (quote if) (list (quote nil?) (nth parameters position)) 1 0))))))
		(define insert (list 'insert target
			(quoted_runtime_list (merge (list names (map null_keys runtime_group_null_column))))
			(list 'list (runtime_cons_list_expr (merge (list row_values null_values))))
			(quoted_runtime_list collision_columns) collision true nil nil))
		(if (empty_list? null_keys) insert (cons (list 'lambda parameters insert) values)))))

(define runtime_group_value_collation (lambda (src collations expr) (begin
	(define alias (if (union_block? src) (qassoc_get (union_facts src) 'alias "__union") (source_alias src)))
	(define facts (filter collations (lambda (entry)
		(equal? (canonical_column_expr_for_alias alias (car entry)) expr))))
	(if (empty_list? facts) (physical_column_collation_expr src expr) (begin
		(define collation (cadr (car facts)))
		(if (reduce facts (lambda (same entry) (and same (equal? (cadr entry) collation))) true)
			collation (neumann_fail "build_queryplan" "runtime grouping cannot combine conflicting collations of one normalized operand")))))))

(define runtime_group_create_expr (lambda (database name names expressions unique_names src collations scope)
	(list 'begin
		(list 'createtable database name
			(cons 'list (cons (list 'list "unique" "group"
				(quoted_runtime_list (merge (list unique_names (map unique_names runtime_group_null_column)))))
				(merge (list
					(mapIndex names (lambda (i column)
						(list 'list "column" column "any" (quoted_runtime_list '())
							(list 'list "collate" (runtime_group_value_collation src collations (nth expressions i))))))
					(map unique_names (lambda (column)
						(list 'list "column" (runtime_group_null_column column) "INT" (quoted_runtime_list '())
							(quoted_runtime_list '()))))))))
			(list 'list "engine" "memory" "ephemeral" true "scope" scope) false (physical_query_tx_symbol))
		(list 'table database false name false (physical_query_tx_symbol)))))

/* Declared scalar reads are immutable invocation inputs. Bind them before
creating row callbacks so delegation captures values rather than a caller's
session forwarding closure. Other calls remain at their original row boundary. */
(define bind_invocation_scalar_reads (lambda (reads formula prefix) (begin
	(define names (mapIndex reads (lambda (i _read) (symbol (concat prefix i)))))
	(define rewrite (lambda (expr) (match expr
		((symbol quote) _datum) expr
		(cons head tail) (begin
			(define read (query_session_read_expr expr))
			(define index (if read (find (produceN (count reads))
				(lambda (i) (expression_equal? read (nth reads i))) nil) nil))
			(if (nil? index) (cons (rewrite head) (map tail rewrite)) (nth names index)))
		_ expr)))
	(if (empty_list? reads) formula
		(cons (list 'lambda names (rewrite formula)) reads)))))

(define runtime_group_bind_domain (lambda (stage formula)
	(bind_invocation_scalar_reads (group_stage_session_domain_keys stage)
		formula "__runtime_group_binding_")))

(define runtime_distinct_aggregate? (lambda (ag)
	(or (count_distinct_descriptor? ag) (distinct_aggregate_descriptor? ag))))
(define runtime_distinct_neutral (lambda (ag)
	(if (count_distinct_descriptor? ag) 0 (nth ag 2))))

(define runtime_group_reducers_supported? (lambda (aggregates)
	(reduce aggregates (lambda (valid ag)
		(and valid (or (count_distinct_descriptor? ag) (native_distinct_aggregate? ag) (bounded_scalar_aggregate? ag)
			(equal? (cadr ag) (scalar_once_reduce_first))))) true)))
/* Ordered FIRST operands are compiler markers with a separate ordered-probe
owner. They are not executable row values for the ordinary native carrier.
Only the actual chosen materializer asks for this bounded reducer contract. */
(define runtime_group_materializer_supported? (lambda (stage)
	(and (runtime_group_reducers_supported? (gs_aggregates stage))
		(reduce (gs_aggregates stage) (lambda (valid aggregate)
			(and valid (nil? (scalar_order_aggregate_parts aggregate)))) true))))

(define lower_runtime_group_relation (lambda (catalog database stage consume retain)
	(if (and (lowering_catalog_caller_input? catalog) (not (runtime_group_materializer_supported? stage)))
		(begin
			(define reason "native grouping requires executable bounded scalar or distinct value reducers")
			(define contracts (lowering_catalog_input_contracts catalog))
			(if (nil? contracts) (neumann_fail "build_queryplan" reason)
				(begin (contracts "__memcp_queryplan_input_refusal" reason) (list 'error reason))))
		(begin
			(define src (gs_input stage))
			(define union_input (union_block? src))
			(define query_input (query_block? src))
			(define relation_input (or union_input query_input))
			(define alias (group_stage_input_alias stage))
			(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
			(define key_names (group_key_cols keys))
			(define ags (gs_aggregates stage))
			(define collations (qassoc_get (gs_facts stage) 'group-value-collations '()))
			(if (runtime_group_reducers_supported? ags)
				true (neumann_fail "build_queryplan" "native grouping requires bounded scalar or distinct value reducers"))
			(define value_names (map ags (lambda (ag) (group_stage_aggregate_column stage ag))))
			(define group_names (merge (list key_names value_names)))
			(define distinct_indices (filter (produceN (count ags)) (lambda (i) (runtime_distinct_aggregate? (nth ags i)))))
			(define group_table '__runtime_group_table)
			(define scope '__runtime_group_scope)
			(define name '__runtime_group_name)
			(define distinct_vars (map distinct_indices (lambda (i) (symbol (concat "__runtime_distinct_" i)))))
			(define key_params (map key_names (lambda (column) (symbol (concat "__runtime_key_" column)))))
			(define value_params (mapIndex ags (lambda (i _ag) (symbol (concat "__runtime_value_" i)))))
			(define key_values (if relation_input key_params (map keys (lambda (expr) (lower_column_expr_for_alias src expr)))))
			(define condition (coalesceNil (qassoc_get (gs_facts stage) 'condition true) true))
			(if (and union_input (not (expression_equal? condition true)))
				(neumann_fail "build_queryplan" "volatile UNION grouping requires its input condition in the UNION branches") true)
			(define input_columns (if relation_input '() (merge_unique (list (extract_columns_for_alias src condition)
				(merge (map keys (lambda (expr) (extract_columns_for_alias src expr))))
				(merge (map ags (lambda (ag) (extract_columns_for_alias src (car ag)))))))))
			(define input_params (map input_columns (lambda (col) (scan_callback_symbol_for_alias alias col))))
			(define row_values (mapIndex ags (lambda (i ag)
				(if (runtime_distinct_aggregate? ag) (runtime_distinct_neutral ag)
					(and union_input (equal? ag aggregate_count_descriptor)) 1
					(if relation_input (nth value_params i) (lower_column_expr_for_alias src (car ag)))))))
			(define old_params (mapIndex ags (lambda (i _ag) (symbol (concat "__old_" i)))))
			(define new_params (mapIndex ags (lambda (i _ag) (symbol (concat "__new_" i)))))
			(define collision (list 'lambda (cons '$update (merge (list old_params new_params)))
				(list '$update (runtime_cons_list_expr (merge (mapIndex ags (lambda (i ag)
					(list (nth value_names i) (if (runtime_distinct_aggregate? ag) (nth old_params i)
						(list (cadr ag) (nth old_params i) (nth new_params i)))))))))))
			(define insert_group (runtime_group_insert_expr group_table group_names
				(merge (list key_values row_values))
				(merge (list '("$update") value_names (map value_names (lambda (col) (concat "NEW." col))))) collision key_names))
			(define fill_distinct (mapIndex distinct_indices (lambda (j i) (begin
				(define value (if relation_input (nth value_params i) (lower_column_expr_for_alias src (car (nth ags i)))))
				(list 'if (list 'nil? value) nil
					(runtime_group_insert_expr (nth distinct_vars j) (merge (list key_names '("value")))
						(merge (list key_values (list value))) '() (list 'lambda '() nil)
						(merge (list key_names '("value")))))))))
			(define fill_row (cons 'begin (merge (list (list insert_group) fill_distinct (list nil)))))
			/* Each branch emits a bounded key/operand tuple straight into the native
			scope-owned relations. Scalar nil reduction carries no accumulated row bag. */
			(define fill (if union_input
				(lower_union_block_as_dataset_reduce src keys key_names ags
					(mapIndex ags (lambda (i _ag) (concat "__runtime_operand_" i)))
					(list 'lambda (merge (list key_params value_params)) fill_row)
					(list 'lambda (list 'acc 'row) nil) nil (list 'lambda (list 'left 'right) nil))
				query_input (begin
					(define input_fields (merge (list
						(merge (mapIndex keys (lambda (i expression) (list (nth key_names i) expression))))
						(merge (mapIndex ags (lambda (i ag) (list (concat "__runtime_operand_" i) (car ag))))))))
					(define catalog (invocation_group_catalog catalog stage))
					/* A source may already name its invocation carrier rather than its
					logical stage-output. Recover its producer from the same catalog before
					consumption, unless the enclosing preparation already owns that packet. */
					(define input_stages (unique_stages_by_id (merge (list (qb_stages src)
						(stage_outputs_from_sources_using catalog (qb_sources src))
						(filter (group_cache_stages_from_sources catalog (qb_sources src))
							(lambda (dependency) (and (invocation_group_stage? catalog dependency)
								(not (qassoc_get (gs_facts dependency) 'eager_prepared false)))))))))
					(define input (make_query_block (qb_schema src) (qb_sources src) input_fields
						(combine_where (qb_where src) condition) (qb_group src) (qb_having src)
						(qb_order src) (qb_limit src) (qb_offset src) (qb_hidden src) input_stages
						(query_block_facts_with_lowering_catalog src catalog
							(lowering_catalog_with_planning_session catalog
								(planner_context_session (qb_facts src))))))
					(define prepared (prepare_simple_query_block_physical_core input))
					(cons '!begin (merge (list (nth prepared 0)
						(list (if (query_group_cardinality_domain? keys ags)
							/* One invocation has one constant key tuple. Keep the complete
							chosen input, reduce its cardinality as a scalar, then insert the
							completed state once instead of updating a native row per input. */
							(list 'reduce_assoc
								(build_query_grouped_assoc_plan (nth prepared 1) keys key_names ags false)
								(list 'lambda (list '__count_acc '__count_keys '__count_values)
									(list 'apply (list 'lambda (merge (list key_params value_params)) fill_row)
										(list 'merge (list 'list '__count_keys '__count_values)))) nil)
							(lower_query_block_as_dataset_reduce (nth prepared 1) (qb_fields (nth prepared 1))
								(list 'lambda (merge (list key_params value_params)) fill_row)
								(list 'lambda (list 'acc 'row) nil) nil (list 'lambda (list 'left 'right) nil))))
						(nth prepared 2)))))
				(compile_scan_plan 'scan (physical_query_tx_symbol) '__runtime_group_input
					(quoted_runtime_list input_columns) (list 'lambda input_params (lower_column_expr_for_alias src condition))
					(quoted_runtime_list input_columns) (list 'lambda (cons '__acc input_params)
						fill_row) nil nil false)))
			/* Native keys deduplicate complete (group,value) tuples. Fold each surviving
			value once through the original bounded reducer; ordinary bag aggregates
			have already consumed their full input and are not touched by this pass. */
			(define fold_distinct (mapIndex distinct_indices (lambda (j i) (begin
				(define ag (nth ags i))
				(define cardinality (count_distinct_descriptor? ag))
				(define reducer (if cardinality '+ (cadr (distinct_aggregate_underlying ag))))
				(define contribution (if cardinality 1 '__distinct_value))
				(define initial (list reducer (runtime_distinct_neutral ag) contribution))
				(define params (map key_names (lambda (col) (symbol (concat "__distinct_" col)))))
				(define payload (mapIndex ags (lambda (k other)
					(if (equal? i k) initial (runtime_distinct_neutral other)))))
				(define update (list 'lambda (list '$update 'old)
					(list '$update (list 'list (nth value_names i) (list reducer 'old contribution)))))
				(compile_scan_plan 'scan (physical_query_tx_symbol) (nth distinct_vars j)
					(quoted_runtime_list '()) (list 'lambda '() true)
					(quoted_runtime_list (if cardinality key_names (append key_names "value")))
					(list 'lambda (cons '__acc (if cardinality params (append params '__distinct_value)))
						(list 'begin (runtime_group_insert_expr group_table group_names (merge (list params payload))
							(list "$update" (nth value_names i)) update key_names) nil)) nil nil false)))))
			(define emit (consume group_table scope))
			(define initialize (if (and (qassoc_get (gs_facts stage) 'preserve_empty_domain false)
				(not (reduce keys (lambda (columns key) (or columns (expr_contains_column_ref? key))) false)))
				(list (runtime_group_insert_expr group_table group_names
					(merge (list (if relation_input keys key_values) (map ags runtime_distinct_neutral)))
					'() (list 'lambda '() nil) key_names)) '()))
			(define execution (cons 'begin (merge (list
				(list (list 'define group_table (runtime_group_create_expr database name group_names
					(merge (list keys (map ags car))) key_names src collations scope)))
				(mapIndex distinct_indices (lambda (j i)
					(list 'define (nth distinct_vars j)
						(runtime_group_create_expr database (list 'concat name ":distinct:" i)
							(merge (list key_names '("value"))) (merge (list keys (list (car (nth ags i)))))
							(merge (list key_names '("value"))) src collations scope))))
				initialize (list fill) fold_distinct (list emit)))))
			(runtime_group_bind_domain stage
				(list 'begin (if relation_input nil (list 'define '__runtime_group_input (source_scan_table_expr src condition)))
					(list 'define name (list 'concat "." (list 'uuid) "#query-group"))
					/* Scratch rows have their own ordinary nil-writer visibility. The
					original query still retains this exact scope through its decision, so
					readers and any prepared carrier keep their existing native lifetime. */
					(list 'define scope (list 'if (list 'nil? (physical_query_tx_symbol))
						(list 'new_relation_scope (physical_query_session_symbol))
						(list 'tx_get_or_compute_resource (physical_query_tx_symbol)
							(list 'concat "query-group/" name)
							(list 'lambda '() (list 'new_relation_scope (physical_query_session_symbol))))))
					(list 'try (list 'lambda '()
						(list (list 'lambda (list 'result)
							(list 'begin (if retain true
								(list 'if (list 'nil? (physical_query_tx_symbol)) (list 'close_relation_scope scope) true))
								'result)) execution))
						(list 'lambda (list 'failure) (list 'begin
							(list 'if (list 'nil? (physical_query_tx_symbol)) (list 'close_relation_scope scope) true)
							(list 'error 'failure))))))))))

/* The native relation owner is shared by terminal GROUP/DISTINCT emission and
prepared stage-output consumers. Only the latter transfer their scope/table
packet to the enclosing query's explicit lifetime owner. */
(define runtime_group_order_items (lambda (stage order_items) (begin
	(define stage (group_stage_with_facts stage
		(qassoc_set (gs_facts stage) 'distinct_cardinality_outputs true)))
	(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
	(map (coalesceNil order_items '()) (lambda (item)
		(list (replace_group_order_expr (gs_input stage) (group_stage_input_alias stage)
			"__runtime_group" keys (group_key_cols keys) stage (car item)) (cadr item)))))))
(define runtime_group_order_supported? (lambda (stage order_items) (begin
	(define src (list "__runtime_group" (group_stage_schema stage) ".runtime-group" false nil nil))
	(reduce (runtime_group_order_items stage order_items) (lambda (supported item)
		(and supported (not (nil? (direct_column_name_for_alias src
			(sql_type_formula (list src) (car item))))))) true))))
(define lower_runtime_group_output (lambda (stage group_table fields order_items offset_value limit_value result_consumer) (begin
	/* This carrier stores completed DISTINCT cardinalities, not value lists.
	Publish that physical representation to every expression consumer. */
	(define stage (group_stage_with_facts stage
		(qassoc_set (gs_facts stage) 'distinct_cardinality_outputs true)))
	(define src (gs_input stage))
	(define alias (group_stage_input_alias stage))
	(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
	(define key_names (group_key_cols keys))
	(define collations (qassoc_get (gs_facts stage) 'group-value-collations '()))
	(define helper_alias "__runtime_group")
	(define helper_src (list helper_alias (group_stage_schema stage) ".runtime-group" false nil nil))
	(define resolved_fields (map_assoc fields (lambda (_title expr) (canonical_column_expr_for_alias alias expr))))
	(define replaced_fields (replace_group_fields_indexed src alias helper_alias keys key_names stage
		(make_group_key_index keys (extract_assoc resolved_fields (lambda (_title expr) expr))) fields resolved_fields))
	(define replaced_having (replace_group_expr src alias helper_alias keys key_names stage
		(coalesceNil (gs_having stage) true)))
	(define replaced_order (runtime_group_order_items stage order_items))
	(define filtercols (extract_columns_for_alias helper_src replaced_having))
	(define mapcols (merge_unique (extract_assoc replaced_fields (lambda (_title expr) (extract_columns_for_alias helper_src expr)))))
	(define ordercols (order_cols_for_alias helper_src replaced_order))
	(define emit (compile_result_scan_plan result_consumer (list 'scan_order (physical_query_tx_symbol) group_table
		(quoted_runtime_list filtercols)
		(list 'lambda (map filtercols (lambda (col) (scan_callback_symbol_for_alias helper_alias col)))
			(lower_column_expr_for_alias helper_src replaced_having))
		(quoted_runtime_list ordercols)
		(cons 'list (map (coalesceNil order_items '()) (lambda (item)
			(canonical_order_relation (cadr item) (runtime_group_value_collation src collations (car item))))))
		0 (coalesceNil offset_value 0) (coalesceNil limit_value -1) (quoted_runtime_list mapcols)
		(list 'lambda (cons '__acc (map mapcols (lambda (col) (scan_callback_symbol_for_alias helper_alias col))))
			(list 'resultrow (cons 'list (map_assoc replaced_fields (lambda (_title expr) (lower_column_expr_for_alias helper_src expr))))))
		nil false)))
	emit)))
(define lower_runtime_rows_group_stage (lambda (catalog database stage fields order_items offset_value limit_value result_consumer)
	(lower_runtime_group_relation catalog database stage
		(lambda (table _scope) (lower_runtime_group_output stage table fields order_items offset_value limit_value result_consumer)) false)))
/* Physical carriers are lexical native handles. Their names and mutable
lifetime resources never enter a logical source declaration or cache key. */
/* A stage owns declarations in addition to its consumer's catalog. Supply
those real scopes before classifying an external reference as unresolved;
missing references still remain conservatively invocation-owned. */
(define invocation_group_catalog (lambda (stages stage)
	(lowering_catalog_rebind stages
		(unique_stages_by_id (merge (list (nested_stage_catalog stage)
			(lowering_catalog_stages (qassoc_get (gs_facts stage) 'stage_catalog nil))
			(lowering_catalog_stages (qassoc_get (gs_facts stage) 'lowering_catalog nil))
			(lowering_catalog_stages (qassoc_get (gs_facts stage) 'probe_catalog nil))
			(lowering_catalog_stages stages)))))))
(define native_distinct_group_stage? (lambda (stage)
	(not (nil? (find (gs_aggregates stage) distinct_aggregate_descriptor? nil)))))
(define invocation_group_stage? (lambda (stages stage)
	(and (group_stage? stage) (not (scoped_relational_program_stage? stage))
		(or
			(expression_equal? (coalesceNil (lowering_catalog_input_profile stages)
				(lowering_catalog_input_profile (group_stage_lowering_catalog stage))) 'caller)
			/* DISTINCT bounded reductions need native value sets. The invocation
			group profile supplies those independently of source reusability; no
			SQL spelling or freshness flag chooses a physical LIST accumulator. */
			(native_distinct_group_stage? stage)
			/* A residual grouped result is consumed below its caller-bound
			cardinality probe. Its native group state belongs to that producer,
			not an eager maintained carrier with a free outer predicate. */
			(and (grouped_scalar_rows_stage? stage) (stage_has_residual_outer_refs? stage))
			(not (query_result_reusable_using_stages? (invocation_group_catalog stages stage) stage))))))
(define invocation_group_packet_symbol (lambda (id)
	(symbol (concat "__invocation_group_" (fnv_hash id)))))
(define scoped_stage_table_relation? (lambda (relation) (match relation
	((symbol scoped-stage-table) _id) true _ false)))
(define scoped_stage_table_id cadr)
(define invocation_group_table_expr (lambda (stage)
	(list 'cadr (invocation_group_packet_symbol (gs_id stage)))))
(define group_stage_table_expr (lambda (catalog stage)
	(if (invocation_group_stage? catalog stage)
		(invocation_group_table_expr stage)
		(list 'table (group_stage_cache_schema stage) false (group_stage_cache_relation stage) false (physical_query_tx_symbol)))))
(define group_carrier_source_declarations (lambda (stage collations) (begin
	(define keys (group_key_cols (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage))))
	(define names (merge (list keys (group_stage_aggregate_columns stage))))
	(define expressions (merge (list (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage))
		(map (gs_aggregates stage) car))))
	/* These are internal reducer states, not frontend result descriptors.
	The original SQL output contracts remain on their logical consumers. */
	(list "columns" (mapIndex names (lambda (index name) (begin
		(define collation (runtime_group_value_collation (gs_input stage) collations (nth expressions index)))
		(list "Field" name "RawType" "any" "Type" "any" "Null" true
			"Collation" (if (string? collation) collation nil)
			"Dimensions" '() "Key" (if (contains? keys name) "PRI" "")))))
		"constraints" (list "Unique" (list (list "Name" "group" "Cols" keys)) "ForeignKeys" '())
		"references" '()))))
(define invocation_group_source_declarations (lambda (stage)
	(merge (list
		(group_carrier_source_declarations stage
			(qassoc_get (gs_facts stage) 'group-value-collations '()))
		(list "lifetime" "invocation")))))
(define lower_runtime_group_carrier (lambda (catalog stage) (begin
	(define value (lower_runtime_group_relation catalog (group_stage_schema stage) stage
		(lambda (table scope) (list 'list scope table)) true))
	(define packet '__created_invocation_group)
	(list (list 'lambda (list packet)
		(list 'begin
			(list 'if (list 'nil? (physical_query_tx_symbol))
				(list '__invocation_group_scopes (list 'uuid) (list 'car packet)) true)
			packet)) value))))
/* Nil-transaction callers have no decision resource owner. Retained native
scopes then close after the complete emitted consumer, including failure. The
registry contains lifetime capabilities only, never SQL rows or group state. */
/* A DML-preserved membership marker consumes a prepared stable key domain.
Read only this query's executable predicates, not nested query scopes or the
stage's declaration/provenance payload. */
(define invocation_membership_stages (lambda (node dml catalog) (match node
	((symbol quote) _datum) '()
	(cons (symbol query-block) _parts) '()
	(cons (symbol union-block) _parts) '()
	((symbol driver_membership_probe) stage _probe)
	(if (and (invocation_group_stage? catalog stage)
		(or dml (nil? (recset_domain_source (gs_input stage))))) (list stage) '())
	((symbol driver_membership_subscan_probe) stage _probe)
	(if (and dml (invocation_group_stage? catalog stage)) (list stage) '())
	((symbol dml_driver_membership_probe) _schema stage _probe)
	(if (invocation_group_stage? catalog stage) (list stage) '())
	(cons head tail) (unique_stages_by_id (merge (cons
		(invocation_membership_stages head dml catalog)
		(map tail (lambda (child) (invocation_membership_stages child dml catalog))))))
	_ '())))
(define query_block_invocation_membership_stages (lambda (block dml)
	(invocation_membership_stages
		(list (qb_fields block) (qb_where block) (qb_having block) (qb_order block)
			(qb_hidden block) (map (qb_sources block) source_join_expr))
		dml (query_block_stage_lookup block))))
(define physical_plan_uses_invocation_groups? (lambda (node) (match node
	((symbol quote) _datum) false
	(cons head tail) (not (nil? (find node physical_plan_uses_invocation_groups? nil)))
	_ (or (expression_equal? node '__invocation_group_scopes)
		(expression_equal? node '__invocation_relation_initializers)))))
/* Regional alternatives share the existing nil-Tx observation registries for
exactly one actual query dispatch, including its bounded guard/miss chain.
Three empty maps are lexical to the profiled dispatch; native group scopes
are allocated only by its selected producers. Nothing here
is retained in a compiled variant, and this supplies no native read authority. */
(define call_with_invocation_group_lifetime (lambda (consume)
	(begin
		(define scopes (newsession)) (define initializers (newsession))
		(define observations (newsession))
		(define cleanup (lambda ()
			(map (scopes) (lambda (key) (close_relation_scope (scopes key))))))
		(try (lambda () (begin
			(define result (consume (lambda () scopes) (lambda () initializers) (lambda () observations)))
			(cleanup) result))
			(lambda (failure) (begin (cleanup) (error failure)))))))

(define bind_invocation_group_lifetimes (lambda (plan)
	(if (not (physical_plan_uses_invocation_groups? plan)) plan (begin
		/* A zero-argument session call returns its keys, never assoc pairs. */
		(define cleanup (list 'map (list '__invocation_group_scopes)
			(list 'lambda (list '_key)
				(list 'close_relation_scope (list '__invocation_group_scopes '_key)))))
		(list (list 'lambda (list '__invocation_group_scopes '__invocation_relation_initializers)
			(list 'try (list 'lambda '()
				(list (list 'lambda (list 'result) (list 'begin cleanup 'result)) plan))
				(list 'lambda (list 'failure) (list 'begin cleanup (list 'error 'failure)))))
			(list 'newsession) (list 'newsession))))))

(define runtime_group_volatile_input? (lambda (input)
	(if (union_block? input)
		(not (nil? (find (union_branches input) runtime_group_volatile_input? nil)))
		(query_block? input)
		(not (nil? (find (qb_sources input) (lambda (src)
			(or (runtime_rows_source? src)
				(if (or (query_block? (source_relation src)) (union_block? (source_relation src)))
					(runtime_group_volatile_input? (source_relation src)) false))) nil)))
		(runtime_rows_source? input))))

/* This physical profile owns the original group consumer only. A derived
wrapper can retain the same stage/source while changing projection, predicates
or independent window barriers; reject it before allocating any native scope. */
(define runtime_union_group_consumer_matches? (lambda (block stage) (begin
	(define output (car (qb_sources block)))
	(define alias (group_stage_input_alias stage))
	(define replace_expr (lambda (expr)
		(replace_group_expr (gs_input stage) alias (source_alias output)
			(gs_keys stage) (group_key_cols (gs_keys stage)) stage expr)))
	(and (not (source_outer? output)) (nil? (source_join_expr output))
		(empty_list? (qb_group block)) (nil? (qb_having block))
		(empty_list? (qb_hidden block))
		(equal? (count (gs_order stage)) (count (qassoc_get (gs_facts stage) 'group-consumer-order '())))
		(expression_equal? (qb_fields block) (map_assoc (gs_output stage) (lambda (_title expr) (replace_expr expr))))
		(expression_equal? (qb_where block) (replace_expr (coalesceNil (gs_having stage) true)))
		(expression_equal? (coalesceNil (qb_order block) '())
			(map (qassoc_get (gs_facts stage) 'group-consumer-order '())
				(lambda (item) (list (replace_expr (car item)) (cadr item)))))
		(expression_equal? (qb_limit block) (gs_limit stage))
		(expression_equal? (qb_offset block) (gs_offset stage))))))

/* Representative descriptors preserve order operands, while the shared SQL
contract owns the comparator. Keep both from the same canonical consumer. */
(define runtime_union_group_order (lambda (stage)
	(map (zip (gs_order stage) (qassoc_get (gs_facts stage) 'group-consumer-order '()))
		(lambda (pair) (list (car (car pair)) (cadr (cadr pair)))))))

/* A root UNION group is already a logical stage. Consume that whole bag with
its original output/HAVING/window contract instead of publishing a maintained
cache for catalogs which have no native maintenance owner. */
(define runtime_union_group_stage_for_block (lambda (block) (begin
	(define stages (filter (qb_stages block) (lambda (stage)
		(and (group_stage? stage) (union_block? (gs_input stage))
			(not (scoped_relational_program_stage? stage))
			(runtime_group_volatile_input? (gs_input stage))))))
	(if (empty_list? stages) nil (begin
		(define stage (car stages))
		(if (and (equal? (count (qb_stages block)) 1) (single_source? (qb_sources block))
			(stage_output_relation? (source_relation (car (qb_sources block))))
			(equal? (stage_output_relation_id (source_relation (car (qb_sources block)))) (gs_id stage))
			(equal? (union_mode (gs_input stage)) 'all)
			(empty_list? (union_order (gs_input stage)))
			(nil? (union_limit (gs_input stage))) (nil? (union_offset (gs_input stage))))
			(if (runtime_union_group_consumer_matches? block stage) stage
				(neumann_fail "build_queryplan" "volatile UNION grouping does not yet support a composed outer consumer"))
			(neumann_fail "build_queryplan" "volatile UNION grouping requires one unbounded UNION ALL group stage")))))))
