
(define parse_sql (lambda (s) (begin

	(define identifier (lambda (s) (match s
		(regex "(?is)^`(.*)`(.*)" _ id rest) '(id rest)
		(regex "(?is)^([a-zA-Z_][a-zA-Z_0-9]*)(.*)" _ id rest) '(id rest)
		(error (concat "expected identifier, found " s))
	)))

	(match s
		(regex "(?s)^\\s*(?m:--.*?$)(.*)" _ rest) /* comment */ (parse_sql rest)
		(concat "\n" rest) (parse_sql rest)
		(regex "(?is)^\\s+(.*)" _ rest) (parse_sql rest)
		(regex "(?is)^CREATE(?:\\s|\\n)+TABLE(?:\\s|\\n)+(.*)" _ rest) (match (identifier rest) '(id rest) (concat "tablecreate " id ", decl: " rest) (error "expected identifier"))
		(error (concat "unknown SQL syntax: " s))
	)
)))

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(concat "/sql/" rest) (begin
				((res "status") 200)
				((res "header") "Content-Type" "text/plain")
				(print (parse_sql rest))
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
		(print (parse_sql sql))
		"TODO: execute"
	))
)
(print "MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)")
