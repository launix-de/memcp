/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Preserve declarations independently of the storage encoding. Catalog readers
use immutable schema snapshots; alias identities are rechecked at publication. */

(define tsql_column_spec (lambda (column) (begin
	(define metadata (column "Metadata"))
	(define frontend (if metadata (metadata "tsql") nil))
	(if frontend (frontend "declaration") nil))))
(define tsql_column_view (lambda (column) (begin
	(define metadata (column "Metadata")) (define frontend (if metadata (metadata "tsql") nil))
	(define spec (tsql_column_spec column))
	/* A frontend declaration replaces neutral encoding metadata in this view.
	Concatenating pairs would leave an earlier Dimensions entry in effect. */
	(if (nil? spec) column (reduce_assoc (list
		"SQLDeclaredType" (car spec) "Dimensions" (cdr spec)
		"SQLAliasID" (coalesceNil (frontend "alias_id") 0)
		"RowVersion" (has? '("ROWVERSION" "TIMESTAMP") (car spec))
		"DefaultExpression" (coalesceNil (frontend "default_sql") ""))
		(lambda (view key value) (set_assoc view key value)) column)))))
(define tsql_get_schema (lambda (schema name) (map (get_schema schema name) tsql_column_view)))
(define tsql_frontend_catalog (lambda (snapshot) (begin
	(define value (get_assoc (coalesceNil snapshot '()) "value"))
	(define frontend (get_assoc (coalesceNil value '()) "tsql"))
	(coalesceNil frontend (list "next_id" 256 "aliases" '() "next_object_id" 1)))))
(define tsql_catalog_payload (lambda (snapshot catalog)
	(set_assoc (coalesceNil (snapshot "value") '()) "tsql" catalog)))
/* Constraint names and durable catalog identities are frontend declarations.
Every mutation binds against one immutable snapshot and publishes the updated
opaque registry with the actual enforcement/default change. */
(define tsql_relation_object_id (lambda (relation) (begin
	/* Other frontends do not declare a t-sql object identity. Read optional
	opaque metadata without invoking missing assoc values as procedures. */
	(define metadata (coalesceNil (get_assoc relation "Metadata") '()))
	(define frontend (coalesceNil (get_assoc metadata "tsql") '()))
	(get_assoc frontend "object_id"))))
(define tsql_live_constraints (lambda (snapshot)
	(filter (coalesceNil ((tsql_frontend_catalog snapshot) "constraints") '()) (lambda (record) (begin
		(define relation (find (snapshot "tables") (lambda (candidate)
			(and (equal? (tsql_relation_object_id candidate) (record "table_id")) (equal? (candidate "Name") (record "table")))) nil))
		(if relation (match (record "kind")
			"default" (find (relation "Columns") (lambda (column)
				(and (column "DefaultPresent") (equal? (column "Field") (record "column")) (equal? (((column "Metadata") "tsql") "default_id") (record "id")))) nil)
			"unique" (find (relation "Unique") (lambda (key) (equal? (key "Id") (record "handle"))) nil)
			"foreign" (find (relation "ForeignKeys") (lambda (key) (and (equal? (key "Role") "referencing") (equal? (key "Id") (record "handle")))) nil)
			_ false) false))))))
(define tsql_constraint_name_taken? (lambda (snapshot name)
	(or (find (tsql_live_constraints snapshot) (lambda (record) (equal?? (record "name") name)) nil)
		(reduce (snapshot "tables") (lambda (taken relation)
			(or taken (find (relation "ForeignKeys") (lambda (key) (and (equal? (key "Role") "referencing") (equal?? (key "Id") name))) nil)
				(find (relation "Unique") (lambda (key) (and (not (equal? (key "Id") "PRIMARY")) (equal?? (key "Id") name))) nil))) false))))
(define tsql_generated_constraint_name (lambda (candidate)
	(if (<= (utf16_len candidate) 128) candidate
		(concat (utf16_prefix candidate 100) "_" (substr (stable_structural_hash candidate true) 0 16)))))
(define tsql_allocate_constraints (lambda (snapshot catalog records) (begin
	(reduce records (lambda (seen record) (begin
		(if (and (> (utf16_len (record "name")) 0) (<= (utf16_len (record "name")) 128)) true (error "constraint identifier exceeds declaration limit"))
		(if (or (has? seen (toLower (record "name"))) (tsql_constraint_name_taken? snapshot (record "name")))
			(error "constraint name already exists") (append seen (toLower (record "name")))))) '())
	(define start (coalesceNil (catalog "next_object_id") 1))
	(if (>= (+ start (count records)) 2147483647) (error "constraint catalog identity exhausted") true)
	(define allocated (mapIndex records (lambda (index record) (set_assoc record "id" (+ start index)))))
	(list (set_assoc (set_assoc catalog "next_object_id" (+ start (count records))) "constraints"
		(merge (tsql_live_constraints snapshot) allocated)) allocated))))
(define tsql_constraint_column (lambda (relation name) (begin
	(define column (find (relation "Columns") (lambda (column) (equal?? (column "Field") name)) nil))
	(if column column (error "unknown constraint column")))))
(define tsql_create_default_constraint (lambda (schema table_name name column_name ast expression_sql) (begin
	(define snapshot (schema_metadata schema))
	(define relation (find (snapshot "tables") (lambda (candidate) (equal?? (candidate "Name") table_name)) nil))
	(if relation true (error "unknown DEFAULT constraint table"))
	(define column (tsql_constraint_column relation column_name))
	(define metadata (coalesceNil (column "Metadata") '())) (define frontend (metadata "tsql"))
	(if (column "DefaultPresent") (error "column already has a DEFAULT") true)
	(define spec (tsql_fk_column_spec column))
	(if (has? '("ROWVERSION" "TIMESTAMP") (car spec)) (error "ROWVERSION does not accept DEFAULT") true)
	(define public_name (coalesceNil name (tsql_generated_constraint_name (concat "DF_" (relation "Name") "_" (column "Field")))))
	(define allocation (tsql_allocate_constraints snapshot (tsql_frontend_catalog snapshot)
		(list (list "name" public_name "kind" "default" "table" (relation "Name") "table_id" (tsql_relation_object_id relation) "column" (column "Field") "definition" (coalesceNil expression_sql "") "system_named" (nil? name)))))
	(define record (car (cadr allocation)))
	(define next_metadata (set_assoc metadata "tsql" (set_assoc (set_assoc (set_assoc (set_assoc frontend "default_ast" ast) "default_name" public_name) "default_sql" (coalesceNil expression_sql "")) "default_id" (record "id"))))
	(altercolumn (table schema (relation "Name")) (column "Field") "options"
		(list "default_calculator" (tsql_default_calculator ast spec) "metadata" next_metadata)
		(list "revision" (snapshot "revision") "value" (tsql_catalog_payload snapshot (car allocation)))))))
(define tsql_create_unique_constraint (lambda (schema table_name name column_names tx) (begin
	(define snapshot (schema_metadata schema))
	(define relation (find (snapshot "tables") (lambda (candidate) (equal?? (candidate "Name") table_name)) nil))
	(if relation true (error "unknown UNIQUE constraint table"))
	(define columns (map column_names (lambda (name) ((tsql_constraint_column relation name) "Field"))))
	(reduce columns (lambda (seen column) (if (has? seen column) (error "duplicate UNIQUE column") (append seen column))) '())
	(define public_name (coalesceNil name (tsql_generated_constraint_name (concat "UQ_" (relation "Name") "_" (stable_structural_hash columns true)))))
	(define allocation (tsql_allocate_constraints snapshot (tsql_frontend_catalog snapshot)
		(list (list "name" public_name "kind" "unique" "table" (relation "Name") "table_id" (tsql_relation_object_id relation) "handle" public_name "columns" columns))))
	(if (createkey (table schema (relation "Name")) public_name true columns tx
		(list "revision" (snapshot "revision") "value" (tsql_catalog_payload snapshot (car allocation))) true) true
		(error "UNIQUE constraint already exists")))))

(define sql_type_aliases (lambda (schema) (coalesceNil ((tsql_frontend_catalog (schema_metadata schema)) "aliases") '())))
(define sql_type_alias (lambda (schema name) (find (sql_type_aliases schema) (lambda (alias) (equal?? (alias "Name") name)) nil)))
(define sql_resolve_table_name (lambda (schema name) (begin
	(define candidates (filter (show schema) (lambda (candidate) (equal?? candidate name))))
	(match candidates '() nil '(actual) actual _ (error "ambiguous case-insensitive table name")))))
(define sql_object_id (lambda (schema name) (begin
	(define actual (sql_resolve_table_name schema name))
	(define metadata (if actual (show (table schema actual) true) nil))
	(if (nil? metadata) nil (begin
		(define frontend ((coalesceNil (metadata "Metadata") '()) "tsql"))
		(if frontend (frontend "object_id") nil))))))
(define create_sql_type_alias (lambda (schema name base dimensions nullable) (begin
	(define snapshot (schema_metadata schema)) (define catalog (tsql_frontend_catalog snapshot))
	(if (or (find (catalog "aliases") (lambda (alias) (equal?? (alias "Name") name)) nil)
		(get_assoc tsql_native_type_ids (toLower name))) (error "type name already exists") true)
	(define id (coalesceNil (catalog "next_id") 256))
	(if (>= id 2147483647) (error "type catalog identity exhausted") true)
	(define next (set_assoc (set_assoc catalog "next_id" (+ id 1)) "aliases"
		(append (coalesceNil (catalog "aliases") '()) (list "Name" name "ID" id "BaseType" base "Dimensions" dimensions "Null" nullable))))
	(publish_schema_metadata schema (snapshot "revision") (tsql_catalog_payload snapshot next)))))
(define drop_sql_type_alias (lambda (schema name if_exists) (begin
	(define snapshot (schema_metadata schema)) (define catalog (tsql_frontend_catalog snapshot))
	(define alias (find (catalog "aliases") (lambda (candidate) (equal?? (candidate "Name") name)) nil))
	(if (nil? alias) (if if_exists false (error "type does not exist")) (begin
		(if (reduce (snapshot "tables") (lambda (used relation)
			(or used (find (relation "Columns") (lambda (column)
				(equal? ((tsql_column_view column) "SQLAliasID") (alias "ID"))) nil))) false)
			(error "type is used by a column") true)
		(publish_schema_metadata schema (snapshot "revision") (tsql_catalog_payload snapshot
			(set_assoc catalog "aliases" (filter (catalog "aliases") (lambda (candidate) (not (equal? (candidate "ID") (alias "ID")))))))))))))

/* Generated columns use the ordinary persisted trigger mechanism. The literal
default satisfies omitted-column validation and partition routing; allocation
belongs exclusively to the synchronous write hooks. */
(define tsql_rowversion_placeholder (base64_decode "AAAAAAAAAAA="))
(define tsql_rowversion_trigger_name (lambda (column timing)
	(concat "__tsql_rowversion_" timing ":" column)))
(define tsql_rowversion_trigger_definition (lambda (column timing)
	(list "trigger" (tsql_rowversion_trigger_name column timing) timing column "tsql-generated" nil false 2147483647 column)))
(define tsql_rowversion_triggers (lambda (column)
	(map '("before_insert" "before_update") (lambda (timing) (tsql_rowversion_trigger_definition column timing)))))
(define tsql_missing_rowversion_triggers (lambda (column context)
	(filter (tsql_rowversion_triggers column) (lambda (definition) (begin
		(define existing (find (context "trigger_definitions") (lambda (trigger)
			(equal? (trigger "Name") (cadr definition))) nil))
		(if existing (begin
			(if (and (equal? (existing "Timing") (nth definition 2))
				(equal? (existing "Source") column) (equal? (existing "Language") "tsql-generated")
				(existing "Hidden") (equal? (existing "Priority") 2147483647)
				(equal? (existing "OwnerColumn") column)) true
				(error "generated column trigger conflicts with persisted declaration"))
			false)
			(if (has? (context "triggers") (cadr definition))
				(error "missing generated column trigger declaration") true)))))))
(define tsql_compile_generated_trigger (lambda (column context) (begin
	(if (and (string? column) (> (strlen column) 0)) true (error "invalid generated column source"))
	(define value (list (quote next_sequence_token) (context "schema")))
	(define update (list (quote set_assoc) (quote new) column value))
	(define body (match (context "timing")
		"before_insert" (list (quote if) (list (quote has_assoc?) (quote new) column)
			(list (quote error) "ROWVERSION columns cannot be explicitly inserted") update)
		"before_update" update
		_ (error "unsupported generated column trigger timing")))
	(list (quote deferred_trigger) (list (quote lambda)
		(list (quote old) (quote new) (quote session) (quote tx)) body)))))
(registertriggerlanguage "tsql-generated" tsql_compile_generated_trigger)
(registerschemainitializer "tsql-rowversion" (lambda (context)
	(merge (map (filter (map (context "columns") tsql_column_view) (lambda (column) (column "RowVersion"))) (lambda (column) (begin
		(define name (column "Field"))
		(merge (if (equal? (column "Default") tsql_rowversion_placeholder) '()
			(list (list "default" name tsql_rowversion_placeholder)))
			(tsql_missing_rowversion_triggers name context))))))))
(define tsql_insertable_columns (lambda (schema name)
	(map (filter (tsql_get_schema schema name) (lambda (column)
		(and (not (internal_column_name? (column "Field"))) (not (column "RowVersion")))))
		(lambda (column) (column "Field")))))
/* Recheck the actual target at execution, including cached DML after DDL.
Ordinary tables and read-only statements never touch the sequence allocator. */
(define tsql_prepare_rowversion_write (lambda (schema name columns) (begin
	(define generated (find (tsql_get_schema schema name) (lambda (column) (column "RowVersion")) nil))
	(if generated (begin
		(if (find columns (lambda (column) (equal?? column (generated "Field"))) nil)
			(error "ROWVERSION columns cannot be explicitly assigned") true)
		(prepare_sequence schema)) true))))
/* SQL dimensions are validated here; storage sees only neutral scalar
encodings. The declaration remains in opaque frontend metadata. */
(define tsql_declaration_dimensions (lambda (spec) (begin
	(define type (car spec)) (define size (tsql_spec_dimension spec 1)) (define scale (tsql_spec_dimension spec 2))
	(if (has? '("ROWVERSION" "TIMESTAMP") type)
		(if (and (nil? size) (nil? scale)) '(8) (error "ROWVERSION does not accept dimensions"))
		(if (has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR" "BINARY" "VARBINARY") type)
			(begin (define length (coalesceNil size 1))
				(if (and (nil? scale) (or (and (> length 0) (<= length (if (has? '("NVARCHAR" "NCHAR") type) 4000 8000)))
					(and (equal? length -1) (has? '("VARCHAR" "NVARCHAR" "VARBINARY") type)))) true (error "invalid character or binary dimensions"))
				(list length))
			(if (tsql_decimal_type? type)
				(begin (define p (tsql_spec_precision spec)) (define s (tsql_spec_scale spec))
					(if (and (> p 0) (<= p 38) (>= s 0) (<= s p)) true (error "invalid decimal precision or scale")) (list p s))
				(if (has? '("DATETIME2" "DATETIMEOFFSET" "TIME") type)
					(begin (define precision (coalesceNil size 7))
						(if (and (nil? scale) (>= precision 0) (<= precision 7)) true (error "invalid temporal precision")) (list precision))
					(if (equal? type "FLOAT")
						(begin (define bits (coalesceNil size 53)) (if (and (nil? scale) (> bits 0) (<= bits 53)) (list bits) (error "invalid floating point precision")))
						(if (and (nil? size) (nil? scale) (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "BIT" "FLOAT" "REAL" "MONEY" "SMALLMONEY" "DATE" "DATETIME" "SMALLDATETIME" "TEXT" "NTEXT") type)) '()
							(error (concat "unsupported type declaration " type)))))))))))
(define tsql_storage_type (lambda (spec)
	(match (car spec)
		"INT" "BIGINT" "INTEGER" "BIGINT" "SMALLINT" "BIGINT" "TINYINT" "BIGINT" "BIT" "BOOLEAN"
		"MONEY" "BIGINT" "SMALLMONEY" "BIGINT" "DATE" "BIGINT" "DATETIME" "BIGINT"
		"SMALLDATETIME" "BIGINT" "DATETIME2" "BIGINT" "TIME" "BIGINT" "FLOAT" "FLOAT" "REAL" "FLOAT"
		_ "ANY")))
(define tsql_native_type (lambda (spec) (list (tsql_storage_type spec) (tsql_declaration_dimensions spec))))
(define tsql_identity_limit (lambda (spec) (begin
	(define carrier_limit (coefficient_encode "9223372036854775807"))
	(define declared_limit (coefficient_encode (if (tsql_decimal_type? (car spec)) (string_repeat "9" (tsql_spec_precision spec)) (cadr (tsql_integer_bounds (car spec))))))
	(coefficient_to_integer (if (< (coefficient_compare declared_limit carrier_limit) 0) declared_limit carrier_limit) 0 true))))
/* Persisted recipes carry declarations as quoted code constants. They retain
no lexical declaration list which procedure closing could reinterpret as code. */
(define tsql_column_sanitizer (lambda (spec) (begin
	(define declaration (list 'quote spec))
	(define body (if (tsql_decimal_type? (car spec)) (list 'tsql_decimal_validate 'value declaration)
		(if (or (tsql_money_type? (car spec)) (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") (car spec)))
			(list 'tsql_integer_validate 'value declaration)
			(if (tsql_temporal_type? (car spec)) (list 'tsql_temporal_validate 'value declaration)
				(list 'tsql_cast_bound 'value declaration declaration)))))
	(eval (list 'lambda (list 'value) body)))))
(define tsql_default_calculator (lambda (ast spec) (begin
	(define info (sql_expr_info '() ast true))
	(eval (list 'lambda (list 'calculator_context) (list 'tsql_cast_bound
		(tsql_clock_binding (sql_info_formula info) 'calculator_context) (list 'quote (sql_info_spec info)) (list 'quote spec)))))))

/* An ADD backfill can use one prepared image only when its declaration is
statement-stable. Unknown/volatile recipes retain their future INSERT callback
and require actual per-row backfill support from the neutral engine. */
(define tsql_stable_default? (lambda (expression) (match expression
	((symbol quote) _value) true
	(cons head arguments) (and (symbol? head)
		(has? '(tsql_cast_value tsql_convert_value tsql_clock_value tsql_number_literal
			tsql_add tsql_subtract tsql_multiply tsql_divide tsql_remainder tsql_negate
			tsql_nchar tsql_binary_literal tsql_decimal_math tsql_isnull tsql_dateadd
			tsql_datediff tsql_datepart if coalesceNil equal? equal?? < > <= >= and or not) head)
		(reduce arguments (lambda (stable item) (and stable (tsql_stable_default? item))) true))
	(symbol _name) false
	_ true)))

(define tsql_view_name (lambda (schema name) (begin
	(define matches (filter (tsql_odbc_view_names schema) (lambda (candidate) (equal?? candidate name))))
	(match matches '() nil '(actual) actual _ (error "ambiguous case-insensitive view name")))))
(define tsql_resolve_table (lambda (schema name)
	(if (tsql_catalog_database schema) name
		(coalesceNil (sql_resolve_table_name schema name) (tsql_view_name schema name) name))))
(define tsql_resolve_column (lambda (schema table name)
	(coalesceNil (resolve_column_name schema table name true) name)))
(define tsql_add_column (lambda (schema table_name name type dimensions attributes) (begin
	(define snapshot (schema_metadata schema))
	(define relation (find (snapshot "tables") (lambda (candidate) (equal?? (candidate "Name") table_name)) nil))
	(if (nil? relation) (error "unknown table in ADD column") true)
	(if (find (relation "Columns") (lambda (column) (equal?? (column "Field") name)) nil) (error "column already exists") true)
	(define frontend ((get_assoc attributes "metadata") "tsql"))
	(if (has? '("ROWVERSION" "TIMESTAMP") (car (frontend "declaration")))
		(error "ROWVERSION must be declared when creating the table") true)
	(define alias_id (frontend "alias_id"))
	(if (or (equal? alias_id 0) (find ((tsql_frontend_catalog snapshot) "aliases") (lambda (alias) (equal? (alias "ID") alias_id)) nil)) true
		(error "type alias changed before column publication"))
	(define constraints (tsql_bind_create_constraints snapshot (tsql_frontend_catalog snapshot) (relation "Name") (tsql_relation_object_id relation)
		(list (list "column" name type dimensions attributes))))
	(define bound_attributes (nth (car (cadr constraints)) 4))
	(if (sql_create_column (table schema (relation "Name")) name type dimensions
		(merge bound_attributes
			(if (and (has_assoc? attributes "default_calculator") (tsql_stable_default? (frontend "default_ast")))
				(list "initial_value" ((get_assoc attributes "default_calculator") (tsql_statement_context))) '())
			(list "schema_publication" (list "revision" (snapshot "revision") "value" (tsql_catalog_payload snapshot (car constraints)))))) true
		(error "t-sql ADD column already exists")))))

/* Recheck names inside execution, so cached CREATE commands and concurrent
t-sql declarations cannot replace another spelling of the same logical view. */
(define tsql_view_ddl_mutex (mutex))

(define tsql_fk_column_spec (lambda (column)
	(coalesceNil (tsql_column_spec column) (cons (toUpper (column "RawType")) (coalesceNil (column "Dimensions") '())))))
(define tsql_fk_type_signature (lambda (spec)
	(cons (match (car spec) "INTEGER" "INT" "NUMERIC" "DECIMAL" _ (car spec)) (cdr spec))))
(define tsql_validate_foreign_keys (lambda (snapshot child_name child_columns definitions) (begin
	(define foreign (filter definitions (lambda (definition) (equal? (car definition) "foreign"))))
	(reduce foreign (lambda (names definition) (begin
		(define name (cadr definition))
		(if (or (has? names (toLower name)) (tsql_constraint_name_taken? snapshot name) (reduce (snapshot "tables") (lambda (found relation)
			(or found (find (relation "ForeignKeys") (lambda (key) (and (equal? (key "Role") "referencing") (equal?? (key "Id") name))) nil))) false))
			(error "foreign key constraint name already exists") true)
		(define columns (nth definition 2)) (define parent_name (nth definition 3)) (define parent_columns (nth definition 4))
		(define parent (find (snapshot "tables") (lambda (relation) (equal?? (relation "Name") parent_name)) nil))
		(if (nil? parent) (error "referenced table does not exist") true)
		(if (equal?? child_name parent_name) (error "self-referencing foreign keys are unsupported") true)
		(if (equal? (count columns) (count parent_columns)) true (error "foreign key column counts do not match"))
		(mapIndex columns (lambda (i column_name) (begin
			(define child (find child_columns (lambda (column) (equal?? (column "Field") column_name)) nil))
			(define other (find (parent "Columns") (lambda (column) (equal?? (column "Field") (nth parent_columns i))) nil))
			(if (and child other) true (error "unknown foreign key column"))
			(if (equal? (tsql_fk_type_signature (tsql_fk_column_spec child)) (tsql_fk_type_signature (tsql_fk_column_spec other))) true
				(error "incompatible declared foreign key column types")))))
		(append names (toLower name)))) '()))))
(define tsql_bind_create_constraints (lambda (snapshot catalog table_name table_id definitions) (begin
	(define records (merge (map definitions (lambda (definition) (match definition
		'("column" name _type _dimensions attributes) (begin
			(define frontend ((get_assoc attributes "metadata") "tsql"))
			(merge (if (has_assoc? frontend "default_ast")
				(list (list "name" (coalesceNil (frontend "default_name") (tsql_generated_constraint_name (concat "DF_" table_name "_" name))) "kind" "default" "table" table_name "table_id" table_id "column" name "definition" (coalesceNil (frontend "default_definition") (frontend "default_sql") "") "system_named" (nil? (frontend "default_name")))) '())
				(if (get_assoc attributes "unique")
					(list (list "name" (tsql_generated_constraint_name (concat "UQ_" table_name "_" name)) "kind" "unique" "table" table_name "table_id" table_id "handle" name "columns" (list name) "system_named" true)) '())))
		'("unique" name columns _nulls_equal) (if (equal? name "PRIMARY") '()
			(list (list "name" name "kind" "unique" "table" table_name "table_id" table_id "handle" name "columns" columns)))
		'("foreign" name _columns _parent _other _update _delete)
		(list (list "name" name "kind" "foreign" "table" table_name "table_id" table_id "handle" name))
		_ '())))))
	(define allocation (tsql_allocate_constraints snapshot catalog records))
	(define bound (map definitions (lambda (definition) (match definition
		'("column" name type dimensions attributes) (begin
			(define record (find (cadr allocation) (lambda (candidate) (and (equal? (candidate "kind") "default") (equal? (candidate "column") name))) nil))
			(if record (begin
				(define metadata (get_assoc attributes "metadata")) (define frontend (metadata "tsql"))
				(list "column" name type dimensions (set_assoc attributes "metadata" (set_assoc metadata "tsql"
					(set_assoc (set_assoc frontend "default_id" (record "id")) "default_name" (record "name")))))) definition))
		_ definition))))
	(list (car allocation) bound))))

(define tsql_create_table (lambda (schema name definitions options if_not_exists tx) (begin
	(define generated (filter (filter definitions (lambda (definition) (equal? (car definition) "column"))) (lambda (definition)
		(equal? (car ((((nth definition 4) "metadata") "tsql") "declaration")) "ROWVERSION"))))
	(if (> (count generated) 1) (error "only one ROWVERSION column is allowed") true)
	(if (empty_list? generated) true (prepare_sequence schema))
	(define snapshot (schema_metadata schema)) (define catalog (tsql_frontend_catalog snapshot))
	(define duplicate (find (snapshot "tables") (lambda (relation) (equal?? (relation "Name") name)) nil))
	(if duplicate (error "table name already exists") true)
	(if (tsql_view_name schema name) (error "table name already belongs to a view") true)
	(define columns (filter definitions (lambda (definition) (equal? (car definition) "column"))))
	(map columns (lambda (definition) (begin
		(define frontend (((nth definition 4) "metadata") "tsql")) (define alias_id (frontend "alias_id"))
		(if (or (equal? alias_id 0) (find (catalog "aliases") (lambda (alias) (equal? (alias "ID") alias_id)) nil)) true
			(error "type alias changed before table publication")))))
	(tsql_validate_foreign_keys snapshot name (map columns (lambda (definition)
		(list "Field" (cadr definition) "Metadata" ((nth definition 4) "metadata") "RawType" (nth definition 2) "Dimensions" (nth definition 3)))) definitions)

	(define id (coalesceNil (catalog "next_object_id") 1))
	(if (>= id 2147483647) (error "object catalog identity exhausted") true)
	(define constraints (tsql_bind_create_constraints snapshot (set_assoc catalog "next_object_id" (+ id 1)) name id definitions))
	(define payload (tsql_catalog_payload snapshot (car constraints)))
	(sql_create_table schema name (merge (cadr constraints) (merge (map generated (lambda (definition) (tsql_rowversion_triggers (cadr definition))))))
		(merge options (list "strict_foreign_keys" true "metadata" (list "tsql" (list "object_id" id))
			"schema_publication" (list "revision" (snapshot "revision") "value" payload))) if_not_exists tx))))
(define tsql_create_view (lambda (tx schema name sql ir)
	(tsql_view_ddl_mutex tx (lambda () (begin
		(if (or (sql_resolve_table_name schema name) (tsql_view_name schema name)) (error "view name already exists") true)
		(create_sql_view tx schema name "tsql" sql ir "error"))))))
(define tsql_drop_view (lambda (tx schema name if_exists)
	(tsql_view_ddl_mutex tx (lambda ()
		(drop_sql_view tx schema (coalesceNil (tsql_view_name schema name) name) if_exists)))))

(define tsql_column_definition (lambda (schema definition) (match definition
	'("column" name spec attributes) (begin
		(define alias (sql_type_alias schema (car spec)))
		(define declared (if alias (cons (alias "BaseType") (alias "Dimensions")) spec))
		(if (and alias (or (not (nil? (tsql_spec_dimension spec 1))) (not (nil? (tsql_spec_dimension spec 2))))) (error "alias types do not accept dimensions") true)
		(define dimensions (tsql_declaration_dimensions declared))
		(define type (if (equal? (car declared) "TIMESTAMP") "ROWVERSION" (car declared)))
		(define declaration (cons type dimensions))
		(define identity (get_assoc attributes "auto_increment")) (define rowversion (equal? type "ROWVERSION"))
		(define default_ast (get_assoc attributes "tsql_default_ast"))
		(if (and rowversion (or identity default_ast)) (error "ROWVERSION does not accept IDENTITY or DEFAULT") true)
		(if identity (begin
			(if (or (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type)
				(and (tsql_decimal_type? type) (equal? (tsql_spec_scale declaration) 0))) true (error "IDENTITY requires an integer declaration"))
			(if (and (has_assoc? attributes "null") (get_assoc attributes "null")) (error "IDENTITY cannot be NULL") true)) true)
		(define frontend (merge (merge (list "declaration" declaration "alias_id" (if alias (alias "ID") 0))
			(if (get_assoc attributes "tsql_default_name") (list "default_name" (get_assoc attributes "tsql_default_name")) '()))
			(if default_ast (list "default_ast" (cadr default_ast) "default_definition" (coalesceNil (get_assoc attributes "tsql_default_sql") "")
				"default_sql" (match (cadr default_ast)
					((symbol tsql_cast_value) ((symbol tsql_clock_value) kind) _) (if (equal? kind "DATETIME2") "SYSDATETIME()" "GETDATE()")
					_ (coalesceNil (get_assoc attributes "tsql_default_sql") ""))) '())))
		(define attrs (merge (merge (extract_assoc attributes (lambda (key value)
			(if (has? '("tsql_default_ast" "tsql_default_name" "tsql_default_sql") key) '() (list key value)))))
			(if identity '("null" false) '())
			(if (or (get_assoc attributes "unique") (get_assoc attributes "primary")) '("key_nulls_equal" true) '())
			(if (and alias (not (has_assoc? attributes "null"))) (list "null" (alias "Null")) '())
			(list "metadata" (list 'quote (list "tsql" frontend)) "value_sanitizer" (list 'tsql_column_sanitizer (list 'quote declaration)))
			(if default_ast (list "default_calculator" (list 'tsql_default_calculator default_ast (list 'quote declaration))) '())
			(if rowversion (list "default" tsql_rowversion_placeholder) '())
			(if (or rowversion (tsql_decimal_type? type) (equal? type "DATETIMEOFFSET") (sql_text_type? type)) '("collate" "bin") '())
			(if (equal? type "DATETIMEOFFSET")
				(list "key_projection" (list 'lambda (list 'value) (list 'if (list 'nil? 'value) nil (list 'integer_from_order_key (list 'substr 'value 0 16))))) '())
			(if identity (list "allocator_max" (tsql_identity_limit declaration)
				"allocator_value" (if (tsql_decimal_type? type)
					(list 'lambda (list 'value) (list 'coefficient_to_integer 'value 0 true))
					(list 'lambda (list 'value) 'value))
				"allocator_encode" (if (tsql_decimal_type? type) (list 'lambda (list 'value) (list 'integer_to_coefficient 'value)) (list 'lambda (list 'value) 'value))) '())))
		'('list "column" name (tsql_storage_type declaration) '('list) (cons list attrs)))
	_ definition)))

/* Foreign key declarations retain actual storage enforcement. Existing ordered
candidate keys are validated before the private child table is published. */
(define tsql_foreign_key_definition (lambda (schema table_name definition resolve) (match definition
	'("foreign" name child_columns target parent_columns update_mode delete_mode) (match target '(parent_db parent_name) (begin
		(if (equal?? schema parent_db) true (error "cross-database foreign keys are unsupported"))
		(define parent (tsql_resolve_table schema parent_name))
		(if (equal?? table_name parent) (error "self-referencing foreign keys are unsupported") true)
		(define constraint_name (coalesceNil name (concat "FK_" table_name "_" (stable_structural_hash definition true))))
		(list (quote list) "foreign" constraint_name (cons list (map child_columns resolve)) parent
			(cons list (map parent_columns (lambda (column) (tsql_resolve_column schema parent column)))) update_mode delete_mode))))))
(define tsql_create_foreign_key (lambda (schema table_name definition tx) (begin
	(define snapshot (schema_metadata schema))
	(define child (find (snapshot "tables") (lambda (relation) (equal?? (relation "Name") table_name)) nil))
	(if child true (error "unknown foreign key child table"))
	(define native (eval (tsql_foreign_key_definition schema (child "Name") definition (lambda (column) (begin
		(define resolved_column (find (child "Columns") (lambda (candidate) (equal?? (candidate "Field") column)) nil))
		(if resolved_column (resolved_column "Field") (error "unknown foreign key child column")))))))
	(tsql_validate_foreign_keys snapshot (child "Name") (child "Columns") (list native))
	(match native '("foreign" name columns parent parent_columns update_mode delete_mode)
		(begin
			(define allocation (tsql_allocate_constraints snapshot (tsql_frontend_catalog snapshot)
				(list (list "name" name "kind" "foreign" "table" (child "Name") "table_id" (tsql_relation_object_id child) "handle" name))))
			(if (createforeignkey (table schema (child "Name")) name columns (table schema parent) parent_columns update_mode delete_mode tx true
				(list "revision" (snapshot "revision") "value" (tsql_catalog_payload snapshot (car allocation)))) true (error "foreign key constraint already exists")))))))
(define tsql_drop_constraint (lambda (schema table_name name tx) (begin
	(define snapshot (schema_metadata schema))
	(define child (find (snapshot "tables") (lambda (relation) (equal?? (relation "Name") table_name)) nil))
	(if child true (error "unknown constraint table"))
	(define record (find (tsql_live_constraints snapshot) (lambda (constraint)
		(and (equal? (constraint "table_id") (tsql_relation_object_id child)) (equal?? (constraint "name") name))) nil))
	(define catalog (set_assoc (tsql_frontend_catalog snapshot) "constraints"
		(filter (tsql_live_constraints snapshot) (lambda (constraint) (or (nil? record) (not (equal? (constraint "id") (record "id"))))))))
	(define publication (list "revision" (snapshot "revision") "value" (tsql_catalog_payload snapshot catalog)))
	(if record (match (record "kind")
		"default" (begin
			(define column (tsql_constraint_column child (record "column"))) (define metadata (column "Metadata"))
			(define frontend (merge (extract_assoc (metadata "tsql") (lambda (key value)
				(if (has? '("default_ast" "default_name" "default_id" "default_sql" "default_definition") key) '() (list key value))))))
			(altercolumn (table schema (child "Name")) (column "Field") "options"
				(list "drop_default" true "metadata" (set_assoc metadata "tsql" frontend)) publication))
		"unique" (dropkey (table schema (child "Name")) (record "handle") tx publication)
		"foreign" (dropforeignkey (table schema (child "Name")) (record "handle") tx true publication)
		_ (error "unsupported constraint kind"))
		(begin
			(define foreign (find (child "ForeignKeys") (lambda (constraint) (and (equal? (constraint "Role") "referencing") (equal?? (constraint "Id") name))) nil))
			(if foreign true (error "unknown or unsupported constraint for DROP CONSTRAINT"))
			(if (dropforeignkey (table schema (child "Name")) (foreign "Id") tx true publication) true
				(error "unknown constraint for DROP CONSTRAINT")))))))

/* A primary key makes each participating column non-nullable even when the
source script omits NOT NULL. Resolve constraint spelling against declarations. */
(define tsql_table_definitions (lambda (schema table_name definitions) (begin
	(define columns (merge (map definitions (lambda (definition) (match definition
		'("column" name _spec _attributes) (list name) _ '())))))
	(reduce columns (lambda (seen name) (if (has? seen (toLower name)) (error "duplicate t-sql column identifier") (append seen (toLower name)))) '())
	(define resolve (lambda (name) (begin (define resolved (find columns (lambda (column) (equal?? column name)) nil))
		(if (nil? resolved) (error (concat "unknown constraint column " name)) resolved))))
	(define primary (merge (map definitions (lambda (definition) (match definition
		((symbol list) "unique" "PRIMARY" (cons _ names)) (map names resolve)
		'("column" name _spec attributes) (if (get_assoc attributes "primary") (list name) '())
		_ '())))))
	(map definitions (lambda (definition) (match definition
		'("column" name spec attributes) (begin
			(define is_primary (has? primary name))
			(if (and is_primary (has_assoc? attributes "null") (get_assoc attributes "null")) (error "primary key columns cannot be declared NULL") true)
			(tsql_column_definition schema (list "column" name spec (if is_primary (set_assoc attributes "null" false) attributes))))
		((symbol list) "unique" name (cons head columns)) (list (quote list) "unique"
			(coalesceNil name (tsql_generated_constraint_name (concat "UQ_" table_name "_" (stable_structural_hash columns true))))
			(cons head (map columns resolve)) true)
		'("foreign" name child_columns target parent_columns update_mode delete_mode)
		(tsql_foreign_key_definition schema table_name definition resolve)
		_ definition))))))

/* Function arguments can contain escaped identifiers. Resolve only this
connection's database so metadata calls cannot bypass its database policy. */
(define tsql_catalog_object (lambda (schema text) (if (nil? text) nil (begin
	(define identifier tsql_identifier)
	(define object (parser (or
		(parser '((define db identifier) "." (define owner identifier) "." (define name identifier))
			(if (and (equal?? db schema) (equal?? owner "dbo")) name nil))
		(parser '((define owner identifier) "." (define name identifier)) (if (equal?? owner "dbo") name nil))
		(parser (define name identifier) name))))
	(try (lambda () (object text)) (lambda (_error) nil))))))

(define tsql_native_type_ids '("bigint" 127 "binary" 173 "bit" 104 "char" 175 "date" 40 "datetime" 61
	"datetime2" 42 "datetimeoffset" 43 "decimal" 106 "numeric" 108 "float" 62 "int" 56 "integer" 56 "money" 60 "nchar" 239
	"ntext" 99 "nvarchar" 231 "real" 59 "smallint" 52 "smallmoney" 122 "smalldatetime" 58
	"text" 35 "time" 41 "timestamp" 189 "rowversion" 189 "tinyint" 48 "varbinary" 165 "varchar" 167))
(define tsql_type_id (lambda (schema name) (begin
	(define object (tsql_catalog_object schema name))
	(if (nil? object) nil (begin (define alias (sql_type_alias schema object))
		(if alias (alias "ID") (get_assoc tsql_native_type_ids (toLower object))))))))
(define tsql_col_length_metadata (lambda (metadata) (if (nil? metadata) nil (begin
	(define declared (coalesceNil (metadata "SQLDeclaredType") ""))
	(define type (toUpper (if (equal? declared "") (metadata "RawType") declared)))
	(define size (if (empty_list? (metadata "Dimensions")) nil (car (metadata "Dimensions"))))
	(if (has? '("NVARCHAR" "NCHAR") type) (if (equal? size -1) -1 (* 2 size))
		(if (has? '("VARCHAR" "CHAR" "BINARY" "VARBINARY") type) size
			(if (has? '("DECIMAL" "NUMERIC") type) (if (<= size 9) 5 (if (<= size 19) 9 (if (<= size 28) 13 17)))
				(if (has? '("TIME" "DATETIME2" "DATETIMEOFFSET") type)
					(+ (match type "DATETIME2" 3 "DATETIMEOFFSET" 5 _ 0) (if (<= size 2) 3 (if (<= size 4) 4 5)))
					(get_assoc '("BIGINT" 8 "FLOAT" 8 "DATETIME" 8 "MONEY" 8 "ROWVERSION" 8 "TIMESTAMP" 8
						"INT" 4 "INTEGER" 4 "REAL" 4 "SMALLDATETIME" 4 "SMALLMONEY" 4 "SMALLINT" 2
						"TINYINT" 1 "BIT" 1 "BOOLEAN" 1 "DATE" 3 "TEXT" 16 "NTEXT" 16) type)))))))))
(define tsql_col_length (lambda (schema name column) (begin
	(define object (tsql_catalog_object schema name))
	(tsql_col_length_metadata (if (nil? object) nil
		(find (tsql_get_schema schema (tsql_resolve_table schema object)) (lambda (col) (equal?? (col "Field") column)) nil))))))
(define tsql_server_property (lambda (name) (if (nil? name) nil (match (toLower name)
	"productversion" nil "edition" "MemCP" "servername" "MemCP"
	"isclustered" 0 "ishadrenabled" 0 "isintegratedsecurityonly" 0 _ nil))))

(define tsql_object_id (lambda (schema name kind) (begin
	(define object (tsql_catalog_object schema name))
	(if (and (not (nil? object)) (or (nil? kind) (equal?? kind "U")))
		(sql_object_id schema (tsql_resolve_table schema object)) nil))))

/* Virtual catalog sources retain only a database name and a logical catalog
name in the AST. Physical lowering reads fresh rows on every invocation. */
(define tsql_catalog_fields (lambda (names) (map names (lambda (name) (begin
	(define type (if (has? '("name" "type_desc" "collation_name" "definition") name) "NVARCHAR"
		(if (equal? name "type") "CHAR"
			(if (has? '("system_type_id" "precision" "scale" "temporal_type") name) "TINYINT"
				(if (equal? name "max_length") "SMALLINT"
					(if (has? '("is_nullable" "is_user_defined" "is_assembly_type" "is_table_type" "is_ms_shipped" "is_replicated" "is_identity" "is_computed" "is_system_named") name) "BIT" "INT"))))))
	(list "Field" name "Type" type "RawType" (match type "NVARCHAR" "VARCHAR" "BIT" "BOOLEAN" _ type)
		"SQLDeclaredType" type "Dimensions" (match type "NVARCHAR" (list (if (equal? name "definition") -1 (if (equal? name "type_desc") 60 128))) "CHAR" '(2) _ '())
		"Null" (has? '("principal_id" "collation_name") name)))))))
(define tsql_catalog_columns (lambda (schema name) (match (toLower name)
	"types" (tsql_catalog_fields '("name" "system_type_id" "user_type_id" "schema_id" "principal_id" "max_length" "precision" "scale" "collation_name" "is_nullable" "is_user_defined" "is_assembly_type" "default_object_id" "rule_object_id" "is_table_type"))
	"tables" (tsql_catalog_fields '("name" "object_id" "schema_id" "type" "type_desc" "is_ms_shipped" "is_replicated" "temporal_type"))
	"objects" (tsql_catalog_fields '("name" "object_id" "schema_id" "type" "type_desc" "is_ms_shipped" "parent_object_id"))
	"columns" (tsql_catalog_fields '("object_id" "name" "column_id" "system_type_id" "user_type_id" "max_length" "precision" "scale" "collation_name" "is_nullable" "is_identity" "is_computed" "default_object_id" "rule_object_id"))
	"default_constraints" (tsql_catalog_fields '("name" "object_id" "schema_id" "type" "type_desc" "is_ms_shipped" "parent_object_id" "parent_column_id" "definition" "is_system_named"))
	"key_constraints" (tsql_catalog_fields '("name" "object_id" "schema_id" "type" "type_desc" "is_ms_shipped" "parent_object_id" "unique_index_id" "is_system_named"))
	"schemas" (tsql_catalog_fields '("name" "schema_id" "principal_id"))
	_ (error (concat "unsupported sys catalog " name)))))
(define tsql_catalog_type_row (lambda (name id user_id dimensions nullable user_defined) (begin
	(define type (toUpper name))
	(define precision (if (has? '("DECIMAL" "NUMERIC") type) (if (empty_list? dimensions) nil (car dimensions))
		(get_assoc '("BIGINT" 19 "INT" 10 "SMALLINT" 5 "TINYINT" 3 "BIT" 1 "FLOAT" 53 "REAL" 24 "MONEY" 19 "SMALLMONEY" 10) type)))
	(list "name" name "system_type_id" id "user_type_id" user_id "schema_id" (if user_defined 1 4)
		"principal_id" nil "max_length" (tsql_col_length_value type dimensions) "precision" (coalesceNil precision 0)
		"scale" (if (has? '("DECIMAL" "NUMERIC") type) (match dimensions '(_ decimals) decimals _ nil) (if (has? '("MONEY" "SMALLMONEY") type) 4 0))
		"collation_name" nil "is_nullable" (if nullable 1 0) "is_user_defined" (if user_defined 1 0)
		"is_assembly_type" 0 "default_object_id" 0 "rule_object_id" 0 "is_table_type" 0))))
(define tsql_col_length_value (lambda (type dimensions)
	(tsql_col_length_metadata (list "RawType" type "SQLDeclaredType" type "Dimensions" dimensions))))
(define tsql_catalog_constraint_row (lambda (record snapshot) (begin
	(define default (equal? (record "kind") "default"))
	(define relation (find (snapshot "tables") (lambda (candidate) (equal? (candidate "Name") (record "table"))) nil))
	(define index_id (reduce (mapIndex (relation "Unique") (lambda (index key) (list index key)))
		(lambda (found indexed) (if (equal? ((cadr indexed) "Id") (record "handle")) (+ (car indexed) 1) found)) 0))
	(define column_id (if default (reduce (mapIndex (relation "Columns") (lambda (index column) (list index column)))
		(lambda (found indexed) (if (equal? ((cadr indexed) "Field") (record "column")) (+ (car indexed) 1) found)) 0) 0))
	(list "name" (record "name") "object_id" (record "id") "schema_id" 1
		"type" (if default "D" (if (equal? (record "kind") "foreign") "F" "UQ"))
		"type_desc" (if default "DEFAULT_CONSTRAINT" (if (equal? (record "kind") "foreign") "FOREIGN_KEY_CONSTRAINT" "UNIQUE_CONSTRAINT"))
		"is_ms_shipped" 0 "parent_object_id" (record "table_id") "parent_column_id" column_id
		"definition" (coalesceNil (record "definition") "") "unique_index_id" index_id "is_system_named" (if (record "system_named") 1 0)))))
(define tsql_catalog_rows (lambda (catalog name) (begin
	(define schema (tsql_catalog_database catalog))
	(match (toLower name)
		"schemas" (list (list "name" "dbo" "schema_id" 1 "principal_id" nil) (list "name" "sys" "schema_id" 4 "principal_id" nil))
		"types" (merge
			(extract_assoc tsql_native_type_ids (lambda (name id)
				(if (has? '("integer" "rowversion") name) '()
					(list (tsql_catalog_type_row name id id
						(cadr (tsql_native_type (list (toUpper name)
							(get_assoc '("char" 8000 "varchar" 8000 "binary" 8000 "varbinary" 8000 "nchar" 4000 "nvarchar" 4000 "decimal" 38 "numeric" 38) name) nil))) true false)))))
			(map (sql_type_aliases schema) (lambda (alias)
				(set_assoc (tsql_catalog_type_row (alias "BaseType") (get_assoc tsql_native_type_ids (toLower (alias "BaseType")))
					(alias "ID") (alias "Dimensions") (alias "Null") true) "name" (alias "Name")))))
		"tables" (map (show schema) (lambda (name) (list "name" name "object_id" (sql_object_id schema name) "schema_id" 1
			"type" "U" "type_desc" "USER_TABLE" "is_ms_shipped" 0 "is_replicated" 0 "temporal_type" 0)))
		"objects" (begin (define snapshot (schema_metadata schema))
			(merge (map (show schema) (lambda (name) (list "name" name "object_id" (sql_object_id schema name) "schema_id" 1
				"type" "U" "type_desc" "USER_TABLE" "is_ms_shipped" 0 "parent_object_id" 0)))
				(map (tsql_live_constraints snapshot) (lambda (record) (tsql_catalog_constraint_row record snapshot)))))
		"default_constraints" (begin (define snapshot (schema_metadata schema))
			(map (filter (tsql_live_constraints snapshot) (lambda (record) (equal? (record "kind") "default")))
				(lambda (record) (tsql_catalog_constraint_row record snapshot))))
		"key_constraints" (begin (define snapshot (schema_metadata schema))
			(map (filter (tsql_live_constraints snapshot) (lambda (record) (equal? (record "kind") "unique")))
				(lambda (record) (tsql_catalog_constraint_row record snapshot))))
		"columns" (merge (map (show schema) (lambda (name) (begin
			(define object_id (sql_object_id schema name))
			(mapIndex (filter (tsql_get_schema schema name) (lambda (col) (not (internal_column_name? (col "Field"))))) (lambda (ordinal col) (begin
				(define type (if (equal? (coalesceNil (col "SQLDeclaredType") "") "") (toUpper (col "RawType")) (col "SQLDeclaredType")))
				(define id (get_assoc tsql_native_type_ids (toLower type)))
				(define type_row (tsql_catalog_type_row type id (if (> (coalesceNil (col "SQLAliasID") 0) 0) (col "SQLAliasID") id)
					(col "Dimensions") (col "Null") (> (coalesceNil (col "SQLAliasID") 0) 0)))
				(merge (set_assoc (set_assoc type_row "name" (col "Field"))
					"default_object_id" (coalesceNil (get_assoc (coalesceNil (get_assoc (coalesceNil (col "Metadata") '()) "tsql") '()) "default_id") 0))
					(list "object_id" object_id "column_id" (+ ordinal 1)
						"is_identity" (if (equal? (col "Extra") "auto_increment") 1 0) "is_computed" 0)))))))))
		_ (error (concat "unsupported sys catalog " name))))))

(define tsql_qualified_type (lambda (owner name)
	(if (equal?? owner "dbo") name
		(if (and (equal?? owner "sys") (not (nil? (get_assoc tsql_native_type_ids (toLower name))))) name
			(error "type declarations support dbo aliases and sys native types")))))

/* Metadata RPC spelling is a frontend concern too. Wire code only transports
this explicitly chosen database/procedure pair. */
(define tsql_metadata_rpc_name (lambda (text current_database) (begin
	(define qualified (parser (+ tsql_identifier ".")))
	(define names (qualified text))
	(match names
		'(procedure) (list current_database (toLower procedure))
		'(owner procedure) (if (has? '("dbo" "sys") (toLower owner))
			(list current_database (toLower procedure)) (error "unsupported metadata procedure owner"))
		'(database owner procedure) (if (has? '("dbo" "sys") (toLower owner))
			(if (equal?? database current_database) (list current_database (toLower procedure)) (error "cross-database metadata RPC is unsupported"))
			(error "unsupported metadata procedure owner"))
		_ (error "unsupported metadata procedure name")))))

/* Restored declarations select their frontend callbacks before the private
schema is published. The engine applies the returned neutral definitions. */
(registerschemainitializer "tsql-columns" (lambda (context)
	(merge (map (filter (context "columns") (lambda (column) (not (nil? (tsql_column_spec column))))) (lambda (column) (begin
		(define name (column "Field")) (define spec (tsql_column_spec column))
		(define frontend ((column "Metadata") "tsql"))
		(merge
			(if (column "ValueSanitizer") '() (list (list "column" name "value_sanitizer" (tsql_column_sanitizer spec))))
			(if (and (has_assoc? frontend "default_ast") (not (column "DefaultCalculator")))
				(list (list "column" name "default_calculator" (tsql_default_calculator (frontend "default_ast") spec))) '())
			(if (equal? (car spec) "ROWVERSION") (list (list "sequence")) '()))))))))
