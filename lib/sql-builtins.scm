(define sql_builtins (coalesce sql_builtins (newsession)))

/* all upper case */
/*(sql_builtins "HELLO" (lambda () "Hello world"))*/

/* time */
(sql_builtins "UNIX_TIMESTAMP" now)
(sql_builtins "UNIX_TIMESTAMP" parse_date)
(sql_builtins "CURRENT_TIMESTAMP" now)
(sql_builtins "NOW" now)

/* time */
(sql_builtins "FROM_UNIXTIME" (lambda (ts) (if (nil? ts) nil (format_date (simplify ts) "%Y-%m-%d %H:%i:%s"))))
(sql_builtins "DATE_FORMAT" format_date)
(sql_builtins "STR_TO_DATE" str_to_date)
(sql_builtins "DATE" date_trunc_day)
(sql_builtins "CURRENT_DATE" current_date)
(sql_builtins "DATEDIFF" datediff)

/* math */
(sql_builtins "FLOOR" floor)
(sql_builtins "CEIL" ceil)
(sql_builtins "CEILING" ceil)
(sql_builtins "ROUND" round)
(sql_builtins "ABS" sql_abs)
(sql_builtins "SQRT" sqrt)
(sql_builtins "RAND" sql_rand)
(sql_builtins "RANDOM" sql_rand)
(sql_builtins "GREATEST" max)
(sql_builtins "LEAST" min)

/* strings */
(sql_builtins "UPPER" toUpper)
(sql_builtins "LOWER" toLower)
(sql_builtins "PASSWORD" password)
/* Base64 helpers */
(sql_builtins "TO_BASE64" base64_encode)
(sql_builtins "FROM_BASE64" base64_decode)
/* SQL LENGTH(str): NULL-safe wrapper around strlen */
(sql_builtins "LENGTH" (lambda (x) (if (nil? x) nil (strlen x))))
(sql_builtins "CHAR_LENGTH" (lambda (x) (if (nil? x) nil (strlen x))))
(sql_builtins "CHARACTER_LENGTH" (lambda (x) (if (nil? x) nil (strlen x))))
(sql_builtins "REPEAT" string_repeat)
/* SQL REPLACE(str, from, to) */
(sql_builtins "REPLACE" (lambda (s from to) (if (nil? s) nil (replace s from to))))
/* TRIM/LTRIM/RTRIM are handled as explicit parser rules in sql-parser.scm and psql-parser.scm */
/* SQL SUBSTR/SUBSTRING: 1-based index via Go primitive */
(sql_builtins "SUBSTR" sql_substr)
(sql_builtins "SUBSTRING" sql_substr)
(sql_builtins "REGEXP_REPLACE" regexp_replace)
(sql_builtins "REGEXP_SUBSTR" (lambda (s pattern)
	(if (or (nil? s) (nil? pattern))
		nil
		(eval (list 'match s (list 'regex (concat "(" pattern ")") '_ 'rx_match) 'rx_match nil))
	)
))

/* null check */
(sql_builtins "ISNULL" (lambda (x) (if (nil? x) 1 0)))

/* phonetic */
(sql_builtins "SOUNDEX" (lambda (s) (if (nil? s) nil (begin
	(define input (toUpper (concat s)))
	(define codes (newsession))
	(codes "B" "1") (codes "F" "1") (codes "P" "1") (codes "V" "1")
	(codes "C" "2") (codes "G" "2") (codes "J" "2") (codes "K" "2") (codes "Q" "2") (codes "S" "2") (codes "X" "2") (codes "Z" "2")
	(codes "D" "3") (codes "T" "3")
	(codes "L" "4")
	(codes "M" "5") (codes "N" "5")
	(codes "R" "6")
	(define first (sql_substr input 1 1))
	(define len (strlen input))
	(define state (for (list 2 first (coalesce (codes first) "0"))
		(lambda (i result prev) (and (<= i len) (< (strlen result) 4)))
		(lambda (i result prev) (begin
			(define ch (sql_substr input i 1))
			(define code (codes ch))
			(if (and (not (nil? code)) (not (equal? code prev)))
				(list (+ i 1) (concat result code) code)
				(list (+ i 1) result (coalesce code "0")))
		))
	))
	(define result (nth state 1))
	(if (< (strlen result) 4)
		(concat result (sql_substr "0000" 1 (- 4 (strlen result))))
		result)
))))

/* vectors */
(sql_builtins "VECTOR_DISTANCE" dot)
(sql_builtins "STRING_TO_VECTOR" json_decode)
(sql_builtins "VECTOR_TO_STRING" json_encode)
(sql_builtins "VECTOR_DIM" json_encode)

/* management: use SQL statements instead (REBUILD, SHOW SHARDS, etc.) */
