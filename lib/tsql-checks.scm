/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* CHECK is a stored row-local scalar recipe. Native storage owns declaration
admission, final row timing, transaction undo and persistence; this compiler owns
SQL names, types, collation and UNKNOWN. ISDATE consumes only its two explicit
locale inputs from the actual native invocation; arbitrary session-dependent
scalars remain rejected. No creator session is retained in the program. */
(define tsql_check_source (lambda (database columns) (list
	"query" database "__check_input" false nil
	(list "columns" (map columns sql_column_declaration)
		"constraints" '("Unique" () "ForeignKeys" ())))))

(define tsql_check_formula_safe? (lambda (formula) (match formula
	((symbol get_column) _alias _alias_ic _name _name_ic) true
	((symbol quote) value) (begin (sql_declaration_data value) true)
	((symbol sql_compare) left right _comparison _operation)
	(and (tsql_check_formula_safe? left) (tsql_check_formula_safe? right))
	/* The exact arithmetic boundary consumes the invocation's scalar option.
	Its thunk is compiler-generated and contains only validated row formulas. */
	(list head enabled ((symbol lambda) '() body))
	(if (and (or (equal? head 'tsql_arithmetic_value) (expression_equal? head tsql_arithmetic_value))
		(expression_equal? enabled (tsql_ansi_warnings_formula)))
		(tsql_check_formula_safe? body) false)
	((symbol session) _key) (error "CHECK session-dependent scalar expressions are not supported")
	(cons head arguments) (if (has? (list tsql_isdate_text) head)
		/* The compiler embeds the actual procedure value, not its symbol. */
		(match arguments
			(list value order language)
			(if (and (expression_equal? order (tsql_dateformat_formula))
				(expression_equal? language (tsql_language_formula)))
				(tsql_check_formula_safe? value)
				(error "CHECK ISDATE requires the canonical invocation locale inputs"))
			_ false)
		(and
			(has? (list 'if 'and 'or 'not 'sql_not 'nil? 'coalesceNil
				'equal? 'equal?? '< '<= '> '>= '+ '- '* '/ 'div_null 'intdiv 'mod
				'tsql_cast_integer 'tsql_cast_float 'tsql_cast_decimal 'tsql_cast_text
				'tsql_cast_fixed_text 'tsql_numeric_input 'sql_decimal_value
				'tsql_checked_integer_operation tsql_checked_integer_operation
				'tsql_integer_divide 'tsql_fractional_divide tsql_integer_divide tsql_fractional_divide
				+ - * / div_null intdiv mod coalesceNil nil? equal? equal?? < <= > >= not sql_not
				tsql_cast_integer tsql_cast_float tsql_cast_decimal tsql_cast_text
				tsql_cast_fixed_text tsql_numeric_input sql_decimal_value
				tsql_isnumeric_text tsql_len tsql_isdate_temporal tsql_isdate_other
				tsql_cast_numeric_text tsql_cast_money_text tsql_decimal_text tsql_float_text
				'strlike strlike
				/* Exact context-free character comparison owners, not arbitrary
				caller procedures or session-dependent functions. */
				'tsql_padded_text_equal tsql_padded_text_equal
				'tsql_padded_text_in tsql_padded_text_in 'list list
				(sql_builtins "SQL_NULL")) head)
			(reduce arguments (lambda (safe child) (and safe (tsql_check_formula_safe? child))) true)))
	_ (or (nil? formula) (number? formula) (string? formula)
		(identical? formula true) (identical? formula false)))))

(define tsql_check_expression (lambda (database source columns) (begin
	(define relation (tsql_check_source database columns))
	(define planning (newsession))
	(planning "__tsql_expression_only" true)
	(planning "__tsql_expression_source" relation)
	(define info (parse_tsql database source nil planning nil))
	(if (has? '("BOOLEAN" "NULL") (sql_info_type info)) true
		(error "CHECK requires a Boolean row-local expression"))
	(define formula (sql_type_lower_annotations (list relation) (sql_info_formula info)))
	(if (tsql_check_formula_safe? formula) true
		(error "CHECK requires a supported row-local scalar expression"))
	(list relation formula (extract_columns_for_alias relation formula)))))

/* The same existing source-compiler return contract is used by both eager ADD
and lazy restart: an ordinary four-runtime-argument procedure, never a tuple or
creator environment. Column locals are formal inputs of an inner scalar call. */
(define tsql_compile_check (lambda (source context) (begin
	(define expression (tsql_check_expression (context "schema") source (context "columns")))
	(define relation (car expression))
	(define inputs (nth expression 2))
	(define declared (map (context "columns") (lambda (column) (column "Field"))))
	(if (expression_equal? inputs declared) true
		(error "CHECK source dependency contract changed"))
	(define body (cons
		(list 'lambda (map inputs (lambda (column) (scan_callback_symbol_for_alias "query" column)))
			(lower_column_expr_for_alias relation (nth expression 1)))
		(map inputs (lambda (column) (list 'NEW column)))))
	/* SQL UNKNOWN satisfies CHECK. Evaluate the row expression exactly once;
	native storage consumes only the language's normalized Boolean result. */
	(define value (quote __tsql_check_result))
	(define predicate (list (quote match) body value
		(list (quote if) (list (quote nil?) value) true value)))
	(eval (optimize (tsql_computed_function
		(list (quote OLD) (quote NEW) (quote session) (quote tx)) predicate))))))
(registertriggerlanguage "tsql-check" tsql_compile_check true true)

(define tsql_define_check (lambda (database name id source tx session) (begin
	(define handle (table database false name false tx))
	(if handle true (error "CHECK table does not exist: " name))
	/* Binding discovers names only. Native ADD recompiles against the actual
	leased input contract, so a concurrent schema publication cannot install a
	predicate typed against an unrelated earlier declaration. */
	(define expression (tsql_check_expression database source ((show handle "declarations" tx) "columns")))
	(createcheck handle id "tsql-check" source (nth expression 2) tx session))))

/* All SQL names are resolved before the ONE native state operation. Its owner
then validates complete pending ownership and historical rows for both kinds. */
(define tsql_constraint_name (lambda (constraints name) (begin
	(define foreign (find (constraints "ForeignKeys") (lambda (key)
		(and (equal? (key "Role") "referencing") (equal?? (key "Id") name))) nil))
	(define check (find (coalesceNil (constraints "Checks") '())
		(lambda (key) (equal?? (key "Id") name)) nil))
	(if foreign (foreign "Id") check (check "Id")
		/* Preserve the old rejection contract for unsupported/missing targets. */
		(error "constraint is not a pending foreign declaration: " name)))))
