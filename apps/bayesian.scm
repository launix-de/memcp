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

(define ignore_words (split "bin die der und in zu den das nicht von sie ist des sich mit dem dass er es ein ich auf so eine auch als an nach wie im für man aber aus durch wenn nur war noch werden bei hat wir was wird sein einen welche sind oder zur um haben einer mir über ihm diese einem ihr uns da zum kann doch vor dieser mich ihn du hatte seine mehr am denn nun unter sehr selbst schon hier bis habe ihre dann ihnen seiner alle wieder meine the at there some my of be use her than and this an would first a have each make to from which like been in or she him is one do into who you had how time that by their has its it word if now he but will two find was no up more long for what other on all about go are were did as we many get with when then no come his your them they can these could may i" " "))

(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/bayes/(.*)$" url text) (begin
				(set words (filter (split (toLower text)) (lambda (word) (! (has? ignore_words word)))))
				(if (has_assoc? (req "query") "classify") (begin
					/* classify algo */
					(set category_ ((req "query") "classify"))
					((res "status") 200)
					((res "header") "Content-Type" "application/json")
					(set agg (scan "bayes" "wordclasses" (lambda (partition word category) (and (equal? partition 1) (has? words word) (equal? category category_))) (lambda (class count) (begin
						'(class count) /* dict with count */
					)) (lambda (a b)
						(merge_assoc a b +) /* add class count to result dict */
					) '()))
					(set result (reduce_assoc agg (lambda (a key value) (match a '(best bestscore total)
						(if (> value bestscore) '(key value (+ total value)) '(best bestscore (+ total value)))
					)) '("unknown" 0 1)))
					((res "jsonl") (match result '(best bestscore total) '("class" best "confidentiality" (/ bestscore total))))
				) (begin
					/* learn algo */
					((res "status") 200)
					((res "header") "Content-Type" "text/plain")
					(map words (lambda (word) (begin
						(map_assoc (req "query") (lambda (category class) (begin
							/* TODO: on duplicate key (partition word category class) update count = count + 1 */
							(insert "bayes" "wordclasses" '("partition" 1 "word" word "category" category "class" class "count" 1))
						)))
					)))
					((res "println") "learn ok")
				))
			)
			/* default */
			(old_handler req res))
	))
))
