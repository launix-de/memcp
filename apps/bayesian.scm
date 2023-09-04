
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/bayes/(.*)$" url text) (begin
				((res "status") 200)
				((res "header") "Content-Type" "text/plain")
				((res "println") "Hello World")
				/*
				TODO: two endpoints -> get, learn
				get -> returns class for category
				learn -> learns multiple categories+classes

				datascheme: bayes(partition, word, category, class, count)

				get: for each word: select class, count from bayes where category=, word=, partition=; normalize count/(sum(count) + 1); summarize weights over all words; choose highest weight, return class+weight

				learn: for each word: for each category: upsert(partition, word, category, class, count+1)

				TODO: string.split, scan->get recordId, delete, :sqlvariables

				(define formula (parse_sql schema query))
				(define resultrow (res "println"))
				(print "received query: " query)
				(eval formula)
				*/
			)
			/* default */
			(old_handler req res))
	))
))
