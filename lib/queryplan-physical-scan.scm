/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.
You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

/* Physical scan lowering owns expression interpretation, value binding and
residual proofs. Storage receives immutable access data and executable
callbacks. Local operator optimizer hooks may subsequently transform the plan. */
(define scan_plan_list (lambda (expr)
	(match (expression_syntax expr)
		((quote quote) values) values
		(cons (quote list) values) values
		(list? values) values
		_ nil)))

(define scan_plan_columns (lambda (expr)
	(begin
		(define values (scan_plan_list expr))
		(if (and (not (nil? values))
			(reduce values (lambda (valid value) (and valid (string? value))) true))
			values nil))))

(define scan_plan_lambda (lambda (expr)
	(match (expression_syntax expr)
		((quote lambda) params body) (list (coalesceNil params '()) body)
		_ nil)))

(define scan_plan_uses (lambda (expr params)
	(match expr
		((quote quote) _) false
		(list? items) (reduce items (lambda (found item) (or found (scan_plan_uses item params))) false)
		_ (reduce params (lambda (found param) (or found (expression_equal? expr param))) false))))

(define scan_plan_lift (lambda (expr)
	(match expr
		((quote quote) _) expr
		((quote outer) depth inner) (if (> depth 0)
			(if (equal? depth 1) inner (list (quote outer) (- depth 1) inner)) expr)
		(list? items) (map items scan_plan_lift)
		_ expr)))

(define scan_plan_values (lambda (values)
	(if (equal? values '()) '()
		(if (reduce values (lambda (static value)
			(and static (not (symbol? value))
				(if (list? value) (match value ((quote quote) _) true _ false) true))) true)
			(list (quote quote) (map values (lambda (value)
				(match value ((quote quote) literal) literal _ value))))
			(cons (quote list) values)))))

(define scan_plan_true (lambda (expr)
	(or (expression_equal? expr true) (expression_equal? expr (quote true)))))

(define scan_plan_covered (lambda (filter)
	(match (scan_plan_lambda filter) '(_ body) (scan_plan_true body) _ false)))

(define scan_plan_mutates (lambda (expr)
	(begin
		(define columns (scan_plan_columns expr))
		(or (nil? columns) (reduce columns (lambda (mutates column)
			(or mutates (equal? column "$update")
				(and (>= (strlen column) 11) (equal? (substr column 0 11) "$increment:")))) false)))))

/* Boundary records describe requirements; none contain table/shard state. */
(define scan_plan_hoist_safe (lambda (expr)
	(match expr
		((quote quote) _) true
		(cons head args) (and (has? (list (quote session) (quote equal?) (quote equal??) (quote nil?)
			(quote not) (quote sql_not) (quote and) (quote or) (quote coalesceNil) (quote bool?)
			(quote int?) (quote float?) (quote string?) (quote <) (quote <=) (quote >) (quote >=)) head)
			(reduce args (lambda (safe arg) (and safe (scan_plan_hoist_safe arg))) true))
		_ (not (symbol? expr)))))

(define scan_plan_computed_safe (lambda (expr params)
	(if (not (scan_plan_uses expr params)) (scan_plan_hoist_safe expr)
		(match expr
			(cons head args) (and (not (has? params head)) (expression_foldable? head)
				(reduce args (lambda (safe arg) (and safe (scan_plan_computed_safe arg params))) true))
			_ (symbol? expr)))))

(define scan_expression_column (lambda (expr params columns)
	(begin
		(define direct (scan_plan_param_column expr params columns))
		(if (not (nil? direct))
			(if (regexp_test direct "^#[0-9]+$") nil (list direct '() nil))
			(if (and (list? expr) (scan_plan_uses expr params) (scan_plan_computed_safe expr params)
				(not (reduce (produceN (count columns)) (lambda (pseudo idx)
					(or pseudo (and (regexp_test (nth columns idx) "^[#$]")
						(scan_plan_uses expr (list (nth params idx)))))) false)))
				(list (concat "." (expression_name expr params columns)) columns (list (quote lambda) params expr))
				nil)))))

(define scan_plan_boundary (lambda (kind column lower upper lower_set upper_set lower_inclusive upper_inclusive collation null_safe)
	(list "kind" kind "column" (car column) "map_columns" (cadr column) "mapper" (nth column 2)
		"lower" lower "upper" upper "lower_set" lower_set "upper_set" upper_set
		"lower_inclusive" lower_inclusive "upper_inclusive" upper_inclusive
		"collation" collation "null_safe" null_safe)))

(define scan_plan_comparison (lambda (expr params columns)
	(match expr
		'(operator left right)
		(if (has? (list (quote equal?) (quote equal??) (quote <) (quote <=) (quote >) (quote >=)) operator)
			(begin
				(define left_column (scan_expression_column left params columns))
				(define right_column (scan_expression_column right params columns))
				(if (equal? (nil? left_column) (nil? right_column)) nil
					(begin
						(define reversed (nil? left_column))
						(define value (if reversed left right))
						(define column (if reversed right_column left_column))
						(if (scan_plan_uses value params) nil
							(begin
								(define lifted (scan_plan_lift value))
								(if (has? (list (quote equal?) (quote equal??)) operator)
									(scan_plan_boundary "equal" column lifted lifted true true true true
										(if (equal? operator (quote equal??)) "utf8mb4_general_ci" "")
										(equal? operator (quote equal??)))
									(begin
										(define inclusive (has? (list (quote <=) (quote >=)) operator))
										(define lower (not (equal? reversed (has? (list (quote >) (quote >=)) operator))))
										(scan_plan_boundary "range" column (if lower lifted nil) (if lower nil lifted)
											lower (not lower) (and lower inclusive) (and (not lower) inclusive) "" false))))))))
			nil)
		_ nil)))

(define scan_plan_special (lambda (expr params columns)
	(match expr
		((quote nil?) value)
		(match (scan_expression_column value params columns)
			'(column map_columns mapper) (if (nil? mapper)
				(scan_plan_boundary "equal" (list column map_columns mapper) nil nil true true true true "" true) nil)
			_ nil)
		(cons (quote strlike) args)
		(if (or (equal? (count args) 2) (and (equal? (count args) 3) (string? (nth args 2))))
			(match (scan_expression_column (car args) params columns)
				'(column map_columns mapper)
				(if (or (not (nil? mapper)) (scan_plan_uses (cadr args) params)) nil
					(scan_plan_boundary "like" (list column map_columns mapper)
						(scan_plan_lift (cadr args)) (scan_plan_lift (cadr args)) true true true true
						(if (equal? (count args) 3) (toLower (nth args 2)) "utf8mb4_general_ci") false))
				_ nil) nil)
		'(callee value)
		(match (scan_expression_column callee params columns)
			'("$recset_contains" _ _) (if (scan_plan_uses value params) nil
				(scan_plan_boundary "recset" (list "$recset_contains" '() nil)
					(scan_plan_lift value) (scan_plan_lift value) true true true true "" false))
			_ nil)
		_ nil)))

(define scan_plan_merge_boundary (lambda (bounds boundary)
	(if (nil? boundary) bounds
		(begin
			(define existing (filter bounds (lambda (item) (equal? (item "column") (boundary "column")))))
			(if (equal? existing '()) (merge bounds (list boundary))
				(begin
					(define have (car existing))
					(if (and (equal? (have "kind") "range") (equal? (boundary "kind") "range"))
						(if (or (and (have "lower_set") (boundary "lower_set"))
							(and (have "upper_set") (boundary "upper_set"))) nil
							(map bounds (lambda (item)
								(if (equal? (item "column") (boundary "column"))
									(reduce (if (boundary "lower_set") '("lower" "lower_set" "lower_inclusive")
										'("upper" "upper_set" "upper_inclusive"))
										(lambda (result key) (set_assoc result key (boundary key))) item)
									item))))
						bounds)))))))

(define scan_plan_collect (lambda (expr params columns bounds)
	(if (nil? bounds) nil
		(match expr
			(cons (quote and) children)
			(reduce children (lambda (result child) (scan_plan_collect child params columns result)) bounds)
			_ (scan_plan_merge_boundary bounds
				(coalesceNil (scan_plan_comparison expr params columns) (scan_plan_special expr params columns)))))))

(define scan_plan_boundary_rank (lambda (boundary)
	(match (boundary "kind") "equal" 0 "range" 1 _ 2)))

(define scan_plan_boundary_order (lambda (left right)
	(if (equal? (scan_plan_boundary_rank left) (scan_plan_boundary_rank right))
		(< (left "column") (right "column"))
		(< (scan_plan_boundary_rank left) (scan_plan_boundary_rank right)))))

(define scan_plan_exact_prefix (lambda (bounds)
	(if (equal? bounds '()) '()
		(match ((car bounds) "kind")
			"equal" (cons (car bounds) (scan_plan_exact_prefix (cdr bounds)))
			"range" (list (car bounds))
			_ '()))))

(define scan_plan_boundary_covers (lambda (have want)
	(and (reduce '("kind" "column" "collation" "null_safe")
		(lambda (same key) (and same (equal? (have key) (want key)))) true)
		(or (not (want "lower_set"))
			(and (have "lower_set") (expression_equal? (have "lower") (want "lower"))
				(equal? (have "lower_inclusive") (want "lower_inclusive"))))
		(or (not (want "upper_set"))
			(and (have "upper_set") (expression_equal? (have "upper") (want "upper"))
				(equal? (have "upper_inclusive") (want "upper_inclusive")))))))

(define scan_plan_prune (lambda (expr params columns covered)
	(match expr
		(cons (quote and) children)
		(begin
			(define remaining (filter (map children (lambda (child) (scan_plan_prune child params columns covered)))
				(lambda (child) (not (scan_plan_true child)))))
			(match remaining '() true '(only) only _ (cons (quote and) remaining)))
		_ (begin
			(define want (scan_plan_comparison expr params columns))
			(if (and (not (nil? want))
				(or (not (want "null_safe"))
					(and (not (nil? (want "lower"))) (not (list? (want "lower"))) (not (symbol? (want "lower")))))
				(reduce covered (lambda (found have) (or found (scan_plan_boundary_covers have want))) false))
				true expr)))))

(define scan_plan_pack (lambda (bounds values)
	(if (equal? bounds '()) (list '() values)
		(begin
			(define boundary (car bounds))
			(define lower_slot (if (boundary "lower_set") (count values) -1))
			(define lower_values (if (boundary "lower_set") (merge values (list (boundary "lower"))) values))
			(define upper_slot (if (boundary "upper_set")
				(if (equal? (boundary "kind") "range") (count lower_values) lower_slot) -1))
			(define upper_values (if (and (boundary "upper_set") (equal? (boundary "kind") "range"))
				(merge lower_values (list (boundary "upper"))) lower_values))
			(define mapper_slot (if (nil? (boundary "mapper")) -1 (count upper_values)))
			(define next_values (if (nil? (boundary "mapper")) upper_values
				(merge upper_values (list (list (quote compile_scan_computed_index) (boundary "mapper")
					(list (quote quote) (boundary "map_columns")))))))
			(define packed (scan_boundary (boundary "kind") (boundary "column") lower_slot upper_slot
				(boundary "lower_inclusive") (boundary "upper_inclusive") (boundary "collation") (boundary "null_safe")
				mapper_slot (boundary "map_columns")))
			(match (scan_plan_pack (cdr bounds) next_values) '(tail bindings)
				(list (cons packed tail) bindings))))))

/* Return schema, value expressions, residual columns and residual lambda as
one compilation result. All consumers share the same boundary/coverage proof. */
(define scan_plan_param_column (lambda (expr params columns)
	(begin
		(define matches (filter (produceN (count params)) (lambda (idx) (expression_equal? expr (nth params idx)))))
		(if (equal? matches '()) nil (nth columns (car matches))))))

(define scan_plan_feedback_visit (lambda (expr params columns state)
	(if (nil? state) nil
		(match state '(parts bindings nodes)
			(if (>= nodes 64) nil
				(begin
					(define column (scan_plan_param_column expr params columns))
					(if (not (nil? column))
						(if (regexp_test column "^[.$#]") nil
							(list (merge parts (list (concat "column:" (json_encode column)))) bindings (+ nodes 1)))
						(if (or (nil? expr) (expression_equal? expr true) (expression_equal? expr false) (number? expr) (string? expr))
							(list (merge parts (list (count bindings))) (merge bindings (list expr)) (+ nodes 1))
							(match expr
								((quote optimize) inner) (scan_plan_feedback_visit inner params columns (list parts bindings (+ nodes 1)))
								((quote quote) value)
								(if (or (nil? value) (expression_equal? value true) (expression_equal? value false) (number? value) (string? value))
									(scan_plan_feedback_visit value params columns (list parts bindings (+ nodes 1))) nil)
								((quote session) (string? _))
								(list (merge parts (list (count bindings))) (merge bindings (list expr)) (+ nodes 1))
								(cons head args)
								(if (has? (list (quote equal?) (quote equal??) (quote =) (quote <) (quote <=) (quote >) (quote >=)
									(quote strlike) (quote nil?) (quote not) (quote sql_not) (quote and) (quote or) (quote mod)
									(quote +) (quote -) (quote *) (quote /) (quote concat) (quote strlen) (quote coalesceNil)
									(quote if) (quote floor) (quote ceil)) head)
									(match (reduce args (lambda (result arg) (scan_plan_feedback_visit arg params columns result))
										(list (merge parts (list (concat "(" head))) bindings (+ nodes 1)))
										'(tokens values size) (list (merge tokens (list ")")) values size)
										_ nil)
									nil)
								_ nil)))))))))

(define scan_plan_feedback (lambda (params columns body values)
	(if (not (scan_plan_uses body params)) (list nil values)
		(match (scan_plan_feedback_visit body params columns (list '() values 0))
			'(parts bindings _nodes)
			(begin
				(define like_info (match body
					(cons (quote strlike) args)
					(if (and (>= (count args) 2) (<= (count args) 3)
						(or (equal? (count args) 2) (string? (nth args 2))))
						(begin
							(define column (scan_plan_param_column (car args) params columns))
							(if (nil? column) nil (list column (if (equal? (count args) 2) "" (nth args 2))))) nil)
					_ nil))
				(define slots (filter parts (lambda (part) (int? part))))
				(define prior (match body
					'(head _ _) (if (has? (list (quote <) (quote <=) (quote >) (quote >=)) head) (/ 1 3) 0.1)
					_ 0.1))
				(define spec (list parts (if (or (nil? like_info) (equal? slots '())) -1 (car slots))
					(if (nil? like_info) "" (concat (json_encode (car like_info)) ":" (json_encode (cadr like_info)))) prior))
				(list (scan_feedback_key spec bindings) bindings))
			_ (list nil values)))))

(define scan_plan_compile_filter (lambda (column_expr filter_expr)
	(begin
		(define columns (scan_plan_columns column_expr))
		(define callback (scan_plan_lambda filter_expr))
		(if (or (nil? columns) (nil? callback) (not (equal? (count columns) (count (car callback)))))
			(list '() '() column_expr filter_expr)
			(match callback '(params body)
				(begin
					(define bounds (sort (coalesceNil (scan_plan_collect body params columns '()) '()) scan_plan_boundary_order))
					(define residual (scan_plan_prune body params columns (scan_plan_exact_prefix bounds)))
					(define used (filter (produceN (count params)) (lambda (idx) (scan_plan_uses residual (list (nth params idx))))))
					(define remaining_columns (if (equal? bounds '()) column_expr
						(list (quote quote) (map used (lambda (idx) (nth columns idx))))))
					(define remaining_filter (if (equal? bounds '()) filter_expr
						(list (quote lambda) (map used (lambda (idx) (nth params idx))) residual)))
					(define mapped (filter bounds (lambda (boundary) (not (nil? (boundary "mapper"))))))
					(match (scan_plan_pack bounds '()) '(packed values)
						(match (scan_plan_feedback params columns body values) '(feedback bindings)
							(list (scan_access_schema packed (if (equal? mapped '()) '() ((car mapped) "map_columns"))
								feedback (scan_plan_covered remaining_filter))
								bindings remaining_columns remaining_filter)))))))))

/* The returned expression is evaluated by the caller, so references to its
lexical bindings never escape into the compiler's environment. */
(define compile_scan_access (lambda arguments
	(match arguments (merge '(columns callback) options)
		(begin
			(define feedback_only (and (not (equal? options '())) (car options)))
			(define compiled (if feedback_only
				(match (scan_plan_lambda callback)
					'(params body) (match (scan_plan_feedback params (scan_plan_columns columns) body '()) '(feedback values)
						(list (scan_access_schema '() '() feedback false) values))
					_ (list '() '()))
				(scan_plan_compile_filter columns callback)))
			(list (quote list) (list (quote quote) (scan_access_cover (car compiled) false))
				(scan_plan_values (cadr compiled)))))))

(define scan_plan_compile_multi (lambda (columns filters shift)
	(if (equal? columns '()) (list '() '() '() '())
		(match (scan_plan_compile_filter (car columns) (car filters)) '(schema values residual_columns residual)
			(match (scan_plan_compile_multi (cdr columns) (cdr filters) (+ shift (count values)))
				'(schemas bindings column_lists callbacks)
				(list (cons (scan_access_shift schema shift) schemas) (merge values bindings)
					(cons (if (nil? (scan_plan_columns residual_columns)) residual_columns
						(list (quote quote) (scan_plan_columns residual_columns))) column_lists)
					(cons residual callbacks)))))))

(define compile_scan_plan (lambda arguments
	(if (< (count arguments) 5) (error "compile_scan_plan expects an operator, transaction, source, filter columns, and filter callback")
		(match arguments (merge '(operator tx source columns filter) rest)
			(begin
				(define static (scan_plan_list columns))
				(if (scan_access_schema? static)
					(cons operator (merge (list tx source
						(list (quote quote) (scan_access_cover static (and (> (count rest) 1) (scan_plan_covered (cadr rest))))) filter) rest))
					(if (and (list? static) (not (equal? static '())) (scan_access_schema? (scan_plan_list (car static))))
						(cons operator (merge (list tx source columns filter) rest))
						(if (has? (list (quote scan_order_multi) (quote scan_join_order)) operator)
							(begin
								(define filters (scan_plan_list filter))
								(if (or (nil? static) (nil? filters) (not (equal? (count static) (count filters))))
									(error "compile_scan_plan expects matching static multi-scan filters")
									(match (scan_plan_compile_multi static filters 0) '(schemas values residual_columns residuals)
										(cons operator (merge (list tx source (list (quote quote) schemas) (scan_plan_values values)
											(cons (quote list) residual_columns) (cons (quote list) residuals)) rest)))))
							(if (not (has? (list (quote scan) (quote scan_order) (quote scan_recset) (quote scan_exists)
								(quote scan_selectivity_estimate)) operator))
								(error "compile_scan_plan received unsupported operator " operator)
								(match (scan_plan_compile_filter columns filter) '(schema values residual_columns residual)
									(begin
										(define mutates (and (equal? operator (quote scan)) (scan_plan_mutates (car rest))))
										(cons operator (merge (list tx source (if (equal? schema '()) schema (list (quote quote) schema))
											(scan_plan_values values) (if mutates columns residual_columns) (if mutates filter residual)) rest)))))))))))))
