(settings "TracePrint" true)
(define bench_loop (lambda (n f)
  (if (> n 0) (begin
    (f)
    (bench_loop (- n 1) f)
  ) nil)
))
(define bench_policy (lambda (schema table write) true))

/* Queries with subqueries, derived tables, EXISTS */
(define q1 "SELECT username FROM user WHERE username IN (SELECT username FROM user)")
(define q2 "SELECT u.username FROM user u WHERE EXISTS (SELECT 1 FROM user u2 WHERE u2.username = u.username)")
(define q3 "INSERT INTO user(username) VALUES ('a','b','c','d')")

/* warm-up */
(bench_loop 50 (lambda () (begin
  (parse_sql "system" q1 bench_policy)
  (parse_sql "system" q2 bench_policy)
  (parse_sql "system" q3 bench_policy)
)))

/* measured run */
(time (bench_loop 500 (lambda () (begin
  (parse_sql "system" q1 bench_policy)
  (parse_sql "system" q2 bench_policy)
  (parse_sql "system" q3 bench_policy)
))) "500x complex parse_sql (3 queries)")
