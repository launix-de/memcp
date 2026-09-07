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
	(parser '((atom "<" false false) (define iri (regex "[^>]*" false false)) (atom ">" false false)) iri) /* ^^<IRI> */
	(regex "[a-zA-Z0-9_]*:[a-zA-Z0-9_]*" false false) /* ^^prefix:name */
	(regex "[a-zA-Z0-9_]+" false false) /* ^^barename */
)))
/* unescape standard TTL/JSON escape sequences in a string */
(define rdf_unescape (lambda (s)
	(replace (replace (replace (replace (replace s "\\n" "\n") "\\t" "\t") "\\\\" "\\") "\\\"" "\"") "\\r" "\r")
))
(define rdf_unescape_single (lambda (s)
	(replace (rdf_unescape s) "\\'" "'")
))
(define rdf_unescape_iri (lambda (s)
	(json_decode_scmer (concat "\"" s "\""))
))
(define rdf_unescape_pname (lambda (s)
	(regexp_replace s "\\\\([~._-])" "$1")
))
(define rdf_typed_literal (lambda (value datatype)
	(if (regexp_test datatype "(?:#|:)integer$")
		(json_decode_scmer value)
		(if (regexp_test datatype "(?:#|:)(?:decimal|double|float)$")
			(simplify value)
			(if (regexp_test datatype "(?:#|:)boolean$")
				(equal? (toLower value) "true")
				value)))
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
(define rdf_string_find_using (lambda (s needle position)
	(if (equal? needle "") position
		(if (> (+ position (strlen needle) -1) (strlen s)) 0
			(if (equal? (sql_substr s position (strlen needle)) needle) position
				(rdf_string_find_using s needle (+ position 1)))))
))
(define rdf_strbefore (lambda (s needle)
	(if (or (nil? s) (nil? needle)) nil
		(begin
			(define position (rdf_string_find_using s needle 1))
			(if (equal? position 0) "" (sql_substr s 1 (- position 1)))))
))
(define rdf_strafter (lambda (s needle)
	(if (or (nil? s) (nil? needle)) nil
		(begin
			(define position (rdf_string_find_using s needle 1))
			(if (equal? position 0) ""
				(sql_substr s (+ position (strlen needle))))))
))
(define rdf_regex_pattern (lambda (pattern flags)
	(if (or (nil? flags) (equal? flags "")) pattern
		(begin
			(define supported (concat
				(if (rdf_contains flags "i") "i" "")
				(if (rdf_contains flags "m") "m" "")
				(if (rdf_contains flags "s") "s" "")))
			(if (equal? supported "") pattern (concat "(?" supported ")" pattern))))
))
(define rdf_regex (lambda (value pattern flags)
	(regexp_test value (rdf_regex_pattern pattern flags))
))
(define rdf_replace_regex (lambda (value pattern replacement flags)
	(regexp_replace value (rdf_regex_pattern pattern flags) replacement)
))
(define rdf_encode_for_uri (lambda (value)
	(if (nil? value) nil (replace (urlencode value) "+" "%20"))
))
(define rdf_timezone (lambda (value)
	(if (nil? value) nil
		(regexp_replace (concat value) ".*(Z|[+-][0-9]{2}:[0-9]{2})$" "$1"))
))
(define rdf_date_component (lambda (value start length part)
	(if (nil? value) nil
		(if (string? value)
			(simplify (sql_substr value start length))
			(extract_date value part)))
))
(define rdf_json_objectagg_reduce (lambda (a b)
	(if (nil? a) b (if (nil? b) a (json_merge_patch a b)))
))
(define rdf_json_arrayagg_reduce (lambda (a b)
	(if (nil? a) b (if (nil? b) a (merge a b)))
))
(define rdf_sample_reduce (lambda (a b) (if (nil? a) b a)))
(define rdf_divide (lambda (left right)
	(if (equal?? right 0) nil (/ left right))
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
	(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) x (rdf_typed_literal x datatype)))
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x)) /* string with language */
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) (rdf_unescape x) (rdf_typed_literal (rdf_unescape x) datatype)))
	(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node _:identifier */
	(parser '((define x (regex "[+-]?[0-9]+(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?" true))) (simplify x))
	(parser '((atom "true" true)) true)
	(parser '((atom "false" true)) false)
	(regex "[a-zA-Z0-9_]+" true) /* bare name */
)))
(define rdf_expression (parser (or
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) '('concat '('definitions pfx) post)) /* as expression */
	rdf_variable
	rdf_constant
	/* TODO: CONCAT() */
)))
(define rdf_iri_expression (parser (or
	(parser '((atom "a" true)) "http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false)
		(define post (regex "[a-zA-Z0-9_]*" false)))
		'('concat '('definitions pfx) post))
	(parser '((atom "<" true) (define iri (regex "[^>]*" false false))
		(atom ">" false false)) iri)
	(regex "[a-zA-Z0-9_]+" true)
)))
(define rdf_subject_expression (parser (or
	rdf_variable
	(parser '((atom "_:" true) (define name (regex "[a-zA-Z0-9_]+" false false)))
		(concat "_:" name))
	rdf_iri_expression
)))
(define rdf_predicate_expression (parser (or
	rdf_variable
	rdf_iri_expression
)))
(define rdf_aggregate_expression (parser (or
	(parser '((atom "JSON_ARRAYAGG" true) "(" (define e rdf_filter_or)
		(atom "ORDER" true) (atom "BY" true) (define key rdf_filter_or)
		(? (define dir (or (atom "DESC" true) (atom "ASC" true)))) ")")
		(list "__rdf_agg__" "JSON_ARRAYAGG_ORDERED" e (list key (coalesce dir "ASC"))))
	(parser '((atom "JSON_ARRAYAGG" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "JSON_ARRAYAGG" e nil))
	(parser '((atom "JSON_OBJECTAGG" true) "(" (define key rdf_filter_or) "," (define value rdf_filter_or) ")") '("__rdf_agg__" "JSON_OBJECTAGG" '('json_objectagg_entry key value) nil))
	(parser '((atom "COUNT" true) "(" "*" ")") '("__rdf_agg__" "COUNT" 1 nil))
	(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define e rdf_filter_or) ")") '("__rdf_agg__" "COUNT_DISTINCT" e nil))
	(parser '((atom "COUNT" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "COUNT" e nil))
	(parser '((atom "SUM" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "SUM" e nil))
	(parser '((atom "AVG" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "AVG" e nil))
	(parser '((atom "MIN" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "MIN" e nil))
	(parser '((atom "MAX" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "MAX" e nil))
	(parser '((atom "SAMPLE" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "SAMPLE" e nil))
	(parser '((atom "GROUP_CONCAT" true) "(" (define e rdf_filter_or) ";" (atom "separator" true) "=" (define sep rdf_filter_or) ")") '("__rdf_agg__" "GROUP_CONCAT" e sep))
	(parser '((atom "GROUP_CONCAT" true) "(" (define e rdf_filter_or) ")") '("__rdf_agg__" "GROUP_CONCAT" e " "))
)))

/* SPARQL filter expressions — no bare names (would eat keywords) */
(define rdf_filter_atom (parser (or
	rdf_variable
	(parser '((define n (regex "[+-]?[0-9]+(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?" true))) (simplify n))
	(parser '((atom "true" true)) true)
	(parser '((atom "false" true)) false)
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x)
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) (rdf_unescape x) (rdf_typed_literal (rdf_unescape x) datatype)))
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
	(parser '((atom "REPLACE" true) "(" (define a rdf_filter_or) "," (define pattern rdf_filter_or) "," (define replacement rdf_filter_or) "," (define flags rdf_filter_or) ")") '('rdf_replace_regex a pattern replacement flags))
	(parser '((atom "STRBEFORE" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('rdf_strbefore a b))
	(parser '((atom "STRAFTER" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) ")") '('rdf_strafter a b))
	(parser '((atom "ENCODE_FOR_URI" true) "(" (define a rdf_filter_or) ")") '('rdf_encode_for_uri a))
	(parser '((atom "REGEX" true) "(" (define a rdf_filter_or) "," (define b rdf_filter_or) "," (define flags rdf_filter_or) ")") '('rdf_regex a b flags))
	(parser '((atom "RAND" true) "(" ")") '('sql_rand))
	(parser '((atom "NOW" true) "(" ")") '('now))
	(parser '((atom "YEAR" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 1 4 "YEAR"))
	(parser '((atom "MONTH" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 6 2 "MONTH"))
	(parser '((atom "DAY" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 9 2 "DAY"))
	(parser '((atom "HOURS" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 12 2 "HOUR"))
	(parser '((atom "MINUTES" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 15 2 "MINUTE"))
	(parser '((atom "SECONDS" true) "(" (define a rdf_filter_or) ")") '('rdf_date_component a 18 2 "SECOND"))
	(parser '((atom "TZ" true) "(" (define a rdf_filter_or) ")") '('rdf_timezone a))
	(parser '((atom "MD5" true) "(" (define a rdf_filter_or) ")") '('md5 a))
	(parser '((atom "SHA1" true) "(" (define a rdf_filter_or) ")") '('sha1 a))
	(parser '((atom "SHA256" true) "(" (define a rdf_filter_or) ")") '('sha256 a))
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
(define rdf_filter_mul (parser (or
	(parser '((define a rdf_filter_not) "*" (define b rdf_filter_mul)) '('* a b))
	(parser '((define a rdf_filter_not) "/" (define b rdf_filter_mul)) '('/ a b))
	rdf_filter_not
)))
(define rdf_filter_add (parser (or
	(parser '((define a rdf_filter_mul) "+" (define b rdf_filter_add)) '('+ a b))
	(parser '((define a rdf_filter_mul) "-" (define b rdf_filter_add)) '('- a b))
	rdf_filter_mul
)))
(define rdf_filter_cmp (parser (or
	(parser '((define a rdf_filter_add) (atom "NOT" true) (atom "IN" true) "(" (define b (+ rdf_filter_or ",")) ")")
		(list (quote not) (cons (quote sql_in) (cons (cons (quote list) b) (list a)))))
	(parser '((define a rdf_filter_add) (atom "IN" true) "(" (define b (+ rdf_filter_or ",")) ")")
		(cons (quote sql_in) (cons (cons (quote list) b) (list a))))
	(parser '((define a rdf_filter_add) "!=" (define b rdf_filter_add)) '('not '('equal? a b)))
	(parser '((define a rdf_filter_add) "=" (define b rdf_filter_add)) '('equal? a b))
	(parser '((define a rdf_filter_add) "<=" (define b rdf_filter_add)) '('<= a b))
	(parser '((define a rdf_filter_add) ">=" (define b rdf_filter_add)) '('>= a b))
	(parser '((define a rdf_filter_add) "<" (define b rdf_filter_add)) '('< a b))
	(parser '((define a rdf_filter_add) ">" (define b rdf_filter_add)) '('> a b))
	rdf_filter_add
)))
(define rdf_filter_and (parser (or
	(parser '((define a rdf_filter_cmp) "&&" (define b rdf_filter_and)) '('and a b))
	rdf_filter_cmp
)))
(define rdf_filter_or (parser (or
	(parser '((define a rdf_filter_and) "||" (define b rdf_filter_or)) '('or a b))
	rdf_filter_and
)))

(define rdf_path_negated_member (parser (or
	(parser '((atom "^" true) (define p rdf_predicate_expression)) (list "inverse" p))
	(parser (define p rdf_predicate_expression) (list "forward" p))
)))
(define rdf_path_atom (parser (or
	(parser '((atom "!" true) "(" (define members (+ rdf_path_negated_member "|")) ")")
		(list "__path_negated__" members))
	(parser '((atom "!" true) (define member rdf_path_negated_member))
		(list "__path_negated__" (list member)))
	(parser '((atom "^" true) "(" (define p rdf_path_alt) ")") (list "__path_inverse__" p))
	(parser '((atom "^" true) (define p rdf_predicate_expression)) (list "__path_inverse__" p))
	(parser '("(" (define p rdf_path_alt) ")") p)
	rdf_predicate_expression
)))
(define rdf_path_postfix (parser (or
	(parser '((define p rdf_path_atom) "*") '("__path_star__" p))
	(parser '((define p rdf_path_atom) "+") '("__path_plus__" p))
	/* A path postfix is adjacent to its path. Disallow leading whitespace so
	`p ?object` cannot consume the object's variable marker as `p?`. */
	(parser '((define p rdf_path_atom) (atom "?" false false)) '("__path_optional__" p))
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
	(parser '((define s rdf_subject_expression) (define ps (+ (parser '((define p rdf_path_alt) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
	(parser '((atom "FILTER" true) "(" (define expr rdf_filter_or) ")") (list (list "__filter__" expr)))
)))
(define rdf_where_inner_basic_items (parser
	(* (parser '((define item rdf_where_basic_item) (? (atom "." true))) item))
))
(define rdf_where_group_items (parser
	(* (parser '((define item rdf_where_item) (? (atom "." true))) item))
))
(define rdf_where_optional_item (parser '(
	(atom "OPTIONAL" true)
	(atom "{" true)
	(define conditions rdf_where_group_items)
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
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__filter_exists__" true (merge (coalesce conditions '('())))))))
(define rdf_where_filter_yes_exists_item (parser '(
	(atom "FILTER" true)
	(atom "EXISTS" true)
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__filter_exists__" false (merge (coalesce conditions '('())))))))
(define rdf_where_filter_exists_item (parser (or
	rdf_where_filter_not_exists_item
	rdf_where_filter_yes_exists_item
)))
(define rdf_where_union_group (parser '(
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (merge (coalesce conditions '('())))))
(define rdf_where_union_tail_item (parser '(
	(atom "UNION" true)
	(define next rdf_where_union_group)
) next))
(define rdf_values_term (parser (or
	(parser (atom "UNDEF" true) (rdf_unbound_expr))
	rdf_expression
)))
(define rdf_values_row (parser '(
	"(" (define vals (* rdf_values_term)) ")"
) vals))
(define rdf_where_values_item (parser '(
	(atom "VALUES" true)
	(define values (or
		(parser '((define var rdf_variable) (atom "{" true)
			(define vals (* rdf_values_term)) (atom "}" true))
			(list "single" var vals))
		(parser '((atom "(" true) (define vars (+ rdf_variable)) (atom ")" true)
			(atom "{" true) (define rows (* rdf_values_row)) (atom "}" true))
			(begin
				(if (reduce rows (lambda (ok row) (and ok (equal? (count row) (count vars)))) true)
					true (error "SPARQL VALUES row arity mismatch"))
				(list "tuple" vars rows)))))
) (match values
	'("single" var vals) (list (list "__values__" var vals))
	'("tuple" vars rows) (list (list "__values_tuple__" vars rows)))))
(define rdf_where_graph_item (parser '(
	(atom "GRAPH" true)
	(define graph rdf_expression)
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__graph__" graph (merge (coalesce conditions '('())))))))
(define rdf_subquery_select_col (parser (or
	(parser '("(" (define v rdf_aggregate_expression) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '("(" (define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable) ")") (match v2 '('get_var s) '((concat s) v)))
	(parser '((define v rdf_filter_or) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v)))
	(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v)))
)))
(define rdf_where_minus_item (parser '(
	(atom "MINUS" true)
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__minus__" (merge (coalesce conditions '()))))))
(define rdf_where_service_item (parser '(
	(atom "SERVICE" true)
	(? (define silent (atom "SILENT" true)))
	(define endpoint rdf_expression)
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__service__" silent endpoint (merge (coalesce conditions '()))))))
(define rdf_where_nested_group_item (parser '(
	(atom "{" true)
	(define conditions rdf_where_group_items)
	(atom "}" true)
) (list (list "__group__" (merge (coalesce conditions '()))))))
(define rdf_where_subquery_item (parser '(
	(atom "{" true)
	(atom "SELECT" true)
	(? (define distinct (atom "DISTINCT" true)))
	(define cols (or
		(parser (atom "*" true) "__select_all__")
		(+ (parser '((define col rdf_subquery_select_col) (? (atom "," true))) col))))
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
			rdf_where_minus_item
			rdf_where_service_item
			rdf_where_nested_group_item
			rdf_where_basic_item))
		(? (atom "." true))
	) item)))
	(atom "}" true)
	(? (atom "GROUP" true) (atom "BY" true)
		(define group (+ (parser '((define var rdf_variable) (? (atom "," true))) var))))
	(? (atom "HAVING" true) "(" (define having rdf_filter_or) ")")
	(? (atom "ORDER" true) (atom "BY" true)
		(define ordercols (+ rdf_order_condition)))
	(? (atom "LIMIT" true) (define limit (parser (define n (regex "[0-9]+" true)) (simplify n))))
	(? (atom "OFFSET" true) (define offset (parser (define n (regex "[0-9]+" true)) (simplify n))))
	(atom "}" true)
) (begin
	(define merged_conditions (merge (coalesce conditions '())))
	(define selected (if (equal? cols "__select_all__")
		(merge (map (rdf_condition_vars merged_conditions) (lambda (var)
			(list (concat var) (list (quote get_var) var)))))
		(merge cols)))
		(list (list "__subquery__"
			(list "select" selected "where" merged_conditions
				"group" (coalesce group '()) "having" having "order" ordercols
				"limit" limit "offset" offset "distinct" distinct))))))
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
	rdf_where_minus_item
	rdf_where_service_item
	rdf_where_nested_group_item
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
		(parser '((define dir (or (atom "DESC" true) (atom "ASC" true))) "(" (define expr rdf_filter_or) ")") '(expr dir))
		(parser (define expr rdf_filter_or) '(expr "ASC")))
	(atom "LIMIT" true)
	(atom "OFFSET" true)
	/* RDFHP embeds SELECT directly before its block delimiters. They must not
	be consumed as legacy bare-name ORDER BY expressions. */
	(atom "BEGIN" true)
	(atom "ELSE" true)
	(atom "END" true)
)))
(define rdf_limit_offset (parser (or
	(parser '((atom "LIMIT" true) (define limit rdf_number) (? (atom "OFFSET" true) (define offset rdf_number))) '(limit offset))
	(parser '((atom "OFFSET" true) (define offset rdf_number) (? (atom "LIMIT" true) (define limit rdf_number))) '(limit offset))
)))
(define rdf_group_item (parser (or
	(parser '("(" (define expr rdf_filter_or) (atom "AS" true)
		(define var rdf_variable) ")") (list expr var))
	(parser (define expr rdf_filter_or) (list expr nil))
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
	(? (define distinct (or (atom "DISTINCT" true) (atom "REDUCED" true))))
	(define cols (or
		(parser (atom "*" true) "__select_all__")
		(+ (parser '((define col rdf_select_col) (? (atom "," true))) col))))
	(define datasets (* rdf_dataset_clause))
	(? (atom "WHERE" true))
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
	(?
		(atom "GROUP" true)
		(atom "BY" true)
		(define group (+ (parser '((define item rdf_group_item) (? (atom "," true))) item)))
	)
	(define havings (* (parser '((atom "HAVING" true) "("
		(define expr rdf_filter_or) ")") expr)))
	(?
		(atom "ORDER" true)
		(atom "BY" true)
		(define ordercols (+ rdf_order_condition))
	)
	(define slice (or rdf_limit_offset (parser empty '(nil nil))))
	(? (define trailing_values rdf_where_values_item))
) (begin
	(define group_items (coalesce group '()))
	(define group_bindings (map (filter group_items
		(lambda (item) (not (nil? (cadr item)))))
		(lambda (item) (list "__bind__" (car item) (cadr item)))))
	(define having (match havings
		'() nil
		(cons only '()) only
		_ (cons (quote and) havings)))
	(list "select" (if (equal? cols "__select_all__") cols (merge cols)) "where"
		(rdf_dataset_conditions datasets
			(merge (list (merge (coalesce conditions '())) group_bindings
				(coalesce trailing_values '()))))
		"group" (map group_items car) "having" having "order" ordercols
		"limit" (car slice) "offset" (cadr slice) "distinct" distinct))
	"^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

(define rdf_template_object (parser (or
	(parser '("[" (define pred rdf_expression) (define obj rdf_expression) "]")
		(list "__template_bnode__" pred obj))
	rdf_expression
)))
(define rdf_template_item (parser '(
	(define s rdf_expression)
	(define ps (+ (parser '((define p rdf_expression) (define os (+ rdf_template_object ","))) (map os (lambda (o) '(p o)))) ";"))
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
) (begin
	(define merged (merge (coalesce triples '())))
	(if (equal? (rdf_condition_vars merged) '())
		(list "insert_data" merged)
		(error "SPARQL INSERT DATA does not allow variables")))))
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
) (begin
	(define merged (merge (coalesce triples '())))
	(if (and (equal? (rdf_condition_vars merged) '())
		(not (match graph '('get_var _name) true _ false)))
		(list "insert_graph_data" graph merged)
		(error "SPARQL INSERT DATA does not allow variables")))))
(define rdf_delete_data (parser '(
	(atom "DELETE" true)
	(atom "DATA" true)
	(atom "{" true)
	(define triples rdf_template_items)
	(atom "}" true)
) (begin
	(define merged (merge (coalesce triples '())))
	(if (equal? (rdf_condition_vars merged) '())
		(list "delete_data" merged)
		(error "SPARQL DELETE DATA does not allow variables")))))
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
) (begin
	(define merged (merge (coalesce triples '())))
	(if (and (equal? (rdf_condition_vars merged) '())
		(not (match graph '('get_var _name) true _ false)))
		(list "delete_graph_data" graph merged)
		(error "SPARQL DELETE DATA does not allow variables")))))
(define rdf_update_dataset_clause (parser '(
	(atom "USING" true)
	(? (define named (atom "NAMED" true)))
	(define graph rdf_expression)
) (list (if named "named" "default") graph)))
(define rdf_delete_where (parser '(
	(atom "DELETE" true) (atom "WHERE" true) (atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) (begin
	(define merged (merge (coalesce conditions '())))
	(list "modify" "graph" "__rdf_default_graph__" "delete" merged
		"insert" '() "where" merged))))
(define rdf_modify (parser '(
	(? (atom "WITH" true) (define with_graph rdf_expression))
	(define delete_part (? (parser '((atom "DELETE" true) (atom "{" true)
		(define triples rdf_template_items) (atom "}" true)) (merge triples))))
	(define insert_part (? (parser '((atom "INSERT" true) (atom "{" true)
		(define triples rdf_template_items) (atom "}" true)) (merge triples))))
	(define datasets (* rdf_update_dataset_clause))
	(atom "WHERE" true) (atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) (begin
	(if (and (nil? delete_part) (nil? insert_part))
		(error "SPARQL MODIFY requires DELETE or INSERT") true)
	(define effective_datasets (if (and (equal? datasets '()) (not (nil? with_graph)))
		(list (list "default" with_graph)) datasets))
	(list "modify" "graph" (coalesce with_graph "__rdf_default_graph__")
		"delete" (coalesce delete_part '()) "insert" (coalesce insert_part '())
		"where" (rdf_dataset_conditions effective_datasets
			(merge (coalesce conditions '())))))))
(define rdf_ask (parser '(
	(atom "ASK" true)
	(define datasets (* rdf_dataset_clause))
	(? (atom "WHERE" true))
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
) (list "ask" "where"
	(rdf_dataset_conditions datasets (merge (coalesce conditions '()))))))
(define rdf_construct (parser '(
	(atom "CONSTRUCT" true)
	(define shorthand (or
		(parser '((atom "WHERE" true) (atom "{" true) (define triples rdf_template_items) (atom "}" true))
			(list (merge triples) (merge triples)))
		(parser '((atom "{" true) (define triples rdf_template_items) (atom "}" true)
			(atom "WHERE" true) (atom "{" true)
			(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
			(atom "}" true)) (list (merge triples) (merge conditions)))))
	(? (atom "ORDER" true) (atom "BY" true) (define ordercols (+ rdf_order_condition)))
	(define slice (or rdf_limit_offset (parser empty '(nil nil))))
) (list "construct" (coalesce (car shorthand) '()) "where"
	(coalesce (cadr shorthand) '()) "order" ordercols
	"limit" (car slice) "offset" (cadr slice))))
(define rdf_describe_target (parser (or
	rdf_variable
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x)
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false)
		(define post (regex "[a-zA-Z0-9_]*" false))) '('concat '('definitions pfx) post))
)))
(define rdf_describe (parser '(
	(atom "DESCRIBE" true)
	(define targets (or
		(parser (atom "*" true) "__describe_all__")
		(+ rdf_describe_target)))
	(define datasets (* rdf_dataset_clause))
	(?
	(atom "WHERE" true)
	(atom "{" true)
	(define conditions (* (parser '((define item rdf_where_item) (? (atom "." true))) item)))
	(atom "}" true)
	)
	(? (atom "ORDER" true) (atom "BY" true) (define ordercols (+ rdf_order_condition)))
	(define slice (or rdf_limit_offset (parser empty '(nil nil))))
) (list "describe" targets "where"
	(rdf_dataset_conditions datasets (merge (coalesce conditions '())))
	"order" ordercols "limit" (car slice) "offset" (cadr slice))))
(define rdf_create_graph (parser '(
	(atom "CREATE" true) (? (define silent (atom "SILENT" true))) (atom "GRAPH" true)
	(define graph rdf_expression)
) '("create_graph" graph silent)))
(define rdf_clear_graph (parser '(
	(atom "CLEAR" true) (? (define silent (atom "SILENT" true)))
	(define target (or
		(parser (atom "DEFAULT" true) "default")
		(parser (atom "NAMED" true) "named")
		(parser (atom "ALL" true) "all")
		(parser '((atom "GRAPH" true) (define graph rdf_expression)) graph)))
) '("clear_graph" target silent)))
(define rdf_drop_graph (parser '(
	(atom "DROP" true) (? (define silent (atom "SILENT" true)))
	(define target (or
		(parser (atom "DEFAULT" true) "default")
		(parser (atom "NAMED" true) "named")
		(parser (atom "ALL" true) "all")
		(parser '((atom "GRAPH" true) (define graph rdf_expression)) graph)))
) '("drop_graph" target silent)))
(define rdf_load (parser '(
	(atom "LOAD" true) (? (define silent (atom "SILENT" true)))
	(define source rdf_expression)
	(? (atom "INTO" true) (atom "GRAPH" true) (define target rdf_expression))
) '("load" source target silent)))
(define rdf_graph_transfer_ref (parser (or
	(parser (atom "DEFAULT" true) "__rdf_default_graph__")
	(parser '((atom "GRAPH" true) (define graph rdf_expression)) graph)
)))
(define rdf_graph_transfer (parser '(
	(define operation (or (atom "COPY" true) (atom "MOVE" true) (atom "ADD" true)))
	(? (define silent (atom "SILENT" true)))
	(define source rdf_graph_transfer_ref)
	(atom "TO" true)
	(define target rdf_graph_transfer_ref)
) '("graph_transfer" operation source target silent)))
(define rdf_query_core (parser (or
	rdf_load
	rdf_graph_transfer
	rdf_create_graph
	rdf_clear_graph
	rdf_drop_graph
	rdf_insert_graph_data
	rdf_delete_graph_data
	rdf_insert_data
	rdf_delete_data
	rdf_delete_where
	rdf_modify
	rdf_ask
	rdf_construct
	rdf_describe
	rdf_select
)))
(define rdf_query (parser '(
	(define operations (+ rdf_query_core ";"))
) (if (equal? (count operations) 1) (car operations)
	(list "update_request" operations))))

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
		'("__group__" inner) (rdf_condition_vars inner)
		'("__minus__" inner) (rdf_condition_vars inner)
		'("__service__" _silent _endpoint inner) (rdf_condition_vars inner)
		'("__graph__" graph inner) (merge_unique (list (rdf_extract_vars graph) (rdf_condition_vars inner)))
		'("__graph_restricted__" graph inner _graphs)
		(merge_unique (list (rdf_extract_vars graph) (rdf_condition_vars inner)))
		'("__bind__" expr var_expr) (merge_unique (list (rdf_extract_vars expr) (list (rdf_var_symbol var_expr))))
		'("__values__" var_expr _vals) (list (rdf_var_symbol var_expr))
		'("__values_tuple__" vars _rows) (map vars rdf_var_symbol)
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
(define rdf_template_triple_expr (lambda (triple ctx blank_prefix blank_index)
	(match triple '(s p o)
		(match o
			'("__template_bnode__" nested_p nested_o)
			(list
				(list (quote lambda) (list (quote __rdf_template_bn))
					(list (quote list)
						(list (quote list) (rdf_replace_ctx s ctx) (rdf_replace_ctx p ctx)
							(quote __rdf_template_bn))
						(list (quote list) (quote __rdf_template_bn)
							(rdf_replace_ctx nested_p ctx) (rdf_replace_ctx nested_o ctx))))
				(list (quote concat) blank_prefix (quote __rdf_update_row) ":" blank_index))
			_ (list (quote list)
				(list (quote list) (rdf_replace_ctx s ctx) (rdf_replace_ctx p ctx)
					(rdf_replace_ctx o ctx)))))
))
(define rdf_template_expr (lambda (triples ctx blank_prefix)
	(if (equal? triples '()) (list (quote quote) '())
		(list (quote merge) (cons (quote list) (mapIndex triples (lambda (blank_index triple)
			(rdf_template_triple_expr triple ctx blank_prefix blank_index))))))
))
(define rdf_describe_subject_vars (lambda (conditions)
	(reduce conditions (lambda (vars condition) (match condition
		'(subject _predicate _object)
		(match subject '('get_var var)
			(if (rdf_key_in_list vars subject) vars (append vars subject))
			vars)
		_ vars)) '())
))
(define rdf_describe_query (lambda (targets conditions order limit offset)
	(begin
		(define effective_targets (if (equal? targets "__describe_all__")
			(rdf_describe_subject_vars conditions) targets))
		(define p (list (quote get_var) (symbol "?__describe_p")))
		(define o (list (quote get_var) (symbol "?__describe_o")))
		(define descriptions (map effective_targets (lambda (target)
			(list (list target p o)))))
		(define describe_pattern (if (equal? descriptions '())
			(list (list "__filter__" false))
			(if (equal? (count descriptions) 1) (car descriptions)
				(list (list "__union__" descriptions)))))
		(define primary (if (equal? effective_targets '()) nil (car effective_targets)))
		(list "select" (list "?__describe_subject" primary
			"?__describe_p" p "?__describe_o" o)
			"where" (merge (list conditions describe_pattern))
			"group" '() "having" nil "order" order "limit" limit "offset" offset
			"distinct" true)))
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
	/* Session key iteration is intentionally unordered. Canonicalize the path
	result before exposing it as an array-backed planner relation so ORDER BY is
	deterministic even when the table-function source is lowered directly. */
	(sort (seen) (lambda (left right) (< left right)))
)))
(define rdf_table_migrations (coalesce rdf_table_migrations (newsession)))
(define rdf_migrate_legacy_type_predicates (lambda (rdf_table) (begin
	(define replacements (newsession))
	(define replacement_count (newsession))
	(replacement_count "value" 0)
	(scan nil rdf_table '() '() '() (lambda () true) '("s" "p" "o")
		(lambda (acc s p o) (begin
			(if (equal? p "a")
				(begin
					(replacements (replacement_count "value")
						(list s "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" o))
					(replacement_count "value" (+ (replacement_count "value") 1))) nil)
			acc)))
	(if (> (replacement_count "value") 0)
		(begin
			/* Insert first so an interruption can never discard the only form of a
			legacy type statement. The triple key makes this idempotent when both
			spellings already exist. */
			(insert rdf_table '("s" "p" "o")
				(map (produceN (replacement_count "value")) (lambda (idx) (replacements idx)))
				'() (lambda () true))
			(scan nil rdf_table '() '() '() (lambda () true) '("p" "$update")
				(lambda (acc p $update) (begin
					(if (equal? p "a") ($update) nil)
					acc)))) nil)
	true)
)))
(define rdf_ensure_table (lambda (schema)
	(begin
		/* Avoid replaying idempotent DDL on every RDF read. Besides being wasted
		work, concurrent CREATE IF NOT EXISTS requests can publish a fresh table
		generation while a point/index plan is being compiled. */
		(if (table schema "rdf") true
			(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf (s TEXT, p TEXT, o TEXT, UNIQUE KEY rdf_spo (s, p, o))" (lambda (schema tblname write) true))))
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
		(define rdf_table (table schema "rdf"))
		/* Key the one-time migration by the table handle rather than the schema
		name, so DROP/CREATE receives a fresh migration pass in the same process. */
		(rdf_table_migrations "get_or_compute_scoped" rdf_table "legacy-type-predicate"
			(lambda () (rdf_migrate_legacy_type_predicates rdf_table)))
		true)
))
(define rdf_ensure_named_table (lambda (schema)
	(begin
		(if (table schema "rdf_named") true
			(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf_named (g TEXT, s TEXT, p TEXT, o TEXT, UNIQUE KEY rdf_gspo (g, s, p, o))" (lambda (schema tblname write) true))))
		(if (table schema "rdf_graphs") true
			(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf_graphs (g TEXT, UNIQUE KEY rdf_graph_name (g))" (lambda (schema tblname write) true)))))
))
(define rdf_graph_exists (lambda (schema graph)
	(begin
		(rdf_ensure_named_table schema)
		(define found (newsession))
		(found "value" false)
		(scan nil (table schema "rdf_graphs")
			(list 369436175368192 (scan_boundary "equal" "g" 0 0 true true "" false))
			(list graph) '() (lambda () true) '("g")
			(lambda (acc _g) (begin (found "value" true) acc)))
		(found "value"))
))
(define rdf_register_graph (lambda (schema graph)
	(begin
		(rdf_ensure_named_table schema)
		(if (rdf_graph_exists schema graph) nil
			(insert (table schema "rdf_graphs") '("g") (list (list graph)) '() (lambda () true))))
))
(define rdf_create_graph_entry (lambda (schema graph silent)
	(if (rdf_graph_exists schema graph)
		(if silent nil (error "SPARQL CREATE: graph already exists " graph))
		(rdf_register_graph schema graph))
))
(define rdf_unregister_graph (lambda (schema graph)
	(begin
		(rdf_ensure_named_table schema)
		(scan nil (table schema "rdf_graphs")
			(list 369436175368192 (scan_boundary "equal" "g" 0 0 true true "" false))
			(list graph) '() (lambda () true) '("$update")
			(lambda (acc $update) (begin ($update) acc)))
		nil)
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
		(rdf_register_graph schema graph)
		(if (equal? triples '()) nil
			(insert (table schema "rdf_named") '("g" "s" "p" "o")
				(map triples (lambda (triple) (cons graph triple))) '() (lambda () true))))
))
(define rdf_clear_default_data (lambda (schema)
	(begin
		(rdf_ensure_table schema)
		(scan nil (table schema "rdf") '() '() '() (lambda () true) '("$update")
			(lambda (acc $update) (begin ($update) acc)))
		nil)
))
(define rdf_clear_all_named_data (lambda (schema drop_graphs)
	(begin
		(rdf_ensure_named_table schema)
		(scan nil (table schema "rdf_named") '() '() '() (lambda () true) '("$update")
			(lambda (acc $update) (begin ($update) acc)))
		(if drop_graphs
			(scan nil (table schema "rdf_graphs") '() '() '() (lambda () true) '("$update")
				(lambda (acc $update) (begin ($update) acc))) nil)
		nil)
))
(define rdf_graph_triples (lambda (schema graph)
	(begin
		(define rows (newsession))
		(define count_state (newsession))
		(count_state "n" 0)
		(if (equal? graph "__rdf_default_graph__")
			(begin
				(rdf_ensure_table schema)
				(scan nil (table schema "rdf") '() '() '() (lambda () true) '("s" "p" "o")
					(lambda (acc s p o) (begin
						(rows (count_state "n") (list s p o))
						(count_state "n" (+ (count_state "n") 1)) acc))))
			(begin
				(rdf_ensure_named_table schema)
				(scan nil (table schema "rdf_named")
					(list 369436175368192 (scan_boundary "equal" "g" 0 0 true true "" false))
					(list graph) '() (lambda () true) '("s" "p" "o")
					(lambda (acc s p o) (begin
						(rows (count_state "n") (list s p o))
						(count_state "n" (+ (count_state "n") 1)) acc)))))
		(map (produceN (count_state "n")) (lambda (idx) (rows idx))))
))
(define rdf_insert_graph_target (lambda (schema graph triples)
	(if (equal? graph "__rdf_default_graph__")
		(begin (rdf_ensure_table schema) (rdf_insert_triples schema triples))
		(rdf_insert_graph_triples schema graph triples))
))
(define rdf_delete_graph_target (lambda (schema graph triples)
	(if (equal? graph "__rdf_default_graph__")
		(begin (rdf_ensure_table schema) (rdf_delete_triples schema triples))
		(rdf_delete_graph_triples schema graph triples))
))
(define rdf_clear_graph_target (lambda (schema graph drop_graph)
	(if (equal? graph "__rdf_default_graph__")
		(rdf_clear_default_data schema)
		(begin (rdf_clear_graph_data schema graph)
			(if drop_graph (rdf_unregister_graph schema graph) nil)))
))
(define rdf_transfer_graph (lambda (schema operation source target)
	(begin
		(define triples (rdf_graph_triples schema source))
		(if (equal? operation "ADD") nil (rdf_clear_graph_target schema target false))
		(rdf_insert_graph_target schema target triples)
		(if (equal? operation "MOVE") (rdf_clear_graph_target schema source true) nil)
		nil)
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
		(if found value nil)
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
		true (list bindings (cons (list (quote equal??) column true) filters))
		false (list bindings (cons (list (quote equal??) column false) filters))
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
		'("__rdf_agg__" "COUNT_DISTINCT" inner _)
		(list (quote count_distinct) (rdf_shared_expr inner bindings outer_ctx))
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
		'("__rdf_agg__" "SAMPLE" inner _)
		(list (quote aggregate) (rdf_shared_expr inner bindings outer_ctx)
			(quote rdf_sample_reduce) nil)
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
		(cons head tail) (cons (if (equal? head (quote equal?)) (quote equal??)
			(if (equal? head (quote /)) (quote rdf_divide) head))
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
(define rdf_assoc_position (lambda (assoc key index)
	(match assoc
		(cons title (cons _value tail))
		(if (rdf_key_equal title key) index (rdf_assoc_position tail key (+ index 1)))
		'() nil)
))
(define rdf_shared_union_output_order (lambda (order cols bindings outer_ctx)
	(if (nil? order) nil
		(map order (lambda (entry) (match entry '(expr dir)
			(begin
				(define resolved (match expr
					'('get_var var) (if (nil? (rdf_assoc_position cols (concat var) 1))
						(rdf_shared_expr expr bindings outer_ctx)
						(rdf_assoc_position cols (concat var) 1))
					_ (rdf_shared_expr expr bindings outer_ctx)))
				(list resolved (if (equal? dir "DESC") > <))))))
	)))
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
		'("__values_tuple__" _vars _rows) false
		'("__optional__" _inner) false
		'("__group__" _inner) false
		'("__minus__" _inner) false
		'("__service__" _silent _endpoint _inner) false
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
(define rdf_shared_fresh_path_var (lambda (state)
	(begin
		(define idx (state "next"))
		(state "next" (+ idx 1))
		(define candidate (symbol (concat "?__rdf_path_" idx)))
		(if (rdf_key_in_list (state "used") candidate)
			(rdf_shared_fresh_path_var state)
			(begin
				(state "used" (cons candidate (state "used")))
				(list (quote get_var) candidate))))
))
(define rdf_path_invert (lambda (path)
	(match path
		'("__path_inverse__" inner) inner
		'("__path_seq__" left right)
		(list "__path_seq__" (rdf_path_invert right) (rdf_path_invert left))
		'("__path_alt__" left right)
		(list "__path_alt__" (rdf_path_invert left) (rdf_path_invert right))
		'("__path_star__" inner) (list "__path_star__" (rdf_path_invert inner))
		'("__path_plus__" inner) (list "__path_plus__" (rdf_path_invert inner))
		'("__path_optional__" inner) (list "__path_optional__" (rdf_path_invert inner))
		path)
))
(define rdf_path_zero_branch (lambda (subject object)
	(match object
		'('get_var _object_var) (list (list "__bind__" subject object))
		_ (match subject
			'('get_var _subject_var) (list (list "__bind__" object subject))
			_ (list (list "__filter__" (list (quote equal?) subject object)))))
))
(define rdf_path_negated_branch (lambda (subject object direction exclusions state)
	(begin
		(define predicate (rdf_shared_fresh_path_var state))
		(list
			(if (equal? direction "inverse")
				(list object predicate subject) (list subject predicate object))
			(list "__filter__" (list (quote not)
				(cons (quote sql_in) (cons (cons (quote list) exclusions)
					(list predicate)))))))
))
(define rdf_path_negated_exclusions (lambda (members direction)
	(match members
		(cons member tail)
		(if (equal? (car member) direction)
			(cons (cadr member) (rdf_path_negated_exclusions tail direction))
			(rdf_path_negated_exclusions tail direction))
		'() '())
))
(define rdf_path_negated_branches (lambda (subject object members state)
	(begin
		(define forward (rdf_path_negated_exclusions members "forward"))
		(define inverse (rdf_path_negated_exclusions members "inverse"))
		(if (equal? forward '())
			(if (equal? inverse '()) '()
				(list (rdf_path_negated_branch subject object "inverse" inverse state)))
			(if (equal? inverse '())
				(list (rdf_path_negated_branch subject object "forward" forward state))
				(list (rdf_path_negated_branch subject object "forward" forward state)
					(rdf_path_negated_branch subject object "inverse" inverse state))))
))
))
(define rdf_shared_expand_paths_using (lambda (conditions state) (match conditions
	(cons condition tail)
	(match condition
		'("__empty_pattern__" triple)
		(rdf_shared_expand_paths_using (cons triple
			(cons (list "__filter__" false) tail)) state)
		'(s p o)
		(match p
			'("__path_negated__" members)
			(begin
				(define branches (rdf_path_negated_branches s o members state))
				(if (equal? (count branches) 1)
					(rdf_shared_expand_paths_using
						(cons (car (car branches))
							(cons (cadr (car branches)) tail)) state)
					(cons (list "__union__" branches)
						(rdf_shared_expand_paths_using tail state))))
			'("__path_inverse__" inner)
			(rdf_shared_expand_paths_using
				(cons (list o (rdf_path_invert inner) s) tail) state)
			'("__path_seq__" p1 p2)
			(begin
				(define intermediate (rdf_shared_fresh_path_var state))
				(rdf_shared_expand_paths_using
					(cons (list s p1 intermediate) (cons (list intermediate p2 o) tail)) state))
			'("__path_alt__" p1 p2)
			(cons (list "__union__" (list (list (list s p1 o)) (list (list s p2 o))))
				(rdf_shared_expand_paths_using tail state))
			'("__path_optional__" inner)
			(cons (list "__union__" (list (list (list s inner o))
				(rdf_path_zero_branch s o)))
				(rdf_shared_expand_paths_using tail state))
			_ (cons condition (rdf_shared_expand_paths_using tail state)))
		_ (cons condition (rdf_shared_expand_paths_using tail state)))
	'() '()
)))
(define rdf_shared_expand_paths (lambda (conditions)
	(begin
		(define state (newsession))
		(state "next" 0)
		(state "used" (rdf_condition_vars conditions))
		(rdf_shared_expand_paths_using conditions state))
))
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
(define rdf_shared_add_last_source_join (lambda (sources extra)
	(match sources
		(cons source '())
		(list (list (source_alias source) (source_schema source) (source_relation source)
			(source_outer? source)
			(rdf_shared_where (filter (list (source_join_expr source) extra)
				(lambda (expr) (not (nil? expr)))))))
		(cons source tail) (cons source (rdf_shared_add_last_source_join tail extra))
		'() '())
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
(define rdf_shared_tuple_values_fields (lambda (vars row)
	(match vars
		(cons var tail_vars)
		(match row
			(cons value tail_values)
			(cons (rdf_shared_input_field_name (rdf_var_symbol var))
				(cons (if (rdf_unbound_expr? value) nil value)
					(rdf_shared_tuple_values_fields tail_vars tail_values)))
			'() '())
		'() '())
))
(define rdf_shared_json_records (lambda (records)
	(match records
		(cons record rest) (concat (json_encode_assoc record)
			(if (equal? rest '()) "" (concat "," (rdf_shared_json_records rest))))
		'() "")
))
(define rdf_shared_tuple_values_relation (lambda (schema vars rows)
	(begin
		(define relation_vars (map vars rdf_var_symbol))
		(define columns (map relation_vars rdf_shared_input_field_name))
		(define records (map rows (lambda (row) (rdf_shared_tuple_values_fields vars row))))
		(define alias "__rdf_tuple_values")
		(define relation (list (quote table-function) "recordset"
			(list (concat "[" (concat (rdf_shared_json_records records) "]"))
				(list (quote quote) columns)) columns))
		(list (make_query_block schema
			(list (list alias schema relation false nil))
			(reduce columns (lambda (fields column)
				(append fields column (rdf_shared_column alias column))) '())
			true nil nil nil nil nil '() '() '())
			relation_vars))
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
(define rdf_shared_optional_union_branch (lambda (schema left_query left_vars right_query right_vars filters index)
	(begin
		(define left_alias (concat "__rdf_optional_left" index))
		(define right_alias (concat "__rdf_optional_right" index))
		(define left_bindings (rdf_shared_relation_refs left_alias left_vars))
		(define right_bindings (rdf_shared_relation_refs right_alias right_vars))
		(define bindings (rdf_shared_merge_bindings left_bindings right_bindings))
		(define joins (merge (list (rdf_shared_join_filters left_bindings right_bindings)
			(rdf_shared_filter_conditions filters bindings '()))))
		(make_query_block schema
			(list
				(list left_alias schema left_query false nil)
				(list right_alias schema right_query false (rdf_shared_where joins)))
			(rdf_shared_relation_fields bindings) true nil nil nil nil nil '() '() '()))
))
(define rdf_shared_optional_exists_branch (lambda (right_query right_vars left_bindings filters)
	(begin
		(define right_bindings (reduce right_vars (lambda (bindings var)
			(append bindings var (get_assoc (qb_fields right_query) (rdf_shared_input_field_name var)))) '()))
		(define bindings (rdf_shared_merge_bindings left_bindings right_bindings))
		(define predicates (merge (list
			(rdf_shared_join_filters left_bindings right_bindings)
			(rdf_shared_filter_conditions filters bindings '()))))
		(make_query_block (qb_schema right_query) (qb_sources right_query) (qb_fields right_query)
			(rdf_shared_where (cons (qb_where right_query) predicates))
			(qb_group right_query) (qb_having right_query) (qb_order right_query)
			(qb_limit right_query) (qb_offset right_query) (qb_hidden right_query)
			(qb_stages right_query) (qb_facts right_query)))
))
(define rdf_shared_optional_union_relation (lambda (schema state inner filters union_query right_vars)
	(begin
		(define left_vars (rdf_shared_relation_vars (rdf_shared_state_bindings state)))
		(define all_vars (merge_unique (list left_vars right_vars)))
		(define left_query (rdf_shared_relation_query schema state))
		(define matched (map (produceN (count (union_branches union_query))) (lambda (index)
			(rdf_shared_optional_union_branch schema left_query left_vars
				(nth (union_branches union_query) index) right_vars filters index))))
		(define unmatched_alias "__rdf_optional_unmatched")
		(define unmatched_left (rdf_shared_relation_refs unmatched_alias left_vars))
		(define unmatched_fields (reduce all_vars (lambda (fields var)
			(append fields (rdf_shared_input_field_name var)
				(coalesceNil (get_assoc unmatched_left var) nil))) '()))
		(define exists_query (make_union_block (quote all)
			(map (union_branches union_query) (lambda (branch)
				(rdf_shared_optional_exists_branch branch right_vars unmatched_left filters)))
			nil nil nil '()))
		(define unmatched (make_query_block schema
			(list (list unmatched_alias schema left_query false nil)) unmatched_fields
			(list (quote not) (list (quote inner_select_exists) exists_query))
			nil nil nil nil nil '() '() '()))
		(list (make_union_block (quote all) (append matched unmatched) nil nil nil '()) all_vars))
))
(define rdf_shared_minus_relation (lambda (schema state query vars shared)
	(begin
		/* Model MINUS as an anti-join against distinct compatible keys. The
		grouped relation is a semantic boundary: right-side multiplicity cannot
		duplicate left mappings, and its COUNT column is an explicit presence
		marker after a LEFT JOIN. */
		(define index (rdf_shared_state_index state))
		(define input_alias (concat "__rdf_minus_input" index))
		(define alias (concat "__rdf_minus" index))
		(define presence "__rdf_minus_present")
		(define input_bindings (rdf_shared_relation_refs input_alias vars))
		(define grouped_bindings (reduce shared (lambda (bindings var)
			(append bindings var (get_assoc input_bindings var))) '()))
		(define grouped_fields (append (rdf_shared_relation_fields grouped_bindings)
			presence (list (quote aggregate) 1 (quote +) 0)))
		(define relation (make_query_block schema
			(list (list input_alias schema query false nil)) grouped_fields true
			(extract_assoc grouped_bindings (lambda (_var expr) expr)) nil nil nil nil '() '() '()))
		(define right (rdf_shared_relation_refs alias shared))
		(define joins (reduce shared (lambda (conditions var)
			(match (rdf_ctx_lookup (rdf_shared_state_bindings state) var) '(left_found left_expr)
				(match (rdf_ctx_lookup right var) '(right_found right_expr)
					(if (and left_found right_found)
						(merge (list conditions
							(list (list (quote equal??) left_expr right_expr)
								(list (quote not) (list (quote nil?) left_expr))
								(list (quote not) (list (quote nil?) right_expr)))))
						conditions)
					'() conditions)
				'() conditions)) '()))
		(list
			(append (rdf_shared_state_sources state)
				(list alias schema relation true (rdf_shared_where joins)))
			(rdf_shared_state_bindings state)
			(cons (list (quote nil?) (rdf_shared_column alias presence))
				(rdf_shared_state_filters state))
			(+ index 1)))
))
(define rdf_shared_exists_relation (lambda (schema state query vars shared negate)
	(begin
		/* EXISTS is a semi-join and NOT EXISTS is its anti-join counterpart.
		Group the right mappings by their shared domain so neither form changes
		left multiplicity, then leave physical join choice to the common planner. */
		(define index (rdf_shared_state_index state))
		(define input_alias (concat "__rdf_exists_input" index))
		(define alias (concat "__rdf_exists" index))
		(define presence "__rdf_exists_present")
		(define flat_input (and (query_block? query)
			(equal? (count (filter (qb_sources query) source_is_base_table?))
				(count (qb_sources query)))))
		(define input_bindings (if flat_input
			(reduce vars (lambda (bindings var)
				(append bindings var (get_assoc (qb_fields query)
					(rdf_shared_input_field_name var)))) '())
			(rdf_shared_relation_refs input_alias vars)))
		(define grouped_bindings (reduce shared (lambda (bindings var)
			(append bindings var (get_assoc input_bindings var))) '()))
		(define relation (make_query_block schema
			(if flat_input (qb_sources query)
				(list (list input_alias schema query false nil)))
			(append (rdf_shared_relation_fields grouped_bindings)
				presence (list (quote aggregate) 1 (quote +) 0))
			(if flat_input (qb_where query) true) (if (equal? shared '()) nil
				(extract_assoc grouped_bindings (lambda (_var expr) expr)))
			nil nil nil nil '() '() '()))
		(define right (rdf_shared_relation_refs alias shared))
		(define joins (rdf_shared_join_filters (rdf_shared_state_bindings state) right))
		(define presence_expr (rdf_shared_column alias presence))
		(if (equal? shared '())
			(list
				(append (rdf_shared_state_sources state)
					(list alias schema relation false nil))
				(rdf_shared_state_bindings state)
				(cons (if negate
					(list (quote equal?) presence_expr 0)
					(list (quote >) presence_expr 0))
					(rdf_shared_state_filters state))
				(+ index 1))
			(if negate
				(list
					(append (rdf_shared_state_sources state)
						(list alias schema relation true (rdf_shared_where joins)))
					(rdf_shared_state_bindings state)
					(cons (list (quote nil?) presence_expr)
						(rdf_shared_state_filters state))
					(+ index 1))
				(list
					(append (rdf_shared_state_sources state)
						(list alias schema relation true (rdf_shared_where joins)))
					(rdf_shared_state_bindings state)
					(cons (list (quote not) (list (quote nil?) presence_expr))
						(rdf_shared_state_filters state))
					(+ index 1))))
)))
(define rdf_shared_exists_direct_safe (lambda (query vars shared)
	(and (not (equal? shared '()))
		(and (equal? (count vars) (count shared))
			(and (equal? (count (filter vars (lambda (var) (rdf_key_in_list shared var))))
				(count vars))
				(and (query_block? query)
					(equal? (count (filter (qb_sources query) source_is_base_table?))
						(count (qb_sources query)))))))
))
(define rdf_shared_exists_direct_relation (lambda (schema state query vars shared negate)
	(begin
		(define index (rdf_shared_state_index state))
		(define alias (concat "__rdf_exists_direct" index))
		(define right (rdf_shared_relation_refs alias vars))
		(define joins (rdf_shared_join_filters (rdf_shared_state_bindings state) right))
		(define flattened_right (reduce vars (lambda (bindings var)
			(append bindings var (get_assoc (qb_fields query) (rdf_shared_input_field_name var)))) '()))
		(define flattened_joins (rdf_shared_join_filters
			(rdf_shared_state_bindings state) flattened_right))
		(if negate
			(if (equal? (count (qb_sources query)) 1)
				(begin
					/* Keep a one-pattern anti-join in the same flat join graph. This
					avoids an unnecessary derived carrier while retaining the query
					planner's normal costing and source reordering. */
					(define source (car (qb_sources query)))
					(list
						(append (rdf_shared_state_sources state)
							(list (source_alias source) (source_schema source)
								(source_relation source) true
								(rdf_shared_where (merge (list
									(list (qb_where query)) flattened_joins)))))
						(rdf_shared_state_bindings state)
						(cons (list (quote nil?) (get_assoc flattened_right (car shared)))
							(rdf_shared_state_filters state))
						(+ index 1)))
				(list
					(append (rdf_shared_state_sources state)
						(list alias schema query true (rdf_shared_where joins)))
					(rdf_shared_state_bindings state)
					(cons (list (quote nil?) (get_assoc right (car shared)))
						(rdf_shared_state_filters state))
					(+ index 1)))
			/* A set-unique positive operand needs no cardinality barrier. Flatten
				its base sources into the current BGP so the common planner can cost
				and reorder the complete semi-join as one join graph. */
			(list
				(merge (list (rdf_shared_state_sources state) (qb_sources query)))
				(rdf_shared_state_bindings state)
				(merge (list (rdf_shared_state_filters state)
					(list (qb_where query)) flattened_joins))
				(+ index 1))))
))
(define rdf_shared_exists_apply_relation (lambda (schema state query vars shared negate)
	(if (rdf_shared_exists_direct_safe query vars shared)
		(rdf_shared_exists_direct_relation schema state query vars shared negate)
		(rdf_shared_exists_relation schema state query vars shared negate))
))
(define rdf_shared_rewrite_source_alias (lambda (expr old_alias new_alias)
	(match expr
		((symbol get_column) alias table_icase column column_icase)
		(if (equal? alias old_alias)
			(list (quote get_column) new_alias table_icase column column_icase) expr)
		((quote get_column) alias table_icase column column_icase)
		(if (equal? alias old_alias)
			(list (quote get_column) new_alias table_icase column column_icase) expr)
		(cons head tail) (cons (rdf_shared_rewrite_source_alias head old_alias new_alias)
			(map tail (lambda (item) (rdf_shared_rewrite_source_alias item old_alias new_alias))))
		expr)
))
(define rdf_shared_exists_union_relation (lambda (schema state branches negate)
	(begin
		(define apply_branch (lambda (input branch branch_negate)
			(match (rdf_shared_conditions_relation schema branch '()) '(query vars)
				(begin
					(define shared (filter vars (lambda (var)
						(rdf_ctx_bound (rdf_shared_state_bindings input) var))))
					(rdf_shared_exists_apply_relation schema input query vars shared branch_negate)))))
		(define candidates (map branches (lambda (branch)
			(match (rdf_shared_conditions_relation schema branch '()) '(query vars)
				(begin
					(define shared (filter vars (lambda (var)
						(rdf_ctx_bound (rdf_shared_state_bindings state) var))))
					(list query vars shared))))))
		(define direct_candidates (filter candidates (lambda (candidate)
			(match candidate '(query vars shared)
				(and (rdf_shared_exists_direct_safe query vars shared)
					(equal? (count (qb_sources query)) 1))))))
		(if (equal? (count direct_candidates) (count candidates))
			(begin
				(define first_candidate (car candidates))
				(define first_query (nth first_candidate 0))
				(define first_vars (nth first_candidate 1))
				(define first_src (car (qb_sources first_query)))
				(define probe_alias (source_alias first_src))
				(define alternatives (map candidates (lambda (candidate)
					(begin
						(define candidate_query (nth candidate 0))
						(define candidate_src (car (qb_sources candidate_query)))
						(rdf_shared_rewrite_source_alias (qb_where candidate_query)
							(source_alias candidate_src) probe_alias)))))
				(define right (reduce first_vars (lambda (bindings var)
					(append bindings var (get_assoc (qb_fields first_query)
						(rdf_shared_input_field_name var)))) '()))
				(define joins (rdf_shared_join_filters (rdf_shared_state_bindings state) right))
				(define match_expr (if (equal? (count alternatives) 1) (car alternatives)
					(cons (quote or) alternatives)))
				(if negate
					(list (append (rdf_shared_state_sources state)
						(list probe_alias (source_schema first_src) (source_relation first_src) true
							(rdf_shared_where (merge (list (list match_expr) joins)))))
						(rdf_shared_state_bindings state)
						(cons (list (quote nil?) (get_assoc right (car (nth first_candidate 2))))
							(rdf_shared_state_filters state))
						(+ (rdf_shared_state_index state) 1))
					(list (append (rdf_shared_state_sources state)
						(list probe_alias (source_schema first_src) (source_relation first_src) false nil))
						(rdf_shared_state_bindings state)
						(merge (list (rdf_shared_state_filters state) (list match_expr) joins))
						(+ (rdf_shared_state_index state) 1))))
			(if negate
			/* NOT EXISTS(A UNION B) = NOT EXISTS(A) AND NOT EXISTS(B). */
			(reduce branches (lambda (current branch)
				(apply_branch current branch true)) state)
			(begin
				/* EXISTS projects a UNION back onto the current mapping domain. */
				(define queries (map branches (lambda (branch)
					(rdf_shared_relation_query schema (apply_branch state branch false)))))
				(define vars (rdf_shared_relation_vars (rdf_shared_state_bindings state)))
				(define union_query (make_union_block (quote union_distinct) queries nil nil nil '()))
				(rdf_shared_attach_relation schema
					(list '() '() '() (rdf_shared_state_index state))
					union_query vars false)))))
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
			(if (rdf_ctx_bound (rdf_shared_state_bindings state) var)
				(error "SPARQL BIND cannot rebind variable " var)
				(list (rdf_shared_state_sources state)
					(append (rdf_shared_state_bindings state) var
						(rdf_shared_expr expr (rdf_shared_state_bindings state) outer_ctx))
					(rdf_shared_state_filters state) (rdf_shared_state_index state))))
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
		'("__values_tuple__" vars rows)
		(match (rdf_shared_tuple_values_relation schema vars rows) '(query relation_vars)
			(rdf_shared_attach_relation schema state query relation_vars false))
			'("__optional__" inner)
			(begin
				(define inner_filters (filter inner rdf_shared_filter_condition?))
				(define inner_relation_conditions (filter inner
					(lambda (condition) (not (rdf_shared_filter_condition? condition)))))
				(match (rdf_shared_conditions_relation schema inner_relation_conditions '()) '(query vars)
					(if (union_block? query)
						(match (rdf_shared_optional_union_relation schema state
							inner_relation_conditions inner_filters query vars) '(optional_query optional_vars)
							(rdf_shared_attach_relation schema (list '() '() '() (rdf_shared_state_index state))
								optional_query optional_vars false))
						(begin
						(define attached (rdf_shared_attach_relation schema state query vars true))
						(define optional_filter (rdf_shared_where
							(rdf_shared_filter_conditions inner_filters
								(rdf_shared_state_bindings attached) outer_ctx)))
					(list (rdf_shared_add_last_source_join
							(rdf_shared_state_sources attached) optional_filter)
							(rdf_shared_state_bindings attached)
							(rdf_shared_state_filters attached)
							(rdf_shared_state_index attached))))))
		'("__group__" inner)
		(match (rdf_shared_conditions_relation schema inner
			(rdf_shared_merge_bindings outer_ctx (rdf_shared_state_bindings state))) '(query vars)
			(rdf_shared_attach_relation schema state query vars false))
		'("__minus__" inner)
		(begin
			(define shared (filter (rdf_condition_vars inner) (lambda (var)
				(rdf_ctx_bound (rdf_shared_state_bindings state) var))))
			(if (equal? shared '()) state
				/* MINUS evaluates its right group independently, then removes
				compatible mappings. Keep its distinct-key relation as an explicit
				semantic boundary: unlike NOT EXISTS, MINUS compatibility is defined
				over the shared mapping domain and must not be flattened into a
				one-pattern physical anti-join. */
				(match (rdf_shared_conditions_relation schema inner '()) '(query vars)
					(rdf_shared_minus_relation schema state query vars shared))))
		'("__service__" silent endpoint _inner)
		(if silent state (error "SPARQL SERVICE endpoint unavailable: " endpoint))
		'("__union__" branches)
		(match (rdf_shared_union_relation schema branches
			(rdf_shared_merge_bindings outer_ctx (rdf_shared_state_bindings state))
			(quote all)) '(query vars)
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
		(match inner
			(list (list "__union__" branches))
			(rdf_shared_exists_union_relation schema state branches negate)
			/* Build the EXISTS operand as an independent relation. Correlation is
				represented explicitly by the semi/anti-join below; carrying outer
				source references into a derived table would bypass normal name binding
				and prevents the common planner from reordering the join. */
			(match (rdf_shared_conditions_relation schema inner '()) '(query vars)
				(begin
					(define shared (filter vars (lambda (var)
						(rdf_ctx_bound (rdf_shared_state_bindings state) var))))
					(rdf_shared_exists_apply_relation schema state query vars shared negate))))
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
					(define result_order (if (union_block? input_query)
						(rdf_shared_union_output_order order cols bindings outer_ctx)
						(rdf_shared_order order bindings outer_ctx)))
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
								result_order
							limit offset '() '()
							(if (> (count (coalesceNil result_order '())) 0)
								(list (list (quote global_order_required) true)) '())))
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
								result_order limit offset '() '()
							(merge (list
								(if distinct (list (list (quote select_distinct) true)) '())
								(if (> (count (coalesceNil result_order '())) 0)
									(list (list (quote global_order_required) true)) '()))))))))
		(error "SPARQL shared planner: expected SELECT query")
	)
))
(define rdf_shared_result_context (lambda (cols outer_ctx row_symbol)
	(match cols
		(cons title (cons _expr tail))
		(merge (rdf_shared_result_context tail outer_ctx row_symbol)
			(list title (list (quote rdf_row_lookup) row_symbol title)))
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
		/* RDFHP can nest query plans. Fixed callback parameter names let an inner
		plan shadow expressions captured from the outer row, producing
		__rdf_row_missing__ for otherwise bound variables. Derive hygienic names
		from the logical query and its outer context. */
		(define scope_id (fnv_hash (concat query "|" outer_ctx)))
		(define values_symbol (symbol (concat "__rdf_values_" scope_id)))
		(define outer_resultrow_symbol (symbol (concat "__rdf_outer_resultrow_" scope_id)))
		(define result_ctx (rdf_shared_result_context (qb_fields ast) outer_ctx values_symbol))
		(define result_body (resultfunc (nth query 1) result_ctx))
		(list
			(list (quote lambda) (list outer_resultrow_symbol)
				(list (quote begin)
					(list (quote set) (quote resultrow)
						(list (quote lambda) (list values_symbol)
							(list
								(list (quote lambda) (list (quote resultrow)) result_body)
								outer_resultrow_symbol)))
					plan))
			(quote resultrow)))
))

(define rdf_queryplan (lambda (schema query definitions ctx resultfunc /* function that gets cols + ctx */)
	(rdf_shared_queryplan schema (rdf_resolve_prefixes query definitions) ctx resultfunc)
))

(define rdf_update_select_query (lambda (conditions)
	(list "select" '() "where" conditions "group" '() "having" nil
		"order" nil "limit" nil "offset" nil "distinct" nil)
))

/* Compile by the explicit operation tag. Keeping update/query-form dispatch out of
the parser's large structural match also avoids optimizer aliasing between match
bindings of neighbouring update alternatives. */
(define rdf_compile_sparql_ast (lambda (schema parsed definitions planning_session tx)
	(begin
		(define kind (car parsed))
		(if (equal? kind "update_request")
			(cons (quote begin) (map (nth parsed 1) (lambda (operation)
				(rdf_compile_sparql_ast schema operation definitions planning_session tx))))
		(if (equal? kind "select")
			(begin
				(define cols (nth parsed 1))
				(define qgroup (nth parsed 5))
				(define qhaving (nth parsed 7))
				(if (or (rdf_select_has_aggregates cols)
					(or (rdf_has_aggregate qhaving) (not (equal? qgroup '()))))
					(rdf_queryplan schema parsed definitions '() rdf_shared_resultrow_ast)
					(rdf_queryplan schema parsed definitions '() rdf_select_resultrow_ast)))
		(if (equal? kind "create_graph")
			(list (quote rdf_create_graph_entry) schema (nth parsed 1) (nth parsed 2))
		(if (equal? kind "clear_graph")
			(begin
				(define target (nth parsed 1))
				(if (equal? target "default") (list (quote rdf_clear_default_data) schema)
				(if (equal? target "named") (list (quote rdf_clear_all_named_data) schema false)
				(if (equal? target "all") (list (quote begin)
					(list (quote rdf_clear_default_data) schema)
					(list (quote rdf_clear_all_named_data) schema false))
					(list (quote rdf_clear_graph_target) schema target false)))))
		(if (equal? kind "drop_graph")
			(begin
				(define target (nth parsed 1))
				(if (equal? target "default") (list (quote rdf_clear_default_data) schema)
				(if (equal? target "named") (list (quote rdf_clear_all_named_data) schema true)
				(if (equal? target "all") (list (quote begin)
					(list (quote rdf_clear_default_data) schema)
					(list (quote rdf_clear_all_named_data) schema true))
					(list (quote rdf_clear_graph_target) schema target true)))))
		(if (equal? kind "load")
			(if (nth parsed 3) (list (quote begin))
				(list (quote error) "SPARQL LOAD failed for " (nth parsed 1) (nth parsed 2)))
		(if (equal? kind "graph_transfer")
			(list (quote rdf_transfer_graph) schema (nth parsed 1) (nth parsed 2) (nth parsed 3))
		(if (equal? kind "insert_data")
			(list (quote begin) (list (quote rdf_ensure_table) schema)
				(list (quote rdf_insert_triples) schema (list (quote quote) (nth parsed 1))))
		(if (equal? kind "insert_graph_data")
			(list (quote rdf_insert_graph_triples) schema (nth parsed 1)
				(list (quote quote) (nth parsed 2)))
		(if (equal? kind "delete_data")
			(list (quote begin) (list (quote rdf_ensure_table) schema)
				(list (quote rdf_delete_triples) schema (list (quote quote) (nth parsed 1))))
		(if (equal? kind "delete_graph_data")
			(list (quote rdf_delete_graph_triples) schema (nth parsed 1)
				(list (quote quote) (nth parsed 2)))
		(if (equal? kind "ask")
			(begin
				(define ask_state (newsession))
				(ask_state "matched" false)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema (rdf_update_select_query (nth parsed 2)) definitions '()
						(lambda (_cols _ctx) (list ask_state "matched" true)))
					(list (quote resultrow) (list (quote list) "?ask" (list ask_state "matched")))))
		(if (equal? kind "construct")
			(begin
				(define triples (nth parsed 1))
				(define conditions (nth parsed 3))
				(define order (nth parsed 5))
				(define limit (nth parsed 7))
				(define offset (nth parsed 9))
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema (list "select" '() "where" conditions "group" '()
						"having" nil "order" order "limit" limit "offset" offset "distinct" nil)
						definitions '() (lambda (_cols ctx)
							(cons (quote begin) (map triples (lambda (triple) (match triple '(s p o)
								(list (quote resultrow) (list (quote list)
									"?s" (rdf_replace_ctx s ctx) "?p" (rdf_replace_ctx p ctx)
									"?o" (rdf_replace_ctx o ctx)))))))))))
		(if (equal? kind "describe")
			(begin
				(define describe_query (rdf_describe_query (nth parsed 1) (nth parsed 3)
					(nth parsed 5) (nth parsed 7) (nth parsed 9)))
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema describe_query definitions '() (lambda (_cols ctx)
						(list (quote resultrow) (list (quote list)
							"?s" (rdf_ctx_value ctx "?__describe_subject")
							"?p" (rdf_ctx_value ctx "?__describe_p")
							"?o" (rdf_ctx_value ctx "?__describe_o")))))))
		(if (equal? kind "modify")
			(begin
				(define update_graph (nth parsed 2))
				(define delete_triples (nth parsed 4))
				(define insert_triples (nth parsed 6))
				(define conditions (nth parsed 8))
				(define delete_rows (newsession))
				(define insert_rows (newsession))
				(define update_row_count (newsession))
				/* Allocate the namespace at template evaluation time. nanotime has a
				straight JIT path; row and template indexes disambiguate every label
				within an update and the persistent row counter covers plan reuse. */
				(define update_blank_prefix
					(list (quote concat) "urn:rdf-template:" (list (quote nanotime)) ":"))
				(update_row_count "value" 0)
				(list (quote begin)
					(list (quote rdf_ensure_table) schema)
					(rdf_queryplan schema (rdf_update_select_query conditions) definitions '()
						(lambda (_cols ctx) (list (quote begin)
							(list (quote set) (quote __rdf_update_row) (list update_row_count "value"))
							(list update_row_count "value" (list (quote +) (quote __rdf_update_row) 1))
							(list delete_rows (quote __rdf_update_row)
								(rdf_template_expr delete_triples ctx update_blank_prefix))
							(list insert_rows (quote __rdf_update_row)
								(rdf_template_expr insert_triples ctx update_blank_prefix)))))
					(list (quote rdf_delete_graph_target) schema update_graph
						(list (quote rdf_session_merged_values) delete_rows))
					(list (quote rdf_insert_graph_target) schema update_graph
						(list (quote rdf_session_merged_values) insert_rows))))
			(error "Unsupported SPARQL operation: " kind)
		))))))))))))))))
)))

(define parse_sparql (lambda (schema s _policy planning_session tx)
	(match (ttl_header s)
		'("prefixes" definitions "rest" rest)
		(rdf_compile_sparql_ast schema
			(rdf_expand_select_star
				(rdf_resolve_prefixes (rdf_query (rdf_strip_leading_ws_comments rest)) definitions))
			definitions planning_session tx)
	)
))


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
			(parser '((atom "a" true)) "http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
			(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "(?:[a-zA-Z0-9_]|\\\\[~._-])*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) (rdf_unescape_pname post))))
			(parser '((atom "<" true) (define iri (regex "[^>]*" false false)) (atom ">" false false)) (rdf_apply_base_iri definitions (rdf_unescape_iri iri)))
			(parser '((atom "'''" true) (define x (regex "[^']*(?:(?:'[^']|''[^'])[^']*)*" false false)) (atom "'''" false false)) (rdf_unescape_single x))
			(parser '((atom "'" true) (define x (regex "(?:[^'\\\\]|\\\\.)*" false false)) (atom "'" false false)) (rdf_unescape_single x))
			(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) x (rdf_typed_literal x datatype)))
			(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x))
			(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) (rdf_unescape x) (rdf_typed_literal (rdf_unescape x) datatype)))
			(parser '((define x (regex "[+-]?[0-9]+(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?" true))) (simplify x))
			(parser '((atom "true" true)) true)
			(parser '((atom "false" true)) false)
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
				(if (not (string? val)) val
					(match val (regex "^_:(.+)$" _ bname) (begin
					(if (nil? (_bn bname)) (_bn bname (concat "urn:uuid:" (uuid))))
					(_bn bname)
				) val)
			)
		))
		(define ttl_simple_constant (parser (or
			(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node before prefix match */
			(parser '((atom "a" true)) "http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
			(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "(?:[a-zA-Z0-9_]|\\\\[~._-])*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) (rdf_unescape_pname post)))) /* add prefix with validation */
			(parser '((atom "<" true) (define iri (regex "[^>]*" false false)) (atom ">" false false)) (rdf_apply_base_iri definitions (rdf_unescape_iri iri)))
			(parser '((atom "'''" true) (define x (regex "[^']*(?:(?:'[^']|''[^'])[^']*)*" false false)) (atom "'''" false false)) (rdf_unescape_single x))
			(parser '((atom "'" true) (define x (regex "(?:[^'\\\\]|\\\\.)*" false false)) (atom "'" false false)) (rdf_unescape_single x))
				(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) x (rdf_typed_literal x datatype)))
			(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) (rdf_unescape x))
				(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false) (? (atom "^^" false false) (define datatype rdf_datatype_suffix))) (if (nil? datatype) (rdf_unescape x) (rdf_typed_literal (rdf_unescape x) datatype)))
			(parser '((define x (regex "[+-]?[0-9]+(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?" true))) (simplify x))
			(parser '((atom "true" true)) true)
			(parser '((atom "false" true)) false)
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
