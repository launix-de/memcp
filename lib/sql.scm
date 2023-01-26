
/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(concat "/sql/" rest) (begin
				((res "status") 200)
				((res "header") "Content-Type" "text/plain")
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
		"TODO: SQL Query"
	))
)
(print "MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)")
