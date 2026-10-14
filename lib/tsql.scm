/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-parser.scm")
(define tsql_queryplan_cache (newcachemap))
(define tsql_statement_text strtrim)

/* Transport tuples describe bytes and unscaled coefficients only. Their SQL
meaning is chosen here from the request's declaration, not in the Go bridge. */
(define tds_parameter_value (lambda (parameter) (begin
	(define value (parameter "value"))
	(define wire (parameter "type"))
	(if (nil? value) nil
		(has? '(106 108) (wire "kind")) (begin
			(define coefficient (value "coefficient"))
			(define scale (wire "scale"))
			(if (and (string? coefficient) (regexp_test coefficient "^-?(?:0|[1-9][0-9]*)$")
				(>= scale 0) (<= scale (wire "precision")) (<= (wire "precision") 38)) true
				(error "invalid numeric parameter payload"))
			(if (equal? scale 0) (tsql_parse_integer coefficient) (begin
				(define negative (equal? (substr coefficient 0 1) "-"))
				(define digits (if negative (substr coefficient 1) coefficient))
				(define padded (concat (string_repeat "0" (max 0 (+ 1 (- scale (strlen digits))))) digits))
				(define point (- (strlen padded) scale))
				(tsql_numeric_input (concat (if negative "-" "") (substr padded 0 point) "." (substr padded point))))))
		value))))
(define tds_result_value (lambda (descriptor value) (if (nil? value) nil
	(match (descriptor "kind")
		165 (list "bytes" value)
		108 (begin
			(define scale (descriptor "scale"))
			/* A zero-scale SQL contract keeps native integer IDs exact. Scaled
			coefficients currently use the existing native numeric precision. */
			(define coefficient (if (equal? scale 0) value
				(round (* value (simplify (concat "1e" scale))))))
			(list "coefficient" (string (tsql_cast_integer coefficient
				(tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807")))))
		231 (string value)
		109 (list "floating" value)
		38 (intdiv value 1)
		_ value))))

/* Bind at the request boundary, not in storage or row loops. NULL is a supplied
value; the wire descriptor and SQL declaration own the type independently. */
(define tds_bind_parameters (lambda (session declarations parameters) (begin
	(if (not (equal? (count declarations) (count parameters)))
		(error "T-SQL parameter count differs from declarations") true)
	(define names (map declarations car))
	(if (not (equal? (count names) (count (merge_unique (list names)))))
		(error "duplicate T-SQL parameter declaration") true)
	(define named (merge (map (produceN (count parameters)) (lambda (i) (begin
		(define parameter (nth parameters i))
		(define name (parameter "name"))
		(define name (if (equal? name "") (car (nth declarations i))
			(toLower (if (equal? (substr name 0 1) "@") (substr name 1) name))))
		(if (or (not (has? names name)) (not (equal? (parameter "flags") 0)))
			(error "unknown or unsupported output parameter: @" name) true)
		(list name parameter))))))
	(if (not (equal? (count parameters) (count (merge_unique (list (extract_assoc named (lambda (name _value) name)))))))
		(error "duplicate T-SQL parameter value") true)
	(define values (map declarations (lambda (declaration) (begin
		(define name (car declaration))
		(define contract (cadr declaration))
		(define parameter (named name))
		(define wire (parameter "type"))
		(if (not (has? '(31 38 104 106 108 109 165 167 173 175 231 239) (wire "kind")))
			(error "unsupported T-SQL parameter wire type: " (wire "kind")) true)
		/* A supplied NULL needs no numeric representation or conversion. Its SQL
		contract still comes exclusively from the declaration. */
		(define source_type (match (wire "kind")
			38 "BIGINT" 104 "BIT" 106 "DECIMAL" 108 "NUMERIC" 109 "FLOAT"
			165 "VARBINARY" 173 "BINARY" 167 "VARCHAR" 175 "CHAR" 231 "NVARCHAR" 239 "NCHAR" _ "NULL"))
		(if (and (not (nil? (parameter "value"))) (tsql_binary_type? source_type)
			(not (tsql_binary_type? (car contract))))
			(error "binary parameter conversion is not implemented") true)
		(define converted (if (nil? (parameter "value")) nil
			(eval (sql_type_lower_annotations
				(tsql_cast_expr (list (quote sql_type_expr) (list (quote quote) (tds_parameter_value parameter)) source_type nil)
					(car contract) (cdr contract))))))
		(list (concat "__tsql_parameter:" name) converted)))))
	/* All declarations and values are validated before installing this request. */
	(map values (lambda (item) (session (car item) (cadr item))))
	(session "__tsql_parameter_declarations" declarations)
	true)))

/* Description parses and resolves SQL contracts without running the compiled
statement. SELECT bypasses physical planning; mutations are refused here until
the parser exposes a pure metadata path for their result contract. */
(define tds_describe (lambda (schema text declarations session) (begin
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" (tds_bind_declarations schema declarations))
	(planning "__tsql_describe" true)
	(match (parse_tsql schema (tsql_statement_text text) (sql_policy (coalesceNil (session "username") "root")) planning nil)
		((symbol quote) fields) fields
		_ (error "unsupported T-SQL describe shape")))))

/* A separate frontend entrypoint keeps dialect decisions out of existing SQL
execution paths. The cache, transaction owner and compiled evaluator are shared. */
(define tsql_execute_statement (lambda (tx schema text session resultrow resultfields) (begin
	/* Expose the active scope's scalar value through the ordinary session binding.
	The common planner can then discover and bind this dependency like parameters;
	an opaque scope-reader call would hide it from shared cache/domain analysis. */
	(session "__tsql_scope_identity" ((tsql_identity_scope session) "__tsql_scope_identity"))
	(define input (tsql_statement_text text))
	(define username (coalesce (session "username") "root"))
	(define declarations (coalesceNil (session "__tsql_parameter_declarations") (list)))
	/* Cache identity includes the complete immutable declaration contract. Values
	remain invocation inputs; differently typed calls cannot reuse wrong metadata. */
	(define cache_scope (json_encode (list username declarations)))
	(define parser (lambda (database text policy planning tx) (begin
		(planning "__tsql_parameter_declarations" declarations)
		(parse_tsql database text policy planning tx))))
	(define formula (cached_parse tsql_queryplan_cache (list parser) schema input
		(list (quote sql-policy-for) username) cache_scope session false tx))
	(sql_execute_formula session tx formula resultrow resultfields))))

(define tsql_http_query (lambda (req res schema) (begin
	(define stored (mysql_auth (req "username")))
	(if (and stored (equal? stored (password (req "password"))))
		(begin
			(define text ((req "body")))
			(try (lambda () (time (begin
				((res "header") "Content-Type" "text/event-stream; charset=utf-8")
				(define session (req "__session"))
				(extract_assoc (req "query") (lambda (key value) (session key value)))
				/* Authentication and the route own identity and database selection;
				query options may supply values, never replace either authority. */
				(session "username" (req "username"))
				(session "schema" schema)
				/* One scope owner per submitted batch, never per native insert row. */
				(session "__tsql_scope" (newsession))
				(define output (newsession))
				(define result (with_autocommit session (req "__session_state") (req "__query_seq") text
					(lambda (tx) (tsql_execute_statement tx schema text session
						(lambda (values) (begin (output "rows" true) ((res "jsonl") values)))
						(lambda args true)))))
				(if (and (not (output "rows")) (number? result))
					((res "jsonl") (list "affected_rows" result)) true)
			) text)) (lambda (err) (begin
					(error_log (concat err) schema (req "username") text)
					((res "header") "Content-Type" "text/plain")
					((res "status") 500)
					((res "print") "SQL Error: " err)))))
		(begin
			((res "header") "Content-Type" "text/plain")
			((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
			((res "status") 401)
			((res "print") "Unauthorized"))))))
(define http_handler (begin
	(define previous http_handler)
	(lambda (req res) (match (req "path")
		(regex "^/tsql/([^/]+)$" _ schema) (tsql_http_query req res schema)
		_ (previous req res)))))
(service_registry "T-SQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/tsql/[database]" "POST, NDJSON"))

/* Driver catalogs adapt existing declarations and constraints. They never
inspect user rows, enumerate automatic access indexes, or retain a second catalog.
Descriptor types below are frontend contracts, including for empty results. */
(define tsql_metadata_arguments (lambda (arguments names) (begin
	(define bind (lambda (remaining index values) (match remaining
		(cons argument rest) (begin
			(define supplied (toLower (car argument)))
			(define supplied (if (and (> (strlen supplied) 0) (equal? (substr supplied 0 1) "@")) (substr supplied 1) supplied))
			(define name (if (equal? supplied "")
				(if (< index (count names)) (nth names index) (error "too many metadata arguments")) supplied))
			(if (has? names name) true (error "unknown metadata argument: " name))
			(if (has? (extract_assoc values (lambda (key _value) key)) name)
				(error "duplicate metadata argument") true)
			(bind rest (+ index 1) (append values name (cadr argument))))
		_ values)))
	(bind arguments 0 '()))))
(define tsql_metadata_match (lambda (name pattern patterns)
	(if (nil? pattern) true
		patterns (begin
			/* ODBC's literal wildcard spellings use the existing LIKE engine. */
			(define escaped (replace (replace pattern "[_]" "\\_") "[%]" "\\%"))
			(if (regexp_test escaped "\\[") (error "unsupported metadata bracket pattern")
				(strlike name escaped "utf8mb4_general_ci")))
		(equal?? name pattern))))
(define tsql_metadata_table_names (lambda (schema pattern patterns policy) (begin
	(if policy (policy schema true false) true)
	(filter (coalesceNil (show schema) '()) (lambda (name)
		(and (not (internal_column_name? name)) (tsql_metadata_match name pattern patterns)))))))
(define tsql_metadata_descriptors (lambda (names integers)
	(map names (lambda (name)
		(if (has? integers name) (list "name" name "kind" 38 "size" 4 "flags" 1)
			(list "name" name "kind" 231 "size" 8000 "flags" 1))))))
/* (type name, ODBC code, precision, byte length, scale, radix, datetime subtype).
Only declarations decide these values: an empty or all-NULL table is identical. */
(define tsql_metadata_type (lambda (column version) (begin
	(define dimensions (column "Dimensions"))
	(define length (if (empty_list? dimensions) 1 (car dimensions)))
	(define type (toLower (column "RawType")))
	(match type
		"bigint" '("bigint" -5 19 8 0 10 nil)
		"int" '("int" 4 10 4 0 10 nil)
		"integer" '("int" 4 10 4 0 10 nil)
		"smallint" '("smallint" 5 5 2 0 10 nil)
		"tinyint" '("tinyint" -6 3 1 0 10 nil)
		"boolean" '("bit" -7 1 1 0 nil nil)
		"bit" '("bit" -7 1 1 0 nil nil)
		"real" '("real" 7 24 4 nil 2 nil)
		"float" '("float" 6 53 8 nil 2 nil)
		"double" '("float" 6 53 8 nil 2 nil)
		"date" (list "date" (if (equal? version 2) 9 91) 10 6 nil nil 1)
		"datetime" (list "datetime" (if (equal? version 2) 11 93) 23 16 3 nil 3)
		"smalldatetime" (list "smalldatetime" (if (equal? version 2) 11 93) 16 16 0 nil 3)
		"rowversion" '("rowversion" -2 8 8 nil nil nil)
		"timestamp" '("rowversion" -2 8 8 nil nil nil)
		_ (if (has? '("decimal" "numeric") type)
			(begin
				(define precision (if (empty_list? dimensions) 18 (car dimensions)))
				(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
				(list type (if (equal? type "decimal") 3 2) precision
					(if (<= precision 9) 5 (<= precision 19) 9 (<= precision 28) 13 17) scale 10 nil))
			(has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary") type)
			(list type (match type "varchar" 12 "char" 1 "nvarchar" -9 "nchar" -8 "binary" -2 "varbinary" -3)
				length (if (and (> length 0) (has? '("nvarchar" "nchar") type)) (* length 2) length) nil nil nil)
			(error "unsupported metadata declaration type: " type))))))
(define tsql_metadata_column_row (lambda (schema name column ordinal version) (begin
	(define contract (tsql_metadata_type column version))
	(define subtype (nth contract 6))
	(define type (car contract))
	(list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name
		"COLUMN_NAME" (column "Field") "DATA_TYPE" (nth contract 1) "TYPE_NAME" type
		"PRECISION" (nth contract 2) "LENGTH" (nth contract 3) "SCALE" (nth contract 4)
		"RADIX" (nth contract 5) "NULLABLE" (if (column "Null") 1 0)
		"REMARKS" (if (equal? (column "Comment") "") nil (column "Comment"))
		"COLUMN_DEF" (if (equal? (column "DefaultExpression") "") (column "Default") (column "DefaultExpression"))
		"SQL_DATA_TYPE" (if subtype 9 (nth contract 1)) "SQL_DATETIME_SUB" subtype
		"CHAR_OCTET_LENGTH" (if (has? '("varchar" "char" "nvarchar" "nchar" "binary" "varbinary") type) (nth contract 3) nil)
		"ORDINAL_POSITION" ordinal "IS_NULLABLE" (if (column "Null") "YES" "NO")))))
/* Type info is a frontend declaration catalog, not a registry of runtime tags. */
(define tsql_metadata_type_info (lambda (options version) (begin
	(define definitions (list (list "bigint" '()) (list "int" '()) (list "smallint" '()) (list "tinyint" '())
		(list "bit" '()) (list "real" '()) (list "float" '()) (list "decimal" '(38 0)) (list "numeric" '(38 0))
		(list "varchar" '(8000)) (list "char" '(8000)) (list "date" '())))
	(define rows (map definitions (lambda (definition) (begin
		(define contract (tsql_metadata_type (list "RawType" (car definition) "Dimensions" (cadr definition)) version))
		(define type (car contract))
		(define character (has? '("varchar" "char") type))
		(define decimal (has? '("decimal" "numeric") type))
		(list "TYPE_NAME" type "DATA_TYPE" (nth contract 1) "PRECISION" (nth contract 2)
			"LITERAL_PREFIX" (if (or character (equal? type "date")) "'" nil)
			"LITERAL_SUFFIX" (if (or character (equal? type "date")) "'" nil)
			"CREATE_PARAMS" (if character "length" decimal "precision,scale" nil)
			"NULLABLE" 1 "CASE_SENSITIVE" (if character 1 0) "SEARCHABLE" 3
			"UNSIGNED_ATTRIBUTE" (if (equal? type "tinyint") 1 (has? '("bigint" "int" "smallint" "decimal" "numeric" "real" "float") type) 0 nil)
			"MONEY" 0 "AUTO_INCREMENT" (if (has? '("bigint" "int" "smallint" "tinyint") type) 1 0)
			"LOCAL_TYPE_NAME" type "MINIMUM_SCALE" (nth contract 4) "MAXIMUM_SCALE" (if decimal 38 (nth contract 4))
			"SQL_DATA_TYPE" (if (nth contract 6) 9 (nth contract 1)) "SQL_DATETIME_SUB" (nth contract 6)
			"NUM_PREC_RADIX" (nth contract 5) "INTERVAL_PRECISION" nil)))))
	(list '("TYPE_NAME" "DATA_TYPE" "PRECISION" "LITERAL_PREFIX" "LITERAL_SUFFIX" "CREATE_PARAMS" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "LOCAL_TYPE_NAME" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		'("DATA_TYPE" "PRECISION" "NULLABLE" "CASE_SENSITIVE" "SEARCHABLE" "UNSIGNED_ATTRIBUTE" "MONEY" "AUTO_INCREMENT" "MINIMUM_SCALE" "MAXIMUM_SCALE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "NUM_PREC_RADIX" "INTERVAL_PRECISION")
		(filter rows (lambda (row) (or (nil? (options "data_type")) (equal? (options "data_type") 0) (equal? (row "DATA_TYPE") (options "data_type")))))))))
(define tsql_metadata_result (lambda (schema procedure arguments policy) (begin
	(define procedure (toLower procedure))
	(define procedure (if (and (>= (strlen procedure) 4) (equal? (substr procedure 0 4) "sys.")) (substr procedure 4) procedure))
	(define names (match procedure
		"sp_datatype_info" '("data_type" "odbcver")
		"sp_datatype_info_100" '("data_type" "odbcver")
		"sp_tables" '("table_name" "table_owner" "table_qualifier" "table_type" "fusepattern")
		"sp_columns" '("table_name" "table_owner" "table_qualifier" "column_name" "odbcver")
		_ (error "unsupported metadata procedure: " procedure)))
	(define options (tsql_metadata_arguments arguments names))
	/* Never silently substitute another database. Policy executes before catalog
	access, and every table access below belongs to this authenticated database. */
	(map '("table_qualifier" "pktable_qualifier" "fktable_qualifier") (lambda (key)
		(if (or (nil? (options key)) (equal?? schema (options key))) true
			(error "metadata qualifier is not the current database"))))
	(if policy (policy schema true false) true)
	(define owner (coalesceNil (options "table_owner") "dbo"))
	(define version (coalesceNil (options "odbcver") 2))
	(if (has? '(2 3) version) true (error "unsupported ODBC version"))
	(define result (match procedure
		"sp_datatype_info" (tsql_metadata_type_info options version)
		"sp_datatype_info_100" (tsql_metadata_type_info options version)
		"sp_tables" (begin
			(define filter_type (options "table_type"))
			(define types (if (nil? filter_type) '("TABLE")
				(map (split filter_type ",") (lambda (value) (toUpper (replace (strtrim value) "'" ""))))))
			(define rows (if (and (tsql_metadata_match "dbo" owner true) (has? types "TABLE"))
				(map (tsql_metadata_table_names schema (options "table_name") (not (equal? (coalesceNil (options "fusepattern") 1) 0)) policy)
					(lambda (name) (list "TABLE_QUALIFIER" schema "TABLE_OWNER" "dbo" "TABLE_NAME" name "TABLE_TYPE" "TABLE" "REMARKS" nil))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "TABLE_TYPE" "REMARKS") '() rows))
		"sp_columns" (begin
			(if (nil? (options "table_name")) (error "sp_columns requires a table name") true)
			(define rows (if (tsql_metadata_match "dbo" owner true)
				(merge (map (tsql_metadata_table_names schema (options "table_name") true policy) (lambda (name) (begin
					(define columns (filter (show (table schema name)) (lambda (column) (not (internal_column_name? (column "Field"))))))
					(filter (map (produceN (count columns)) (lambda (i)
						(tsql_metadata_column_row schema name (nth columns i) (+ i 1) version)))
						(lambda (row) (tsql_metadata_match (row "COLUMN_NAME") (options "column_name") true))))))) '()))
			(list '("TABLE_QUALIFIER" "TABLE_OWNER" "TABLE_NAME" "COLUMN_NAME" "DATA_TYPE" "TYPE_NAME" "PRECISION" "LENGTH" "SCALE" "RADIX" "NULLABLE" "REMARKS" "COLUMN_DEF" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "CHAR_OCTET_LENGTH" "ORDINAL_POSITION" "IS_NULLABLE")
				'("DATA_TYPE" "PRECISION" "LENGTH" "SCALE" "RADIX" "NULLABLE" "SQL_DATA_TYPE" "SQL_DATETIME_SUB" "CHAR_OCTET_LENGTH" "ORDINAL_POSITION") rows))
		_ (error "unsupported metadata procedure: " procedure)))
	(list "columns" (tsql_metadata_descriptors (car result) (cadr result)) "rows" (nth result 2)))))
(define tsql_execute_metadata (lambda (schema procedure arguments policy resultrow resultfields) (begin
	(define result (tsql_metadata_result schema procedure arguments policy))
	(resultfields (map (result "columns") (lambda (column) (column "name"))) (result "columns"))
	(map (result "rows") resultrow)
	0)))

/* Network callbacks use the same accounts, transaction owner and SQL execution
path as the other frontends. Wire actions are interpreted here, never in scm/. */
(define tds_frontend (lambda (event session payload row fields state sequence) (match event
	"login" (begin
		(define user (payload "username"))
		(define database (payload "database"))
		(define stored (mysql_auth user))
		(if (and stored (equal? stored (password (payload "password"))) (mysql_schema user database))
			(begin
				((sql_policy user) database true false)
				(define session (newsession))
				(session "username" user)
				(session "schema" database)
				(session "syntax" "tsql")
				(session "wire_metadata" true)
				session) nil))
	"close" (tx_rollback session)
	"reset" (begin
		(tx_rollback session)
		(define replacement (newsession))
		(replacement "username" (session "username"))
		(replacement "schema" (session "schema"))
		(replacement "syntax" "tsql")
		(replacement "wire_metadata" true)
		replacement)
	"transaction" (begin
		(if (and (equal? (payload "action") 1) (not (has? '(0 2) (payload "isolation"))))
			(error "unsupported TDS transaction isolation") true)
		(define sql (match (payload "action") 1 "BEGIN" 2 "COMMIT" 3 "ROLLBACK" _ (error "unknown transaction action")))
		(with_autocommit session state sequence sql (lambda (tx)
			(tsql_execute_statement tx (session "schema") sql session (lambda (_row) true) (lambda args true))))
		(list "rows" 0 "transaction" (payload "action")))
	"execute" (begin
		(session "__tsql_scope" (newsession))
		(tds_bind_parameters session (tds_bind_declarations (payload "database") (payload "declarations"))
			(payload "parameters"))
		(define output (newsession))
		(define result (with_autocommit session state sequence (payload "text") (lambda (tx)
			(tsql_execute_statement tx (payload "database") (payload "text") session
				(lambda (values) (row (map (output "descriptors") (lambda (descriptor) (begin
					(define value (values (descriptor "name")))
					(tds_result_value descriptor value))))))
				(lambda (_names descriptors) (begin
					(output "descriptors" descriptors)
					(fields descriptors)))))))
		/* NOCOUNT changes protocol completion only; native DML still computes
		its affected-row count and result rows follow the shared planner. */
		(if (session "__tsql_nocount")
			(list "rows" (if (number? result) result 0) "no_count" true)
			(if (number? result) (list "rows" result) nil)))
	"describe" (tds_describe (payload "database") (payload "text") (payload "declarations") session)
	"metadata" (begin
		(define arguments (map (payload "parameters") (lambda (parameter) (begin
			(if (equal? (parameter "flags") 0) true (error "metadata output parameters are unsupported"))
			(if (has? '(31 38 104 167 175 231 239) ((parameter "type") "kind")) true
				(error "unsupported metadata argument wire type"))
			(list (parameter "name") (parameter "value"))))))
		(define result (tsql_metadata_result (session "schema") (payload "procedure") arguments (sql_policy (session "username"))))
		(fields (result "columns"))
		(map (result "rows") (lambda (values)
			(row (map (result "columns") (lambda (column) (begin
				(define value (values (column "name")))
				(if (nil? value) nil (equal? (column "kind") 231) (string value) (intdiv value 1))))))))
		(list "rows" (count (result "rows"))))
	_ (error "unknown TDS frontend callback"))))
(if (arg "tds-port" nil) (begin
	(define port (arg "tds-port" nil))
	(tds port tds_frontend (arg "tds-database" "memcp-tests"))
	(service_registry "T-SQL protocol" (list port "" "TDS"))
	(print "TDS listener on port " port)))
