(define sql_builtins (coalesce sql_builtins (newsession)))

/* all upper case */
/*(sql_builtins "HELLO" (lambda () "Hello world"))*/

/* time */
(sql_builtins "UNIX_TIMESTAMP" now)
(sql_builtins "UNIX_TIMESTAMP" parse_date)
(sql_builtins "CURRENT_TIMESTAMP" now)

/* math */
(sql_builtins "FLOOR" floor)
(sql_builtins "CEIL" ceil)
(sql_builtins "CEILING" ceil)
(sql_builtins "ROUND" round)

/* strings */
(sql_builtins "UPPER" toUpper)
(sql_builtins "LOWER" toLower)
(sql_builtins "PASSWORD" password)

/* vectors */
(sql_builtins "VECTOR_DISTANCE" dot)
(sql_builtins "STRING_TO_VECTOR" json_decode)
(sql_builtins "VECTOR_TO_STRING" json_encode)
(sql_builtins "VECTOR_DIM" json_encode)
