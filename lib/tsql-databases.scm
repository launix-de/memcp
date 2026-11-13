/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* SQL Server file declarations are virtual compatibility labels, never backend
configuration or host filenames. SIZE is an admission hint, not preallocation;
MemCP owns its actual layout and durability through the selected native engine. */
(define tsql_database_file (lambda (kind fields) (begin
	(define keys (map fields car))
	(if (equal? (count keys) (count (merge_unique (list keys)))) true
		(error "duplicate database file option"))
	(define options (merge fields))
	(if (and (has_assoc? options "name") (has_assoc? options "filename")) true
		(error "database file requires NAME and FILENAME"))
	(if (or (equal? (options "name") "") (> (utf16_length (options "name")) 128))
		(error "invalid logical file name") true)
	(define filename (options "filename"))
	(if (regexp_test filename "(?:^|[\\\\/])\\.\\.(?:[\\\\/]|$)")
		(error "parent traversal is not allowed in virtual file declarations") true)
	(define path (tsql_filesystem_path filename))
	(if (has? (list (tsql_filesystem_path (tsql_filesystem_location "data"))
		(tsql_filesystem_path (tsql_filesystem_location "log"))) (tsql_filesystem_parent path)) true
		(error "database file is outside the virtual data and log directories"))
	(if (regexp_test path (if (equal? kind "data") "\\.mdf$" "\\.ldf$")) true
		(error "database file has an unsupported virtual extension"))
	(if (has_assoc? options "size")
		(if (and (> (options "size") 0) (<= (options "size") 2147483647)) true
			(error "database file SIZE hint is outside the supported range")) true)
	(list "kind" kind "name" (options "name") "path" path))))
(define tsql_database_validate_files (lambda (files) (begin
	(define files (map files (lambda (file) (tsql_database_file (car file) (cadr file)))))
	(if (or (empty_list? files) (and (equal? (count (filter files (lambda (f) (equal? (f "kind") "data")))) 1)
		(<= (count (filter files (lambda (f) (equal? (f "kind") "log")))) 1))) true
		(error "unsupported database file layout"))
	(define paths (map files (lambda (file) (file "path"))))
	(define names (map files (lambda (file) (toLower (file "name")))))
	(if (and (equal? (count paths) (count (merge_unique (list paths))))
		(equal? (count names) (count (merge_unique (list names))))) true
		(error "database file declarations must have distinct names and paths"))
	true)))
(define tsql_create_database (lambda (tx name ifnotexists files session) (begin
	(if (session "transaction") (error "database DDL inside explicit transactions is not supported") true)
	/* Validate every data/log argument before publishing the native namespace. */
	(tsql_database_validate_files files)
	(create_sql_database tx name ifnotexists nil nil
		(sql_database_create_actor (tsql_session_actor session) name ifnotexists)))))

/* Native public DROP is immediate. Reject an explicit transaction before
resolving a relation or acquiring the database DDL gate; ROLLBACK cannot restore
the native files. Keep this check at execution so BEGIN and DROP in one batch
observe the same connection transaction, including dynamic execution. */
(define tsql_drop_table (lambda (database name ifexists session tx) (begin
	(if (session "transaction")
		(error "DROP TABLE inside explicit transactions is not supported") true)
	(define canonical (tsql_relation_name database name tx))
	(droptable database canonical ifexists))))
(define tsql_drop_database (lambda (tx name ifexists session) (begin
	(if (session "transaction")
		(error "DROP DATABASE inside explicit transactions is not supported") true)
	(drop_sql_database tx name ifexists))))

/* This fixed profile exposes native resource-management behavior honestly:
there is no SQL Server log-backup chain, automatic close, or automatic shrink.
FULL/BULK_LOGGED and ON resource options remain unsupported. TRUSTWORTHY is
inert frontend metadata only: no policy, impersonation, CLR or host-operation
admission consults this flag. Native administrators or explicit T-SQL database
authority authorize DDL independently of the inert compatibility flag. */
(define tsql_database_trustworthy (lambda (name)
	(coalesceNil (scan_lookup nil (table "system" false "tsql_database_options" false nil)
		(list 372734710317056 (scan_boundary "equal" "database" 0 0 true true "" false) "trustworthy")
		(list name)) false)))
(define tsql_alter_database (lambda (tx name option value session) (begin
	(tsql_require_permission (tsql_session_actor session) name "ALTER DATABASE")
	(if (session "transaction") (error "database DDL inside explicit transactions is not supported") true)
	(define database (database_name name true))
	(if database true (error "database does not exist: " name))
	(if (or (and (equal? option "recovery") (equal? value "simple"))
		(and (has? '("auto_close" "auto_shrink") option) (equal? value "off"))
		(and (equal? option "trustworthy") (has? '("on" "off") value))) true
		(error "unsupported database option profile: " option " " value))
	(sql_database_ddl database tx (lambda () (begin
		(if (database_name database false) true (error "database does not exist: " database))
		(if (equal? option "trustworthy")
			(insert (table "system" false "tsql_database_options" false tx) '("database" "trustworthy")
				(list (list database (equal? value "on"))) '("$update")
				(lambda (update) (update (list "trustworthy" (equal? value "on")))) false nil tx)
			true)
		0))))))
