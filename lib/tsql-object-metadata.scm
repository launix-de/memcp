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
		(scan_boundary "equal" "database" 0 0 true true "" false)
		(scan_boundary "equal" "incarnation" 1 1 true true "" false) "id"))))
/* SQL source kinds are frontend declaration facts shared by execution and
catalog reflection. Import this classifier before closing metadata helpers. */
(define tsql_source_module_kind (close_procedure ((lambda (profile)
	(lambda (language)
		(if (profile "tsql-procedure" language) "P"
			(profile "tsql-table-function" language) "TF"
			(or (profile "tsql-scalar-function" language)
				(profile "tsql-relational-scalar-function" language)) "FN" nil)))
	tsql_source_language_profile)))
(define tsql_source_catalog_entries (close_procedure ((lambda (classify internal_name)
	(lambda (database tx)
		(map (filter (showprocedures database "" tx) (lambda (definition)
			(and (not (nil? (classify (definition "language"))))
				(not (internal_name (definition "name"))))))
			(lambda (definition)
				(list "name" (definition "name") "language" (definition "language"))))))
	tsql_source_module_kind internal_column_name?)))
(define tsql_catalog_module_id (close_procedure ((lambda (registry)
	(lambda (database name kind)
		(registry database (json_encode (list "source" kind name))))) tsql_object_registry_id)))
(define tsql_catalog_module_definition (lambda (database name tx)
	(find (showprocedures database (toLower name) tx) (lambda (_definition) true) nil)))

(define tsql_legacy_catalog_names '("sysobjects" "syscolumns"))
(define tsql_legacy_catalog_name? (close_procedure ((lambda (names)
	(lambda (name) (has? names (toLower name)))) tsql_legacy_catalog_names)))
(define tsql_virtual_catalog_names '("databases" "objects" "columns" "all_columns" "sysobjects" "syscolumns"
	"database_permissions" "database_principals" "server_principals" "database_role_members"))
(define tsql_native_object_id (close_procedure ((lambda (visible relation registry)
	(lambda (database name actor tx)
		(if (visible actor database true) (begin
			(define handle (relation database name tx))
			(if (nil? handle) nil (begin
				(define identity (show handle "identity" tx))
				(if (nil? identity) (error "stable native relation identity is unavailable")
					(registry database identity))))) nil)))
	tsql_metadata_visible? tsql_native_relation_handle tsql_object_registry_id)))
/* These slots identify existing frontend declarations, not fabricated native
incarnations. A dropped/re-added view or same-named FK in the same containing
native table may reuse its slot; current catalog visibility supplies existence.
The shared registry keeps them disjoint from native UUIDs and virtual catalogs. */
(define tsql_catalog_relation_id (close_procedure ((lambda (native_id registry visible)
	(lambda (database name kind actor tx)
		(if (visible actor database true)
			(if (equal? kind "VIEW") (registry database (json_encode (list "view" name)))
				(native_id database name actor tx)) nil)))
	tsql_native_object_id tsql_object_registry_id tsql_metadata_visible?)))
(define tsql_catalog_foreign_id (close_procedure ((lambda (relation registry visible)
	(lambda (database containing name actor tx)
		(if (visible actor database true) (begin
			(define handle (relation database containing tx))
			(if (nil? handle) nil (begin
				(define identity (show handle "identity" tx))
				(if (nil? identity) (error "stable containing relation identity is unavailable")
					(registry database (json_encode (list "foreign" identity name))))))) nil)))
	tsql_native_relation_handle tsql_object_registry_id tsql_metadata_visible?)))
/* This frontend currently exposes ordinary objects in the dbo namespace.
Unknown IDs and SQL NULL are not aliases of that supported schema. */
(define tsql_schema_name_value (lambda (id)
	(if (nil? id) nil (equal? id 1) "dbo" nil)))

/* An address value is one complete multipart identifier, not a SQL prefix.
Ordinary surrounding whitespace is lexical; comments are not address suffixes. */
/* Address values are scanned at row execution, where native parser/regexp
objects cannot be captured in a transferable SQL callable. Keep the same
identifier boundaries using plain strings, including doubled quote escapes. */
(define tsql_object_name_parts (close_procedure ((lambda (private_name regular_name)
	(lambda (text) (begin
		/* State: byte position, lexical mode, current part, prior parts,
		quoted-part flag, validity. ASCII delimiters preserve UTF-8 bytes. */
		(define state (for (list 0 0 "" '() false true)
			(lambda (position _mode _part _parts _quoted valid)
				(and valid (< position (strlen text))))
			(lambda (position mode part parts quoted valid) (begin
				(define character (substr text position 1))
				(define following (if (< (+ position 1) (strlen text)) (substr text (+ position 1) 1) ""))
				(define whitespace (has? '(" " "\t" "\r" "\n") character))
				(if (or (equal? mode 2) (equal? mode 3)) (begin
					(define delimiter (if (equal? mode 2) "]" "\""))
					(if (equal? character delimiter)
						(if (equal? following delimiter)
							(list (+ position 2) mode (concat part delimiter) parts quoted true)
							(list (+ position 1) 4 part parts quoted (not (equal? part ""))))
						(list (+ position 1) mode (concat part character) parts quoted true)))
					(equal? character ".")
					(list (+ position 1) 0 "" (append parts (list part quoted)) false true)
					whitespace (list (+ position 1) (if (equal? mode 0) 0 4) part parts quoted true)
					(equal? mode 4) (list position mode part parts quoted false)
					(and (equal? mode 0) (or (equal? character "[") (equal? character "\"")))
					(list (+ position 1) (if (equal? character "[") 2 3) part parts true true)
					(list (+ position 1) 1 (concat part character) parts false true))))))
		(define mode (nth state 1))
		(define entries (append (nth state 3) (list (nth state 2) (nth state 4))))
		(define valid (and (nth state 5) (not (or (equal? mode 2) (equal? mode 3)))
			(<= (count entries) 3)
			(reduce entries (lambda (valid entry) (begin
				(define name (car entry))
				(and valid (not (private_name name))
					(or (cadr entry) (equal? name "")
						(regular_name name))))) true)))
		(define names (map entries car))
		(if (not valid) (error "invalid object address")
			(equal? (count names) 1)
			(if (equal? (car names) "") (error "invalid object address") (list nil "dbo" (car names)))
			(equal? (count names) 2)
			(if (or (equal? (car names) "") (equal? (cadr names) "")) (error "invalid object address")
				(list nil (car names) (cadr names)))
			(if (or (equal? (car names) "") (equal? (nth names 2) "")) (error "invalid object address")
				(list (car names) (if (equal? (cadr names) "") "dbo" (cadr names)) (nth names 2)))))))
	sql_private_relation? tsql_regular_identifier?)))
(define tsql_object_id_value (close_procedure ((lambda (parse_name qualifier visible native_id view_name registry catalogs module_definition module_kind module_id legacy_name?)
	(lambda (context name kind actor tx options)
		(if (nil? name) nil (begin
			(define parts (try (lambda () (parse_name name)) (lambda (_invalid_address) nil)))
			(if (nil? parts) nil (begin
				(define database (database_name (qualifier (car parts) context options) true))
				(define owner (cadr parts))
				(define object (nth parts 2))
				(if (or (nil? database) (not (visible actor database false))) nil
					(equal?? owner "sys")
					(if (and (or (nil? kind) (equal?? kind "V")) (has? catalogs (toLower object)))
						(registry database (concat "catalog:sys." (toLower object))) nil)
					(not (equal?? owner "dbo")) nil
					(begin
						(define native (if (or (nil? kind) (equal?? kind "U"))
							(native_id database object actor tx) nil))
						(define view (if (or (nil? kind) (equal?? kind "V"))
							(view_name database object) nil))
						(define module (if (or (nil? kind) (equal?? kind "P") (equal?? kind "FN") (equal?? kind "TF"))
							(module_definition database object tx) nil))
						(define declared_kind (if (nil? module) nil (module_kind (module "language"))))
						(if (and native (or (nil? kind) (equal?? kind "U"))) native
							(and (not (nil? view)) (or (nil? kind) (equal?? kind "V")))
							(registry database (json_encode (list "view" view)))
							(and (not (nil? declared_kind)) (or (nil? kind) (equal?? kind declared_kind)))
							(module_id database (module "name") declared_kind)
							(and (or (nil? kind) (equal?? kind "V")) (legacy_name? object))
							(registry database (concat "catalog:sys." (toLower object)))
							nil)))))))))
	tsql_object_name_parts tsql_qualifier_database tsql_metadata_visible? tsql_native_object_id
	tsql_view_name tsql_object_registry_id tsql_virtual_catalog_names tsql_catalog_module_definition
	tsql_source_module_kind tsql_catalog_module_id tsql_legacy_catalog_name?)))

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
		(list "column_id" "INT" '() false) (list "graph_type" "INT" '() true)))
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
(define tsql_legacy_catalog_declarations
	(list (list "sysobjects" (list
		(list "name" "NVARCHAR" '(128) false) (list "id" "INT" '() false)
		(list "xtype" "CHAR" '(2) false) (list "type" "CHAR" '(2) false)
		(list "uid" "SMALLINT" '() false) (list "parent_obj" "INT" '() false)))
		(list "syscolumns" (list (list "name" "NVARCHAR" '(128) false)
			(list "id" "INT" '() false) (list "colid" "SMALLINT" '() false)))))
(define tsql_virtual_catalog_declarations
	(merge (list tsql_legacy_catalog_declarations (list (list "database_permissions" tsql_permission_catalog_columns)
		(list "database_principals" tsql_principal_catalog_columns)
		(list "server_principals" tsql_principal_catalog_columns)
		(list "database_role_members" tsql_role_member_catalog_columns)
		(list "columns" tsql_catalog_column_declarations)
		(list "all_columns" tsql_catalog_column_declarations)
		(list "objects" (list (list "name" "NVARCHAR" '(128) false)
			(list "type" "CHAR" '(2) false) (list "type_desc" "NVARCHAR" '(60) false)
			(list "object_id" "INT" '() false) (list "parent_object_id" "INT" '() false)
			(list "schema_id" "INT" '() false) (list "is_ms_shipped" "BIT" '() false)
			(list "modify_date" "DATETIME" '() true)))
		(list "databases" (list (list "name" "NVARCHAR" '(128) false)
			(list "database_id" "INT" '() false) (list "is_auto_close_on" "BIT" '() false)
			(list "is_auto_shrink_on" "BIT" '() false) (list "recovery_model_desc" "NVARCHAR" '(60) false)
			(list "is_trustworthy_on" "BIT" '() false)))))))
(define tsql_catalog_columns_query (lambda (database include_system) (begin
	(define alias "__tsql_native_columns")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define actor (tsql_actor_formula))
	(define tag (list (list 'tsql_column_catalog (list database include_system))))
	(define fields (list
		"object_id" (list 'sql_type_expr (list tsql_native_object_id database (column "table_name") actor 'tx) "INT" nil)
		"name" (list 'sql_type_expr (column "column_name") (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2))
		"column_id" (list 'sql_type_expr (column "ordinal_position") "INT" nil)
		"graph_type" (list 'sql_type_expr nil "INT" nil)))
	(define source (make_query_block database
		(list (list alias "information_schema" "columns" false nil nil)) fields
		(list 'and (list 'equal? (column "table_schema") database)
			(list 'not (list internal_column_name? (column "table_name")))
			(list 'not (list internal_column_name? (column "column_name")))
			(list tsql_metadata_visible? actor database true))
		nil nil nil nil nil '() '() (if include_system '() tag)))
	(if (not include_system) source (begin
		/* Each finite catalog declaration is one constant row domain. Repeating
		the runtime identity/authority projection once per column would duplicate
		its closed procedure code throughout the UNION plan. The ordinary native
		literal-row scan keeps that projection on its existing SQL boundary. */
		(define self (map tsql_virtual_catalog_declarations (lambda (catalog) (begin
			(define alias "__tsql_virtual_columns")
			(define rows (mapIndex (cadr catalog) (lambda (i declaration)
				(list "name" (car declaration) "column_id" (+ i 1) "graph_type" nil))))
			(define column (lambda (name) (list 'get_column alias false name false)))
			(make_query_block database
				(list (list alias database (list 'literal-rows '("name" "column_id" "graph_type") rows) false nil nil))
				(list "object_id" (list 'sql_type_expr
					(list tsql_object_registry_id database (concat "catalog:sys." (car catalog))) "INT" nil)
					"name" (list 'sql_type_expr (column "name") (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2))
					"column_id" (list 'sql_type_expr (column "column_id") "INT" nil)
					"graph_type" (list 'sql_type_expr (column "graph_type") "INT" nil))
				(list tsql_metadata_visible? actor database false)
				nil nil nil nil nil '() '() (list (list 'tsql_virtual_catalog (car catalog))))))))
		(list 'union-block 'all (cons source self) nil nil nil tag))))))

/* Objects consume the same native relation and declaration producers as other
catalog views. Only the referencing endpoint owns an FK row; the generic
referential producer already excludes incoming mirrors and deduplicates self.
Native declarations do not record historical DDL timestamps. Expose that missing
history as a nullable SQL DATETIME, consistently for relations, constraints and
source modules; current time or filesystem metadata would invent a history. */
(define tsql_catalog_objects_query (lambda (database planning tx) (begin
	(define modules (tsql_source_catalog_entries database tx))
	(planner_record_guard_condition
		(list 'expression_equal? (list tsql_source_catalog_entries database 'tx)
			(list 'quote modules)) planning)
	(define alias "__tsql_objects")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define actor (tsql_actor_formula))
	(define name (column "table_name"))
	(define kind (column "table_type"))
	(define is_view (list 'equal? kind "VIEW"))
	(define relations (make_query_block database
		(list (list alias "information_schema" "tables" false nil nil))
		(list "name" (list 'sql_type_expr name (list "NVARCHAR" (list 128)) '("utf8mb4_general_ci" 2))
			"type" (list 'sql_type_expr (list 'if is_view "V" "U") (list "CHAR" (list 2)) nil)
			"type_desc" (list 'sql_type_expr (list 'if is_view "VIEW" "USER_TABLE") (list "NVARCHAR" (list 60)) '("utf8mb4_general_ci" 2))
			"object_id" (list 'sql_type_expr (list tsql_catalog_relation_id database name kind actor 'tx) "INT" nil)
			"parent_object_id" (list 'sql_type_expr 0 "INT" nil)
			"schema_id" (list 'sql_type_expr 1 "INT" nil)
			"is_ms_shipped" (list 'sql_type_expr false "BIT" nil)
			"modify_date" (list 'sql_type_expr nil "DATETIME" nil))
		(list 'and (list 'equal? (column "table_schema") database)
			(list 'not (list internal_column_name? name))
			(list tsql_metadata_visible? actor database true))
		nil nil nil nil nil '() '() '()))
	(define alias "__tsql_foreign_objects")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define containing (column "table_name"))
	(define name (column "constraint_name"))
	(define foreign (make_query_block database
		(list (list alias "information_schema" "referential_constraints" false nil nil))
		(list "name" (list 'sql_type_expr name (list "NVARCHAR" (list 128)) '("utf8mb4_general_ci" 2))
			"type" (list 'sql_type_expr "F" (list "CHAR" (list 2)) nil)
			"type_desc" (list 'sql_type_expr "FOREIGN_KEY_CONSTRAINT" (list "NVARCHAR" (list 60)) '("utf8mb4_general_ci" 2))
			"object_id" (list 'sql_type_expr (list tsql_catalog_foreign_id database containing name actor 'tx) "INT" nil)
			"parent_object_id" (list 'sql_type_expr (list tsql_native_object_id database containing actor 'tx) "INT" nil)
			"schema_id" (list 'sql_type_expr 1 "INT" nil)
			"is_ms_shipped" (list 'sql_type_expr false "BIT" nil)
			"modify_date" (list 'sql_type_expr nil "DATETIME" nil))
		(list 'and (list 'equal? (column "constraint_schema") database)
			(list 'not (list internal_column_name? containing))
			(list tsql_metadata_visible? actor database true))
		nil nil nil nil nil '() '() '()))
	(define alias "__tsql_source_objects")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define rows (map modules (lambda (definition)
		(list "name" (definition "name") "type" (tsql_source_module_kind (definition "language"))))))
	(define sources (make_query_block database
		(list (list alias database (list 'literal-rows '("name" "type") rows) false nil nil))
		(list "name" (list 'sql_type_expr (column "name") (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 2))
			"type" (list 'sql_type_expr (column "type") (list "CHAR" '(2)) nil)
			"type_desc" (list 'sql_type_expr (list 'if (list 'equal? (column "type") "P")
				"SQL_STORED_PROCEDURE" (list 'equal? (column "type") "TF")
				"SQL_TABLE_VALUED_FUNCTION" "SQL_SCALAR_FUNCTION") (list "NVARCHAR" '(60)) '("utf8mb4_general_ci" 2))
			"object_id" (list 'sql_type_expr (list tsql_catalog_module_id database (column "name") (column "type")) "INT" nil)
			"parent_object_id" (list 'sql_type_expr 0 "INT" nil)
			"schema_id" (list 'sql_type_expr 1 "INT" nil)
			"is_ms_shipped" (list 'sql_type_expr false "BIT" nil)
			"modify_date" (list 'sql_type_expr nil "DATETIME" nil))
		(list tsql_metadata_visible? actor database false)
		nil nil nil nil nil '() '() '()))
	/* This contract names the authoritative row producer, not column names
	from arbitrary SQL inputs. Its fields share the statement's native metadata
	transaction with intrinsic lookups constructed from this same owner. */
	(make_union_block 'all (list relations foreign sources) nil nil nil
		(list (list 'tsql_object_catalog database))))))

/* SQL Server 2000 compatibility views rename actual current catalog fields.
Only represented declarations are exposed; native metadata and DB visibility
remain the same producers as sys.objects/sys.columns, with no SCM row bags.
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-sysobjects-transact-sql
https://learn.microsoft.com/sql/relational-databases/system-compatibility-views/sys-syscolumns-transact-sql */
(define tsql_legacy_catalog_query (lambda (database name planning tx) (begin
	(define objects (equal?? name "sysobjects"))
	(define alias "__tsql_legacy_catalog")
	(define column (lambda (name) (list 'get_column alias false name false)))
	(define fields (if objects
		(list "name" (column "name") "id" (column "object_id")
			"xtype" (column "type")
			"type" (list 'sql_type_expr
				(list 'if (list 'has? (list 'quote '("PK" "UQ")) (column "type")) "K" (column "type"))
				(list "CHAR" '(2)) nil)
			"uid" (tsql_implicit_cast_expr (column "schema_id") "SMALLINT" '())
			"parent_obj" (column "parent_object_id"))
		(list "name" (column "name") "id" (column "object_id")
			"colid" (tsql_implicit_cast_expr (column "column_id") "SMALLINT" '()))))
	(make_query_block database
		(list (list alias database
			(if objects (tsql_catalog_objects_query database planning tx)
				(tsql_catalog_columns_query database false)) false nil nil))
		fields true nil nil nil nil nil '() '() '()))))

/* This is an address implication, not a substitute for OBJECT_ID equality.
Only a dominating positive equality and a literal address can narrow catalog
production. The original predicate and its authority/incarnation checks stay. */
(define tsql_catalog_unwrap (lambda (expr) (match expr
	((symbol sql_type_expr) formula _type _collation) (tsql_catalog_unwrap formula)
	((symbol sql_compile_column) _compiler formula) (tsql_catalog_unwrap formula)
	_ expr)))
(define tsql_catalog_object_literal (lambda (expr) (match (tsql_catalog_unwrap expr)
	((symbol sql_compile_expr) ((symbol tsql_compile_object_id) context) name _kind)
	(begin (define value (tsql_catalog_unwrap name))
		(if (string? value) (list context value) nil))
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
		(make_query_block database sources fields (combine_where condition restriction)
			grouping having ordering limit offset hidden stages facts))
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
	(define alias "__tsql_object_name_source")
	(define user_objects (make_query_block database
		(list (list alias database (tsql_catalog_objects_query database planning tx) false nil nil))
		(list "name" (list 'get_column alias false "name" false)
			"object_id" (list 'get_column alias false "object_id" false)
			"type" (list 'get_column alias false "type" false)
			"is_ms_shipped" (list 'get_column alias false "is_ms_shipped" false))
		true nil nil nil nil nil '() '() '()))
	(define columns (list (list "name" "NVARCHAR" '(128) false) (list "object_id" "INT" '() false)
		(list "type" "CHAR" '(2) false) (list "is_ms_shipped" "BIT" '() false)))
	/* The catalog directory is one finite native row domain, as in
	all_columns. Its authority and identity projection belongs to that domain,
	not a duplicated closed procedure in every constant UNION branch. */
	(define virtual_alias "__tsql_virtual_object_names")
	(define name (list 'get_column virtual_alias false "name" false))
	(define virtuals (make_query_block database
		(list (list virtual_alias database (list 'literal-rows '("name")
			(map tsql_virtual_catalog_names (lambda (name) (list "name" name)))) false nil nil))
		(tsql_catalog_typed_fields columns
			(list name (list tsql_object_registry_id database (list 'concat "catalog:sys." name)) "V" true))
		(list tsql_metadata_visible? (tsql_actor_formula) database false)
		nil nil nil nil nil '() '() (list (list 'fresh_invocation true))))
	(make_union_block 'all (list user_objects virtuals) nil nil nil '()))))
/* These two intrinsic lookups have at most one matching declaration.
Principal IDs 0/1 are disjoint from registry IDs >=10000; the account row always
uses its registry ID, independently of the current admin predicate. Object IDs
have the registry's unique primary key and disjoint native/view/FK/module/catalog
namespaces; current native name uniqueness and FK deduplication yield one row
per identity. The finite principal domain uses the ordinary scalar-cardinality
owner. TOP(1) for the completed object UNION is redundant, not suppression of a
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
		(list (quote tsql_relational_scalar_call)
			(close_procedure ((lambda (database kind lookup)
				(lambda (arguments _outer_sources _policy planning tx)
					(lookup database kind (car arguments) planning tx)))
				database kind tsql_catalog_name_query))
			(list value))
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
classification fields. Reuse only a direct bound object_id from that exact
current-database catalog; all computed and ordinary inputs retain lookup. */
(define tsql_catalog_property_from_row (lambda (database object property sources)
	(match (tsql_catalog_unwrap object)
		((symbol get_column) owner _owner_ic "object_id" _column_ic) (begin
			(define source (find sources (lambda (src) (equal? (source_alias src) owner)) nil))
			(define relation (if (nil? source) nil (source_relation source)))
			(define contract (if (union_block? relation)
				(qassoc_get (union_facts relation) 'tsql_object_catalog nil) nil))
			(if (equal? contract database)
				(list 'sql_type_expr (list 'if
					(list 'and (list 'not (list 'nil? object))
						(list tsql_metadata_visible? (tsql_actor_formula) database false))
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
(register_queryplan_callables (list tsql_actor_value tsql_object_id_value
	tsql_native_object_id tsql_metadata_visible? tsql_object_registry_id
	tsql_catalog_relation_id tsql_catalog_foreign_id tsql_catalog_module_id
	tsql_catalog_actor_field tsql_object_property_value tsql_permissions_statements
	tsql_permissions_object tsql_permissions_column))
