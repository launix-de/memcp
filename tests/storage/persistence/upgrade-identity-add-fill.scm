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
			(lambda (row) (result "rows" (append (result "rows") (list row))))
			(lambda fields true))
		(result "rows"))))
	(define expect (lambda (text names expected) (begin
		(define actual (map (run text) (lambda (row) (map names (lambda (name) (get_assoc row name))))))
		(if (equal? (json_encode actual) (json_encode expected)) true
			(error (concat "candidate identity upgrade result mismatch: " text " expected=" (json_encode expected) " actual=" (json_encode actual)))))))
	/* Candidate fixtures are outside up_*: the complete old-writer snapshot
	and all old query references remain immutable and independently mandatory. */
	(run "CREATE TABLE candidate_c04_rows(code INT NOT NULL PRIMARY KEY,payload INT NOT NULL)")
	(run "INSERT candidate_c04_rows VALUES(1,10),(2,20),(3,30)")
	(run "ALTER TABLE candidate_c04_rows ADD CONSTRAINT CK_candidate_c04_payload CHECK(payload>=0)")
	(rebuild (table "memcp-tests" false "candidate_c04_rows" false nil) true false nil true)
	(run "CREATE TABLE candidate_c04_rollback(code INT NOT NULL PRIMARY KEY,payload INT NOT NULL)")
	(run "INSERT candidate_c04_rollback VALUES(1,10),(2,20)")
	(run "CREATE TABLE candidate_c04_empty(code INT NOT NULL PRIMARY KEY,payload INT NOT NULL)")
	(run "BEGIN TRANSACTION")
	(run "ALTER TABLE candidate_c04_rows DROP CONSTRAINT CK_candidate_c04_payload")
	(run "UPDATE candidate_c04_rows SET payload=11 WHERE code=1")
	(run "DELETE candidate_c04_rows WHERE code=2")
	(run "INSERT candidate_c04_rows VALUES(4,40)")
	(run "ALTER TABLE candidate_c04_rows ADD ID INT NOT NULL IDENTITY(42,1)")
	(run "INSERT candidate_c04_rows(code,payload) VALUES(5,50)")
	(run "ALTER TABLE candidate_c04_rows ADD CONSTRAINT CK_candidate_c04_payload CHECK(payload>0)")
	(run "COMMIT TRANSACTION")
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50)))
	(expect "SELECT COUNT(*) AS n,COUNT(DISTINCT ID) AS distinct_ids,MIN(ID) AS lo,MAX(ID) AS hi,SUM(ID) AS total,IDENT_SEED(N'candidate_c04_rows') AS seed,IDENT_CURRENT(N'candidate_c04_rows') AS current_value,@@TRANCOUNT AS depth FROM candidate_c04_rows" (list "n" "distinct_ids" "lo" "hi" "total" "seed" "current_value" "depth") (list (list 4 4 42 45 174 42 45 0)))
	(expect "SELECT name FROM sys.check_constraints WHERE parent_object_id=OBJECT_ID(N'candidate_c04_rows') ORDER BY name" (list "name") (list (list "CK_candidate_c04_payload")))
	(run "BEGIN TRANSACTION")
	(run "ALTER TABLE candidate_c04_rollback ADD ID INT NOT NULL IDENTITY(500,1)")
	(run "INSERT candidate_c04_rollback(code,payload) VALUES(3,30)")
	(run "ROLLBACK TRANSACTION")
	(expect "SELECT code,payload,COL_LENGTH(N'candidate_c04_rollback',N'ID') AS identity_length,@@TRANCOUNT AS depth FROM candidate_c04_rollback ORDER BY code" (list "code" "payload" "identity_length" "depth") (list (list 1 10 nil 0) (list 2 20 nil 0)))
	(run "ALTER TABLE candidate_c04_empty ADD ID INT NOT NULL IDENTITY(7,1)")
	(expect "SELECT COUNT(*) AS n,IDENT_SEED(N'candidate_c04_empty') AS seed,IDENT_CURRENT(N'candidate_c04_empty') AS current_value FROM candidate_c04_empty" (list "n" "seed" "current_value") (list (list 0 7 7)))
	true)
