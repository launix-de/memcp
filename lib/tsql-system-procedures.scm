/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Actual frontend system-procedure declarations are shared by dispatch and
permission introspection. This is not a catalog of unimplemented SQL Server
procedures: every entry below has an execution owner in tsql_invoke_procedure.
Use that dispatcher's existing lowercase identity, not the broader Unicode
SimpleFold relation (SQL2022 rejects sys.ſp_tables: https://dbfiddle.uk/MJkxE5pd).
Actual P/X kinds for the original entries: https://dbfiddle.uk/X483JyiE. */
(define tsql_system_procedure_declarations (list
	(list "sp_describe_undeclared_parameters" "X" "metadata" "DATABASE ACCESS")
	(list "sp_tables" "P" "metadata" "DATABASE ACCESS")
	(list "sp_columns" "P" "metadata" "DATABASE ACCESS")
	(list "sp_pkeys" "P" "metadata" "DATABASE ACCESS")
	(list "sp_statistics" "P" "metadata" "DATABASE ACCESS")
	(list "sp_fkeys" "P" "metadata" "DATABASE ACCESS")
	(list "sp_datatype_info" "P" "metadata" "DATABASE ACCESS")
	(list "sp_datatype_info_100" "P" "metadata" "DATABASE ACCESS")
	(list "sp_special_columns" "P" "metadata" "DATABASE ACCESS")
	(list "sp_tablecollations" "P" "metadata" "DATABASE ACCESS")
	(list "sp_tablecollations_100" "P" "metadata" "DATABASE ACCESS")
	(list "sp_executesql" "X" "dynamic" "DATABASE ACCESS")
	(list "sp_refreshview" "P" "command" "DATABASE ACCESS")
	(list "sp_rename" "P" "command" "DATABASE ACCESS")
	(list "sp_changedbowner" "P" "command" "DATABASE ACCESS")
	(list "xp_fileexist" "X" "command" "DATABASE ACCESS")
	(list "xp_cmdshell" "X" "command" "NATIVE ADMIN")
	(list "xp_msver" "X" "command" "DATABASE ACCESS")
	(list "xp_regread" "X" "command" "DATABASE ACCESS")
	(list "xp_instance_regread" "X" "command" "DATABASE ACCESS")
	(list "xp_logevent" "X" "command" "NATIVE ADMIN")))
(define tsql_system_procedure_declaration (close_procedure ((lambda (declarations)
	(lambda (name) (find declarations (lambda (declaration)
		(equal? (car declaration) (toLower name))) nil))) tsql_system_procedure_declarations)))
