/*
Copyright (C) 2023 - 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* Query-plan builders emit session as code before a request-local session is
in scope. Keep the global binding as an inert self-representing symbol; SQL
execution shadows it with the concrete session captured by the frontend. */
(define session (quote session))

(import "sql-parser.scm")
(import "sql-parameters.scm")
(import "psql-parser.scm")
(import "sql-builtins.scm")
(import "sql-metadata.scm")
(import "queryplan.scm")
(import "sql-views.scm")
(import "sql-parameter-cost.scm")

/* Root is the recovery identity for a fresh/local installation. User-facing
maintenance commands may change its password, but may not remove the account
or its administrator capability. */
(define protect_sql_root_admin (lambda (username operation)
	(if (equal?? username "root")
		(error (concat "cannot " operation " protected user root"))
		true)))

(define protected_sql_admin_revoke_value (lambda (username) (begin
	(protect_sql_root_admin username "remove administrator privileges from")
	false)))

/* query plan caches: separate cachemap per parser dialect */
(set sql_queryplan_cache (newcachemap))
(set psql_queryplan_cache (newcachemap))
(set sql_literal_shape_cache (newcachemap))

/* Statistics guards deliberately specialize physical plans, but exact row
counts can change after every small write. Keep only the newest regimes: old
variants remain correct only while their guards match, and can be recompiled
if an old regime returns. Bounding the list prevents large plans from growing
the dispatch formula and retained ASTs without limit. */
(set sql_queryplan_max_variants 4)

(define sql_queryplan_recent_variants (lambda (variants)
	(map (produceN (min sql_queryplan_max_variants (count variants)))
		(lambda (idx) (nth variants idx)))))

/* A parse can mention the same physical table through many aliases. Resolve
and authorize each readable table once, then keep using show's immutable table
snapshot through the stable handle for the rest of this compile only. */
(define sql_compile_table_policy (lambda (policy tx access readonly)
	(begin
		/* This one compile owner publishes independent target facts. Policy
		callbacks may run in parallel; per-database session stores cannot lose
		a target through a read/append/write race. The plan retains strings only. */
		(access "write_databases" (newsession))
		(define catalog (newsession))
		(lambda (schema tbl write)
			(if (and write (sql_database_create_request? tbl)) (begin
				(define actor (sql_database_create_policy_result (if policy (policy schema tbl true) true)))
				/* Preserve the authorization value across this wrapper. The
				collector retains request data, never an admin/member decision. */
				(access "database-create" (cons (list schema (cadr tbl) actor)
					(coalesceNil (access "database-create") '())))
				actor)
				write (begin
					(if policy (policy schema tbl true) true)
					(access "write" true)
					((access "write_databases") schema true)
					(if (and (equal? schema "system") (equal? tbl true)) (access "admin" true) true)
					/* Native schema/shard access acquires ownership. Keep write
					intent on cached plans; admin commands route to the master. */
					(if (or readonly (equal? schema "system")) true (replication_write tx schema)))
				(begin
					(define handle (table schema false tbl false tx))
					(if (and handle (catalog handle))
						true
						(begin
							(if policy (policy schema tbl false) true)
							(if handle
								(begin
									(show handle)
									(catalog handle true))
								true)
							true))))))))

/* Copy request bindings and catalog context into an isolated planning session.
The request transaction is deliberately excluded: cached plans must obtain
__memcp_tx from the executing session, never retain the transaction which
happened to compile a shared variant. The condition accumulator belongs to
exactly one compile and substitutes for threading guard state through every
functional planner return value. */
(define sql_queryplan_initialize_collector (lambda (planning_session)
	(begin
		(planning_session "__memcp_queryplan_guard_conditions" (newsession))
		(planning_session "__memcp_queryplan_guard_condition_catalog" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_guard_bindings" (newsession))
		(planning_session "__memcp_queryplan_guard_binding_catalog" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_statistics_dependencies" (newsession))
		(planning_session "__memcp_queryplan_statistics_dependency_catalog"
			(make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_guarded_session_keys" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_observed_session_keys" (newsession))
		(planning_session "__memcp_queryplan_preparations" (newsession))
		planning_session)))

(define sql_queryplan_fork_collector (lambda (parent)
	(if (nil? parent) nil
		(begin
			(define branch (newsession))
			(reduce (parent) (lambda (_ key) (branch key (parent key))) nil)
			(sql_queryplan_initialize_collector branch)
			(define records (newsession))
			(records "dispatch_owner" (sql_queryplan_input_dispatch_owner parent))
			(branch "__memcp_queryplan_input_alternatives" records)
			branch))))

(define sql_queryplan_compile_session (lambda (source_session tx input_lifetime dispatch_owner)
	(begin
		(define planning_session (newsession))
		(define compile_bindings (newsession))
		(define observation_scope (source_session
			"get_or_compute_scoped"
			(tx_query tx)
			"__memcp_queryplan_observations"
			tx
			(lambda (_tx) (newsession))))
		(reduce (source_session) (lambda (_ key)
			(if (match key (regex "^v[0-9]+$" _) true false)
				(begin
					(compile_bindings key (source_session key))
					(planning_session key (source_session key)))
				/* Prepared carriers are opaque native values owned by the executing
				query. Compilation consumes only their scalar observation; copying a
				RecSet here would also make structural guard/catalog code inspect it. */
				(if (or (equal? key "__memcp_tx")
					(match key (regex "^__memcp_queryplan_observation_value_" _) true false))
					nil
					(planning_session key (source_session key))))) nil)
		(planning_session "__memcp_queryplan_compile_bindings" compile_bindings)
		(planning_session "__memcp_queryplan_observation_scope" observation_scope)
		(planning_session "__memcp_queryplan_public_observation_scope" observation_scope)
		(planning_session "__memcp_queryplan_input_lifetime" input_lifetime)
		(sql_queryplan_initialize_collector planning_session)
		(define input_records (newsession))
		/* Only an actual builtin parser may later grant its sole semantic query.
		Opaque factories, DML bodies and batches retain complete runtime arms. */
		(input_records "read_parser_owner" dispatch_owner)
		(input_records "dispatch_owner" nil)
		(planning_session "__memcp_queryplan_input_alternatives" input_records)
		planning_session)))

/* Physical records are compile-only. Existing ordinary declaration eligibility
stays outside source resolution; actual input mode is an immutable execution
parameter, never a Session freshness/authority flag. */
/* Only the actual cache dispatcher supplies this compile-only provenance.
Direct compilers and DML collectors explicitly leave it absent. */
/* Explicit lexical compile view for one builtin frontend's root SELECT.
Every ordinary collector write still reaches the SAME guard/work/record stores.
This view is compile-only; no new registry or runtime Session mode is emitted. */
(define sql_queryplan_read_dispatch_session (lambda (planning parser_owner)
	(if (nil? planning) nil
		(begin
			(define records (planning "__memcp_queryplan_input_alternatives"))
			(define owner (if (nil? records) nil (records "read_parser_owner")))
			(if (or (nil? owner) (not (identical? (records "read_parser") parser_owner))) planning
				(begin
					(define read_records (lambda args (match args
						'(key) (if (equal? key "dispatch_owner") owner (records key))
						_ (apply records args))))
					(lambda args (match args
						'(key) (if (equal? key "__memcp_queryplan_input_alternatives") read_records (planning key))
						_ (apply planning args)))))))))

/* Procedure identity certifies the complete frontend owner, not a SQL prefix.
A custom parser can wrap transaction controls around several SELECT builders;
it deliberately receives no sole-query grant even if it calls a builtin. */


(define sql_queryplan_input_dispatch_owner (lambda (planning)
	(if (nil? planning) nil
		(begin
			(define records (planning "__memcp_queryplan_input_alternatives"))
			(if (nil? records) nil (records "dispatch_owner"))))))
(define sql_queryplan_input_record_dispatched? (lambda (record)
	(and (> (count record) 4) (not (nil? (nth record 4))))))
(define sql_queryplan_input_admission (lambda (records)
	(sql_queryplan_conjoin_guards (map (filter records sql_queryplan_input_record_dispatched?)
		(lambda (record)
			(list 'equal?
				(list 'if (list 'or (list 'tx_requires_query_local_cache 'tx) (car record)) true false)
				(car (nth record 4))))))))

(define sql_queryplan_input_collector_active? (lambda (planning)
	(if (nil? planning) false
		(not (nil? (planning "__memcp_queryplan_guard_conditions"))))))

(define sql_queryplan_input_collector (lambda (planning)
	(if (not (sql_queryplan_input_collector_active? planning)) (list true '() true)
		(begin
			(define guard (sql_queryplan_guard_from_session planning))
			(list guard (sql_queryplan_preparations_from_session planning)
				(coalesceNil (planning "__memcp_queryplan_source_eligibility") true))))))
(define sql_queryplan_take_input_collector (lambda (planning)
	(begin
		(define result (sql_queryplan_input_collector planning))
		(if (sql_queryplan_input_collector_active? planning) (sql_queryplan_initialize_collector planning) true)
		result)))
(define sql_queryplan_register_input_alternatives (lambda (planning caller_lifetime common public caller selected_profile)
	(begin
		(define records (planning "__memcp_queryplan_input_alternatives"))
		(define index (coalesceNil (records "count") 0))
		/* A whole-query grant owns one complete cohort. Refuse independent
		mixed regions before publishing any executable variant. Nested result
		consumers inherit the existing catalog, not another grant. */
		(if (and (> index 0)
			(or (not (nil? selected_profile))
				(not (nil? (find (sql_queryplan_input_alternatives planning)
					sql_queryplan_input_record_dispatched? nil)))))
			(error "query plan requires one complete read dispatch region") true)
		(records (concat "program:" index) (list caller_lifetime common public caller selected_profile))
		(records "count" (+ index 1)))))
(define sql_queryplan_input_alternatives (lambda (planning)
	(begin
		(define records (planning "__memcp_queryplan_input_alternatives"))
		(if (nil? records) '()
			(map (produceN (coalesceNil (records "count") 0))
				(lambda (index) (records (concat "program:" index))))))))
(define sql_queryplan_input_guard (lambda (records ordinary eligibility)
	(begin
		(define conditions (append (merge (map records (lambda (record) (begin
			(define caller (nth (nth record 3) (if eligibility 2 0)))
			(define public (nth (nth record 2) (if eligibility 2 0)))
			(list (nth (nth record 1) (if eligibility 2 0))
				(if (sql_queryplan_input_record_dispatched? record)
					(if (car (nth record 4)) caller public)
					(if (expression_equal? caller public) public
						(list 'if (list 'car '__physical_input_context) caller public)))))))) ordinary))
		/* Re-evaluate the native profile before declaration/header/binding
		producers, independently of the context passed to the selected program. */
		(sql_queryplan_conjoin_guards (cons (sql_queryplan_input_admission records)
			(filter conditions (lambda (condition) (not (expression_equal? condition true)))))))))
/* Keep each original observation ID. Only the chosen producer/metric runs;
request-local native observations retain their existing once semantics. */
(define sql_queryplan_input_preparations (lambda (records ordinary)
	(merge (list
		(merge (map records (lambda (record) (begin
			(define public (nth (nth record 2) 1))
			(define caller (nth (nth record 3) 1))
			(if (sql_queryplan_input_record_dispatched? record)
				(merge (list (nth (nth record 1) 1) (if (car (nth record 4)) caller public)))
				(begin
					(define ids (merge_unique (list (map public car) (map caller car))))
					(merge (list (nth (nth record 1) 1)
						(map ids (lambda (id) (begin
							(define caller_input (find caller (lambda (item) (equal? (car item) id)) nil))
							(define public_input (find public (lambda (item) (equal? (car item) id)) nil))
							(if (expression_equal? caller_input public_input) public_input
								(list id (list 'physical-input-preparation caller_input public_input) nil)))))))))))))
		ordinary))))
(define sql_queryplan_input_profiled? (lambda (variant)
	(qassoc_get (nth variant 3) 'physical_input_profile false)))
/* The public compiled callable still has four request arguments. Guarded entry
execution borrows its explicit fifth-input kernel instead; the direct callable
owns one complete invocation lifetime without retaining runtime objects. */
(define sql_queryplan_wrap_input_formula (lambda (compile_tx procedure caller_lifetime)
	(sql_queryplan_compile_formula compile_tx
		(list 'call_with_invocation_group_lifetime
			(list 'lambda (list '__physical_input_scopes '__physical_input_initializers '__physical_input_observations)
				(list 'apply procedure (list 'list 'session 'tx 'resultrow 'resultfields
					(list 'list (list 'or (list 'tx_requires_query_local_cache 'tx) caller_lifetime)
						'__physical_input_scopes '__physical_input_initializers '__physical_input_observations))))))))
(define sql_queryplan_program_kernel (lambda (variant)
	(if (sql_queryplan_input_profiled? variant)
		(qassoc_get (nth variant 3) 'input_program nil) (cadr variant))))
(define sql_queryplan_guard_kernel (lambda (variant)
	(if (sql_queryplan_input_profiled? variant)
		(qassoc_get (nth variant 3) 'input_guard true) (car variant))))
(define sql_queryplan_eligibility_kernel (lambda (variant)
	(qassoc_get (nth variant 3)
		(if (sql_queryplan_input_profiled? variant) 'input_source_eligibility 'source_eligibility) true)))

(define sql_queryplan_input_context (lambda (variant tx lifetime)
	(if (not (sql_queryplan_input_profiled? variant)) nil
		(list (or (tx_requires_query_local_cache tx)
			(qassoc_get (nth variant 3) 'caller_lifetime false)) (car lifetime) (cadr lifetime) (nth lifetime 2)))))
/* A guard/preparation/program receives the same actual profile and the existing
query-owned registry suppliers. This adds no read permission or native Tx. */
(define sql_queryplan_apply_input (lambda (kernel_fn variant session tx resultrow resultfields context)
	(if (sql_queryplan_input_profiled? variant)
		(kernel_fn session tx resultrow resultfields context)
		(kernel_fn session tx resultrow resultfields))))

(define sql_queryplan_preparations_from_session (lambda (planning_session)
	(begin
		(define preparations (planning_session "__memcp_queryplan_preparations"))
		(if (nil? preparations) '()
			(map (produceN (coalesceNil (preparations "count") 0))
				(lambda (idx) (preparations (concat "preparation:" idx))))))))

(define sql_queryplan_bind_execution_session (lambda (expr)
	(match expr
		((symbol quote) _value) expr
		((symbol context) "session") (quote session)
		((quote context) "session") (quote session)
		(cons (symbol session) tail) (cons (quote session)
			(map tail sql_queryplan_bind_execution_session))
		(cons (quote session) tail) (cons (quote session)
			(map tail sql_queryplan_bind_execution_session))
		(cons (symbol session_globalvar) tail) (cons session_globalvar_explicit
			(cons (quote session) (map tail sql_queryplan_bind_execution_session)))
		(cons (quote session_globalvar) tail) (cons session_globalvar_explicit
			(cons (quote session) (map tail sql_queryplan_bind_execution_session)))
		(cons (symbol connection_id) _tail) (list (quote tx_connection_id) (quote tx))
		(cons (quote connection_id) _tail) (list (quote tx_connection_id) (quote tx))
		(cons head tail) (if (expression_equal? head session_globalvar)
			(cons session_globalvar_explicit
				(cons (quote session) (map tail sql_queryplan_bind_execution_session)))
			(if (expression_equal? head connection_id)
				(list (quote tx_connection_id) (quote tx))
				(if (and (expression_equal? head context)
					(and (equal? (count tail) 1) (equal? (car tail) "session")))
					(quote session)
					(cons
						(sql_queryplan_bind_execution_session head)
						(map tail sql_queryplan_bind_execution_session)))))
		_ expr)))

/* Legacy named SHOW reports missing ordinary relations. Resolving through
native transaction handles must preserve that contract instead of feeding a
nullable handle to SHOW. Unpublished helper names keep their empty description. */
(define sql_show_table_columns (lambda (tx database name) (begin
	(define handle (table database false name false tx))
	(if handle (show handle)
		(and (strlike name ".%") (not (nil? (show database)))) '()
		(error "table " database "." name " does not exist")))))

/* One request-expression visitor owns primitive binding. With a search
catalog it also hoists programs; nil retains the standalone binding contract.
A legacy dynamic SHOW keeps its operand-binding boundary while search programs
inside it may still be prepared. Program arguments bind before key creation,
without hoisting nested programs out of the enclosing program. */
(define sql_queryplan_rewrite_request (lambda (expr search_bindings bind_tx)
	(match expr
		((symbol quote) _value) expr
		((symbol show) db name)
		(if bind_tx
			(if (and (string? db) (string? name))
				(list (quote sql_show_table_columns) (quote tx) db name)
				(if search_bindings (sql_queryplan_rewrite_request expr search_bindings false) expr))
			(cons (car expr) (map (cdr expr) (lambda (part)
				(sql_queryplan_rewrite_request part search_bindings false)))))
		((symbol show) db name "constraints")
		(if bind_tx
			(list (quote show) (list (quote table)
				(sql_queryplan_rewrite_request db search_bindings true) false
				(sql_queryplan_rewrite_request name search_bindings true) false (quote tx)) "constraints")
			(cons (car expr) (map (cdr expr) (lambda (part)
				(sql_queryplan_rewrite_request part search_bindings false)))))
		(cons (symbol connection_id) _tail)
		(if bind_tx (list (quote tx_connection_id) (quote tx))
			(cons (car expr) (map (cdr expr) (lambda (part)
				(sql_queryplan_rewrite_request part search_bindings false)))))
		(cons (quote connection_id) _tail)
		(if bind_tx (list (quote tx_connection_id) (quote tx))
			(cons (car expr) (map (cdr expr) (lambda (part)
				(sql_queryplan_rewrite_request part search_bindings false)))))
		/* Callable recognition is syntactic. Scalar equality can coerce boolean
		heads or render whole lambda/data trees before comparing unlike tags. */
		(cons head tail) (if (and bind_tx (expression_equal? head connection_id))
			(list (quote tx_connection_id) (quote tx))
			(and search_bindings (expression_equal? head sql_text_search_prepare)) (begin
				(define args (if bind_tx (map tail (lambda (part)
					(sql_queryplan_rewrite_request part nil true))) tail))
				(define key (json_encode args))
				(define entry (search_bindings key))
				(if entry (car entry) (begin
					(define name (symbol (concat "search_program_" (sha256 key))))
					(search_bindings key (list name (cons head args)))
					name)))
			/* A callable head owns its lexical parameters. Bind its primitives,
			but keep its search programs inside that callable's scope. */
			(cons (sql_queryplan_rewrite_request head nil bind_tx)
				(map tail (lambda (part) (sql_queryplan_rewrite_request part search_bindings bind_tx)))))
		_ expr)))

/* Exact DML binds request primitives without the SELECT session rewrite. */
(define sql_queryplan_bind_tx_calls (lambda (expr)
	(sql_queryplan_rewrite_request expr nil true)))

(define sql_queryplan_uncovered_binding_conditions (lambda (planning_session)
	(begin
		(define covered (planning_session "__memcp_queryplan_guarded_session_keys"))
		(define structural (planning_session "__memcp_queryplan_guard_condition_catalog"))
		(define observed (planning_session "__memcp_queryplan_observed_session_keys"))
		(define compile_bindings (planning_session "__memcp_queryplan_compile_bindings"))
		(map (filter (observed) (lambda (key)
			(not (covered (if (nil? structural)
				(string (list (quote session) key))
				(list (quote session) key))))))
			(lambda (key)
				(begin
					(define value (compile_bindings key))
					(list (quote equal?)
						(list (quote session) key)
						(if (list? value) (list (quote quote) value) value))))))))

/* Guards execute outside the optimized query lambda. Rewrite the query AST's
session pseudo-call to an explicit context lookup so raw eval observes the
current request bindings. Quoted planner/catalog payloads remain data. */
(define sql_queryplan_runtime_guard_expr (lambda (expr observed_metrics)
	(match expr
		((symbol quote) _value) expr
		((symbol session) key) (if (match key
			(regex "^__memcp_queryplan_observation_(?:value|metric)_" _) true
			_ false)
			(planner_queryplan_observation_current_read_expr key)
			(list (quote session) key))
		((quote session) key) (if (match key
			(regex "^__memcp_queryplan_observation_(?:value|metric)_" _) true
			_ false)
			(planner_queryplan_observation_current_read_expr key)
			(list (quote session) key))
		'(binding key) (if (and (not (nil? observed_metrics))
			(and (string? key) (and (strlike key "__memcp_queryplan_observation_metric_%")
				(and (equal? binding (planner_queryplan_observation_session_expr))
					(not (contains? observed_metrics key))))))
			/* Logical enumeration can mention a metric whose physical prefix was
			not selected. Its guard must use the fallback without creating a session. */
			nil
			(list (sql_queryplan_runtime_guard_expr binding observed_metrics)
				(sql_queryplan_runtime_guard_expr key observed_metrics)))
		(cons head tail) (cons
			(sql_queryplan_runtime_guard_expr head observed_metrics)
			(map tail (lambda (part) (sql_queryplan_runtime_guard_expr part observed_metrics))))
		_ expr)))

(define sql_queryplan_guard_references_symbol? (lambda (expr target)
	(match expr
		((symbol quote) _value) false
		(cons head tail) (or
			(sql_queryplan_guard_references_symbol? head target)
			(reduce tail (lambda (found item)
				(or found (sql_queryplan_guard_references_symbol? item target))) false))
		/* equal? admits truth-value coercion: true compares equal to a symbol.
		Liveness is syntactic, not SQL/SCM value equality. Without this type
		check a literal true keeps every discarded cost binding alive, executing
		unused estimator calls on each cached query. */
		_ (and (symbol? expr) (equal? expr target)))))

/* Avoid degenerate generated `and` forms while retaining normal short-circuit
semantics for multiple independent cache assumptions. */
(define sql_queryplan_conjoin_guards (lambda (guards)
	(match guards
		'() true
		(cons guard '()) guard
		_ (cons (quote and) guards))))

(define sql_queryplan_statistics_guard_from_session (lambda (planning_session)
	(begin
		(define dependency_session
			(planning_session "__memcp_queryplan_statistics_dependencies"))
		(define dependencies (if (nil? dependency_session) '() (map
			(produceN (coalesceNil (dependency_session "count") 0))
			(lambda (idx) (dependency_session (concat "dependency:" idx))))))
		/* EXPLAIN retains displayed estimates, including predicate-local feedback.
		Only metadata guards belong here; executable carrier preparations and cost
		choice guards must never run just to refresh diagnostic output. */
		(define feedback (planning_session "__memcp_queryplan_diagnostic_guards"))
		(define feedback_guards (if (nil? feedback) '()
			(map (feedback) (lambda (key) (feedback key)))))
		(define dependency_guards (map dependencies (lambda (dependency)
			(list (quote equal?)
				(list (quote table_planner_statistics) (car dependency))
				(list (quote quote) (cadr dependency))))))
		(sql_queryplan_conjoin_guards (merge (list dependency_guards feedback_guards))))))

/* Bindings are registered producer-first. Walk backwards from surviving
decision guards to retain their complete dependency closure, not unused cost
enumeration work. A guarded derived estimate covers its session inputs too:
adding a raw value equality would needlessly partition the cache per user. */
(define sql_queryplan_guard_live_bindings (lambda (guard bindings)
	(reduce (reverse bindings) (lambda (live binding)
		(if (or (sql_queryplan_guard_references_symbol? guard (car binding))
			(reduce live (lambda (needed consumer)
				(or needed (sql_queryplan_guard_references_symbol? (cadr consumer) (car binding)))) false))
			(cons binding live) live)) '())))

/* Keep an original leading condition before producers whose lexical values
it does not consume. Later conditions retain their order once a dependency
requires the binding scope; quoted data never creates a dependency. */
(define sql_queryplan_guard_leading_conditions (lambda (conditions bindings)
	(match conditions
		(cons condition rest)
		(if (reduce bindings (lambda (found binding)
			(or found (sql_queryplan_guard_references_symbol? condition (car binding)))) false)
			(list '() conditions)
			(match (sql_queryplan_guard_leading_conditions rest bindings) '(leading remaining)
				(list (cons condition leading) remaining)))
		_ (list '() '()))))

/* These already registered checks are necessary source facts, not cost
assumptions. Literal addresses/declarations and the executing tx are their
only inputs; quoted metadata never depends on a preparation or binding. */
(define sql_queryplan_source_eligibility_condition? (lambda (condition) (begin
	(define quoted (lambda (value)
		(and (list? value) (equal? (count value) 2)
			(expression_equal? (car value) 'quote))))
	(match condition
		'(op schema relation tx expected)
		(and (expression_equal? op 'sql_relation_declarations_match?)
			(string? schema) (string? relation) (expression_equal? tx 'tx) (quoted expected))
		'(op schema relation tx expected fields)
		(and (expression_equal? op 'sql_relation_local_declarations_match?)
			(string? schema) (string? relation) (expression_equal? tx 'tx)
			(quoted expected) (quoted fields))
		_ false))))

(define sql_queryplan_guard_from_session (lambda (planning_session)
	(begin
		(define condition_accumulator (planning_session "__memcp_queryplan_guard_conditions"))
		(define condition_catalog (planning_session "__memcp_queryplan_guard_condition_catalog"))
		(define conditions (if (nil? condition_catalog)
			(map (condition_accumulator) (lambda (key) (condition_accumulator key)))
			(map (produceN (coalesceNil (condition_accumulator "count") 0))
				(lambda (idx) (condition_accumulator (concat "condition:" idx))))))
		(define statistics_guard (sql_queryplan_statistics_guard_from_session planning_session))
		(define raw_guard (sql_queryplan_conjoin_guards
			(merge (list conditions (list statistics_guard)))))
		(define binding_session (planning_session "__memcp_queryplan_guard_bindings"))
		(define binding_catalog (planning_session "__memcp_queryplan_guard_binding_catalog"))
		(define all_bindings (if (nil? binding_catalog)
			(map (binding_session) (lambda (key) (binding_session key)))
			(map (produceN (coalesceNil (binding_session "count") 0))
				(lambda (idx) (binding_session (concat "binding:" idx))))))
		(planning_session "__memcp_queryplan_source_eligibility"
			(sql_queryplan_conjoin_guards (filter conditions (lambda (condition)
				(and (sql_queryplan_source_eligibility_condition? condition)
					(not (reduce all_bindings (lambda (dependent binding)
						(or dependent (sql_queryplan_guard_references_symbol? condition (car binding)))) false)))))))
		/* Cost enumeration may bind expressions which no surviving guard uses.
		Materializing those lambda arguments would retain the discarded planning
		work on every cached execution. */
		(define bindings (sql_queryplan_guard_live_bindings raw_guard all_bindings))
		(define covered (planning_session "__memcp_queryplan_guarded_session_keys"))
		(reduce bindings (lambda (_ binding)
			(reduce (nth binding 2) (lambda (_ expr)
				(covered (if (nil? condition_catalog) (string expr) expr) true)) nil)) nil)
		(define uncovered_conditions (sql_queryplan_uncovered_binding_conditions planning_session))
		(define complete_guard (sql_queryplan_conjoin_guards (cons raw_guard uncovered_conditions)))
		(define split (if (empty_list? bindings) (list '() conditions)
			(sql_queryplan_guard_leading_conditions conditions bindings)))
		(define leading (car split))
		(define scoped_guard (if (empty_list? leading) complete_guard
			(sql_queryplan_conjoin_guards (cons
				(sql_queryplan_conjoin_guards (merge (list (cadr split) (list statistics_guard))))
				uncovered_conditions))))
		/* Bind producers in dependency order inside one lexical scope. A lambda
		per binding duplicates the remaining guard through nested compilation;
		sequential definitions preserve dependencies without that nesting. The
		binding-free prefix must reject stale source declarations before any
		dependent estimate dereferences its retained generation address. */
		(define guard_body (if (empty_list? bindings) complete_guard
			(list (list (quote lambda) '()
				(cons (quote !begin) (merge (list
					(map bindings (lambda (binding)
						(list (quote define) (car binding) (cadr binding))))
					(list scoped_guard))))))))
		(sql_queryplan_runtime_guard_expr
			(if (empty_list? leading) guard_body
				(sql_queryplan_conjoin_guards (merge (list leading (list guard_body)))))
			(map (sql_queryplan_preparations_from_session planning_session)
				(lambda (preparation) (planner_queryplan_observation_metric_key (car preparation))))))))

/* A parser envelope is (parser [options]). The optional exact_guards flag
collects generic compile-time dependencies on an exact-cache miss. Guard-free
formulas retain the direct callable path; guarded formulas reuse the existing
variant dispatcher and recompilation. Other parser envelopes are unchanged. */
(define sql_invoke_parse_fn (lambda (parse_fn schema parse_query policy planning_session tx)
	(if (list? parse_fn)
		((car parse_fn) schema parse_query policy planning_session tx)
		(parse_fn schema parse_query policy))))

/* Query plans are closed procedures rather than code evaluated in whichever
execution environment happens to call them. Keep every request-local input in
the parameter list: this prevents a cached plan from retaining the compiling
request and makes the complete dispatcher eligible for native compilation.
The Proc retains its optimized Scheme body, so cachemap memory accounting and
serialization still see the complete plan. */
(define sql_queryplan_write_guard (lambda (formula access readonly schema parse_query parse_fn)
	(if (and (not readonly) (access "database-create"))
		(if (or (access "write") (not (equal? (count (access "database-create")) 1)))
			(error "mixed database creation statements are not supported")
			(list (quote if) (list (quote replication_slave))
				(list (quote sql_database_create_replica) (quote tx)
					(list (quote quote) (access "database-create")) (quote session) schema parse_query
					(if (list? parse_fn) (equal? (car parse_fn) parse_psql) false)
					(quote resultrow) (quote resultfields)) formula))
		(and (not readonly) (access "write")
			(not (match formula
				(cons head tail) (and (empty_list? tail)
					(or (equal? head shutdown) (equal? head (quote shutdown))))
				_ false)))
		(if (access "admin")
			(list (quote if) (list (quote replication_slave))
				(list (quote replication_admin) (quote session) schema parse_query
					(if (list? parse_fn) (equal? (car parse_fn) parse_psql) false)
					(quote resultrow) (quote resultfields))
				formula)
			(if (nil? (access "write_databases"))
				(error "write plan is missing its actual database targets")
				(cons (quote !begin) (merge
					(map ((access "write_databases")) (lambda (database)
						(list (quote replication_write) (quote tx) database)))
					(list formula)))))
		formula)))

/* Search programs depend on statement inputs, never on the scanned row. Bind
identical programs once outside row closures, including parameterized plans. */
(define sql_queryplan_prepare_search (lambda (formula) (begin
	(define bindings (newsession))
	(define rewritten (sql_queryplan_rewrite_request formula bindings true))
	(cons (quote !begin)
		(merge (list
			(map (bindings) (lambda (key)
				(list (quote define) (car (bindings key)) (cadr (bindings key)))))
			(list rewritten)))))))

/* This exact existing observation input is part of the physical emitted-code
contract. Only the caller nil-Tx arm borrows the explicit dispatch resource;
public nil-Tx and every original-T native observation keep their old owner. */
(define sql_queryplan_bind_input_observations (lambda (expression)
	(match expression
		((symbol quote) _datum) expression
		(cons head tail)
		(if (and (expression_equal? head 'session)
			(and (equal? (count tail) 5) (equal? (car tail) "get_or_compute_scoped")
				(equal? (nth tail 2) "__memcp_queryplan_observations"))
			(expression_equal? expression (planner_queryplan_observation_session_expr)))
			(list 'if (list 'and (list 'car '__physical_input_context) (list 'nil? 'tx))
				(list (list 'nth '__physical_input_context 3)) expression)
			(map expression sql_queryplan_bind_input_observations))
		_ expression)))

(define sql_queryplan_compile_formula (lambda (compile_tx formula)
	(begin
		(define program (match formula
			((symbol physical-query-program) inputs body) (list inputs body)
			_ (list '() formula)))
		(define raw_body (if (contains? (car program) '__physical_input_context)
			(sql_queryplan_bind_input_observations (cadr program)) (cadr program)))
		(define body (if (and (contains? (car program) '__physical_input_context)
			(physical_plan_uses_invocation_groups? raw_body))
			(list (list 'lambda (list '__invocation_group_scopes '__invocation_relation_initializers) raw_body)
				(list (list 'nth '__physical_input_context 1))
				(list (list 'nth '__physical_input_context 2))) raw_body))
		(jit (close_procedure (eval (optimize (list 'lambda
			(merge (list (list 'session 'tx 'resultrow 'resultfields) (car program)))
			(source "SQL Query" 1 1
				(publish_queryplan_callables (sql_queryplan_prepare_search body)
					(car program))))))) compile_tx))))

(define sql_compile_queryplan_variant (lambda (parse_fn schema parse_query policy source_session tx parameter_mode input_lifetime dispatch_owner)
	(begin
		(tx_check tx)
		(define compile_started (nanotime))
		(define planning_session (sql_queryplan_compile_session source_session tx input_lifetime
			dispatch_owner))
		/* The direct frontend later compares this exact immutable procedure with
		its own parser owner. Wrapping factories cannot inherit that grant. */
		((planning_session "__memcp_queryplan_input_alternatives") "read_parser"
			(if (list? parse_fn) (car parse_fn) nil))
		(planning_session "__memcp_queryplan_parameter_mode" parameter_mode)
		(define statistics_diagnostic (match (toUpper parse_query)
			(regex "^\\s*EXPLAIN\\b" _) true
			_ false))
		/* EXPLAIN reports metadata, not just a reusable execution decision. Keep
		its table-wide refresh dependency without executing relational guards. */
		(planning_session "__memcp_queryplan_diagnostic_statistics" statistics_diagnostic)
		(if statistics_diagnostic
			(planning_session "__memcp_queryplan_diagnostic_guards" (newsession)) nil)
		(define access (newsession))
		(define compile_policy (sql_compile_table_policy
			(sql_policy_spec_resolve policy source_session tx) tx access statistics_diagnostic))
		(define raw_plan (with_session planning_session (lambda ()
			(sql_invoke_parse_fn parse_fn schema parse_query compile_policy planning_session tx))))
		/* Parsing includes logical and physical planning. Never spend optimizer
		work or install a variant after its requesting context was cancelled. */
		(tx_check tx)
		(define input_records (sql_queryplan_input_alternatives planning_session))
		(define profiled (not (empty_list? input_records)))
		(define caller_lifetime (reduce input_records (lambda (caller record)
			(or caller (car record))) false))
		(define bind_program (lambda (body)
			(if profiled (list 'physical-query-program (list '__physical_input_context) body) body)))
		(define plan (sql_queryplan_compile_formula tx (bind_program
			(sql_queryplan_write_guard (sql_queryplan_bind_execution_session raw_plan) access statistics_diagnostic schema parse_query parse_fn))))
		(tx_check tx)
		/* Diagnostics must refresh their captured estimates when table statistics
		change, but checking their cache must never execute candidate preparations. */
		(define guard (if statistics_diagnostic
			(sql_queryplan_statistics_guard_from_session planning_session)
			(sql_queryplan_guard_from_session planning_session)))
		(define guard (if (or statistics_diagnostic (not profiled)) guard
			(sql_queryplan_input_guard input_records guard false)))
		(define parameter_reads (merge_unique (list
			(query_expr_session_reads raw_plan) (query_expr_session_reads guard))))
		/* A variant publishes one compiled guard owner. Its immutable literal
		metadata stays in that procedure; registry and dispatch reuse the callable
		instead of retaining and repeatedly cloning an optimizer blueprint. */
		(define guard (if (expression_equal? guard true) true
			(sql_queryplan_compile_formula tx (bind_program (sql_queryplan_bind_execution_session guard)))))
		(define preparations (if statistics_diagnostic '()
			(sql_queryplan_preparations_from_session planning_session)))
		(define preparations (if (or statistics_diagnostic (not profiled)) preparations
			(sql_queryplan_input_preparations input_records preparations)))
		(define source_eligibility (if (empty_list? preparations) true
			(coalesceNil (planning_session "__memcp_queryplan_source_eligibility") true)))
		(define source_eligibility (if (or statistics_diagnostic (not profiled)) source_eligibility
			(if (empty_list? preparations) (sql_queryplan_input_admission input_records)
				(sql_queryplan_input_guard input_records source_eligibility true))))
		(define source_eligibility (if (expression_equal? source_eligibility true) true
			(sql_queryplan_compile_formula tx (bind_program (sql_queryplan_bind_execution_session source_eligibility)))))
		(tx_check tx)
		(define input_program plan) (define input_guard guard) (define input_eligibility source_eligibility)
		(define plan (if profiled (sql_queryplan_wrap_input_formula tx plan caller_lifetime) plan))
		(define guard (if (and profiled (not (expression_equal? guard true)))
			(sql_queryplan_wrap_input_formula tx guard caller_lifetime) guard))
		(define source_eligibility (if (and profiled (not (expression_equal? source_eligibility true)))
			(sql_queryplan_wrap_input_formula tx source_eligibility caller_lifetime) source_eligibility))
		(list guard plan preparations
			(merge (list
				(if profiled (list (list 'physical_input_profile true)
					(list 'input_program input_program) (list 'input_guard input_guard)
					(list 'input_source_eligibility input_eligibility)
					(list 'caller_lifetime caller_lifetime)) '())
				(list
					(list (quote parameter_mode) parameter_mode)
					(list (quote compile_ns) (- (nanotime) compile_started))
					(list (quote execution_cost) (planning_session "__memcp_queryplan_candidate_cost"))
					(list (quote work_profile) (planning_session "__memcp_queryplan_parameter_work"))
					(list (quote parameters) parameter_reads)
					(list (quote source_eligibility) source_eligibility))))))))

(define sql_queryplan_preparation_expr (lambda (preparation)
	(match preparation '(decision_id producer metric_expr)
		(if (match producer ((symbol physical-input-preparation) _caller _public) true _ false)
			(list 'if (list 'car '__physical_input_context)
				(if (nil? (cadr producer)) true (sql_queryplan_preparation_expr (cadr producer)))
				(if (nil? (nth producer 2)) true (sql_queryplan_preparation_expr (nth producer 2))))
			(begin
				(define value_key (planner_queryplan_observation_value_key decision_id))
				(define metric_key (planner_queryplan_observation_metric_key decision_id))
				(define observation_session (symbol "__queryplan_observation_session"))
				(define prepared_value (symbol "__queryplan_prepared_value"))
				(list
					(list (quote lambda) (list observation_session)
						(list (quote if)
							(list (quote number?) (list observation_session metric_key))
							true
							(list (quote !begin)
								(list (quote tx_check) (quote tx))
								(list
									(list (quote lambda) (list prepared_value)
										(list (quote !begin)
											(list observation_session value_key prepared_value)
											(list observation_session metric_key
												(list
													(list (quote lambda) (list (symbol "__queryplan_observed_value")) metric_expr)
													prepared_value))
											true))
									(sql_queryplan_runtime_guard_expr producer nil))
								(list (quote tx_check) (quote tx))
								true)))
					(planner_queryplan_observation_session_expr)))))))

/* Walk variants in priority order. A cheap guarded variant can return before
an older expensive observation is prepared. Once a variant needs an
observation, prepare each physical decision at most once and then evaluate its
otherwise side-effect-free guard. */
(define sql_queryplan_variant_call (lambda (variant context)
	(if (sql_parameter_generic? variant)
		(list 'sql_parameter_execute (list 'quote (nth variant 3))
			(if (sql_queryplan_input_profiled? variant)
				(list 'lambda (list 'session 'tx 'resultrow 'resultfields)
					(list (sql_queryplan_program_kernel variant) 'session 'tx 'resultrow 'resultfields context)) (sql_queryplan_program_kernel variant))
			'session 'tx 'resultrow 'resultfields)
		(if (sql_queryplan_input_profiled? variant)
			(list (sql_queryplan_program_kernel variant) 'session 'tx 'resultrow 'resultfields context)
			(list (sql_queryplan_program_kernel variant) 'session 'tx 'resultrow 'resultfields)))))

(define sql_queryplan_formula_dispatch (lambda (variants prepared_catalog miss_expr)
	(match variants
		(cons variant rest) (begin
			(define profiled (sql_queryplan_input_profiled? variant))
			(define source_eligibility (sql_queryplan_eligibility_kernel variant))
			(define new_preparations (filter (nth variant 2) (lambda (preparation)
				(if (prepared_catalog (car preparation)) false
					(begin (if (expression_equal? source_eligibility true)
						(prepared_catalog (car preparation) true) true) true)))))
			(define guard (sql_queryplan_guard_kernel variant))
			(define full_guard (if (expression_equal? guard true) true
				(if profiled (list 'apply guard (list 'list 'session 'tx 'resultrow 'resultfields '__physical_input_context))
					(list 'apply guard (list 'list 'session 'tx 'resultrow 'resultfields)))))
			(define prepared_guard (if (empty_list? new_preparations) full_guard
				(cons '!begin (merge (list (map new_preparations sql_queryplan_preparation_expr) (list full_guard))))))
			(define eligible_guard (if (expression_equal? source_eligibility true) prepared_guard
				(list 'and (if profiled
					(list 'apply source_eligibility (list 'list 'session 'tx 'resultrow 'resultfields '__physical_input_context))
					(list 'apply source_eligibility (list 'list 'session 'tx 'resultrow 'resultfields))) prepared_guard)))
			(define tail (sql_queryplan_formula_dispatch rest prepared_catalog miss_expr))
			(define selected (sql_queryplan_variant_call variant '__physical_input_context))
			(define raw_dispatch (if (expression_equal? eligible_guard true) selected
				(list 'if eligible_guard selected tail)))
			(define dispatched (if profiled (sql_queryplan_bind_input_observations raw_dispatch) raw_dispatch))
			(if profiled
				(list (list 'lambda (list '__physical_input_context)
					(if (physical_plan_uses_invocation_groups? dispatched)
						(list (list 'lambda (list '__invocation_group_scopes '__invocation_relation_initializers) dispatched)
							(list (list 'nth '__physical_input_context 1))
							(list (list 'nth '__physical_input_context 2))) dispatched))
					(list 'list (list 'or (list 'tx_requires_query_local_cache 'tx)
						(qassoc_get (nth variant 3) 'caller_lifetime false))
						'__physical_input_scopes '__physical_input_initializers '__physical_input_observations))
				dispatched))
		_ miss_expr)))

(define sql_queryplan_miss_expr (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy lifetime owner_context)
	/* The miss arm invokes the Scheme compiler, not query work. Keep that cold
	boundary behind apply so optimizing the hot dispatcher cannot specialize and
	re-optimize the compiler procedure against cache/session literal arguments. */
	(list (quote apply)
		(quote sql_queryplan_variant_miss)
		(cons (quote list) (list
			queryplan_cache cache_key entry
			(if (list? parse_fn) (list (quote quote) parse_fn) parse_fn)
			schema parse_query
			(if (list? policy) (list (quote quote) policy) policy)
			(quote session) (quote tx) (quote resultrow) (quote resultfields) lifetime
			(if (nil? owner_context) nil (list (quote quote) owner_context)))))))

/* Reuse the frontend's existing native map placement rule. The returned
bounded tuple keeps the map and its compile-owner decision from ONE native
predicate read; no private Tx is captured in shared routing metadata. */
(define sql_queryplan_cache_selection (lambda (owner_context tx)
	(begin
		(define query_local (tx_requires_query_local_cache tx))
		(list (if query_local
			(tx_get_or_compute_resource tx (concat "sql-query-plans:" (nth owner_context 2))
				(lambda () (newcachemap)))
			(cadr owner_context)) query_local))))

/* Called only after the actual invocation selects a different existing map.
No old-entry guards, preparers, locks or metadata have run at this boundary. */
(define sql_queryplan_execute_owned_cache (lambda (owner_context cache_key parse_fn schema parse_query policy session tx resultrow resultfields lifetime)
	(begin
		(tx_check tx)
		(define selected (sql_queryplan_cache_selection owner_context tx))
		(define target_cache (car selected))
		(define compile_tx (if (cadr selected) tx nil))
		(define existing (target_cache cache_key))
		(define cached (if (nil? existing)
			(target_cache cache_key tx (lambda ()
				(sql_queryplan_new_entry target_cache cache_key parse_fn schema parse_query
					policy session compile_tx nil owner_context lifetime))) existing))
		(define formula (if (list? cached) (cadr cached) cached))
		(if (and (not (nil? lifetime)) (list? cached))
			/* This is still the same bounded invocation. Its existing observation
			and group suppliers survive the owner delegation, never shared code. */
			(sql_queryplan_variant_miss target_cache cache_key (car cached) parse_fn schema parse_query policy
				session tx resultrow resultfields lifetime owner_context)
			(sql_execute_formula session tx formula resultrow resultfields)))))

(define sql_queryplan_formula (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy variants)
	(begin
		(define owner_context (entry "cache_owner"))
		(define profiled (not (nil? (find variants sql_queryplan_input_profiled? nil))))
		(define miss (sql_queryplan_miss_expr queryplan_cache cache_key entry parse_fn schema parse_query policy
			(if profiled (list 'list '__physical_input_scopes '__physical_input_initializers '__physical_input_observations) nil)
			owner_context))
		(define dispatch (if profiled
			(list 'call_with_invocation_group_lifetime
				(list 'lambda (list '__physical_input_scopes '__physical_input_initializers '__physical_input_observations)
					(sql_queryplan_formula_dispatch variants (make_structural_catalog 'ast) miss)))
			(sql_queryplan_formula_dispatch variants (make_structural_catalog 'ast) miss)))
		(if (nil? owner_context) dispatch
			(list 'if
				(list 'identical? (list 'car (list 'sql_queryplan_cache_selection (list 'quote owner_context) 'tx)) queryplan_cache)
				dispatch
				/* Keep compiler/owner routing behind the existing cold APPLY seam. */
				(list 'apply 'sql_queryplan_execute_owned_cache
					(cons 'list (list (list 'quote owner_context) cache_key
						(if (list? parse_fn) (list 'quote parse_fn) parse_fn)
						schema parse_query (if (list? policy) (list 'quote policy) policy)
						'session 'tx 'resultrow 'resultfields nil))))))))

(define sql_queryplan_install_variants (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy variants tx)
	(begin
		(define recent_variants (sql_queryplan_recent_variants variants))
		/* Published variants contain immutable compiled callables. Dispatcher
		construction owns only its small call containers, never guard syntax. */
		(entry "variants" recent_variants)
		(entry "formula" (sql_queryplan_compile_formula tx
			(sql_queryplan_formula queryplan_cache cache_key entry parse_fn schema parse_query policy recent_variants)))
		(entry "formula"))))

(define sql_queryplan_new_entry (lambda (queryplan_cache cache_key parse_fn schema parse_query policy source_session tx initial_variant owner_context input_lifetime)
	(begin
		(define entry (newsession))
		(entry "compile_lock" (mutex))
		(entry "cache_owner" owner_context)
		(define formula (sql_queryplan_install_variants queryplan_cache cache_key entry parse_fn schema parse_query policy
			(list (if (nil? initial_variant) (sql_compile_queryplan_variant parse_fn schema parse_query policy source_session tx "specialized" input_lifetime owner_context) initial_variant)) tx))
		(list entry formula))))

(define sql_queryplan_matching_variant (lambda (variants session tx resultrow resultfields lifetime)
	(match variants
		(cons variant rest) (begin
			(define context (sql_queryplan_input_context variant tx lifetime))
			(define source_eligibility (sql_queryplan_eligibility_kernel variant))
			(define guard (sql_queryplan_guard_kernel variant))
			(if (and (if (expression_equal? source_eligibility true) true
				(sql_queryplan_apply_input source_eligibility variant session tx resultrow resultfields context))
				(begin
					(map (nth variant 2) (lambda (preparation) (begin
						(define expression (sql_queryplan_preparation_expr preparation))
						(define compiled (sql_queryplan_compile_formula tx
							(if (sql_queryplan_input_profiled? variant)
								(list 'physical-query-program (list '__physical_input_context) expression) expression)))
						(sql_queryplan_apply_input compiled variant session tx resultrow resultfields context))))
					(if (expression_equal? guard true) true
						(sql_queryplan_apply_input guard variant session tx resultrow resultfields context))))
				(list variant context)
				(sql_queryplan_matching_variant rest session tx resultrow resultfields lifetime)))
		_ nil)))

/* Called only by the final else branch of a cached plan. Recheck after taking
the entry lock because another request may already have installed a matching
variant. New variants are prepended so the most recent statistics regime wins
the common guard-dispatch path. */
(define sql_queryplan_variant_miss (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy session tx resultrow resultfields lifetime owner_context)
	(begin
		(tx_check tx)
		(define selection (if (nil? owner_context) (list queryplan_cache true)
			(sql_queryplan_cache_selection owner_context tx)))
		(if (not (identical? (car selection) queryplan_cache))
			(sql_queryplan_execute_owned_cache owner_context cache_key parse_fn schema parse_query policy
				session tx resultrow resultfields lifetime)
			(begin
				(define compile_tx (if (or (nil? owner_context) (cadr selection)) tx nil))
				(define execute (lambda (scopes initializers observations) (begin
					(define lifetime (list scopes initializers observations))
					(define selected ((entry "compile_lock") tx (lambda ()
						(if (and (not (nil? owner_context))
							(not (identical? (car (sql_queryplan_cache_selection owner_context tx)) queryplan_cache)))
							/* Empty functional result delegates ONLY after unlocking. */
							(list)
							(begin
								(define current_variants (entry "variants"))
								(define matching (sql_queryplan_matching_variant current_variants session tx resultrow resultfields lifetime))
								(if matching (list matching false) (begin
									(define specialized (sql_compile_queryplan_variant parse_fn schema parse_query policy session compile_tx
										"specialized" lifetime owner_context))
									(define variant (sql_parameter_choose_candidate specialized entry current_variants
										parse_fn schema parse_query policy session tx lifetime compile_tx))
									(define formula (sql_queryplan_install_variants queryplan_cache cache_key entry parse_fn schema parse_query policy
										(cons variant current_variants) compile_tx))
									(queryplan_cache cache_key (list entry formula))
									(list variant true))))))))
					/* A cold compile/locked recheck may outlive a native lock-profile
					change. Keep the compiled variant in its original selected map;
					admit the CURRENT owner again before inspecting/using that code. */
					(if (or (empty_list? selected)
						(and (not (nil? owner_context))
							(not (identical? (car (sql_queryplan_cache_selection owner_context tx)) queryplan_cache))))
						(sql_queryplan_execute_owned_cache owner_context cache_key parse_fn schema parse_query policy
							session tx resultrow resultfields lifetime)
						(begin
							(define admitted (if (cadr selected)
								(sql_queryplan_matching_variant (list (car selected)) session tx resultrow resultfields lifetime)
								(car selected)))
							/* A compile is not execution admission. A fresh false assumption
							refuses before effects after this one compile; the retained-variant
							cap is not a bound on recursive compilation attempts. */
							(if (nil? admitted)
								(error "query plan assumptions changed during compilation")
								(begin
									(define variant (car admitted)) (define context (cadr admitted))
									(if (sql_parameter_generic? variant)
										(sql_parameter_execute (nth variant 3)
											(if (sql_queryplan_input_profiled? variant)
												(lambda (session tx resultrow resultfields)
													((sql_queryplan_program_kernel variant) session tx resultrow resultfields context)) (cadr variant))
											session tx resultrow resultfields)
										(sql_queryplan_apply_input (sql_queryplan_program_kernel variant) variant session tx resultrow resultfields context)))))))))
				(if (nil? lifetime) (call_with_invocation_group_lifetime execute)
					(execute (car lifetime) (cadr lifetime) (nth lifetime 2))))))))

/* cached_parse: wraps SELECT planning with a lazy polymorphic Scheme plan
cache. DDL, DML and transaction-control formulas retain the original exact
cache path because their AST may intentionally operate on session state.
cache_key = principal:schema:view-generation:hash(query-shape), retaining policy
isolation while sharing safe SELECT plans across literal variants. Each entry
is a compiled callable whose dispatcher contains guarded specialized plans and
one compile-miss arm.
On parse error the result is not cached. */
/* Frontends defer the system.user lookup until a cache producer actually
runs. A warm plan already owns its compiled policy and must not rescan the
user table merely to discard a newly constructed policy closure. */
/* Deferred policies are compile inputs, not cached account/session decisions.
Resolve them from the executing facts only at an actual compile, before the
existing access collector. Entries retain the unresolved immutable producer. */
(define sql_policy_spec_resolve (lambda (policy_spec source_session tx)
	(if (and (list? policy_spec)
		(equal? (car policy_spec) (quote sql-policy-for)))
		(sql_policy (if (equal? (count policy_spec) 3)
			(list (cadr policy_spec) (nth policy_spec 2)) (cadr policy_spec)))
		(and (list? policy_spec)
			(equal? (car policy_spec) (quote sql-compile-policy)))
		((cadr policy_spec) source_session tx)
		policy_spec)))

(define sql_queryplan_cache_key (lambda (username schema query shape_hash)
	(concat username ":" schema ":" (sql_view_query_generation query) ":" shape_hash)))

(define cached_parse (lambda (queryplan_cache cache_domain parse_fn schema query policy username session parameterize_literals tx)
	(begin
		/* Only built-in authenticated policy specs have a native account owner.
		Custom trusted policies retain their existing independent cache contract. */
		(define builtin_policy (and (list? policy) (equal? (car policy) (quote sql-policy-for))))
		(define actor (if builtin_policy (sql_session_actor session (cadr policy)
			(if (equal? (count policy) 3) (nth policy 2) nil)) nil))
		(if (and (not (nil? actor)) (not (equal? username (car actor))))
			(error "access denied: authenticated login identity changed") true)
		(define policy (if builtin_policy
			(list (quote sql-policy-for) (car actor) (cadr actor)) policy))
		(define cache_principal (if actor (json_encode actor) username))
		(define owner_context (list 'physical-dispatch-owner queryplan_cache cache_domain))
		(define cache_selection (sql_queryplan_cache_selection owner_context tx))
		(define query_local (cadr cache_selection))
		(define queryplan_cache (car cache_selection))
		(define explain_query (match (toUpper query)
			(regex "^\\s*EXPLAIN\\b" _) true
			_ false))
		(define cached_shape (if (and parameterize_literals (not explain_query)) (sql_literal_shape_cache query) nil))
		(define parameterized (if cached_shape cached_shape
			(begin
				(define exact_shape (list query '() (fnv_hash query)))
				/* An exact-only statement already occupies its query-cache entry.
				Reuse that lexical decision without retaining a second cache entry or
				repeating SQL tokenization. Normal policy and plan guards still run. */
				(if (or explain_query (not parameterize_literals)
					(queryplan_cache (sql_queryplan_cache_key cache_principal schema query (nth exact_shape 2))))
					exact_shape (sql_parameterize_select_literals query true)))))
		(match parameterized '(parse_query bindings shape_hash) (begin
			(define cache_key (sql_queryplan_cache_key cache_principal schema parse_query shape_hash))
			(define compile_diagnostic (and explain_query (match (toUpper parse_query)
				(regex "^\\s*EXPLAIN\\s+COMPILE\\b" _) true
				_ false)))
			/* Cachemap factories take no arguments. Only a transaction-owned cache
			may compile against private declarations; shared misses use independent
			authority and cannot inherit their first waiter's transaction. Diagnostics
			are uncached and retain the requesting transaction explicitly. */
			(define compile_tx (if (or query_local compile_diagnostic) tx nil))
			(if (not (equal? bindings '()))
				(reduce (produceN (count bindings)) (lambda (_ idx)
					(session (concat "v" (string (+ idx 1))) (nth bindings idx))) nil)
				nil)
			/* Compile diagnostics measure true misses and must not turn their own
			previous result into a cache hit. The inspected query is never run. */
			/* A hit needs only the retained dispatcher. Construct miss producers
			after this lookup; the getter still serializes concurrent misses. */
			(define cached_entry (if compile_diagnostic nil (queryplan_cache cache_key)))
			(define formula (if (not (nil? cached_entry))
				/* Guarded entries retain (entry dispatcher); exact entries are
				compiled procedures. Their dispatchers still evaluate every guard. */
				(if (list? cached_entry) (cadr cached_entry) cached_entry)
				(begin
					/* Only SELECT owns read-plan declaration guards. In particular,
					CREATE TRIGGER must leave inline DML guards inside its callback.
					Reuse the lexical token pass on this cold miss so leading comments
					cannot bypass wildcard declaration validation. */
					(define tokens (sql_parameter_tokens parse_query))
					(define first_token (sql_parameter_next_significant_index tokens 0))
					(define select_query (if (< first_token (count tokens))
						(equal?? (nth tokens first_token) "SELECT") false))
					/* Cost-sensitive queries retain their existing variant search.
					Simple SELECTs use the smaller exact variant path, including
					their semantic declaration guards. */
					(define guarded_query (or explain_query (and select_query
						(match (toUpper parse_query)
							(regex "\\b(?:LIKE|MATCH|JOIN|EXISTS|IN)\\b|\\(\\s*SELECT\\b" _) true
							_ false))))
					(if (or compile_diagnostic (not guarded_query))
						/* Select the producer before constructing its closure. The ordinary
						exact producer remains identical to its guard-free implementation;
						its cold optimizer must not compile an unused variant producer. */
						(if (or (and (list? parse_fn) (> (count parse_fn) 1) ((cadr parse_fn) "exact_guards"))
							/* Every SELECT may consume declared type/name/key contracts.
							The existing variant path collapses source-free constant
							queries with no dependencies to a direct compiled procedure. */
							select_query
							/* DML and ALTER can bind physical storage coordinates. Retain
							their emitted declaration guards when a visible name is reused;
							guard-free declarations still collapse to a direct procedure. */
							(if (< first_token (count tokens))
								(contains? (list "INSERT" "REPLACE" "UPDATE" "DELETE" "ALTER")
									(toUpper (nth tokens first_token))) false))
							(begin
								(define exact_compile (lambda () (begin
									(define variant (sql_compile_queryplan_variant parse_fn schema parse_query policy session compile_tx "specialized" nil owner_context))
									(if (and (expression_equal? (car variant) true) (empty_list? (nth variant 2)))
										(cadr variant)
										(sql_queryplan_new_entry queryplan_cache cache_key parse_fn schema parse_query policy session compile_tx variant owner_context nil)))))
								(define compiled (if compile_diagnostic (exact_compile)
									(queryplan_cache cache_key tx exact_compile)))
								(if (list? compiled) (cadr compiled) compiled))
							(begin
								/* Build the exact-cache compiler only for its own branch. Guarded
								cache hits must not repeatedly optimize an unused closure. */
								(define exact_compile (lambda ()
									(begin
										(define resolved_policy (sql_policy_spec_resolve policy session compile_tx))
										(define access (newsession))
										(define compile_policy (sql_compile_table_policy resolved_policy compile_tx access compile_diagnostic))
										(define raw_plan (sql_queryplan_bind_tx_calls
											(sql_queryplan_bind_execution_session
												(with_session session (lambda ()
													(sql_invoke_parse_fn parse_fn schema parse_query compile_policy session compile_tx))))))
										(sql_queryplan_compile_formula compile_tx (sql_queryplan_write_guard raw_plan access compile_diagnostic schema parse_query parse_fn)))))
								(if compile_diagnostic
									(exact_compile)
									(queryplan_cache cache_key tx exact_compile))))
						(begin
							(define cached_entry (queryplan_cache cache_key tx
								(lambda () (sql_queryplan_new_entry queryplan_cache cache_key parse_fn schema parse_query
									policy session compile_tx nil owner_context nil))))
							(cadr cached_entry))))))
			formula)))))

/* Frontends supply parsed recovery admission to the generic request executor. */
(define sql_with_statement (lambda (session state sequence text body)
	(with_query_request session state sequence text "cursor-stability" (lambda (statement)
		(statement text (sql_recovery_statement? text) body)))))

(define sql_execute_formula (lambda (session tx formula resultrow resultfields)
	(formula session tx resultrow resultfields)))

/* helper: build a policy function for table-level access checks
usage: create a policy by (set policy (sql_policy "username")),
then you can query the policy by
(policy "database" "tablename" false) for read
(policy "database" "tablename" true) for write
(policy "system" true true) to check for admin access like CREATE DATABASE, CREATE USER, DROP DATABASE, SHUTDOWN and so on
if everything is fine, the function call will do nothing.
if the user is not allowed to access this property, the function will throw an error and the query is aborted before it has run
*/
(define sql_private_relation? (lambda (name)
	(and (string? name) (regexp_test name "^\\.[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}#|^\\.##"))))
(define sql_policy (lambda (username)
	(begin
		(define actor (if (list? username) username nil))
		(define username (if actor (car actor) username))
		(define is_admin (scan_lookup nil (table "system" false "user" false nil)
			(list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "admin") (list username)))
		(define account_policy (if is_admin (lambda (schema tblname write) true) /* admin -> allow all */
			/* else: complicated policy */
			(lambda (schema tblname write)
				(begin
					/* Allow virtual INFORMATION_SCHEMA for all users */
					(if (equal?? schema "information_schema") true (begin
						/* Database-level check via system.access */
						(define has_access (scan_lookup nil (table "system" false "access" false nil)
							(list 371635467059200
								(scan_boundary "equal" "username" 0 0 true true "" false)
								(scan_boundary "equal" "database" 1 1 true true "" false)) (list username schema)))
						(if has_access true (error (concat "access denied: user '" username "' may not " (if write "write" "read") " " schema "." tblname)))
					))
			))
		))
		(lambda (schema tblname write)
			(if (sql_database_create_request? tblname)
				(sql_database_create_actor (if actor actor username) schema (cadr tblname))
				(sql_private_relation? tblname) (error "access denied to private SQL relation")
				(account_policy schema tblname write)))
	)
))

/* create user tables */
(print "Initializing SQL frontend")
(if (has? (show) "system") true (begin
	(print "creating database system")
	(createdatabase "system")
))
(if (has? (show "system") "user") true (begin
	(print "creating table system.user")
	(eval (parse_sql "system" "CREATE TABLE `user`(username text, password text, admin boolean DEFAULT FALSE) ENGINE=SAFE" (lambda (schema tblname write) true)))
	(insert (table "system" false "user" false nil) '("username" "password" "admin") '('("root" (password (arg "root-password" "admin")) true)))
))

/* migration: older instances may miss the admin column; add it and mark all existing users as admin */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "admin")
			true
			(begin
				(createcolumn (table "system" false "user" false nil) "admin" "boolean" '() '())
				(scan nil (table "system" false "user" false nil) '(369435906932736) '() '() (lambda () true) '("$update") (lambda (acc $update) (begin ($update '("admin" true)) acc)))
			)
		)
	) true)
)) (lambda (e) true))

/* migration: drop legacy id column that caused NOT NULL errors on CREATE USER */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "id")
			(migratedropcolumn (table "system" false "user" false nil) "id")
			true)
	) true)
)) (lambda (e) true))

/* migration: ensure root always has admin=true */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(scan nil (table "system" false "user" false nil) (list 369436175368192 (scan_boundary "equal" "username" 0 0 true true "" false)) '("root") '() (lambda () true) '("$update") (lambda (acc $update) (begin ($update '("admin" true)) acc)))
		true)
)) (lambda (e) true))

/* error query log table */
(if (not (has? (show "system_statistic") "errors")) (begin
	(print "creating table system_statistic.errors")
	(eval (parse_sql "system_statistic" "CREATE TABLE errors(datetime text, database text, user text, query text, error text) ENGINE=SLOPPY" (lambda (schema tblname write) true)))
))

/* global counter incremented on each logged error — used by dashboard WebSocket to trigger refresh */
(set error_log_counter (newsession))
(error_log_counter "count" 0)

/* error_log: insert a failed query into system_statistic.errors (no-op when ErrorQueryLog is off)
errmsg — error message (required)
db     — database name (pass "" when unknown)
usr    — username (pass "" when unknown)
qry    — query text (pass "" when unknown) */
(define error_log (lambda (errmsg db usr qry) (begin
	/* always print to stdout for system logs */
	(print (if (equal? db "") "" (concat "[" db "] ")) errmsg)
	/* always count errors regardless of ErrorQueryLog setting */
	(error_log_counter "count" (+ (error_log_counter "count") 1))
	(if (settings "ErrorQueryLog") (begin
		(try (lambda () (begin
			(insert (table "system_statistic" false "errors" false nil)
				'("datetime" "database" "user" "query" "error")
				(list (list (now) db usr qry (concat errmsg))))
			/* trimming moved to 15-minute cron in dashboard.scm */
		)) (lambda (e) true)) /* silently ignore logging errors to avoid infinite recursion */
	) true)
)))

/* print log table */
(if (not (has? (show "system_statistic") "logs")) (begin
	(print "creating table system_statistic.logs")
	(eval (parse_sql "system_statistic" "CREATE TABLE logs(datetime text, message text) ENGINE=SLOPPY" (lambda (schema tblname write) true)))
))

/* access control: which user can access which database */
(if (has? (show "system") "access") true (begin
	(print "creating table system.access")
	(eval (parse_sql "system" "CREATE TABLE `access`(username text, database text) ENGINE=SAFE" (lambda (schema tblname write) true)))
))

/* Logical SQL views. Both the original SELECT and neutral parser IR are kept;
the latter is expanded before logical planning and never materialized. */
(if (has? (show "system") "views") true (begin
	(print "creating table system.views")
	(eval (parse_sql "system" "CREATE TABLE `views`(`database` text, `name` text, `dialect` text, `sql` text, `ir` text) ENGINE=SAFE" (lambda (schema tblname write) true)))
))

(if (scan_exists nil (table "system" false "views" false nil)
	'(369435906932736) '() '() (lambda () true))
	(sql_view_catalog_note_presence) true)

/* Bootstrap owns these fixed catalog constraints. Do not suppress failures:
without them repeated grants create duplicate accounts and ambiguous authentication. */
(init_sql_catalog_keys)

(import "sql-accounts.scm")

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(define handle_query (lambda (req res schema query) (begin
		/* check for password */
		(define account (sql_authenticate_account (req "username") (req "password")))
		(if account
			(begin
				(try (lambda () (time (begin
					((res "header") "Content-Type" "text/event-stream; charset=utf-8")
					(define resultrow (res "jsonl"))
					(define session (sql_bind_http_session req account))
					(define session_state (req "__session_state"))
					(define query_seq (req "__query_seq"))
					(session "username" (account "username"))
					(session "schema" schema)
					/* Bind URL query params (v1=, v2=, ...) as prepared-statement args into the session
					before parse/build so session-sensitive planner rewrites see the right values. */
					(sql_http_query_parameters req session)
					(set resultrow_called false)
					(set original_resultrow resultrow)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(original_resultrow row))))
					(set query_result (sql_with_statement session session_state query_seq query
						(lambda (tx) (begin
							(define formula (cached_parse sql_queryplan_cache "sql" (list parse_sql) schema query
								(list (quote sql-policy-for) (req "username")) (req "username") session true tx))
							(sql_execute_formula session tx formula resultrow (lambda (_fields) true))))))
					/* If no resultrow was called and we got a number, return it as affected_rows */
					(if (and (not resultrow_called) (number? query_result)) (begin
						(original_resultrow '("affected_rows" query_result))
					))
				) query)) (lambda(e) (begin
						(error_log (concat e) schema (req "username") query)
						((res "header") "Content-Type" "text/plain")
						((res "status") 500)
						((res "print") "SQL Error: " e)
				)))
			)
			(begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized")
			)
		)
	)))
	(define handle_query_postgres (lambda (req res schema query) (begin
		/* check for password */
		(define account (sql_authenticate_account (req "username") (req "password")))
		(if account
			(begin
				(try (lambda () (time (begin
					((res "header") "Content-Type" "text/plain")
					(define resultrow (res "jsonl"))
					(define session (sql_bind_http_session req account))
					(define session_state (req "__session_state"))
					(define query_seq (req "__query_seq"))
					(session "username" (account "username"))
					(session "schema" schema)
					(set resultrow_called false)
					(set original_resultrow resultrow)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(original_resultrow row))))
					(define handled (match query
						(regex "SELECT\\s+c\\.relname\\s+as\\s+tblname\\s+FROM\\s+pg_catalog\\.pg_class" _)
						(begin
							(map (show schema) (lambda (tbl) (resultrow (list "tblname" tbl))))
							true)
						(regex "FROM\\s+pg_attribute" _)
						(match query
							(regex "c\\.relname\\s*=\\s*'([^']+)'" _ tbl)
							(begin
								(map (show schema tbl) (lambda (line) (resultrow line)))
								true)
							true)
						(regex "FROM\\s+pg_indexes" _) true
						(regex "FROM\\s+pg_constraint" _) true
						false))
					(define query_result (sql_with_statement session session_state query_seq query
						(lambda (tx) (if handled nil (begin
							/* Bind URL query params (v1=, v2=, ...) as prepared-statement args into the session
							before parse/build so session-sensitive planner rewrites see the right values. */
							(sql_http_query_parameters req session)
							(define formula (cached_parse psql_queryplan_cache "psql" (list parse_psql) schema query
								(list (quote sql-policy-for) (req "username")) (req "username") session false tx))
							(sql_execute_formula session tx formula resultrow (lambda (_fields) true)))))))
					/* The PostgreSQL frontend owns both command tags and count absence.
					Transport the declared completion through the existing JSON result edge;
					ordinary declarations never acquire a synthetic affected_rows field. */
					(define command_tag (psql_command_tag query_result))
					(if command_tag (original_resultrow (list "command_tag" command_tag))
						(and (not resultrow_called) (number? query_result))
						(original_resultrow (list "affected_rows" query_result)))
				) query)) (lambda(e) (begin
						(error_log (concat e) schema (req "username") query)
						((res "header") "Content-Type" "text/plain")
						((res "status") 500)
						((res "print") "SQL Error: " e)
				)))
			)
			(begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized")
			)
		)
	)))
	/* handler for raw Scheme code execution (global, no schema) */
	(define handle_scm (lambda (req res code) (begin
		/* check for password - must be admin */
		(define account (sql_authenticate_account (req "username") (req "password")))
		(if (if account (account "admin") false)
			(begin
				(try (lambda () (begin
					((res "header") "Content-Type" "application/json")
					(define session (sql_bind_http_session req account))
					(session "username" (account "username"))
					(session "schema" "")
					(set result (eval (scheme code)))
					((res "print") (json_encode result))
				)) (lambda(e) (begin
						(error_log (concat e) "" (req "username") code)
						((res "header") "Content-Type" "text/plain")
						((res "status") 500)
						((res "print") "SCM Error: " e)
				)))
			)
			(begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized (admin required)")
			)
		)
	)))
	old_handler old_handler /* workaround for optimizer bug */
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			/* Scheme code execution endpoint (global, admin only) */
			"/scm" (begin
				(set code ((req "body")))
				(handle_scm req res code)
			)
			(regex "^/sql/([^/]+)$" url schema) (begin
				(set query ((req "body")))
				/* tolerate an optional trailing ';' - must be at end of string */
				(set query (match query (regex "^((?s:.*));\\s*$" _ body) body query))
				(handle_query req res schema query)
			)
			(regex "^/sql/([^/]+)/(.*)$" url schema query_un) (begin
				(set query (urldecode query_un))
				/* tolerate an optional trailing ';' - must be at end of string */
				(set query (match query (regex "^((?s:.*));\\s*$" _ body) body query))
				(handle_query req res schema query)
			)
			(regex "^/psql/([^/]+)$" url schema) (begin
				(set query ((req "body")))
				/* tolerate an optional trailing ';' - must be at end of string */
				(set query (match query (regex "^((?s:.*));\\s*$" _ body) body query))
				(handle_query_postgres req res schema query)
			)
			(regex "^/psql/([^/]+)/(.*)$" url schema query_un) (begin
				(set query (urldecode query_un))
				/* tolerate an optional trailing ';' - must be at end of string */
				(set query (match query (regex "^((?s:.*));\\s*$" _ body) body query))
				(handle_query_postgres req res schema query)
			)
			/* default */
			(old_handler req res))
	))
))

/* register SQL frontends in service registry */
(service_registry "SQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/sql/[database]" "POST, NDJSON"))
(service_registry "PSQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/psql/[database]" "POST, NDJSON"))
(service_registry "SCM Frontend" (list (arg "api-port" (env "PORT" "4321")) "/scm" "POST, JSON"))

/* shared callbacks for mysql protocol (TCP and Unix socket) */
(set mysql_auth (lambda (username_)
	(scan_lookup nil (table "system" false "user" false nil) (list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "password") (list username_))))
(set mysql_schema (lambda (username schema) (or (equal?? schema "information_schema") (list? (show schema)))))

/* Top-level so the interpreter compiles this body once instead of rebuilding a
large lambda Proc (and re-walking it via procCanUseNumberedOnly) on every
statement inside the per-query transaction callback. */
(define mysql_run_sql_statement (lambda (tx schema sql session resultrow resultfields_sql) (begin
	(define sql_parse_input (strtrim sql))
	/* tolerate an optional trailing ';' - input is already trimmed, so a
	single last-character check replaces a backtracking regex on the hot path */
	(define sql_parse_input_len (strlen sql_parse_input))
	(if (and (> sql_parse_input_len 0)
		(equal? (substr sql_parse_input (- sql_parse_input_len 1) 1) ";"))
		(set sql_parse_input (substr sql_parse_input 0 (- sql_parse_input_len 1)))
		nil)
	(define mysql_username (coalesce (session "username") "root"))
	(define formula (if (equal? (session "syntax") "postgresql")
		(cached_parse psql_queryplan_cache "psql" (list parse_psql) schema sql_parse_input
			(list (quote sql-policy-for) mysql_username) mysql_username session false tx)
		(cached_parse sql_queryplan_cache "sql" (list parse_sql) schema sql_parse_input
			(list (quote sql-policy-for) mysql_username) mysql_username session true tx)))
	(define result (sql_execute_formula session tx formula resultrow resultfields_sql))
	/* PostgreSQL syntax can be tunneled over the MySQL connection. This is
	still a MySQL OK packet, whose affected_rows field is mandatory; mapping a
	count-free PostgreSQL command tag here is transport adaptation, not PG wire. */
	(if (and (equal? (session "syntax") "postgresql") (psql_command_tag result)) 0 result))))

(set mysql_handler (lambda (schema sql resultrow_sql resultfields_sql session session_state query_seq) (begin
	(session "schema" schema)
	(define resultrow resultrow_sql)
	(try (lambda () (begin
		(if (equal? (session "syntax") "scheme") (begin
			/* Recheck the authenticated user's current privilege for every Scheme
			statement, including sessions switched before an admin revocation. */
			(if (not (scan_lookup nil (table "system" false "user" false nil)
				(list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "admin")
				(list (session "username"))))
				(error "Scheme execution requires administrator privileges"))
			(set print (lambda args (resultrow '("result" (concat args)))))
			(resultrow '("result" (eval (scheme sql))))
		) (time (sql_with_statement session session_state query_seq sql
				(lambda (tx) (mysql_run_sql_statement tx schema sql session resultrow resultfields_sql))) sql))
	)) (lambda (e) (begin
			(error_log (concat e) schema (coalesce (session "username") "root") sql)
			(error e) /* re-throw so MySQL protocol sends proper error packet */
	)))
)))

/* dedicated mysql protocol listening at specified port */
(try (lambda () (begin
	(if (not (arg "disable-mysql" false)) (begin
		(set port (arg "mysql-port" (env "MYSQL_PORT" "3307")))
		(mysql port mysql_auth mysql_schema mysql_handler)
		(if (not (nil? service_registry)) (service_registry "MySQL Protocol" (list port "" "MySQL Wire Protocol")))
		(print "MySQL server listening on port " port " (connect with `mysql -P " port " -u root -p` using password '" (arg "root-password" "admin") "'), set with --mysql-port")
	)) ; close the if for disable-mysql
)) print)

/* dedicated mysql unix socket */
(try (lambda () (begin
	(set socketpath (arg "mysql-socket" (env "MYSQL_SOCKET" "/tmp/memcp.sock")))
	(if (not (equal? socketpath ""))
		(begin
			(mysql_socket socketpath mysql_auth mysql_schema mysql_handler)
			(if (not (nil? service_registry)) (service_registry "MySQL Socket" (list socketpath "" "MySQL Unix Socket")))
			(print "MySQL socket listening on " socketpath)
	))
)) print)

(import "tsql.scm")
