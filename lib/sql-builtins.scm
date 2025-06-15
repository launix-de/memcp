(define sql_builtins (coalesce sql_builtins (newsession)))

/* all upper case */
/*(sql_builtins "HELLO" (lambda () "Hello world"))*/
(sql_builtins "UNIX_TIMESTAMP" now)
(sql_builtins "UNIX_TIMESTAMP" parse_date)
(sql_builtins "CURRENT_TIMESTAMP" now)
(sql_builtins "FLOOR" floor)
(sql_builtins "CEIL" ceil)
(sql_builtins "CEILING" ceil)
(sql_builtins "ROUND" round)
(sql_builtins "UPPER" toUpper)
(sql_builtins "LOWER" toLower)
(sql_builtins "PASSWORD" password)
