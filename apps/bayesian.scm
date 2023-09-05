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

/* create schema */
(print "Loading Bayesian Classifier app")
(if (has? (show) "bayes") true (begin
	(print "creating database")
	(createdatabase "bayes")
))
(if (has? (show "bayes") "wordclasses") true (begin
	(print "creating tables")
	(eval (parse_sql "bayes" "CREATE TABLE wordclasses(partition int, word text, category text, class int, count int)"))
))

(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/bayes/(.*)$" url text) (begin
				(print req)
				(if (has? (req "query") "classify") (begin
					/* classify algo */
					((res "status") 200)
					((res "header") "Content-Type" "text/plain")
					((res "println") (concat "TODO: classify " text " for " ((req "query") "classify")))
				) (begin
					/* learn algo */
					((res "status") 200)
					((res "header") "Content-Type" "text/plain")
					((res "println") (concat "TODO: learn " text " for classes " (req "query")))
				))
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
