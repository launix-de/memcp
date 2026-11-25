/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Forward dynamic cursors consume current native rows at each FETCH. Their
only position is the complete declared driver PK tuple, never storage record
IDs or an OPEN-time keyset. A LEFT anti-semijoin preserves that identity even
when its ON collation allows multiple RHS matches: only the single unmatched
row can satisfy RHS.nonnullable-column IS NULL.
https://learn.microsoft.com/sql/relational-databases/cursors
SQL Server 2022 membership, key movement and sticky EOF: https://dbfiddle.uk/vQ6ky6sW */
(define tsql_cursor_dynamic_column (lambda (expression)
	(tsql_cursor_update_expression expression)))
(define tsql_cursor_dynamic_anti_term? (lambda (source term)
	(match (tsql_cursor_dynamic_column term)
		((symbol nil?) probe) (begin
			(define name (direct_column_name_for_alias source (tsql_cursor_dynamic_column probe)))
			(if (nil? name) false (source_column_guaranteed_nonnull? source name)))
		_ false)))
(define tsql_cursor_dynamic_recipe (lambda (query options) (begin
	(define sources (if (query_block? query) (qb_sources query) '()))
	(if (or (options "scroll") (not (query_block? query))
		(not (has? '(1 2) (count sources)))) nil (begin
			(define driver (car sources))
			(define keys (source_primary_key_columns driver))
			(define columns (map keys (lambda (name)
				(list 'get_column (source_alias driver) false name false))))
			(define direct (lambda (expression) (match (tsql_cursor_dynamic_column expression)
				((symbol get_column) _alias _alias_ic _name _name_ic) true _ false)))
			(define order (coalesceNil (qb_order query) '()))
			(define semantic (list (qb_fields query) (qb_where query) (map sources source_join_expr)))
			(if (and (not (source_outer? driver))
				(reduce sources (lambda (ok source) (and ok (string? (source_relation source)))) true)
				(not (empty_list? keys))
				(reduce keys (lambda (ok key) (and ok (source_column_guaranteed_nonnull? driver key))) true)
				(empty_list? (coalesceNil (qb_group query) '())) (nil? (qb_having query))
				(empty_list? (coalesceNil (qb_stages query) '()))
				(nil? (qb_limit query)) (nil? (qb_offset query))
				(not (query_contains_subquery? query)) (not (query_requires_fresh_invocation? query nil false))
				(empty_list? (query_expr_session_reads semantic))
				(reduce (extract_assoc (expand_query_block_fields sources (qb_fields query))
					(lambda (_title expression) (direct expression))) (lambda (ok value) (and ok value)) true)
				(or (empty_list? order) (and (equal? (count order) (count keys))
					(reduce (produceN (count keys)) (lambda (ok index)
						(and ok (identical? (cadr (nth order index)) <)
							(equal? (direct_column_name_for_alias driver
								(tsql_cursor_dynamic_column (car (nth order index)))) (nth keys index)))) true)))
				(if (equal? (count sources) 1) true (begin
					(define right (cadr sources))
					(and (source_outer? right)
						(not (nil? (find (split_and_terms (qb_where query))
							(lambda (term) (tsql_cursor_dynamic_anti_term? right term)) nil)))))))
				(list query columns) nil))))))

/* Internal scalar parameter names contain ':' and therefore cannot collide
with a user-declared T-SQL variable. The ordinary parameter binder hoists their
values before a row callback is published or delegated. This invocation adapter
is never retained by the cursor, its plan or a storage computed column. */
(define tsql_cursor_dynamic_input (lambda (parent started keys) (begin
	(define values (merge (cons (list "__tsql_parameter:__cursor:started" started)
		(mapIndex keys (lambda (index key) (list (concat "__tsql_parameter:__cursor:key:" index) key))))))
	(lambda args (match args
		'() (merge_unique (list (parent) (extract_assoc values (lambda (key _value) key))))
		'(key) (if (has_assoc? values key) (values key) (parent key))
		_ (apply parent args))))))
(define tsql_cursor_dynamic_after (lambda (sources columns index)
	(if (equal? index (count columns)) false (begin
		(define column (nth columns index))
		(define info (sql_expr_info sources column))
		(define marker (list 'sql_type_expr
			(list 'session (concat "__tsql_parameter:__cursor:key:" index))
			(sql_info_datatype info) (sql_info_collation info)))
		/* Bookmark equivalence is the native ORDER BY relation, not SQL's
		space-padded text equality. Use exactly that relation in both directions. */
		(define less (if (sql_text_type? (sql_info_type info)) (collate (sql_collation_name info) false) <))
		(list 'or (list 'sql_type_expr (list sql_compare column marker less ">") "BOOLEAN" nil)
			(list 'and (list 'sql_type_expr (list sql_compare column marker less "equal?") "BOOLEAN" nil)
				(tsql_cursor_dynamic_after sources columns (+ index 1))))))))
(define tsql_cursor_dynamic_row (close_procedure (lambda (left right)
	(if (nil? left) right (nil? right) left
		(error "cursor one-row consumer received more than one completed row")))))
(define tsql_cursor_dynamic_source_contracts (lambda (query tx)
	(map (qb_sources query) (lambda (source) (begin
		(define handle (table (source_schema source) false (source_relation source) false tx))
		(if (nil? handle) (error "cursor source no longer exists") true)
		(define identity (show handle "identity" tx))
		(if (nil? identity) (error "cursor source has no native incarnation") true)
		(list (source_schema source) (source_relation source) identity
			(sql_relation_local_declarations (source_schema source) (source_relation source) tx)))))))
(define tsql_cursor_dynamic_validate (lambda (cursor session tx) (begin
	(tx_check tx)
	(map (cursor "source_contracts") (lambda (contract) (begin
		(tsql_require_database_access (tsql_session_actor session) (car contract))
		(define handle (table (car contract) false (cadr contract) false tx))
		(if (nil? handle) (tsql_raise_error 16943 "cursor source was dropped" 1)
			(not (equal? (show handle "identity" tx) (nth contract 2)))
			(tsql_raise_error 16943 "cursor source incarnation changed" 1)
			(not (sql_relation_local_declarations_match? (car contract) (cadr contract) tx (nth contract 3) '("columns" "constraints")))
			(tsql_raise_error 16943 "cursor source declaration changed" 1) true)))))))

(define tsql_cursor_dynamic_open (lambda (cursor session tx) (begin
	(define planning (tsql_cursor_planning session))
	(define bindings (tsql_temporary_bindings session))
	(define policy (tsql_temporary_policy (session "username") bindings))
	/* OPEN owns the only SQL parse. Later variant misses consume this same
	bound logical declaration through the common planner, never SQL source. */
	(define parsed (parse_tsql (cursor "schema") (cursor "text") policy planning tx))
	(define query (match parsed ((symbol quote) query) query _ (error "cursor requires SELECT")))
	(define query (tsql_describe_query query policy planning tx))
	(define recipe (tsql_cursor_dynamic_recipe query (cursor "options")))
	(if (nil? recipe) (tsql_raise_error 16943 "cursor source no longer supports its dynamic declaration" 1) true)
	(define sources (qb_sources query))
	(define keys (cadr recipe))
	(define visible (expand_query_block_fields sources (qb_fields query)))
	(define fields (tsql_result_columns query '()))
	(define contracts (append (cursor "contracts") (list "__cursor:started" '("BIT"))))
	(define contracts (merge (list contracts (mapIndex keys (lambda (index column) (begin
		(define type (sql_info_datatype (sql_expr_info sources column)))
		(list (concat "__cursor:key:" index) (if (list? type) (cons (car type) (cadr type)) (list type)))))))))
	(define projections (merge (list (map (produceN (count fields)) (lambda (index)
		(list (concat "c" index) (nth visible (+ 1 (* 2 index))))))
		(mapIndex keys (lambda (index column) (list (concat "k" index) column))))))
	/* A coarse first-axis bound avoids rescanning the already consumed
	prefix. Text cursor keys use this exact native ASC relation, whose NULL
	is first; the strict full-key residual still decides tuple membership. */
	(define first_key (car keys))
	(define first_info (sql_expr_info sources first_key))
	(define seek (if (sql_text_type? (sql_info_type first_info))
		(list 'not (list (collate (sql_collation_name first_info) false) first_key
			(list 'if (list 'session "__tsql_parameter:__cursor:started")
				(list 'session "__tsql_parameter:__cursor:key:0") nil))) true))
	(define query (make_query_block (qb_schema query) sources (merge projections)
		(combine_where (combine_where (qb_where query) seek) (list 'or
			(list 'not (list 'session "__tsql_parameter:__cursor:started"))
			(tsql_cursor_dynamic_after sources keys 0)))
		nil nil (map keys (lambda (column) (list column <))) 1 nil '() '()
		(qassoc_set (qb_facts query) 'fresh_invocation true)))
	(define source_contracts (tsql_cursor_dynamic_source_contracts query tx))
	(define factory (close_procedure (lambda (_database _text policy planning tx) (begin
		(map sources (lambda (source) (policy (source_schema source) (source_relation source) false)))
		(build_reducing_queryplan_term query
			(make_result_reduction tsql_cursor_dynamic_row nil tsql_cursor_dynamic_row) planning tx
			(lambda (formula) (tsql_bind_parameter_inputs formula contracts)))))))
	(define input (tsql_cursor_dynamic_input session false (map keys (lambda (_key) nil))))
	(define cache (newcachemap))
	(define entry (sql_queryplan_new_entry cache "cursor" (list factory) (cursor "schema")
		(cursor "text") policy input tx nil nil nil))
	/* Publication follows successful binding/compilation. A failed OPEN does
	not replace the prior last-opened cursor status or install partial state. */
	(cursor "fields" fields) (cursor "source_contracts" source_contracts)
	(cache "cursor" entry)
	(cursor "dynamic_cache" cache)
	/* The cache may forget every compiled entry. Retain only OPEN's closed
	logical factory, whose invocation receives fresh policy/planning/tx inputs. */
	(cursor "dynamic_factory" factory)
	(cursor "bookmark" (map keys (lambda (_key) nil)))
	(cursor "started" false) (cursor "eof" false) (cursor "position" 0) (cursor "count" 0)
	(cursor "open" true)
	(tsql_cursor_publish_rows cursor -1)
	0)))
(define tsql_cursor_dynamic_fetch (lambda (cursor session tx) (begin
	(tsql_cursor_dynamic_validate cursor session tx)
	(if (cursor "eof") nil (begin
		(define input (tsql_cursor_dynamic_input session (cursor "started") (cursor "bookmark")))
		(define cache (cursor "dynamic_cache"))
		(define factory (cursor "dynamic_factory"))
		(define schema (cursor "schema"))
		(define text (cursor "text"))
		/* Read-through recovery belongs to the existing cache owner. The
		access-local producer is not part of its payload; the new dispatcher
		retains only the closed logical factory and ordinary policy metadata. */
		(define entry (cache "cursor" tx (lambda ()
			(sql_queryplan_new_entry cache "cursor" (list factory) schema text
				(tsql_temporary_policy (input "username") (tsql_temporary_bindings input)) input tx nil nil nil))))
		(define row (sql_execute_formula input tx (cadr entry)
			(lambda (row) row) (lambda args true)))
		(if (nil? row) (begin (cursor "eof" true) nil) (begin
			(define width (count (cursor "fields")))
			(define keys (map (produceN (count (cursor "bookmark"))) (lambda (index)
				(nth row (+ 1 (* 2 (+ width index)))))))
			(cursor "bookmark" keys) (cursor "started" true)
			(map (produceN width) (lambda (index) (nth row (+ 1 (* 2 index))))))))))))
