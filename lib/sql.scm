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
(import "queryplan.scm")

/* query plan caches: separate cachemap per parser dialect */
(set sql_queryplan_cache (newcachemap))
(set psql_queryplan_cache (newcachemap))

/* cached_parse: wraps a parser with cachemap-based caching.
cache_key = username:schema:hash(query) — per-user isolation (policy checked at parse time).
The query is hashed with FNV-1a (fnv_hash) so long SQL strings don't bloat the cache index.
Session-sensitive plans must not be reused under that key because their lowered
runtime helper names and cache domains may depend on current session variables.
On parse error the result is not cached (e.g. table does not exist yet). */
(define cached_parse (lambda (queryplan_cache parse_fn schema query policy username session)
	(begin
		(define cache_key (concat username ":" schema ":" (fnv_hash query)))
		(define cached (queryplan_cache cache_key))
		(if cached cached
			(begin
				(define formula (with_session session (lambda () (parse_fn schema query policy))))
				(queryplan_cache cache_key formula)
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
		(define is_admin (scan nil "system" "user"
			'("username") (lambda (u) (equal?? u username))
			'("admin") (lambda (a) a)
			(lambda (a b) (or a b))
			false))
		(if is_admin (lambda (schema table write) true) /* admin -> allow all */
			/* else: complicated policy */
			(lambda (schema table write)
				(begin
					/* Allow virtual INFORMATION_SCHEMA for all users */
					(if (equal?? schema "information_schema") true (begin
						/* Database-level check via system.access */
						(define access_count (scan nil "system" "access"
							'("username" "database") (lambda (u db) (and (equal?? u username) (equal?? db schema)))
							'() (lambda () 1)
							+ 0))
						(if (> access_count 0) true (error (concat "access denied: user '" username "' may not " (if write "write" "read") " " schema "." table)))
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
	(eval (parse_sql "system" "CREATE TABLE `user`(username text, password text, admin boolean DEFAULT FALSE) ENGINE=SAFE" (lambda (schema table write) true)))
	(insert "system" "user" '("username" "password" "admin") '('("root" (password (arg "root-password" "admin")) true)))
))

/* migration: older instances may miss the admin column; add it and mark all existing users as admin */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "admin")
			true
			(begin
				(createcolumn "system" "user" "admin" "boolean" '() '())
				(scan nil "system" "user" '() (lambda () true) '("$update") (lambda ($update) ($update '("admin" true))))
			)
		)
	) true)
)) (lambda (e) true))

/* migration: drop legacy id column that caused NOT NULL errors on CREATE USER */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (map (show "system" "user") (lambda (col) (get_assoc col "Field"))) "id")
			(dropcolumn "system" "user" "id")
			true)
	) true)
)) (lambda (e) true))

/* migration: ensure root always has admin=true */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(scan nil "system" "user" '("username") (lambda (username) (equal? username "root")) '("$update") (lambda ($update) ($update '("admin" true))))
		true)
)) (lambda (e) true))

/* ensure unique username constraint to avoid duplicates */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(createkey "system" "user" "uniq_username" true '("username"))
		true)
)) (lambda (e) true))

/* error query log table */
(if (not (has? (show "system_statistic") "errors")) (begin
	(print "creating table system_statistic.errors")
	(eval (parse_sql "system_statistic" "CREATE TABLE errors(datetime text, database text, user text, query text, error text) ENGINE=SLOPPY" (lambda (schema table write) true)))
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
			(insert "system_statistic" "errors"
				'("datetime" "database" "user" "query" "error")
				(list (list (now) db usr qry (concat errmsg))))
			/* trimming moved to 15-minute cron in dashboard.scm */
		)) (lambda (e) true)) /* silently ignore logging errors to avoid infinite recursion */
	) true)
)))

/* print log table */
(if (not (has? (show "system_statistic") "logs")) (begin
	(print "creating table system_statistic.logs")
	(eval (parse_sql "system_statistic" "CREATE TABLE logs(datetime text, message text) ENGINE=SLOPPY" (lambda (schema table write) true)))
))

/* access control: which user can access which database */
(if (has? (show "system") "access") true (begin
	(print "creating table system.access")
	(eval (parse_sql "system" "CREATE TABLE `access`(username text, database text) ENGINE=SAFE" (lambda (schema table write) true)))
))

/* migration: ensure unique (username, database) constraint on system.access */
(try (lambda () (begin
	(if (has? (show "system") "access")
		(createkey "system" "access" "uniq_user_db" true '("username" "database"))
		true)
)) (lambda (e) true))

/* global variables exposed via @@ and SHOW VARIABLES */
(set globalvars (newsession))
(globalvars "lower_case_table_names" 0)
(globalvars "character_set_server" "utf8mb4")
(globalvars "collation_server" "utf8mb4_general_ci")
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
		(set pw (scan nil "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
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
					(define formula (cached_parse sql_queryplan_cache parse_sql schema query (sql_policy (req "username")) (req "username") session))
					(set resultrow_called false)
					(set original_resultrow resultrow)
					(set last_row nil)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(if (equal? row last_row)
							true
							(begin (set last_row row) (original_resultrow row))))))
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
		(set pw (scan nil "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
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
					(set last_row nil)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(if (equal? row last_row)
							true
							(begin (set last_row row) (original_resultrow row))))))
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
						(define formula (cached_parse psql_queryplan_cache parse_psql schema query (sql_policy (req "username")) (req "username") session))
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
		(set pw (scan nil "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password" "admin") (lambda (password admin) (list password admin)) (lambda (a b) b) nil))
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
(set mysql_auth (lambda (username_) (scan nil "system" "user" '("username") (lambda (username) (equal? username username_)) '("password") (lambda (password) password) (lambda (a b) b) nil)))
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
					(cached_parse psql_queryplan_cache parse_psql schema sql (sql_policy mysql_username) mysql_username session)
					(cached_parse sql_queryplan_cache parse_sql schema sql (sql_policy mysql_username) mysql_username session)))
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
