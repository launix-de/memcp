/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* A bounded virtual SQL-server environment, never host introspection. The
version is the SQL-server compatibility level, not MemCP's binary version.
Windows labels describe the virtual path namespace; processor/memory entries
are a virtual slot and unknown memory, not the machine running this process. */
(define tsql_server_compatibility_parts '(16 0 0 1))
(define tsql_server_compatibility_version (concat
	(nth tsql_server_compatibility_parts 0) "." (nth tsql_server_compatibility_parts 1) "."
	(nth tsql_server_compatibility_parts 2) "." (nth tsql_server_compatibility_parts 3)))
/* @@MICROSOFTVERSION packs major/minor/build into 8/8/16 bits; revision
is separate. xp_msver's Internal_Value uses its own existing encoding. */
(define tsql_server_microsoft_version (+
	(bit_shift_left (nth tsql_server_compatibility_parts 0) 24)
	(bit_shift_left (nth tsql_server_compatibility_parts 1) 16)
	(nth tsql_server_compatibility_parts 2)))
(define tsql_server_info_entries (list
	(list 1 "ProductName" nil "MemCP")
	(list 2 "ProductVersion" 1048576 tsql_server_compatibility_version)
	(list 3 "Language" 1033 "English")
	(list 4 "Platform" nil "Windows")
	(list 15 "WindowsVersion" nil "10.0")
	(list 16 "ProcessorCount" 1 "1")
	(list 19 "PhysicalMemory" nil nil)))
(define tsql_server_info_rows (lambda (option)
	(if (nil? option) tsql_server_info_entries
		(begin
			(define entry (find tsql_server_info_entries (lambda (entry) (equal?? (nth entry 1) option)) nil))
			(if entry (list entry) (error "unknown virtual server information option"))))))
(define tsql_server_profile_property (lambda (property) (match property
	"productversion" (list "type" "NVARCHAR" "value" tsql_server_compatibility_version)
	"collation" (list "type" "NVARCHAR" "value" "utf8mb4_general_ci")
	"isclustered" (list "type" "INT" "value" 0)
	_ nil)))

/* Registry-shaped reads are an exact virtual allowlist. No OS registry, path
resolution, environment expansion, network lookup or arbitrary key traversal is
available, even to a SQL administrator. Mutating registry calls have no route. */
(define tsql_virtual_registry_read ((lambda (location) (lambda (root key name)
	(if (or (nil? root) (nil? key) (nil? name)) (error "virtual registry arguments cannot be NULL")
		(not (equal?? root "HKEY_LOCAL_MACHINE")) (error "virtual registry root is unavailable")
		(begin
			(define key (toLower key))
			(define name (toLower name))
			(if (and (equal? key "software\\microsoft\\mssqlserver\\setup") (equal? name "sqlpath")) (location "root")
				(and (equal? key "software\\microsoft\\mssqlserver\\mssqlserver") (equal? name "defaultdata")) (location "data")
				(and (equal? key "software\\microsoft\\mssqlserver\\mssqlserver") (equal? name "defaultlog")) (location "log")
				(error "virtual registry key or value is unavailable")))))) tsql_filesystem_location))
