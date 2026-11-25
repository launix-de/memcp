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
	/* The existing identity hook already owns this fixture. Do not create it
	again or touch any up_* table, old reference, predecessor code or oracle.
	Its complete subsequent program adds only (code6,payload60); all original
	payloads11/30/40/50 remain distinct. An UPDATE violation avoids consuming
	the identity value46 required by that unchanged subsequent hook. */
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50)))
	(expect_indexes nil (list))
	(run "BEGIN TRANSACTION")
	(define tx (caller "__memcp_tx"))
	(define target (table "memcp-tests" false "candidate_c04_rows" false tx))
	(define declarations (get_schema "memcp-tests" "candidate_c04_rows" tx))
	(define physical (lambda (name) (begin
		(define declaration (find declarations (lambda (column) (equal? (column "Field") name)) nil))
		(if (nil? declaration) (error (concat "missing candidate fixture column: " name))
			(sql_column_storage_name declaration)))))
	(define payload (physical "payload"))
	(define code (physical "code"))
	(if (create_index target "candidate_c07_nonunique" (list payload) false tx
		(list "included" (list code) "descending" (list true) "placement" "nonclustered")) true
		(error "candidate nonunique declaration already exists"))
	(if (create_index target "candidate_c07_unique" (list payload) true tx
		(list "descending" (list false) "placement" "nonclustered" "nulls_not_distinct" true)) true
		(error "candidate unique declaration already exists"))
	(if (create_index target "candidate_c07_retired" (list code) false tx) true
		(error "candidate temporary index already exists"))
	(if (drop_index target "candidate_c07_retired" tx false) true
		(error "candidate exact index retirement failed"))
	(run "SAVE TRANSACTION c07_index_header")
	(if (create_index target "candidate_c07_rewound" (list code) false tx) true
		(error "candidate rewind declaration already exists"))
	(run "ROLLBACK TRANSACTION c07_index_header")
	(expect_indexes tx (list nonunique enforcing))
	(run "UPDATE candidate_c04_rows SET payload=31 WHERE code=3")
	(expect "SELECT code,payload,@@TRANCOUNT AS depth FROM candidate_c04_rows ORDER BY code" (list "code" "payload" "depth") (list (list 1 11 1) (list 3 31 1) (list 4 40 1) (list 5 50 1)))
	(run "UPDATE candidate_c04_rows SET payload=30 WHERE code=3")
	(run "COMMIT TRANSACTION")
	(expect_indexes nil (list nonunique enforcing))
	(expect "BEGIN TRY UPDATE candidate_c04_rows SET payload=11 WHERE code=3; END TRY BEGIN CATCH SELECT ERROR_NUMBER() AS number; END CATCH" (list "number") (list (list 1062)))
	(expect "SELECT code,payload FROM candidate_c04_rows ORDER BY code" (list "code" "payload") (list (list 1 11) (list 3 30) (list 4 40) (list 5 50)))
	(expect "SELECT IDENT_CURRENT(N'candidate_c04_rows') AS current_value,@@TRANCOUNT AS depth" (list "current_value" "depth") (list (list 45 0)))
	true)
