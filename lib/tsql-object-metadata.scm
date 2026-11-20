/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Numeric compatibility identifiers belong to this frontend. Native tables
keep their durable incarnation keys; virtual/declaration slots have separate
lossless namespaces in this same registry, never hash-derived integer IDs. */
(define tsql_initialize_object_ids (lambda () (begin
	(eval (parse_sql "system" "CREATE TABLE IF NOT EXISTS tsql_object_ids (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, database VARCHAR(255) COLLATE utf8mb4_bin NOT NULL, incarnation VARCHAR(80) COLLATE utf8mb4_bin NOT NULL, UNIQUE KEY object_identity(database,incarnation)) ENGINE=SAFE AUTO_INCREMENT=10000" nil))
	/* Composite declaration slots must remain lossless. Widen only the known
	legacy bound, keeping type, collation, values and unique-key ownership. The
	native expected-declaration owner also revalidates concurrent initialization. */
	(define registry (table "system" false "tsql_object_ids" false nil))
	(define column (show registry "column" "incarnation" true))
	(if (and (equal? (column "RawType") "VARCHAR")
		(equal? (column "Null") false) (equal? (column "Collation") "utf8mb4_bin")) true
		(error "unsupported object registry key declaration"))
	(define dimensions (coalesceNil (column "Dimensions") '()))
	(if (empty_list? dimensions) true
		(equal? dimensions '(80))
		(migratealtercolumn registry "incarnation"
			(list "VARCHAR" '() false "utf8mb4_bin")
			(list "VARCHAR" '(80) false "utf8mb4_bin"))
		(error "unsupported object registry key width")))))
(define tsql_object_registry_id (close_procedure ((lambda (access)
	(lambda (database incarnation) (begin
		(define ids (table "system" false "tsql_object_ids" false nil))
		(define lookup (lambda () (scan_lookup nil ids access (list database incarnation))))
		(define existing (lookup))
		(if existing (if (and (> existing 0) (<= existing 2147483647)) existing
			(error "object identity exceeds T-SQL INT domain")) (begin
				(insert ids '("database" "incarnation") (list (list database incarnation)) '() (lambda () true))
				(define id (lookup))
				(if (and id (> id 0) (<= id 2147483647)) id (error "object identity exceeds T-SQL INT domain")))))))
	(list (compiled_scan_access_header 2 "value" 1 -1)
		/* Lead with the selective identity. The database discriminator remains
		a required equality bound, while its skewed compressed column no longer
		drives every bisection step. Parameter slots keep their original order. */
		(scan_boundary "equal" "incarnation" 1 1 true true "" false)
		(scan_boundary "equal" "database" 0 0 true true "" false) "id"))))
/* SQL source kinds are frontend declaration facts shared by execution and
catalog reflection. Import this classifier before closing metadata helpers. */
(define tsql_source_module_kind_factory (close_procedure (lambda (profile)
	(lambda (language)
		(if (profile "tsql-procedure" language) "P"
			(profile "tsql-table-function" language) "TF"
			(or (profile "tsql-scalar-function" language)
				(profile "tsql-relational-scalar-function" language)) "FN" nil)))))
(define tsql_source_module_kind (close_procedure (tsql_source_module_kind_factory tsql_source_language_profile)))
(define tsql_source_catalog_entries_factory (close_procedure (lambda (classify internal_name)
	(lambda (database tx)
		(map (filter (showprocedures database "" tx) (lambda (definition)
			(and (not (nil? (classify (definition "language"))))
				(not (internal_name (definition "name"))))))
			(lambda (definition)
				(list "name" (definition "name") "language" (definition "language"))))))))
(define tsql_source_catalog_entries (close_procedure (tsql_source_catalog_entries_factory tsql_source_module_kind internal_column_name?)))
(define tsql_catalog_module_id_factory (close_procedure (lambda (registry)
	(lambda (database name kind)
		(registry database (json_encode (list "source" kind name)))))))
(define tsql_catalog_module_id (close_procedure (tsql_catalog_module_id_factory tsql_object_registry_id)))
(define tsql_catalog_module_definition (lambda (database name tx)
	(find (showprocedures database (toLower name) tx) (lambda (_definition) true) nil)))

(define tsql_legacy_catalog_names '("sysobjects" "syscolumns" "syscomments" "syslogins" "sysdatabases"))
(define tsql_legacy_catalog_name? (close_procedure ((lambda (names)
	(lambda (name) (has? names (toLower name)))) tsql_legacy_catalog_names)))
(define tsql_virtual_catalog_names '("databases" "sysdatabases" "schemas" "objects" "columns" "all_columns" "sysobjects" "syscolumns"
	"database_permissions" "database_principals" "server_principals" "database_role_members"
	"foreign_keys" "key_constraints" "indexes" "check_constraints" "default_constraints" "triggers" "syscomments" "syslogins"))
(define tsql_native_object_id_factory (close_procedure (lambda (visible relation registry)
	(lambda (database name actor tx)
		(if (visible actor database true) (begin
			(define handle (relation database name tx))
			(if (nil? handle) nil (begin
				(define identity (show handle "identity" tx))
				(if (nil? identity) (error "stable native relation identity is unavailable")
					(registry database identity))))) nil)))))
(define tsql_native_object_id (close_procedure (tsql_native_object_id_factory tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id)))
/* These slots identify existing frontend declarations, not fabricated native
incarnations. A dropped/re-added view or same-named constraint in the same containing
native table may reuse its slot; current catalog visibility supplies existence.
The shared registry keeps them disjoint from native UUIDs and virtual catalogs. */
(define tsql_catalog_relation_id_factory (close_procedure (lambda (native_id registry visible)
	(lambda (database name kind actor tx)
		(if (visible actor database true)
			(if (equal? kind "VIEW") (registry database (json_encode (list "view" name)))
				(native_id database name actor tx)) nil)))))
(define tsql_catalog_relation_id (close_procedure (tsql_catalog_relation_id_factory tsql_native_object_id tsql_object_registry_id tsql_metadata_visible?)))
(define tsql_catalog_constraint_id_factory (close_procedure (lambda (relation registry visible)
	(lambda (database containing name kind actor tx)
		(if (visible actor database true) (begin
			(define handle (relation database containing tx))
			(if (nil? handle) nil (begin
				(define identity (show handle "identity" tx))
				(if (nil? identity) (error "stable containing relation identity is unavailable")
					(registry database (json_encode (list kind identity name))))))) nil)))))
(define tsql_catalog_constraint_id (close_procedure (tsql_catalog_constraint_id_factory tsql_native_relation_handle tsql_object_registry_id tsql_metadata_visible?)))
/* These are the frontend's existing ordinary-object and virtual metadata
namespaces. One declaration owns both intrinsic directions and sys.schemas;
no user namespace or additional authority is introduced.
https://learn.microsoft.com/sql/t-sql/functions/schema-id-transact-sql
SQL Server 2022 names/padding/type reference: https://dbfiddle.uk/SmQdMKPu */
(define tsql_schema_names (list (list "dbo" 1) (list "INFORMATION_SCHEMA" 3) (list "sys" 4)))
(define tsql_schema_name_value (close_procedure ((lambda (schemas)
	(lambda (id) (if (nil? id) nil (begin
		(define schema (find schemas (lambda (schema) (equal? (cadr schema) id)) nil))
		(if (nil? schema) nil (car schema)))))) tsql_schema_names)))
(define tsql_schema_id_value (close_procedure ((lambda (schemas)
	(lambda (name) (if (nil? name) nil (begin
		(define schema (find schemas (lambda (schema) (equal?? (car schema) (sql_rtrim name))) nil))
		(if (nil? schema) nil (cadr schema)))))) tsql_schema_names)))

/* Metadata addresses contain sysname values, not SQL source tokens. An
unquoted part may contain spaces, hyphens, reserved words or initial digits.
Dots still delimit parts; quotes retain their complete-part/escape contract.
Unquoted trailing spaces are padding, but leading spaces and whitespace after
quoted parts are not silently removed. No SQL identifier grammar is changed.
SQL Server 2022 references: https://dbfiddle.uk/65LLyGfy,
https://dbfiddle.uk/J_4XF8di and https://dbfiddle.uk/twoRgJZT. */
/* Address values are scanned at row execution, where native parser/regexp
objects cannot be captured in a transferable SQL callable. Keep multipart
and delimiter boundaries using plain strings, including doubled quote escapes. */
(define tsql_object_name_parts (close_procedure ((lambda (private_name)
	(lambda (text) (begin
		/* State: byte position, lexical mode, current part, prior parts,
		quoted-part flag, validity. ASCII delimiters preserve UTF-8 bytes. */
		(define state (for (list 0 0 "" '() false true)
			(lambda (position _mode _part _parts _quoted valid)
				(and valid (< position (strlen text))))
			(lambda (position mode part parts quoted valid) (begin
				(define character (substr text position 1))
				(define following (if (< (+ position 1) (strlen text)) (substr text (+ position 1) 1) ""))
				(if (or (equal? mode 2) (equal? mode 3)) (begin
					(define delimiter (if (equal? mode 2) "]" "\""))
					(if (equal? character delimiter)
						(if (equal? following delimiter)
							(list (+ position 2) mode (concat part delimiter) parts quoted true)
							(list (+ position 1) 4 part parts quoted (not (equal? part ""))))
						(list (+ position 1) mode (concat part character) parts quoted true)))
					(equal? character ".")
					(list (+ position 1) 0 "" (append parts (list (if quoted part (sql_rtrim part)) quoted)) false true)
					(equal? mode 4) (list position mode part parts quoted false)
					(and (equal? mode 0) (or (equal? character "[") (equal? character "\"")))
					(list (+ position 1) (if (equal? character "[") 2 3) part parts true true)
					(list (+ position 1) 1 (concat part character) parts false true))))))
		(define mode (nth state 1))
		(define entries (append (nth state 3) (list
			(if (nth state 4) (nth state 2) (sql_rtrim (nth state 2))) (nth state 4))))
		(define valid (and (nth state 5) (not (or (equal? mode 2) (equal? mode 3)))
			(<= (count entries) 3)
			(reduce entries (lambda (valid entry) (begin
				(define name (car entry))
				(and valid (not (private_name name))))) true)))
		(define names (map entries car))
		(if (not valid) (error "invalid object address")
			(equal? (count names) 1)
			(if (equal? (car names) "") (error "invalid object address") (list nil "dbo" (car names)))
			(equal? (count names) 2)
			(if (or (equal? (car names) "") (equal? (cadr names) "")) (error "invalid object address")
				(list nil (car names) (cadr names)))
			(if (or (equal? (car names) "") (equal? (nth names 2) "")) (error "invalid object address")
				(list (car names) (if (equal? (cadr names) "") "dbo" (cadr names)) (nth names 2)))))))
	sql_private_relation?)))
(define tsql_object_id_value (close_procedure ((lambda (captured_parse_name captured_qualifier captured_visible captured_native_id captured_view_name captured_registry captured_catalogs captured_module_definition captured_module_kind captured_module_id captured_legacy_name? captured_system_procedure captured_relation captured_profile)
	(lambda (context name kind actor tx options)
		(begin
			(define parse_name captured_parse_name)
			(define qualifier captured_qualifier)
			(define visible captured_visible)
			(define registry captured_registry)
			(define relation captured_relation)
			(define native_id (captured_native_id visible relation registry))
			(define view_name captured_view_name)
			(define catalogs captured_catalogs)
			(define module_definition captured_module_definition)
			(define module_kind (captured_module_kind captured_profile))
			(define module_id (captured_module_id registry))
			(define legacy_name? captured_legacy_name?)
			(define system_procedure captured_system_procedure)
			(if (nil? name) nil (begin
				(define parts (try (lambda () (parse_name name)) (lambda (_invalid_address) nil)))
				(if (nil? parts) nil (begin
					(define database (database_name (qualifier (car parts) context options) true))
					(define owner (cadr parts))
					(define object (nth parts 2))
					(if (or (nil? database) (not (visible actor database false))) nil
						(equal?? owner "sys")
						(begin
							(define builtin (system_procedure object))
							(if (if builtin (or (nil? kind) (equal?? kind (cadr builtin))) false)
								(registry database (json_encode (list "system-procedure" (car builtin))))
								(and (or (nil? kind) (equal?? kind "V")) (has? catalogs (toLower object)))
								(registry database (concat "catalog:sys." (toLower object))) nil))
						(not (equal?? owner "dbo")) nil
						(begin
							(define native (if (or (nil? kind) (equal?? kind "U"))
								(native_id database object actor tx) nil))
							(define view (if (or (nil? kind) (equal?? kind "V"))
								(view_name database object tx) nil))
							(define module (if (or (nil? kind) (equal?? kind "P") (equal?? kind "FN") (equal?? kind "TF"))
								(module_definition database object tx) nil))
							(define declared_kind (if (nil? module) nil (module_kind (module "language"))))
							(define builtin (system_procedure object))
							(if (and native (or (nil? kind) (equal?? kind "U"))) native
								(and (not (nil? view)) (or (nil? kind) (equal?? kind "V")))
								(registry database (json_encode (list "view" view)))
								(and (not (nil? declared_kind)) (or (nil? kind) (equal?? kind declared_kind)))
								(module_id database (module "name") declared_kind)
								(and (or (nil? kind) (equal?? kind "V")) (legacy_name? object))
								(registry database (concat "catalog:sys." (toLower object)))
								(if builtin (or (nil? kind) (equal?? kind (cadr builtin))) false)
								(registry database (json_encode (list "system-procedure" (car builtin))))
								nil))))))))))
	tsql_object_name_parts tsql_qualifier_database tsql_metadata_visible? tsql_native_object_id_factory
	tsql_view_name tsql_object_registry_id tsql_virtual_catalog_names tsql_catalog_module_definition
	tsql_source_module_kind_factory tsql_catalog_module_id_factory tsql_legacy_catalog_name? tsql_system_procedure_declaration
	tsql_native_relation_handle tsql_source_language_profile)))

/* SQL conversions and result contracts are selected once by the expression
compiler. Runtime lookup receives only plain names, authority and native tx. */
(define tsql_compile_object_id (lambda (context) (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "OBJECT_ID requires a name and optional type"))
	(define name (sql_info_formula (tsql_convert_value_info (car infos) "NVARCHAR" '(776))))
	(define kind (sql_info_formula (tsql_convert_value_info (cadr infos) "NVARCHAR" '(2))))
	(sql_info (list tsql_object_id_value context name kind (tsql_actor_formula) 'tx
		(list 'list "tsql_import_target" (list 'session "tsql_import_target")
			"tsql_import_source" (list 'session "tsql_import_source"))) "INT" nil)))))

/* Catalog columns are ordinary logical projections over the native metadata
source. Finite virtual-view declarations describe the schema we actually expose. */
(define tsql_catalog_column_declarations
	(list (list "object_id" "INT" '() false) (list "name" "NVARCHAR" '(128) false)
		(list "column_id" "INT" '() false) (list "graph_type" "INT" '() true)
		(list "default_object_id" "INT" '() false)
		(list "is_nullable" "BIT" '() false) (list "max_length" "SMALLINT" '() true)
		(list "is_rowguidcol" "BIT" '() false)))
/* Security catalogs project only the native database-wide authority of the
current authenticated incarnation. They are descriptions, never ACL storage. */
(define tsql_principal_catalog_columns (list
	(list "principal_id" "INT" '() false) (list "name" "NVARCHAR" '(128) false)
	(list "type" "CHAR" '(1) false) (list "type_desc" "NVARCHAR" '(60) false)
	(list "sid" "VARBINARY" '(85) true) (list "is_fixed_role" "BIT" '() false)))
(define tsql_permission_catalog_columns (list
	(list "class" "TINYINT" '() false) (list "class_desc" "NVARCHAR" '(60) false)
	(list "major_id" "INT" '() false) (list "minor_id" "INT" '() false)
	(list "grantee_principal_id" "INT" '() false) (list "grantor_principal_id" "INT" '() false)
	(list "type" "CHAR" '(4) false) (list "permission_name" "NVARCHAR" '(128) false)
	(list "state" "CHAR" '(1) false) (list "state_desc" "NVARCHAR" '(60) false)))
(define tsql_role_member_catalog_columns (list
	(list "role_principal_id" "INT" '() false) (list "member_principal_id" "INT" '() false)))
/* The legacy 28-column login profile has no credential projection. Dates are
the durable SQL-login catalog registration/change times: preexisting accounts
are adopted once, never assigned a fabricated historical native creation date.
Legacy accdate aliases registration time; its old usage-accounting facility is
not implemented. Unknown default database/language/obsolete metrics stay NULL.
SQL2022 types/nullability: https://dbfiddle.uk/84ZBYWWC . Its additional fixed-role
columns are outside this legacy profile; roles follow IS_SRVROLEMEMBER. */
(define tsql_syslogins_catalog_columns (list
	(list "sid" "VARBINARY" '(85) true) (list "status" "SMALLINT" '() true)
	(list "createdate" "DATETIME" '() false) (list "updatedate" "DATETIME" '() false)
	(list "accdate" "DATETIME" '() false)
	(list "totcpu" "INT" '() true) (list "totio" "INT" '() true)
	(list "spacelimit" "INT" '() true) (list "timelimit" "INT" '() true)
	(list "resultlimit" "INT" '() true)
	(list "name" "NVARCHAR" '(128) false) (list "dbname" "NVARCHAR" '(128) true)
	(list "password" "NVARCHAR" '(128) true) (list "language" "NVARCHAR" '(128) true)
	(list "denylogin" "INT" '() true) (list "hasaccess" "INT" '() true)
	(list "isntname" "INT" '() true) (list "isntgroup" "INT" '() true)
	(list "isntuser" "INT" '() true) (list "sysadmin" "INT" '() true)
	(list "securityadmin" "INT" '() true) (list "serveradmin" "INT" '() true)
	(list "setupadmin" "INT" '() true) (list "processadmin" "INT" '() true)
	(list "diskadmin" "INT" '() true) (list "dbcreator" "INT" '() true)
	(list "bulkadmin" "INT" '() true) (list "loginname" "NVARCHAR" '(128) false)))
(define tsql_legacy_catalog_declarations
	(list (list "sysobjects" (list
		(list "name" "NVARCHAR" '(128) false) (list "id" "INT" '() false)
		(list "xtype" "CHAR" '(2) false) (list "type" "CHAR" '(2) false)
		(list "uid" "SMALLINT" '() false) (list "parent_obj" "INT" '() false)))
		(list "syscolumns" (list (list "name" "NVARCHAR" '(128) false)
			(list "id" "INT" '() false) (list "colid" "SMALLINT" '() false)
			(list "cdefault" "INT" '() false)))
		(list "syscomments" (list (list "id" "INT" '() false)
			(list "text" "NVARCHAR" '(4000) true)))
		(list "syslogins" tsql_syslogins_catalog_columns)
		(list "sysdatabases" (list (list "name" "NVARCHAR" '(128) false)
			(list "dbid" "SMALLINT" '() false) (list "sid" "VARBINARY" '(85) true)))))
/* These views expose declaration facts that the native catalog actually owns.
No physical index ordinal/clustered flags, historical trust or generated names
are inferred from acceleration indexes or registry slots. */
(define tsql_constraint_catalog_columns (list
	(list "name" "NVARCHAR" '(128) false) (list "object_id" "INT" '() false)
	(list "parent_object_id" "INT" '() false) (list "schema_id" "INT" '() false)
	(list "type" "CHAR" '(2) false) (list "type_desc" "NVARCHAR" '(60) false)))
(define tsql_default_catalog_columns (append tsql_constraint_catalog_columns
	(list "parent_column_id" "INT" '() false) (list "definition" "NVARCHAR" '(-1) false)))
(define tsql_index_catalog_columns (list (list "name" "NVARCHAR" '(128) false)
	(list "object_id" "INT" '() false)))
/* Only current declaration facts are represented. The native catalog has no
creation/modification timestamps or replication-trigger profile to invent.
https://learn.microsoft.com/sql/relational-databases/system-catalog-views/sys-triggers-transact-sql */
(define tsql_trigger_catalog_columns (list
	(list "name" "NVARCHAR" '(128) false) (list "object_id" "INT" '() false)
	(list "parent_class" "TINYINT" '() false) (list "parent_class_desc" "NVARCHAR" '(60) false)
	(list "parent_id" "INT" '() false) (list "type" "CHAR" '(2) false)
	(list "type_desc" "NVARCHAR" '(60) false) (list "is_ms_shipped" "BIT" '() false)
	(list "is_disabled" "BIT" '() false) (list "is_instead_of_trigger" "BIT" '() false)))
(define tsql_virtual_catalog_declarations
	(merge (list tsql_legacy_catalog_declarations (list (list "database_permissions" tsql_permission_catalog_columns)
		(list "database_principals" tsql_principal_catalog_columns)
		(list "server_principals" tsql_principal_catalog_columns)
		(list "database_role_members" tsql_role_member_catalog_columns)
		(list "foreign_keys" tsql_constraint_catalog_columns)
		/* The native owner exposes the inherited declaration facts. Physical
		index ordinals and generated-name provenance are not fabricated.
		https://learn.microsoft.com/sql/relational-databases/system-catalog-views/sys-key-constraints-transact-sql */
		(list "key_constraints" tsql_constraint_catalog_columns)
		(list "check_constraints" tsql_constraint_catalog_columns)
		(list "default_constraints" tsql_default_catalog_columns)
		(list "indexes" tsql_index_catalog_columns)
		(list "triggers" tsql_trigger_catalog_columns)
		(list "columns" tsql_catalog_column_declarations)
		(list "all_columns" tsql_catalog_column_declarations)
		(list "schemas" (list (list "name" "NVARCHAR" '(128) false)
			(list "schema_id" "INT" '() false)))
		(list "objects" (list (list "name" "NVARCHAR" '(128) false)
			(list "type" "CHAR" '(2) false) (list "type_desc" "NVARCHAR" '(60) false)
			(list "object_id" "INT" '() false) (list "parent_object_id" "INT" '() false)
			(list "schema_id" "INT" '() false) (list "is_ms_shipped" "BIT" '() false)
			(list "modify_date" "DATETIME" '() true)))
		(list "databases" (list (list "name" "NVARCHAR" '(128) false)
			(list "database_id" "INT" '() false) (list "is_auto_close_on" "BIT" '() false)
			(list "is_auto_shrink_on" "BIT" '() false) (list "recovery_model_desc" "NVARCHAR" '(60) false)
			(list "is_trustworthy_on" "BIT" '() false) (list "owner_sid" "VARBINARY" '(85) true)))))))
/* A fixed declaration domain uses the common literal-row producer. Filters,
joins and aggregation remain ordinary relational planning operations. */
(define tsql_catalog_schemas_query (lambda (database) (begin
	(define alias "__tsql_schemas")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(make_query_block database
		(list (list alias database (list 'literal-rows '("name" "schema_id")
			(map tsql_schema_names (lambda (schema)
				(list "name" (car schema) "schema_id" (cadr schema))))) false nil nil))
		(list "name" (list 'sql_type_expr (column "name") (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2))
			"schema_id" (list 'sql_type_expr (column "schema_id") "INT" nil))
		(list tsql_metadata_visible? (tsql_actor_formula) database false)
		nil nil nil nil nil '() '() (list (list 'fresh_invocation true))))))
/* Folded defaults retain a native declared type and value; contextual defaults
retain their actual SQL source. Render a semantically equivalent T-SQL literal
from that declaration, never a procedure printer or runtime SQL type guess.
Unnamed native defaults have no durable SQL Server-generated name and are not
represented as fabricated named objects in this declaration profile. */
(define tsql_catalog_default_definition (close_procedure ((lambda (tsql_temporal_quantize empty_list?)
	(lambda (column) (begin
		(define source (column "DefaultExpression"))
		(define value (column "Default"))
		(define type (toUpper (column "RawType")))
		(if (not (equal? source "")) source
			(nil? value) "NULL"
			(has? '("CHAR" "VARCHAR" "TEXT" "NCHAR" "NVARCHAR" "NTEXT" "UNIQUEIDENTIFIER") type)
			(concat (if (has? '("NCHAR" "NVARCHAR" "NTEXT") type) "N" "") "'" (replace value "'" "''") "'")
			(has? '("BOOL" "BOOLEAN" "BIT") type) (if value "1" "0")
			(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY" "REAL" "FLOAT" "DOUBLE") type)
			(concat value)
			(has? '("BINARY" "VARBINARY" "IMAGE") type) (concat "0x" (bin2hex value))
			(has? '("DATE" "DATETIME" "SMALLDATETIME" "DATETIME2" "TIME" "DATETIMEOFFSET") type) (begin
				/* ISO literals are independent of LANGUAGE/DATEFORMAT. Precision
				comes from the persisted declaration, not the native value's tag.
				https://learn.microsoft.com/sql/t-sql/data-types/datetime2-transact-sql
				https://learn.microsoft.com/sql/t-sql/data-types/datetimeoffset-transact-sql */
				(define value (if (equal? type "DATETIME")
					(tsql_temporal_quantize value 300 -6847804800) value))
				(define parts (date_parts value))
				(define dimensions (coalesceNil (column "Dimensions") '()))
				(define digits (if (has? '("DATE" "SMALLDATETIME") type) 0
					(equal? type "DATETIME") 3
					(if (empty_list? dimensions) 7 (car dimensions))))
				/* The native date zone is a registry ID, not an offset in minutes.
				Until its existing registry supports lookup, do not invent an offset. */
				(if (and (equal? type "DATETIMEOFFSET") (not (equal? (nth parts 2) 0)))
					(error "folded DATETIMEOFFSET DEFAULT requires native zone lookup") true)
				(define nanos (cadr parts))
				(define nanos (if (equal? type "DATETIME")
					(* 1000000 (intdiv (+ nanos 500000) 1000000)) nanos))
				(define fraction (if (equal? digits 0) "" (concat "."
					(substr (concat (string_repeat "0" (- 9 (strlen (string nanos)))) nanos) 0 digits))))
				(concat "'" (format_date value (if (equal? type "DATE") "%Y-%m-%d"
					(equal? type "TIME") "%H:%i:%s" "%Y-%m-%dT%H:%i:%s"))
					fraction (if (equal? type "DATETIMEOFFSET") "+00:00" "") "'"))
			(error "unsupported folded DEFAULT catalog declaration: " type)))))
	tsql_temporal_quantize empty_list?)))
/* This compatibility profile exposes single-piece DEFAULT definitions.
Long source/module text needs the documented colid chunk producer, not silent
truncation or invented text. The normal DEFAULT catalog remains NVARCHAR(MAX).
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-syscomments-transact-sql */
(define tsql_catalog_default_comment (close_procedure (lambda (definition)
	(if (<= (utf16_length definition) 4000) definition
		(error "syscomments multi-piece definition requires a source chunk producer")))))
(define tsql_catalog_default_field_factory (close_procedure (lambda (visible relation identity definition)
	(lambda (database containing name field actor tx)
		(if (visible actor database true) (begin
			(define handle (relation database containing tx))
			(define column (if (nil? handle) nil (show handle "column" name false tx)))
			(define declared (if (nil? column) nil (column "DefaultName")))
			(if (or (nil? declared) (equal? declared ""))
				(if (equal? field "id") 0 nil)
				(equal? field "name") declared
				(equal? field "id") (identity database containing declared "default" actor tx)
				(equal? field "definition") (definition column)
				(error "unknown DEFAULT catalog field"))) nil)))))
(define tsql_catalog_default_field (close_procedure (tsql_catalog_default_field_factory tsql_metadata_visible? tsql_native_relation_handle tsql_catalog_constraint_id tsql_catalog_default_definition)))
/* Named metadata relations expose declared values through the common table
function source. Numeric compatibility IDs are produced once per metadata row,
not re-evaluated for every pair considered by a catalog join. */
(define tsql_catalog_kind_selected? (lambda (selection kind) (begin
	(define kinds (get_assoc (coalesceNil selection '()) "kinds" nil))
	(or (nil? kinds) (has? kinds kind)))))
/* Keep the runtime family in one lexical program. Each explicit dependency is
bound once, rather than embedding its complete callable graph in every tuple
helper. Catalog selection and permission checks remain at their original calls. */
/* Runtime factories keep each helper body in one place. The declaration
profile includes the complete object, DEFAULT and trigger domains; column rows
use the general profile. Neither profile caches metadata rows or authority. */
(define tsql_catalog_metadata_rows_factory (close_procedure (lambda (information_schema_rows)
	(lambda (database relation tx selection)
		(information_schema_rows "information_schema" relation tx
			(if (expression_equal? selection false) false
				(set_assoc (coalesceNil selection '()) "schema" (list "exact" database))))))))
(define tsql_catalog_public_native_row?_factory (close_procedure (lambda (internal_column_name?)
	(lambda (row)
		(and (not (internal_column_name? (row "table_name")))
			(if (has_assoc? row "column_name") (not (internal_column_name? (row "column_name"))) true))))))
(define tsql_catalog_object_name_matches?_factory (close_procedure (lambda (info_schema_name_matches)
	(lambda (selection name)
		(info_schema_name_matches (get_assoc (coalesceNil selection '()) "object" nil) name)))))
(define tsql_catalog_default_tuple_factory (close_procedure (lambda (tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value)
	(lambda (database name metadata actor tx definition_projection selection) (begin
		(define declared (metadata "DefaultName"))
		(if (or (nil? declared) (equal? declared "")) nil
			(not (tsql_catalog_object_name_matches? selection declared)) nil
			(list declared (tsql_catalog_constraint_id database name declared "default" actor tx)
				(tsql_native_object_id database name actor tx) (tsql_schema_id_value "dbo")
				"D" "DEFAULT_CONSTRAINT" nil (definition_projection metadata))))))))
(define tsql_catalog_default_tuples_factory (close_procedure (lambda (tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_catalog_default_tuple)
	(lambda (database actor tx selection definition_projection)
		(if (not (tsql_catalog_kind_selected? selection "D")) '()
			(merge (map (filter (info_schema_selected_tables database tx selection)
				(lambda (name) (not (internal_column_name? name)))) (lambda (name) (begin
					(define handle (tsql_native_relation_handle database name tx))
					(define columns (if (nil? handle) '() (show handle "columns" tx)))
					(filter (mapIndex columns (lambda (ordinal column)
						(if (internal_column_name? (column "Field")) nil (begin
							(define result (tsql_catalog_default_tuple database name column actor tx definition_projection selection))
							(if (nil? result) nil (mapIndex result (lambda (index value)
								(if (equal? index 6) (+ ordinal 1) value))))))))
						(lambda (row) (not (nil? row)))))))))))))
(define tsql_catalog_column_tuples_factory (close_procedure (lambda (selected_tables internal_name? relation native_id default_field virtual_catalogs registry byte_length)
	(lambda (database actor tx selection include_system) (begin
		(define catalog (if (list? selection) (get_assoc selection "catalog" nil) nil))
		/* Other native dialects may declare wider bounds than SMALLINT can
		represent. Unknown is NULL, never a wrapped or fabricated MAX bound. */
		(define length (lambda (column) (begin
			(define value (byte_length column))
			(if (nil? value) nil (and (>= value -1) (<= value 32767)) value nil))))
		/* Bind each actual transaction's immutable native column directory once
		per table. Lengths describe SQL declarations, never stored payloads.
		https://learn.microsoft.com/sql/relational-databases/system-catalog-views/sys-columns-transact-sql */
		(define native (if (nil? catalog)
			(merge (map (filter (selected_tables database tx selection) (lambda (name) (not (internal_name? name))))
				(lambda (name) (begin
					(define handle (relation database name tx))
					(define columns (if (nil? handle) '() (show handle "columns" tx)))
					(define id (native_id database name actor tx))
					(filter (mapIndex columns (lambda (ordinal column)
						(if (internal_name? (column "Field")) nil
							(list id (column "Field") (+ ordinal 1) nil
								(default_field database name (column "Field") "id" actor tx)
								(column "Null") (length column)
								(if (column "Properties") (if ((column "Properties") "tsql.rowguidcol") true false) false)))))
						(lambda (row) (not (nil? row)))))))) '()))
		(if include_system (merge (list native
			(merge (map (filter virtual_catalogs (lambda (declaration)
				(if (nil? catalog) true (equal?? (car declaration) catalog)))) (lambda (declaration) (begin
					(define id (registry database (concat "catalog:sys." (car declaration))))
					(mapIndex (cadr declaration) (lambda (index column)
						(list id (car column) (+ index 1) nil 0 (nth column 3)
							(length (list "RawType" (cadr column) "Dimensions" (nth column 2))) false))))))))) native))))))
(define tsql_catalog_key_tuples_factory (close_procedure (lambda (tsql_catalog_kind_selected? tsql_catalog_metadata_rows tsql_catalog_public_native_row? tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value)
	(lambda (database actor tx selection)
		(if (not (or (tsql_catalog_kind_selected? selection "PK")
			(tsql_catalog_kind_selected? selection "UQ"))) '()
			(map (filter (tsql_catalog_metadata_rows database "table_constraints" tx selection)
				(lambda (row) (and (tsql_catalog_public_native_row? row)
					(has? '("PRIMARY KEY" "UNIQUE") (row "constraint_type"))
					(tsql_catalog_kind_selected? selection (if (equal? (row "constraint_type") "PRIMARY KEY") "PK" "UQ"))
					(tsql_catalog_object_name_matches? selection (row "constraint_name")))))
				(lambda (row) (begin
					(define primary (equal? (row "constraint_type") "PRIMARY KEY"))
					(list (row "constraint_name") (if primary "PK" "UQ")
						(if primary "PRIMARY_KEY_CONSTRAINT" "UNIQUE_CONSTRAINT")
						(tsql_catalog_constraint_id database (row "table_name") (row "constraint_name")
							(if primary "primary" "unique") actor tx)
						(tsql_native_object_id database (row "table_name") actor tx)
						(tsql_schema_id_value "dbo") false nil)))))))))
(define tsql_catalog_trigger_tuples_factory (close_procedure (lambda (tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_source_language_profile tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id)
	(lambda (database actor tx selection)
		(if (not (tsql_catalog_kind_selected? selection "TR")) '()
			(merge (map (filter (info_schema_selected_tables database tx selection)
				(lambda (name) (not (internal_column_name? name)))) (lambda (name) (begin
					(define handle (tsql_native_relation_handle database name tx))
					(define declarations (if (nil? handle) '() (show handle "statement_triggers" tx)))
					(map (filter declarations (lambda (declaration)
						(and (not (internal_column_name? (declaration "name")))
							(tsql_source_language_profile "tsql-statement" (declaration "language"))
							(tsql_catalog_object_name_matches? selection (declaration "name")))))
						(lambda (declaration) (list (declaration "name")
							(tsql_catalog_constraint_id database name (declaration "name") "trigger" actor tx)
							1 "OBJECT_OR_COLUMN" (tsql_native_object_id database name actor tx)
							"TR" "SQL_TRIGGER" false (declaration "disabled")
							(equal? (declaration "timing") "replace_statement")))))))))))))
(define tsql_catalog_object_tuples_factory (close_procedure (lambda (info_schema_name_matches tsql_catalog_object_name_matches? tsql_catalog_metadata_rows sql_stored_view_catalog_rows tsql_catalog_public_native_row? tsql_catalog_relation_id tsql_catalog_kind_selected? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value tsql_catalog_default_tuples tsql_source_catalog_entries tsql_source_module_kind tsql_catalog_module_id tsql_catalog_trigger_tuples tsql_catalog_key_tuples definition_none)
	(lambda (database actor tx selection) (begin
		(define relation_selection (if (has_assoc? (coalesceNil selection '()) "object")
			(set_assoc (coalesceNil selection '()) "table" (list "predicate"
				(close_procedure ((lambda (selection) (lambda (name)
					(and (info_schema_name_matches (get_assoc selection "table" nil) name)
						(tsql_catalog_object_name_matches? selection name)))) selection)))) selection))
		(define relations (map (filter (merge (list
			(if (tsql_catalog_kind_selected? selection "U")
				(tsql_catalog_metadata_rows database "tables" tx relation_selection) '())
			(if (tsql_catalog_kind_selected? selection "V")
				(sql_stored_view_catalog_rows database tx relation_selection) '()))) tsql_catalog_public_native_row?)
			(lambda (row) (begin
				(define view (equal? (row "table_type") "VIEW"))
				(list (row "table_name") (if view "V" "U") (if view "VIEW" "USER_TABLE")
					(tsql_catalog_relation_id database (row "table_name") (row "table_type") actor tx)
					0 (tsql_schema_id_value "dbo") false nil)))))
		(define foreign (if (not (tsql_catalog_kind_selected? selection "F")) '()
			(map (filter (tsql_catalog_metadata_rows database "referential_constraints" tx selection)
				(lambda (row) (and (tsql_catalog_public_native_row? row)
					(tsql_catalog_object_name_matches? selection (row "constraint_name")))))
				(lambda (row) (list (row "constraint_name") "F" "FOREIGN_KEY_CONSTRAINT"
					(tsql_catalog_constraint_id database (row "table_name") (row "constraint_name") "foreign" actor tx)
					(tsql_native_object_id database (row "table_name") actor tx) (tsql_schema_id_value "dbo") false nil)))))
		(define checks (if (not (tsql_catalog_kind_selected? selection "C")) '()
			(map (filter (tsql_catalog_metadata_rows database "table_constraints" tx selection)
				(lambda (row) (and (equal? (row "constraint_type") "CHECK") (tsql_catalog_public_native_row? row)
					(tsql_catalog_object_name_matches? selection (row "constraint_name")))))
				(lambda (row) (list (row "constraint_name") "C" "CHECK_CONSTRAINT"
					(tsql_catalog_constraint_id database (row "table_name") (row "constraint_name") "check" actor tx)
					(tsql_native_object_id database (row "table_name") actor tx) (tsql_schema_id_value "dbo") false nil)))))
		(define defaults (map (tsql_catalog_default_tuples database actor tx selection definition_none)
			(lambda (row) (list (nth row 0) (nth row 4) (nth row 5) (nth row 1) (nth row 2) (nth row 3) false nil))))
		(define modules (if (not (find '("P" "TF" "FN") (lambda (kind)
			(tsql_catalog_kind_selected? selection kind)) nil)) '()
			(map (filter (tsql_source_catalog_entries database tx)
				(lambda (definition) (and (tsql_catalog_object_name_matches? selection (definition "name"))
					(tsql_catalog_kind_selected? selection (tsql_source_module_kind (definition "language")))))) (lambda (definition) (begin
					(define kind (tsql_source_module_kind (definition "language")))
					(list (definition "name") kind
						(if (equal? kind "P") "SQL_STORED_PROCEDURE" (equal? kind "TF") "SQL_TABLE_VALUED_FUNCTION" "SQL_SCALAR_FUNCTION")
						(tsql_catalog_module_id database (definition "name") kind)
						0 (tsql_schema_id_value "dbo") false nil))))))
		(define triggers (map (tsql_catalog_trigger_tuples database actor tx selection)
			(lambda (row) (list (nth row 0) (nth row 5) (nth row 6) (nth row 1)
				(nth row 4) (tsql_schema_id_value "dbo") false nil))))
		(merge (list relations (tsql_catalog_key_tuples database actor tx selection) foreign checks defaults modules triggers)))))))
(define tsql_catalog_lookup_object_tuples_factory (close_procedure (lambda (tsql_catalog_object_tuples tsql_catalog_kind_selected? tsql_virtual_catalog_names tsql_catalog_object_name_matches? tsql_object_registry_id)
	(lambda (database actor tx selection)
		(merge (list
			(map (tsql_catalog_object_tuples database actor tx selection) (lambda (row)
				(list (nth row 0) (nth row 3) (nth row 1) (nth row 6))))
			(if (tsql_catalog_kind_selected? selection "V")
				(map (filter tsql_virtual_catalog_names (lambda (name)
					(tsql_catalog_object_name_matches? selection name))) (lambda (name)
						(list name (tsql_object_registry_id database (concat "catalog:sys." name)) "V" true))) '())))))))
(define tsql_catalog_object_rows_factory (close_procedure (lambda (objects lookup)
	(lambda (database kind actor tx selection)
		(if (equal? kind "objects") (objects database actor tx selection)
			(equal? kind "lookup_objects") (lookup database actor tx selection)
			(equal? kind "sysobjects") (map (objects database actor tx selection)
				(lambda (row) (list (nth row 0) (nth row 3) (nth row 1)
					(if (has? '("PK" "UQ") (nth row 1)) "K" (nth row 1)) 1 (nth row 4))))
			(error "unsupported T-SQL object metadata source" kind))))))
(define tsql_catalog_declaration_rows_factory (close_procedure (lambda (object_rows defaults triggers default_comment definition)
	(lambda (database kind actor tx selection)
		(if (has? '("objects" "lookup_objects" "sysobjects") kind) (object_rows database kind actor tx selection)
			(equal? kind "defaults") (map (defaults database actor tx selection definition) (lambda (row) (append row database)))
			(equal? kind "triggers") (map (triggers database actor tx selection) (lambda (row) (append row database)))
			(equal? kind "syscomments") (map (defaults database actor tx selection definition)
				(lambda (row) (list (nth row 1) (default_comment (nth row 7)))))
			(has? '("constraints" "foreign_keys" "check_constraints" "key_constraints" "indexes") kind) (begin
				/* Constraint catalogs share the same finite declaration producer as
				objects. Bind their numeric identities in that producer once per
				invocation, before joins or scalar probes consume its table source. */
				(define families (match kind "foreign_keys" '("F") "check_constraints" '("C")
					"constraints" '("F" "C") _ '("PK" "UQ")))
				(define requested (get_assoc (coalesceNil selection '()) "kinds" nil))
				(define families (if (nil? requested) families
					(filter families (lambda (family) (has? requested family)))))
				(map (object_rows database "objects" actor tx
					(set_assoc (coalesceNil selection '()) "kinds" families))
					(lambda (row) (append (if (equal? kind "indexes")
						(list (nth row 0) (nth row 4))
						(list (nth row 0) (nth row 3) (nth row 4) (nth row 5) (nth row 1) (nth row 2))) database))))
			(error "unsupported T-SQL metadata source" kind))))))
(define tsql_catalog_all_rows_factory (close_procedure (lambda (declarations columns)
	(lambda (database kind actor tx selection)
		(if (equal? kind "syscolumns") (map (columns database actor tx selection false)
			(lambda (row) (list (nth row 1) (nth row 0) (nth row 2) (nth row 4))))
			(has? '("columns" "all_columns") kind) (columns database actor tx selection (equal? kind "all_columns"))
			(declarations database kind actor tx selection))))))
(define tsql_catalog_authorized_rows_factory (close_procedure (lambda (visible rows)
	(lambda (database kind actor tx selection)
		(if (expression_equal? selection false) '()
			(not (visible actor database true)) '()
			(rows database kind actor tx selection))))))
(define tsql_catalog_object_metadata_rows_factory (close_procedure (lambda (tables table_rows constraints foreign_rows)
	(lambda (schema relation tx selection)
		(if (and (equal? schema "information_schema") (equal? relation "tables")) (tables tx selection)
			(and (equal? schema "information_schema") (equal? relation "table_constraints"))
			(table_rows tx selection (lambda (database name tx) (constraints database name tx false)))
			(and (equal? schema "information_schema") (equal? relation "referential_constraints"))
			(table_rows tx selection foreign_rows)
			(error "unsupported T-SQL object metadata relation" relation))))))
(define tsql_catalog_runtime_rows (close_procedure ((lambda (__runtime_capture_0 __runtime_capture_1 __runtime_capture_2 __runtime_capture_3 __runtime_capture_4 __runtime_capture_5 __runtime_capture_6 __runtime_capture_7 __runtime_capture_8 __runtime_capture_9 __runtime_capture_10 __runtime_capture_11 __runtime_capture_12 __runtime_capture_13 __runtime_capture_14 __runtime_capture_15 __runtime_capture_16 __runtime_capture_17 __runtime_capture_18 __runtime_capture_19 __runtime_capture_20 __runtime_capture_21 __runtime_capture_22 __runtime_capture_23 __runtime_capture_24 __runtime_capture_25 __runtime_capture_26 __runtime_capture_27 __runtime_capture_28 __runtime_capture_29 __runtime_capture_30 __runtime_capture_31 __runtime_capture_32 __runtime_capture_33 __runtime_capture_34 __runtime_capture_35 __runtime_capture_36 __runtime_capture_37)
	(lambda (database kind actor tx selection) (begin
		(define internal_column_name? __runtime_capture_0)
		(define info_schema_name_matches __runtime_capture_1)
		(define tsql_metadata_visible? __runtime_capture_2)
		(define tsql_native_relation_handle __runtime_capture_3)
		(define tsql_object_registry_id __runtime_capture_4)
		(define tsql_native_object_id (__runtime_capture_5 tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id))
		(define tsql_catalog_constraint_id (__runtime_capture_6 tsql_native_relation_handle tsql_object_registry_id tsql_metadata_visible?))
		(define tsql_schema_id_value __runtime_capture_7)
		(define info_schema_selected_tables (__runtime_capture_8 info_schema_name_matches))
		(define info_schema_selected_databases (__runtime_capture_9 info_schema_name_matches))
		(define tsql_catalog_kind_selected? __runtime_capture_10)
		(define tsql_source_language_profile __runtime_capture_11)
		(define tsql_source_module_kind (__runtime_capture_12 tsql_source_language_profile))
		(define tsql_source_catalog_entries (__runtime_capture_13 tsql_source_module_kind internal_column_name?))
		(define sql_stored_view_catalog_rows __runtime_capture_14)
		(define tsql_catalog_relation_id (__runtime_capture_15 tsql_native_object_id tsql_object_registry_id tsql_metadata_visible?))
		(define tsql_catalog_module_id (__runtime_capture_16 tsql_object_registry_id))
		(define tsql_virtual_catalog_names __runtime_capture_17)
		(define definition_none (lambda (_column) nil))
		(define information_schema_rows __runtime_capture_18)
		(define tsql_catalog_default_definition __runtime_capture_19)
		(define tsql_catalog_default_field (__runtime_capture_20 tsql_metadata_visible? tsql_native_relation_handle tsql_catalog_constraint_id tsql_catalog_default_definition))
		(define tsql_virtual_catalog_declarations __runtime_capture_21)
		(define tsql_catalog_default_comment __runtime_capture_22)
		(define tsql_catalog_metadata_rows (__runtime_capture_23 information_schema_rows))
		(define tsql_catalog_public_native_row? (__runtime_capture_24 internal_column_name?))
		(define tsql_catalog_object_name_matches? (__runtime_capture_25 info_schema_name_matches))
		(define tsql_catalog_default_tuple (__runtime_capture_26 tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value))
		(define tsql_catalog_default_tuples (__runtime_capture_27 tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_catalog_default_tuple))
		(define tsql_catalog_column_tuples (__runtime_capture_28 info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_native_object_id tsql_catalog_default_field tsql_virtual_catalog_declarations tsql_object_registry_id __runtime_capture_37))
		(define tsql_catalog_key_tuples (__runtime_capture_29 tsql_catalog_kind_selected? tsql_catalog_metadata_rows tsql_catalog_public_native_row? tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value))
		(define tsql_catalog_trigger_tuples (__runtime_capture_30 tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_source_language_profile tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id))
		(define tsql_catalog_object_tuples (__runtime_capture_31 info_schema_name_matches tsql_catalog_object_name_matches? tsql_catalog_metadata_rows sql_stored_view_catalog_rows tsql_catalog_public_native_row? tsql_catalog_relation_id tsql_catalog_kind_selected? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value tsql_catalog_default_tuples tsql_source_catalog_entries tsql_source_module_kind tsql_catalog_module_id tsql_catalog_trigger_tuples tsql_catalog_key_tuples definition_none))
		(define tsql_catalog_lookup_object_tuples (__runtime_capture_32 tsql_catalog_object_tuples tsql_catalog_kind_selected? tsql_virtual_catalog_names tsql_catalog_object_name_matches? tsql_object_registry_id))
		(define object_rows (__runtime_capture_33 tsql_catalog_object_tuples tsql_catalog_lookup_object_tuples))
		(define declaration_rows (__runtime_capture_36 object_rows tsql_catalog_default_tuples tsql_catalog_trigger_tuples tsql_catalog_default_comment tsql_catalog_default_definition))
		(define all_rows (__runtime_capture_34 declaration_rows tsql_catalog_column_tuples))
		(define rows (__runtime_capture_35 tsql_metadata_visible? all_rows))
		(if (nil? database)
			(merge (map (info_schema_selected_databases selection) (lambda (database) (rows database kind actor tx selection))))
			(rows database kind actor tx selection)))))
	internal_column_name? info_schema_name_matches tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id tsql_native_object_id_factory tsql_catalog_constraint_id_factory tsql_schema_id_value info_schema_selected_tables_factory info_schema_selected_databases_factory tsql_catalog_kind_selected? tsql_source_language_profile tsql_source_module_kind_factory tsql_source_catalog_entries_factory sql_stored_view_catalog_rows tsql_catalog_relation_id_factory tsql_catalog_module_id_factory tsql_virtual_catalog_names information_schema_rows tsql_catalog_default_definition tsql_catalog_default_field_factory tsql_virtual_catalog_declarations tsql_catalog_default_comment tsql_catalog_metadata_rows_factory tsql_catalog_public_native_row?_factory tsql_catalog_object_name_matches?_factory tsql_catalog_default_tuple_factory tsql_catalog_default_tuples_factory tsql_catalog_column_tuples_factory tsql_catalog_key_tuples_factory tsql_catalog_trigger_tuples_factory tsql_catalog_object_tuples_factory tsql_catalog_lookup_object_tuples_factory tsql_catalog_object_rows_factory tsql_catalog_all_rows_factory tsql_catalog_authorized_rows_factory tsql_catalog_declaration_rows_factory (close_procedure tsql_declared_byte_length))))
(define tsql_catalog_declaration_runtime_rows (close_procedure ((lambda (__runtime_capture_0 __runtime_capture_1 __runtime_capture_2 __runtime_capture_3 __runtime_capture_4 __runtime_capture_5 __runtime_capture_6 __runtime_capture_7 __runtime_capture_8 __runtime_capture_9 __runtime_capture_10 __runtime_capture_11 __runtime_capture_12 __runtime_capture_13 __runtime_capture_14 __runtime_capture_15 __runtime_capture_16 __runtime_capture_17 __runtime_capture_18 __runtime_capture_19 __runtime_capture_20 __runtime_capture_21 __runtime_capture_22 __runtime_capture_23 __runtime_capture_24 __runtime_capture_25 __runtime_capture_26 __runtime_capture_27 __runtime_capture_28 __runtime_capture_29 __runtime_capture_30 __runtime_capture_31 __runtime_capture_32 __runtime_capture_33 __runtime_capture_34 __runtime_capture_35 __runtime_capture_36 __runtime_capture_37)
	(lambda (database kind actor tx selection) (begin
		(define internal_column_name? __runtime_capture_0)
		(define info_schema_name_matches __runtime_capture_1)
		(define tsql_metadata_visible? __runtime_capture_2)
		(define tsql_native_relation_handle __runtime_capture_3)
		(define tsql_object_registry_id __runtime_capture_4)
		(define tsql_native_object_id (__runtime_capture_5 tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id))
		(define tsql_catalog_constraint_id (__runtime_capture_6 tsql_native_relation_handle tsql_object_registry_id tsql_metadata_visible?))
		(define tsql_schema_id_value __runtime_capture_7)
		(define info_schema_selected_tables (__runtime_capture_8 info_schema_name_matches))
		(define info_schema_selected_databases (__runtime_capture_9 info_schema_name_matches))
		(define tsql_catalog_kind_selected? __runtime_capture_10)
		(define tsql_source_language_profile __runtime_capture_11)
		(define tsql_source_module_kind (__runtime_capture_12 tsql_source_language_profile))
		(define tsql_source_catalog_entries (__runtime_capture_13 tsql_source_module_kind internal_column_name?))
		(define sql_stored_view_catalog_rows __runtime_capture_14)
		(define tsql_catalog_relation_id (__runtime_capture_15 tsql_native_object_id tsql_object_registry_id tsql_metadata_visible?))
		(define tsql_catalog_module_id (__runtime_capture_16 tsql_object_registry_id))
		(define tsql_virtual_catalog_names __runtime_capture_17)
		(define definition_none (lambda (_column) nil))
		(define info_schema_native_table_rows (__runtime_capture_18 info_schema_selected_databases info_schema_name_matches __runtime_capture_37))
		(define info_schema_table_rows (__runtime_capture_19 info_schema_selected_databases info_schema_selected_tables))
		(define info_schema_constraint_rows __runtime_capture_20)
		(define information_schema_rows (__runtime_capture_21 info_schema_native_table_rows info_schema_table_rows info_schema_constraint_rows __runtime_capture_33))
		(define tsql_catalog_metadata_rows (__runtime_capture_22 information_schema_rows))
		(define tsql_catalog_public_native_row? (__runtime_capture_23 internal_column_name?))
		(define tsql_catalog_object_name_matches? (__runtime_capture_24 info_schema_name_matches))
		(define tsql_catalog_default_tuple (__runtime_capture_25 tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value))
		(define tsql_catalog_default_tuples (__runtime_capture_26 tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_catalog_default_tuple))
		(define tsql_catalog_key_tuples (__runtime_capture_27 tsql_catalog_kind_selected? tsql_catalog_metadata_rows tsql_catalog_public_native_row? tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value))
		(define tsql_catalog_trigger_tuples (__runtime_capture_28 tsql_catalog_kind_selected? info_schema_selected_tables internal_column_name? tsql_native_relation_handle tsql_source_language_profile tsql_catalog_object_name_matches? tsql_catalog_constraint_id tsql_native_object_id))
		(define tsql_catalog_object_tuples (__runtime_capture_29 info_schema_name_matches tsql_catalog_object_name_matches? tsql_catalog_metadata_rows sql_stored_view_catalog_rows tsql_catalog_public_native_row? tsql_catalog_relation_id tsql_catalog_kind_selected? tsql_catalog_constraint_id tsql_native_object_id tsql_schema_id_value tsql_catalog_default_tuples tsql_source_catalog_entries tsql_source_module_kind tsql_catalog_module_id tsql_catalog_trigger_tuples tsql_catalog_key_tuples definition_none))
		(define tsql_catalog_lookup_object_tuples (__runtime_capture_30 tsql_catalog_object_tuples tsql_catalog_kind_selected? tsql_virtual_catalog_names tsql_catalog_object_name_matches? tsql_object_registry_id))
		(define tsql_catalog_default_definition __runtime_capture_35)
		(define tsql_catalog_default_comment __runtime_capture_36)
		(define object_rows (__runtime_capture_31 tsql_catalog_object_tuples tsql_catalog_lookup_object_tuples))
		(define declaration_rows (__runtime_capture_34 object_rows tsql_catalog_default_tuples tsql_catalog_trigger_tuples tsql_catalog_default_comment tsql_catalog_default_definition))
		(define rows (__runtime_capture_32 tsql_metadata_visible? declaration_rows))
		(if (nil? database)
			(merge (map (info_schema_selected_databases selection) (lambda (database) (rows database kind actor tx selection))))
			(rows database kind actor tx selection)))))
	internal_column_name? info_schema_name_matches tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id tsql_native_object_id_factory tsql_catalog_constraint_id_factory tsql_schema_id_value info_schema_selected_tables_factory info_schema_selected_databases_factory tsql_catalog_kind_selected? tsql_source_language_profile tsql_source_module_kind_factory tsql_source_catalog_entries_factory sql_stored_view_catalog_rows tsql_catalog_relation_id_factory tsql_catalog_module_id_factory tsql_virtual_catalog_names info_schema_native_table_rows_factory info_schema_table_rows_factory info_schema_constraint_rows tsql_catalog_object_metadata_rows_factory tsql_catalog_metadata_rows_factory tsql_catalog_public_native_row?_factory tsql_catalog_object_name_matches?_factory tsql_catalog_default_tuple_factory tsql_catalog_default_tuples_factory tsql_catalog_key_tuples_factory tsql_catalog_trigger_tuples_factory tsql_catalog_object_tuples_factory tsql_catalog_lookup_object_tuples_factory tsql_catalog_object_rows_factory tsql_catalog_authorized_rows_factory info_schema_referential_constraints_for_table tsql_catalog_declaration_rows_factory tsql_catalog_default_definition tsql_catalog_default_comment info_schema_table_row)))
(define tsql_catalog_runtime_query (lambda (database kind columns facts selection) (begin
	(define alias "__tsql_catalog_values")
	(define names (map columns car))
	(define declarations (map columns (lambda (column) (list
		"Field" (car column) "RawType" (cadr column) "Dimensions" (nth column 2) "Null" (nth column 3)
		"Collation" (if (sql_text_type? (cadr column)) "utf8mb4_general_ci" nil)))))
	(define source (list alias (coalesceNil database "system")
		(list 'table-function (if (has? '("objects" "lookup_objects" "sysobjects" "defaults" "syscomments" "triggers"
			"constraints" "foreign_keys" "check_constraints" "key_constraints" "indexes") kind)
			tsql_catalog_declaration_runtime_rows tsql_catalog_runtime_rows) (list database kind (tsql_actor_formula) 'tx selection) names)
		false nil (list "columns" declarations "constraints" '("Unique" () "ForeignKeys" ()) "lifetime" "invocation")))
	(make_query_block (coalesceNil database "system") (list source)
		(tsql_catalog_typed_fields columns (map names (lambda (name) (list 'get_column alias false name false))))
		true nil nil nil nil nil '() '() (cons (list 'fresh_invocation true) facts)))))

(define tsql_catalog_public_projection (lambda (database query columns) (begin
	(define alias "__tsql_catalog_projection")
	(make_query_block database (list (list alias database query false nil nil))
		(merge (map columns (lambda (column) (list (car column) (list 'get_column alias false (car column) false)))))
		true nil nil nil nil nil '() '() (list (list 'fresh_invocation true))))))
(define tsql_catalog_defaults_query (lambda (database selection)
	(tsql_catalog_runtime_query database "defaults"
		(append tsql_default_catalog_columns (list "__database" "NVARCHAR" '(128) false)) '() selection)))
(define tsql_catalog_triggers_query (lambda (database selection)
	(tsql_catalog_runtime_query database "triggers"
		(append tsql_trigger_catalog_columns (list "__database" "NVARCHAR" '(128) false)) '() selection)))
(define tsql_catalog_constraints_query (lambda (database kind selection)
	(tsql_catalog_runtime_query database kind
		(append (if (equal? kind "indexes") tsql_index_catalog_columns tsql_constraint_catalog_columns)
			(list "__database" "NVARCHAR" '(128) false)) '() selection)))
(define tsql_catalog_columns_query (lambda (database include_system)
	(tsql_catalog_runtime_query database (if include_system "all_columns" "columns")
		tsql_catalog_column_declarations (list (list 'tsql_column_catalog (list database include_system))) nil)))
(define tsql_catalog_objects_query (lambda (database planning tx)
	(tsql_catalog_runtime_query database "objects"
		(cadr (find tsql_virtual_catalog_declarations (lambda (catalog) (equal? (car catalog) "objects")) nil))
		(list (list 'tsql_object_catalog database)) nil)))
(define tsql_catalog_address_field (close_procedure ((lambda (parse_name qualifier)
	(lambda (context name options field)
		(if (nil? name) nil (begin
			(define parts (try (lambda () (parse_name name)) (lambda (_invalid_address) nil)))
			(if (nil? parts) nil
				(equal? field "database") (database_name (qualifier (car parts) context options) true)
				(equal? field "owner") (cadr parts)
				(equal? field "name") (nth parts 2)
				(error "unknown object address field"))))))
	tsql_object_name_parts tsql_qualifier_database)))
/* A literal OBJECT_ID address permits a source restriction before identity
construction. Resolve its database at invocation time, including import
mapping. Keep the ordinary SQL predicate as the semantic authority. */
(define tsql_catalog_address_selection (close_procedure ((lambda (address_field)
	(lambda (context name options) (begin
		(define owner (address_field context name options "owner"))
		(define object_name (address_field context name options "name"))
		(if (and (not (nil? owner)) (equal?? owner "dbo"))
			(list "schema" (list "exact" (address_field context name options "database"))
				"object" (list "predicate" (lambda (candidate) (equal?? candidate object_name))))
			false)))) tsql_catalog_address_field)))
/* Bind the authoritative native lookup once, then enter the relational
constraint probe only for SQL NULL. MATCH/IF are existing scalar syntax (as in
CHOOSE), so argument typing and ordinary scalar-probe lowering remain visible.
A lambda in call-head position would hide its body from those SQL visitors. */
(define tsql_catalog_object_id_choice_compiler (lambda (infos) (begin
	(define native (sql_info_formula (car infos)))
	(define constraints (sql_info_formula (cadr infos)))
	(define binding (queryplan_callable_parameter
		(queryplan_code_symbols (list 'list native constraints)) 0))
	(sql_info (list 'match native binding (list 'if (list 'nil? binding) constraints binding)) "INT" nil))))

(define tsql_catalog_unwrap (lambda (expr) (match expr
	((symbol sql_type_expr) formula _type _collation) (tsql_catalog_unwrap formula)
	((symbol sql_compile_column) _compiler formula) (tsql_catalog_unwrap formula)
	_ expr)))
/* Literal namespace proof uses the same metadata-value address grammar as
runtime lookup. Database resolution and authority remain invocation inputs. */
(define tsql_catalog_system_object_literal? (lambda (name) (begin
	(define literal (tsql_catalog_unwrap name))
	(if (string? literal)
		(try (lambda () (equal?? (cadr (tsql_object_name_parts literal)) "sys"))
			(lambda (_invalid_address) false)) false))))
(define tsql_catalog_object_id_query (lambda (database arguments planning tx) (begin
	(define name (tsql_implicit_cast_expr (car arguments) "NVARCHAR" '(776)))
	(define kind (tsql_implicit_cast_expr (cadr arguments) "NVARCHAR" '(2)))
	/* A short literal survives the declared NVARCHAR(2) conversion unchanged.
	The general conversion also carries invocation error options and therefore
	need not be globally foldable. Keep that conversion in the result predicate. */
	(define input_kind (if (string? (tsql_catalog_unwrap (cadr arguments)))
		(try (lambda () (sql_expr_info '() (cadr arguments)))
			(lambda (_unbound_input) nil)) nil))
	(define known (if (and (not (nil? input_kind))
		(has? '("VARCHAR" "NVARCHAR") (sql_info_type input_kind))
		(string? (sql_info_formula input_kind)) (<= (utf16_length (sql_info_formula input_kind)) 2))
		(list true (sql_info_formula input_kind))
		(try (lambda () (list true (sql_constant_value (sql_info_formula (sql_expr_info '() kind)))))
			(lambda (_nonconstant) (list false nil)))))
	/* A system address cannot denote a dbo constraint. Its native lookup is
	still an authoritative fresh producer: unchanged arguments cannot freeze
	membership or declarations inside a reusable grouped payload. */
	(if (tsql_catalog_system_object_literal? (car arguments))
		(list 'inner_select (make_query_block database '()
			(list "object_id" (list 'sql_compile_expr (list 'tsql_compile_object_id database) name kind))
			true nil nil nil nil nil '() '() (list (list 'fresh_invocation true))))
		/* Existing U/V/module lookups stay on their direct declaration owner.
		Only a constraint kind needs the relation below, including PK/UQ. */
		(if (car known) (if (nil? (cadr known)) false
			(not (has? '("C" "D" "F" "PK" "UQ" "TR") (toUpper (cadr known))))) false)
		(list 'sql_compile_expr (list 'tsql_compile_object_id database) name kind)
		(begin
			(define options (list 'list "tsql_import_target" (list 'session "tsql_import_target")
				"tsql_import_source" (list 'session "tsql_import_source")))
			(define address (lambda (field) (list tsql_catalog_address_field database name options field)))
			(define input_name (if (string? (tsql_catalog_unwrap (car arguments)))
				(try (lambda () (sql_expr_info '() (car arguments))) (lambda (_unbound_name) nil)) nil))
			(define selection (if (and (not (nil? input_name))
				(has? '("VARCHAR" "NVARCHAR") (sql_info_type input_name))
				(string? (sql_info_formula input_name)) (<= (utf16_length (sql_info_formula input_name)) 776))
				(list tsql_catalog_address_selection database (sql_info_formula input_name) options) nil))
			(define actor (tsql_actor_formula))
			(define alias (binding_fresh_alias (binding_collect_string_catalog arguments '()) "__tsql_object_id"))
			(define column (lambda (name) (list 'get_column alias false name false)))
			/* Both branches share the actual native declaration shape; a plain
			projection keeps the DEFAULT-only ordinal/definition fields private. */
			(define project (lambda (query) (make_query_block database
				(list (list alias database query false nil nil))
				(merge (map '("name" "object_id" "type" "__database") (lambda (name) (list name (column name)))))
				true nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))
			/* A literal kind selects its declaration family before enumerating
			tables. Existing PK/default probes must not acquire a new trigger scan. */
			(define known_kind (if (and (car known) (not (nil? (cadr known))))
				(toUpper (cadr known)) nil))
			(define trigger_probe (lambda () (project (tsql_catalog_triggers_query nil selection))))
			(define constraints (if (equal? known_kind "TR") (trigger_probe)
				(equal? known_kind "D") (project (tsql_catalog_defaults_query nil selection))
				(has? '("F" "C") known_kind)
				(project (tsql_catalog_constraints_query nil
					(if (equal? known_kind "F") "foreign_keys" "check_constraints") selection))
				(has? '("PK" "UQ") known_kind)
				(project (tsql_catalog_constraints_query nil "key_constraints" selection))
				(make_union_block 'all (merge (list (list
					(project (tsql_catalog_constraints_query nil "constraints" selection))
					(project (tsql_catalog_constraints_query nil "key_constraints" selection))
					(project (tsql_catalog_defaults_query nil selection)))
					(if (and (car known) (not (nil? (cadr known)))) '() (list (trigger_probe)))))
					nil nil nil '())))
			(define constraint_value (list 'inner_select (make_query_block database
				(list (list alias database constraints false nil nil))
				(list "object_id" (column "object_id"))
				(list 'and (list 'equal?? (address "owner") "dbo")
					(list 'equal?? (column "__database") (address "database"))
					(list 'equal?? (column "name") (address "name"))
					(list 'or (list 'nil? kind) (list 'equal?? (column "type") kind)))
				nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))
			/* Preserve current native authority/incarnation checks; the fallback
			is an ordinary fresh scalar probe under the existing lazy IF arm. */
			(list 'sql_compile_expr 'tsql_catalog_object_id_choice_compiler
				(list 'sql_compile_expr (list 'tsql_compile_object_id database) name kind)
				constraint_value))))))
(define tsql_catalog_object_id_factory (close_procedure ((lambda (lookup)
	(lambda (arguments _outer_sources _policy planning tx)
		(lookup (car arguments) (cdr arguments) planning tx))) tsql_catalog_object_id_query)))
(define tsql_catalog_object_id_call (lambda (database name kind) (begin
	(define literal (tsql_catalog_unwrap kind))
	/* Only non-system kinds retain the existing direct U/V/module path.
	The shared factory owns both fresh system lookup and dbo constraints, so
	literal-address narrowing retains its one identifiable binding owner. */
	(if (and (not (tsql_catalog_system_object_literal? name))
		(if (string? literal) (not (has? '("C" "D" "F" "PK" "UQ" "TR") (toUpper literal))) false))
		(list 'sql_compile_expr (list 'tsql_compile_object_id database) name kind)
		(list 'sql_type_expr
			(list 'tsql_relational_scalar_call tsql_catalog_object_id_factory
				(list database name kind)) "INT" nil)))))

/* SQL Server 2000 compatibility views rename actual current catalog fields.
Only represented declarations are exposed; native metadata and DB visibility
remain the same producers as sys.objects/sys.columns, with no SCM row bags.
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-sysobjects-transact-sql
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-syscolumns-transact-sql */
(define tsql_syslogins_catalog_query (lambda (database tx) (begin
	(define alias "__tsql_native_login")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define actor (tsql_actor_formula))
	/* Existing incarnation/permission admission stays authoritative. The
	native account relation supplies only public identity and role inputs;
	password is neither read nor passed to any expression/callback. */
	(define admin (list tsql_catalog_actor_field database actor "admin"))
	(define values (map tsql_syslogins_catalog_columns (lambda (declaration)
		(match (car declaration)
			"sid" (list hex2bin (list replace (column "sql_principal_id") "-" ""))
			"name" (column "username") "loginname" (column "username")
			"createdate" (column "sql_login_created_at") "accdate" (column "sql_login_created_at")
			"updatedate" (column "sql_login_updated_at")
			"sysadmin" (list 'if (column "admin") 1 0)
			"hasaccess" 1
			"denylogin" 0 "isntname" 0 "isntgroup" 0 "isntuser" 0
			"securityadmin" 0 "serveradmin" 0 "setupadmin" 0 "processadmin" 0
			"diskadmin" 0 "dbcreator" 0 "bulkadmin" 0
			_ nil))))
	(make_query_block database
		(list (list alias "system" "user" false nil
			(set_assoc (sql_relation_declarations "system" "user" tx) "lifetime" "invocation")))
		(tsql_catalog_typed_fields tsql_syslogins_catalog_columns values)
		(list 'and (list 'not (list 'nil? (column "username")))
			(list 'not (list 'nil? (column "sql_principal_id")))
			(list 'or admin (list 'and
				(list 'equal? (column "username") (list 'car actor))
				(list 'equal? (column "sql_principal_id") (list 'cadr actor)))))
		nil nil nil nil nil '() '() (list (list 'fresh_invocation true)
			(list 'tsql_output_columns tsql_syslogins_catalog_columns))))))
/* SUSER_SID's SQL declaration is VARBINARY(85), independently of the
native account UUID byte width. Omission denotes the authenticated context;
explicit NULL is a nullable name lookup. No new login/ACL namespace is created.
https://learn.microsoft.com/sql/t-sql/functions/suser-sid-transact-sql */
(define tsql_suser_sid_query (lambda (database arguments _planning tx) (begin
	(define name (car arguments))
	(define alias (binding_fresh_alias (binding_collect_string_catalog name '()) "__tsql_login_sid"))
	(list 'inner_select (make_query_block database
		(list (list alias database (tsql_syslogins_catalog_query database tx) false nil nil))
		(list "sid" (list 'get_column alias false "sid" false))
		(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
			(list 'get_column alias false "name" false) (tsql_cast_expr name "NVARCHAR" '(128)))
		nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))))
(define tsql_suser_sid_call (lambda (database arguments) (begin
	(if (<= (count arguments) 1) true
		(error "SUSER_SID native SQL login profile supports zero or one argument"))
	(list 'sql_type_expr
		(if (empty_list? arguments)
			/* Keep the actual caller's incarnation explicit, including deferred
			DEFAULT execution. The existing owner validates that incarnation. */
			(list tsql_catalog_actor_field database (tsql_actor_formula) "sid")
			(list 'tsql_relational_scalar_call
				(close_procedure ((lambda (database lookup)
					(lambda (arguments _outer_sources _policy planning tx)
						(lookup database arguments planning tx))) database tsql_suser_sid_query)) arguments))
		(list "VARBINARY" '(85)) nil))))

/* Modern and compatibility database views share the native identity and
per-row authority producer. Only their declared SQL projection differs.
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-sysdatabases-transact-sql */
(define tsql_catalog_databases_query (lambda (database) (begin
	(define alias "__tsql_databases")
	(define name (list 'get_column alias false "schema_name" false))
	(make_query_block database
		(list (list alias "information_schema" "schemata" false nil nil))
		(list "name" (list 'sql_type_expr name (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2))
			"database_id" (list 'sql_type_expr (list 'tsql_database_id name) "INT" nil)
			/* Native database metadata has no recorded creator. An unknown SID
			is NULL in both catalog projections, never the querying actor's SID
			or a database identity reinterpreted as an account identity. */
			"owner_sid" (list 'sql_type_expr nil (list "VARBINARY" '(85)) nil)
			"is_auto_close_on" (list 'sql_type_expr false "BIT" nil)
			"is_auto_shrink_on" (list 'sql_type_expr false "BIT" nil)
			"recovery_model_desc" (list 'sql_type_expr "SIMPLE" (list "NVARCHAR" '(60)) nil)
			"is_trustworthy_on" (list 'sql_type_expr (list 'tsql_database_trustworthy name) "BIT" nil))
		(list 'equal? (list 'tsql_has_dbaccess name (tsql_actor_formula)) 1)
		nil nil nil nil nil '() '() '()))))
/* Inverse SID lookup uses the same fresh native login projection. Omission
retains the existing security-context/default expression owner.
https://learn.microsoft.com/sql/t-sql/functions/suser-sname-transact-sql */
(define tsql_suser_sname_query (lambda (database arguments _planning tx) (begin
	(define sid (car arguments))
	(define alias (binding_fresh_alias (binding_collect_string_catalog sid '()) "__tsql_login_name"))
	(list 'inner_select (make_query_block database
		(list (list alias database (tsql_syslogins_catalog_query database tx) false nil nil))
		(list "name" (list 'get_column alias false "name" false))
		(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
			(list 'get_column alias false "sid" false) (tsql_cast_expr sid "VARBINARY" '(85)))
		nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))))
(define tsql_suser_sname_call (lambda (database arguments) (begin
	(if (<= (count arguments) 1) true
		(error "SUSER_SNAME native SQL login profile supports zero or one argument"))
	(list 'sql_type_expr
		(if (empty_list? arguments)
			(list tsql_security_context_name (list 'session "username"))
			(list 'tsql_relational_scalar_call
				(close_procedure ((lambda (database lookup)
					(lambda (arguments _outer_sources _policy planning tx)
						(lookup database arguments planning tx))) database tsql_suser_sname_query)) arguments))
		(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4)))))

(define tsql_legacy_catalog_query (lambda (database name planning tx)
	(if (equal?? name "sysdatabases") (begin
		(define alias "__tsql_legacy_databases")
		(make_query_block database
			(list (list alias database (tsql_catalog_databases_query database) false nil nil))
			(list "name" (list 'get_column alias false "name" false)
				"dbid" (tsql_implicit_cast_expr (list 'get_column alias false "database_id" false) "SMALLINT" '())
				"sid" (list 'get_column alias false "owner_sid" false))
			true nil nil nil nil nil '() '() (list (list 'tsql_legacy_catalog "sysdatabases"))))
		(tsql_catalog_runtime_query database (toLower name)
			(cadr (find tsql_virtual_catalog_declarations (lambda (catalog) (equal?? (car catalog) name)) nil))
			(list (list 'tsql_legacy_catalog (toLower name))) nil))))
(define tsql_catalog_name_column? (lambda (expr alias unqualified) (match (tsql_catalog_unwrap expr)
	((symbol get_column) owner _owner_ic name _name_ic)
	(and (or (and unqualified (nil? owner)) (and (string? owner) (equal?? owner alias))) (equal?? name "name"))
	_ false)))
(define tsql_catalog_name_term (lambda (term alias projection unqualified) (begin
	(define term (tsql_catalog_unwrap term))
	(match term
		((symbol sql_compile_expr) ((symbol tsql_comparison_compiler) "eq") left right)
		(if (and (tsql_catalog_name_column? left alias unqualified) (string? (tsql_catalog_unwrap right)))
			(list 'sql_compile_expr (cadr term) projection right)
			(and (tsql_catalog_name_column? right alias unqualified) (string? (tsql_catalog_unwrap left)))
			(list 'sql_compile_expr (cadr term) left projection)
			nil)
		_ nil))))
/* OBJECT_ID(name,kind) can only return an identity of that kind. This is
independent of the address/name value; neither NULL nor a missing object can
make another kind satisfy the retained equality. OR/NOT are not traversed. */
(define tsql_catalog_object_kind_literal (lambda (expr) (begin
	(define kind (match (tsql_catalog_unwrap expr)
		((symbol sql_compile_expr) ((symbol tsql_compile_object_id) _context) _name kind) kind
		((symbol tsql_relational_scalar_call) factory '(_context _name kind))
		(if (identical? factory tsql_catalog_object_id_factory) kind nil)
		_ nil))
	(define value (tsql_catalog_unwrap kind))
	(if (and (string? value) (<= (strlen value) 2)) (toUpper (sql_rtrim value)) nil))))
(define tsql_catalog_object_id_column? (lambda (expr alias unqualified) (match (tsql_catalog_unwrap expr)
	((symbol get_column) owner _owner_ic name _name_ic)
	(and (or (and unqualified (nil? owner)) (and (string? owner) (equal?? owner alias)))
		(equal?? name "object_id"))
	_ false)))
(define tsql_catalog_necessary_kind (lambda (condition alias unqualified) (match (tsql_catalog_unwrap condition)
	(cons (symbol and) terms)
	(find (map terms (lambda (term) (tsql_catalog_necessary_kind term alias unqualified))) (lambda (kind) kind) nil)
	((symbol sql_compile_expr) ((symbol tsql_comparison_compiler) "eq") left right)
	(if (tsql_catalog_object_id_column? left alias unqualified) (tsql_catalog_object_kind_literal right)
		(tsql_catalog_object_id_column? right alias unqualified) (tsql_catalog_object_kind_literal left) nil)
	_ nil)))
(define tsql_catalog_restrict_name_source (lambda (src condition unqualified) (begin
	(define query (source_relation src))
	(define sources (if (query_block? query) (qb_sources query) '()))
	(define relation (if (equal? (count sources) 1) (source_relation (car sources)) nil))
	(define runtime (match relation
		((symbol table-function) producer arguments _columns)
		(and (or (identical? producer tsql_catalog_runtime_rows)
			(identical? producer tsql_catalog_declaration_runtime_rows)) (has? '("objects" "lookup_objects" "sysobjects" "defaults") (nth arguments 1)))
		_ false))
	(if (and runtime (not (source_outer? src))) (begin
		(define parameter '__catalog_candidate_name)
		(define projection (list 'sql_type_expr parameter (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2)))
		(define predicates (filter (map (split_and_terms condition) (lambda (term)
			(tsql_catalog_name_term term (source_alias src) projection unqualified)))
			(lambda (term) (not (nil? term)))))
		(define kind (tsql_catalog_necessary_kind condition (source_alias src) unqualified))
		(if (and (empty_list? predicates) (nil? kind)) src (begin
			(define predicate (if (empty_list? predicates) nil
				(close_procedure (eval (list 'lambda (list parameter)
					(sql_info_formula (sql_expr_info '() (combine_where_terms predicates true))))))))
			(define arguments (nth relation 2))
			(define current (match (nth arguments 4) ((symbol quote) selection) selection _ nil))
			(define selection (if (expression_equal? current false) false (begin
				(define named (if (nil? predicate) (coalesceNil current '())
					(set_assoc (coalesceNil current '()) "object" (list "predicate" predicate))))
				(if (nil? kind) named (set_assoc named "kinds"
					(filter (list kind) (lambda (kind) (tsql_catalog_kind_selected? current kind))))))))
			(define narrowed (list 'table-function (nth relation 1)
				(mapIndex arguments (lambda (index argument) (if (equal? index 4) (list 'quote selection) argument))) (nth relation 3)))
			(source_with_relation src (make_query_block (qb_schema query)
				(list (source_with_relation (car sources) narrowed)) (qb_fields query) (qb_where query)
				(qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query)
				(qb_hidden query) (qb_stages query) (qb_facts query)))))) src))))

/* This is an address implication, not a substitute for OBJECT_ID equality.
Only a dominating positive equality and a literal address can narrow catalog
production. The original predicate and its authority/incarnation checks stay. */
(define tsql_catalog_object_literal (lambda (expr) (match (tsql_catalog_unwrap expr)
	((symbol sql_compile_expr) ((symbol tsql_compile_object_id) context) name _kind)
	(begin (define value (tsql_catalog_unwrap name))
		(if (string? value) (list context value) nil))
	((symbol tsql_relational_scalar_call) factory '(context name _kind))
	(if (identical? factory tsql_catalog_object_id_factory) (begin
		(define value (tsql_catalog_unwrap name))
		(if (string? value) (list context value) nil)) nil)
	_ nil)))
(define tsql_catalog_id_column? (lambda (expr alias) (match (tsql_catalog_unwrap expr)
	((symbol get_column) owner _owner_ic name _name_ic)
	(and (or (nil? owner) (equal?? owner alias)) (equal?? name "object_id"))
	_ false)))
(define tsql_catalog_literal_scope (lambda (condition alias) (match (tsql_catalog_unwrap condition)
	(cons (symbol and) clauses)
	(find (map clauses (lambda (clause) (tsql_catalog_literal_scope clause alias))) (lambda (scope) scope) nil)
	((symbol sql_compile_expr) ((symbol tsql_comparison_compiler) "eq") left right)
	(if (tsql_catalog_id_column? left alias) (tsql_catalog_object_literal right)
		(tsql_catalog_id_column? right alias) (tsql_catalog_object_literal left) nil)
	_ nil)))
(define tsql_catalog_restrict_branch (lambda (query owner parts) (match query
	((symbol union-block) mode branches ordering limit offset facts)
	(list 'union-block mode (map branches (lambda (branch)
		(tsql_catalog_restrict_branch branch owner parts))) ordering limit offset facts)
	((symbol query-block) database sources fields condition grouping having ordering limit offset hidden stages facts)
	(begin
		(define target_database (car parts))
		(define target_owner (cadr parts))
		(define target_name (nth parts 2))
		(define virtual (qassoc_get facts 'tsql_virtual_catalog nil))
		(define restriction (if virtual
			(and (equal? owner target_database) (equal?? target_owner "sys") (equal?? virtual target_name))
			(and (equal? owner target_database) (equal?? target_owner "dbo") (equal? (count sources) 1))
			(list 'equal?? (list 'get_column (source_alias (car sources)) false "table_name" false) target_name)
			false))
		(define generated (if (equal? (count sources) 1) (source_relation (car sources)) nil))
		(define runtime (match generated
			((symbol table-function) producer _arguments _columns) (or (identical? producer tsql_catalog_runtime_rows)
				(identical? producer tsql_catalog_declaration_runtime_rows))
			_ false))
		(if runtime
			(begin
				(define selector (if (not (equal? owner target_database)) false
					(equal?? target_owner "sys") (list "catalog" target_name)
					(equal?? target_owner "dbo") (list "table" (list "predicate"
						(close_procedure ((lambda (target) (lambda (name) (equal?? name target))) target_name))))
					false))
				(define relation (list 'table-function (nth generated 1)
					(mapIndex (nth generated 2) (lambda (index argument)
						(if (equal? index 4) (list 'quote selector) argument))) (nth generated 3)))
				(make_query_block database (list (source_with_relation (car sources) relation)) fields condition
					grouping having ordering limit offset hidden stages facts))
			(make_query_block database sources fields (combine_where condition restriction)
				grouping having ordering limit offset hidden stages facts)))
	_ query)))
(define tsql_catalog_restrict_source (lambda (src condition options) (begin
	(define query (source_relation src))
	(define facts (if (query_block? query) (qb_facts query)
		(union_block? query) (union_facts query) '()))
	(define tag (qassoc_get facts 'tsql_column_catalog nil))
	(define scope (if (and tag (not (source_outer? src)))
		(tsql_catalog_literal_scope condition (source_alias src)) nil))
	(if (nil? scope) src (begin
		(define parsed (try (lambda () (tsql_object_name_parts (cadr scope))) (lambda (_invalid_address) nil)))
		(if (nil? parsed) src (begin
			(define target_database (database_name (tsql_qualifier_database (car parsed) (car scope) options) true))
			(source_with_relation src (tsql_catalog_restrict_branch query (car tag)
				(list target_database (cadr parsed) (nth parsed 2)))))))))))

/* Authentication and membership are checked when metadata is consumed, not
frozen as authorized flags in a shared query plan. Principal numbers reuse the
lossless compatibility registry with disjoint account-incarnation namespaces;
SIDs contain only the native random account UUID, never credential material. */
(define tsql_catalog_actor_field (close_procedure ((lambda (require_access native_admin registry)
	(lambda (database actor field) (begin
		(require_access actor database)
		(define username (car actor))
		(define incarnation (cadr actor))
		(define admin (native_admin username))
		(match field
			"admin" admin
			"login_name" username
			"account_database_id" (registry database (json_encode (list "database-principal" incarnation)))
			"database_id" (if admin 1 (registry database (json_encode (list "database-principal" incarnation))))
			"login_id" (registry "" (json_encode (list "login-principal" incarnation)))
			"sid" (hex2bin (replace incarnation "-" ""))
			_ (error "unknown principal metadata field")))))
	tsql_require_database_access tsql_native_admin? tsql_object_registry_id)))
(define tsql_catalog_typed_fields (lambda (columns values)
	(merge (mapIndex columns (lambda (index column)
		(list (car column) (list 'sql_type_expr (nth values index)
			(if (empty_list? (nth column 2)) (cadr column) (list (cadr column) (nth column 2)))
			(if (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") (cadr column)) '("utf8mb4_general_ci" 2) nil))))))))
(define tsql_catalog_typed_row (lambda (database columns values condition)
	(make_query_block database '() (tsql_catalog_typed_fields columns values)
		condition nil nil nil nil nil '() '() (list (list 'fresh_invocation true)))))
/* These domains contain only finite catalog declarations. Runtime account
fields remain typed projections over a normal native literal-row source; no
relational SQL results are collected in Scheme or frozen into declarations. */
(define tsql_security_catalog_query (lambda (database kind) (begin
	(define actor (tsql_actor_formula))
	(define field (lambda (name) (list tsql_catalog_actor_field database actor name)))
	(define alias "__tsql_security_catalog")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define principal (equal? kind "database_principals"))
	(define login (equal? kind "server_principals"))
	(define permissions (equal? kind "database_permissions"))
	(define roles (equal? kind "database_role_members"))
	(if (or principal login permissions roles) true (error "unsupported security catalog"))
	(define columns (if (or principal login) tsql_principal_catalog_columns
		permissions tsql_permission_catalog_columns tsql_role_member_catalog_columns))
	(define rows (if principal (list
		(list "__subject" "public" "principal_id" 0 "name" "public" "type" "R" "type_desc" "DATABASE_ROLE" "sid" nil "is_fixed_role" false)
		(list "__subject" "dbo" "principal_id" 1 "name" "dbo" "type" "S" "type_desc" "SQL_USER" "sid" nil "is_fixed_role" false)
		(list "__subject" "account" "principal_id" nil "name" nil "type" "S" "type_desc" "SQL_USER" "sid" nil "is_fixed_role" false))
		login (list (list "__subject" "account" "principal_id" nil "name" nil "type" "S" "type_desc" "SQL_LOGIN" "sid" nil "is_fixed_role" false))
		permissions (map '("CONTROL" "CONNECT") (lambda (permission)
			(list "__subject" "account" "class" 0 "class_desc" "DATABASE" "major_id" 0 "minor_id" 0
				"grantee_principal_id" nil "grantor_principal_id" 1 "type" (if (equal? permission "CONTROL") "CL" "CO")
				"permission_name" permission "state" "G" "state_desc" "GRANT"))) '()))
	(define account (list 'equal? (column "__subject") "account"))
	(define values (map columns (lambda (declaration) (begin
		(define name (car declaration))
		(if (and (or principal login) (equal? name "principal_id"))
			(list 'if account (field (if login "login_id" "account_database_id")) (column name))
			(and (or principal login) (equal? name "name"))
			(list 'if account (field "login_name") (column name))
			(and (or principal login) (equal? name "sid"))
			(list 'if (list 'or account (list 'and (list 'equal? (column "__subject") "dbo") (field "admin")))
				(field "sid") nil)
			(and permissions (equal? name "grantee_principal_id")) (field "database_id")
			(column name))))))
	(define condition (combine_where (list tsql_metadata_visible? actor database false)
		(if principal (list 'or (list 'not account) (list 'not (field "admin"))) true)))
	(make_query_block database
		(list (list alias database (list 'literal-rows (append (map columns car) "__subject") rows) false nil nil))
		(tsql_catalog_typed_fields columns values) condition nil nil nil nil nil '() '()
		/* Native authority may change with identical SQL/session arguments. */
		(list (list 'fresh_invocation true))))))
/* sys.objects describes user objects. OBJECT_ID also recognizes these
immutable virtual catalog declarations, so name lookup includes their current
declared identities without changing the public user-object relation. */
(define tsql_catalog_lookup_objects_query (lambda (database planning tx) (begin
	(define columns (list (list "name" "NVARCHAR" '(128) false) (list "object_id" "INT" '() false)
		(list "type" "CHAR" '(2) false) (list "is_ms_shipped" "BIT" '() false)))
	(tsql_catalog_runtime_query database "lookup_objects" columns '() nil))))
/* These two intrinsic lookups have at most one matching declaration.
Principal IDs 0/1 are disjoint from registry IDs >=10000; the account row always
uses its registry ID, independently of the current admin predicate. Object IDs
have the registry's unique primary key and disjoint native/view/FK/module/catalog
namespaces; current native name uniqueness and FK deduplication yield one row
per identity. The finite principal domain uses the ordinary scalar-cardinality
owner. TOP(1) for the completed object domain is redundant, not suppression of a
scalar-query cardinality error. Restrict this owner to these constructions. */
(define tsql_catalog_name_query (lambda (database kind value planning tx) (begin
	(define principal (equal? kind "principal"))
	(if (or principal (equal? kind "object")) true (error "unsupported catalog name intrinsic"))
	(define query (if principal (tsql_security_catalog_query database "database_principals")
		(tsql_catalog_lookup_objects_query database planning tx)))
	(define key (if principal "principal_id" "object_id"))
	(define alias (binding_fresh_alias (binding_collect_string_catalog value '()) "__tsql_name_lookup"))
	(list 'inner_select (make_query_block database
		(list (list alias database query false nil nil))
		(list "name" (list 'get_column alias false "name" false))
		(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
			(list 'get_column alias false key false) (tsql_implicit_cast_expr value "INT" '()))
		nil nil nil (if principal nil 1) nil '() '() '())))))

/* Intrinsic arguments belong to the calling SQL scope. Reuse relational
call expansion so unqualified names bind before a catalog source with the
same column name is introduced by the lookup. The factory keeps syntax only.
Its public sysname contract belongs to this frontend constructor, before
CASE/UNION type resolution and independently of the delayed catalog query. */
(define tsql_catalog_name_call (lambda (database kind value)
	(list (quote sql_type_expr)
		/* A supplied literal NULL has no object name. Keep its protected
		marker and declared sysname contract without constructing a lookup. */
		(if (and (equal? kind "object")
			(or (nil? value) (logical_sql_null_literal? value))) value
			(list (quote tsql_relational_scalar_call)
				(close_procedure ((lambda (database kind lookup)
					(lambda (arguments _outer_sources _policy planning tx)
						(lookup database kind (car arguments) planning tx)))
					database kind tsql_catalog_name_query))
				(list value)))
		(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4))))

/* OBJECTPROPERTY classification reads exactly the current catalog object kind,
including native identity, source declarations and the finite sys-view directory.
Neither a stale registry ID nor database access alone establishes existence.
https://learn.microsoft.com/en-us/sql/t-sql/functions/objectproperty-transact-sql
SQL Server 2022 reference: https://dbfiddle.uk/inz585N_ */
(define tsql_object_property_classifications
	(list "ischeckcnst" '("C") "isconstraint" '("C" "D" "F" "PK" "UQ")
		"isdefaultcnst" '("D") "isextendedproc" '("X")
		"isforeignkey" '("F") "isinlinefunction" '("IF") "isprimarykey" '("PK")
		"isprocedure" '("P" "PC" "X" "RF") "isqueue" '("SQ") "isreplproc" '("RF")
		"isrule" '("R") "isscalarfunction" '("FN" "FS") "issystemtable" '("S")
		"istable" '("U" "S" "IT") "istablefunction" '("IF" "TF" "FT")
		"istrigger" '("TR" "TA") "isuniquecnst" '("UQ") "isusertable" '("U")
		"isview" '("V") "isexecuted" '("V" "P" "PC" "X" "RF" "FN" "FS" "IF" "TF" "FT" "TR" "TA")))
(define tsql_object_property_pending '("cnstisclustkey" "cnstiscolumn" "cnstisdeletecascade"
	"cnstisdisabled" "cnstisnonclustkey" "cnstisnotrepl" "cnstisnottrusted" "cnstisupdatecascade"
	"execisaftertrigger" "execisansinullson" "execisdeletetrigger" "execisfirstdeletetrigger"
	"execisfirstinserttrigger" "execisfirstupdatetrigger" "execisinserttrigger" "execisinsteadoftrigger"
	"execislastdeletetrigger" "execislastinserttrigger" "execislastupdatetrigger" "execisquotedidenton"
	"execisstartup" "execistriggerdisabled" "execistriggernotforrepl" "execisupdatetrigger" "execiswithnativecompilation"
	"hasaftertrigger" "hasdeletetrigger" "hasinserttrigger" "hasinsteadoftrigger" "hasupdatetrigger"
	"isansinullson" "isdefault" "isdeterministic" "isencrypted" "isindexed" "isindexable" "isquotedidenton"
	"isschemabound" "issystemverified" "ownerid" "schemaid" "tabledeletetrigger" "tabledeletetriggercount"
	"tablefulltextmergestatus" "tablefulltextbackgroundupdateindexon" "tablefulltextcatalogid"
	"tablefulltextchangetrackingon" "tablefulltextdocsprocessed" "tablefulltextfailcount" "tablefulltextitemcount"
	"tablefulltextkeycolumn" "tablefulltextpendingchanges" "tablefulltextpopulatestatus" "tablehasactivefulltextindex"
	"tablehascheckcnst" "tablehasclustindex" "tablehasdefaultcnst" "tablehasdeletetrigger" "tablehasforeignkey"
	"tablehasforeignref" "tablehasidentity" "tablehasindex" "tablehasinserttrigger" "tablehasnonclustindex"
	"tablehasprimarykey" "tablehasrowguidcol" "tablehastextimage" "tablehastimestamp" "tablehasuniquecnst"
	"tablehasupdatetrigger" "tablehasvardecimalstorageformat" "tableinserttrigger" "tableinserttriggercount"
	"tableisfake" "tableislockedonbulkload" "tableismemoryoptimized" "tableispinned" "tabletextinrowlimit"
	"tableupdatetrigger" "tableupdatetriggercount" "tablehascolumnset" "tabletemporaltype"))
(define tsql_object_property_value (close_procedure ((lambda (classifications pending)
	(lambda (kind shipped property)
		(if (nil? property) nil (begin
			(define property (toLower (regexp_replace property " +$" "")))
			(define kinds (classifications property))
			(if kinds (if (has? kinds kind) 1 0)
				(equal? property "ismsshipped") (if shipped 1 0)
				(has? pending property) (error "OBJECTPROPERTY metadata is not implemented for " property)
				nil))))) tsql_object_property_classifications tsql_object_property_pending)))
(define tsql_catalog_property_query (lambda (database object property planning tx) (begin
	(define query (tsql_catalog_lookup_objects_query database planning tx))
	(define alias (binding_fresh_alias (binding_collect_string_catalog (list object property) '()) "__tsql_property_lookup"))
	(define column (lambda (name) (list 'get_column alias false name false)))
	(list 'inner_select (make_query_block database
		(list (list alias database query false nil nil))
		(list "value" (list 'sql_type_expr (list tsql_object_property_value
			(column "type") (column "is_ms_shipped") (tsql_implicit_cast_expr property "NVARCHAR" '(128))) "INT" nil))
		(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
			(column "object_id") (tsql_implicit_cast_expr object "INT" '()))
		nil nil nil nil nil '() '() '())))))
/* The producer's stable object identity functionally determines these
classification fields. Its current-invocation producer already enforces database
membership and login incarnation; rechecking worker-local accounts would violate
that authority. Reuse only a direct bound object_id from that exact current-database
catalog, retaining the NULL-extension guard. Computed and ordinary inputs use lookup. */
(define tsql_catalog_property_from_row (lambda (database object property sources)
	(match (tsql_catalog_unwrap object)
		((symbol get_column) owner _owner_ic "object_id" _column_ic) (begin
			(define source (find sources (lambda (src) (equal? (source_alias src) owner)) nil))
			(define relation (if (nil? source) nil (source_relation source)))
			(define contract (if (union_block? relation)
				(qassoc_get (union_facts relation) 'tsql_object_catalog nil)
				(query_block? relation) (qassoc_get (qb_facts relation) 'tsql_object_catalog nil) nil))
			(if (equal? contract database)
				(list 'sql_type_expr (list 'if
					(list 'not (list 'nil? object))
					(list tsql_object_property_value
						(list 'get_column owner false "type" false)
						(list 'get_column owner false "is_ms_shipped" false)
						(tsql_implicit_cast_expr property "NVARCHAR" '(128))) nil) "INT" nil) nil))
		((quote get_column) owner owner_ic "object_id" column_ic)
		(tsql_catalog_property_from_row database (list (symbol "get_column") owner owner_ic "object_id" column_ic) property sources)
		_ nil)))
(define tsql_catalog_property_call (lambda (database object property)
	(list 'sql_type_expr
		(list 'tsql_relational_scalar_call
			(close_procedure ((lambda (database lookup from_row)
				(lambda (arguments outer_sources _policy planning tx) (begin
					(define direct (from_row database (car arguments) (cadr arguments) outer_sources))
					(if (nil? direct) (lookup database (car arguments) (cadr arguments) planning tx) direct))))
				database tsql_catalog_property_query tsql_catalog_property_from_row))
			(list object property)) "INT" nil)))

/* All forms use the current visible catalog. A column must exist in that
object's real metadata; an unknown ID/name or a retired identity returns NULL.
Native membership gives the complete database capability set. System catalog
views remain read-only declarations, independently of the account's authority.
SQL Server 2022 reference: https://dbfiddle.uk/ZelE9QQJ */
(define tsql_permissions_column (close_procedure ((lambda (access)
	(lambda (id) (begin
		/* The visible column producer already established current existence and
		database authority. Its registry ID has one durable namespace record. */
		(define incarnation (scan_lookup nil (table "system" false "tsql_object_ids" false nil) access (list id)))
		(if (regexp_test incarnation "^catalog:sys\\.") 1 7))))
	(list (compiled_scan_access_header 1 "value" 1 -1)
		(scan_boundary "equal" "id" 0 0 true true "" false) "incarnation"))))
(define tsql_catalog_permissions_query (lambda (database arguments planning tx) (begin
	(if (<= (count arguments) 2) true (error "PERMISSIONS accepts at most two arguments"))
	(define alias (binding_fresh_alias (binding_collect_string_catalog arguments '()) "__tsql_permissions_object"))
	(define column (lambda (name) (list 'get_column alias false name false)))
	(if (empty_list? arguments)
		(list 'inner_select (make_query_block database
			(list (list alias database (tsql_security_catalog_query database "database_permissions") false nil nil))
			(list "value" (list 'sql_type_expr
				(list tsql_permissions_statements database (tsql_actor_formula)) "INT" nil))
			(list 'equal? (column "permission_name") "CONTROL")
			nil nil nil nil nil '() '() '()))
		(equal? (count arguments) 2) (begin
			/* Column metadata itself owns both object identity and column
			existence. Joining it back to every object-catalog UNION branch
			would repeat the same current metadata domain unnecessarily. */
			(list 'inner_select (make_query_block database
				(list (list alias database (tsql_catalog_columns_query database true) false nil nil))
				(list "value" (list 'sql_type_expr (list tsql_permissions_column (column "object_id")) "INT" nil))
				(list 'and
					(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
						(column "object_id") (tsql_implicit_cast_expr (car arguments) "INT" '()))
					(list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
						(column "name") (tsql_implicit_cast_expr (cadr arguments) "NVARCHAR" '(128))))
				nil nil nil nil nil '() '() '())))
		(begin
			(define objects (tsql_catalog_lookup_objects_query database planning tx))
			(define sources (list (list alias database objects false nil nil)))
			(define condition (list 'sql_compile_expr (list 'tsql_comparison_compiler "eq")
				(column "object_id") (tsql_implicit_cast_expr (car arguments) "INT" '())))
			(define expression (list tsql_permissions_object (column "type") (column "is_ms_shipped")))
			(list 'inner_select (make_query_block database sources
				(list "value" (list 'sql_type_expr expression "INT" nil))
				condition nil nil nil nil nil '() '() '())))))))
(define tsql_catalog_permissions_call (lambda (database arguments)
	(list 'sql_type_expr
		(list 'tsql_relational_scalar_call
			(close_procedure ((lambda (database lookup)
				(lambda (arguments _outer_sources _policy planning tx)
					(lookup database arguments planning tx))) database tsql_catalog_permissions_query))
			arguments) "INT" nil)))

/* Register actual values after their lexical captures have been closed. The
common final-code publisher owns sharing across every SQL frontend and scalar
compiler; logical metadata expressions continue to carry real callables. */
(register_queryplan_callables (list tsql_catalog_runtime_rows tsql_catalog_declaration_runtime_rows tsql_actor_value tsql_object_id_value
	tsql_native_object_id tsql_metadata_visible? tsql_object_registry_id
	tsql_catalog_relation_id tsql_catalog_constraint_id tsql_catalog_module_id
	tsql_catalog_default_field tsql_catalog_default_comment tsql_catalog_address_field
	tsql_catalog_actor_field tsql_object_property_value tsql_permissions_statements
	tsql_permissions_object tsql_permissions_column))
