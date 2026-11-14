/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* FORMAT owns .NET pattern and culture semantics in the SQL frontend.
The native date/string primitives receive plain values, never SQL types.
https://learn.microsoft.com/en-us/sql/t-sql/functions/format-transact-sql
https://learn.microsoft.com/en-us/dotnet/standard/base-types/custom-numeric-format-strings
https://learn.microsoft.com/en-us/dotnet/standard/base-types/custom-date-and-time-format-strings */
(define tsql_format_culture (close_procedure ((lambda (raise_error)
	(lambda (culture)
		(if (or (nil? culture) (equal? culture ""))
			(raise_error 9818 "FORMAT culture must not be NULL or empty" 1)
			(begin
				(define culture (toLower culture))
				(if (has? '("en" "en-us" "en-gb") culture) "us_english"
					(has? '("de" "de-de") culture) "Deutsch"
					(error "unsupported FORMAT culture: " culture)))))) tsql_raise_error)))

/* Decimal rounding operates on declared numeric text. Integral BIGINT values
and DECIMAL(p,0) never pass through floating-point conversion or ABS(MIN_INT).
The carry is a bounded scalar pair; its digits belong to one scalar value. */
(define tsql_format_zero (close_procedure ((lambda (culture_profile)
	(lambda (value pattern culture) (begin
		(culture_profile culture)
		(if (nil? value) nil (nil? pattern) (error "unsupported FORMAT default numeric pattern") (begin
			(if (regexp_test pattern "^0+$") true
				(error "unsupported FORMAT numeric pattern: " pattern))
			(define negative (equal? (substr value 0 1) "-"))
			(define parts (split (if negative (substr value 1) value) "."))
			(define digits (car parts))
			(define fraction (if (> (count parts) 1) (cadr parts) ""))
			(define carry (and (> (strlen fraction) 0)
				(has? '("5" "6" "7" "8" "9") (substr fraction 0 1))))
			(define rounded (reduce (reverse (split digits "")) (lambda (state digit)
				(if (not (car state)) (list false (concat digit (cadr state)))
					(equal? digit "9") (list true (concat "0" (cadr state)))
					(list false (concat (+ (simplify digit) 1) (cadr state))))) (list carry "")))
			(define digits (concat (if (car rounded) "1" "") (cadr rounded)))
			(concat (if (and negative (not (equal? digits "0"))) "-" "")
				(string_repeat "0" (max 0 (- (strlen pattern) (strlen digits)))) digits)))))) tsql_format_culture)))

(define tsql_format_integral_text (close_procedure (lambda (value)
	(if (nil? value) nil (string value)))))
(define tsql_format_float_text (close_procedure (lambda (value)
	(if (nil? value) nil (format_fixed (round value) 0)))))

/* .NET's German month abbreviations differ from SQL DATENAME (Mrz/Mär).
These bounded locale tokens are frontend data, independent of date carriers. */
(define tsql_format_short_months '("Jan" "Feb" "Mrz" "Apr" "Mai" "Jun" "Jul" "Aug" "Sep" "Okt" "Nov" "Dez"))
(define tsql_format_short_weekdays '("Mo" "Di" "Mi" "Do" "Fr" "Sa" "So"))
(define tsql_format_calendar_token (close_procedure ((lambda (months weekday short_months short_weekdays english_months)
	(lambda (value token language) (begin
		(define width (strlen token))
		(define head (substr token 0 1))
		(if (equal? head "\\") (if (equal? width 2) (substr token 1) nil)
			(or (equal? head "'") (equal? head "\""))
			(if (and (>= width 2) (equal? (substr token (- width 1)) head))
				(regexp_replace (substr token 1 (- width 2)) "\\\\(.)" "$1") nil)
			(equal? head "%") nil
			(equal? head "d") (if (>= width 4) (weekday value language)
				(equal? width 3) (if (equal? language "Deutsch")
					(nth short_weekdays (extract_date value "WEEKDAY")) (substr (weekday value language) 0 3))
				(equal? width 2) (format_date value "%d") (string (extract_date value "DAY")))
			(equal? head "M") (if (>= width 4) (months value language)
				(equal? width 3) (if (equal? language "Deutsch")
					(nth short_months (- (extract_date value "MONTH") 1)) (nth english_months (- (extract_date value "MONTH") 1)))
				(equal? width 2) (format_date value "%m") (string (extract_date value "MONTH")))
			(equal? head "y") (begin
				(define year (extract_date value "YEAR"))
				(define text (string (if (<= width 2) (mod year 100) year)))
				(concat (string_repeat "0" (max 0 (- width (strlen text)))) text))
			(equal? token "/") (if (equal? language "Deutsch") "." "/")
			(regexp_test token "^[hHmsftFzKg]$") (error "unsupported FORMAT clock or era pattern: " token)
			token)))) tsql_datename_month tsql_datename_weekday tsql_format_short_months tsql_format_short_weekdays tsql_calendar_month_names)))

(define tsql_format_calendar (close_procedure ((lambda (culture_profile fragment)
	(lambda (value pattern culture) (begin
		(define language (culture_profile culture))
		(if (nil? value) nil (nil? pattern) (error "unsupported FORMAT default calendar pattern") (begin
			(if (<= (strlen pattern) 1) (error "unsupported FORMAT standard calendar pattern: " pattern) true)
			/* Escapes and quotes take precedence over token runs. Percent selects
			a single custom token even where a lone letter means a standard format. */
			(define tokens (regexp_matches pattern "(?s)\\\\.|'(?:[^'\\\\]|\\\\.)*'|\"(?:[^\"\\\\]|\\\\.)*\"|%[dMy]|d+|M+|y+|."))
			(reduce tokens (lambda (text token)
				(if (nil? text) nil (begin
					(define token (if (and (equal? (strlen token) 2) (equal? (substr token 0 1) "%"))
						(substr token 1) token))
					(define part (fragment value token language))
					(if (nil? part) nil (concat text part))))) "")))))) tsql_format_culture tsql_format_calendar_token)))

(define tsql_compile_format (lambda (infos) (begin
	(if (or (equal? (count infos) 2) (equal? (count infos) 3)) true
		(error "FORMAT requires value, pattern and optional culture"))
	(define input (car infos))
	(define type (sql_info_type input))
	(define numeric (has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "DECIMAL" "NUMERIC" "REAL" "FLOAT" "MONEY" "SMALLMONEY") type))
	(define calendar (has? '("DATE" "DATETIME" "DATETIME2" "SMALLDATETIME") type))
	(if (or numeric calendar (has? '("TIME" "DATETIMEOFFSET") type)) true
		(tsql_raise_error 8116 (concat "Argument data type " type " is invalid for FORMAT") 1))
	(if (or numeric calendar) true (error "unsupported FORMAT temporal declaration: " type))
	(define pattern (sql_info_formula (tsql_convert_value_info (cadr infos) "NVARCHAR" '(4000))))
	(define culture (if (equal? (count infos) 3)
		(sql_info_formula (tsql_convert_value_info (nth infos 2) "NVARCHAR" '(4000)))
		(list 'if (list 'equal? (tsql_language_formula) "Deutsch") "de-DE" "en-US")))
	(define formula (if calendar (tsql_calendar_difference_formula input)
		(has? '("DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY") type)
		(list tsql_decimal_text (sql_info_formula input)
			(if (has? '("MONEY" "SMALLMONEY") type) 4
				(cadr (tsql_decl_dimensions type (sql_info_dimensions input)))))
		(list (if (has? '("FLOAT" "REAL") type) tsql_format_float_text tsql_format_integral_text) (sql_info_formula input))))
	(sql_info (list (if calendar tsql_format_calendar tsql_format_zero) formula pattern culture)
		(list "NVARCHAR" '(4000)) '("utf8mb4_general_ci" 4)))))
