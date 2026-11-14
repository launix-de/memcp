/* Copyright (C) 2026 Carl-Philip Haensch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Scoped programs are explicit group-stage barriers. Their input children
remain typed logical queries, bound and decorrelated before physical lowering.
UNION ALL is a named child container for iteration/choice, never their bag
semantics. This one classifier owns all shared child-scope boundaries. */
(define scoped_relational_program_stage? (lambda (stage)
	(and (group_stage? stage)
		(reduce (list 'relational_iteration 'relational_choice 'relational_sequence 'local_relation_scope 'local_relation_insert 'local_relation_update)
			(lambda (found purpose) (or found
				(expression_equal? purpose (qassoc_get (gs_facts stage) 'purpose nil)))) false))))

/* Freshness follows executable logical children, never catalog/provenance
facts. Constructors pass nil for an intrinsic/visible-child summary. Reuse
owners supply resolve_stage(id): an unresolved external reference then cannot
claim purity. Query-block declarations shadow the enclosing resolver, including
forward references to another locally declared stage. Resolution and cycle
tracking stay here rather than widening the preparation dependency graph. */
/* Each lexical stage catalog owns its analysis entries. References and the
explicit declaration list visit the same entry; repeated readers never expand
that stage's complete dependency program again. An inherited declaration is
analysed with its declaring resolver, not a shadowing caller's local catalog. */
(define fresh_invocation_scope_resolver (lambda (stages parent lookup) (begin
	(define entries (make_structural_catalog 'ast))
	(define register (lambda (id stage) (begin
		(define entry (newsession))
		(entry "stage" stage)
		(entries id entry)
		entry)))
	(map stages (lambda (stage) (if (group_stage? stage) (register (gs_id stage) stage) nil)))
	(define resolve (lambda (resolve id visit) (begin
		(define entry (entries id))
		(define entry (if (or entry (nil? lookup)) entry (begin
			(define stage (lookup id))
			(if (nil? stage) nil (register id stage)))))
		(if (nil? entry) (parent id visit)
			(equal? (entry "status") "done") (entry "fresh")
			/* A genuine recursive dependency remains non-reusable. The normal
			stage dependency owner retains its explicit cycle error. */
			(equal? (entry "status") "visiting") true
			(begin
				(entry "status" "visiting")
				(define fresh (visit (entry "stage")
					(lambda (id visit) (resolve resolve id visit))))
				(entry "fresh" fresh)
				(entry "status" "done")
				fresh)))))
	(lambda (id visit) (resolve resolve id visit)))))
/* These are the existing invocation-volatile scalar intrinsics, expressed in
both reader and callable forms. Their execution effect is independent of SQL
source naming; arbitrary procedures are not classified through a whitelist. */
(define fresh_invocation_intrinsic? (lambda (head)
	(reduce (list 'now now 'sql_rand sql_rand 'randomBytes randomBytes
		'uuid uuid 'uuid_v1 uuid_v1 'uuid_v4 uuid_v4 'uuid_v7 uuid_v7)
		(lambda (fresh intrinsic) (or fresh (expression_equal? head intrinsic))) false)))
(define query_requires_fresh_invocation? (lambda (node resolve_stage) (begin
	(define complete (not (nil? resolve_stage)))
	(define source_fresh (lambda (walk source resolver)
		(or (source_invocation_lifetime? source)
			(walk walk (source_relation source) resolver)
			(walk walk (source_join_expr source) resolver))))
	(define walk_stages (lambda (walk stages resolver)
		(reduce stages (lambda (fresh stage) (or fresh
			(if (group_stage? stage)
				(resolver (gs_id stage) (lambda (declaration owner) (walk walk declaration owner)))
				(walk walk stage resolver)))) false)))
	(define walk (lambda (walk node resolver) (match node
		((symbol quote) _datum) false
		((symbol now)) true
		((symbol literal-rows) _columns _data) false
		((symbol local-relation) _symbol _columns) true
		((symbol stage-output) id)
		(resolver id (lambda (stage owner) (walk walk stage owner)))
		((symbol neumann-ir) _kind root stages _context _return)
		(begin
			(define local_resolver (fresh_invocation_scope_resolver stages resolver nil))
			(or (walk walk root local_resolver)
				(walk_stages walk stages local_resolver)))
		((symbol group-stage) _id input domain keys aggregates having output order limit offset facts)
		(or (qassoc_get facts 'fresh_invocation false)
			(if (or (query_block? input) (union_block? input)) (walk walk input resolver)
				(source_fresh walk input resolver))
			(reduce (list domain keys aggregates having output order limit offset
				(qassoc_get facts 'iteration_predicate nil))
				(lambda (fresh child) (or fresh (walk walk child resolver))) false))
		((symbol query-block) _db sources fields condition grouping having ordering limit offset hidden stages facts)
		(begin
			(define local_resolver (fresh_invocation_scope_resolver stages resolver nil))
			(or (qassoc_get facts 'fresh_invocation false)
				(reduce sources (lambda (fresh source)
					(or fresh (source_fresh walk source local_resolver))) false)
				(reduce (list fields condition grouping having ordering limit offset hidden)
					(lambda (fresh child) (or fresh (walk walk child local_resolver))) false)
				(walk_stages walk stages local_resolver)))
		((symbol union-block) _mode branches ordering limit offset _facts)
		(reduce (list branches ordering limit offset)
			(lambda (fresh child) (or fresh (walk walk child resolver))) false)
		((symbol window-stage) _id source _column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit facts)
		(or (qassoc_get facts 'fresh_invocation false)
			(source_fresh walk source resolver)
			(reduce (list sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit)
				(lambda (fresh child) (or fresh (walk walk child resolver))) false))
		((symbol orc-stage) _id source _column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit)
		(or (source_fresh walk source resolver)
			(reduce (list sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit)
				(lambda (fresh child) (or fresh (walk walk child resolver))) false))
		((symbol sql_type_expr) formula _type _collation) (walk walk formula resolver)
		(cons head tail)
		(or (fresh_invocation_intrinsic? head)
			(walk walk head resolver)
			(reduce tail (lambda (fresh child) (or fresh (walk walk child resolver))) false))
		_ false)))
	(walk walk node (fresh_invocation_scope_resolver '()
		(lambda (_id _visit) complete) resolve_stage)))))
(define scoped_relational_program_input (lambda (stage children)
	(if (union_block? (gs_input stage))
		(make_union_block 'all children '() nil nil (union_facts (gs_input stage)))
		(car children))))
(define scoped_relational_program_children (lambda (stage)
	(if (union_block? (gs_input stage)) (union_branches (gs_input stage)) (list (gs_input stage)))))
(define scoped_relational_program_external_reads (lambda (purpose input facts) (begin
	(define children (if (union_block? input) (union_branches input) (list input)))
	(define state (qassoc_get facts 'state_symbol nil))
	(merge_unique (mapIndex children (lambda (index child)
		(filter (query_expr_session_reads child) (lambda (read)
			/* Feedback and clock operands belong to the chosen branch/iteration.
			The seed's free reads still belong to the invocation's outer domain. */
			(not (and (> index 0)
				(or (expression_equal? purpose 'relational_choice)
					(expression_equal? purpose 'relational_iteration)
					(expression_equal? purpose 'relational_sequence))
				(match read
					(cons head (cons value '()))
					(and (expression_equal? head relational_iteration_parameter)
						(or (expression_equal? value state)
							(expression_equal? value (list 'now))))
					_ false)))))))))))
(define scoped_relational_program_rebuild (lambda (stage input domain facts)
	(make_group_stage (gs_id stage) input domain domain (gs_aggregates stage)
		nil '() '() nil nil (qassoc_set facts 'lookup-keys domain))))
(define make_scoped_relational_program (lambda (purpose database input sources facts) (begin
	(define facts (qassoc_set facts 'fresh_invocation
		(or (expression_equal? purpose 'local_relation_scope)
			(expression_equal? purpose 'local_relation_insert)
			(expression_equal? purpose 'local_relation_update)
			(query_requires_fresh_invocation? input nil))))
	(define identity (stable_structural_hash (list purpose input facts) false))
	(define id (concat "scoped-program:" identity))
	(define alias (concat "__program_" identity))
	(define descriptor (list '__program_state 'relational_iteration_last_reduce nil))
	(define column (aggregate_col_name descriptor))
	(define domain (merge_unique (list (relational_iteration_outer_refs input sources)
		(scoped_relational_program_external_reads purpose input facts))))
	(define stage (make_group_stage id input domain domain (list descriptor) nil '() '() nil nil
		(merge (list facts (list (list 'purpose purpose) (list 'lookup-keys domain)
			(list 'null_semantics 'scoped_program) (list 'result_max_rows_per_partition 1))))))
	(define declarations (list "columns" (list (list "Field" column "Type" "any" "RawType" "any"
		"Dimensions" '() "Null" false "Collation" "")) "constraints" '("Unique" () "ForeignKeys" ())))
	(make_query_block database
		(list (list alias database (make_stage_output_relation id) false true declarations))
		(list "__fn_state" (list 'get_column alias false column false))
		true nil nil nil nil nil nil (list stage) '()))))

(define bind_scoped_relational_program_stage (lambda (stage scopes path strings)
	(if (relational_iteration_stage? stage) (bind_relational_iteration_stage stage scopes path strings)
		(begin
			(define children (mapIndex (scoped_relational_program_children stage) (lambda (index child)
				(bind_query_names_at child scopes (binding_path path "program" index) strings))))
			(define input (scoped_relational_program_input stage children))
			(define domain (merge_unique (list
				(relational_iteration_outer_refs input (map (merge scopes) binding_entry_source))
				(scoped_relational_program_external_reads
					(qassoc_get (gs_facts stage) 'purpose nil) input (gs_facts stage)))))
			(scoped_relational_program_rebuild stage input domain (gs_facts stage))))))
(define untangle_scoped_relational_program_stage (lambda (stage ctx)
	(if (relational_iteration_stage? stage) (untangle_relational_iteration_stage stage ctx)
		(begin
			(define local_context (make_uctx ctx (list (list 'defer-subquery-rewrites false))))
			(scoped_relational_program_rebuild stage
				(scoped_relational_program_input stage
					(map (scoped_relational_program_children stage) (lambda (child)
						(untangle_query child local_context))))
				(gs_domain stage) (gs_facts stage))))))

/* Seed is evaluated once, then exactly one branch sees the immutable lexical
state. In particular no INSERT or invalid conversion in the unchosen branch
may be prepared or evaluated before the native IF. */
(define make_relational_choice (lambda (seed yes no state_symbol outer_sources predicate)
	(make_scoped_relational_program 'relational_choice (qb_schema seed)
		(make_union_block 'all (list seed
			(relational_iteration_inputs yes state_symbol)
			(relational_iteration_inputs no state_symbol)
			(relational_iteration_inputs predicate state_symbol)) '() nil nil
			(list (list 'scoped_program 'relational_choice))) outer_sources
		(list (list 'state_symbol state_symbol)))))
/* Sequential query assignments have an explicit lexical state producer.
The transition is a logical child, not a correlated row-time interpreter. */
(define make_relational_sequence (lambda (seed step state_symbol outer_sources)
	(make_scoped_relational_program 'relational_sequence (qb_schema seed)
		(make_union_block 'all (list seed (relational_iteration_inputs step state_symbol))
			'() nil nil (list (list 'scoped_program 'relational_sequence))) outer_sources
		(list (list 'state_symbol state_symbol)))))
(define make_local_relation_scope (lambda (query declarations outer_sources)
	(make_scoped_relational_program 'local_relation_scope (qb_schema query) query outer_sources
		(list (list 'local_relations declarations)))))
(define make_local_relation_insert (lambda (query relation columns outer_sources)
	(make_scoped_relational_program 'local_relation_insert (qb_schema query) query outer_sources
		(list (list 'target_relation relation) (list 'insert_columns columns)))))

/* UPDATE's input is a normal mutation query; its bounded output is the
unchanged predecessor state. The enclosing sequence owns that lexical value. */
(define make_local_relation_update (lambda (query state_symbol outer_sources)
	(make_scoped_relational_program 'local_relation_update (qb_schema query) query outer_sources
		(list (list 'state_symbol state_symbol)))))

/* A multiset producer owns declarations and one explicit initialization
dependency. Its output remains an ordinary typed query, never a scalar state
aggregate. The initializer is an executable stage, not code hidden in facts. */
(define query_block_invocation_scope? (lambda (query)
	(and (query_block? query) (not (nil? (qassoc_get (qb_facts query) 'invocation_initializer nil))))))
(define query_block_invocation_initializer (lambda (query stages)
	(if (query_block_invocation_scope? query)
		(stage_by_id stages (qassoc_get (qb_facts query) 'invocation_initializer nil)) nil)))
(define make_invocation_relation_query (lambda (query declarations initializer)
	(make_query_block (qb_schema query) (qb_sources query) (qb_fields query) (qb_where query)
		(qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query)
		(qb_hidden query) (cons initializer (qb_stages query))
		(qassoc_set (qassoc_set (qassoc_set (qb_facts query) 'invocation_initializer (gs_id initializer))
			'invocation_relations declarations) 'fresh_invocation true))))

/* Preparation owns the native handles lexically through its complete query
consumer. Active native writers retain their scope through the transaction
decision; standalone consumers use the emitted query's error-safe registry. */
(define invocation_relation_prepare (lambda (query)
	(begin
		(define catalog (query_block_stage_lookup query))
		(define initializer (query_block_invocation_initializer query catalog))
		(if (and initializer (scoped_relational_program_stage? initializer)) true
			(neumann_fail "build_queryplan" "invocation relation lost its executable initializer"))
		(define scope '__invocation_relation_scope)
		(define name '__invocation_relation_name)
		(define tx (physical_query_tx_symbol))
		(define declarations (qassoc_get (qb_facts query) 'invocation_relations '()))
		(define identity (qassoc_get (qb_facts query) 'invocation_scope_identity nil))
		(if (string? identity) true
			(neumann_fail "build_queryplan" "invocation relation has no bound lexical identity"))
		(define packet '__invocation_relation_packet)
		(define prefix (list
			(list 'define name (list 'concat ".invocation-relation:" (list 'uuid)))
			(list 'define scope (list 'if (list 'nil? tx)
				(list 'new_relation_scope (physical_query_session_symbol))
				(list 'tx_get_or_compute_resource tx name
					(list 'lambda '() (list 'new_relation_scope (physical_query_session_symbol))))))
			(list 'if (list 'nil? tx) (list '__invocation_group_scopes name scope) true)))
		(define create (merge (map declarations (lambda (declaration)
			(list
				(list 'define name (list 'concat ".local-relation:" (list 'uuid)))
				(list 'createtable (nth declaration 1) name (list 'quote (nth declaration 2))
					(list 'list "engine" "memory" "ephemeral" true "scope" scope) false tx)
				(list 'define (car declaration) (list 'table (nth declaration 1) false name false tx)))))))
		(define core (make_query_block (qb_schema query) (qb_sources query) (qb_fields query)
			(qb_where query) (qb_group query) (qb_having query) (qb_order query) (qb_limit query)
			(qb_offset query) (qb_hidden query)
			(filter (qb_stages query) (lambda (stage) (not (equal? (gs_id stage) (gs_id initializer)))))
			(filter (qb_facts query) (lambda (entry)
				(not (has? '("invocation_initializer" "invocation_relations") (string (car entry))))))))
		(define initialize (cons 'begin (merge (list prefix create
			(list (lower_scoped_relational_program_probe '() nil initializer tx)
				(cons 'list (cons scope (map declarations car))))))))
		(define once (list '__invocation_relation_initializers "get_or_compute_scoped" 0 identity tx
			(list 'lambda (list 'tx) initialize)))
		(list (cons (list 'define packet once)
			(mapIndex declarations (lambda (index declaration)
				(list 'define (car declaration) (list 'nth packet (+ index 1)))))) core))))

/* All invocation dependencies become actual scalar lambda operands before
entering any child program. The rewrite includes nested stage descriptors and
their logical predecessors, not merely the visible parent projection. */
(define lower_scoped_relational_program_probe (lambda (sources default_alias stage tx)
	(if (relational_iteration_stage? stage)
		(lower_relational_iteration_probe sources default_alias stage tx)
		(begin
			(define lookup (qassoc_get (gs_facts stage) 'lookup-keys '()))
			(define parameters (mapIndex lookup (lambda (index _key)
				(symbol (concat "__program_domain_" index)))))
			(define logical (qassoc_get (gs_facts stage) 'btw2025_lookup_keys lookup))
			(define index (scalar_query_probe_param_index lookup logical parameters))
			(define bound (rewrite_scalar_query_probe_params index stage))
			(define purpose (qassoc_get (gs_facts bound) 'purpose nil))
			(define program (if (expression_equal? purpose 'relational_choice)
				(lower_relational_choice_program bound tx)
				(expression_equal? purpose 'relational_sequence)
				(lower_relational_sequence_program bound tx)
				(expression_equal? purpose 'local_relation_scope)
				(lower_local_relation_scope_program bound tx)
				(expression_equal? purpose 'local_relation_insert)
				(lower_local_relation_insert_program bound tx)
				(expression_equal? purpose 'local_relation_update)
				(lower_local_relation_update_program bound tx)
				(neumann_fail "build_queryplan" "unknown scoped relational program")))
			(cons (list 'lambda parameters program)
				(map lookup (lambda (key) (lower_column_expr_for_join sources default_alias key))))))))
(define lower_relational_sequence_program (lambda (stage tx) (begin
	(define children (scoped_relational_program_children stage))
	(define symbol (qassoc_get (gs_facts stage) 'state_symbol nil))
	(list (list 'lambda (list symbol)
		(list 'begin (list 'tx_check tx)
			(lower_relational_iteration_program (nth children 1))))
		(lower_relational_iteration_program (car children))))))
(define lower_relational_choice_program (lambda (stage tx) (begin
	(define children (scoped_relational_program_children stage))
	(define symbol (qassoc_get (gs_facts stage) 'state_symbol nil))
	(define predicate (list 'car (lower_relational_iteration_program (nth children 3))))
	(list (list 'lambda (list symbol)
		(list 'begin (list 'tx_check tx)
			(list 'if predicate
				(lower_relational_iteration_program (nth children 1))
				(lower_relational_iteration_program (nth children 2)))))
		(lower_relational_iteration_program (car children))))))

/* Each declaration is (symbol database native-columns SQL-columns). The only
fresh physical identity belongs to this invocation's native scope. It never
participates in a logical cache identity or crosses the scalar result boundary.
Callbacks capture ordinary native table handles, not the lifetime capability. */
(define lower_local_relation_scope_program (lambda (stage tx) (begin
	(define declarations (qassoc_get (gs_facts stage) 'local_relations '()))
	(define scope '__local_relation_scope)
	(define outcome '__local_relation_outcome)
	(define program (reduce (reverse declarations) (lambda (body declaration) (begin
		(define name '__local_relation_name)
		(define create (list 'begin
			(list 'createtable (nth declaration 1) name (list 'quote (nth declaration 2))
				(list 'list "engine" "memory" "ephemeral" true "scope" scope) false tx)
			(list 'table (nth declaration 1) false name false tx)))
		(list (list 'lambda (list name)
			(list (list 'lambda (list (car declaration)) body) create))
			(list 'concat ".local-relation:" (list 'uuid)))))
		(lower_relational_iteration_program (gs_input stage))))
	(list (list 'lambda (list scope)
		(list (list 'lambda (list outcome)
			(list 'begin (list 'close_relation_scope scope)
				(list 'if (list 'car outcome) (list 'cadr outcome) (list 'error (list 'cadr outcome)))))
			(list 'try (list 'lambda '() (list 'list true program))
				(list 'lambda (list '__local_relation_failure)
					(list 'list false '__local_relation_failure)))))
		(list 'new_relation_scope (list 'newsession))))))

/* The input is the previous one-row state LEFT JOIN the INSERT query. Its
explicit presence column separates zero input rows from a real nullable row.
This normal relational producer is compiled once; its native sink streams rows
without a Scheme bag. LAST combines only the fixed-size forwarded state. */
(define lower_local_relation_insert_program (lambda (stage tx) (begin
	(define columns (qassoc_get (gs_facts stage) 'insert_columns '()))
	(define relation (qassoc_get (gs_facts stage) 'target_relation nil))
	(define input (gs_input stage))
	(define state '__local_insert_state)
	(define present '__local_insert_present)
	(define values (mapIndex columns (lambda (index _column)
		(symbol (concat "__local_insert_value_" index)))))
	(define fields (merge (list (list "__fn_state" (get_assoc (qb_fields input) "__fn_state")
		"__insert_present" (get_assoc (qb_fields input) "__insert_present"))
		(merge (mapIndex columns (lambda (index column)
			(list column (get_assoc (qb_fields input) (concat "__insert_value_" index)))))))))
	(define insert (list 'insert relation (list 'quote columns)
		(list 'list (cons 'list values)) (list 'list) nil false nil 'tx))
	(define map_row (list 'lambda (merge (list (list state present) values))
		(list 'begin (list 'if present insert true) state)))
	/* The sink consumes the same prepared projection as an ordinary dataset
	consumer. Its predecessor's stages remain inside this insertion step. */
	(define prepared (prepare_simple_query_block_physical_core
		(make_query_block (qb_schema input) (qb_sources input) fields (qb_where input)
			(qb_group input) (qb_having input) (qb_order input) (qb_limit input) (qb_offset input)
			(qb_hidden input) (qb_stages input) (qb_facts input))))
	(define core (nth prepared 1))
	(define stream (lower_query_block_as_dataset_reduce core (qb_fields core) map_row
		'relational_iteration_last_reduce nil nil))
	(define stream (cons '!begin (merge (list (nth prepared 0)
		(list (list 'define '__local_insert_result stream))
		(nth prepared 2) (list '__local_insert_result)))))
	(physical_mutation_producer (list relation) (list "insert" '()) stream))))

/* The generic DML owner executes the complete selection and mutation once.
Local native handles are lexical physical inputs, not catalog-name lookups. */
(define lower_local_relation_update_program (lambda (stage tx) (begin
	(define input (gs_input stage))
	(define relation (qassoc_get (qb_facts input) 'local_update_target nil))
	(define target (dml_target_source (qb_sources input) (qb_schema input) relation
		(qassoc_get (qb_facts input) 'dml_target_alias nil)))
	(if (and target (local_relation_reference? (source_relation target))) true
		(neumann_fail "build_queryplan" "local UPDATE target lost its lexical declaration"))
	(list 'begin
		(physical_mutation_producer (list (source_table_expr target)) (physical_dml_descriptor input)
			(lower_dml_query_block_with_stages input (qb_schema input) relation))
		(qassoc_get (gs_facts stage) 'state_symbol nil)))))
