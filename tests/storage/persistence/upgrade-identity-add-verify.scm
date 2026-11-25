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
			(error (concat "candidate identity upgrade result mismatch: " text " expected=" (json_encode expected) " actual=" (json_encode actual)))))))
	/* All initial reads run before new candidate-fixture DML in this cold
	process. Values and metadata below are declared constants, never candidate-
	captured replacements for the original writer oracle. */
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50)))
	(expect "SELECT COUNT(*) AS n,COUNT(DISTINCT ID) AS distinct_ids,MIN(ID) AS lo,MAX(ID) AS hi,SUM(ID) AS total,IDENT_SEED(N'candidate_c04_rows') AS seed,IDENT_CURRENT(N'candidate_c04_rows') AS current_value FROM candidate_c04_rows" (list "n" "distinct_ids" "lo" "hi" "total" "seed" "current_value") (list (list 4 4 42 45 174 42 45)))
	(expect "SELECT name FROM sys.check_constraints WHERE parent_object_id=OBJECT_ID(N'candidate_c04_rows') ORDER BY name" (list "name") (list (list "CK_candidate_c04_payload")))
	(expect "SELECT code,payload,COL_LENGTH(N'candidate_c04_rollback',N'ID') AS identity_length FROM candidate_c04_rollback ORDER BY code" (list "code" "payload" "identity_length") (list (list 1 10 nil) (list 2 20 nil)))
	(expect "SELECT COUNT(*) AS n,IDENT_SEED(N'candidate_c04_empty') AS seed,IDENT_CURRENT(N'candidate_c04_empty') AS current_value FROM candidate_c04_empty" (list "n" "seed" "current_value") (list (list 0 7 7)))
	(run "INSERT candidate_c04_rows(code,payload) VALUES(6,60)")
	(expect "SELECT code,payload,ID FROM candidate_c04_rows WHERE code>=5 ORDER BY code" (list "code" "payload" "ID") (list (list 5 50 45) (list 6 60 46)))
	(run "INSERT candidate_c04_empty(code,payload) VALUES(1,10)")
	(expect "SELECT code,payload,ID FROM candidate_c04_empty" (list "code" "payload" "ID") (list (list 1 10 7)))
	(run "ALTER TABLE candidate_c04_rollback ADD ID INT NOT NULL IDENTITY(7,1)")
	(run "INSERT candidate_c04_rollback(code,payload) VALUES(3,30)")
	(expect "SELECT COUNT(*) AS n,COUNT(DISTINCT ID) AS distinct_ids,MIN(ID) AS lo,MAX(ID) AS hi,SUM(ID) AS total,IDENT_SEED(N'candidate_c04_rollback') AS seed,@@TRANCOUNT AS depth FROM candidate_c04_rollback" (list "n" "distinct_ids" "lo" "hi" "total" "seed" "depth") (list (list 3 3 7 9 24 7 0)))
	true)
