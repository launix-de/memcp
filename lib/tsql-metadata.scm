/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Wire metadata belongs to the compiler. Keep declared column semantics even
when a result is empty or its first rows are NULL; protocols must not reparse
SQL or guess whether a string contains binary data. Unknown expressions stay
unknown and use the protocol's bounded runtime discovery. */
(define tsql_field_description (lambda (name type dimensions nullable source)
	(merge (tsql_encode_description (cons type dimensions)) (list "name" name "sql_type" type
		"size" (if (empty_list? dimensions) nil (car dimensions))
		"precision" (tsql_descriptor_integer (if (and (has? '("DECIMAL" "NUMERIC") type) (not (empty_list? dimensions))) (car dimensions) nil))
		"scale" (tsql_descriptor_integer (if (has? '("TIME" "DATETIME2" "DATETIMEOFFSET") type) (if (empty_list? dimensions) 7 (car dimensions))
			(if (tsql_money_type? type) 4 (if (> (count dimensions) 1) (cadr dimensions) nil))))
		"nullable" nullable "source" source "identity" false))))

/* These flags describe declarations, never live row or shard state. A query
must separately prove that its direct base projection is editable. */
(define tsql_base_field_description (lambda (name type dimensions nullable source column)
	(set_assoc (set_assoc (set_assoc (set_assoc
		(tsql_field_description name type dimensions nullable source)
		"rowversion" (coalesceNil (column "RowVersion") false))
		"computed" (coalesceNil (column "Computed") false))
		"key" (has? '("PRI" "UNI") (column "Key")))
		"updatable" (and (not (coalesceNil (column "RowVersion") false))
			(not (coalesceNil (column "Computed") false))
			(not (equal? (column "Extra") "auto_increment"))))))

/* Numeric metadata follows the same native precision/scale calculation as
execution. Describe never needs to evaluate a row or round an unknown value. */
(define tsql_description_numeric_spec (lambda (description)
	(if (nil? description) nil (list (description "sql_type")
		(description "precision") (description "scale")))))
(define tsql_operand_description_spec (lambda (description declared_type)
	(if (nil? description) (if (equal? declared_type "any") nil (list declared_type))
		(if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "BIT" "FLOAT" "REAL") declared_type)
			(list declared_type) (tsql_description_numeric_spec description)))))
(define tsql_description_from_spec (lambda (spec)
	(if (nil? spec) nil (tsql_field_description nil (car spec) (cdr spec) true nil))))
(define tsql_sum_description (lambda (description)
	(if (nil? description) nil
		(match (description "sql_type")
			"DECIMAL" (tsql_field_description nil "DECIMAL" (list 38 (coalesceNil (description "scale") 0)) true nil)
			"NUMERIC" (tsql_field_description nil "DECIMAL" (list 38 (coalesceNil (description "scale") 0)) true nil)
			"MONEY" (tsql_field_description nil "MONEY" '(19 4) true nil)
			"SMALLMONEY" (tsql_field_description nil "MONEY" '(19 4) true nil)
			"REAL" (tsql_field_description nil "FLOAT" '() true nil)
			"FLOAT" (tsql_field_description nil "FLOAT" '() true nil)
			"BIGINT" (tsql_field_description nil "BIGINT" '() true nil)
			"INT" (tsql_field_description nil "INT" '() true nil)
			"SMALLINT" (tsql_field_description nil "INT" '() true nil)
			"TINYINT" (tsql_field_description nil "INT" '() true nil)
			_ nil))))

(define tsql_column_description (lambda (sources alias name planning_session)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (or (not (nil? found)) (and (not (nil? alias)) (not (equal?? alias (source_alias src)))))
			found
			(if (or (source_is_base_table? src) (tsql_catalog_database (source_schema src)))
				(begin
					(define column (find (tsql_get_schema (source_schema src) (source_relation src))
						(lambda (candidate) (equal?? (candidate "Field") name)) nil))
					(define declared (if (nil? column) nil (column "SQLDeclaredType")))
					(if (nil? column) nil
						(set_assoc (tsql_base_field_description name (toUpper (if (or (nil? declared) (equal? declared ""))
							(coalesceNil (column "RawType") "any") declared))
							(coalesceNil (column "Dimensions") '()) (coalesceNil (column "Null") true)
							(list (source_schema src) "dbo" (source_relation src) (column "Field")) column) "identity" (equal? (column "Extra") "auto_increment"))))
				(find (tsql_query_descriptions (source_relation src) planning_session)
					(lambda (candidate) (equal?? (candidate "name") name)) nil)))) nil)))

/* The common expression binder is the sole authority for derived types.
Describing never evaluates a data row and therefore preserves NULL/empty-row
precision and scale. Base projections additionally retain provenance flags. */
(define tsql_expr_description (lambda (sources expr planning_session)
	(match expr
		((symbol get_column) alias _ name _) (tsql_column_description sources alias name planning_session)
		_ (begin
			(define info (sql_expr_info sources (tsql_parameter_expression expr planning_session) true))
			(define spec (sql_info_spec info))
			(if (has? '("any" "NULL") (car spec)) nil
				(tsql_field_description nil (car spec) (cdr spec) true nil))))))

(define tsql_describe_projection (lambda (sources fields planning_session)
	(match fields
		(cons name (cons expr rest)) (begin
			(define description (tsql_expr_description sources expr planning_session))
			(cons (if (nil? description) (tsql_field_description name "any" '() true nil)
				(set_assoc (match expr ((symbol get_column) _ _ _ _) description
					_ (tsql_projection_flags description false)) "name" name))
				(tsql_describe_projection sources rest planning_session)))
		_ '())))

/* UNION declarations use the same coercion rule as the expression binder. */
(define tsql_merge_descriptions (lambda (left right)
	(match left
		(cons first rest) (match right
			(cons other other_rest) (begin
				(define spec (tsql_merge_specs (tsql_description_spec first) (tsql_description_spec other)))
				(cons (tsql_projection_flags (tsql_field_description (first "name") (car spec) (cdr spec) true nil) false)
					(tsql_merge_descriptions rest other_rest)))
			_ (error "UNION column counts do not match"))
		_ (if (empty_list? right) '() (error "UNION column counts do not match")))))

(define tsql_direct_base_projection? (lambda (query)
	(and (query_block? query) (equal? (count (qb_sources query)) 1)
		(source_is_base_table? (car (qb_sources query)))
		(not (source_outer? (car (qb_sources query))))
		(nil? (qb_group query)) (nil? (qb_having query))
		(empty_list? (qb_stages query)))))

(define tsql_projection_flags (lambda (description editable)
	(if (and editable (not (nil? (description "source")))) description
		(set_assoc (set_assoc (set_assoc (set_assoc (set_assoc description
			"updatable" false) "key" false) "source" nil) "nullable" true) "identity" false))))

(define tsql_browse_projection (lambda (query descriptions)
	(if (qassoc_get (qb_facts query) 'tsql_browse false) (begin
		(if (tsql_direct_base_projection? query) true (error "FOR BROWSE requires a direct base-table SELECT"))
		(define src (car (qb_sources query)))
		(define metadata (show (table (source_schema src) (source_relation src)) true))
		(define primary (if metadata (find (metadata "Unique") (lambda (key) (equal? (key "Id") "PRIMARY")) nil) nil))
		(if (and primary (reduce (primary "Cols") (lambda (included name)
			(and included (not (nil? (find descriptions (lambda (description) (begin
				(define source (description "source"))
				(and source (equal?? (nth source 3) name)))) nil))))) true)) true
			(error "FOR BROWSE requires every primary-key column in the projection"))
		(map descriptions (lambda (description) (set_assoc description "browse" true)))) descriptions)))

(define tsql_query_descriptions (lambda (query planning_session)
	(begin
		(define normalized (normalize_query_ast query))
		(if (query_block? normalized)
			(tsql_browse_projection normalized (map (tsql_describe_projection (qb_sources normalized)
				(expand_query_block_fields (qb_sources normalized) (qb_fields normalized)) planning_session)
				(lambda (description) (tsql_projection_flags description (tsql_direct_base_projection? normalized)))))
			(if (union_block? normalized)
				(begin
					(if (reduce (union_branches normalized) (lambda (browse branch)
						(or browse (and (query_block? branch) (qassoc_get (qb_facts branch) 'tsql_browse false)))) false)
						(error "FOR BROWSE cannot describe a UNION") true)
					(match (union_branches normalized)
						(cons first rest) (reduce rest (lambda (descriptions branch)
							(tsql_merge_descriptions descriptions (tsql_query_descriptions branch planning_session)))
							(tsql_query_descriptions first planning_session))
						_ '()))
				'())))))
