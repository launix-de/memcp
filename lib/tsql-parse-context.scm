/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-source-settings.scm")

/* QUOTED_IDENTIFIER belongs to lexical analysis. The shared SQL parser only
sees unambiguous bracket identifiers and single-quoted literals; comments and
literal contents are never searched for SET statements. All state below is
bounded by this source text, never by table cardinality or a row callback. */
(define tsql_initialize_parse_context (lambda (session) (begin
	(if (nil? (session "__tsql_ansi_nulls")) (session "__tsql_ansi_nulls" true) true)
	(if (nil? (session "__tsql_quoted_identifier")) (session "__tsql_quoted_identifier" true) true)
	session)))

(define tsql_procedure_language? (lambda (language)
	(equal? (tsql_source_module_kind language) "P")))
(define tsql_procedure_language (lambda (session)
	(tsql_source_language "tsql-procedure" session)))
(define tsql_statement_language (lambda (session)
	(tsql_source_language "tsql-statement" session)))
(define tsql_cached_batch_context (lambda (schema text quoted static_module ansi)
	(tsql_script_cache (json_encode (list "parse-context" schema text quoted static_module ansi)) nil
		(close_procedure (lambda (_tx) (tsql_parse_batch_context schema text quoted static_module ansi))))))
(define tsql_context_tokens (lambda (text)
	(merge (map (tsql_script_tokens text) (lambda (token)
		(if (equal? (car token) "text")
			(map (regexp_matches (cadr token) "[@#]?[a-zA-Z_][a-zA-Z0-9_]*|[0-9]+|[ \\t]+|.")
				(lambda (part) (list (if (regexp_test part "^[@#]?[a-zA-Z_][a-zA-Z0-9_]*$") "word" "text") part)))
			(list token)))))))
(define tsql_context_double_quote (lambda (text quoted)
	(if (and (>= (strlen text) 2) (equal? (substr text 0 1) "\"")) (begin
		(define value (replace (substr text 1 (- (strlen text) 2)) "\"\"" "\""))
		(if quoted (concat "[" (replace value "]" "]]") "]")
			(concat "'" (replace value "'" "''") "'"))) text)))

/* The recursive script lexer has already validated this complete block
comment. Keep its outer delimiters for the parser's ordinary regex skipper,
but neutralize nested delimiters without changing bytes, runes or newlines.
Original module source and span coordinates continue to use the raw token.
https://learn.microsoft.com/sql/t-sql/language-elements/slash-star-comment-transact-sql */
(define tsql_context_block_comment (lambda (text)
	(concat "/*" (replace (replace (substr text 2 (- (strlen text) 4)) "/*" "  ") "*/" "  ") "*/")))

/* An ordinary module declaration owns the rest of its batch. Its static body
uses the mode at CREATE/ALTER, and SET inside that body cannot change parsing.
AS after a procedure parameter name is a type marker, not the body boundary.
Unsupported header options still go through the real declaration grammar. */
(define tsql_context_header (lambda (stage word)
	(match stage
		"first" (match word "CREATE" "create" "ALTER" "alter" _ "none")
		"create" (if (equal? word "OR") "or"
			(has? '("PROC" "PROCEDURE" "TRIGGER" "VIEW" "FUNCTION") word) "module" "none")
		"or" (if (equal? word "ALTER") "alter" "none")
		"alter" (if (has? '("PROC" "PROCEDURE" "TRIGGER" "VIEW" "FUNCTION") word) "module" "none")
		_ stage)))
(define tsql_parse_context (lambda (text initial static_module) (begin
	/* Retain one fixed-width transition per token. Native lists are flat, so
	prepending every new span/part to its complete prefix copies O(n²) cells.
	produce owns the linear output collection; lexical transitions remain pure. */
	(define tokens (tsql_context_tokens text))
	(define initial_state (list "index" 0 "mode" initial "header" "first" "module" nil "body" nil "depth" 0
		"previous" nil "header_options" false "setting" "none" "has_quoted" false "offset" 0 "canonical_offset" 0))
	(define states (produce initial_state (lambda (state) (<= (state "index") (count tokens)))
		(lambda (state) (if (equal? (state "index") (count tokens)) (list "index" (+ (count tokens) 1)) (begin
			(define token (nth tokens (state "index")))
			(define kind (car token))
			(define raw (cadr token))
			(define word (if (equal? kind "word") (toUpper raw) nil))
			(define trivia (or (equal? kind "comment") (equal? kind "newline")
				(and (equal? kind "text") (equal? (strtrim raw) ""))))
			(define header (if (or trivia (nil? word)) (state "header")
				(tsql_context_header (state "header") word)))
			(define module_mode (if (and (equal? header "module") (nil? (state "module")))
				(state "mode") (state "module")))
			(define fixed (or static_module (not (nil? module_mode))))
			(define body (if (and (not (nil? module_mode)) (nil? (state "body"))
				(equal? word "AS") (equal? (state "depth") 0)
				(not (regexp_test (coalesceNil (state "previous") "") "^@"))
				/* The AS inside WITH EXECUTE AS introduces the database
				execution principal, not the persisted module body. */
				(not (and (state "header_options")
					(has? '("EXEC" "EXECUTE") (toUpper (coalesceNil (state "previous") ""))))))
				(+ (state "offset") (strlen raw)) (state "body")))
			(define setting (state "setting"))
			(define has_quoted (state "has_quoted"))
			(define mode (state "mode"))
			(define cooked (if (equal? kind "quoted") (tsql_context_double_quote raw mode)
				(and (equal? kind "comment") (equal? (substr raw 0 2) "/*"))
				(tsql_context_block_comment raw) raw))
			(define next (if (or trivia fixed) (list setting has_quoted mode)
				(and (equal? setting "option") (has? '("ON" "OFF") word))
				(list "none" false (if has_quoted (equal? word "ON") mode))
				(and (has? '("first" "comma") setting) (has? '("ANSI_NULLS" "QUOTED_IDENTIFIER" "ANSI_PADDING") word))
				(list "option" (or has_quoted (equal? word "QUOTED_IDENTIFIER")) mode)
				(and (equal? setting "option") (equal? raw ",")) (list "comma" has_quoted mode)
				(equal? word "SET") (list "first" false mode)
				(list "none" false mode)))
			(list "index" (+ (state "index") 1) "mode" (nth next 2) "header" header "module" module_mode "body" body
				"module_kind" (if (and (equal? header "module") (nil? (state "module_kind"))) word (state "module_kind"))
				"depth" (+ (state "depth") (if (equal? raw "(") 1 (equal? raw ")") -1 0))
				"previous" (if trivia (state "previous") raw)
				"header_options" (or (state "header_options")
					(and (not (nil? module_mode)) (nil? body) (equal? (state "depth") 0) (equal? word "WITH")))
				"setting" (car next) "has_quoted" (cadr next)
				"offset" (+ (state "offset") (strlen raw))
				"canonical_offset" (+ (state "canonical_offset") (strlen cooked))
				"span" (list (state "canonical_offset") (state "offset")
					(strlen cooked) (strlen raw) kind mode word)
				"part" cooked))))))
	(define state (nth states (count tokens)))
	(define transitions (cdr states))
	(list "text" (apply concat (map transitions (lambda (state) (state "part")))) "quoted_identifier" (state "mode")
		"source" text "spans" (map transitions (lambda (state) (state "span")))
		"module_quoted_identifier" (state "module") "module_kind" (state "module_kind")
		"module_source" (if (nil? (state "body")) nil (strtrim (substr text (state "body"))))))))

/* Source spans belong to lexical analysis, not to query binding. A deferred
VIEW statement uses its original bytes and their parse mode when it reaches
the ordinary executor. Neither the scope nor a parser procedure is retained. */
(define tsql_context_span (lambda (spans position low high)
	(if (>= low high) nil (begin
		(define middle (intdiv (+ low high) 2))
		(define span (nth spans middle))
		(if (< position (car span)) (tsql_context_span spans position low middle)
			(>= position (+ (car span) (nth span 2))) (tsql_context_span spans position (+ middle 1) high)
			span)))))
(define tsql_context_source_offset (lambda (context position)
	(if (equal? position (strlen (context "text"))) (strlen (context "source")) (begin
		(define spans (context "spans"))
		(define span (tsql_context_span spans position 0 (count spans)))
		(if (nil? span) (error "SQL source capture is outside its lexical input") true)
		(define relative (- position (car span)))
		(if (or (equal? relative 0) (equal? (nth span 2) (nth span 3)))
			(+ (cadr span) relative) (error "SQL source capture splits a rewritten quoted token"))))))
(define tsql_context_statement (lambda (context statement) (begin
	(define parts (split (context "text") statement))
	(define size (strlen statement))
	(define spans (context "spans"))
	(define matches (reduce (produceN (- (count parts) 1)) (lambda (state index) (begin
		(define start (+ (car state) (strlen (nth parts index))))
		(define end (+ start size))
		(define span (tsql_context_span spans start 0 (count spans)))
		(define valid (if (nil? span) false (and (equal? (car span) start)
			(equal? (nth span 4) "word") (has? '("CREATE" "ALTER") (nth span 6)))))
		(list end (if valid (begin
			(define original (tsql_context_source_offset context start))
			(cons (list (substr (context "source") original
				(- (tsql_context_source_offset context end) original)) (nth span 5)) (cadr state)))
			(cadr state))))) (list 0 '())))
	(define unique (merge_unique (list (cadr matches))))
	(if (equal? (count unique) 1) (list "statement-source" (car (car unique)) (cadr (car unique)))
		(error "VIEW definition requires an unambiguous original SQL source span")))))
(define tsql_parse_source_session (lambda (parent quoted)
	(lambda args (match args
		'() (merge_unique (list '("__tsql_quoted_identifier") (parent)))
		'(key) (if (equal? key "__tsql_quoted_identifier") quoted (parent key))
		'(key value) (if (equal? key "__tsql_quoted_identifier") (error "source parse context is immutable")
			(parent key value))
		_ (apply parent args)))))
