
(define parse_sql (lambda (schema s) (begin

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
					(regex "^(?s)\\(([0-9]+),([0-9]+)\\)([^,]*),(.*)" _ dim1 dim2 typeparams rest) (cons '(colname typename '(dim1 dim2) typeparams) (tabledecl rest))
					(regex "^(?s)\\(([0-9]+),([0-9]+)\\)([^,]*)\)(.*)" _ dim1 dim2 typeparams rest) '('(colname typename '(dim1 dim2) typeparams)) /* TODO: rest */
					(regex "^(?s)\\(([0-9]+)\\)([^,]*),(.*)" _ dim1 typeparams rest) (cons '(colname typename '(dim1) typeparams) (tabledecl rest))
					(regex "^(?s)\\(([0-9]+)\\)([^,]*)\)(.*)" _ dim1 typeparams rest) '('(colname typename '(dim1) typeparams)) /* TODO: rest */
					(regex "^(?s)([^,]*),(.*)" _ typeparams rest) (cons '(colname typename '() typeparams) (tabledecl rest))
					(regex "^(?s)([^,]*)\)(.*)" _ typeparams rest) '('(colname typename '() typeparams)) /* TODO: rest */
					(error (concat "expected , or ) but found " rest))
				)
			)
		)
	)))

	(define expression (lambda (s) (match s
		(regex "^([0-9]+(?:\\.[0-9*])?)(?:\\s|\\n)*($|[^0-9].*)" _ num rest) (expression_extend (simplify num) rest)
		(error (concat "could not parse " s))
	)))

	(define expression_extend (lambda (expr s) (match s
		(regex "^([+\\-*\\/])(?:\\s|\\n)*(.*)" _ operator rest) (match (expression rest) '(expr2 rest) '('(operator expr expr2) rest))
		'(expr s) /* no extension */
	)))

	(define select (lambda (rest) (begin
		(match (expression rest) '(expr rest) (begin
			(print "expr=" expr)
			(print "rest=" rest)
		))
		(scan "test" "foo" (lambda () true) (lambda (bar) (print "bar=" bar)))
	)))

	(match s
		(regex "(?s)^\\s*(?m:--.*?$)(.*)" _ rest) /* comment */ (parse_sql schema rest)
		(concat "\n" rest) (parse_sql schema rest)
		(regex "(?is)^\\s+(.*)" _ rest) (parse_sql schema rest)
		(regex "(?is)^CREATE(?:\\s|\\n)+TABLE(?:\\s|\\n)+(.*)" _ rest) (match (identifier rest) '(id rest) '(createtable schema id (tabledecl (parenthesis rest))) (error "expected identifier"))
		(regex "(?is)^SELECT(?:\\s|\\n)+(.*)" _ rest) (select rest)
		(error (concat "unknown SQL syntax: " s))
	)
)))

/* TODO: session state handling -> which schema */
(createdatabase "test")
(createtable "test" "foo" '('("bar" "int" '() "")))
(insert "test" "foo" '("bar" 12))
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
				(print (parse_sql schema rest))
				((res "println") (concat "TODO: query " rest))
			)
			/* default */
			(old_handler req res))
	))
))

/* dedicated mysql protocol listening at port 3307 */
(mysql 3307
	(lambda (username) "TODO: return pwhash") /* auth */
	(lambda (schema) true) /* switch schema */
	(lambda (sql) (begin /* sql */
		(print "received query: " sql)
		(print (parse_sql schema sql))
		"TODO: execute"
	))
)
(print "MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)")
