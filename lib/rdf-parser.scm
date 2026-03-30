/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* RDF parser according to: https://www.w3.org/TR/sparql11-query/ */

(define rdf_variable (parser (define x (regex "\?[a-zA-Z0-9_]+" true)) '('get_var (symbol x))))
/* RDF planner contract: variable keys may flow through the parser as symbols and
aggregate aliases/materialized rows as strings. All planner lookups must treat
those forms as equivalent and defer the actual row/column substitution until the
consumer stage. */
(define rdf_key_equal (lambda (a b)
	(or (equal? a b) (equal? (concat a) (concat b)))
))
(define rdf_key_in_list (lambda (items key)
	(reduce items (lambda (acc item) (or acc (rdf_key_equal item key))) false)
))
/* datatype suffix parser: consumes ^^<IRI> or ^^prefix:name or ^^barename */
(define rdf_datatype_suffix (parser (or
	(parser '((atom "<" false false) (regex "[^>]*" false false) (atom ">" false false)) nil) /* ^^<IRI> */
	(regex "[a-zA-Z0-9_]*:[a-zA-Z0-9_]*" false false) /* ^^prefix:name */
	(regex "[a-zA-Z0-9_]+" false false) /* ^^barename */
	)))
/* unescape standard TTL/JSON escape sequences in a string */
(define rdf_unescape (lambda (s)
	(replace (replace (replace (replace (replace s "\\n" "\n") "\\t" "\t") "\\\\" "\\") "\\\"" "\"") "\\r" "\r")
))
(define rdf_unbound_expr (lambda () '("__rdf_unbound__")))
(define rdf_unbound_expr? (lambda (expr) (equal? expr '("__rdf_unbound__"))))
(define rdf_ctx_lookup (lambda (ctx sym) (match ctx
	(cons key (cons val tail))
		(if (rdf_key_equal key sym) (list true val) (rdf_ctx_lookup tail sym))
	'()
		(list false nil)
)))
(define rdf_ctx_bound (lambda (ctx sym)
	(match (rdf_ctx_lookup ctx sym) '(found val)
		(and found (not (rdf_unbound_expr? val)))
	)
))
(define rdf_ctx_value (lambda (ctx sym)
	(match (rdf_ctx_lookup ctx sym) '(found val)
		(if found
			(if (rdf_unbound_expr? val) nil val)
			nil
		)
	)
))
(define rdf_contains (lambda (s needle)
	(if (or (nil? s) (nil? needle))
		nil
		(not (equal? (replace s needle "") s))
	)
))
(define rdf_strlen (lambda (s) (if (nil? s) nil (strlen s))))
(define rdf_startswith (lambda (s prefix)
	(if (or (nil? s) (nil? prefix))
		nil
		(equal? (sql_substr s 1 (strlen prefix)) prefix)
	)
))
(define rdf_endswith (lambda (s suffix)
	(if (or (nil? s) (nil? suffix))
		nil
		(equal? (sql_substr s (+ (- (strlen s) (strlen suffix)) 1) (strlen suffix)) suffix)
	)
))
/* produce a quoted TTL string literal from a raw value: rdf_quote("hello") -> "\"hello\"" */
(define rdf_quote (lambda (s)
	(concat "\"" (replace (replace (replace (replace (replace s "\\" "\\\\") "\"" "\\\"") "\n" "\\n") "\t" "\\t") "\r" "\\r") "\"")
))
(define rdf_constant (parser (or
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x) /* IRI */
	(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) x) /* triple-quoted string, optional datatype ignored */
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x)) /* string with language */
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) (rdf_unescape x)) /* string with escapes */
	(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node _:identifier */
	(regex "[a-zA-Z0-9_]+" true) /* bare name */
)))
(define rdf_expression (parser (or
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) '('concat '('definitions pfx) post)) /* as expression */
	rdf_variable
	rdf_constant
	/* TODO: CONCAT() */
)))
(define rdf_aggregate_expression (parser (or
	(parser '((atom "COUNT" true) "(" "*" ")") '("__rdf_agg__" "COUNT" 1 nil))
	(parser '((atom "COUNT" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "COUNT" e nil))
	(parser '((atom "SUM" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "SUM" e nil))
	(parser '((atom "AVG" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "AVG" e nil))
	(parser '((atom "MIN" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "MIN" e nil))
	(parser '((atom "MAX" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "MAX" e nil))
	(parser '((atom "GROUP_CONCAT" true) "(" (define e rdf_filter_or) ";" (atom "separator" true) "=" (define sep rdf_filter_or) ")") '("__rdf_agg__" "GROUP_CONCAT" e sep))
	(parser '((atom "GROUP_CONCAT" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "GROUP_CONCAT" e ","))
)))

/* SPARQL filter expressions — no bare names (would eat keywords) */
(define rdf_filter_atom (parser (or
	rdf_variable
	(parser '((define n (regex "[0-9]+" true))) (simplify n))
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x)
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false)) (rdf_unescape x))
	(parser '((atom "STR" true) "(" (define a rdf_filter_or) ")") '('concat a))
	(parser '((atom "IRI" true) "(" (define a rdf_filter_or) ")") '('concat a))
	(parser '((atom "CONCAT" true) "(" (define args (+ rdf_filter_or ",")) ")") (cons 'sql_concat args))
	(parser '((atom "STRLEN" true) "(" (define a rdf_filter_or) ")") '('rdf_strlen a))
	(parser '((atom "CONTAINS" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('rdf_contains a b))
	(parser '((atom "STRSTARTS" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('rdf_startswith a b))
	(parser '((atom "STRENDS" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('rdf_endswith a b))
	(parser '((atom "COALESCE" true) "(" (define args (+ rdf_filter_or ",")) ")") (cons (quote coalesceNil) args))
	(parser '((atom "IF" true) "(" (define cond rdf_filter_or) "," (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('if cond a b))
	(parser '((atom "BOUND" true) "(" (define v rdf_variable) ")") '('rdf_bound v))
	(parser '("(" (define e rdf_filter_or) ")") e)
	(parser '((atom "regex" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('regexp_test a b))
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) '('concat '('definitions pfx) post))
)))
(define rdf_filter_not (parser (or
	(parser '("!" (define e rdf_filter_atom)) '('not e))
	rdf_filter_atom
)))
(define rdf_filter_cmp (parser (or
	(parser '((define a rdf_filter_not) "!=" (define b rdf_filter_not)) '('not '('equal? a b)))
	(parser '((define a rdf_filter_not) "=" (define b rdf_filter_not)) '('equal? a b))
	(parser '((define a rdf_filter_not) "<=" (define b rdf_filter_not)) '('<= a b))
	(parser '((define a rdf_filter_not) ">=" (define b rdf_filter_not)) '('>= a b))
	(parser '((define a rdf_filter_not) "<" (define b rdf_filter_not)) '('< a b))
	(parser '((define a rdf_filter_not) ">" (define b rdf_filter_not)) '('> a b))
	rdf_filter_not
)))
(define rdf_filter_and (parser (or
	(parser '((define a rdf_filter_cmp) "&&" (define b rdf_filter_and)) '('and a b))
	rdf_filter_cmp
)))
(define rdf_filter_or (parser (or
	(parser '((define a rdf_filter_and) "||" (define b rdf_filter_or)) '('or a b))
	rdf_filter_and
)))

(define rdf_path_atom (parser (or
	(parser '("(" (define p rdf_path_alt) ")") p)
	rdf_expression
)))
(define rdf_path_postfix (parser (or
	(parser '((define p rdf_path_atom) "*") '("__path_star__" p))
	(parser '((define p rdf_path_atom) "+") '("__path_plus__" p))
	rdf_path_atom
)))
(define rdf_path_seq (parser (or
	(parser '((define a rdf_path_postfix) "/" (define b rdf_path_seq)) '("__path_seq__" a b))
	rdf_path_postfix
)))
(define rdf_path_alt (parser (or
	(parser '((define a rdf_path_seq) "|" (define b rdf_path_alt)) '("__path_alt__" a b))
	rdf_path_seq
)))

(define rdf_where_basic_item (parser (or
	(parser '((define s rdf_expression) (define ps (+ (parser '((define p rdf_path_alt) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
	(parser '((atom "FILTER" true) "(" (define expr rdf_filter_or) ")") (list (list "__filter__" expr)))
)))
(define rdf_where_inner_basic_items (parser
	(* (parser '((define item rdf_where_basic_item) (? (atom "." true))) item))
))
(define rdf_where_optional_item (parser '(
	(atom "OPTIONAL" true)
	(atom "{" true)
	(define conditions rdf_where_inner_basic_items)
	(atom "}" true)
) (list (list "__optional__" (merge (coalesce conditions '('())))))))
(define rdf_where_bind_item (parser '(
	(atom "BIND" true)
	"("
	(define expr rdf_filter_or)
	(atom "AS" true)
	(define var rdf_variable)
	")"
) (list (list "__bind__" expr var))))
(define rdf_where_filter_not_exists_item (parser '(
	(atom "FILTER" true)
	(atom "NOT" true)
	(atom "EXISTS" true)
	(atom "{" true)
	(define conditions rdf_where_inner_basic_items)
	(atom "}" true)
) (list (list "__filter_exists__" true (merge (coalesce conditions '('())))))))
(define rdf_where_filter_yes_exists_item (parser '(
	(atom "FILTER" true)
	(atom "EXISTS" true)
	(atom "{" true)
	(define conditions rdf_where_inner_basic_items)
	(atom "}" true)
) (list (list "__filter_exists__" false (merge (coalesce conditions '('())))))))
(define rdf_where_filter_exists_item (parser (or
	rdf_where_filter_not_exists_item
	rdf_where_filter_yes_exists_item
)))
(define rdf_where_union_group (parser '(
	(atom "{" true)
	(define conditions rdf_where_inner_basic_items)
	(atom "}" true)
) (merge (coalesce conditions '('())))))
(define rdf_where_union_tail_item (parser '(
	(atom "UNION" true)
	(define next rdf_where_union_group)
) next))
(define rdf_where_values_item (parser '(
	(atom "VALUES" true)
	(define var rdf_variable)
	(atom "{" true)
	(define vals (* rdf_expression))
	(atom "}" true)
) (list (list "__values__" var vals))))
(define rdf_where_subquery_item (parser '(
	(atom "{" true)
	(atom "SELECT" true)
	"("
	(define agg rdf_aggregate_expression)
	(atom "AS" true)
	(define var rdf_variable)
	")"
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_basic_item) (? (atom "." true))) item)))
	(atom "}" true)
	(atom "}" true)
	) (match var '('get_var s)
		(list (list "__subquery__"
			(list "select" (merge (list (list s agg))) "where" (merge (coalesce conditions '('()))) "group" '() "order" nil "limit" nil "offset" nil "distinct" nil)
		)))
	)))
(define rdf_where_union_item (parser '(
	(define first rdf_where_union_group)
	(atom "UNION" true)
	(define second rdf_where_union_group)
	(define rest (* rdf_where_union_tail_item))
) (list (list "__union__" (cons first (cons second rest))))))
(define rdf_where_item (parser (or
	rdf_where_union_item
	rdf_where_filter_exists_item
	rdf_where_values_item
	rdf_where_subquery_item
	rdf_where_optional_item
	rdf_where_bind_item
	rdf_where_basic_item
)))
(define rdf_var_symbol (lambda (expr) (match expr
	'('get_var sym) sym
	'((quote get_var) sym) sym
	(error "SPARQL error: expected variable, got " expr)
)))
(define rdf_select_col (parser (or
	(parser '("(" (define v rdf_aggregate_expression) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '((define v rdf_expression) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v)))
	(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v)))
)))

(define rdf_number (parser (define x (regex "[0-9]+" true)) (simplify x)))
(define rdf_select (parser '(
	(atom "SELECT" true)
	(? (define distinct (atom "DISTINCT" true)))
	(define cols (+ rdf_select_col ","))
	(?
		(atom "WHERE" true)
		(atom "{" true)
		(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
		(atom "}" true) /* TODO: {} UNION {} */
	)
	(?
		(atom "GROUP" true)
		(atom "BY" true)
		(define group (+ rdf_variable ","))
	)
	(?
		(atom "ORDER" true)
		(atom "BY" true)
		(define ordercols (+ (or
			(parser '((define dir (or (atom "DESC" true) (atom "ASC" true))) "(" (define expr rdf_expression) ")") '(expr dir))
			(parser (define expr rdf_expression) '(expr "ASC"))
		) ","))
	)
	(? (atom "LIMIT" true) (define limit rdf_number))
	(? (atom "OFFSET" true) (define offset rdf_number))
) '("select" (merge cols) "where" (merge (coalesce conditions '('()))) "group" (coalesce group '()) "order" ordercols "limit" limit "offset" offset "distinct" distinct) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

(define rdf_template_item (parser '(
	(define s rdf_expression)
	(define ps (+ (parser '((define p rdf_expression) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))
) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1))))))))
(define rdf_template_items (parser
	(* (parser '((define item rdf_template_item) (? (atom "." true))) item))
))
(define rdf_insert_data (parser '(
	(atom "INSERT" true)
	(atom "DATA" true)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
) '("insert_data" (merge (coalesce triples '('()))))))
(define rdf_delete_data (parser '(
	(atom "DELETE" true)
	(atom "DATA" true)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
) '("delete_data" (merge (coalesce triples '('()))))))
(define rdf_delete_insert_where (parser '(
	(atom "DELETE" true)
	(atom "{" true)
	(define delete_triples rdf_template_items)
	(atom "}" true)
	(atom "INSERT" true)
	(atom "{" true)
	(define insert_triples rdf_template_items)
	(atom "}" true)
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) '("modify" "delete" (merge (coalesce delete_triples '('()))) "insert" (merge (coalesce insert_triples '('()))) "where" (merge (coalesce conditions '('()))))))
(define rdf_ask (parser '(
	(atom "ASK" true)
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) '("ask" "where" (merge (coalesce conditions '('()))))))
(define rdf_construct (parser '(
	(atom "CONSTRUCT" true)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) '("construct" (merge (coalesce triples '('()))) "where" (merge (coalesce conditions '('()))))))
(define rdf_query (parser (or
	rdf_delete_insert_where
	rdf_insert_data
	rdf_delete_data
	rdf_ask
	rdf_construct
	rdf_select
)))

(define ttl_header (parser '(
	(define definitions (*
		(or
			(parser '((atom "@prefix" true) (define pfx (regex "[a-zA-Z0-9_]*" false)) (atom ":" false false) (define content rdf_constant) ".") '(pfx content))
			(parser '((atom "@base" true) (define content rdf_constant) ".") '("" content)) /* @base sets the empty prefix */
		)
	))
	(define rest rest)
) '("prefixes" (merge definitions) "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

(define rdf_replace_ctx (lambda (expr ctx) (match expr
	'('rdf_bound ('get_var sym)) (rdf_ctx_bound ctx sym)
	'('rdf_bound ((quote get_var) sym)) (rdf_ctx_bound ctx sym)
	'((quote rdf_bound) ('get_var sym)) (rdf_ctx_bound ctx sym)
	'((quote rdf_bound) ((quote get_var) sym)) (rdf_ctx_bound ctx sym)
	'('get_var sym) (rdf_ctx_value ctx sym)
	'((quote get_var) sym) (rdf_ctx_value ctx sym)
	(cons head tail) (cons head (map tail (lambda (x) (rdf_replace_ctx x ctx))))
	expr
)))

(define rdf_extract_vars (lambda (expr) (match expr
	'('get_var sym) (list sym)
	'((quote get_var) sym) (list sym)
	(cons head tail) (merge_unique (cons (rdf_extract_vars head) (map tail rdf_extract_vars)))
	'()
)))

(define rdf_condition_vars (lambda (conditions)
	(merge_unique (map conditions (lambda (cond) (match cond
		'("__filter__" expr) (rdf_extract_vars expr)
		'("__union__" branches)
			(reduce branches (lambda (acc branch)
				(merge_unique (list acc (rdf_condition_vars branch)))
			) '())
		'("__optional__" inner) (rdf_condition_vars inner)
		'("__bind__" expr var_expr) (merge_unique (list (rdf_extract_vars expr) (list (rdf_var_symbol var_expr))))
		'("__values__" var_expr _vals) (list (rdf_var_symbol var_expr))
		'("__subquery__" subquery)
			(match subquery
				'("select" subcols "where" subconds "group" subgroup "order" suborder "limit" sublimit "offset" suboffset "distinct" subdistinct)
					(reduce_assoc subcols (lambda (acc alias expr) (append acc alias)) '())
				'()
			)
		'(s p o) (merge_unique (list (rdf_extract_vars s) (rdf_extract_vars p) (rdf_extract_vars o)))
		'()
	))))))
(define rdf_missing_select_vars (lambda (cols conditions)
	(begin
		(define available_vars (rdf_condition_vars conditions))
		(define selected_vars
			(reduce_assoc cols (lambda (acc _alias expr)
				(merge_unique (list acc (rdf_extract_vars expr)))
			) '()))
		(filter
			selected_vars
			(lambda (var) (not (rdf_key_in_list available_vars var)))
		)
	)
))
(define rdf_strip_leading_ws_comments (lambda (s) (match s
	(regex "(?s)^(?:[\\r\\n\\t ]+|/\\*.*?\\*/|--[^\\r\\n]*(?:\\r?\\n|$)|#[^\\r\\n]*(?:\\r?\\n|$))(.*)$" _ rest)
		(rdf_strip_leading_ws_comments rest)
	s
)))
(define rdf_resolve_prefixes (lambda (expr definitions) (match expr
	'('concat ('definitions pfx) post)
		(if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post))
	'((quote concat) ((quote definitions) pfx) post)
		(if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post))
	(cons head tail) (cons (rdf_resolve_prefixes head definitions) (map tail (lambda (x) (rdf_resolve_prefixes x definitions))))
	expr
)))
(define rdf_row_items (lambda (cols ctx) (match cols
	(cons key (cons val tail))
		(cons (concat key) (cons (rdf_replace_ctx val ctx) (rdf_row_items tail ctx)))
	'()
)))
(define rdf_select_resultrow_ast (lambda (row_cols ctx)
	(list (quote resultrow) (cons list (rdf_row_items row_cols ctx)))
))
(define rdf_wrapped_resultrow_ast (lambda (state row_expr distinct effective_offset effective_limit)
	(if distinct
		(list (quote begin)
			(list (quote set) (quote _row) row_expr)
			(list (quote set) (quote _dkey) (list (quote json_encode) (quote _row)))
			(list (quote if)
				(list (quote not) (list (state "seen") (quote _dkey)))
				(list (quote begin)
					(list (state "seen") (quote _dkey) true)
					(list (quote set) (quote _c) (list state "cnt"))
					(list (quote if)
						(list (quote and)
							(list (quote >=) (quote _c) effective_offset)
							(list (quote <) (quote _c) (+ effective_offset effective_limit)))
						(list (quote resultrow) (quote _row))
						nil)
					(list state "cnt" (list (quote +) (quote _c) 1)))
				nil))
		(list (quote begin)
			(list (quote set) (quote _row) row_expr)
			(list (quote set) (quote _c) (list state "cnt"))
			(list (quote if)
				(list (quote and)
					(list (quote >=) (quote _c) effective_offset)
					(list (quote <) (quote _c) (+ effective_offset effective_limit)))
				(list (quote resultrow) (quote _row))
				nil)
			(list state "cnt" (list (quote +) (quote _c) 1)))
	)
))
(define rdf_aggregate_expr (lambda (expr) (match expr
	'("__rdf_agg__" fn inner sep) '(fn inner sep)
	nil
)))
(define rdf_has_aggregate (lambda (expr) (match expr
	'("__rdf_agg__" _ _ _) true
	(cons head tail) (or (rdf_has_aggregate head) (reduce tail (lambda (acc item) (or acc (rdf_has_aggregate item))) false))
	false
)))
(define rdf_select_has_aggregates (lambda (cols)
	(reduce_assoc cols (lambda (acc _ expr) (or acc (rdf_has_aggregate expr))) false)
))
(define rdf_select_capture_vars (lambda (cols group)
	(merge_unique (list
		(reduce group (lambda (acc gexpr) (merge acc (rdf_extract_vars gexpr))) '())
		(reduce_assoc cols (lambda (acc _ expr) (merge acc (rdf_extract_vars expr))) '())
	))
))
(define rdf_capture_row_items (lambda (vars ctx) (match vars
		(cons var tail) (cons
			(list (quote list) (list (quote quote) var) (rdf_ctx_value ctx var))
			(rdf_capture_row_items tail ctx))
		'()
)))
(define rdf_row_missing (lambda () '("__rdf_row_missing__")))
(define rdf_row_key_equal (lambda (a b) (rdf_key_equal a b)))
(define rdf_row_lookup (lambda (row sym) (match row
	(cons (cons key (cons val '())) tail)
		(if (rdf_row_key_equal key sym) val (rdf_row_lookup tail sym))
	(cons key (cons val tail))
		(if (rdf_row_key_equal key sym) val (rdf_row_lookup tail sym))
	'()
		(rdf_row_missing)
)))
(define rdf_replace_row (lambda (expr row) (match expr
	'('get_var sym) (begin
		(define v (rdf_row_lookup row sym))
		(if (equal? v (rdf_row_missing))
			nil
			v))
	'((quote get_var) sym) (begin
		(define v (rdf_row_lookup row sym))
		(if (equal? v (rdf_row_missing))
			nil
			v))
	(cons head tail) (cons head (map tail (lambda (x) (rdf_replace_row x row))))
	expr
)))
(define rdf_row_eval (lambda (expr row)
	(eval (rdf_replace_row expr row))
))
(define rdf_numeric_value (lambda (v)
	(if (number? v)
		v
		(simplify (concat v))
	)
))
(define rdf_agg_init (lambda (expr) (match expr
	'("__rdf_agg__" "COUNT" _ _) 0
	'("__rdf_agg__" "SUM" _ _) 0
	'("__rdf_agg__" "AVG" _ _) '(0 0)
	'("__rdf_agg__" "MIN" _ _) nil
	'("__rdf_agg__" "MAX" _ _) nil
	'("__rdf_agg__" "GROUP_CONCAT" _ _) nil
	(error "unsupported RDF aggregate " expr)
)))
(define rdf_agg_step (lambda (expr state row) (match expr
	'("__rdf_agg__" "COUNT" inner _)
		(if (nil? (rdf_row_eval inner row)) state (+ state 1))
	'("__rdf_agg__" "SUM" inner _)
		(if (nil? (rdf_row_eval inner row)) state (+ state (rdf_numeric_value (rdf_row_eval inner row))))
	'("__rdf_agg__" "AVG" inner _)
		(if (nil? (rdf_row_eval inner row))
			state
			(list (+ (car state) (rdf_numeric_value (rdf_row_eval inner row))) (+ (cadr state) 1)))
	'("__rdf_agg__" "MIN" inner _)
		(begin
			(define v (rdf_row_eval inner row))
			(if (nil? v)
				state
				(if (nil? state)
					v
					(if (< (rdf_numeric_value v) (rdf_numeric_value state)) v state))))
	'("__rdf_agg__" "MAX" inner _)
		(begin
			(define v (rdf_row_eval inner row))
			(if (nil? v)
				state
				(if (nil? state)
					v
					(if (> (rdf_numeric_value v) (rdf_numeric_value state)) v state))))
	'("__rdf_agg__" "GROUP_CONCAT" inner sep)
		(begin
			(define v (rdf_row_eval inner row))
			(if (nil? v)
				state
				(if (nil? state) (concat v) (concat state sep v))))
	(error "unsupported RDF aggregate " expr)
)))
(define rdf_agg_finalize (lambda (expr state) (match expr
	'("__rdf_agg__" "AVG" _ _) (if (equal? (cadr state) 0) nil (/ (car state) (cadr state)))
	_ state
)))
(define rdf_groups_lookup (lambda (groups gkey) (match groups
	(cons (cons key (cons gstate '())) tail)
		(if (equal? key gkey) gstate (rdf_groups_lookup tail gkey))
	'()
		nil
)))
(define rdf_groups_upsert (lambda (groups gkey gstate) (match groups
	(cons (cons key (cons oldstate '())) tail)
		(if (equal? key gkey)
			(cons (list gkey gstate) tail)
			(cons (list key oldstate) (rdf_groups_upsert tail gkey gstate)))
	'()
		(list (list gkey gstate))
)))
(define rdf_init_group_state (lambda (cols row) (begin
	(define gstate (newsession))
	(reduce_assoc cols (lambda (_ alias expr) (begin
		(gstate alias (if (rdf_has_aggregate expr) (rdf_agg_init expr) (if (nil? row) nil (rdf_row_eval expr row))))
		nil
	)) nil)
	gstate
)))
(define rdf_ensure_group (lambda (groups cols gkey row) (begin
	(define existing (rdf_groups_lookup groups gkey))
	(if existing
		(list groups existing)
		(begin
			(define gstate (rdf_init_group_state cols row))
			(list (rdf_groups_upsert groups gkey gstate) gstate)
		))
)))
(define rdf_apply_row_to_groups (lambda (groups cols group row) (begin
	(define gvals (map group (lambda (gexpr) (rdf_row_eval gexpr row))))
	(define gkey (json_encode gvals))
	(match (rdf_ensure_group groups cols gkey row) '(new_groups gstate) (begin
		(reduce_assoc cols (lambda (_ alias expr) (begin
			(if (rdf_has_aggregate expr)
				(gstate alias (rdf_agg_step expr (gstate alias) row))
				nil)
			nil
		)) nil)
		new_groups
	))
)))
(define rdf_emit_aggregated_rows (lambda (row_store cols group emit_row) (begin
	(define rows (map (row_store) (lambda (k) (row_store k))))
	(define initial_groups (if (and (equal? rows '()) (or (nil? group) (equal? group '())) (rdf_select_has_aggregates cols))
		(match (rdf_ensure_group '() cols "[]" nil) '(new_groups _gstate) new_groups)
		'()))
	(define groups (reduce rows (lambda (acc row) (rdf_apply_row_to_groups acc cols group row)) initial_groups))
	(map groups (lambda (entry) (match entry '(_gkey gstate) (begin
		(emit_row (reduce_assoc cols (lambda (acc alias expr)
			(append acc (concat alias) (if (rdf_has_aggregate expr) (rdf_agg_finalize expr (gstate alias)) (gstate alias)))
		) '()))
		nil
	))))
)))

(define rdf_template_expr (lambda (triples ctx)
	(cons (quote list) (map triples (lambda (triple) (match triple '(s p o)
		(list (quote list) (rdf_replace_ctx s ctx) (rdf_replace_ctx p ctx) (rdf_replace_ctx o ctx))
	))))
))
(define rdf_session_values (lambda (sess)
	(map (sess) (lambda (k) (sess k)))
))
(define rdf_session_merged_values (lambda (sess)
	(merge (rdf_session_values sess))
))
(define rdf_relation_targets (lambda (schema subj pred) (begin
	(define out (newsession))
	(scan schema "rdf" '("s" "p") (lambda (s p) (and (equal? s subj) (equal? p pred))) '("o") (lambda (o) (out o true)))
	(out)
)))
(define rdf_path_targets (lambda (schema start pred include_self) (begin
	(define seen (newsession))
	(define visit (lambda (node) (begin
		(if (seen node)
			nil
			(begin
				(seen node true)
				(map (rdf_relation_targets schema node pred) visit)
			))
		nil
	)))
	(if include_self
		(visit start)
		(map (rdf_relation_targets schema start pred) visit))
	(seen)
)))
(define rdf_ensure_table (lambda (schema)
	(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf (s TEXT, p TEXT, o TEXT)" (lambda (schema table write) true)))
))
(define rdf_insert_triples (lambda (schema triples)
	(if (equal? triples '())
		nil
		(insert schema "rdf" '("s" "p" "o") triples '() (lambda () true))
	)
))
(define rdf_delete_triples (lambda (schema triples) (begin
	(map triples (lambda (triple) (match triple '(subj pred obj)
		(scan schema "rdf" '("s" "p" "o") (lambda (s p o) (and (equal? s subj) (equal? p pred) (equal? o obj))) '("$update") (lambda ($update) ($update)))
	)))
	nil
)))

(define rdf_queryplan (lambda (schema query definitions ctx resultfunc /* function that gets cols + ctx */) (begin
	(match query '("select" cols "where" conditions "group" group "order" order "limit" limit "offset" offset "distinct" distinct) (begin
		/* ctx: array with predefined variables */
		/* no join reordering yet */
			(define rdf_path_subject_value_local (lambda (expr ctx) (match expr
				'('get_var var)
					(if (rdf_ctx_bound ctx var)
						(ctx var)
						(error "SPARQL error: property path subject must be bound"))
				(string? sval) sval
				(list? l) (eval l)
				(error "SPARQL error: unsupported property path subject " expr)
			)))
			(define rdf_path_target_plan_local (lambda (obj target_sym tail order ctx resultfunc2) (match obj
				'('get_var var)
					(if (rdf_ctx_bound ctx var)
						(list (quote if) (list (quote equal?) target_sym (ctx var)) (build_scan tail order ctx resultfunc2) nil)
						(build_scan tail order (merge ctx (merge (list (list var target_sym)))) resultfunc2))
				(string? oval)
					(list (quote if) (list (quote equal?) target_sym oval) (build_scan tail order ctx resultfunc2) nil)
				(list? l)
					(list (quote if) (list (quote equal?) target_sym (eval l)) (build_scan tail order ctx resultfunc2) nil)
				(error "SPARQL error: unsupported property path object " obj)
			)))
			(define build_scan (lambda (conditions order ctx resultfunc2)
				(match conditions
					(cons '("__union__" branches) tail)
						(cons (quote begin) (map branches (lambda (branch)
							(build_scan (merge (list branch tail)) order ctx resultfunc2)
						)))
					(cons '("__filter_exists__" negate exists_conditions) tail)
						(begin
							(define exists_hash (fnv_hash (concat exists_conditions "|" tail "|" negate)))
							(define exists_state (symbol (concat "__rdf_exists_" exists_hash)))
							(list (quote begin)
								(list (quote set) exists_state (list (quote newsession)))
								(list exists_state "matched" false)
								(build_scan exists_conditions '() ctx (lambda (_ _exists_ctx)
									(list exists_state "matched" true)))
								(list (quote if)
									(if negate
										(list (quote not) (list exists_state "matched"))
										(list exists_state "matched"))
									(build_scan tail order ctx resultfunc2)
									nil)))
					(cons '("__optional__" optional_conditions) tail)
						(begin
							(define optional_vars_all (rdf_condition_vars optional_conditions))
							/* A variable can be bound to nil by OPTIONAL. Presence in ctx determines scope;
							the bound expression itself may still evaluate to nil later in build_scan. */
							(define optional_new_vars (filter optional_vars_all (lambda (var) (not (has_assoc? ctx var)))))
							(define optional_ctx_nil (merge ctx (merge (map optional_new_vars (lambda (var) (list var (rdf_unbound_expr)))))))
							(define optional_hash (fnv_hash (concat optional_conditions "|" tail "|" optional_new_vars)))
							(define optional_state (symbol (concat "__rdf_optional_" optional_hash)))
							(list (quote begin)
								(list (quote set) optional_state (list (quote newsession)))
								(list optional_state "matched" false)
								(build_scan optional_conditions '() ctx (lambda (_ optional_ctx)
									(list (quote begin)
										(list optional_state "matched" true)
										(build_scan tail order optional_ctx resultfunc2))))
								(list (quote if) (list (quote not) (list optional_state "matched"))
									(build_scan tail order optional_ctx_nil resultfunc2)
									nil)))
					(cons '("__bind__" bind_expr bind_var_expr) tail)
						(build_scan tail order (merge ctx (merge (list (list (rdf_var_symbol bind_var_expr) (rdf_replace_ctx bind_expr ctx))))) resultfunc2)
					(cons '("__values__" var_expr vals) tail)
						(cons (quote begin) (map vals (lambda (val)
							(build_scan tail order (merge ctx (merge (list (list (rdf_var_symbol var_expr) (rdf_replace_ctx val ctx))))) resultfunc2)
						)))
					(cons '("__subquery__" subquery) tail)
						(match subquery
							'("select" subcols "where" _subconds "group" subgroup "order" _suborder "limit" _sublimit "offset" _suboffset "distinct" _subdistinct)
								(if (or (rdf_select_has_aggregates subcols) (not (equal? subgroup '())))
									(begin
										(define sub_rows (symbol (concat "__rdf_subquery_rows_" (uuid))))
										(define sub_rowvars (rdf_select_capture_vars subcols subgroup))
										(define sub_rowctx (map_assoc subcols (lambda (k _v)
											(list (quote rdf_row_lookup) (quote row) (concat k))
										)))
										(list (quote begin)
											(list (quote set) sub_rows (list (quote newsession)))
											(rdf_queryplan schema subquery definitions ctx (lambda (_cols inner_ctx)
												(list sub_rows (list (quote uuid)) (cons list (rdf_capture_row_items sub_rowvars inner_ctx)))
											))
											(list (quote rdf_emit_aggregated_rows)
												sub_rows
												(list (quote quote) subcols)
												(list (quote quote) subgroup)
												(list (quote lambda) (list (quote row))
													(build_scan tail order (merge ctx sub_rowctx) resultfunc2))))
									)
									(rdf_queryplan schema subquery definitions ctx (lambda (subcols subctx)
										(build_scan tail order (merge ctx (merge (map_assoc subcols (lambda (k v) (list k (rdf_replace_ctx v subctx)))))) resultfunc2)
									)))
							(rdf_queryplan schema subquery definitions ctx (lambda (subcols subctx)
								(build_scan tail order (merge ctx (merge (map_assoc subcols (lambda (k v) (list k (rdf_replace_ctx v subctx)))))) resultfunc2)
							)))
					(cons '(s p) tail)
						(if (equal? (concat s) "__filter__")
							(list (quote if) (rdf_replace_ctx p ctx) (build_scan tail order ctx resultfunc2))
							(error "SPARQL error: expected triple pattern (s p o), got 2 elements"))
					(cons '(s p o) tail)
						(match p
							'("__path_seq__" p1 p2)
								(begin
									(define tmp_var (concat "?__rdf_path_" (uuid)))
									(build_scan (cons (list s p1 (list (quote get_var) tmp_var)) (cons (list (list (quote get_var) tmp_var) p2 o) tail)) order ctx resultfunc2))
							'("__path_alt__" p1 p2)
								(list (quote begin)
									(build_scan (cons (list s p1 o) tail) order ctx resultfunc2)
									(build_scan (cons (list s p2 o) tail) order ctx resultfunc2))
							'("__path_star__" pred)
								(begin
									(define start_expr (rdf_path_subject_value_local s ctx))
									(define target_sym (symbol (concat "__rdf_path_target_" (uuid))))
									(list (quote map)
										(list (quote rdf_path_targets) schema start_expr pred true)
										(list (quote lambda) (list target_sym)
											(rdf_path_target_plan_local o target_sym tail order ctx resultfunc2))))
							'("__path_plus__" pred)
								(begin
									(define start_expr (rdf_path_subject_value_local s ctx))
									(define target_sym (symbol (concat "__rdf_path_target_" (uuid))))
									(list (quote map)
										(list (quote rdf_path_targets) schema start_expr pred false)
										(list (quote lambda) (list target_sym)
											(rdf_path_target_plan_local o target_sym tail order ctx resultfunc2))))
							_
								(begin
									(define process (lambda (v sym conditions vars) (match v
										'('get_var var)
											(if (rdf_ctx_bound ctx var)
												'((append conditions sym (ctx var)) vars)
												'(conditions (append vars sym (symbol var))))
										(string? s) '((append conditions sym s) vars)
										(list? l) '((append conditions sym (eval l)) vars)
										(error "SPARQL error: unsupported expression type in WHERE clause: " v)
									)))
									(match (process s "s" '() '()) '(conditions vars)
										(match (process p "p" conditions vars) '(conditions vars)
											(match (process o "o" conditions vars) '(conditions vars) (begin
												/* check if one of the orders matches (currently only raw-variable support) */
												/* TODO: for general expressions: two cases: s/p/o is bound to a variable and we bind against only variables from s/p/o: use scan_order; otherwise: collect all results in a list and use scm's sort */
												(set order_head (match order
													(cons '(expr dir) order_rest)
														(match expr (eval s) '("s" dir) (eval p) '("p" dir) (eval o) '("o" dir))
												))
												(set inner_ctx (merge ctx (merge (extract_assoc vars (lambda (k v) '(v (symbol v)))))))
												(set filter_cols (cons list (extract_assoc conditions (lambda (k v) k))))
												(set filter_fn '('lambda (extract_assoc conditions (lambda (k v) (symbol k))) (cons 'and (extract_assoc conditions (lambda (k v) '('equal? (symbol k) v))))))
												(set map_cols (cons list (extract_assoc vars (lambda (k v) k))))
												(set map_fn (list (quote lambda) (extract_assoc vars (lambda (k v) (symbol v))) (build_scan tail (if order_head order_rest order) inner_ctx resultfunc2)))
												(match order_head
													'(col dir)
														(list (quote scan_order) schema "rdf"
															filter_cols filter_fn
															(list (quote list) col) (list (quote list) (match dir "DESC" > <)) 0 0 -1
															map_cols map_fn (quote cons) nil)
													(list (quote scan) schema "rdf" filter_cols filter_fn map_cols map_fn)
												)
											)))
										))
								)
					'()
						(match order
							(cons _ _) (error (concat "order not consumed: " order))
							(resultfunc2 cols ctx))
				)
			))
			(build_scan conditions order ctx resultfunc)
		) (error "wrong rdf layout " query))
	)))

(define parse_sparql (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest) (begin
		(set cleaned_rest (rdf_strip_leading_ws_comments rest))
		(set parsed (rdf_query cleaned_rest))
		(set parsed (rdf_resolve_prefixes parsed definitions))
		(match parsed
			'("insert_data" triples)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(list (quote rdf_insert_triples) schema (list (quote quote) triples)))
			'("delete_data" triples)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(list (quote rdf_delete_triples) schema (list (quote quote) triples)))
			'("ask" "where" conditions) (begin
				(set _ask_state (newsession))
				(_ask_state "matched" false)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema '("select" '() "where" conditions "group" '() "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols _ctx)
						(list _ask_state "matched" true)))
					(list (quote resultrow) (list (quote list) "?ask" (list _ask_state "matched")))
				))
			'("construct" triples "where" conditions)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema '("select" '() "where" conditions "group" '() "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols ctx)
						(cons (quote begin) (map triples (lambda (triple) (match triple '(s p o)
							(list (quote resultrow) (list (quote list) (rdf_replace_ctx s ctx) (rdf_replace_ctx p ctx) (rdf_replace_ctx o ctx)))
						))))
					))
				)
			'("modify" "delete" delete_triples "insert" insert_triples "where" conditions) (begin
				(set _delete_rows (newsession))
				(set _insert_rows (newsession))
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema '("select" '() "where" conditions "group" '() "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols ctx)
						(list (quote begin)
							(list _delete_rows (list (quote uuid)) (rdf_template_expr delete_triples ctx))
							(list _insert_rows (list (quote uuid)) (rdf_template_expr insert_triples ctx))
						)
					))
					(list (quote rdf_delete_triples) schema (list (quote rdf_session_merged_values) _delete_rows))
					(list (quote rdf_insert_triples) schema (list (quote rdf_session_merged_values) _insert_rows))
				))
			'("select" cols "where" conditions "group" qgroup "order" qorder "limit" qlimit "offset" qoffset "distinct" qdistinct) (begin
				(set missing_select_vars (rdf_missing_select_vars cols conditions))
				(if (not (equal? missing_select_vars '()))
					(error "SPARQL error: unbound SELECT variable" missing_select_vars)
					nil
				)
				(set qhasagg (rdf_select_has_aggregates cols))
				(set qrowvars (rdf_select_capture_vars cols qgroup))
				(set needs_wrap (or (not (nil? qlimit)) (not (nil? qoffset)) (not (nil? qdistinct))))
				(set effective_offset (coalesce qoffset 0))
				(set effective_limit (coalesce qlimit 999999999))
				/* build resultfunc that includes limit/offset/distinct logic */
				(if (or qhasagg (not (equal? qgroup '())))
					(begin
						(set _rows (newsession))
						(list (quote begin)
							(rdf_queryplan schema parsed definitions '() (lambda (_cols ctx) (begin
								(set row_expr (cons list (rdf_capture_row_items qrowvars ctx)))
								(list _rows (list (quote uuid)) row_expr)
							)))
							(list (quote rdf_emit_aggregated_rows)
								_rows
								(list (quote quote) cols)
								(list (quote quote) qgroup)
								(list (quote lambda) (list (quote row))
									(list (quote resultrow) (quote row))))
						)
					)
					(begin
						(if needs_wrap
							(begin
								/* state session is created at compile time, initialized here, used at eval time */
								(set _st (newsession))
								(_st "cnt" 0)
								(if qdistinct (_st "seen" (newsession)))
								(rdf_queryplan schema parsed definitions '() (lambda (row_cols ctx)
									(rdf_wrapped_resultrow_ast _st (cons list (rdf_row_items row_cols ctx)) qdistinct effective_offset effective_limit)
								)))
							(rdf_queryplan schema parsed definitions '() rdf_select_resultrow_ast))
					))
				))
		)
	)
)))


(define rdf_apply_base_iri (lambda (definitions iri)
	(if (and (not (nil? (definitions ""))) (not (regexp_test iri "^[a-zA-Z][a-zA-Z0-9+.-]*:")))
		(concat (definitions "") iri)
		iri
	)
))
(define rdf_expand_ttl_object (lambda (subject pred obj) (match obj
	'("__ttl_inline_node__" bn facts) (cons (list subject pred bn) facts)
	_ (list (list subject pred obj))
)))


/* helper: parse TTL into list of (s p o) triples without loading */
(define parse_ttl_triples (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest)
		(begin
			(define ttl_simple_constant (parser (or
				(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x))
				(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post)))
				(parser '((atom "<" true) (define iri (regex "[^>]*" false false)) (atom ">" false false)) (rdf_apply_base_iri definitions iri))
				(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) x)
				(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x))
				(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) (rdf_unescape x))
				(regex "[a-zA-Z0-9_]+" true)
			)))
			(define ttl_object (parser (or
				(parser '(
					"["
					(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_simple_constant ",")) (? ";")) (map os (lambda (o) '(p o))))))
					"]"
				) (begin
					(define bn (concat "_:anon_" (uuid)))
					(list "__ttl_inline_node__" bn (merge (map ps (lambda (p) (map p (lambda (p1) (cons bn p1)))))))
				))
				(parser '(
					"("
					(define _items (* ttl_simple_constant))
					")"
				) (concat "_:list_" (uuid)))
				ttl_simple_constant
			)))
			(define ttl_fact (parser '(
				(define facts
					(parser '(
						(define s ttl_simple_constant)
						(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_object ",")) (? ";")) (map os (lambda (o) '(p o))))))
						"."
					) (merge (map ps (lambda (p) (merge (map p (lambda (p1) (match p1 '(pred obj) (rdf_expand_ttl_object s pred obj)))))))))
				)
				(define rest rest)
			) '("facts" facts "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))
		(set _pt (newsession))
		(_pt "triples" '())
		(define process_fact (lambda (rest) (match (ttl_fact rest)
			'("facts" facts "rest" (regex "^[ \\n\\r\\t]*$" _)) (_pt "triples" (merge (_pt "triples") facts))
			'("facts" facts "rest" rest) (!begin (_pt "triples" (merge (_pt "triples") facts)) (process_fact rest))
			rest (error "couldnt parse: " rest)
		)))
		(process_fact rest)
		(_pt "triples")
	)
)))

/* delete triples from the store that match the given TTL */
(define delete_ttl (lambda (schema s) (begin
	(set triples (parse_ttl_triples schema s))
	(map triples (lambda (triple) (match triple '(subj pred obj)
		(scan schema "rdf" '("s" "p" "o") (lambda (s p o) (and (equal? s subj) (equal? p pred) (equal? o obj))) '("$update") (lambda ($update) ($update)))
	)))
)))

(define load_ttl (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest)
		(begin
			/* blank node registry: maps _:id to urn:uuid:... per load */
			(set _bn (newsession))
		(define resolve_blank (lambda (val)
			(if (nil? val) val
				(match val (regex "^_:(.+)$" _ bname) (begin
					(if (nil? (_bn bname)) (_bn bname (concat "urn:uuid:" (uuid))))
					(_bn bname)
				) val)
			)
		))
			(define ttl_simple_constant (parser (or
				(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node before prefix match */
				(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post))) /* add prefix with validation */
				(parser '((atom "<" true) (define iri (regex "[^>]*" false false)) (atom ">" false false)) (rdf_apply_base_iri definitions iri))
				(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) x)
				(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x))
				(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) (rdf_unescape x))
				(regex "[a-zA-Z0-9_]+" true)
			)))
			(define ttl_object (parser (or
				(parser '(
					"["
					(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_simple_constant ",")) (? ";")) (map os (lambda (o) '(p o))))))
					"]"
				) (begin
					(define bn (concat "_:anon_" (uuid)))
					(list "__ttl_inline_node__" bn (merge (map ps (lambda (p) (map p (lambda (p1) (cons bn p1)))))))
				))
				(parser '(
					"("
					(define _items (* ttl_simple_constant))
					")"
				) (concat "_:list_" (uuid)))
				ttl_simple_constant
			)))
			(define ttl_fact (parser '(
				(define facts
					(parser '(
						(define s ttl_simple_constant)
						(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_object ",")) (? ";")) (map os (lambda (o) '(p o))))))
						"."
					) (merge (map ps (lambda (p) (merge (map p (lambda (p1) (match p1 '(pred obj) (rdf_expand_ttl_object s pred obj)))))))))
				)
				(define rest rest)
			) '("facts" facts "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))
		(set load (lambda (facts) (begin
			/* resolve blank nodes to UUIDs and insert */
			(insert schema "rdf" '("s" "p" "o") (map facts (lambda (triple) (list (resolve_blank (car triple)) (resolve_blank (car (cdr triple))) (resolve_blank (car (cdr (cdr triple))))))) '() (lambda () true))
		)))
		(define process_fact (lambda (rest) (match (ttl_fact rest)
			'("facts" facts "rest" (regex "^[ \\n\\r\\t]*$" _)) (load facts)
			'("facts" facts "rest" rest) (!begin (load facts) (process_fact rest))
			rest (error "couldnt parse: " rest)
		)))
		(process_fact rest)
	)
)))
