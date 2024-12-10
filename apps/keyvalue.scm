/* microservice democase: a simple key value store with prepared statements */
(import "../lib/sql-parser.scm")
(import "../lib/queryplan.scm")

/* usage:

set a key-value pair:
curl -d "my_value" http://localhost:1266/my_key

retrieve a key-value pair:
http://localhost:1266/my_key

*/

/* initialize database and prepare sql statements */
(createdatabase "keyvalue" true)
(eval (parse_sql "keyvalue" "CREATE TABLE IF NOT EXISTS kv(key TEXT, value TEXT, UNIQUE KEY PRIMARY(key))"))

(set item_get (parse_sql "keyvalue" "SELECT value FROM kv WHERE key = @key"))
(set item_set (parse_sql "keyvalue" "INSERT INTO kv(key, value) VALUES (@key, @value) ON DUPLICATE KEY UPDATE value = @value"))
/*(set item_list (parse_sql "keyvalue" "SELECT key, value FROM kv"))*/


(define http_handler (begin
	(lambda (req res) (begin
		(set session (newsession))
		(session "key" (req "path"))
		(if (equal? (req "method") "GET") (begin
			/* GET = load */
			(set resultrow (lambda (resultset) ((res "print") (resultset "value"))))
			(eval item_get)
		) (begin
			/* PUT / POST: store */
			(session "value" ((req "body")))
			(eval item_set)
			((res "print") "ok")
		))
	))
))

(set port 1266)
(serve port (lambda (req res) (http_handler req res)))
