/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Authentication names and credentials are reusable; an account incarnation
is not. This ordinary SAFE account column is shared by the SQL frontends and
carries no native grant or administrator authority. UUID-v4 exposes no host
identity, wall clock, operating-system information, or filesystem location. */
(define sql_initialize_account_incarnations (lambda () (begin
	/* Historical rows stay NULL: ordinary ADD freezes one default for history.
	Publish the future-row generator atomically with the nullable column, then
	give each historical account its own incarnation through a native mutation
	scan. Repeating initialization leaves existing incarnations untouched. */
	(createcolumn (table "system" false "user" false nil) "sql_principal_id" "VARCHAR" '(36)
		'("default_expression" "UUID_V4" "fill_existing" false))
	(scan nil (table "system" false "user" false nil) '() '()
		'("sql_principal_id") (lambda (id) (nil? id)) '("$update")
		(lambda (count update) (begin (update (list "sql_principal_id" (uuid_v4))) (+ count 1))) 0 +)
	true)))
(define sql_account_incarnation (lambda (username)
	(scan_lookup nil (table "system" false "user" false nil)
		(list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "sql_principal_id") (list username))))

/* Database creation is server administration. Only its existing-namespace
IF NOT EXISTS case is ordinary native database access. Retain an account
incarnation, never a cached privilege decision, in the executable declaration. */
(define sql_database_create_access (lambda (actor schema if_not_exists exists) (begin
	(if (nil? actor) true (begin
		(define username (car actor))
		(define id (cadr actor))
		(define current (sql_account_incarnation username))
		(if (and id (equal? id current)) true (error "access denied: authenticated login identity changed"))
		(define admin (scan_lookup nil (table "system" false "user" false nil)
			(list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "admin") (list username)))
		(define member (and if_not_exists (nth actor 2) exists (scan_lookup nil (table "system" false "access" false nil)
			(list 371635467059200 (scan_boundary "equal" "username" 0 0 true true "" false)
				(scan_boundary "equal" "database" 1 1 true true "" false)) (list username schema))))
		(if (and (or admin member) (equal? id (sql_account_incarnation username))) true
			(error "access denied: database creation requires native administrator or existing database access")))))))
(define sql_database_create_actor (lambda (actor schema if_not_exists) (begin
	/* The last scalar is source syntax, not a grant: some import dialects
	map plain CREATE SCHEMA onto an idempotent native database operation. */
	(define actor (if (list? actor) (list (car actor) (cadr actor) if_not_exists)
		(list actor (sql_account_incarnation actor) if_not_exists)))
	(sql_database_create_access actor schema if_not_exists (not (nil? (show schema))))
	actor)))
/* A nil/custom trusted policy keeps the privileged SCM caller contract.
Authenticated frontend policies return the pure actor tuple for runtime checks. */
(define sql_database_create_request? (lambda (name)
	(and (list? name) (equal? (count name) 2) (equal? (car name) "database-create"))))
(define sql_database_create_policy_result (lambda (authorization)
	(if (or (nil? authorization) (expression_equal? authorization true)) nil
		(and (list? authorization) (equal? (count authorization) 3)) authorization
		(error "invalid database creation policy result"))))
(define sql_database_create_authorization (lambda (policy schema if_not_exists)
	(if (nil? policy) nil
		(sql_database_create_policy_result (policy schema (list "database-create" if_not_exists) true)))))

/* Credential and incarnation are read from one native row. Password checking
runs after the read, so a scan callback never captures plaintext credentials.
A missing account, missing incarnation or wrong password returns NULL. */
(define sql_authenticate_account (lambda (username plaintext) (begin
	(define credentials (scan nil (table "system" false "user" false nil)
		(list 369436175368192 (scan_boundary "equal" "username" 0 0 true true "" false)) (list username)
		'() (lambda () true) '("password" "sql_principal_id" "admin" "username")
		(lambda (_stored hash id admin canonical) (list hash id admin canonical))
		nil (lambda (a b) (if (nil? a) b (nil? b) a (error "ambiguous login")))))
	(match credentials '(hash id admin canonical)
		(if (and id (equal? hash (password plaintext)))
			(list "principal_id" id "admin" admin "username" canonical) nil) _ nil))))
(define sql_authenticate_incarnation (lambda (username plaintext) (begin
	(define account (sql_authenticate_account username plaintext))
	(if account (account "principal_id") nil))))

/* Identity belongs to the session, not to a reusable login name or a SQL SET
variable. The existing scoped slot computes once, outside its native mutex.
HTTP supplies its validated incarnation; protocol SQL callers bind on their
first SQL statement. Neither path can replace a previously retained actor. */
(define sql_session_actor (lambda (session username authenticated_id) (begin
	(define actor (session "get_or_compute_scoped" sql_account_incarnation "sql-principal"
		(lambda () (begin
			(define id (coalesceNil authenticated_id (sql_account_incarnation username)))
			(if id (list username id) (error "access denied: authenticated login does not exist"))))))
	(if (and (equal? (car actor) username)
		(or (nil? authenticated_id) (equal? (cadr actor) authenticated_id))) actor
		(error "access denied: authenticated login identity changed")))))

/* Native session authority cannot be changed by URL parameters or SQL locals.
The authenticated account snapshot owns identity; credentials are never retained
as session keys or values. A recreated account must start a fresh session. */
(define sql_bind_http_session (lambda (req account) (begin
	(bind_session_principal (req "__session_state") "sql" (account "principal_id"))
	(sql_session_actor (req "__session") (account "username") (account "principal_id"))
	(req "__session"))))
(define sql_http_query_parameters (lambda (req session) (begin
	(extract_assoc (req "query") (lambda (key value)
		(if (or (regexp_test key "^__") (has? '("transaction" "syntax") key))
			(error "reserved session parameter") true)))
	(extract_assoc (req "query") (lambda (key value)
		(if (has? '("username" "schema") key) true (session key value)))))))

(sql_initialize_account_incarnations)
