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

/* math */
(sql_builtins "FLOOR" floor)
(sql_builtins "CEIL" ceil)
(sql_builtins "CEILING" ceil)
(sql_builtins "ROUND" round)
(sql_builtins "ABS" sql_abs)
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

/* vectors */
(sql_builtins "VECTOR_DISTANCE" dot)
(sql_builtins "STRING_TO_VECTOR" json_decode)
(sql_builtins "VECTOR_TO_STRING" json_encode)
(sql_builtins "VECTOR_DIM" json_encode)

/* management: use SQL statements instead (REBUILD, SHOW SHARDS, etc.) */
