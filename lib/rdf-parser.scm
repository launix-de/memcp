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

(define parse_rdf (lambda (schema s) (begin

	(define rdf_variable (parser (define x (regex "\?[a-zA-Z_][a-zA-Z0-9_]*")) '('get_var (symbol x))))
	(define rdf_expression (parser (or
		rdf_variable
		(parser '((atom "<" false false) (define x (regex "[^>]*" false)) (atom ">" false)) x) /* string */
		(parser '((atom "\"" false false) (define x (regex "[^\"]*" false)) (atom "\"" false)) x) /* string */
		(regex "[a-zA-Z_][a-zA-Z0-9_]*" false) /* string */
	)))

	(define rdf_select (parser '(
		(atom "SELECT" true)
		(define cols (+ (or
			rdf_variable /* TODO: other expressions */
		) ","))
		(?
			(atom "WHERE" true)
			(atom "{" true)
			(define conditions (* (or
				(parser '((define s rdf_expression) (define p rdf_expression) (define o rdf_expression)) '(s p o))
				(parser '((define p rdf_expression) (define o rdf_expression)) '(nil p o))
				(parser '((define o rdf_expression)) '(nil nil o))
			) ";"))
			(atom "}" true)
		)
		/* TODO: GROUP etc. */
	) '(schema cols /* TODO: merge cols -> AS */ conditions)))

	((parser (define command rdf_select) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))

