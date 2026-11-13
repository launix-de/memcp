/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* T-SQL projects the native database-only account model. Compatibility GRANT,
REVOKE, DENY and owner requests never create accounts, memberships or extra
rights. Native server administration remains a distinct native capability. */
(define tsql_account_id sql_account_incarnation)
(define tsql_native_admin? (lambda (username)
	(if (scan_lookup nil (table "system" false "user" false nil)
		(list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "admin") (list username)) true false)))
(define tsql_native_member? (lambda (username database)
	(if (scan_lookup nil (table "system" false "access" false nil)
		(list 371635467059200 (scan_boundary "equal" "username" 0 0 true true "" false)
			(scan_boundary "equal" "database" 1 1 true true "" false)) (list username database)) true false)))
/* Row expressions pass only these declared identity values. Passing the full
session as an opaque helper argument would publish unrelated parser/planning
state instead of the two-key view needed by native database authorization. */
(define tsql_actor_value (close_procedure ((lambda (account_id)
	(lambda (username incarnation)
		(list username (coalesceNil incarnation (account_id username))))) tsql_account_id)))
(define tsql_actor_formula (lambda ()
	(list tsql_actor_value (list 'session "username") (list 'session "__tsql_principal_id"))))
(define tsql_session_actor (close_procedure ((lambda (actor_value)
	(lambda (session)
		(actor_value (session "username") (session "__tsql_principal_id")))) tsql_actor_value)))
(define tsql_permission? (close_procedure ((lambda (account_id native_admin native_member)
	(lambda (actor database action) (begin
		(define username (if (list? actor) (car actor) actor))
		(define id (if (list? actor) (cadr actor) (account_id username)))
		(define allowed (and (not (nil? id))
			(if (equal? action "NATIVE ADMIN") (native_admin username)
				(or (native_admin username) (native_member username database)))))
		/* Reused login names cannot authorize an old authenticated connection. */
		(and allowed (equal? id (account_id username))))))
	tsql_account_id tsql_native_admin? tsql_native_member?)))
(define tsql_require_permission (close_procedure ((lambda (permission)
	(lambda (actor database action) (begin
		(define database (database_name database true))
		(if (nil? database) (error "database does not exist") true)
		(if (permission actor database action) true
			(error "access denied: " (if (equal? action "NATIVE ADMIN")
				"native administrator required" (concat "database " database))))))) tsql_permission?)))
(define tsql_require_database_access (close_procedure ((lambda (require_permission)
	(lambda (actor database) (require_permission actor database "DATABASE ACCESS"))) tsql_require_permission)))
(define tsql_permission_database (lambda (session) (begin
	(define database (database_name (session "schema") true))
	(if database database (error "database does not exist")))))

/* Syntactically valid access requests are compatibility no-ops. In particular,
unknown objects/principals are not resolved, and no native membership is added. */
(define tsql_change_permissions (lambda (_mode _actions _principals session _tx)
	(begin (tsql_require_database_access (tsql_session_actor session)
		(tsql_permission_database session)) 0)))
(define tsql_change_database_owner (lambda (arguments statement caller) (begin
	(define declarations (list
		(list "loginame" (tsql_parameter_declaration (caller "schema") "SYSNAME" '()) nil false)
		(list "map" (tsql_parameter_declaration (caller "schema") "VARCHAR" '(30)) "NULL" false)))
	(define binding (tsql_bind_call arguments declarations statement caller))
	(define database (tsql_permission_database caller))
	(define login (cadr (find (car binding) (lambda (item) (equal? (car item) "loginame")) nil)))
	(if (nil? login) (error "database owner login cannot be NULL") true)
	(statement "database ownership compatibility" false (lambda (_tx) (begin
		(tsql_require_database_access (tsql_session_actor caller) database)
		0))))))

/* Common policy retains intrinsic private-relation admission and checks only
native database access. Server operations request their actual admin capability. */
(define tsql_policy (lambda (actor)
	(lambda (database name write)
		(if (sql_database_create_request? name)
			(sql_database_create_actor actor database (cadr name))
			(sql_private_relation? name) (error "access denied to private SQL relation")
			(equal?? database "information_schema") true
			(and (equal? database "system") (expression_equal? name true) write)
			(tsql_require_permission actor "system" "NATIVE ADMIN")
			(tsql_require_database_access actor database)))))
(define tsql_record_permission (lambda (planning database action) (begin
	(define actor (list (coalesceNil (planning "username") "root")
		(coalesceNil (planning "__tsql_principal_id") (tsql_account_id (coalesceNil (planning "username") "root")))))
	(define access (if (equal? action "NATIVE ADMIN") "NATIVE ADMIN" "DATABASE ACCESS"))
	(tsql_require_permission actor database access)
	(define requirement (list actor database access))
	(define previous (coalesceNil (planning "__tsql_database_access_requirements") '()))
	(if (has? previous requirement) true
		(planning "__tsql_database_access_requirements" (cons requirement previous)))
	true)))
(define tsql_permission_formula (lambda (formula planning)
	(cons (quote !begin) (append
		(map (coalesceNil (planning "__tsql_database_access_requirements") '()) (lambda (requirement)
			(list tsql_require_permission (list (quote quote) (car requirement)) (cadr requirement) (nth requirement 2)))) formula))))

/* An authenticated connection owns its login incarnation. Reusing a dropped
login name never transforms an already-open connection into the replacement. */
(define tsql_validate_login_identity (lambda (session) (begin
	(define current (tsql_account_id (session "username")))
	(define expected (session "__tsql_principal_id"))
	(if (nil? current) (error "access denied: authenticated login no longer exists") true)
	(if (nil? expected) (session "__tsql_principal_id" current)
		(equal? expected current) true (error "access denied: authenticated login identity changed"))
	true)))

(define tsql_authenticate sql_authenticate_incarnation)

/* Metadata retains the immutable actor and the same native database gate. */
(define tsql_metadata_policy (close_procedure ((lambda (require_access)
	(lambda (actor) (close_procedure (lambda (database _name _write)
		(require_access actor database))))) tsql_require_database_access)))
(define tsql_metadata_visible? (close_procedure ((lambda (permission)
	(lambda (actor database _declarations)
		(permission actor database "DATABASE ACCESS"))) tsql_permission?)))
