/*
Copyright (C) 2024  Carl-Philip Hänsch

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
/* datatype suffix parser: consumes ^^<IRI> or ^^prefix:name or ^^barename */
(define rdf_datatype_suffix (parser (or
	(parser '((atom "<" false false) (regex "[^>]*" false false) (atom ">" false false)) nil) /* ^^<IRI> */
	(regex "[a-zA-Z0-9_]*:[a-zA-Z0-9_]*" false false) /* ^^prefix:name */
	(regex "[a-zA-Z0-9_]+" false false) /* ^^barename */
)))
(define rdf_constant (parser (or
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x) /* IRI */
	(parser '((atom "\"\"\"" true) (define x (regex "[^\"]*(?:(?:\"[^\"]|\"\"[^\"])[^\"]*)*" false false)) (atom "\"\"\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) x) /* triple-quoted string, optional datatype ignored */
	(parser '((atom "\"" true) (define x (regex "[^\"]*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) x) /* string with language, ignore language */
	(parser '((atom "\"" true) (define x (regex "[^\"]*" false false)) (atom "\"" false false) (? (atom "^^" false false) rdf_datatype_suffix)) x) /* string, optional datatype ignored */
	(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node _:identifier */
	(regex "[a-zA-Z0-9_]+" true) /* bare name */
)))
(define rdf_expression (parser (or
	(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) '('concat '('definitions pfx) post)) /* as expression */
	rdf_variable
	rdf_constant
	/* TODO: SUM(...), COUNT(), AVG, MIN, MAX, GROUP_CONCAT */
	/* TODO: CONCAT() */
)))

/* TODO: blank nodes
[ p o ]
oder [] p o
oder _:identifier

*/

(define rdf_select (parser '(
	(atom "SELECT" true)
	(define cols (+ (or
		(parser '((define v rdf_expression) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v))) /* rdf_variable AS rdf_variable */
		(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v))) /* rdf_variable */
	) ","))
	(?
		(atom "WHERE" true)
		(atom "{" true)
		(define conditions (* (or
			(parser '((define s rdf_expression) (define ps (+ (parser '((define p rdf_expression) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
			/* TODO: FILTER regex(?var "pattern") */
			/* TODO: OPTIONAL {subquery} */
		) "."))
		(? (atom "." true))
		(atom "}" true) /* TODO: {} UNION {} */
	)
	(?
		(atom "GROUP" true)
		(atom "BY" true)
		(define group (+ rdf_variable ","))
	)
	/* TODO: OFFSET xyz LIMIT xyz */
) '("select" (merge cols) /* TODO: merge cols -> AS */ "where" (merge (coalesce conditions '('())))) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

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
	'('get_var sym) (coalesce (ctx sym) (error "SPARQL error: variable " sym " is used in SELECT but not bound in WHERE clause"))
	(cons head tail) (cons head (map tail (lambda (x) (replace_ctx x ctx))))
	expr
)))

(define rdf_queryplan (lambda (schema query definitions ctx resultfunc /* function that gets cols + ctx */) (begin
	(match query '("select" cols "where" conditions) (begin
		/* ctx: array with predefined variables */
		/* no join reordering yet */
		(define build_scan (lambda (conditions ctx) (match conditions
			(cons '(s p o) tail) (begin
				(define process (lambda (v sym conditions vars) (match v
					'('get_var var) (if (ctx var)
						'((append conditions sym (ctx var)) vars) /* variable is bound: match value */
						'(conditions (append vars sym (symbol var)))) /* variable is free: collect in scope */
					(string? s) '((append conditions sym s) vars)
					(list? l) '((append conditions sym (eval l)) vars)
					(error "SPARQL error: unsupported expression type in WHERE clause: " v)
				)))
				(match (process s "s" '() '()) '(conditions vars)
					(match (process p "p" conditions vars) '(conditions vars)
						(match (process o "o" conditions vars) '(conditions vars)
							'('scan schema "rdf"
								/* condition */ (cons list (extract_assoc conditions (lambda (k v) k))) '('lambda (extract_assoc conditions (lambda (k v) (symbol k))) (cons 'and (extract_assoc conditions (lambda (k v) '('equal? (symbol k) v)))))
								/* map */ (cons list (extract_assoc vars (lambda (k v) k))) '('lambda (extract_assoc vars (lambda (k v) (symbol v))) (build_scan tail (merge ctx (merge (extract_assoc vars (lambda (k v) '(v (symbol v))))))))
							)
				)))
			)
			'() (resultfunc cols ctx)
		)))
		(build_scan conditions ctx)
	) (error "wrong rdf layout " query))
)))

(define parse_sparql (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest) (rdf_queryplan schema (rdf_select rest) definitions '() (lambda (cols ctx) '('resultrow (cons list (map_assoc cols (lambda (k v) (rdf_replace_ctx v ctx)))))))
)))


(define load_ttl (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest)
	(begin
		/* blank node registry: maps _:id to urn:uuid:... per load */
		(set blank_nodes (newsession))
		(define resolve_blank (lambda (val)
			(match val (regex "^_:(.+)$" _ id) (begin
				(set existing (blank_nodes id))
				(if (nil? existing) (begin
					(set generated (concat "urn:uuid:" (uuid)))
					(blank_nodes id generated)
					generated
				) existing)
			) val)
		))
		(define rdf_constant_pfx (parser (or
			(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x)) /* blank node before prefix match */
			(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post))) /* add prefix with validation */
			rdf_constant
		)))
		(define ttl_fact (parser '(
			(define facts
				(parser '(
					(define s rdf_constant_pfx)
					(define ps (+ (parser '((define p rdf_constant_pfx) (define os (+ rdf_constant_pfx ",")) (? ";")) (map os (lambda (o) '(p o))))))
					"."
				) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
			)
			(define rest rest)
		) '("facts" facts "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))
		(set load (lambda (facts) (!begin
			/* resolve blank nodes to UUIDs */
			(set resolved_facts (map facts (lambda (triple) (match triple '(s p o) '((resolve_blank s) (resolve_blank p) (resolve_blank o))))))
			(insert schema "rdf" '("s" "p" "o") resolved_facts '() (lambda () true))
		)))
		(define process_fact (lambda (rest) (match (ttl_fact rest)
			'("facts" facts "rest" (regex "^[ \\n\\r\\t]*$" _)) (load facts)
			'("facts" facts "rest" rest) (!begin (load facts) (process_fact rest))
			rest (error "couldnt parse: " rest)
		)))
		(process_fact rest)
	)
)))
