/*
Copyright (C) 2023 - 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

(define sql_builtins (coalesce sql_builtins (newsession)))

/* all upper case */
/*(sql_builtins "HELLO" (lambda () "Hello world"))*/

/* time */
(sql_builtins "UNIX_TIMESTAMP" unix_timestamp)
(sql_builtins "CURRENT_TIMESTAMP" now)
(sql_builtins "NOW" now)

/* time */
(sql_builtins "FROM_UNIXTIME" from_unixtime)
(sql_builtins "DATE_FORMAT" format_date)
(sql_builtins "STR_TO_DATE" str_to_date)
(sql_builtins "DATE" date_trunc_day)
(sql_builtins "CURRENT_DATE" current_date)
(sql_builtins "DATEDIFF" datediff)
(sql_builtins "TIMESTAMPDIFF" timestampdiff)
/* timezone functions */
(sql_builtins "CONVERT_TZ" convert_tz)
(sql_builtins "UTC_TIMESTAMP" utc_timestamp)
(sql_builtins "UTC_DATE" utc_date)
(sql_builtins "UTC_TIME" utc_time)
(sql_builtins "SYSDATE" sysdate)
/* PostgreSQL aliases */
(sql_builtins "TO_TIMESTAMP" from_unixtime)
(sql_builtins "CLOCK_TIMESTAMP" now) /* approximation: same as now() */
(sql_builtins "TRANSACTION_TIMESTAMP" now)
/* MySQL-style date part extraction shortcuts */
(sql_builtins "YEAR" (lambda (d) (extract_date d "YEAR")))
(sql_builtins "MONTH" (lambda (d) (extract_date d "MONTH")))
(sql_builtins "DAY" (lambda (d) (extract_date d "DAY")))
(sql_builtins "HOUR" (lambda (d) (extract_date d "HOUR")))
(sql_builtins "MINUTE" (lambda (d) (extract_date d "MINUTE")))
(sql_builtins "SECOND" (lambda (d) (extract_date d "SECOND")))
(sql_builtins "DAYOFMONTH" (lambda (d) (extract_date d "DAY")))
(sql_builtins "DAYOFWEEK" (lambda (d) (extract_date d "DAYOFWEEK")))
(sql_builtins "WEEKDAY" (lambda (d) (extract_date d "WEEKDAY")))
(sql_builtins "WEEK" (lambda (d) (extract_date d "WEEK")))
(sql_builtins "QUARTER" (lambda (d) (extract_date d "QUARTER")))
(sql_builtins "YEARWEEK" (lambda (d mode) (if (nil? d) nil (+ (* (extract_date d "YEAR") 100) (extract_date d "WEEK")))))

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

/* MySQL JSON functions. */
(sql_builtins "JSON_ARRAY" json_array)
(sql_builtins "JSON_OBJECT" json_object)
(sql_builtins "JSON_QUOTE" json_quote)
(sql_builtins "JSON_UNQUOTE" json_unquote)
(sql_builtins "JSON_VALID" json_valid)
(sql_builtins "JSON_TYPE" json_type)
(sql_builtins "JSON_DEPTH" json_depth)
(sql_builtins "JSON_LENGTH" json_length)
(sql_builtins "JSON_EXTRACT" json_extract)
(sql_builtins "JSON_KEYS" json_keys)
(sql_builtins "JSON_CONTAINS_PATH" json_contains_path)
(sql_builtins "JSON_CONTAINS" json_contains)
(sql_builtins "JSON_OVERLAPS" json_overlaps)
(sql_builtins "JSON_PRETTY" json_pretty)
(sql_builtins "JSON_STORAGE_SIZE" json_storage_size)
(sql_builtins "JSON_STORAGE_FREE" json_storage_free)
(sql_builtins "JSON_SCHEMA_VALID" json_schema_valid)
(sql_builtins "JSON_SCHEMA_VALIDATION_REPORT" json_schema_validation_report)
(sql_builtins "JSON_SET" json_set)
(sql_builtins "JSON_INSERT" json_insert)
(sql_builtins "JSON_REPLACE" json_replace)
(sql_builtins "JSON_REMOVE" json_remove)
(sql_builtins "JSON_ARRAY_APPEND" json_array_append)
(sql_builtins "JSON_ARRAY_INSERT" json_array_insert)
(sql_builtins "JSON_MERGE_PATCH" json_merge_patch)
(sql_builtins "JSON_MERGE_PRESERVE" json_merge_preserve)
(sql_builtins "JSON_MERGE" json_merge_preserve)
(sql_builtins "JSON_SEARCH" json_search)
(sql_builtins "JSON_VALUE" json_value)
(sql_aggregates "JSON_ARRAYAGG" (list (quote json_arrayagg_reduce) nil true))

/* PostgreSQL JSON/jsonb scalar functions. Set-returning functions are parsed
through the PostgreSQL table-function rules. */
(sql_builtins "TO_JSON" pg_to_json)
(sql_builtins "TO_JSONB" pg_to_json)
(sql_builtins "ARRAY_TO_JSON" pg_to_json)
(sql_builtins "ROW_TO_JSON" pg_row_to_json)
(sql_builtins "JSON_BUILD_ARRAY" pg_json_build_array)
(sql_builtins "JSONB_BUILD_ARRAY" pg_json_build_array)
(sql_builtins "JSON_BUILD_OBJECT" pg_json_build_object)
(sql_builtins "JSONB_BUILD_OBJECT" pg_json_build_object)
(sql_builtins "JSONB_OBJECT" pg_json_object)
(sql_builtins "JSON_SCALAR" pg_to_json)
(sql_builtins "JSON_ARRAY_LENGTH" pg_json_array_length)
(sql_builtins "JSONB_ARRAY_LENGTH" pg_json_array_length)
(sql_builtins "JSON_EXTRACT_PATH" pg_json_extract_path)
(sql_builtins "JSONB_EXTRACT_PATH" pg_json_extract_path)
(sql_builtins "JSON_EXTRACT_PATH_TEXT" pg_json_extract_path_text)
(sql_builtins "JSONB_EXTRACT_PATH_TEXT" pg_json_extract_path_text)
(sql_builtins "JSONB_SET" pg_jsonb_set)
(sql_builtins "JSONB_SET_LAX" pg_jsonb_set_lax)
(sql_builtins "JSONB_INSERT" pg_jsonb_insert)
(sql_builtins "JSON_STRIP_NULLS" pg_json_strip_nulls)
(sql_builtins "JSONB_STRIP_NULLS" pg_json_strip_nulls)
(sql_builtins "JSONB_PRETTY" json_pretty)
(sql_builtins "JSON_TYPEOF" pg_json_typeof)
(sql_builtins "JSONB_TYPEOF" pg_json_typeof)
(sql_builtins "JSONB_POPULATE_RECORD_VALID" pg_jsonb_populate_record_valid)
(sql_builtins "JSONB_PATH_EXISTS" pg_jsonb_path_exists)
(sql_builtins "JSONB_PATH_EXISTS_TZ" pg_jsonb_path_exists)
(sql_builtins "JSONB_PATH_MATCH" pg_jsonb_path_match)
(sql_builtins "JSONB_PATH_MATCH_TZ" pg_jsonb_path_match)
(sql_builtins "JSONB_PATH_QUERY_ARRAY" pg_jsonb_path_query_array)
(sql_builtins "JSONB_PATH_QUERY_ARRAY_TZ" pg_jsonb_path_query_array)
(sql_builtins "JSONB_PATH_QUERY_FIRST" pg_jsonb_path_query_first)
(sql_builtins "JSONB_PATH_QUERY_FIRST_TZ" pg_jsonb_path_query_first)

/* session / processlist */
(sql_builtins "DATABASE" (lambda () (coalesceNil (session "schema") nil)))
(sql_builtins "CURRENT_USER" (lambda () (match (session "username")
	nil nil
	username (concat username "@%")
)))
(sql_builtins "USER" (lambda () (match (session "username")
	nil nil
	username (concat username "@%")
)))
(sql_builtins "SESSION_USER" (lambda () (match (session "username")
	nil nil
	username (concat username "@%")
)))
(sql_builtins "CONNECTION_ID" connection_id)
(sql_builtins "KILL_QUERY" kill_query)

/* management: use SQL statements instead (REBUILD, SHOW SHARDS, etc.) */
