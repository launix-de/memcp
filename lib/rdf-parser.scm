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
(define rdf_json_objectagg_reduce (lambda (a b)
	(if (nil? a) b (if (nil? b) a (json_merge_patch a b)))
))
(define rdf_json_arrayagg_reduce (lambda (a b)
	(if (nil? a) b (if (nil? b) a (merge a b)))
))
(define rdf_ordered_json_arrayagg_finalize (lambda (values descending)
	(if (nil? values)
		(json_arrayagg_finalize nil)
		(json_arrayagg_finalize (map (sort values (lambda (left right)
			(if descending (> (car left) (car right)) (< (car left) (car right))))) cadr)))
))
(define rdf_ordered_json_arrayagg_finalize_asc (lambda (values)
	(rdf_ordered_json_arrayagg_finalize values false)
))
(define rdf_ordered_json_arrayagg_finalize_desc (lambda (values)
	(rdf_ordered_json_arrayagg_finalize values true)
))
(define rdf_is_iri (lambda (value)
	(and (string? value) (regexp_test value "^[a-zA-Z][a-zA-Z0-9+.-]*:"))
))
(define rdf_is_blank (lambda (value)
	(and (string? value) (regexp_test value "^(?:_:|urn:uuid:)"))
))
(define rdf_is_literal (lambda (value)
	(and (not (nil? value)) (not (rdf_is_iri value)))
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
	(parser '((atom "JSON_ARRAYAGG" true) "(" (define e rdf_filter_or)
		(atom "ORDER" true) (atom "BY" true) (define key rdf_filter_or)
		(? (define dir (or (atom "DESC" true) (atom "ASC" true)))) ")")
		(list "__rdf_agg__" "JSON_ARRAYAGG_ORDERED" e (list key (coalesce dir "ASC"))))
	(parser '((atom "JSON_ARRAYAGG" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "JSON_ARRAYAGG" e nil))
	(parser '((atom "JSON_OBJECTAGG" true) "(" (define key rdf_filter_or) "," (define value rdf_filter_or) ")") '("__rdf_agg__" "JSON_OBJECTAGG" '('json_objectagg_entry key value) nil))
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
	(parser '((atom "UCASE" true) "(" (define a rdf_filter_or) ")") '('toUpper a))
	(parser '((atom "LCASE" true) "(" (define a rdf_filter_or) ")") '('toLower a))
	(parser '((atom "SUBSTR" true) "(" (define a rdf_filter_or) "," (define start rdf_filter_or) "," (define len rdf_filter_or) ")") '('sql_substr a start len))
	(parser '((atom "SUBSTR" true) "(" (define a rdf_filter_or) "," (define start rdf_filter_or) ")") '('sql_substr a start))
	(parser '((atom "REPLACE" true) "(" (define a rdf_filter_or) "," (define from rdf_filter_or) "," (define to rdf_filter_or) ")") '('replace a from to))
	(parser '((atom "ABS" true) "(" (define a rdf_filter_or) ")") '('sql_abs a))
	(parser '((atom "ROUND" true) "(" (define a rdf_filter_or) ")") '('round a))
	(parser '((atom "CEIL" true) "(" (define a rdf_filter_or) ")") '('ceil a))
	(parser '((atom "FLOOR" true) "(" (define a rdf_filter_or) ")") '('floor a))
	(parser '((atom "sameTerm" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('equal?? a b))
	(parser '((atom "isIRI" true) "(" (define a rdf_filter_or) ")") '('rdf_is_iri a))
	(parser '((atom "isURI" true) "(" (define a rdf_filter_or) ")") '('rdf_is_iri a))
	(parser '((atom "isBlank" true) "(" (define a rdf_filter_or) ")") '('rdf_is_blank a))
	(parser '((atom "isLiteral" true) "(" (define a rdf_filter_or) ")") '('rdf_is_literal a))
	(parser '((atom "STRUUID" true) "(" ")") '('concat '('uuid)))
	(parser '((atom "UUID" true) "(" ")") '('concat "urn:uuid:" '('uuid)))
	(parser '((atom "COALESCE" true) "(" (define args (+ rdf_filter_or ",")) ")") (cons (quote coalesceNil) args))
	(parser '((atom "IF" true) "(" (define cond rdf_filter_or) "," (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('if cond a b))
	rdf_aggregate_expression
	/* MemCP extension: expose the SQL JSON scalar-function registry to SPARQL.
	This intentionally accepts only JSON_* names; the emitted expression is the
	same common IR node produced by the SQL frontend. */
	(parser '((define fn (regex "JSON_[a-zA-Z0-9_]+" true)) "(" (define args (* rdf_filter_or ",")) ")")
		(begin
			(define builtin (sql_builtins (toUpper fn)))
			(if (nil? builtin)
				(error "unknown JSON function " fn)
				(cons builtin args))))
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
(define rdf_where_graph_item (parser '(
	(atom "GRAPH" true)
	(define graph rdf_expression)
	(atom "{" true)
	(define conditions rdf_where_inner_basic_items)
	(atom "}" true)
) (list (list "__graph__" graph (merge (coalesce conditions '('())))))))
(define rdf_subquery_select_col (parser (or
	(parser '("(" (define v rdf_aggregate_expression) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '("(" (define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '((define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v)))
	(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v)))
)))
(define rdf_where_subquery_item (parser '(
	(atom "{" true)
	(atom "SELECT" true)
	(? (define distinct (atom "DISTINCT" true)))
	(define cols (+ (parser '((define col rdf_subquery_select_col) (? (atom "," true))) col)))
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '(
		(define item (or
			rdf_where_subquery_item
			rdf_where_filter_exists_item
			rdf_where_values_item
			rdf_where_graph_item
			rdf_where_optional_item
			rdf_where_bind_item
			rdf_where_basic_item))
		(? (atom "." true))
	) item)))
	(atom "}" true)
	(? (atom "GROUP" true) (atom "BY" true)
		(define group (+ (parser '((define var rdf_variable) (? (atom "," true))) var))))
	(? (atom "HAVING" true) "(" (define having rdf_filter_or) ")")
	(? (atom "ORDER" true) (atom "BY" true)
		(define ordercols (+ (parser (define expr rdf_expression) '(expr "ASC")))))
	(? (atom "LIMIT" true) (define limit (parser (define n (regex "[0-9]+" true)) (simplify n))))
	(? (atom "OFFSET" true) (define offset (parser (define n (regex "[0-9]+" true)) (simplify n))))
	(atom "}" true)
) (list (list "__subquery__"
		(list "select" (merge cols) "where" (merge (coalesce conditions '()))
			"group" (coalesce group '()) "having" having "order" ordercols
			"limit" limit "offset" offset "distinct" distinct)))))
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
	rdf_where_graph_item
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
	(parser '("(" (define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '((define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v)))
	(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v)))
)))

(define rdf_number (parser (define x (regex "[0-9]+" true)) (simplify x)))
(define rdf_order_condition (parser (not
	(or
		(parser '((define dir (or (atom "DESC" true) (atom "ASC" true))) "(" (define expr rdf_expression) ")") '(expr dir))
		(parser (define expr rdf_expression) '(expr "ASC")))
	(atom "LIMIT" true)
	(atom "OFFSET" true)
)))
(define rdf_limit_offset (parser (or
	(parser '((atom "LIMIT" true) (define limit rdf_number) (? (atom "OFFSET" true) (define offset rdf_number))) '(limit offset))
	(parser '((atom "OFFSET" true) (define offset rdf_number) (? (atom "LIMIT" true) (define limit rdf_number))) '(limit offset))
)))
(define rdf_dataset_restrict_named_condition (lambda (condition named_graphs)
	(match condition
		'("__graph__" graph inner)
		(match graph
			'('get_var var)
			(list "__graph_restricted__" graph inner named_graphs)
			_ (if (rdf_key_in_list named_graphs graph)
				condition (list "__filter__" false)))
		'("__optional__" inner)
		(list "__optional__" (rdf_dataset_restrict_named inner named_graphs))
		'("__union__" branches)
		(list "__union__" (map branches (lambda (branch)
			(rdf_dataset_restrict_named branch named_graphs))))
		'("__filter_exists__" negate inner)
		(list "__filter_exists__" negate (rdf_dataset_restrict_named inner named_graphs))
		'("__subquery__" subquery)
		(match subquery
			'("select" cols "where" inner "group" group "having" having "order" order "limit" limit "offset" offset "distinct" distinct)
			(list "__subquery__" (list "select" cols "where"
				(rdf_dataset_restrict_named inner named_graphs) "group" group
				"having" having "order" order "limit" limit "offset" offset
				"distinct" distinct))
			_ condition)
		_ condition)
))
(define rdf_dataset_restrict_named (lambda (conditions named_graphs)
	(map conditions (lambda (condition)
		(rdf_dataset_restrict_named_condition condition named_graphs)))
))
(define rdf_dataset_default_condition (lambda (condition default_graphs)
	(match condition
		'("__graph__" _graph _inner) condition
		'("__graph_restricted__" _graph _inner _graphs) condition
		'("__optional__" inner)
		(list "__optional__" (rdf_dataset_default_conditions inner default_graphs))
		'("__union__" branches)
		(list "__union__" (map branches (lambda (branch)
			(rdf_dataset_default_conditions branch default_graphs))))
		'("__filter_exists__" negate inner)
		(list "__filter_exists__" negate (rdf_dataset_default_conditions inner default_graphs))
		'("__subquery__" subquery)
		(match subquery
			'("select" cols "where" inner "group" group "having" having "order" order "limit" limit "offset" offset "distinct" distinct)
			(list "__subquery__" (list "select" cols "where"
				(rdf_dataset_default_conditions inner default_graphs) "group" group
				"having" having "order" order "limit" limit "offset" offset
				"distinct" distinct))
			_ condition)
		'("__bind__" _expr _var) condition
		'("__values__" _var _values) condition
		'(s p o)
		(if (equal? default_graphs '())
			(list "__empty_pattern__" condition)
			(if (equal? (count default_graphs) 1)
				(list "__graph__" (car default_graphs) (list condition))
				(list "__union_distinct__" (map default_graphs (lambda (graph)
					(list (list "__graph__" graph (list condition))))))))
		_ condition)
))
(define rdf_dataset_default_conditions (lambda (conditions default_graphs)
	(map conditions (lambda (condition)
		(rdf_dataset_default_condition condition default_graphs)))
))
(define rdf_dataset_conditions (lambda (datasets conditions)
	(if (equal? datasets '())
		conditions
		(begin
			(define default_graphs (map (filter datasets (lambda (entry)
				(equal? (car entry) "default"))) cadr))
			(define named_graphs (map (filter datasets (lambda (entry)
				(equal? (car entry) "named"))) cadr))
			(rdf_dataset_default_conditions
				(rdf_dataset_restrict_named conditions named_graphs) default_graphs)))
))
(define rdf_dataset_clause (parser '(
	(atom "FROM" true)
	(? (define named (atom "NAMED" true)))
	(define graph rdf_expression)
) (list (if named "named" "default") graph)))
(define rdf_select (parser '(
	(atom "SELECT" true)
	(? (define distinct (atom "DISTINCT" true)))
	(define cols (or
		(parser (atom "*" true) "__select_all__")
		(+ (parser '((define col rdf_select_col) (? (atom "," true))) col))))
	(define datasets (* rdf_dataset_clause))
	(?
		(atom "WHERE" true)
		(atom "{" true)
		(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
		(atom "}" true) /* TODO: {} UNION {} */
	)
	(?
		(atom "GROUP" true)
		(atom "BY" true)
		(define group (+ (parser '((define var rdf_variable) (? (atom "," true))) var)))
	)
	(?
		(atom "HAVING" true)
		"("
		(define having rdf_filter_or)
		")"
	)
	(?
		(atom "ORDER" true)
		(atom "BY" true)
		(define ordercols (+ rdf_order_condition))
	)
	(define slice (or rdf_limit_offset (parser empty '(nil nil))))
) (list "select" (if (equal? cols "__select_all__") cols (merge cols)) "where"
		(rdf_dataset_conditions datasets (merge (coalesce conditions '())))
		"group" (coalesce group '()) "having" having "order" ordercols
		"limit" (car slice) "offset" (cadr slice) "distinct" distinct)
	"^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

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
(define rdf_insert_graph_data (parser '(
	(atom "INSERT" true)
	(atom "DATA" true)
	(atom "{" true)
	(atom "GRAPH" true)
	(define graph rdf_expression)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
	(atom "}" true)
) '("insert_graph_data" graph (merge (coalesce triples '('()))))))
(define rdf_delete_data (parser '(
	(atom "DELETE" true)
	(atom "DATA" true)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
) '("delete_data" (merge (coalesce triples '('()))))))
(define rdf_delete_graph_data (parser '(
	(atom "DELETE" true)
	(atom "DATA" true)
	(atom "{" true)
	(atom "GRAPH" true)
	(define graph rdf_expression)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
	(atom "}" true)
) '("delete_graph_data" graph (merge (coalesce triples '('()))))))
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
(define rdf_create_graph (parser '(
	(atom "CREATE" true) (? (atom "SILENT" true)) (atom "GRAPH" true)
	(define graph rdf_expression)
) '("create_graph" graph)))
(define rdf_clear_graph (parser '(
	(atom "CLEAR" true) (? (atom "SILENT" true)) (atom "GRAPH" true)
	(define graph rdf_expression)
) '("clear_graph" graph)))
(define rdf_drop_graph (parser '(
	(atom "DROP" true) (? (atom "SILENT" true)) (atom "GRAPH" true)
	(define graph rdf_expression)
) '("drop_graph" graph)))
(define rdf_query (parser (or
	rdf_create_graph
	rdf_clear_graph
	rdf_drop_graph
	rdf_delete_insert_where
	rdf_insert_graph_data
	rdf_delete_graph_data
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
			(parser '((atom "PREFIX" true) (define pfx (regex "[a-zA-Z0-9_]*" false)) (atom ":" false false) (define content rdf_constant)) '(pfx content))
			(parser '((atom "BASE" true) (define content rdf_constant)) '("" content))
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
		'("__union_distinct__" branches)
		(reduce branches (lambda (acc branch)
			(merge_unique (list acc (rdf_condition_vars branch)))
		) '())
		'("__optional__" inner) (rdf_condition_vars inner)
		'("__graph__" graph inner) (merge_unique (list (rdf_extract_vars graph) (rdf_condition_vars inner)))
		'("__graph_restricted__" graph inner _graphs)
		(merge_unique (list (rdf_extract_vars graph) (rdf_condition_vars inner)))
		'("__bind__" expr var_expr) (merge_unique (list (rdf_extract_vars expr) (list (rdf_var_symbol var_expr))))
		'("__values__" var_expr _vals) (list (rdf_var_symbol var_expr))
		'("__empty_pattern__" triple) (rdf_condition_vars (list triple))
		'("__subquery__" subquery)
		(match subquery
			'("select" subcols "where" subconds "group" subgroup "having" subhaving "order" suborder "limit" sublimit "offset" suboffset "distinct" subdistinct)
			(reduce_assoc subcols (lambda (acc alias expr) (append acc alias)) '())
			'()
		)
		'(s p o) (merge_unique (list (rdf_extract_vars s) (rdf_extract_vars p) (rdf_extract_vars o)))
		'()
))))))
(define rdf_expand_select_star (lambda (query)
	(match query
		'("select" "__select_all__" "where" conditions "group" group "having" having "order" order "limit" limit "offset" offset "distinct" distinct)
		(begin
			/* Build each association independently. The optimizer may reuse a reduce
			accumulator list, which would alias successive SELECT-* expressions. */
			(define cols (merge (map (rdf_condition_vars conditions) (lambda (var)
				(list (concat var) (list (quote get_var) var))))))
			(list "select" cols "where" conditions "group" group "having" having
				"order" order "limit" limit "offset" offset "distinct" distinct))
		_ query)
))
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
(define rdf_shared_result_items (lambda (cols ctx) (match cols
	(cons title (cons _expr tail))
	(cons (concat title) (cons (rdf_ctx_value ctx title) (rdf_shared_result_items tail ctx)))
	'()
)))
(define rdf_shared_resultrow_ast (lambda (row_cols ctx)
	(list (quote resultrow) (cons list (rdf_shared_result_items row_cols ctx)))
))
(define rdf_row_missing (lambda () '("__rdf_row_missing__")))
(define rdf_row_lookup (lambda (row sym) (match row
	(cons (cons key (cons val '())) tail)
	(if (rdf_key_equal key sym) val (rdf_row_lookup tail sym))
	(cons key (cons val tail))
	(if (rdf_key_equal key sym) val (rdf_row_lookup tail sym))
	'() (rdf_row_missing)
)))
(define rdf_has_aggregate (lambda (expr) (match expr
	'("__rdf_agg__" _ _ _) true
	(cons head tail) (or (rdf_has_aggregate head) (reduce tail (lambda (acc item) (or acc (rdf_has_aggregate item))) false))
	false
)))
(define rdf_select_has_aggregates (lambda (cols)
	(reduce_assoc cols (lambda (acc _ expr) (or acc (rdf_has_aggregate expr))) false)
))
(define rdf_numeric_value (lambda (value)
	(if (number? value) value (simplify (concat value)))
))
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
	(scan nil (table schema "rdf") (list 369436443803648 (scan_boundary "equal" "p" 0 0 true true "" false) (scan_boundary "equal" "s" 1 1 true true "" false)) (list pred subj) '() (lambda () true) '("o") (lambda (acc o) (begin (out o true) acc)))
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
	(begin
		(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf (s TEXT, p TEXT, o TEXT, UNIQUE KEY rdf_spo (s, p, o))" (lambda (schema tblname write) true)))
		(define info (show schema "rdf" true))
		(define unique_keys ((info "meta") "Unique"))
		(define has_spo (find unique_keys (lambda (key)
			(and (equal? (count (key "Cols")) 3)
				(and (rdf_key_in_list (key "Cols") "s")
					(and (rdf_key_in_list (key "Cols") "p")
						(rdf_key_in_list (key "Cols") "o")))))))
		(if has_spo true
			(begin
				/* Old RDF tables predate set semantics. Remove only duplicate physical
				rows before installing the key; the first occurrence remains untouched. */
				(define seen (newsession))
				(scan nil (table schema "rdf") '() '() '() (lambda () true)
					'("s" "p" "o" "$update")
					(lambda (acc s p o $update) (begin
						(define identity (json_encode (list s p o)))
						(if (seen identity) ($update) (seen identity true))
						acc)))
				(createkey (table schema "rdf") "rdf_spo" true '("s" "p" "o"))))
		true)
))
(define rdf_ensure_named_table (lambda (schema)
	(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf_named (g TEXT, s TEXT, p TEXT, o TEXT, UNIQUE KEY rdf_gspo (g, s, p, o))" (lambda (schema tblname write) true)))
))
(define rdf_insert_triples (lambda (schema triples)
	(if (equal? triples '())
		nil
		(insert (table schema "rdf") '("s" "p" "o") triples '() (lambda () true))
	)
))
(define rdf_delete_triples (lambda (schema triples) (begin
	(map triples (lambda (triple) (match triple '(subj pred obj)
		(scan nil (table schema "rdf") (list 369436712239104 (scan_boundary "equal" "o" 0 0 true true "" false) (scan_boundary "equal" "p" 1 1 true true "" false) (scan_boundary "equal" "s" 2 2 true true "" false)) (list obj pred subj) '() (lambda () true) '("$update") (lambda (acc $update) (begin ($update) acc)))
	)))
	nil
)))
(define rdf_insert_graph_triples (lambda (schema graph triples)
	(begin
		(rdf_ensure_named_table schema)
		(if (equal? triples '()) nil
			(insert (table schema "rdf_named") '("g" "s" "p" "o")
				(map triples (lambda (triple) (cons graph triple))) '() (lambda () true))))
))
(define rdf_delete_graph_triples (lambda (schema graph triples)
	(begin
		(rdf_ensure_named_table schema)
		(map triples (lambda (triple) (match triple '(subj pred obj)
			(scan nil (table schema "rdf_named")
				(list 369436980674560
					(scan_boundary "equal" "o" 0 0 true true "" false)
					(scan_boundary "equal" "p" 1 1 true true "" false)
					(scan_boundary "equal" "s" 2 2 true true "" false)
					(scan_boundary "equal" "g" 3 3 true true "" false))
				(list obj pred subj graph) '() (lambda () true) '("$update")
				(lambda (acc $update) (begin ($update) acc))))))
		nil)
))
(define rdf_clear_graph_data (lambda (schema graph)
	(begin
		(rdf_ensure_named_table schema)
		(scan nil (table schema "rdf_named")
			(list 369436175368192 (scan_boundary "equal" "g" 0 0 true true "" false))
			(list graph) '() (lambda () true) '("$update")
			(lambda (acc $update) (begin ($update) acc)))
		nil)
))

/* Basic graph patterns are ordinary self-joins over rdf(s, p, o). Lower them
to the same neutral query-block consumed by the SQL frontend so decorrelation,
join reordering, RecSet selection, and physical scan costing have one owner. */
(define rdf_shared_column (lambda (alias column)
	(list (quote get_column) alias false column false)
))
(define rdf_shared_lookup (lambda (bindings var)
	(match (rdf_ctx_lookup bindings var) '(found value)
		(if found value (error "SPARQL error: unbound shared-planner variable " var))
	)
))
(define rdf_shared_bind_term (lambda (term column bindings filters outer_ctx)
	(match term
		'('get_var var)
		(match (rdf_ctx_lookup outer_ctx var) '(outer_found outer_value)
			(if outer_found
				(list bindings (cons (list (quote equal??) column outer_value) filters))
				(match (rdf_ctx_lookup bindings var) '(found value)
					(if found
						(list bindings (cons (list (quote equal??) column value) filters))
						(list (append bindings var column) filters)))))
		(string? value) (list bindings (cons (list (quote equal??) column value) filters))
		(number? value) (list bindings (cons (list (quote equal??) column value) filters))
		(error "SPARQL shared planner: unsupported triple term " term)
	)
))
(define rdf_shared_add_pattern (lambda (pattern alias bindings filters outer_ctx)
	(match pattern '(s p o)
		(match (rdf_shared_bind_term s (rdf_shared_column alias "s") bindings filters outer_ctx) '(b1 f1)
			(match (rdf_shared_bind_term p (rdf_shared_column alias "p") b1 f1 outer_ctx) '(b2 f2)
				(rdf_shared_bind_term o (rdf_shared_column alias "o") b2 f2 outer_ctx)))
	)
))
(define rdf_shared_build_sources (lambda (schema patterns outer_ctx index sources bindings filters)
	(match patterns
		(cons pattern tail)
		(begin
			(define alias (concat "__rdf_t" index))
			(match (rdf_shared_add_pattern pattern alias bindings filters outer_ctx) '(next_bindings next_filters)
				(rdf_shared_build_sources schema tail outer_ctx (+ index 1)
					(append sources (list alias schema "rdf" false nil))
					next_bindings next_filters)))
		'() (list sources bindings filters)
	)
))
(define rdf_shared_add_named_pattern (lambda (pattern graph alias bindings filters outer_ctx)
	(match (rdf_shared_bind_term graph (rdf_shared_column alias "g") bindings filters outer_ctx) '(gb gf)
		(match pattern '(s p o)
			(match (rdf_shared_bind_term s (rdf_shared_column alias "s") gb gf outer_ctx) '(b1 f1)
				(match (rdf_shared_bind_term p (rdf_shared_column alias "p") b1 f1 outer_ctx) '(b2 f2)
					(rdf_shared_bind_term o (rdf_shared_column alias "o") b2 f2 outer_ctx)))))
))
(define rdf_shared_build_named_sources (lambda (schema graph patterns outer_ctx index sources bindings filters)
	(match patterns
		(cons pattern tail)
		(begin
			(define alias (concat "__rdf_g" index))
			(match (rdf_shared_add_named_pattern pattern graph alias bindings filters outer_ctx) '(next_bindings next_filters)
				(rdf_shared_build_named_sources schema graph tail outer_ctx (+ index 1)
					(append sources (list alias schema "rdf_named" false nil))
					next_bindings next_filters)))
		'() (list sources bindings filters))
))
(define rdf_shared_filter_condition? (lambda (condition)
	(match condition
		'("__filter__" _expr) true
		_ false
	)
))
(define rdf_shared_triple_conditions (lambda (conditions)
	(filter conditions (lambda (condition) (not (rdf_shared_filter_condition? condition))))
))
(define rdf_shared_filter_conditions (lambda (conditions bindings outer_ctx)
	(map (filter conditions rdf_shared_filter_condition?) (lambda (condition)
		(match condition '("__filter__" expr) (rdf_shared_expr expr bindings outer_ctx))))
))
(define rdf_shared_expr (lambda (expr bindings outer_ctx)
	(match expr
		'('rdf_bound ('get_var var))
		(list (quote not) (list (quote nil?) (rdf_shared_expr (list (quote get_var) var) bindings outer_ctx)))
		'((quote rdf_bound) ((quote get_var) var))
		(list (quote not) (list (quote nil?) (rdf_shared_expr (list (quote get_var) var) bindings outer_ctx)))
		'("__rdf_agg__" "COUNT" inner _)
		(list (quote aggregate)
			(list (quote if) (list (quote nil?) (rdf_shared_expr inner bindings outer_ctx)) 0 1)
			(quote +) 0)
		'("__rdf_agg__" "SUM" inner _)
		(list (quote aggregate)
			(list (quote rdf_numeric_value) (rdf_shared_expr inner bindings outer_ctx))
			(quote sql_sum_reduce) nil)
		'("__rdf_agg__" "AVG" inner _)
		(sql_avg_expr
			(list (quote rdf_numeric_value) (rdf_shared_expr inner bindings outer_ctx))
			(sql_aggregates "SUM") (sql_aggregates "COUNT"))
		'("__rdf_agg__" "MIN" inner _)
		(list (quote aggregate) (rdf_shared_expr inner bindings outer_ctx) (quote min) nil)
		'("__rdf_agg__" "MAX" inner _)
		(list (quote aggregate) (rdf_shared_expr inner bindings outer_ctx) (quote max) nil)
		'("__rdf_agg__" "GROUP_CONCAT" inner sep)
		(list (quote aggregate)
			(list (quote concat) (rdf_shared_expr inner bindings outer_ctx))
			(list (quote lambda) (list (quote a) (quote b))
				(list (quote if) (list (quote nil?) (quote a)) (quote b)
					(list (quote concat) (quote a) sep (quote b)))) nil)
		'("__rdf_agg__" "JSON_ARRAYAGG" inner _)
		(rdf_shared_expr (json_arrayagg_expr inner false) bindings outer_ctx)
		'("__rdf_agg__" "JSON_ARRAYAGG_ORDERED" inner order_spec)
		(list (quote aggregate)
			(list (quote list)
				(list (quote list)
					(rdf_shared_expr (car order_spec) bindings outer_ctx)
					(rdf_shared_expr inner bindings outer_ctx)))
			(quote rdf_json_arrayagg_reduce) nil
			(if (equal? (cadr order_spec) "DESC")
				(quote rdf_ordered_json_arrayagg_finalize_desc)
				(quote rdf_ordered_json_arrayagg_finalize_asc)))
		'("__rdf_agg__" "JSON_OBJECTAGG" ((quote json_objectagg_entry) key value) _)
		(list (quote aggregate)
			(list (quote json_object)
				(rdf_shared_expr key bindings outer_ctx)
				(rdf_shared_expr value bindings outer_ctx))
			(quote rdf_json_objectagg_reduce) nil)
		'('get_var var)
		(match (rdf_ctx_lookup outer_ctx var) '(outer_found outer_value)
			(if outer_found outer_value (rdf_shared_lookup bindings var)))
		(cons head tail) (cons (if (equal? head (quote equal?)) (quote equal??) head)
			(map tail (lambda (item) (rdf_shared_expr item bindings outer_ctx))))
		expr
	)
))
(define rdf_shared_where (lambda (filters)
	(match filters
		'() true
		(cons only '()) only
		_ (cons (quote and) filters)
	)
))
(define rdf_shared_order (lambda (order bindings outer_ctx)
	(if (nil? order)
		nil
		(map order (lambda (entry) (match entry '(expr dir)
			(list (rdf_shared_expr expr bindings outer_ctx) (if (equal? dir "DESC") > <)))))
	)
))
(define rdf_shared_complete_fields (lambda (fields bindings)
	(match bindings
		(cons var (cons value tail))
		(match (rdf_ctx_lookup fields var) '(found _existing)
			(rdf_shared_complete_fields
				(if found fields (append fields (concat var) value)) tail))
		'() fields
	)
))
(define rdf_shared_input_field_name (lambda (var)
	(concat "rdf_" (replace (concat var) "?" ""))
))
(define rdf_shared_direct_pattern? (lambda (condition)
	(match condition
		'("__filter__" _expr) false
		'("__bind__" _expr _var) false
		'("__filter_exists__" _negate _inner) false
		'("__values__" _var _values) false
		'("__optional__" _inner) false
		'("__union__" _branches) false
		'("__union_distinct__" _branches) false
		'("__empty_pattern__" _triple) false
		'("__subquery__" _query) false
		'("__graph__" _graph _inner) false
		'("__graph_restricted__" _graph _inner _graphs) false
		'(s p o) (match p
			'("__path_seq__" _ _) false
			'("__path_alt__" _ _) false
			'("__path_star__" _) false
			'("__path_plus__" _) false
			_ true)
		_ false
	)
))
(define rdf_shared_graph_relation (lambda (schema graph raw_conditions outer_ctx)
	(begin
		(define conditions (rdf_shared_expand_paths raw_conditions))
		(define patterns (filter conditions rdf_shared_direct_pattern?))
		(define source_index (fnv_hash (concat graph "|" conditions "|" outer_ctx)))
		(match (rdf_shared_build_named_sources schema graph patterns outer_ctx source_index '() '() '()) '(sources bindings filters)
			(begin
				(define filter_exprs (rdf_shared_filter_conditions conditions bindings outer_ctx))
				(define state (list sources bindings (merge (list filters filter_exprs)) (+ source_index (count patterns))))
				(list (rdf_shared_relation_query schema state) (rdf_shared_relation_vars bindings)))))
))
(define rdf_shared_expand_paths (lambda (conditions) (match conditions
	(cons condition tail)
	(match condition
		'("__empty_pattern__" triple)
		(rdf_shared_expand_paths (cons triple
			(cons (list "__filter__" false) tail)))
		'(s p o)
		(match p
			'("__path_seq__" p1 p2)
			(begin
				(define intermediate (list (quote get_var) (symbol (concat "?__rdf_path_" (uuid)))))
				(rdf_shared_expand_paths (cons (list s p1 intermediate) (cons (list intermediate p2 o) tail))))
			'("__path_alt__" p1 p2)
			(cons (list "__union__" (list (list (list s p1 o)) (list (list s p2 o))))
				(rdf_shared_expand_paths tail))
			_ (cons condition (rdf_shared_expand_paths tail)))
		_ (cons condition (rdf_shared_expand_paths tail)))
	'() '()
)))
(define rdf_shared_state_sources (lambda (state) (nth state 0)))
(define rdf_shared_state_bindings (lambda (state) (nth state 1)))
(define rdf_shared_state_filters (lambda (state) (nth state 2)))
(define rdf_shared_state_index (lambda (state) (nth state 3)))
(define rdf_shared_relation_fields (lambda (bindings)
	(reduce_assoc bindings (lambda (acc var expr)
		(append acc (rdf_shared_input_field_name var) expr)) '())
))
(define rdf_shared_relation_query (lambda (schema state)
	(make_query_block schema
		(rdf_shared_state_sources state)
		(rdf_shared_relation_fields (rdf_shared_state_bindings state))
		(rdf_shared_where (rdf_shared_state_filters state))
		nil nil nil nil nil '() '() '())
))
(define rdf_shared_relation_vars (lambda (bindings)
	(extract_assoc bindings (lambda (var _expr) var))
))
(define rdf_shared_relation_refs (lambda (alias vars)
	(reduce vars (lambda (acc var)
		(append acc var (rdf_shared_column alias (rdf_shared_input_field_name var)))) '())
))
(define rdf_shared_join_filters (lambda (left right)
	(reduce_assoc right (lambda (filters var right_expr)
		(match (rdf_ctx_lookup left var) '(found left_expr)
			(if found (cons (list (quote equal??) left_expr right_expr) filters) filters)
		)
	) '())
))
(define rdf_shared_merge_bindings (lambda (left right)
	(reduce_assoc right (lambda (bindings var expr)
		(match (rdf_ctx_lookup bindings var) '(found _old)
			(if found bindings (append bindings var expr)))
	) left)
))
(define rdf_shared_attach_relation (lambda (schema state relation vars outer)
	(begin
		(define index (rdf_shared_state_index state))
		(define alias (concat "__rdf_rel" index))
		(define right (rdf_shared_relation_refs alias vars))
		(define joins (rdf_shared_join_filters (rdf_shared_state_bindings state) right))
		(list
			(append (rdf_shared_state_sources state)
				(list alias schema relation outer
					(if (equal? joins '()) nil (rdf_shared_where joins))))
			(rdf_shared_merge_bindings (rdf_shared_state_bindings state) right)
			(rdf_shared_state_filters state)
			(+ index 1)))
))
(define rdf_shared_values_relation (lambda (schema var vals)
	(begin
		(define field (rdf_shared_input_field_name var))
		(define alias "__rdf_values")
		/* VALUES is a bounded query literal. Reuse the SQL frontend's native JSON
		table source so it follows the same physical table-function lowering. */
		(define relation (list (quote table-function) "array_text"
			(list (list (quote json_encode) (list (quote quote) vals))) (list "value")))
		(make_query_block schema
			(list (list alias schema relation false nil))
			(list field (rdf_shared_column alias "value"))
			true nil nil nil nil nil '() '() '())
	)
))
(define rdf_shared_reproject_query (lambda (query vars)
	(if (query_block? query)
		(make_query_block (qb_schema query) (qb_sources query)
			(reduce vars (lambda (fields var)
				(begin
					(define title (rdf_shared_input_field_name var))
					(append fields title (coalesceNil (get_assoc (qb_fields query) title) nil)))) '())
			(qb_where query) (qb_group query) (qb_having query) (qb_order query)
			(qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query))
		(error "SPARQL algebra: UNION branch must lower to query-block"))
))
(define rdf_shared_union_relation (lambda (schema branches outer_ctx mode)
	(begin
		(define vars (reduce branches (lambda (acc branch)
			(merge_unique (list acc (rdf_condition_vars branch)))) '()))
		(define queries (map branches (lambda (branch)
			(match (rdf_shared_conditions_relation schema branch outer_ctx) '(query _vars)
				(rdf_shared_reproject_query query vars)))))
		(list (make_union_block mode queries nil nil nil '()) vars)
	)
))
(define rdf_shared_path_relation (lambda (schema state subject pred object include_self)
	(begin
		(define bindings (rdf_shared_state_bindings state))
		(define start (rdf_shared_expr subject bindings '()))
		/* Reuse the planner's native single-column JSON table function as the
		physical row adapter. rdf_path_targets remains the RDF access primitive;
		the surrounding relation is ordinary common IR. */
		(define relation (list (quote table-function) "array_text"
			(list (list (quote json_encode)
				(list (quote rdf_path_targets) schema start pred include_self)))
			(list "value")))
		(define var (rdf_var_symbol object))
		(if (and (equal? (rdf_shared_state_sources state) '())
			(equal? bindings '()))
			(begin
				(define alias (concat "__rdf_path_values" (rdf_shared_state_index state)))
				(list (list (list alias schema relation false nil))
					(list var (rdf_shared_column alias "value"))
					(rdf_shared_state_filters state)
					(+ (rdf_shared_state_index state) 1)))
			(begin
				(define query (make_query_block schema
					(list (list "__rdf_path_values" schema relation false nil))
					(list (rdf_shared_input_field_name var)
						(rdf_shared_column "__rdf_path_values" "value"))
					true nil nil nil nil nil '() '() '()))
				(rdf_shared_attach_relation schema state query (list var) false)))
	)
))
(define rdf_shared_subquery_relation (lambda (schema subquery outer_ctx correlation_bindings)
	(match subquery
		'("select" cols "where" conditions "group" group "having" having "order" order "limit" limit "offset" offset "distinct" distinct)
		(begin
			/* SPARQL subqueries normally expose only projected variables. For the
			application's correlated read-model extension, decorrelate references to
			already-bound variables into hidden join keys in the same relational IR. */
			(define correlation_vars (filter (rdf_condition_vars conditions) (lambda (var)
				(rdf_ctx_bound correlation_bindings var))))
			(define projected_vars (reduce_assoc cols (lambda (acc title _expr)
				(append acc title)) '()))
			(define hidden_vars (filter correlation_vars (lambda (var)
				(not (rdf_key_in_list projected_vars var)))))
			(define correlated_cols (merge (list cols (merge (map hidden_vars (lambda (var)
				(list (concat var) (list (quote get_var) var))))))))
			(define correlated_group (if (and (rdf_select_has_aggregates correlated_cols)
				(not (equal? hidden_vars '())))
				(merge_unique (list group (map hidden_vars (lambda (var) (list (quote get_var) var)))))
				group))
			(define correlated (list "select" correlated_cols "where" conditions
				"group" correlated_group "having" having "order" order
				"limit" limit "offset" offset "distinct" distinct))
			(define inner (rdf_shared_query_ast schema correlated outer_ctx))
			(list inner (merge_unique (list projected_vars hidden_vars))))
		(error "SPARQL algebra: malformed subquery"))
))
(define rdf_shared_apply_operator (lambda (schema condition outer_ctx state)
	(match condition
		'("__bind__" expr var_expr)
		(begin
			(define var (rdf_var_symbol var_expr))
			(list (rdf_shared_state_sources state)
				(append (rdf_shared_state_bindings state) var
					(rdf_shared_expr expr (rdf_shared_state_bindings state) outer_ctx))
				(rdf_shared_state_filters state) (rdf_shared_state_index state)))
		'("__values__" var_expr vals)
		(begin
			(define var (rdf_var_symbol var_expr))
			(match (rdf_ctx_lookup (rdf_shared_state_bindings state) var) '(found value)
				(if found
					(begin
						(define comparisons (map vals (lambda (allowed)
							(list (quote equal??) value allowed))))
						(list (rdf_shared_state_sources state)
							(rdf_shared_state_bindings state)
							(cons (if (equal? comparisons '()) false
								(if (equal? (count comparisons) 1) (car comparisons)
									(cons (quote or) comparisons)))
								(rdf_shared_state_filters state))
							(rdf_shared_state_index state)))
					(rdf_shared_attach_relation schema state
						(rdf_shared_values_relation schema var vals) (list var) false)))
		)
		'("__optional__" inner)
		(match (rdf_shared_conditions_relation schema inner '()) '(query vars)
			(rdf_shared_attach_relation schema state query vars true))
		'("__union__" branches)
		(match (rdf_shared_union_relation schema branches '() (quote all)) '(query vars)
			(rdf_shared_attach_relation schema state query vars false))
		'("__union_distinct__" branches)
		(match (rdf_shared_union_relation schema branches '() (quote union_distinct)) '(query vars)
			(rdf_shared_attach_relation schema state query vars false))
		'("__graph__" graph inner)
		(begin
			(define graph_outer (rdf_shared_merge_bindings outer_ctx (rdf_shared_state_bindings state)))
			(match (rdf_shared_graph_relation schema graph inner graph_outer) '(query vars)
				(rdf_shared_attach_relation schema state query vars false)))
		'("__graph_restricted__" graph inner graphs)
		(begin
			(define graph_outer (rdf_shared_merge_bindings outer_ctx (rdf_shared_state_bindings state)))
			(match (rdf_shared_graph_relation schema graph inner graph_outer) '(query vars)
				(rdf_shared_apply_operator schema
					(list "__values__" graph graphs) outer_ctx
					(rdf_shared_attach_relation schema state query vars false))))
		'("__filter_exists__" negate inner)
		(match (rdf_shared_conditions_relation schema inner (rdf_shared_state_bindings state)) '(query _vars)
			(list (rdf_shared_state_sources state) (rdf_shared_state_bindings state)
				(cons
					(if negate (list (quote not) (list (quote inner_select_exists) query))
						(list (quote inner_select_exists) query))
					(rdf_shared_state_filters state))
				(rdf_shared_state_index state)))
		'(subject path object)
		(match path
			'("__path_star__" pred)
			(rdf_shared_path_relation schema state subject pred object true)
			'("__path_plus__" pred)
			(rdf_shared_path_relation schema state subject pred object false)
			_ state)
		'("__subquery__" subquery)
		(match (rdf_shared_subquery_relation schema subquery outer_ctx
			(rdf_shared_state_bindings state)) '(query vars)
			(begin
				(define index (rdf_shared_state_index state))
				(define alias (concat "__rdf_subquery" index))
				(define right (reduce vars (lambda (acc var)
					(append acc var (rdf_shared_column alias (concat var)))) '()))
				(define joins (rdf_shared_join_filters (rdf_shared_state_bindings state) right))
				(list
					(append (rdf_shared_state_sources state)
						(list alias schema query false
							(if (equal? joins '()) nil (rdf_shared_where joins))))
					(rdf_shared_merge_bindings (rdf_shared_state_bindings state) right)
					(rdf_shared_state_filters state)
					(+ index 1))))
		_ state
	)
))
(define rdf_shared_apply_operators (lambda (schema conditions outer_ctx state)
	(reduce conditions (lambda (current condition)
		(rdf_shared_apply_operator schema condition outer_ctx current)) state)
))
(define rdf_shared_direct_relation_query (lambda (query vars)
	(make_query_block (qb_schema query) (qb_sources query)
		(reduce vars (lambda (fields var)
			(append fields (rdf_shared_input_field_name var)
				(coalesceNil (get_assoc (qb_fields query) (concat var)) nil))) '())
		(qb_where query) (qb_group query) (qb_having query) (qb_order query)
		(qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query))
))
(define rdf_shared_conditions_relation (lambda (schema raw_conditions outer_ctx)
	(begin
		(define conditions (rdf_shared_expand_paths raw_conditions))
		(define patterns (filter conditions rdf_shared_direct_pattern?))
		/* Nested/correlated blocks must never reuse an outer source alias: name
		binding would otherwise turn the correlation predicate into x = x. */
		(define source_index (fnv_hash (concat conditions "|" outer_ctx)))
		(match (rdf_shared_build_sources schema patterns outer_ctx source_index '() '() '()) '(sources bindings filters)
			(begin
				(define state (rdf_shared_apply_operators schema conditions outer_ctx
					(list sources bindings filters (+ source_index (count patterns)))))
				(define filter_exprs (rdf_shared_filter_conditions conditions
					(rdf_shared_state_bindings state) outer_ctx))
				(define final_state (list (rdf_shared_state_sources state)
					(rdf_shared_state_bindings state)
					(merge (list (rdf_shared_state_filters state) filter_exprs))
					(rdf_shared_state_index state)))
				(define final_sources (rdf_shared_state_sources final_state))
				(define final_filters (rdf_shared_state_filters final_state))
				(define final_vars (rdf_shared_relation_vars (rdf_shared_state_bindings final_state)))
				(list
					(if (and (equal? (count final_sources) 1)
						(and (equal? final_filters '())
							(union_block? (source_relation (car final_sources)))))
						(source_relation (car final_sources))
						(if (and (equal? (count final_sources) 1)
							(and (equal? final_filters '())
								(and (rdf_startswith (source_alias (car final_sources)) "__rdf_subquery")
									(and (nil? (source_join_expr (car final_sources)))
										(query_block? (source_relation (car final_sources)))))))
							(rdf_shared_direct_relation_query (source_relation (car final_sources)) final_vars)
							(rdf_shared_relation_query schema final_state)))
					final_vars))))
)
))
(define rdf_shared_query_ast (lambda (schema query outer_ctx)
	(match query '("select" cols "where" conditions "group" group "having" having "order" order "limit" limit "offset" offset "distinct" distinct)
		(match (rdf_shared_conditions_relation schema conditions outer_ctx) '(input_query input_vars)
			(begin
				(define input_alias "__rdf_input")
				(define direct_table_function (and (query_block? input_query)
					(and (equal? (count (qb_sources input_query)) 1)
						(table_function_relation? (source_relation (car (qb_sources input_query)))))))
				(define bindings (if direct_table_function
					(reduce input_vars (lambda (acc var)
						(append acc var (get_assoc (qb_fields input_query) (rdf_shared_input_field_name var)))) '())
					(rdf_shared_relation_refs input_alias input_vars)))
				(define sources (if direct_table_function (qb_sources input_query)
					(list (list input_alias schema input_query false nil))))
				(define input_where (if direct_table_function (qb_where input_query) true))
				(if (or (rdf_select_has_aggregates cols)
					(or (not (equal? group '())) (not (nil? having))))
					(begin
						(define selected_fields (map_assoc cols (lambda (_title expr)
							(rdf_shared_expr expr bindings outer_ctx))))
						(make_query_block schema sources
							selected_fields true
							(if (equal? group '()) nil
								(map group (lambda (expr) (rdf_shared_expr expr bindings outer_ctx))))
							(if (nil? having) nil (rdf_shared_expr having bindings outer_ctx))
							(rdf_shared_order order bindings outer_ctx)
							limit offset '() '() '()))
					(begin
						(define selected_fields (map_assoc cols (lambda (_title expr)
							(rdf_shared_expr expr bindings outer_ctx))))
						/* DISTINCT is represented as a grouping fact. Only projected columns
						may reach that grouping layer; helper bindings remain available for
						ASK/CONSTRUCT/update callbacks on non-DISTINCT queries. */
						(define fields (if distinct selected_fields
							(rdf_shared_complete_fields selected_fields bindings)))
						(define projected (extract_assoc selected_fields (lambda (_title expr) expr)))
						(make_query_block schema sources fields input_where
							(if distinct projected nil) nil
							(rdf_shared_order order bindings outer_ctx) limit offset '() '()
							(if distinct (list (list (quote select_distinct) true)) '()))))))
		(error "SPARQL shared planner: expected SELECT query")
	)
))
(define rdf_shared_result_context (lambda (cols outer_ctx)
	(match cols
		(cons title (cons _expr tail))
		(merge (rdf_shared_result_context tail outer_ctx)
			(list title (list (quote rdf_row_lookup) (quote __rdf_values) title)))
		'() outer_ctx
	)
))
(define rdf_shared_queryplan (lambda (schema query outer_ctx resultfunc)
	(begin
		(define ast (rdf_shared_query_ast schema query outer_ctx))
		/* Correlated EXISTS/subqueries use the same request-local planning and
		transaction carriers as SQL. Passing nil here was sufficient for a BGP,
		but loses the runtime session preparation required by decorrelation. */
		(define plan (build_queryplan_term ast planning_session tx))
		(define result_ctx (rdf_shared_result_context (qb_fields ast) outer_ctx))
		(define result_body (resultfunc (nth query 1) result_ctx))
		(list
			(list (quote lambda) (list (quote __rdf_outer_resultrow))
				(list (quote begin)
					(list (quote set) (quote resultrow)
						(list (quote lambda) (list (quote __rdf_values))
							(list
								(list (quote lambda) (list (quote resultrow)) result_body)
								(quote __rdf_outer_resultrow))))
					plan))
			(quote resultrow)))
))

(define rdf_queryplan (lambda (schema query definitions ctx resultfunc /* function that gets cols + ctx */)
	(rdf_shared_queryplan schema query ctx resultfunc)
))

(define parse_sparql (lambda (schema s _policy planning_session tx) (match (ttl_header s)
	'("prefixes" definitions "rest" rest) (begin
		(set cleaned_rest (rdf_strip_leading_ws_comments rest))
		(set parsed (rdf_query cleaned_rest))
		(set parsed (rdf_resolve_prefixes parsed definitions))
		(set parsed (rdf_expand_select_star parsed))
		(match parsed
			'("create_graph" _graph)
			(list (quote rdf_ensure_named_table) schema)
			'("clear_graph" graph)
			(list (quote rdf_clear_graph_data) schema graph)
			'("drop_graph" graph)
			(list (quote rdf_clear_graph_data) schema graph)
			'("insert_data" triples)
			(list (quote begin)
				(list (quote rdf_ensure_table) schema)
				(list (quote rdf_insert_triples) schema (list (quote quote) triples)))
			'("insert_graph_data" graph triples)
			(list (quote rdf_insert_graph_triples) schema graph (list (quote quote) triples))
			'("delete_data" triples)
			(list (quote begin)
				(list (quote rdf_ensure_table) schema)
				(list (quote rdf_delete_triples) schema (list (quote quote) triples)))
			'("delete_graph_data" graph triples)
			(list (quote rdf_delete_graph_triples) schema graph (list (quote quote) triples))
			'("ask" "where" conditions) (begin
				(set _ask_state (newsession))
				(_ask_state "matched" false)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema '("select" '() "where" conditions "group" '() "having" nil "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols _ctx)
						(list _ask_state "matched" true)))
					(list (quote resultrow) (list (quote list) "?ask" (list _ask_state "matched")))
			))
			'("construct" triples "where" conditions)
			(list (quote begin)
				(list (quote rdf_ensure_table) schema)
				(rdf_queryplan schema '("select" '() "where" conditions "group" '() "having" nil "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols ctx)
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
					(rdf_queryplan schema '("select" '() "where" conditions "group" '() "having" nil "order" nil "limit" nil "offset" nil "distinct" nil) definitions '() (lambda (_cols ctx)
						(list (quote begin)
							(list _delete_rows (list (quote uuid)) (rdf_template_expr delete_triples ctx))
							(list _insert_rows (list (quote uuid)) (rdf_template_expr insert_triples ctx))
						)
					))
					(list (quote rdf_delete_triples) schema (list (quote rdf_session_merged_values) _delete_rows))
					(list (quote rdf_insert_triples) schema (list (quote rdf_session_merged_values) _insert_rows))
			))
			'("select" cols "where" conditions "group" qgroup "having" qhaving "order" qorder "limit" qlimit "offset" qoffset "distinct" qdistinct) (begin
				(set missing_select_vars (rdf_missing_select_vars cols conditions))
				(if (not (equal? missing_select_vars '()))
					(error "SPARQL error: unbound SELECT variable" missing_select_vars)
					nil
				)
				(set qhasagg (or (rdf_select_has_aggregates cols) (rdf_has_aggregate qhaving)))
				(if (or qhasagg (not (equal? qgroup '())))
					(rdf_queryplan schema parsed definitions '() rdf_shared_resultrow_ast)
					(rdf_queryplan schema parsed definitions '() rdf_select_resultrow_ast)
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
(define rdf_expand_ttl_facts (lambda (facts)
	(merge (map facts (lambda (triple) (match triple '(subject pred obj)
		(rdf_expand_ttl_object subject pred obj)))))
))
(define rdf_expand_ttl_collection_cells (lambda (head item tail)
	(begin
		(define next (match tail
			'() "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
			_ (concat head "_rest")))
		(cons (list head "http://www.w3.org/1999/02/22-rdf-syntax-ns#first" item)
			(cons (list head "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest" next)
				(match tail
					(cons next_item remaining)
					(rdf_expand_ttl_collection_cells next next_item remaining)
					_ '()))))
))
(define rdf_expand_ttl_collection (lambda (subject pred items head)
	(match items
		'() (list (list subject pred "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"))
		(cons item tail) (begin
			(cons (list subject pred head)
				(rdf_expand_ttl_collection_cells head item tail)))
	)
))
(define rdf_expand_ttl_object (lambda (subject pred obj) (match obj
	'("__ttl_inline_node__" bn facts)
	(cons (list subject pred bn) (rdf_expand_ttl_facts facts))
	'("__ttl_collection__" items head)
	(rdf_expand_ttl_collection subject pred items head)
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
				(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_object ",")) (? ";")) (map os (lambda (o) '(p o))))))
				"]"
			) (begin
					(define bn (concat "_:anon_" (uuid)))
					(list "__ttl_inline_node__" bn (merge (map ps (lambda (p) (map p (lambda (p1) (cons bn p1)))))))
			))
			(parser '(
				"("
				/* Keep the list-item grammar non-recursive. Recursive parser objects are
				mis-specialized by the experimental JIT and silently lose their items. */
				(define items (* ttl_simple_constant))
				")"
			) (list "__ttl_collection__" items (concat "_:list_" (uuid))))
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
	(rdf_delete_triples schema triples)
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
				(define ps (+ (parser '((define p ttl_simple_constant) (define os (+ ttl_object ",")) (? ";")) (map os (lambda (o) '(p o))))))
				"]"
			) (begin
					(define bn (concat "_:anon_" (uuid)))
					(list "__ttl_inline_node__" bn (merge (map ps (lambda (p) (map p (lambda (p1) (cons bn p1)))))))
			))
			(parser '(
				"("
				(define items (* ttl_simple_constant))
				")"
			) (list "__ttl_collection__" items (concat "_:list_" (uuid))))
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
			(rdf_insert_triples schema (map facts (lambda (triple) (list (resolve_blank (car triple)) (resolve_blank (car (cdr triple))) (resolve_blank (car (cdr (cdr triple))))))))
		)))
		(define process_fact (lambda (rest) (match (ttl_fact rest)
			'("facts" facts "rest" (regex "^[ \\n\\r\\t]*$" _)) (load facts)
			'("facts" facts "rest" rest) (!begin (load facts) (process_fact rest))
			rest (error "couldnt parse: " rest)
		)))
		(process_fact rest)
	)
)))
