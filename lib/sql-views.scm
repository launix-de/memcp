/*
Copyright (C) 2026  Carl-Philip Hänsch

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

/* Views remain logical query sources. Their parser IR is expanded into a
derived SELECT before untangle_query sees the complete query. */

(set sql_view_catalog_state (coalesce sql_view_catalog_state (newsession)))
(if (nil? (sql_view_catalog_state "version"))
	(sql_view_catalog_state "version" 0)
	true)
(if (nil? (sql_view_catalog_state "count"))
	(sql_view_catalog_state "count" 0)
	true)

(define sql_view_query_generation (lambda (query)
	(if (> (sql_view_catalog_state "count") 0)
		(begin
			(define normalized_query (toLower query))
			(define references_view (scan nil (table "system" "views")
				'("scan_access" 0 "scan" 0 -1) '()
				'("name")
				(lambda (name) (> (count (split normalized_query (toLower name))) 1))
				'() (lambda (acc) true) false (lambda (a b) (or a b))))
			(if references_view (sql_view_catalog_state "version") 0))
		0)))

(define sql_view_catalog_set_count (lambda (count)
	(sql_view_catalog_state "count" count)))

(define sql_view_catalog_changed (lambda ()
	(sql_view_catalog_state "version" (+ (sql_view_catalog_state "version") 1))))

(define sql_apply_view_column_aliases (lambda (query aliases)
	(if (nil? aliases)
		query
		(match query
			((symbol query-block) query_schema tables fields condition group having order limit offset hidden stages facts)
			(if (equal? (* 2 (count aliases)) (count fields))
				(list (quote query-block) query_schema tables
					(merge (map (produceN (count aliases)) (lambda (i)
						(list (nth aliases i) (nth fields (+ (* i 2) 1))))))
					condition group having order limit offset hidden stages facts)
				(error "view column alias count does not match its output columns"))
			_ (error "view column aliases currently require one query block")))))

(define sql_find_view (lambda (schema name)
	(scan nil (table "system" "views")
		'("scan_access" 2 "scan" 0 -1 "equal" "database" 0 0 3 "" "equal" "name" 1 1 3 "") (list schema name)
		'() (lambda () true)
		'("dialect" "sql" "ir")
		(lambda (acc dialect sql ir) (list dialect sql ir))
		nil (lambda (a b) b))))

(define sql_expand_views (lambda (query policy) (begin
	(define expand_node (lambda (node stack)
		(match node
			((symbol query-block) query_schema sources fields condition group having order limit offset hidden stages facts)
			(list (quote query-block)
				query_schema
				(map sources (lambda (source)
					(match source
						'(alias source_schema relation outer join_condition)
						(begin
							(define expanded_relation
								(if (string? relation)
									(begin
										(if policy (policy source_schema relation false) true)
										(if (has? (show source_schema) relation)
											relation
											(begin
												(define view (sql_find_view source_schema relation))
												(if (nil? view)
													relation
													(begin
														(define key (concat source_schema "." relation))
														(if (contains? stack key)
															(error (concat "circular view reference: " key))
															true)
														(define stored_ir (nth view 2))
														(if (nil? stored_ir)
															(error (concat "view " key " has no parsed IR"))
															(expand_node (json_decode_scmer stored_ir) (cons key stack))))))))
									(expand_node relation stack)))
							(list alias source_schema expanded_relation outer (expand_node join_condition stack)))
						_ (error "invalid query source while expanding views"))))
				(expand_node fields stack)
				(expand_node condition stack)
				(expand_node group stack)
				(expand_node having stack)
				(expand_node order stack)
				limit offset
				(expand_node hidden stack)
				(expand_node stages stack)
				(expand_node facts stack))
			((symbol union-block) mode branches order limit offset facts)
			(list (quote union-block) mode
				(map branches (lambda (branch) (expand_node branch stack)))
				(expand_node order stack) limit offset (expand_node facts stack))
			(cons head tail)
			(cons (expand_node head stack)
				(map tail (lambda (item) (expand_node item stack))))
			_ node)))
	(if (> (sql_view_catalog_state "count") 0)
		(expand_node query '())
		query))))

(define create_sql_view (lambda (tx schema name dialect sql ir mode) (begin
	(define serialized_ir (json_encode ir))
	(define existing (scan tx (table "system" "views")
		'("scan_access" 2 "scan" 0 -1 "equal" "database" 0 0 3 "" "equal" "name" 1 1 3 "") (list schema name)
		'() (lambda () true)
		'() (lambda (acc) (+ acc 1)) 0 +))
	(define result
		(if (> existing 0)
			(match mode
				"replace"
				(scan tx (table "system" "views")
					'("scan_access" 2 "scan" 0 -1 "equal" "database" 0 0 3 "" "equal" "name" 1 1 3 "") (list schema name)
					'() (lambda () true)
					'("$update")
					(lambda (acc $update) (begin
						($update (list "dialect" dialect "sql" sql "ir" serialized_ir))
						(+ acc 1)))
					0 +)
				"ignore" 0
				_ (error (concat "view " schema "." name " already exists")))
			(insert (table "system" "views")
				'("database" "name" "dialect" "sql" "ir")
				(list (list schema name dialect sql serialized_ir)))))
	(if (or (equal? existing 0) (equal? mode "replace"))
		(begin
			(if (equal? existing 0)
				(sql_view_catalog_set_count (+ (sql_view_catalog_state "count") 1))
				true)
			(sql_view_catalog_changed))
		true)
	result)))

(define drop_sql_view (lambda (tx schema name if_exists) (begin
	(define removed (scan tx (table "system" "views")
		'("scan_access" 2 "scan" 0 -1 "equal" "database" 0 0 3 "" "equal" "name" 1 1 3 "") (list schema name)
		'() (lambda () true)
		'("$update")
		(lambda (acc $update) (begin ($update) (+ acc 1)))
		0 +))
	(if (> removed 0)
		(begin
			(sql_view_catalog_set_count (max 0 (- (sql_view_catalog_state "count") removed)))
			(sql_view_catalog_changed)
			removed)
		(if if_exists 0 (error (concat "view " schema "." name " does not exist")))))))

/* Drop the physical database before cleaning its catalog rows. This ordering
prevents a refused or failed protected-schema drop from changing access or view
metadata. */
(define drop_sql_database (lambda (tx schema if_exists) (begin
	(define dropped (dropdatabase schema if_exists))
	(if dropped (begin
		(scan tx (table "system" "access")
			'("scan_access" 1 "scan" 0 -1 "equal" "database" 0 0 3 "") (list schema)
			'() (lambda () true)
			'("$update")
			(lambda (acc $update) (begin ($update) (+ acc 1)))
			0 +)
		(define removed_views (scan tx (table "system" "views")
			'("scan_access" 1 "scan" 0 -1 "equal" "database" 0 0 3 "") (list schema)
			'() (lambda () true)
			'("$update")
			(lambda (acc $update) (begin ($update) (+ acc 1)))
			0 +))
		(if (> removed_views 0) (begin
			(sql_view_catalog_set_count (max 0 (- (sql_view_catalog_state "count") removed_views)))
			(sql_view_catalog_changed)
		) true)
	) true)
	dropped)))
