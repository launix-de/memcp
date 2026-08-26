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

(import "sql-parser.scm")
(import "psql-parser.scm")
(import "sql-builtins.scm")
(import "sql-metadata.scm")
(import "queryplan.scm")
(import "sql-views.scm")

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

/* Keep exact SQL variants out of the parser while sharing their compiled plan.
Only parameterized results enter the small front cache; exact-only statements
continue to occupy just their existing query-plan entry. The third result item
is the normalized shape hash. Keeping it beside the bindings avoids traversing
the same normalized query again on every warm literal-specialization hit. */
(define sql_parameterized_shape_result (lambda (result)
	(match result
		'(normalized bindings shape_hash) result
		'(normalized bindings) (list normalized bindings (fnv_hash normalized)))))

(define sql_parameterize_select_literals_cached (lambda (cache query enabled)
	(if (not enabled)
		(list query '() (fnv_hash query))
		(begin
			(define cached (cache query))
			(if cached
				cached
				(match (parameterize_sql_select_literals query) '(normalized bindings shape_hash)
					(begin
						(define result (if (equal? bindings '())
							(sql_parameterized_shape_result (sql_parameterize_select_like_strings query enabled))
							(list normalized bindings shape_hash)))
						(if (equal? (cadr result) '())
							result
							(cache query result)))))))))

(define sql_parameterize_select_literals (lambda (query enabled)
	(sql_parameterize_select_literals_cached sql_literal_shape_cache query enabled)))

/* A parse can mention the same physical table through many aliases. Resolve
and authorize each readable table once, then keep using show's immutable table
snapshot through the stable handle for the rest of this compile only. */
(define sql_compile_table_policy (lambda (policy)
	(begin
		(define catalog (newsession))
		(lambda (schema tbl write)
			(if write
				(if policy (policy schema tbl true) true)
				(begin
					(define handle (table schema tbl))
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

/* sql_parameterize_select_like_strings: query-plan-cache helper for ad-hoc
fulltext-ish SELECTs. It replaces string literals directly following LIKE or
inside MATCH...AGAINST(...) with ? placeholders and returns (normalized-query bindings). Other string
literals, DDL/DML and already-parameterized statements keep exact cache keys. */
(define sql_parameterize_select_like_strings (lambda (query enabled) (begin
	(define starts_like_select (lambda (q)
		(match q (regex "^\\s*SELECT\\b" _) true false)))
	(define parameterized_rhs_literal? (lambda (q pos) (begin
		(define prefix (toUpper (strrtrim (substr q 0 pos))))
		(or
			(match prefix (regex "(?s:.*)\\bLIKE$" _) true false)
			(match prefix (regex "(?s:.*)\\bAGAINST\\s*\\($" _) true false)))))
	(define read_string_literal (lambda (q start quote_ch) (begin
		(define len (strlen q))
		(for (list (+ start 1) "" false)
			(lambda (i value done) (and (not done) (< i len)))
			(lambda (i value done) (begin
				(define ch (substr q i 1))
				(if (equal? ch "\\")
					(if (< (+ i 1) len)
						(list (+ i 2) (concat value (substr q (+ i 1) 1)) false)
						(list (+ i 1) value false))
					(if (equal? ch quote_ch)
						(list (+ i 1) value true)
						(list (+ i 1) (concat value ch) false)))))))))
	(if (or
		(not enabled)
		(not (starts_like_select query))
		(match (toUpper query)
			(regex "(?:\\b(?:COUNT|SUM|AVG|MIN|MAX|GROUP_CONCAT)\\s*\\(|\\bGROUP\\s+BY\\b|\\bHAVING\\b)" _)
			true false)
		(not (or
			(match (toUpper query) (regex "\\bLIKE\\b" _) true false)
			(match (toUpper query) (regex "\\bAGAINST\\b" _) true false)))
		(match query (regex "\\?" _) true false))
		(list query '())
		(begin
			(define len (strlen query))
			(define state (for (list 0 "" '() false)
				(lambda (i out bindings invalid) (and (not invalid) (< i len)))
				(lambda (i out bindings invalid) (begin
					(define ch (substr query i 1))
					(if (and (or (equal? ch "'") (equal? ch "\"")) (parameterized_rhs_literal? query i))
						(match (read_string_literal query i ch) '(next_i value done)
							(if done
								(list next_i (concat out "?") (merge bindings (list value)) false)
								(list len query '() true)))
						(list (+ i 1) (concat out ch) bindings false))))))
			(match state '(end_i normalized bindings invalid)
				(if (or invalid (equal? bindings '()))
					(list query '())
					(list normalized bindings))))))))

/* Copy request bindings and catalog context into an isolated planning session.
The request transaction is deliberately excluded: cached plans must obtain
__memcp_tx from the executing session, never retain the transaction which
happened to compile a shared variant. The condition accumulator belongs to
exactly one compile and substitutes for threading guard state through every
functional planner return value. */
(define sql_queryplan_compile_session (lambda (source_session)
	(begin
		(define planning_session (newsession))
		(define compile_bindings (newsession))
		(define observation_scope (source_session
			"get_or_compute_scoped"
			(context "query")
			"__memcp_queryplan_observations"
			(lambda () (newsession))))
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
		(planning_session "__memcp_queryplan_guard_conditions" (newsession))
		(planning_session "__memcp_queryplan_guard_condition_catalog" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_guard_bindings" (newsession))
		(planning_session "__memcp_queryplan_guard_binding_catalog" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_guarded_session_keys" (make_structural_catalog (quote ast)))
		(planning_session "__memcp_queryplan_observed_session_keys" (newsession))
		(planning_session "__memcp_queryplan_preparations" (newsession))
		(planning_session "__memcp_queryplan_statistics" (newsession))
		planning_session)))

(define sql_queryplan_preparations_from_session (lambda (planning_session)
	(begin
		(define preparations (planning_session "__memcp_queryplan_preparations"))
		(map (produceN (coalesceNil (preparations "count") 0))
			(lambda (idx) (preparations (concat "preparation:" idx)))))))

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
(define sql_queryplan_runtime_guard_expr (lambda (expr)
	(match expr
		((symbol quote) _value) expr
		((symbol session) key) (if (match key
			(regex "^__memcp_queryplan_observation_(?:value|metric)_" _) true
			_ false)
			(planner_queryplan_observation_current_read_expr key)
			(list (list (quote context) "session") key))
		((quote session) key) (if (match key
			(regex "^__memcp_queryplan_observation_(?:value|metric)_" _) true
			_ false)
			(planner_queryplan_observation_current_read_expr key)
			(list (list (quote context) "session") key))
		(cons head tail) (cons
			(sql_queryplan_runtime_guard_expr head)
			(map tail sql_queryplan_runtime_guard_expr))
		_ expr)))

(define sql_queryplan_guard_from_session (lambda (planning_session)
	(begin
		(define condition_accumulator (planning_session "__memcp_queryplan_guard_conditions"))
		(define condition_catalog (planning_session "__memcp_queryplan_guard_condition_catalog"))
		(define conditions (merge (list
			(if (nil? condition_catalog)
				(map (condition_accumulator) (lambda (key) (condition_accumulator key)))
				(map (produceN (coalesceNil (condition_accumulator "count") 0))
					(lambda (idx) (condition_accumulator (concat "condition:" idx)))))
			(sql_queryplan_uncovered_binding_conditions planning_session))))
		(define raw_guard (match conditions
			(cons condition '()) condition
			(cons _head _tail) (cons (quote and) conditions)
			_ true))
		(define binding_session (planning_session "__memcp_queryplan_guard_bindings"))
		(define binding_catalog (planning_session "__memcp_queryplan_guard_binding_catalog"))
		(define bindings (if (nil? binding_catalog)
			(map (binding_session) (lambda (key) (binding_session key)))
			(map (produceN (coalesceNil (binding_session "count") 0))
				(lambda (idx) (binding_session (concat "binding:" idx))))))
		(sql_queryplan_runtime_guard_expr (if (empty_list? bindings)
			raw_guard
			(cons
				(list (quote lambda) (map bindings car) raw_guard)
				(map bindings cadr)))))))

(define sql_compile_queryplan_variant (lambda (parse_fn schema parse_query policy source_session)
	(begin
		(context "check")
		(define planning_session (sql_queryplan_compile_session source_session))
		(define compile_policy (sql_compile_table_policy policy))
		(define raw_plan (with_session planning_session (lambda ()
			(parse_fn schema parse_query compile_policy))))
		/* Parsing includes logical and physical planning. Never spend optimizer
		work or install a variant after its requesting context was cancelled. */
		(context "check")
		(define plan (optimize raw_plan))
		(context "check")
		(list
			(sql_queryplan_guard_from_session planning_session)
			plan
			(sql_queryplan_preparations_from_session planning_session)))))

(define sql_queryplan_preparation_expr (lambda (preparation)
	(match preparation '(decision_id producer metric_expr) (begin
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
						(list (quote context) "check")
						(list
							(list (quote lambda) (list prepared_value)
								(list (quote !begin)
									(list observation_session value_key prepared_value)
									(list observation_session metric_key
										(list
											(list (quote lambda) (list (symbol "__queryplan_observed_value")) metric_expr)
											prepared_value))
									true))
							(sql_queryplan_runtime_guard_expr producer))
						(list (quote context) "check")
						true)))
			(planner_queryplan_observation_session_expr))))))

/* Walk variants in priority order. A cheap guarded variant can return before
an older expensive observation is prepared. Once a variant needs an
observation, prepare each physical decision at most once and then evaluate its
otherwise side-effect-free guard. */
(define sql_queryplan_formula_dispatch (lambda (variants prepared_catalog miss_expr)
	(match variants
		(cons variant rest) (begin
			(define new_preparations (filter (nth variant 2) (lambda (preparation)
				(if (prepared_catalog (car preparation))
					false
					(begin (prepared_catalog (car preparation) true) true)))))
			(define tail_expr (sql_queryplan_formula_dispatch rest prepared_catalog miss_expr))
			(define dispatch (if (equal? (car variant) true)
				(cadr variant)
				(list (quote if) (car variant) (cadr variant) tail_expr)))
			(if (empty_list? new_preparations)
				dispatch
				(cons (quote !begin)
					(merge (list (map new_preparations sql_queryplan_preparation_expr)
						(list dispatch))))))
		_ miss_expr)))

(define sql_queryplan_miss_expr (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy)
	(list (quote eval)
		(list (quote sql_queryplan_variant_miss)
			queryplan_cache cache_key entry parse_fn schema parse_query policy))))

(define sql_queryplan_formula (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy variants)
	(begin
		(define miss_expr (sql_queryplan_miss_expr
			queryplan_cache cache_key entry parse_fn schema parse_query policy))
		(sql_queryplan_formula_dispatch variants
			(make_structural_catalog (quote ast)) miss_expr))))

(define sql_queryplan_install_variants (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy variants)
	(begin
		(define recent_variants (sql_queryplan_recent_variants variants))
		(entry "variants" recent_variants)
		(entry "formula" (sql_queryplan_formula queryplan_cache cache_key entry parse_fn schema parse_query policy recent_variants))
		(entry "formula"))))

(define sql_queryplan_new_entry (lambda (queryplan_cache cache_key parse_fn schema parse_query policy source_session)
	(begin
		(define entry (newsession))
		(entry "compile_lock" (mutex))
		(define formula (sql_queryplan_install_variants queryplan_cache cache_key entry parse_fn schema parse_query policy
			(list (sql_compile_queryplan_variant parse_fn schema parse_query policy source_session))))
		(list entry formula))))

(define sql_queryplan_matching_variant (lambda (variants)
	(match variants
		(cons variant rest) (begin
			/* This request may hold an older formula while another request prepends a
			variant under the compile lock. Prepare that newly visible variant before
			rechecking its guard; preparation expressions are request-idempotent. */
			(map (nth variant 2) (lambda (preparation)
				(eval (sql_queryplan_preparation_expr preparation))))
			(if (eval (car variant)) variant (sql_queryplan_matching_variant rest)))
		_ nil)))

/* Called only by the final else branch of a cached plan. Recheck after taking
the entry lock because another request may already have installed a matching
variant. New variants are prepended so the most recent statistics regime wins
the common guard-dispatch path. */
(define sql_queryplan_variant_miss (lambda (queryplan_cache cache_key entry parse_fn schema parse_query policy)
	((entry "compile_lock") (lambda ()
		(begin
			(define current_variants (entry "variants"))
			(define matching (sql_queryplan_matching_variant current_variants))
			(if (not (nil? matching))
				(cadr matching)
				(begin
					(define variant (sql_compile_queryplan_variant parse_fn schema parse_query policy (context "session")))
					(define formula (sql_queryplan_install_variants queryplan_cache cache_key entry parse_fn schema parse_query policy
						(cons variant current_variants)))
					(queryplan_cache cache_key (list entry formula))
					(cadr variant))))))))

/* cached_parse: wraps SELECT planning with a lazy polymorphic Scheme plan
cache. DDL, DML and transaction-control formulas retain the original exact
cache path because their AST may intentionally operate on (context "session").
cache_key = username:schema:view-generation:hash(query-shape), retaining policy
isolation while sharing safe SELECT plans across literal variants. Each entry
is a variadic if chain of guarded specialized plans plus one compile miss arm.
On parse error the result is not cached. */
(define cached_parse (lambda (queryplan_cache parse_fn schema query policy username session parameterize_literals)
	(begin
		(define explain_query (match (toUpper query)
			(regex "^\\s*EXPLAIN\\b" _) true
			_ false))
		(define parameterized (if explain_query
			(list query '() (fnv_hash query))
			(sql_parameterize_select_literals query parameterize_literals)))
		(match parameterized '(parse_query bindings shape_hash) (begin
			(define cache_key (concat username ":" schema ":" (sql_view_query_generation parse_query) ":" shape_hash))
			(define select_query (match (toUpper parse_query)
				(regex "^\\s*SELECT\\b" _) true
				_ false))
			/* Polymorphic entries exist only where the planner can actually make a
			parameter/statistics-dependent physical choice. Ordinary point and
			ordered scans retain the smaller exact cache path. */
			(define guarded_select (and select_query
				(match (toUpper parse_query)
					(regex "\\b(?:LIKE|MATCH|JOIN|EXISTS)\\b" _) true
					_ false)))
			(define compile_diagnostic (match (toUpper parse_query)
				(regex "^\\s*EXPLAIN\\s+COMPILE\\b" _) true
				_ false))
			(if (not (equal? bindings '()))
				(reduce (produceN (count bindings)) (lambda (_ idx)
					(session (concat "v" (string (+ idx 1))) (nth bindings idx))) nil)
				nil)
			/* Compile diagnostics measure true misses and must not turn their own
			previous result into a cache hit. The inspected query is never run. */
			(define exact_compile (lambda ()
				(begin
					(define compile_policy (sql_compile_table_policy policy))
					(optimize (with_session session (lambda ()
						(parse_fn schema parse_query compile_policy)))))))
			(define formula (if (or compile_diagnostic (not guarded_select))
				(if compile_diagnostic
					(exact_compile)
					(queryplan_cache "get_or_compute" cache_key exact_compile))
				(begin
					(define cached_entry (queryplan_cache "get_or_compute" cache_key
						(lambda () (sql_queryplan_new_entry queryplan_cache cache_key parse_fn schema parse_query policy session))))
					(cadr cached_entry))))
			formula)))))

/* helper: build a policy function for table-level access checks
usage: create a policy by (set policy (sql_policy "username")),
then you can query the policy by
(policy "database" "tablename" false) for read
(policy "database" "tablename" true) for write
(policy "system" true true) to check for admin access like CREATE DATABASE, CREATE USER, DROP DATABASE, SHUTDOWN and so on
if everything is fine, the function call will do nothing.
if the user is not allowed to access this property, the function will throw an error and the query is aborted before it has run
*/
(define sql_policy (lambda (username)
	(begin
		(define is_admin (scan nil (table "system" "user")
			'("username") (lambda (u) (equal?? u username))
			'("admin") (lambda (a) a)
			(lambda (a b) (or a b))
			false))
		(if is_admin (lambda (schema tblname write) true) /* admin -> allow all */
			/* else: complicated policy */
			(lambda (schema tblname write)
				(begin
					/* Allow virtual INFORMATION_SCHEMA for all users */
					(if (equal?? schema "information_schema") true (begin
						/* Database-level check via system.access */
						(define access_count (scan nil (table "system" "access")
							'("username" "database") (lambda (u db) (and (equal?? u username) (equal?? db schema)))
							'() (lambda () 1)
							+ 0))
						(if (> access_count 0) true (error (concat "access denied: user '" username "' may not " (if write "write" "read") " " schema "." tblname)))
					))
			))
		)
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
	(insert (table "system" "user") '("username" "password" "admin") '('("root" (password (arg "root-password" "admin")) true)))
))

/* migration: older instances may miss the admin column; add it and mark all existing users as admin */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "admin")
			true
			(begin
				(createcolumn (table "system" "user") "admin" "boolean" '() '())
				(scan nil (table "system" "user") '() (lambda () true) '("$update") (lambda ($update) ($update '("admin" true))))
			)
		)
	) true)
)) (lambda (e) true))

/* migration: drop legacy id column that caused NOT NULL errors on CREATE USER */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "id")
			(migratedropcolumn (table "system" "user") "id")
			true)
	) true)
)) (lambda (e) true))

/* migration: ensure root always has admin=true */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(scan nil (table "system" "user") '("username") (lambda (username) (equal? username "root")) '("$update") (lambda ($update) ($update '("admin" true))))
		true)
)) (lambda (e) true))

/* ensure unique username constraint to avoid duplicates */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(createkey (table "system" "user") "uniq_username" true '("username"))
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
			(insert (table "system_statistic" "errors")
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

(try (lambda () (begin
	(if (has? (show "system") "views")
		(createkey (table "system" "views") "uniq_database_name" true '("database" "name"))
		true)
)) (lambda (e) true))

(sql_view_catalog_set_count
	(scan nil (table "system" "views") '() (lambda () true) '() (lambda () 1) + 0))

/* migration: ensure unique (username, database) constraint on system.access */
(try (lambda () (begin
	(if (has? (show "system") "access")
		(createkey (table "system" "access") "uniq_user_db" true '("username" "database"))
		true)
)) (lambda (e) true))

/* global variables exposed via @@ and SHOW VARIABLES */
(set globalvars (newsession))
(globalvars "lower_case_table_names" 0)
(globalvars "character_set_server" "utf8mb4")
(globalvars "collation_server" "utf8mb4_general_ci")
(globalvars "character_set_client" "utf8mb4")
(globalvars "character_set_results" "utf8mb4")
(globalvars "collation_connection" "utf8mb4_general_ci")
(globalvars "collation_database" "utf8mb4_general_ci")
(globalvars "sql_mode" "")
(globalvars "unique_checks" 1)
(globalvars "foreign_key_checks" 1)
(globalvars "sql_notes" 1)
(globalvars "time_zone" "UTC")
(globalvars "system_time_zone" (system_time_zone))

/* session_globalvar: reads from session first, falls back to globalvars.
Used for @@var resolution so per-session SET affects @@var reads. */
(define session_globalvar (lambda (key) (coalesceNil ((context "session") key) (globalvars key))))


/* persistent HTTP sessions for transaction support */
(set http_sessions (newsession))

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(define handle_query (lambda (req res schema query) (begin
		/* check for password */
		(set pw (scan nil (table "system" "user") '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password"))))
			(begin
				(try (lambda () (time (begin
					((res "header") "Content-Type" "text/event-stream; charset=utf-8")
					(define resultrow (res "jsonl"))
					/* Use persistent session if X-Session-Id header is present */
					(define session_id ((req "header") "X-Session-Id"))
					(define session (if session_id
						(begin
							(define existing (http_sessions session_id))
							(if existing existing (begin
								(define new_sess (newsession))
								(http_sessions session_id new_sess)
								new_sess
							))
						)
						(context "session")
					))
					(session "username" (req "username"))
					(session "schema" schema)
					/* Bind URL query params (v1=, v2=, ...) as prepared-statement args into the session
					before parse/build so session-sensitive planner rewrites see the right values. */
					(extract_assoc (req "query") (lambda (k v) (session k v)))
					(define formula (cached_parse sql_queryplan_cache parse_sql schema query (sql_policy (req "username")) (req "username") session true))
					(set resultrow_called false)
					(set original_resultrow resultrow)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(original_resultrow row))))
					/* Execute inside auto-commit tx (or existing explicit tx) */
					(set query_result (with_session session (lambda () (with_autocommit session (lambda () (eval (source "SQL Query" 1 1 formula)))))))
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
		(set pw (scan nil (table "system" "user") '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password"))))
			(begin
				(try (lambda () (time (begin
					((res "header") "Content-Type" "text/plain")
					(define resultrow (res "jsonl"))
					(define session (context "session"))
					(session "username" (req "username"))
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
					(define query_result (if handled nil (begin
						/* Bind URL query params (v1=, v2=, ...) as prepared-statement args into the session
						before parse/build so session-sensitive planner rewrites see the right values. */
						(extract_assoc (req "query") (lambda (k v) (session k v)))
						(define formula (cached_parse psql_queryplan_cache parse_psql schema query (sql_policy (req "username")) (req "username") session false))
						(with_autocommit session (lambda () (eval (source "SQL Query" 1 1 formula))))
					)))
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
	/* handler for raw Scheme code execution (global, no schema) */
	(define handle_scm (lambda (req res code) (begin
		/* check for password - must be admin */
		(set pw (scan nil (table "system" "user") '("username") (lambda (username) (equal? username (req "username"))) '("password" "admin") (lambda (password admin) (list password admin)) (lambda (a b) b) nil))
		(if (and pw (equal? (car pw) (password (req "password"))) (car (cdr pw)))
			(begin
				(try (lambda () (begin
					((res "header") "Content-Type" "application/json")
					(define session (context "session"))
					(session "username" (req "username"))
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
(set mysql_auth (lambda (username_) (scan nil (table "system" "user") '("username") (lambda (username) (equal? username username_)) '("password") (lambda (password) password) (lambda (a b) b) nil)))
(set mysql_schema (lambda (username schema) (or (equal?? schema "information_schema") (list? (show schema)))))
(set mysql_handler (lambda (schema sql resultrow_sql session) (begin
	(session "schema" schema)
	(define resultrow resultrow_sql)
	(try (lambda () (begin
		(if (equal? (session "syntax") "scheme") (begin
			/* scheme syntax mode */
			(set print (lambda args (resultrow '("result" (concat args)))))
			(resultrow '("result" (eval (scheme sql))))
		) (time (begin
				/* SQL syntax mode */
				/* tolerate an optional trailing ';' - must be at end of string */
				(set sql (match sql (regex "^((?s:.*));\\s*$" _ body) body sql))
				(define mysql_username (coalesce (session "username") "root"))
				(define formula (if (equal? (session "syntax") "postgresql")
					(cached_parse psql_queryplan_cache parse_psql schema sql (sql_policy mysql_username) mysql_username session false)
					(cached_parse sql_queryplan_cache parse_sql schema sql (sql_policy mysql_username) mysql_username session true)))
				(with_autocommit session (lambda () (eval (source "SQL Query" 1 1 formula))))
			) sql))
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
