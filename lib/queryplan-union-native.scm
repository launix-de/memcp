/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
/* A UNION source owns its complete tuple and window before its parent can
project/filter/join it. The carrier is a physical lexical native relation;
logical UNION nodes and declaration facts contain no native identity. */
(define lower_native_union_carrier (lambda (source consume)
	(begin
		(define relation (source_relation source))
		(define names (queryplan_result_titles relation))
		(if (equal? (count names) (count (merge_unique (list names)))) true
			(neumann_fail "build_queryplan" "derived UNION output names must be unambiguous"))
		(define columns (mapIndex names (lambda (index name)
			(begin
				(define info (sql_projection_info_at relation index '()))
				(list (quote list) "column" name "any" (quoted_runtime_list '())
					(list (quote list) "collate" (coalesceNil (sql_collation_name info) "bin")))))))
		(define scope (symbol (concat "__union_scope:" (source_alias source))))
		(define name (symbol (concat "__union_name:" (source_alias source))))
		(define table (symbol (concat "__union_table:" (source_alias source))))
		(define tx (physical_query_tx_symbol))
		(define row (quote __union_source_row))
		(define sink (list (quote lambda) (list row)
			(runtime_group_insert_expr table names
				(mapIndex names (lambda (index _name) (list (quote nth) row (+ (* index 2) 1))))
				'() (list (quote lambda) '() nil) '())))
		(define fill (physical_mutation_producer (list table) (list "insert" '())
			(list (list (quote lambda) (list (quote resultrow)) (lower_union_block relation)) sink)))
		(define close (list (quote if) (list (quote nil?) tx)
			(list (quote close_relation_scope) scope) true))
		(define program (list (quote begin)
			(list (quote createtable) (source_schema source) name (cons (quote list) columns)
				(list (quote list) "engine" "memory" "ephemeral" true "scope" scope) false tx)
			(list (quote define) table (list (quote table) (source_schema source) false name false tx))
			fill (consume table)))
		(define outcome (quote __union_outcome))
		(list (list (quote lambda) (list name)
			(list (list (quote lambda) (list scope)
				(list (list (quote lambda) (list outcome)
					(list (quote begin) close
						(list (quote if) (list (quote car) outcome) (list (quote cadr) outcome)
							(list (quote error) (list (quote cadr) outcome)))))
					(list (quote try) (list (quote lambda) '() (list (quote list) true program))
						(list (quote lambda) (list (quote __union_failure))
							(list (quote list) false (quote __union_failure))))))
				(list (quote if) (list (quote nil?) tx)
					(list (quote new_relation_scope) (physical_query_session_symbol))
					(list (quote tx_get_or_compute_resource) tx name
						(list (quote lambda) '() (list (quote new_relation_scope) (physical_query_session_symbol)))))))
			(list (quote concat) ".union-source:" (list (quote uuid)))))))

(define physical_union_source_binding (lambda (source table)
	(source_with_relation source
		(make_local_relation_reference table (source_columns source)))))
(define physical_query_with_union_binding (lambda (block source table)
	(make_query_block (qb_schema block)
		(map (qb_sources block) (lambda (candidate)
			(if (equal? (source_alias candidate) (source_alias source))
				(physical_union_source_binding candidate table) candidate)))
		(qb_fields block) (qb_where block) (qb_group block) (qb_having block)
		(qb_order block) (qb_limit block) (qb_offset block) (qb_hidden block)
		(qb_stages block) (qb_facts block))))
/* Bind the caller's row window once before constructing an input carrier.
An empty consumer never invokes its UNION producer; the ordinary native scan
continues to own actual ordering, offset and row-limit enforcement. */
(define lower_query_block_union_window (lambda (block source)
	(begin
		(define offset (quote __union_window_offset))
		(define limit (quote __union_window_limit))
		(define bounded (make_query_block (qb_schema block) (qb_sources block) (qb_fields block)
			(qb_where block) (qb_group block) (qb_having block) (qb_order block)
			limit offset (qb_hidden block) (qb_stages block) (qb_facts block)))
		(list (list (quote lambda) (list offset limit)
			(list (quote if) (list (quote equal?) limit 0) nil
				(lower_native_union_carrier source (lambda (table)
					(lower_query_block_core (physical_query_with_union_binding bounded source table))))))
			(coalesceNil (qb_offset block) 0) (coalesceNil (qb_limit block) -1)))))

/* An unordered ALL relation streams its branch reducers directly. A set or
window boundary requires the native complete-tuple carrier before a caller can
consume its result. Both choices run at the actual operand invocation. */
(define lower_union_as_dataset_reduce (lambda (block fields mapper reducer neutral shard_reducer)
	(if (and (equal? (union_mode block) (quote all))
		(empty_list? (union_order block)) (nil? (union_limit block)) (nil? (union_offset block))
		(reduce (union_branches block) (lambda (streamable branch)
			(and streamable (query_block? branch) (empty_list? (qb_group branch))
				(nil? (qb_having branch)) (not (query_block_has_aggregates? branch)))) true))
		(reduce (union_branches block) (lambda (acc branch)
			(begin
				(define aligned (union_align_branch_fields branch (projection_titles fields) (count fields)))
				(define prepared (prepare_simple_query_block_physical_core aligned))
				(define core (nth prepared 1))
				(define result (lower_query_block_as_dataset_reduce core (qb_fields core) mapper reducer neutral shard_reducer))
				(define saved (quote __union_branch_reduced))
				(define result (cons (quote !begin) (append (nth prepared 0)
					(list (list (quote lambda) (list saved)
						(cons (quote !begin) (append (nth prepared 2) saved))) result))))
				(list shard_reducer acc result))) neutral)
		(begin
			(define wrapper (logical_union_relation_query block))
			(define source (car (qb_sources wrapper)))
			(lower_native_union_carrier source (lambda (table)
				(lower_query_block_as_dataset_reduce
					(physical_query_with_union_binding wrapper source table)
					(qb_fields wrapper) mapper reducer neutral shard_reducer)))))))

/* A retained set is not a maintained native base table. When it feeds an
ordinary group boundary, use the existing native query-local group recipe;
this is a physical capability boundary, not a semantic freshness claim. */
(define physical_union_group_stage_for_block (lambda (block) (begin
	(define direct (if (and (or (not (empty_list? (qb_group block))) (not (nil? (qb_having block)))
		(query_block_has_aggregates? block))
		(not (nil? (find (qb_sources block) (lambda (source)
			(union_block? (source_relation source))) nil))))
		(make_group_stage_for_query_block block) nil))
	/* Normalization may already have made the complete UNION grouping an
	explicit stage. This terminal consumer can keep native NULL-safe group
	identity through emission without publishing an intermediate cache. */
	(define staged (if (and (nil? direct) (equal? (count (qb_stages block)) 1)
		(single_source? (qb_sources block)))
		(car (qb_stages block)) nil))
	(define terminal (if (and (not (nil? staged)) (group_stage? staged)
		(union_block? (gs_input staged)) (not (scoped_relational_program_stage? staged))
		(stage_output_relation? (source_relation (car (qb_sources block))))
		(equal? (stage_output_relation_id (source_relation (car (qb_sources block)))) (gs_id staged))
		(runtime_union_group_consumer_matches? block staged)) staged nil))
	(define stage (coalesceNil direct terminal))
	(if (and (not (nil? stage))
		(reduce (gs_aggregates stage) (lambda (valid aggregate)
			(and valid (or (count_distinct_descriptor? aggregate) (bounded_scalar_aggregate? aggregate)
				(equal? (cadr aggregate) (scalar_once_reduce_first))))) true)) stage nil))))

/* A grouped branch publishes its completed SQL rows before a surrounding
set can inspect tuple coordinates. Bind that semantic barrier to one native
carrier and run the downstream reducer before retiring its reader scope. */
(define lower_grouped_union_branch_reduce (lambda (alias branch keys key_names ags value_cols mapper reducer neutral shard_reducer)
	(begin
		(define database (logical_relation_schema branch))
		(define relation (make_union_block (quote all) (list branch) '() nil nil '()))
		(define source (list alias database relation false nil (union_source_declarations relation)))
		(define fields (merge (map (queryplan_result_titles relation) (lambda (name)
			(list name (list (quote get_column) alias false name false))))))
		(define wrapper (make_query_block database (list source) fields true '() nil '() nil nil '() '()
			(list (list (quote default_alias) alias))))
		(lower_native_union_carrier source (lambda (table)
			(begin
				(define input (physical_query_with_union_binding wrapper source table))
				(lower_query_block_as_dataset_reduce input
					(union_branch_group_row_fields alias input keys key_names ags value_cols)
					mapper reducer neutral shard_reducer)))))))
