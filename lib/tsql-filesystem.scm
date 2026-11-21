/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* This frontend namespace has no mapping to an operating-system path. SQL
administration never authorizes host filesystem access. All compatibility
probes and advertised locations share these logical entries. There are no
files until an actual frontend file declaration owns their identity. */
(define tsql_filesystem_location (lambda (kind) (match kind
	"root" "C:\\MemCP\\"
	"data" "C:\\MemCP\\Data\\"
	"log" "C:\\MemCP\\Log\\"
	_ (error "unknown virtual storage location"))))
(define tsql_filesystem_directories '("c:/" "c:/memcp" "c:/memcp/data" "c:/memcp/log"))

/* Parse Windows syntax independently of the host OS. Reject device/UNC paths,
drive-relative names, alternate streams, wildcards and ambiguous Win32 names.
No decoding, filesystem fallback, symlink resolution or network lookup occurs. */
(define tsql_filesystem_path (lambda (path) (if (nil? path) nil (begin
	(define path (toLower (replace path "\\" "/")))
	(if (or (equal? path "") (> (strlen path) 4000)) (error "invalid virtual path") true)
	(if (or (< (strlen path) 3) (not (equal? (substr path 0 3) "c:/")))
		(error "path is outside the virtual drive") true)
	(define parts (split (substr path 3) "/"))
	(define reversed (reduce parts (lambda (stack part)
		(if (or (equal? part "") (equal? part ".")) stack
			(equal? part "..") (if (<= (count stack) 1)
				(error "path escapes the virtual namespace") (cdr stack))
			(or (regexp_test part "[:*?\"<>|\\x00-\\x1f]")
				(regexp_test part "[ .]$")
				(regexp_test part "^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\\..*)?$")) (error "invalid virtual path component")
			(cons part stack))) '()))
	(if (empty_list? reversed) "c:/"
		(concat "c:/" (reduce reversed (lambda (suffix part)
			(if (equal? suffix "") part (concat part "/" suffix))) "")))))))
(define tsql_filesystem_parent (lambda (path) (begin
	(define parts (split (substr path 3) "/"))
	(if (or (equal? path "c:/") (<= (count parts) 1)) "c:/"
		(concat "c:/" (reduce (slice parts 0 (- (count parts) 1))
			(lambda (prefix part) (if (equal? prefix "") part (concat prefix "/" part))) ""))))))
(define tsql_filesystem_stat (lambda (path) (begin
	(define canonical (tsql_filesystem_path path))
	/* Unknown entries are absent, never discovered through host I/O. These
	three bits describe the virtual namespace, not the physical backend. */
	(list "file" 0
		"directory" (if (and canonical (has? tsql_filesystem_directories canonical)) 1 0)
		"parent" (if (and canonical (has? tsql_filesystem_directories (tsql_filesystem_parent canonical))) 1 0)))))

/* Recognize a deliberately small command language, never a shell. Anchored
patterns consume the complete input; expansion, chaining, redirection and
control bytes are rejected before arguments enter the virtual path parser. */
(define tsql_filesystem_command (lambda (command) (begin
	(if (or (nil? command) (> (strlen command) 4000)
		(regexp_test command "[&|<>^%!;`\\x00-\\x1f]"))
		(error "unsupported virtual command syntax") true)
	(define path (lambda (token)
		(tsql_filesystem_path (if (equal? (substr token 0 1) "\"")
			(substr token 1 (- (strlen token) 2)) token))))
	(match command
		(regex "(?i)^DIR +(\"[^\"]+\"|[^ \"]+) *\\z" _ target)
		(list "list" (path target))
		(regex "(?i)^COPY +(\"[^\"]+\"|[^ \"]+) +(\"[^\"]+\"|[^ \"]+) *\\z" _ source target)
		(list "copy" (path source) (path target))
		(regex "(?i)^DEL +(\"[^\"]+\"|[^ \"]+) *\\z" _ target)
		(list "delete" (path target))
		_ (error "unsupported virtual command")))))

/* This profile lists only its bounded set of declared virtual directories.
It cannot report host drive identities, physical files or free capacity. */
(define tsql_filesystem_list (lambda (path) (begin
	(if (has? tsql_filesystem_directories path) true (error "virtual directory does not exist"))
	(map (filter tsql_filesystem_directories (lambda (entry)
		(and (not (equal? entry path)) (equal? (tsql_filesystem_parent entry) path))))
		(lambda (entry) (begin
			(define parts (split entry "/"))
			(list "name" (nth parts (- (count parts) 1)) "path" entry "directory" true)))))))

/* The caller passes concrete engine capabilities explicitly. No mutable
command registry, process runner or fallback exists. Missing write operations
fail before success can be reported or a virtual file can be invented. */
(define tsql_filesystem_execute (lambda (command operations) (begin
	(define recipe (tsql_filesystem_command command))
	(define operation (car recipe))
	(define result (if (equal? operation "list") (tsql_filesystem_list (cadr recipe))
		(begin
			(define implementation (operations operation))
			(if (nil? implementation) (error "virtual operation requires an engine capability: " operation) true)
			(apply implementation (cdr recipe)))))
	(list "operation" operation "result" result))))
