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

/* The stored dialect owns syntax-specific declaration references. Resolve
them once in the consuming plan, with its policy, guards and transaction.
Only the original pure declaration IR is durable; these callbacks are parser
modules, never persisted view values or row-time SQL interpreters. */
(define sql_view_expansion_compilers (newsession))
(define register_sql_view_expansion (lambda (dialect compiler)
	(sql_view_expansion_compilers dialect compiler)))
(define sql_expand_view_declaration (lambda (query dialect policy planning tx) (begin
	(define compiler (sql_view_expansion_compilers dialect))
	(if compiler (compiler query policy planning tx) query))))
(define sql_view_planning_scope (lambda (parent path)
	(lambda args (match args
		'(key) (if (equal? key "__sql_view_expansion_path") path (parent key))
		'(key value) (if (equal? key "__sql_view_expansion_path")
			(error "view expansion path is immutable") (parent key value))
		_ (apply parent args)))))

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
			/* Only existence is needed for this conservative lexical dependency.
			Keep literal lowercase substring semantics, without allocating split
			lists or visiting later rows after a match has already been found. */
			(define references_view (scan_exists nil (table "system" false "views" false nil)
				'(369435906932736) '() '("name")
				(lambda (name)
					/* Empty names cannot be declared in SQL, but preserve the old
					split predicate for a legacy/native catalog entry as well. */
					(if (equal? name "") (> (text_length normalized_query "rune") 1)
						(>= (string_find normalized_query (toLower name) 0 false "byte") 0)))))
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
	(scan nil (table "system" false "views" false nil)
		(list 369436443803648 (scan_boundary "equal" "database" 0 0 true true "" false) (scan_boundary "equal" "name" 1 1 true true "" false)) (list schema name)
		'() (lambda () true)
		'("dialect" "sql" "ir")
		(lambda (acc dialect sql ir) (list dialect sql ir))
		nil (lambda (a b) b))))

/* Restore legacy source layouts only at source positions. A declared SQL name
belongs to the stored lexical scope. Acquire native declarations separately
from the consuming transaction when the restored view is expanded. */
(define sql_restore_source_layout (lambda (node) (match node
	((symbol quote) _literal) node
	((symbol query-block) database sources fields condition grouping having ordering limit offset hidden stages facts)
	(make_query_block database (map sources (lambda (src) (begin
		(define source (match src
			'(alias schema relation outer join)
			(list alias schema relation outer join nil)
			'(_alias _schema _relation _outer _join _declarations)
			(begin
				(define binding_name (source_binding_name src))
				(source_with_declarations src (if (nil? binding_name) nil
					(list "binding_name" binding_name))))
			_ (error "invalid stored logical source declaration")))
		(source_with_join_expr (source_with_relation source
			(sql_restore_source_layout (source_relation source)))
			(sql_restore_source_layout (source_join_expr source))))))
		(sql_restore_source_layout fields) (sql_restore_source_layout condition)
		(sql_restore_source_layout grouping) (sql_restore_source_layout having)
		(sql_restore_source_layout ordering) limit offset
		(sql_restore_source_layout hidden) (sql_restore_source_layout stages) facts)
	(cons head tail) (cons (sql_restore_source_layout head) (map tail sql_restore_source_layout))
	_ node)))

/* Virtual table catalogs combine the native catalog producer with a native
scan of stored view declarations. The generated UNION is already expanded;
its base branch deliberately denotes only the native metadata producer. */
(define sql_table_catalog_query (lambda (database) (begin
	(define native_alias "__native_table_catalog")
	(define view_alias "__stored_view_catalog")
	(define columns (map (information_schema_column_catalog "information_schema" "tables")
		(lambda (column) (column "Field"))))
	(define native (make_query_block database
		(list (list native_alias "information_schema" "tables" false nil nil))
		(merge (map columns (lambda (column)
			(list column (list 'get_column native_alias false column false)))))
		true nil nil nil nil nil nil nil nil))
	(define views (make_query_block database
		(list (list view_alias "system" "views" false nil nil))
		(list "table_catalog" "def"
			"table_schema" (list 'get_column view_alias false "database" false)
			"table_name" (list 'get_column view_alias false "name" false)
			"table_type" "VIEW" "engine" nil "table_rows" nil "data_length" nil
			"table_collation" nil "table_comment" "VIEW")
		true nil nil nil nil nil nil nil nil))
	(make_union_block 'all (list native views) nil nil nil nil))))

(define sql_has_table_catalog_source? (lambda (node) (match node
	((symbol quote) _data) false
	((symbol query-block) _database sources fields condition grouping having ordering _limit _offset hidden stages _facts)
	(or (reduce sources (lambda (found source)
		(or found (and (equal?? (source_schema source) "information_schema")
			(equal?? (source_relation source) "tables"))
			(sql_has_table_catalog_source? (source_relation source))
			(sql_has_table_catalog_source? (source_join_expr source)))) false)
		(sql_has_table_catalog_source? (list fields condition grouping having ordering hidden stages)))
	(cons head tail) (or (sql_has_table_catalog_source? head)
		(reduce tail (lambda (found item) (or found (sql_has_table_catalog_source? item))) false))
	_ false)))

(define sql_expand_views (lambda (query policy planning tx) (begin
	(define planning (if (nil? planning) (newsession) planning))
	(define expand_node (lambda (node stack)
		(match node
			((symbol query-block) query_schema sources fields condition group having order limit offset hidden stages facts)
			(list (quote query-block)
				query_schema
				(map sources (lambda (source)
					(match source
						'(alias source_schema relation outer join_condition declarations)
						(begin
							(define expanded_relation
								(if (string? relation)
									(begin
										(if policy (policy source_schema relation false) true)
										(if (and (equal?? source_schema "information_schema") (equal?? relation "tables"))
											(sql_table_catalog_query query_schema)
											(if (table source_schema false relation false tx)
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
																(sql_expand_view_declaration
																	(expand_node (sql_restore_source_layout (restore_storage_code (json_decode_scmer stored_ir))) (cons key stack))
																	(car view) policy (sql_view_planning_scope planning (cons key stack)) tx))))))))
									(expand_node relation stack)))
							(list alias source_schema expanded_relation outer (expand_node join_condition stack) declarations))
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
	(sql_bind_catalog_declarations (if (or (> (sql_view_catalog_state "count") 0) (sql_has_table_catalog_source? query))
		(expand_node query (coalesceNil (planning "__sql_view_expansion_path") '())) query) tx planning))))

(define create_sql_view (lambda (tx schema name dialect sql ir mode) (begin
	(define serialized_ir (json_encode ir))
	(define existing (scan tx (table "system" false "views" false tx)
		(list 369436443803648 (scan_boundary "equal" "database" 0 0 true true "" false) (scan_boundary "equal" "name" 1 1 true true "" false)) (list schema name)
		'() (lambda () true)
		'() (lambda (acc) (+ acc 1)) 0 +))
	(define result
		(if (> existing 0)
			(match mode
				"replace"
				(scan tx (table "system" false "views" false tx)
					(list 369436443803648 (scan_boundary "equal" "database" 0 0 true true "" false) (scan_boundary "equal" "name" 1 1 true true "" false)) (list schema name)
					'() (lambda () true)
					'("$update")
					(lambda (acc $update) (begin
						($update (list "dialect" dialect "sql" sql "ir" serialized_ir))
						(+ acc 1)))
					0 +)
				"ignore" 0
				_ (error (concat "view " schema "." name " already exists")))
			(insert (table "system" false "views" false tx)
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
	(define removed (scan tx (table "system" false "views" false tx)
		(list 369436443803648 (scan_boundary "equal" "database" 0 0 true true "" false) (scan_boundary "equal" "name" 1 1 true true "" false)) (list schema name)
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

/* SQL database lifecycle and alias declaration publication share one DDL
gate per database. Ordinary reads and DML never acquire it. The stable scope
retains gates, not requests, transactions or executable plans.
Privileged native createdatabase/dropdatabase intentionally bypass this SQL
catalog contract; their callers own any frontend metadata cleanup. */
(define sql_database_ddl_gates (newsession))
(define sql_database_ddl_scope (newsession))
(define sql_database_ddl (lambda (schema tx operation)
	((sql_database_ddl_gates "get_or_compute_scoped" sql_database_ddl_scope schema (lambda () (mutex))) tx operation)))

/* Optional frontend declarations are ordinary rows, not native databases.
Deletions use native autocommit so their WAL is durable before a new namespace
appears. New compatibility flags must not survive DROP/recreation either. */
(define sql_remove_database_frontend_metadata (lambda (schema)
	(reduce '("tsql_types" "tsql_database_options") (lambda (count name) (begin
		(define catalog (table "system" false name false nil))
		(+ count (if catalog
			(scan nil catalog
				(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false)) (list schema)
				'() (lambda () true) '("$update")
				(lambda (acc remove) (begin (remove) (+ acc 1))) 0 +)
			0)))) 0)))

/* Caller holds the existing database DDL gate. The decision is consumed
immediately there, never cached as an existence or privilege fact. */
(define sql_database_create_admission (lambda (actor schema if_not_exists) (begin
	(define exists (not (nil? (show schema))))
	(sql_database_create_access actor schema if_not_exists exists)
	(not (and if_not_exists exists)))))
(define create_sql_database (lambda (tx schema if_not_exists options source actor)
	(sql_database_ddl schema tx (lambda () (begin
		/* An authorized existing-DB request is complete here. In particular,
		a privileged native DROP racing after this observation cannot turn the
		client's no-op into a CREATE or frontend metadata deletion. */
		(if (sql_database_create_admission actor schema if_not_exists) (begin
			/* A crash can leave frontend rows after the physical DROP. Retire them
			while the namespace is still absent, before native CREATE publishes it.
			An existing namespace, including IF NOT EXISTS, keeps its declarations. */
			(if (nil? (show schema)) (sql_remove_database_frontend_metadata schema) true)
			(createdatabase schema if_not_exists options source)) false))))))

/* Existing member no-ops finish locally, even on a replica. Actual namespace
creation retains the existing admin-forwarding protocol. Admin forwarding runs
after leaving the SQL DDL gate; no local CREATE follows an admitted no-op. */
(define sql_database_create_replica (lambda (tx requests session schema query postgresql resultrow resultfields) (begin
	(if (equal? (count requests) 1) true
		(error "mixed database creation forwarding is not supported"))
	(define request (car requests))
	(define create (sql_database_ddl (car request) tx (lambda ()
		(sql_database_create_admission (nth request 2) (car request) (cadr request)))))
	(if create (replication_admin session schema query postgresql resultrow resultfields) false))))

/* Drop the physical database before cleaning its catalog rows. This ordering
prevents a refused or failed protected-schema drop from changing access or view
metadata. */
(define drop_sql_database (lambda (tx schema if_exists)
	(sql_database_ddl schema tx (lambda () (begin
		(define dropped (dropdatabase schema if_exists))
		(if dropped (begin
			(sql_remove_database_frontend_metadata schema)
			(scan tx (table "system" false "access" false tx)
				(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false)) (list schema)
				'() (lambda () true)
				'("$update")
				(lambda (acc $update) (begin ($update) (+ acc 1)))
				0 +)
			(define removed_views (scan tx (table "system" false "views" false tx)
				(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false)) (list schema)
				'() (lambda () true)
				'("$update")
				(lambda (acc $update) (begin ($update) (+ acc 1)))
				0 +))
			(if (> removed_views 0) (begin
				(sql_view_catalog_set_count (max 0 (- (sql_view_catalog_state "count") removed_views)))
				(sql_view_catalog_changed)
			) true)
		) true)
		dropped)))))
