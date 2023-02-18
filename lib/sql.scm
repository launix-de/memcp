(define parse_sql (lambda (s) (begin

	(define identifier (lambda (s) (match s
		(regex "(?is)^(?:\\s|\\n)*`(.*)`(.*)" _ id rest) '(id rest)
		(regex "(?is)^(?:\\s|\\n)*([a-zA-Z_][a-zA-Z_0-9]*)(.*)" _ id rest) '(id rest)
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

	/* build queryplan from parsed query */
	(define build_queryplan (lambda (tables fields) (begin
		/* parse_sql will compile (get_column tblvar col) into the formulas. we have to replace it with the correct variable */
		/* tables: '('(alias schema tbl) ...) */
		/* fields: '(colname expr ...) */

		/* returns a list of '(tblvar col) */
		(define extract_columns (lambda (expr) (match expr
			'((symbol get_column) tblvar col) '('(tblvar col))
			(cons sym args) /* function call */ (merge (map args extract_columns)) /* TODO: use collector */
			'()
		)))

		(define map_assoc (lambda (columns fn)
			(match columns
				(cons colid (cons colvalue rest)) (cons colid (cons (fn colvalue) (map_assoc rest fn)))
				'()
			)
		))

		(define extract_assoc (lambda (columns fn)
			(match columns
				(cons colid (cons colvalue rest)) (cons (fn colvalue) (extract_assoc rest fn))
				'()
			)
		))

		/* changes (get_column tblvar col) into its counterpart */
		(define replace_columns (lambda (expr) (match expr
			'((symbol get_column) tblvar col) (symbol col) /* TODO: rename in outer scans */
			(cons sym args) /* function call */ (cons sym (map args replace_columns))
			expr /* literals */
		)))

		/* columns: '('(tblalias colname) ...) */
		(set columns (merge (extract_assoc fields extract_columns)))
		(print "cols=" columns)
		(print (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))))

		/* TODO: sort tables according to join plan */
		(define build_scan (lambda (tables)
			(match tables
				'('("1x1" "system" "1x1")) '((symbol "resultrow") (cons (symbol "list") fields))
				(cons '(alias schema tbl) tables) /* outer scan */
					'((quote scan) schema tbl
						'((quote lambda) '() (quote true)) /* TODO: filter */
						/* todo filter columns for alias */
						'((quote lambda) (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))) (build_scan tables))
						/* TODO: reduce+neutral */)
				'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields replace_columns)))
			)
		))
		(build_scan tables)
	)))
	/* compile select */
	(define select (lambda (rest fields) (begin
		(define parse_afterexpr (lambda (expr id rest) (match rest
			/* no FROM */ "" (build_queryplan '('("1x1" "system" "1x1")) (append fields id expr))
			/* followed by comma: */ (regex "^(?s),(?:\\s|\\n)*(.*)" _ rest) (select rest (append fields id expr))
			/* followed by AS: */ (regex "^(?is)AS(?:\\s|\\n)*(.*)" _ rest) (match (identifier rest) '(id rest) (parse_afterexpr expr id rest) (error (concat "expected identifier after AS, found: " rest)))
			/* followed by FROM: */ (regex "^(?is)FROM(?:\\s|\\n)*(.*)" _ rest) (match (identifier rest)
				'(id rest) (build_queryplan '('(id (quote schema) id)) (append fields id expr))
				/* TODO: FROM () AS tbl | tbl | tbl as alias ... | comma tablelist */
			) /* TODO: FROM, WHERE, GROUP usw. */
			/* otherwise */ (error (concat "expected , AS or FROM but found: " rest))
		)))

		/* after select, there must be an expression */
		(match
			(expression rest) '(expr rest) (parse_afterexpr expr (concat expr) rest)
			(error (concat "expected expression, found " rest)))
	)))

	(match s
		(regex "(?s)^\\s*(?m:--.*?$)(.*)" _ rest) /* comment */ (parse_sql rest)
		(concat "\n" rest) (parse_sql rest)
		(regex "(?is)^\\s+(.*)" _ rest) (parse_sql rest)
		(regex "(?is)^CREATE(?:\\s|\\n)+TABLE(?:\\s|\\n)+(.*)" _ rest) (match (identifier rest) '(id rest) '((symbol "createtable") (quote schema) id (cons (symbol "list") (tabledecl (parenthesis rest)))) (error "expected identifier"))
		(regex "(?is)^SELECT(?:\\s|\\n)+(.*)" _ rest) (select rest '())
		(error (concat "unknown SQL syntax: " s))
	)
)))

/* TODO: session state handling -> which schema */
(createdatabase "test")
(createtable "test" "foo" '('("bar" "int" '() "")))
(insert "test" "foo" '("bar" 12))
(insert "test" "foo" '("bar" 44))
(set schema "test")

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(concat "/sql/" rest) (begin
				((res "status") 200)
				((res "header") "Content-Type" "text/plain")
				(define formula (parse_sql rest))
				(define resultrow (res "println"))
				(print "received query: " rest)
				(eval formula)
			)
			/* default */
			(old_handler req res))
	))
))

/* dedicated mysql protocol listening at port 3307 */
(mysql 3307
	(lambda (username) "TODO: return pwhash") /* auth */
	(lambda (schema) true) /* switch schema */
	(lambda (sql resultrow_sql) (begin /* sql */
		(print "received query: " sql)
		(define formula (parse_sql sql))
		(define resultrow resultrow_sql)
		(eval formula)
	))
)
(print "MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)")
