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

(define rdf_variable (parser (define x (regex "\?[a-zA-Z0-9_]+")) '('get_var (symbol x))))
(define rdf_constant (parser (or
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x) /* string */
	(parser '((atom "\"" true) (define x (regex "[^\"]*" false false)) (atom "\"@" false false) (regex "[a-zA-Z_0-9]+" false)) x) /* string with language, ignore language, TODO: escape tbnrf */
	(parser '((atom "\"" true) (define x (regex "[^\"]*" false false)) (atom "\"" false false)) x) /* string, TODO: escape tbnrf */
	(regex "[a-zA-Z0-9_]+" true) /* string, TODO: handle prefixing */
)))
(define rdf_expression (parser (or
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

(define parse_rdf (lambda (schema s) (begin
	(set group nil)
	(define rdf_select (parser '(
		(atom "SELECT" true)
		(define cols (+ (or
			rdf_variable /* TODO: other expressions AS xyz */
		) ","))
		(?
			(atom "WHERE" true)
			(atom "{" true)
			(define conditions (* (or
				(parser '((define s rdf_expression) (define ps (+ (parser '((define p rdf_expression) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
				/* TODO: FILTER regex(?var "pattern") */
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
	) '(schema cols /* TODO: merge cols -> AS */ (merge conditions))))

	((parser (define command rdf_select) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))

(define ttl_header (parser '(
	(define definitions (*
		(parser '((atom "@prefix" true) (define pfx (regex "[a-zA-Z0-9_]*" false)) (atom ":" false false) (define content rdf_constant) ".") '(pfx content))
	))
	(define rest rest)
) '("prefixes" (merge definitions) "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))


(define load_ttl (lambda (schema s) (match (ttl_header s)
	       '("prefixes" definitions "rest" rest)
		(begin
			(define rdf_constant_pfx (parser (or
				(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) (concat (definitions pfx) post)) /* add prefix */
				rdf_constant
			)))
			(define ttl_fact (parser '(
				(define facts 
					(parser '((define s rdf_constant_pfx) (define ps (+ (parser '((define p rdf_constant_pfx) (define os (+ rdf_constant_pfx ","))) (map os (lambda (o) '(p o)))) ";")) (? ";") ".") (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
				)
				(define rest rest)
			) '("facts" facts "rest" rest) "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))
			(set load (lambda (facts) (!begin
				/* (print "start ======== " facts "-- end") */
				(insert schema "rdf" '("s" "p" "o") facts true)
			)))
			(define process_fact (lambda (rest) (match (ttl_fact rest)
				'("facts" facts "rest" (regex "[ \\n\\r\\t]*" _)) (load facts)
				'("facts" facts "rest" rest) (!begin (load facts) (process_fact rest))
			)))
			(process_fact rest)
		)
)))
