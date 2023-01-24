
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
