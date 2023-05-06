/*
Copyright (C) 2023  Carl-Philip Hänsch

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

(define parse_sql (lambda (s) (begin
	/* lots of small parsers that can be combined */
	(define identifier (lambda (s) (match s
		(regex "(?is)^(?:\\s|\\n)*`(.*)`(?:\\s|\\n)*(.*)" _ id rest) '(id rest)
		(regex "(?is)^(?:\\s|\\n)*([a-zA-Z_][a-zA-Z_0-9]*)(?:\\s|\\n)*(.*)" _ id rest) '(id rest)
		(error (concat "expected identifier, found " s))
	)))

	(define parenthesis (lambda (s) (match s
		(regex "(?is)^(?:\\s|\\n)*\((?:\\s|\\n)*(.*)" _ rest) rest
		(error (concat "expected (, found " s))
	)))

	(define tabledecl (lambda (s) (match s
		(concat ")" rest) '() /* TODO: rest??? */
		rest (match (identifier rest)
			'(colname rest) (match (identifier rest)
				'(typename rest) (match rest
					/* todo: allow white spaces in dimension */
					(regex "^(?s)\\(([0-9]+),([0-9]+)\\)([^,]*),(.*)" _ dim1 dim2 typeparams rest) (cons '((symbol "list") colname typename '((symbol "list") dim1 dim2) typeparams) (tabledecl rest))
					(regex "^(?s)\\(([0-9]+),([0-9]+)\\)([^,]*)\)(.*)" _ dim1 dim2 typeparams rest) '('((symbol "list") colname typename '((symbol "list") dim1 dim2) typeparams)) /* TODO: rest */
					(regex "^(?s)\\(([0-9]+)\\)([^,]*),(.*)" _ dim1 typeparams rest) (cons '((symbol "list") colname typename '((symbol "list") dim1) typeparams) (tabledecl rest))
					(regex "^(?s)\\(([0-9]+)\\)([^,]*)\)(.*)" _ dim1 typeparams rest) '('((symbol "list") colname typename '((symbol "list") dim1) typeparams)) /* TODO: rest */
					(regex "^(?s)([^,]*),(.*)" _ typeparams rest) (cons '((symbol "list") colname typename '((symbol "list")) typeparams) (tabledecl rest))
					(regex "^(?s)([^,]*)\)(.*)" _ typeparams rest) '('((symbol "list") colname typename '((symbol "list")) typeparams)) /* TODO: rest */
					(error (concat "expected , or ) but found " rest))
				)
			)
		)
	)))

	/* eat a identifier from string */
	(define expression (lambda (s) (match s
		/* constant */
		(regex "^(-?[0-9]+(?:\\.[0-9*])?)(?:\\s|\\n)*($|[^0-9].*)" _ num rest) (expression_extend (simplify num) rest)
		/* identifier (TODO: tblalias.identifier) */
		(regex "(?is)^(?:\\s|\\n)*`(.*)`(?:\\s|\\n)*(.*)" _ id rest) (expression_extend '((quote get_column) "*" id) rest)
		(regex "(?is)^(?:\\s|\\n)*([a-zA-Z_][a-zA-Z_0-9]*)(?:\\s|\\n)*(.*)" _ id rest) (expression_extend '((quote get_column) "*" id) rest)
		/* parenthesis */
		(concat "(" rest) (match (expression rest) '(expr (concat ")" rest)) '('((quote begin) expr) rest) (error (concat "expected expression found " rest)))
		(error (concat "could not parse " s))
	)))

	/* try to find other operators to add to the expression */
	(define expression_extend (lambda (expr s) (match s
		/* + - */
		(regex "^([+\\-])(?:\\s|\\n)*(.*)" _ operator rest)
			(match (expression rest) '(expr2 rest) '('((symbol operator) expr expr2) rest))
		/* * / */
		(regex "^([*\\/])(?:\\s|\\n)*(.*)" _ operator rest)
			(match (expression rest) '(expr2 rest)
				(match expr2
					/* shift down * and / before + and - */
					'(+ a b) '('((quote +) '((symbol operator) expr a) b) rest)
					'(- a b) '('((quote -) '((symbol operator) expr a) b) rest)
					'('((symbol operator) expr expr2) rest)))
		'(expr s) /* no extension */
	)))

	/* derive the description of a column from its expression */
	(define extract_title (lambda (expr) (match expr
		'((symbol get_column) "*" col) col
		'((symbol get_column) tblvar col) (concat tblvar "." col)
		(cons sym args) /* function call */ (concat (cons sym (map args extract_title)))
		(concat expr)
	)))

	/* compile select */
	(define select (lambda (rest fields) (begin
		(define parse_afterexpr (lambda (expr id rest) (match rest
			/* no FROM */ "" (build_queryplan '() (append fields id expr))
			/* followed by comma: */ (regex "^(?s),(?:\\s|\\n)*(.*)" _ rest) (select rest (append fields id expr))
			/* followed by AS: */ (regex "^(?is)AS(?:\\s|\\n)*(.*)" _ rest) (match (identifier rest) '(id rest) (parse_afterexpr expr id rest) (error (concat "expected identifier after AS, found: " rest)))
			/* followed by FROM: */ (regex "^(?is)FROM(?:\\s|\\n)*(.*)" _ rest) (match (identifier rest)
				'(tblid rest) (build_queryplan '('(tblid (quote schema) tblid)) (append fields id expr))
				/* TODO: FROM () AS tbl | tbl | tbl as alias ... | comma tablelist */
			) /* TODO: FROM, WHERE, GROUP usw. */
			/* otherwise */ (error (concat "expected , AS or FROM but found: " rest))
		)))

		/* after select, there must be an expression */
		(match
			(expression rest) '(expr rest) (parse_afterexpr expr (extract_title expr) rest)
			(error (concat "expected expression, found " rest)))
	)))

	/* compile insert */
	(define parse_insert (lambda (rest) (match (identifier rest)
		'(tbl rest) (begin
			(define zip_cols (lambda(cols tuple) (match cols
				(cons col cols) (cons col (cons (car tuple) (zip_cols cols (cdr tuple))))
				'()
			)))
			(define tuplelist (lambda(tuples tuple rest)(begin
				(match (expression rest)
					'(value rest) (match rest
						(regex "(?is)^,(?:\\s|\\n)*(.*)" _ rest) (tuplelist tuples (append tuple value) rest) /* append value to tuple */
						(regex "(?is)^\\)(?:\\s|\\n)*,(?:\\s|\\n)*\\((.*)" _ rest) (tuplelist (append tuples (append tuple value)) '() rest) /* move on to next tuple */
						(concat ")" rest) '((append tuples (append tuple value)) rest) /* finished -> return list of tuples */
						(error (concat "expected , or ) in column list but found " rest))
					)
					(error (concat "expected expression found " rest))
				)
			)))
			(define columnlist (lambda(cols rest)(begin
				(match (identifier rest)
					'(col rest) (match rest
						(concat "," rest) (columnlist (append cols col) rest)
						(concat ")" rest) (match rest
							(regex "(?is)^(?:\\s|\\n)*VALUES(?:\\s|\\n)+\\((.*)" _ rest) (match (tuplelist '() '() rest)
								'(tuples rest) (begin
									(define cols (append cols col))
									(merge '('((quote begin)) (map tuples (lambda (tuple) '((quote insert) (quote schema) tbl (cons (quote list) (zip_cols cols tuple))))))) /* TODO: what if something is left in rest??? */
								)
								(error (concat "expected tuple list but found " rest))
							)
							(regex "(?is)^(?:\\s|\\n)*SELECT(?:\\s|\\n)+(.*)" _ rest) (error "TODO: implement INSERT INTO SELECT")
							(error (concat "expected VALUES or SELECT but found " rest))
						)
						(error (concat "expected , or ) in column list but found " rest))
					)
					(error (concat "expected identifier found " rest))
				)
			)))
			(match rest
				(concat "(" rest) (columnlist '() rest)
				(error (concat "expected ( but found " rest))
			)
			/*
			(print "TODO INSERT " tbl)
			'((quote insert) schema tbl '((quote list) "bar" 551))
			*/
		)
		(error (concat "expected table name, found " rest))
	)))

	/* main compile function -> decide which kind of SQL query it is */
	(match s
		(regex "(?s)^\\s*(?m:--.*?$)(.*)" _ rest) /* comment */ (parse_sql rest)
		(concat "\n" rest) (parse_sql rest)
		(regex "(?is)^\\s+(.*)" _ rest) (parse_sql rest)
		(regex "(?is)^CREATE(?:\\s|\\n)+TABLE(?:\\s|\\n)+(.*)" _ rest) (match (identifier rest) '(id rest) '((symbol "createtable") (quote schema) id (cons (symbol "list") (tabledecl (parenthesis rest)))) (error "expected identifier"))
		(regex "(?is)^SELECT(?:\\s|\\n)+(.*)" _ rest) (select rest '())
		(regex "(?is)^INSERT(?:\\s|\\n)+INTO(?:\\s|\\n)+(.*)" _ rest) (parse_insert rest)
		(error (concat "unknown SQL syntax: " s))
	)
)))

