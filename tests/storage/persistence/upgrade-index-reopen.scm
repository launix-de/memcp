/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
(begin
	(define caller (newsession))
	(caller "username" "root")
	(caller "__tsql_principal_id" (tsql_account_id "root"))
	(caller "schema" "memcp-tests")
	(caller "__tsql_scope" (newsession))
	(define run (lambda (text) (begin
		(define result (newsession)) (result "rows" (list))
		(tsql_execute_request "memcp-tests" text caller nil 0
			(lambda (row) (result "rows" (append (result "rows") row)))
			(lambda fields true))
		(result "rows"))))
	(define expect (lambda (text names expected) (begin
		(define actual (map (run text) (lambda (row) (map names (lambda (name) (get_assoc row name))))))
		(if (equal? (json_encode actual) (json_encode expected)) true
			(error (concat "candidate index upgrade result mismatch: " text " expected=" (json_encode expected) " actual=" (json_encode actual)))))))
	(define expect_indexes (lambda (tx expected) (begin
		(define target (table "memcp-tests" false "candidate_c04_rows" false tx))
		(define actual (map (show target "logical_indexes" tx) (lambda (index)
			(map (list "Id" "Kind" "Columns" "Included" "Descending" "RequestedPlacement") (lambda (field) (index field))))))
		(if (equal? (json_encode actual) (json_encode expected)) true
			(error (concat "candidate index metadata mismatch: expected=" (json_encode expected) " actual=" (json_encode actual)))))))
	(define nonunique (list "candidate_c07_nonunique" "nonunique" (list "payload") (list "code") (list true) "nonclustered"))
	(define enforcing (list "candidate_c07_unique" "unique" (list "payload") (list) (list false) "nonclustered"))
	/* Cold read after the runner-owned normal checkpoint/restart. No new
	fixture creation, index repair or oracle capture is permitted here. */
	(expect_indexes nil (list nonunique))
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50) (list 6 60)))
	(expect "SELECT COUNT(*) AS n,COUNT(DISTINCT ID) AS distinct_ids,MIN(ID) AS lo,MAX(ID) AS hi,SUM(ID) AS total FROM candidate_c04_rows" (list "n" "distinct_ids" "lo" "hi" "total") (list (list 5 5 42 46 220)))
	(expect "SELECT code,payload,ID FROM candidate_c04_rows WHERE code>=5 ORDER BY code" (list "code" "payload" "ID") (list (list 5 50 45) (list 6 60 46)))
	(run "BEGIN TRANSACTION")
	(run "UPDATE candidate_c04_rows SET payload=11 WHERE code=3")
	(expect "SELECT code,payload FROM candidate_c04_rows WHERE code IN(1,3) ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 11)))
	(run "ROLLBACK TRANSACTION")
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50) (list 6 60)))
	(expect "SELECT COUNT(*) AS n,COUNT(DISTINCT ID) AS distinct_ids,MIN(ID) AS lo,MAX(ID) AS hi,SUM(ID) AS total FROM candidate_c04_rows" (list "n" "distinct_ids" "lo" "hi" "total") (list (list 5 5 42 46 220)))
	(expect "SELECT code,payload,ID FROM candidate_c04_rows WHERE code>=5 ORDER BY code" (list "code" "payload" "ID") (list (list 5 50 45) (list 6 60 46)))
	(expect "SELECT IDENT_CURRENT(N'candidate_c04_rows') AS current_value,@@TRANCOUNT AS depth" (list "current_value" "depth") (list (list 46 0)))
	true)
