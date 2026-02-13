/*
Copyright (C) 2023  Carl-Philip Hänsch

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
		(define is_admin (scan "system" "user"
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
						(define access_count (scan "system" "access"
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
	(eval (parse_sql "system" "CREATE TABLE `user`(id int, username text, password text, admin boolean DEFAULT FALSE) ENGINE=SAFE" (lambda (schema table write) true)))
	(insert "system" "user" '("id" "username" "password" "admin") '('(1 "root" (password (arg "root-password" "admin")) true)))
))

/* migration: older instances may miss the admin column; add it and mark all existing users as admin */
(try (lambda () (begin
	(if (has? (show "system") "user") (begin
		(if (has? (show "system" "user") "admin")
			true
			(begin
				(createcolumn "system" "user" "admin" "boolean" '() '())
				(scan "system" "user" '() (lambda () true) '("$update") (lambda ($update) ($update '("admin" true))))
			)
		)
	) true)
)) (lambda (e) true))

/* ensure unique username constraint to avoid duplicates */
(try (lambda () (begin
	(if (has? (show "system") "user")
		(createkey "system" "user" "uniq_username" true '("username"))
		true)
)) (lambda (e) true))

/* access control: which user can access which database */
(if (has? (show "system") "access") true (begin
	(print "creating table system.access")
	(eval (parse_sql "system" "CREATE TABLE `access`(username text, database text) ENGINE=SAFE" (lambda (schema table write) true)))
))

/* global variables exposed via @@ and SHOW VARIABLES */
(set globalvars (newsession))
(globalvars "lower_case_table_names" 0)
(globalvars "character_set_server" "utf8mb4")
(globalvars "collation_server" "utf8mb4_general_ci")

/* persistent HTTP sessions for transaction support */
(set http_sessions (newsession))

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(define handle_query (lambda (req res schema query) (begin
		/* check for password */
		(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password"))))
			(begin
				(try (lambda () (time (begin
					(define formula (parse_sql schema query (sql_policy (req "username"))))
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
					(set resultrow_called false)
					(set original_resultrow resultrow)
					(set last_row nil)
					(define resultrow (lambda (row) (begin
						(set resultrow_called true)
						(if (equal? row last_row)
							true
							(begin (set last_row row) (original_resultrow row))))))
					/* Execute with session in GLS so storage layer can access tx state */
					(set query_result (with_session session (lambda () (eval (source "SQL Query" 1 1 formula)))))
					/* If no resultrow was called and we got a number, return it as affected_rows */
					(if (and (not resultrow_called) (number? query_result)) (begin
						(original_resultrow '("affected_rows" query_result))
					))
				) query)) (lambda(e) (begin
						(print "SQL query: " query)
						(print "error: " e)
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
		(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password"))))
			(begin
				(try (lambda () (time (begin
					((res "header") "Content-Type" "text/plain")
					(define resultrow (res "jsonl"))
					(define session (context "session"))
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
						(define formula (parse_psql schema query (sql_policy (req "username"))))
						(eval (source "SQL Query" 1 1 formula))
					)))
					/* If no resultrow was called and we got a number, return it as affected_rows */
					(if (and (not resultrow_called) (number? query_result)) (begin
						(original_resultrow '("affected_rows" query_result))
					))
				) query)) (lambda(e) (begin
						(print "SQL query: " query)
						(print "error: " e)
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
		(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password" "admin") (lambda (password admin) (list password admin)) (lambda (a b) b) nil))
		(if (and pw (equal? (car pw) (password (req "password"))) (car (cdr pw)))
			(begin
				(try (lambda () (begin
					((res "header") "Content-Type" "application/json")
					(define session (context "session"))
					(set result (eval (scheme code)))
					((res "print") (json_encode result))
				)) (lambda(e) (begin
						(print "SCM code: " code)
						(print "error: " e)
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

/* dedicated mysql protocol listening at specified port */
(try (lambda () (begin
	(if (not (arg "disable-mysql" false)) (begin
		(set port (arg "mysql-port" (env "MYSQL_PORT" "3307")))
		(mysql port
			(lambda (username_) (scan "system" "user" '("username") (lambda (username) (equal? username username_)) '("password") (lambda (password) password) (lambda (a b) b) nil)) /* auth: load pw hash from system.user */
			(lambda (username schema) (or (equal?? schema "information_schema") (list? (show schema)))) /* allow virtual INFORMATION_SCHEMA, otherwise check db existence */
			(lambda (schema sql resultrow_sql session) (begin /* sql */
				(define resultrow resultrow_sql)
				(if (equal? (session "syntax") "scheme") /* TODO: check access to system.* */ (begin
					/* scheme syntax mode */
					(set print (lambda args (resultrow '("result" (concat args)))))
					(resultrow '("result" (eval (scheme sql))))
				) (time (begin
						/* SQL syntax mode */
						/* tolerate an optional trailing ';' - must be at end of string */
						(set sql (match sql (regex "^((?s:.*));\\s*$" _ body) body sql))
						(define formula ((if (equal? (session "syntax") "postgresql") (lambda (schema sql policy) (parse_psql schema sql policy)) (lambda (schema sql policy) (parse_sql schema sql policy))) schema sql (sql_policy (coalesce (session "username") "root"))))
						(eval (source "SQL Query" 1 1 formula))
					) sql))
			))
		)
		(print "MySQL server listening on port " port " (connect with `mysql -P " port " -u root -p` using password '" (arg "root-password" "admin") "'), set with --mysql-port")
	)) ; close the if for disable-mysql
)) print)
