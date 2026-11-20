/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* This frontend projects current native database-wide grants, never table
ACLs. Names and permissions are SQL Server declaration vocabulary; arbitrary
server capabilities use only the existing native administrator capability.
Other security-securable classes are not inferred from database membership.
https://learn.microsoft.com/sql/t-sql/functions/has-perms-by-name-transact-sql
SQL Server 2022 references: https://dbfiddle.uk/YKi4olqw and
https://dbfiddle.uk/fjsQnFv0. Class vocabulary: https://dbfiddle.uk/iVBtwE1m.
All51 server permissions and scope boundaries: https://dbfiddle.uk/MJkxE5pd. */
(define tsql_named_permission_vocabulary (list
	(list "SERVER" (list
		"ADMINISTER BULK OPERATIONS" "ALTER ANY AVAILABILITY GROUP" "ALTER ANY CONNECTION"
		"ALTER ANY CREDENTIAL" "ALTER ANY DATABASE" "ALTER ANY ENDPOINT"
		"ALTER ANY EVENT NOTIFICATION" "ALTER ANY EVENT SESSION" "ALTER ANY EVENT SESSION ADD EVENT"
		"ALTER ANY EVENT SESSION ADD TARGET" "ALTER ANY EVENT SESSION DISABLE" "ALTER ANY EVENT SESSION DROP EVENT"
		"ALTER ANY EVENT SESSION DROP TARGET" "ALTER ANY EVENT SESSION ENABLE" "ALTER ANY EVENT SESSION OPTION"
		"ALTER ANY LINKED SERVER" "ALTER ANY LOGIN" "ALTER ANY SERVER AUDIT"
		"ALTER ANY SERVER ROLE" "ALTER RESOURCES" "ALTER SERVER STATE"
		"ALTER SETTINGS" "ALTER TRACE" "AUTHENTICATE SERVER"
		"CONNECT ANY DATABASE" "CONNECT SQL" "CONTROL SERVER"
		"CREATE ANY DATABASE" "CREATE ANY EVENT SESSION" "CREATE AVAILABILITY GROUP"
		"CREATE DDL EVENT NOTIFICATION" "CREATE ENDPOINT" "CREATE LOGIN"
		"CREATE SERVER ROLE" "CREATE TRACE EVENT NOTIFICATION" "DROP ANY EVENT SESSION"
		"EXTERNAL ACCESS ASSEMBLY" "IMPERSONATE ANY LOGIN" "SELECT ALL USER SECURABLES"
		"SHUTDOWN" "UNSAFE ASSEMBLY" "VIEW ANY CRYPTOGRAPHICALLY SECURED DEFINITION"
		"VIEW ANY DATABASE" "VIEW ANY DEFINITION" "VIEW ANY ERROR LOG"
		"VIEW ANY PERFORMANCE DEFINITION" "VIEW ANY SECURITY DEFINITION" "VIEW SERVER PERFORMANCE STATE"
		"VIEW SERVER SECURITY AUDIT" "VIEW SERVER SECURITY STATE" "VIEW SERVER STATE"))
	(list "DATABASE" (list
		"ADMINISTER DATABASE BULK OPERATIONS" "ALTER" "ALTER ANY APPLICATION ROLE"
		"ALTER ANY ASSEMBLY" "ALTER ANY ASYMMETRIC KEY" "ALTER ANY CERTIFICATE"
		"ALTER ANY COLUMN ENCRYPTION KEY" "ALTER ANY COLUMN MASTER KEY" "ALTER ANY CONTRACT"
		"ALTER ANY DATABASE AUDIT" "ALTER ANY DATABASE DDL TRIGGER" "ALTER ANY DATABASE EVENT NOTIFICATION"
		"ALTER ANY DATABASE EVENT SESSION" "ALTER ANY DATABASE EVENT SESSION ADD EVENT" "ALTER ANY DATABASE EVENT SESSION ADD TARGET"
		"ALTER ANY DATABASE EVENT SESSION DISABLE" "ALTER ANY DATABASE EVENT SESSION DROP EVENT" "ALTER ANY DATABASE EVENT SESSION DROP TARGET"
		"ALTER ANY DATABASE EVENT SESSION ENABLE" "ALTER ANY DATABASE EVENT SESSION OPTION" "ALTER ANY DATABASE SCOPED CONFIGURATION"
		"ALTER ANY DATASPACE" "ALTER ANY EXTERNAL DATA SOURCE" "ALTER ANY EXTERNAL FILE FORMAT"
		"ALTER ANY EXTERNAL JOB" "ALTER ANY EXTERNAL LANGUAGE" "ALTER ANY EXTERNAL LIBRARY"
		"ALTER ANY EXTERNAL STREAM" "ALTER ANY FULLTEXT CATALOG" "ALTER ANY MASK"
		"ALTER ANY MESSAGE TYPE" "ALTER ANY REMOTE SERVICE BINDING" "ALTER ANY ROLE"
		"ALTER ANY ROUTE" "ALTER ANY SCHEMA" "ALTER ANY SECURITY POLICY"
		"ALTER ANY SENSITIVITY CLASSIFICATION" "ALTER ANY SERVICE" "ALTER ANY SYMMETRIC KEY"
		"ALTER ANY USER" "ALTER LEDGER" "ALTER LEDGER CONFIGURATION"
		"AUTHENTICATE" "BACKUP DATABASE" "BACKUP LOG"
		"CHECKPOINT" "CONNECT" "CONNECT REPLICATION"
		"CONTROL" "CREATE AGGREGATE" "CREATE ANY DATABASE EVENT SESSION"
		"CREATE ASSEMBLY" "CREATE ASYMMETRIC KEY" "CREATE CERTIFICATE"
		"CREATE CONTRACT" "CREATE DATABASE" "CREATE DATABASE DDL EVENT NOTIFICATION"
		"CREATE DEFAULT" "CREATE EXTERNAL LANGUAGE" "CREATE EXTERNAL LIBRARY"
		"CREATE FULLTEXT CATALOG" "CREATE FUNCTION" "CREATE MESSAGE TYPE"
		"CREATE PROCEDURE" "CREATE QUEUE" "CREATE REMOTE SERVICE BINDING"
		"CREATE ROLE" "CREATE ROUTE" "CREATE RULE"
		"CREATE SCHEMA" "CREATE SERVICE" "CREATE SYMMETRIC KEY"
		"CREATE SYNONYM" "CREATE TABLE" "CREATE TYPE"
		"CREATE USER" "CREATE VIEW" "CREATE XML SCHEMA COLLECTION"
		"DELETE" "DROP ANY DATABASE EVENT SESSION" "ENABLE LEDGER"
		"EXECUTE" "EXECUTE ANY EXTERNAL ENDPOINT" "EXECUTE ANY EXTERNAL SCRIPT"
		"INSERT" "KILL DATABASE CONNECTION" "REFERENCES"
		"SELECT" "SHOWPLAN" "SUBSCRIBE QUERY NOTIFICATIONS"
		"TAKE OWNERSHIP" "UNMASK" "UPDATE"
		"VIEW ANY COLUMN ENCRYPTION KEY DEFINITION" "VIEW ANY COLUMN MASTER KEY DEFINITION" "VIEW ANY SENSITIVITY CLASSIFICATION"
		"VIEW CRYPTOGRAPHICALLY SECURED DEFINITION" "VIEW DATABASE PERFORMANCE STATE" "VIEW DATABASE SECURITY AUDIT"
		"VIEW DATABASE SECURITY STATE" "VIEW DATABASE STATE" "VIEW DEFINITION"
		"VIEW LEDGER CONTENT" "VIEW PERFORMANCE DEFINITION" "VIEW SECURITY DEFINITION"))
	(list "OBJECT" (list "ALTER" "CONTROL" "DELETE" "EXECUTE" "INSERT" "RECEIVE" "REFERENCES" "SELECT" "TAKE OWNERSHIP" "UNMASK" "UPDATE" "VIEW CHANGE TRACKING" "VIEW DEFINITION"))
	(list "SCHEMA" (list "ALTER" "CONTROL" "CREATE SEQUENCE" "DELETE" "EXECUTE" "INSERT" "REFERENCES" "SELECT" "TAKE OWNERSHIP" "UNMASK" "UPDATE" "VIEW CHANGE TRACKING" "VIEW DEFINITION"))))
/* Parse a single quoted identifier with the existing complete-address owner.
The two fixed prefix parts make any additional unquoted dot invalid. Only
syntax conversion is caught; metadata/native failures retain their own cause. */
(define tsql_permission_single_name (close_procedure ((lambda (parse_name)
	(lambda (name) (if (nil? name) nil
		(try (lambda () (nth (parse_name (concat "__permission_db.__permission_owner." name)) 2))
			(lambda (_invalid_address) nil))))) tsql_object_name_parts)))
(define tsql_has_perms_value (close_procedure ((lambda (vocabulary permission parse_name single_name qualifier native_relation view_name module_definition module_kind system_procedure catalog_declarations legacy_name?)
	(lambda (context arguments actor tx options) (begin
		(define name (nth arguments 0))
		(define class (if (nil? (nth arguments 1)) nil (toUpper (regexp_replace (nth arguments 1) " +$" ""))))
		(define requested (if (nil? (nth arguments 2)) nil (toUpper (regexp_replace (nth arguments 2) " +$" ""))))
		(define sub (nth arguments 3))
		(define subclass (if (nil? (nth arguments 4)) nil (toUpper (regexp_replace (nth arguments 4) " +$" ""))))
		(define column? (or (not (nil? sub)) (not (nil? subclass))))
		/* SQL Server's server form is (NULL,NULL,permission). The explicit
		SERVER spelling instead needs a non-NULL name; it has the same held
		capabilities. Neither form accepts an object COLUMN sub-scope. */
		(define server? (if (nil? class) (nil? name)
			(and (equal? class "SERVER") (not (nil? name)))))
		(define lookup_class (if server? "SERVER" class))
		(define admitted (if lookup_class (find vocabulary (lambda (entry) (equal? (car entry) lookup_class)) nil) nil))
		(if (or (nil? requested)
			(if admitted (not (or (equal? requested "ANY") (has? (cadr admitted) requested))) false)) nil
			server? (if column? nil (if (permission actor context "NATIVE ADMIN") 1 0))
			(nil? admitted)
			(if (nil? class) nil
				(has? '("SERVER" "LOGIN" "USER" "ROLE" "TYPE" "ASSEMBLY" "AVAILABILITY GROUP" "CREDENTIAL" "EXTERNAL LANGUAGE" "CERTIFICATE" "SYMMETRIC KEY" "ASYMMETRIC KEY" "XML SCHEMA COLLECTION" "SERVICE" "CONTRACT" "MESSAGE TYPE" "REMOTE SERVICE BINDING" "ROUTE" "FULLTEXT CATALOG" "FULLTEXT STOPLIST" "SEARCH PROPERTY LIST" "DATABASE SCOPED CREDENTIAL" "APPLICATION ROLE" "ENDPOINT" "SERVER ROLE") class)
				(error "HAS_PERMS_BY_NAME server/security securable profile is not implemented") nil)
			(and column? (not (and (equal? class "OBJECT") (not (nil? sub)) (equal? subclass "COLUMN")
				(has? '("SELECT" "UPDATE" "REFERENCES" "UNMASK") requested)))) nil
			(equal? class "DATABASE") (begin
				(define target (if (nil? name) context (single_name name)))
				(define database (if target (database_name (qualifier target context options) true) nil))
				(if database (if (permission actor database (if (equal? requested "CREATE DATABASE") "NATIVE ADMIN" "DATABASE ACCESS")) 1 0) 0))
			(nil? name) nil
			(equal? class "SCHEMA")
			(if (and (has? '("dbo" "sys" "information_schema") (toLower (coalesceNil (single_name name) "")))
				(permission actor context "DATABASE ACCESS")) 1 0)
			(begin
				(define parts (try (lambda () (parse_name name)) (lambda (_invalid_address) nil)))
				(define database (if parts (database_name (qualifier (car parts) context options) true) nil))
				(if (or (nil? database) (not (permission actor database "DATABASE ACCESS"))) 0 (begin
					(define owner (cadr parts))
					(define object (nth parts 2))
					(define system? (equal?? owner "sys"))
					(define dbo? (equal?? owner "dbo"))
					(define native (if dbo? (native_relation database object tx) nil))
					(define view (if dbo? (view_name database object tx) nil))
					(define module (if dbo? (module_definition database object tx) nil))
					(define builtin (if (or system? (and dbo? (nil? native) (nil? view) (nil? module)))
						(system_procedure object) nil))
					(define catalog (if (or system? (and dbo? (legacy_name? object)))
						(find catalog_declarations (lambda (entry) (equal?? (car entry) object)) nil) nil))
					(define kind (if native "U" view "V" module (module_kind (module "language"))
						builtin (cadr builtin) catalog "V" nil))
					(if (nil? kind) 0
						(if builtin (not (permission actor database (nth builtin 3))) false) 0
						column? (begin
							(define column (single_name sub))
							(define columns (if native ((show native "declarations" tx) "columns")
								catalog (map (cadr catalog) (lambda (declaration) (list "Field" (car declaration))))
								view (error "HAS_PERMS_BY_NAME stored view column declarations are not implemented") '()))
							(if (if column (find columns (lambda (declaration) (equal?? (declaration "Field") column)) nil) false) 1 0))
						(permission actor database "NATIVE ADMIN") 1
						(or (equal? requested "ANY") (has? '("ALTER" "CONTROL" "TAKE OWNERSHIP" "VIEW DEFINITION") requested)) 1
						(has? '("U" "V" "TF" "IF" "FT") kind)
						(if (has? '("SELECT" "INSERT" "UPDATE" "DELETE" "REFERENCES" "UNMASK" "VIEW CHANGE TRACKING") requested) 1 0)
						(has? '("FN" "FS") kind) (if (has? '("EXECUTE" "REFERENCES") requested) 1 0)
						(has? '("P" "PC" "X" "RF") kind) (if (equal? requested "EXECUTE") 1 0)
						0))))))))
	tsql_named_permission_vocabulary tsql_permission? tsql_object_name_parts tsql_permission_single_name
	tsql_qualifier_database tsql_native_relation_handle tsql_view_name tsql_catalog_module_definition
	tsql_source_module_kind tsql_system_procedure_declaration
	tsql_virtual_catalog_declarations tsql_legacy_catalog_name?)))
/* Fresh invocation belongs to this authoritative metadata producer. Repeated
arguments do not freeze grants, account incarnations or current declarations.
Only actual scalar names/actor/native tx enter its result formula. */
(define tsql_has_perms_query (lambda (database arguments _planning _tx) (begin
	(define widths '(128 60 128 128 60))
	(define converted (map (produceN (count arguments)) (lambda (i)
		(tsql_implicit_cast_expr (nth arguments i) "NVARCHAR" (list (nth widths i))))))
	(define values (merge (list converted (map (produceN (- 5 (count converted))) (lambda (_i) nil)))))
	(list 'inner_select (make_query_block database '()
		(list "value" (list 'sql_type_expr (list tsql_has_perms_value database (cons 'list values)
			(tsql_actor_formula) 'tx (list 'list "tsql_import_target" (list 'session "tsql_import_target")
				"tsql_import_source" (list 'session "tsql_import_source"))) "INT" nil))
		true nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))))
(define tsql_has_perms_call (lambda (database arguments) (begin
	(if (and (>= (count arguments) 3) (<= (count arguments) 5)) true
		(error "HAS_PERMS_BY_NAME accepts three to five arguments"))
	(list 'sql_type_expr (list 'tsql_relational_scalar_call
		(close_procedure ((lambda (database lookup)
			(lambda (arguments _outer_sources _policy planning tx)
				(lookup database arguments planning tx))) database tsql_has_perms_query)) arguments) "INT" nil))))
(register_queryplan_callables (list tsql_has_perms_value))
