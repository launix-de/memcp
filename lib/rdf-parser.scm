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
/* unescape standard TTL/JSON escape sequences in a string */
(define rdf_unescape (lambda (s)
	(replace (replace (replace (replace (replace s "\\n" "\n") "\\t" "\t") "\\\\" "\\") "\\\"" "\"") "\\r" "\r")
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
	/* TODO: SUM(...), COUNT(), AVG, MIN, MAX, GROUP_CONCAT */
	/* TODO: CONCAT() */
)))

/* SPARQL filter expressions — no bare names (would eat keywords) */
(define rdf_filter_atom (parser (or
	rdf_variable
	(parser '((atom "<" true) (define x (regex "[^>]*" false false)) (atom ">" false false)) x)
	(parser '((atom "\"" true) (define x (regex "(?:[^\"\\\\]|\\\\.)*" false false)) (atom "\"" false false)) (rdf_unescape x))
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

(define rdf_number (parser (define x (regex "[0-9]+" true)) (simplify x)))
(define rdf_select (parser '(
	(atom "SELECT" true)
	(? (define distinct (atom "DISTINCT" true)))
	(define cols (+ (or
		(parser '((define v rdf_expression) (atom "AS" true) (define v2 rdf_variable)) (match v2 '('get_var s) '((concat s) v))) /* rdf_variable AS rdf_variable */
		(parser (define v rdf_variable) (match v '('get_var s) '((concat s) v))) /* rdf_variable */
	) ","))
	(?
		(atom "WHERE" true)
		(atom "{" true)
		(define conditions (* (or
			(parser '((define s rdf_expression) (define ps (+ (parser '((define p rdf_expression) (define os (+ rdf_expression ","))) (map os (lambda (o) '(p o)))) ";"))) (merge (map ps (lambda (p) (map p (lambda (p1) (cons s p1)))))))
			(parser '((atom "FILTER" true) "(" (define expr rdf_filter_or) ")") (list (list "__filter__" expr)))
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
	(define order (parser (? (atom "ORDER" true) (atom "BY" true) (define ordercols (+ (or (parser '((define dir (or (atom "DESC" true) (atom "ASC" true))) "(" (define expr rdf_expression) ")") '(expr dir)) (parser (define expr rdf_expression) '(expr "ASC"))) ","))) ordercols))
	(? (atom "LIMIT" true) (define limit rdf_number))
	(? (atom "OFFSET" true) (define offset rdf_number))
) '("select" (merge cols) "where" (merge (coalesce conditions '('()))) "order" order "limit" limit "offset" offset "distinct" distinct) "^(?:(?s:/\\*.*?\\*/)|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

(define ttl_header (parser '(
	(define definitions (*
		(or
			(parser '((atom "@prefix" true) (define pfx (regex "[a-zA-Z0-9_]*" false)) (atom ":" false false) (define content rdf_constant) ".") '(pfx content))
			(parser '((atom "@base" true) (define content rdf_constant) ".") '("" content)) /* @base sets the empty prefix */
		)
	))
	(define rest rest)
) '("prefixes" (merge definitions) "rest" rest) "^(?:(?s:/\\*.*?\\*/)|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))

(define rdf_replace_ctx (lambda (expr ctx) (match expr
	'('get_var sym) (coalesce (ctx sym) (error "SPARQL error: variable " sym " is used in SELECT but not bound in WHERE clause"))
	(cons head tail) (cons head (map tail (lambda (x) (rdf_replace_ctx x ctx))))
	expr
)))

(define rdf_queryplan (lambda (schema query definitions ctx resultfunc /* function that gets cols + ctx */) (begin
	(match query '("select" cols "where" conditions "order" order "limit" limit "offset" offset "distinct" distinct) (begin
		/* ctx: array with predefined variables */
		/* no join reordering yet */
		(define build_scan (lambda (conditions order ctx) (match conditions
			(cons '(s p) tail) (if (equal? (concat s) "__filter__")
				/* FILTER: wrap inner plan in if-check */
				'('if (rdf_replace_ctx p ctx) (build_scan tail order ctx))
				/* 2-element: error */
				(error "SPARQL error: expected triple pattern (s p o), got 2 elements")
			)
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
							(set map_fn '('lambda (extract_assoc vars (lambda (k v) (symbol v))) (build_scan tail (if order_head order_rest order) inner_ctx)))
							(match order_head
								'(col dir)
								/* ordered scan */
								'('scan_order schema "rdf"
									filter_cols filter_fn
									'(list col) '(list (match dir "DESC" > <)) 0 0 -1
									map_cols map_fn 'cons nil)
								/* normal scan */
								'('scan schema "rdf" filter_cols filter_fn map_cols map_fn)
							)
				))))
			)
			'() (match order (cons _ _) (error (concat "order not consumed: " order)) (resultfunc cols ctx))
		)))
		(build_scan conditions order ctx)
	) (error "wrong rdf layout " query))
)))

(define parse_sparql (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest) (begin
		(set parsed (rdf_select rest))
		(set qlimit (parsed "limit"))
		(set qoffset (parsed "offset"))
		(set qdistinct (parsed "distinct"))
		(set needs_wrap (or (not (nil? qlimit)) (not (nil? qoffset)) (not (nil? qdistinct))))
		(set effective_offset (coalesce qoffset 0))
		(set effective_limit (coalesce qlimit 999999999))
		/* build resultfunc that includes limit/offset/distinct logic */
		(if needs_wrap (begin
			/* state session is created at compile time, initialized here, used at eval time */
			(set _st (newsession))
			(_st "cnt" 0)
			(if qdistinct (_st "seen" (newsession)))
			(rdf_queryplan schema parsed definitions '() (lambda (cols ctx) (begin
				(set row_expr (cons list (map_assoc cols (lambda (k v) (rdf_replace_ctx v ctx)))))
				(if qdistinct
					/* DISTINCT + LIMIT/OFFSET: check seen, then count */
					'('begin
						'('set '_row row_expr)
						'('set '_dkey '('json_encode '_row))
						'('if '('not '((_st "seen") '_dkey)) '('begin
							'((_st "seen") '_dkey true)
							'('set '_c '(_st "cnt"))
							'('if '('and '('>= '_c effective_offset) '('< '_c '('+ effective_offset effective_limit)))
								'('resultrow '_row)
							)
							'(_st "cnt" '('+ '_c 1))
						))
					)
					/* LIMIT/OFFSET only: just count */
					'('begin
						'('set '_row row_expr)
						'('set '_c '(_st "cnt"))
						'('if '('and '('>= '_c effective_offset) '('< '_c '('+ effective_offset effective_limit)))
							'('resultrow '_row)
						)
						'(_st "cnt" '('+ '_c 1))
					)
				)
			)))
		) (rdf_queryplan schema parsed definitions '() (lambda (cols ctx) '('resultrow (cons list (map_assoc cols (lambda (k v) (rdf_replace_ctx v ctx))))))))
	)
)))


/* helper: parse TTL into list of (s p o) triples without loading */
(define parse_ttl_triples (lambda (schema s) (match (ttl_header s)
	'("prefixes" definitions "rest" rest)
	(begin
		(define rdf_constant_pfx (parser (or
			(parser '((atom "_:" true) (define x (regex "[a-zA-Z0-9_]+" false false))) (concat "_:" x))
			(parser '((define pfx (regex "[a-zA-Z0-9_]*" true)) (atom ":" false false) (define post (regex "[a-zA-Z0-9_]*" false))) (if (nil? (definitions pfx)) (error "undefined prefix: " pfx) (concat (definitions pfx) post)))
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
		) '("facts" facts "rest" rest) "^(?:(?s:/\\*.*?\\*/)|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))
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
		) '("facts" facts "rest" rest) "^(?:(?s:/\\*.*?\\*/)|--[^\r\n]*[\r\n]|--[^\r\n]*$|#[^\r\n]*[\r\n]|#[^\r\n]*$|[\r\n\t ]+)+"))
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
