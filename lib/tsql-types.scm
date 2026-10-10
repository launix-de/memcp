/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Declaration semantics belong to the frontend. Exact decimals are ordinary
canonical coefficient strings; dates are ordinary signed Unix 100ns integers.
Neither representation changes the meaning of ordinary Scheme arithmetic.
The common SQL binder chooses scales, casts and callbacks before execution. */
/* Scheme source numbers are Float values. Bind exact integral constants once
from canonical text, so ordinary integer/Unix arithmetic remains integer-only.
Quoted declaration data and match patterns keep their existing representation. */
(define tsql_in621355968000000000 (coefficient_to_integer (coefficient_encode "-621355968000000000") 0 true))
(define tsql_in68478048000000000 (coefficient_to_integer (coefficient_encode "-68478048000000000") 0 true))
(define tsql_in22089888000000000 (coefficient_to_integer (coefficient_encode "-22089888000000000") 0 true))
(define tsql_in53690 (coefficient_to_integer (coefficient_encode "-53690") 0 true))
(define tsql_in25567 (coefficient_to_integer (coefficient_encode "-25567") 0 true))
(define tsql_in840 (coefficient_to_integer (coefficient_encode "-840") 0 true))
(define tsql_in128 (coefficient_to_integer (coefficient_encode "-128") 0 true))
(define tsql_in38 (coefficient_to_integer (coefficient_encode "-38") 0 true))
(define tsql_in1 (coefficient_to_integer (coefficient_encode "-1") 0 true))
(define tsql_i0 (coefficient_to_integer (coefficient_encode "0") 0 true))
(define tsql_i1 (coefficient_to_integer (coefficient_encode "1") 0 true))
(define tsql_i2 (coefficient_to_integer (coefficient_encode "2") 0 true))
(define tsql_i3 (coefficient_to_integer (coefficient_encode "3") 0 true))
(define tsql_i4 (coefficient_to_integer (coefficient_encode "4") 0 true))
(define tsql_i5 (coefficient_to_integer (coefficient_encode "5") 0 true))
(define tsql_i6 (coefficient_to_integer (coefficient_encode "6") 0 true))
(define tsql_i7 (coefficient_to_integer (coefficient_encode "7") 0 true))
(define tsql_i8 (coefficient_to_integer (coefficient_encode "8") 0 true))
(define tsql_i9 (coefficient_to_integer (coefficient_encode "9") 0 true))
(define tsql_i10 (coefficient_to_integer (coefficient_encode "10") 0 true))
(define tsql_i12 (coefficient_to_integer (coefficient_encode "12") 0 true))
(define tsql_i13 (coefficient_to_integer (coefficient_encode "13") 0 true))
(define tsql_i16 (coefficient_to_integer (coefficient_encode "16") 0 true))
(define tsql_i17 (coefficient_to_integer (coefficient_encode "17") 0 true))
(define tsql_i18 (coefficient_to_integer (coefficient_encode "18") 0 true))
(define tsql_i19 (coefficient_to_integer (coefficient_encode "19") 0 true))
(define tsql_i23 (coefficient_to_integer (coefficient_encode "23") 0 true))
(define tsql_i24 (coefficient_to_integer (coefficient_encode "24") 0 true))
(define tsql_i28 (coefficient_to_integer (coefficient_encode "28") 0 true))
(define tsql_i30 (coefficient_to_integer (coefficient_encode "30") 0 true))
(define tsql_i32 (coefficient_to_integer (coefficient_encode "32") 0 true))
(define tsql_i38 (coefficient_to_integer (coefficient_encode "38") 0 true))
(define tsql_i50 (coefficient_to_integer (coefficient_encode "50") 0 true))
(define tsql_i59 (coefficient_to_integer (coefficient_encode "59") 0 true))
(define tsql_i60 (coefficient_to_integer (coefficient_encode "60") 0 true))
(define tsql_i100 (coefficient_to_integer (coefficient_encode "100") 0 true))
(define tsql_i128 (coefficient_to_integer (coefficient_encode "128") 0 true))
(define tsql_i150 (coefficient_to_integer (coefficient_encode "150") 0 true))
(define tsql_i300 (coefficient_to_integer (coefficient_encode "300") 0 true))
(define tsql_i840 (coefficient_to_integer (coefficient_encode "840") 0 true))
(define tsql_i1440 (coefficient_to_integer (coefficient_encode "1440") 0 true))
(define tsql_i2050 (coefficient_to_integer (coefficient_encode "2050") 0 true))
(define tsql_i2079 (coefficient_to_integer (coefficient_encode "2079") 0 true))
(define tsql_i8000 (coefficient_to_integer (coefficient_encode "8000") 0 true))
(define tsql_i9999 (coefficient_to_integer (coefficient_encode "9999") 0 true))
(define tsql_i10000 (coefficient_to_integer (coefficient_encode "10000") 0 true))
(define tsql_i55296 (coefficient_to_integer (coefficient_encode "55296") 0 true))
(define tsql_i57343 (coefficient_to_integer (coefficient_encode "57343") 0 true))
(define tsql_i65535 (coefficient_to_integer (coefficient_encode "65535") 0 true))
(define tsql_i86400 (coefficient_to_integer (coefficient_encode "86400") 0 true))
(define tsql_i719162 (coefficient_to_integer (coefficient_encode "719162") 0 true))
(define tsql_i1114111 (coefficient_to_integer (coefficient_encode "1114111") 0 true))
(define tsql_i2958463 (coefficient_to_integer (coefficient_encode "2958463") 0 true))
(define tsql_i3652058 (coefficient_to_integer (coefficient_encode "3652058") 0 true))
(define tsql_i5000000 (coefficient_to_integer (coefficient_encode "5000000") 0 true))
(define tsql_i10000000 (coefficient_to_integer (coefficient_encode "10000000") 0 true))
(define tsql_i25920000 (coefficient_to_integer (coefficient_encode "25920000") 0 true))
(define tsql_i299990000 (coefficient_to_integer (coefficient_encode "299990000") 0 true))
(define tsql_i600000000 (coefficient_to_integer (coefficient_encode "600000000") 0 true))
(define tsql_i864000000000 (coefficient_to_integer (coefficient_encode "864000000000") 0 true))
(define tsql_i2534023007999999999 (coefficient_to_integer (coefficient_encode "2534023007999999999") 0 true))

(define tsql_descriptor_integer (lambda (value) (if (nil? value) nil (coefficient_to_integer (integer_to_coefficient value) 0 true))))
(define tsql_power_ten (lambda (power) (coefficient_to_integer (coefficient_encode (concat "1" (string_repeat "0" power))) 0 true)))

/* The Scheme reader has no exponent literal syntax; bind this Float once. */
(define tsql_float_limit (simplify "1.7976931348623157e308"))
(define tsql_ticks_per_second tsql_i10000000)
(define tsql_ticks_per_day tsql_i864000000000)
(define tsql_unix_epoch_days tsql_i719162)
(define tsql_spec (lambda (spec) (if (string? spec) (list (toUpper spec)) spec)))
(define tsql_spec_type (lambda (spec) (car (tsql_spec spec))))
/* Dimensions are optional; generic list readers deliberately reject missing slots. */
(define tsql_spec_dimension (lambda (spec position)
	(if (> (count spec) position) (nth spec position) nil)))
(define tsql_cast_spec (lambda (spec) (begin
	(define type (car spec))
	(cons type (if (tsql_decimal_type? type) (list (tsql_spec_precision spec) (tsql_spec_scale spec))
		(if (has? '("DATETIME2" "DATETIMEOFFSET" "TIME") type) (list (tsql_spec_scale spec))
			(if (has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR" "BINARY" "VARBINARY") type) (list (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i30)) (cdr spec))))))))
(define tsql_exact_type? (lambda (type) (has? '("DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY") type)))
(define tsql_decimal_type? (lambda (type) (has? '("DECIMAL" "NUMERIC") type)))
(define tsql_money_type? (lambda (type) (has? '("MONEY" "SMALLMONEY") type)))
(define tsql_temporal_type? (lambda (type) (has? '("DATE" "DATETIME" "SMALLDATETIME" "DATETIME2" "DATETIMEOFFSET" "TIME") type)))
(define tsql_spec_precision (lambda (spec) (match (tsql_spec_type spec)
	"MONEY" tsql_i19 "SMALLMONEY" tsql_i10 "BIGINT" tsql_i19 "SMALLINT" tsql_i5 "TINYINT" tsql_i3 "BIT" tsql_i1
	"DECIMAL" (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i18) "NUMERIC" (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i18) _ tsql_i10)))
(define tsql_spec_scale (lambda (spec) (match (tsql_spec_type spec)
	"MONEY" tsql_i4 "SMALLMONEY" tsql_i4
	"DECIMAL" (coalesceNil (tsql_spec_dimension spec tsql_i2) tsql_i0) "NUMERIC" (coalesceNil (tsql_spec_dimension spec tsql_i2) tsql_i0)
	"DATETIME2" (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i7) "DATETIMEOFFSET" (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i7)
	"TIME" (coalesceNil (tsql_spec_dimension spec tsql_i1) tsql_i7) _ tsql_i0)))
(define tsql_integer_bounds (lambda (type) (match type
	"TINYINT" '("0" "255") "SMALLINT" '("-32768" "32767")
	"INT" '("-2147483648" "2147483647") "INTEGER" '("-2147483648" "2147483647")
	"SMALLMONEY" '("-2147483648" "2147483647")
	_ '("-9223372036854775808" "9223372036854775807"))))
(define tsql_check_coefficient (lambda (value minimum maximum)
	(if (or (< (coefficient_compare value minimum) tsql_i0) (> (coefficient_compare value maximum) tsql_i0))
		(error "numeric overflow") value)))
(define tsql_decimal_validate (lambda (value spec) (if (nil? value) nil (begin
	(define digits (string_repeat "9" (tsql_spec_precision spec)))
	(tsql_check_coefficient value (coefficient_encode (concat "-" digits)) (coefficient_encode digits))))))
(define tsql_integer_validate (lambda (value spec) (if (nil? value) nil (begin
	(define bounds (tsql_integer_bounds (car spec)))
	(tsql_check_coefficient (integer_to_coefficient value) (coefficient_encode (car bounds)) (coefficient_encode (cadr bounds)))
	value))))

/* Text parsing never sends a 38-digit coefficient through simplify/Float.
Only the exponent itself is an ordinary small integer. */
(define tsql_canonical_integer_text (lambda (text) (begin
	(define negative (regexp_test text "^-"))
	(define digits (regexp_replace (regexp_replace text "^[+-]" "") "^0+" ""))
	(if (equal? digits "") "0" (concat (if negative "-" "") digits)))))
(define tsql_parse_coefficient (lambda (value) (match (regexp_replace (concat value) "^\\s+|\\s+$" "")
	(regex "^([+-]?)([0-9]*)(?:\\.([0-9]*))?(?:[eE]([+-]?[0-9]+))?$" _ sign integral fraction exponent)
	(begin
		(define integral (coalesceNil integral ""))
		(define fraction (coalesceNil fraction ""))
		(if (> (+ (strlen integral) (strlen fraction)) tsql_i0) true (error "invalid numeric text"))
		(define scale (- (strlen fraction) (if (or (nil? exponent) (equal? exponent "")) tsql_i0 (simplify exponent))))
		(if (or (< scale tsql_in128) (> scale tsql_i128)) (error "numeric exponent overflow") true)
		(list (coefficient_encode (tsql_canonical_integer_text (concat sign integral fraction (if (< scale tsql_i0) (string_repeat "0" (- tsql_i0 scale)) "")))) (max tsql_i0 scale)))
	_ (error "invalid numeric text"))))
(define tsql_literal_spec (lambda (text) (begin
	(define parts (tsql_parse_coefficient text))
	(define digits (regexp_replace (coefficient_decode (car parts)) "^-|^0+" ""))
	(define precision (max tsql_i1 (strlen digits) (cadr parts)))
	(if (> precision tsql_i38) (error "numeric literal exceeds precision 38") true)
	(if (and (equal? (cadr parts) tsql_i0) (<= precision tsql_i10)
		(<= (coefficient_compare (car parts) (coefficient_encode "2147483647")) tsql_i0)) '("INT")
		(list "DECIMAL" precision (cadr parts))))))
(define tsql_numeric_literal (lambda (text)
	(if (regexp_test text "[eE]") (simplify text)
		(begin (define spec (tsql_literal_spec text))
			(if (equal? (car spec) "INT") (coefficient_to_integer (car (tsql_parse_coefficient text)) tsql_i0 true)
				(car (tsql_parse_coefficient text)))))))

/* Scalar truth coercion is not a boolean type test. Only actual booleans
map to 0/1; numeric text and floating values keep their numeric magnitude. */
(define tsql_value_coefficient (lambda (value source) (begin
	(define type (tsql_spec_type source))
	(if (tsql_decimal_type? type) (list value (tsql_spec_scale source))
		(if (tsql_money_type? type) (list (integer_to_coefficient value) tsql_i4)
			(if (int? value) (list (integer_to_coefficient value) tsql_i0)
				(if (or (expression_equal? value true) (expression_equal? value false)) (list (integer_to_coefficient (if value tsql_i1 tsql_i0)) tsql_i0)
					(tsql_parse_coefficient value))))))))
(define tsql_decimal_cast (lambda (value source target) (if (nil? value) nil (begin
	(define parts (tsql_value_coefficient value source))
	(tsql_decimal_validate (fixed_point_rescale (car parts) (cadr parts) (tsql_spec_scale target) false) target)))))
(define tsql_integer_cast (lambda (value source target) (if (nil? value) nil (begin
	(define parts (tsql_value_coefficient value source))
	(define integer (coefficient_to_integer (car parts) (cadr parts) (not (tsql_money_type? (tsql_spec_type source)))))
	(tsql_integer_validate integer target)))))
(define tsql_money_cast (lambda (value source target) (if (nil? value) nil (begin
	(define parts (tsql_value_coefficient value source))
	(define coefficient (fixed_point_rescale (car parts) (cadr parts) tsql_i4 false))
	(define bounds (tsql_integer_bounds (car target)))
	(tsql_check_coefficient coefficient (coefficient_encode (car bounds)) (coefficient_encode (cadr bounds)))
	(coefficient_to_integer coefficient tsql_i0 true)))))
(define tsql_numeric_text (lambda (value spec style) (begin
	(define parts (tsql_value_coefficient value spec))
	(define scale (if (tsql_money_type? (car spec)) (if (has? '(2 126) style) tsql_i4 tsql_i2) (cadr parts)))
	(define text (coefficient_format (fixed_point_rescale (car parts) (cadr parts) scale false) scale))
	(if (and (equal? style tsql_i1) (tsql_money_type? (car spec)))
		(tsql_comma_numeric_text text) text))))
(define tsql_comma_numeric_text (lambda (text) (begin
	(define changed (regexp_replace text "([0-9])([0-9]{3})([,.]|$)" "$1,$2$3"))
	(if (equal? changed text) text (tsql_comma_numeric_text changed)))))

/* Precision rules are applied once while binding, including the 38-digit cap.
Native fixed-point primitives receive only scales and a rounding mode. */
(define tsql_arithmetic_spec (lambda (op left right) (begin
	(define lt (tsql_spec_type left)) (define rt (tsql_spec_type right))
	(define p1 (tsql_spec_precision left)) (define p2 (tsql_spec_precision right))
	(define s1 (tsql_spec_scale left)) (define s2 (tsql_spec_scale right))
	(if (or (has? '("FLOAT" "REAL" "DOUBLE") lt) (has? '("FLOAT" "REAL" "DOUBLE") rt)) '("FLOAT" 53 0)
		(if (or (tsql_exact_type? lt) (tsql_exact_type? rt))
			(if (and (not (tsql_decimal_type? lt)) (not (tsql_decimal_type? rt)))
				(list (if (or (equal? lt "MONEY") (equal? rt "MONEY")) "MONEY" "SMALLMONEY")
					(if (or (equal? lt "MONEY") (equal? rt "MONEY")) tsql_i19 tsql_i10) tsql_i4)
				(begin
					(define shape (match op
						"*" (list (+ p1 p2 tsql_i1) (+ s1 s2))
						"/" (begin (define scale (max tsql_i6 (+ s1 p2 tsql_i1))) (list (+ (- p1 s1) s2 scale) scale))
						"%" (list (+ (min (- p1 s1) (- p2 s2)) (max s1 s2)) (max s1 s2))
						_ (list (+ (max (- p1 s1) (- p2 s2)) (max s1 s2) tsql_i1) (max s1 s2))))
					(define p (car shape)) (define s (cadr shape)) (define integral (- p s))
					(list "DECIMAL" (min tsql_i38 p)
						(if (<= p tsql_i38) s (if (has? '("+" "-" "%") op) (max tsql_i0 (min s (- tsql_i38 integral)))
							(if (< integral tsql_i32) (min s (- tsql_i38 integral)) (if (> s tsql_i6) tsql_i6 s)))))))
			(if (or (equal? lt "BIGINT") (equal? rt "BIGINT")) '("BIGINT" 19 0) '("INT" 10 0)))))))
(define tsql_arithmetic_bound (lambda (op left right left_spec right_spec result_spec)
	(if (or (nil? left) (nil? right)) nil (begin
		(define lt (car left_spec)) (define rt (car right_spec)) (define target (car result_spec))
		(if (and (equal? op "+") (sql_text_type? lt) (sql_text_type? rt)) (concat left right)
			(if (equal? target "FLOAT")
				(begin
					(define a (tsql_cast_bound left left_spec '("FLOAT"))) (define b (tsql_cast_bound right right_spec '("FLOAT")))
					(tsql_cast_bound (match op "+" (+ a b) "-" (- a b) "*" (* a b) "/" (if (equal? b tsql_i0) (error "division by zero") (/ a b))
						"%" (if (equal? b tsql_i0) (error "division by zero") (- a (* b (if (< (/ a b) tsql_i0) (ceil (/ a b)) (floor (/ a b)))))) _ (error "unknown numeric operator")) '("FLOAT") result_spec))
				(begin
					(define a (tsql_value_coefficient left left_spec)) (define b (tsql_value_coefficient right right_spec))
					(define value (fixed_point_binary op (car a) (car b) (cadr a) (cadr b) (tsql_spec_scale result_spec)
						(or (not (tsql_exact_type? target)) (and (tsql_money_type? target) (has? '("*" "/") op)))))
					(if (tsql_decimal_type? target) (tsql_decimal_validate value result_spec)
						(tsql_integer_validate (coefficient_to_integer value tsql_i0 true) result_spec)))))))))

(define tsql_pad_digits (lambda (value width) (begin (define text (concat value))
	(concat (string_repeat "0" (max tsql_i0 (- width (strlen text)))) text))))
(define tsql_floor_divide (lambda (value divisor)
	(begin (define q (intdiv value divisor)) (if (< (- value (* q divisor)) tsql_i0) (- q tsql_i1) q))))
(define tsql_positive_mod (lambda (value divisor) (- value (* divisor (tsql_floor_divide value divisor)))))
(define tsql_round_ticks (lambda (value step)
	(* step (tsql_floor_divide (+ value (intdiv step tsql_i2)) step))))
(define tsql_offset_minutes (lambda (value)
	(if (equal? value "Z") tsql_i0 (match value
		(regex "^([+-])([0-9]{2}):([0-9]{2})$" _ sign hours minutes) (begin
			(define offset (+ (* tsql_i60 (coefficient_to_integer (coefficient_encode (tsql_canonical_integer_text hours)) 0 true)) (coefficient_to_integer (coefficient_encode (tsql_canonical_integer_text minutes)) 0 true)))
			(if (or (> (coefficient_to_integer (coefficient_encode (tsql_canonical_integer_text minutes)) 0 true) tsql_i59) (> offset tsql_i840)) (error "invalid temporal offset") true)
			(if (equal? sign "-") (- tsql_i0 offset) offset)) _ (error "invalid temporal offset")))))
(define tsql_offset_value (lambda (ticks offset)
	(concat (integer_order_key ticks) (if (< offset tsql_i0) "-" "+") (tsql_pad_digits (sql_abs offset) tsql_i4))))
(define tsql_offset_ticks (lambda (value) (integer_from_order_key (substr value tsql_i0 tsql_i16))))
(define tsql_offset_value_minutes (lambda (value) (coefficient_to_integer (coefficient_encode (tsql_canonical_integer_text (substr value tsql_i16))) 0 true)))
(define tsql_temporal_validate (lambda (ticks spec) (if (nil? ticks) nil (begin
	(define type (car spec))
	(define value (if (equal? type "DATETIMEOFFSET") (tsql_offset_ticks ticks) ticks))
	(define minimum (if (equal? type "DATETIME") tsql_in68478048000000000
		(if (equal? type "SMALLDATETIME") tsql_in22089888000000000 tsql_in621355968000000000)))
	(define maximum (if (equal? type "SMALLDATETIME") (unix_from_parts tsql_i2079 tsql_i6 tsql_i6 tsql_i23 tsql_i59 tsql_i0 tsql_i0 tsql_ticks_per_second) tsql_i2534023007999999999))
	(if (equal? type "DATETIMEOFFSET") (begin
		(define offset (tsql_offset_value_minutes ticks))
		(define local (+ value (* offset tsql_i600000000)))
		(if (and (>= offset tsql_in840) (<= offset tsql_i840) (>= local minimum) (<= local maximum)) true (error "offset datetime out of range"))) true)
	(if (equal? type "TIME")
		(if (and (>= value tsql_i0) (< value tsql_ticks_per_day)) ticks (error "TIME out of range"))
		(if (and (>= value minimum) (<= value maximum)) ticks (error "temporal value out of range")))))))
(define tsql_temporal_round (lambda (ticks spec) (begin
	(define type (car spec))
	(define rounded (match type
		"DATE" (* tsql_ticks_per_day (tsql_floor_divide ticks tsql_ticks_per_day))
		"SMALLDATETIME" (* tsql_i600000000 (tsql_floor_divide (+ ticks tsql_i299990000) tsql_i600000000))
		"DATETIME" (begin
			(define seconds (tsql_floor_divide ticks tsql_ticks_per_second))
			(define units (intdiv (+ (* (tsql_positive_mod ticks tsql_ticks_per_second) tsql_i300) tsql_i5000000) tsql_i10000000))
			(+ (* seconds tsql_ticks_per_second) (intdiv (+ (* units tsql_i10000000) tsql_i150) tsql_i300)))
		_ (tsql_round_ticks ticks (tsql_power_ten (- tsql_i7 (tsql_spec_scale spec))))))
	(tsql_temporal_validate (if (equal? type "TIME") (tsql_positive_mod rounded tsql_ticks_per_day) rounded) spec))))
(define tsql_parse_temporal (lambda (text) (begin
	(define normalized (regexp_replace text "T" " "))
	(define offset (match normalized (regex "^(.*?)(Z|[+-][0-9]{2}:[0-9]{2})$" _ _base zone) (tsql_offset_minutes zone) _ tsql_i0))
	(define local (regexp_replace normalized "(Z|[+-][0-9]{2}:[0-9]{2})$" ""))
	(define value (if (regexp_test local "^[0-9]{1,2}:[0-9]{2}") (concat "1900-01-01 " local)
		(if (regexp_test local "^[0-9]{8}$") (concat (substr local tsql_i0 tsql_i4) "-" (substr local tsql_i4 tsql_i2) "-" (substr local tsql_i6 tsql_i2)) local)))
	(define layout (if (regexp_test value " ") "2006-01-02 15:04:05.999999999" "2006-01-02"))
	(define fractional (match value (regex "\\.([0-9]+)$" _ digits) digits _ ""))
	(define whole (regexp_replace value "\\.[0-9]+$" ""))
	(define remainder (if (equal? fractional "") tsql_i0
		(coefficient_to_integer (fixed_point_rescale (coefficient_encode (tsql_canonical_integer_text fractional)) (strlen fractional) tsql_i7 false) tsql_i0 true)))
	(list (+ (unix_parse whole layout tsql_ticks_per_second offset) remainder) offset))))
(define tsql_temporal_cast (lambda (value source target) (if (nil? value) nil (begin
	(define source_type (car source)) (define type (car target))
	(define parts (if (equal? source_type "DATETIMEOFFSET") (list (tsql_offset_ticks value) (tsql_offset_value_minutes value))
		(if (tsql_temporal_type? source_type) (list (if (and (equal? source_type "TIME") (not (equal? type "TIME"))) (+ tsql_in22089888000000000 value) value) tsql_i0)
			(if (string? value) (tsql_parse_temporal value)
				(if (equal? type "DATETIME") (begin
					(define number (tsql_value_coefficient value source))
					(define day_ticks (fixed_point_binary "*" (car number) (integer_to_coefficient tsql_ticks_per_day) (cadr number) tsql_i0 tsql_i0 false))
					(list (coefficient_to_integer (fixed_point_binary "+" day_ticks (integer_to_coefficient tsql_in22089888000000000) tsql_i0 tsql_i0 tsql_i0 true) tsql_i0 true) tsql_i0))
					(error "numeric conversion requires legacy DATETIME"))))))
	(define ticks (car parts)) (define offset (cadr parts))
	(if (equal? type "DATETIMEOFFSET")
		(begin
			(define rounded (tsql_round_ticks ticks (tsql_power_ten (- tsql_i7 (tsql_spec_scale target)))))
			(tsql_temporal_validate (tsql_offset_value rounded offset) target))
		(tsql_temporal_round (+ ticks (if (equal? source_type "DATETIMEOFFSET") (* offset tsql_i600000000) tsql_i0)) target))))))
(define tsql_temporal_text (lambda (value spec style) (begin
	(define type (car spec)) (define offset (if (equal? type "DATETIMEOFFSET") (tsql_offset_value_minutes value) tsql_i0))
	(define ticks (if (equal? type "DATETIMEOFFSET") (tsql_offset_ticks value) value))
	(define layout (match style
		112 "20060102" 23 "2006-01-02" 101 "01/02/2006" 1 "01/02/06" 104 "02.01.2006" 4 "02.01.06"
		103 "02/01/2006" 3 "02/01/06" 105 "02-01-2006" 5 "02-01-06"
		108 "15:04:05" 114 "15:04:05.000" 120 "2006-01-02 15:04:05" 121 "2006-01-02 15:04:05.000"
		126 "2006-01-02T15:04:05" 127 "2006-01-02T15:04:05"
		0 (match type "DATE" "2006-01-02" "TIME" "15:04:05" _ "2006-01-02 15:04:05")
		_ (error "unsupported temporal conversion style")))
	(define display_ticks (if (equal? type "DATETIME") (tsql_round_ticks ticks tsql_i10000) ticks))
	(define base (unix_format display_ticks tsql_ticks_per_second layout offset))
	(define scale (if (has? '("DATETIME2" "DATETIMEOFFSET" "TIME") type) (tsql_spec_scale spec) (if (equal? type "DATETIME") tsql_i3 tsql_i0)))
	(define fraction (if (and (> scale tsql_i0) (or (equal? style tsql_i0) (has? '(126 127) style)))
		(concat "." (substr (tsql_pad_digits (tsql_positive_mod display_ticks tsql_ticks_per_second) tsql_i7) tsql_i0 scale)) ""))
	(concat base fraction (if (equal? type "DATETIMEOFFSET") (concat (if (< offset tsql_i0) "-" "+")
		(tsql_pad_digits (intdiv (sql_abs offset) tsql_i60) tsql_i2) ":" (tsql_pad_digits (tsql_positive_mod (sql_abs offset) tsql_i60) tsql_i2)) "")))))

(define tsql_character_cast (lambda (text target numeric)
	(begin
		(define type (car target)) (define width (coalesceNil (tsql_spec_dimension target tsql_i1) tsql_i30))
		(define unicode (has? '("NVARCHAR" "NCHAR" "NTEXT") type))
		(define size (if unicode (utf16_len text) (codepoint_length text)))
		(if (and numeric (> width tsql_i0) (> size width)) (error "numeric character conversion too narrow") true)
		(define cut (if (or (equal? width tsql_in1) (<= size width)) text (if unicode (utf16_prefix text width) (codepoint_prefix text width))))
		(if (has? '("CHAR" "NCHAR") type) (concat cut (string_repeat " " (max tsql_i0 (- width (if unicode (utf16_len cut) (codepoint_length cut)))))) cut))))
(define tsql_cast_bound (lambda (value source target) (if (nil? value) nil (begin
	(define type (car target)) (define source_type (car source))
	(if (tsql_decimal_type? type) (tsql_decimal_cast value source target)
		(if (tsql_money_type? type) (tsql_money_cast value source target)
			(if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) (tsql_integer_cast value source target)
				(if (tsql_temporal_type? type) (tsql_temporal_cast value source target)
					(if (has? '("FLOAT" "REAL" "DOUBLE") type)
						(begin
							(define number (if (tsql_exact_type? source_type) (begin (define parts (tsql_value_coefficient value source)) (simplify (coefficient_format (car parts) (cadr parts))))
								(if (or (expression_equal? value true) (expression_equal? value false)) (if value tsql_i1 tsql_i0) (if (number? value) value (simplify value)))))
							(if (and (number? number) (<= (sql_abs number) tsql_float_limit)) true (error "invalid or overflowing floating point value"))
							(if (or (equal? type "REAL") (and (equal? type "FLOAT") (not (nil? (tsql_spec_dimension target tsql_i1))) (<= (tsql_spec_dimension target tsql_i1) tsql_i24))) (float32_round number) number))
						(if (equal? type "BIT") (if (tsql_exact_type? source_type) (not (equal? (coefficient_compare (car (tsql_value_coefficient value source)) (coefficient_encode "0")) tsql_i0))
							(if (or (expression_equal? value true) (expression_equal? value false)) value
								(if (string? value) (if (equal?? value "true") true (if (equal?? value "false") false
									(begin (define number (simplify value)) (if (number? number) (not (equal? number tsql_i0)) (error "invalid BIT conversion")))))
									(not (equal? value tsql_i0)))))
							(if (has? '("BINARY" "VARBINARY" "ROWVERSION") type) (begin
								(define width (coalesceNil (tsql_spec_dimension target tsql_i1) tsql_i30))
								(if (string? value) true (error "binary conversion requires raw byte values"))
								(if (and (equal? type "ROWVERSION") (not (equal? (strlen value) tsql_i8))) (error "ROWVERSION requires exactly eight bytes") true)
								(define cut (if (or (equal? width tsql_in1) (<= (strlen value) width)) value (substr value tsql_i0 width)))
								(if (equal? type "BINARY") (concat cut (string_repeat (codepoint_string tsql_i0) (max tsql_i0 (- width (strlen cut))))) cut))
								(if (has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR" "TEXT" "NTEXT") type)
									(tsql_character_cast (if (tsql_temporal_type? source_type) (tsql_temporal_text value source tsql_i0)
										(if (tsql_exact_type? source_type) (tsql_numeric_text value source tsql_i0) (concat value))) target (sql_numeric_type? source_type))
									(error (concat "unsupported cast target " type))))))))))))))
(define tsql_cast_value (lambda (value target)
	(tsql_cast_bound value (list (sql_runtime_value_type value)) target)))
(define tsql_binary_literal (lambda (text) (hex2bin (if (equal? (tsql_positive_mod (strlen text) tsql_i2) tsql_i0) text (concat "0" text)))))
(define tsql_nchar (lambda (value) (if (or (nil? value) (< value tsql_i0) (> value tsql_i1114111) (and (>= value tsql_i55296) (<= value tsql_i57343))) nil (codepoint_string value))))
/* A statement binds one immutable clock carrier. The engine receives this
opaque value only as a declared calculator argument. */
(define tsql_statement_context (lambda () (begin
	(define ticks (unix_clock tsql_ticks_per_second))
	(list "utc_ticks" ticks "zone_offset_seconds" (unix_zone_offset ticks tsql_ticks_per_second (system_time_zone))))))
(define tsql_clock_at (lambda (context kind) (begin
	(define context (if (nil? context) (tsql_statement_context) context))
	(define wall (+ (context "utc_ticks") (* (context "zone_offset_seconds") tsql_ticks_per_second)))
	(tsql_temporal_round wall (if (equal? kind "DATETIME2") '("DATETIME2" 7) (list kind))))))
(define tsql_clock_value (lambda (kind) (tsql_clock_at nil kind)))
(define tsql_uses_statement_context (lambda (formula) (match formula
	((symbol quote) _value) false
	(symbol tsql_statement_values) true
	(cons head tail) (or (tsql_uses_statement_context head) (reduce tail (lambda (used part) (or used (tsql_uses_statement_context part))) false))
	_ false)))
(define tsql_statement_formula (lambda (formula)
	(if (tsql_uses_statement_context formula)
		(list (list 'lambda (list 'tsql_statement_values) formula) (list 'tsql_statement_context)) formula)))
(define tsql_clock_binding (lambda (formula binding) (match formula
	((symbol quote) _value) formula
	(symbol tsql_statement_values) binding
	(cons head tail) (cons (tsql_clock_binding head binding) (map tail (lambda (part) (tsql_clock_binding part binding))))
	_ formula)))

(define tsql_temporal_input_style (lambda (text style) (begin
	(define layout (get_assoc '(0 "Jan _2 2006 3:04PM" 100 "Jan _2 2006 3:04PM"
		1 "01/02/06" 101 "01/02/2006" 3 "02/01/06" 103 "02/01/2006"
		4 "02.01.06" 104 "02.01.2006" 5 "02-01-06" 105 "02-01-2006"
		112 "20060102" 23 "2006-01-02" 120 "2006-01-02 15:04:05"
		121 "2006-01-02 15:04:05.999999999" 126 "2006-01-02T15:04:05.999999999"
		127 "2006-01-02T15:04:05.999999999Z07:00") style))
	(if (nil? layout) (error "unsupported temporal conversion style") true)
	(define normalized (if (has? '(4 104) style) (regexp_replace text "[-/]" ".") text))
	(define value (if (and (not (has? '(0 100 120 121 126 127) style)) (regexp_test normalized " "))
		(unix_parse (regexp_replace normalized "(?i)\\s*(AM|PM)$" "") (concat layout " 15:04:05") tsql_ticks_per_second tsql_i0)
		(unix_parse normalized layout tsql_ticks_per_second tsql_i0)))
	(define parts (unix_parts value tsql_ticks_per_second))
	(if (and (has? '(1 3 4 5) style) (>= (car parts) tsql_i2050))
		(unix_from_parts (- (car parts) tsql_i100) (nth parts tsql_i1) (nth parts tsql_i2) (nth parts tsql_i3) (nth parts tsql_i4) (nth parts tsql_i5) (nth parts tsql_i6) tsql_ticks_per_second) value))))
(define tsql_convert_bound (lambda (value source target style) (if (or (nil? value) (nil? style)) nil
	(if (and (tsql_temporal_type? (car target)) (string? value) (not (tsql_temporal_type? (car source))) (not (equal? style tsql_in1)))
		(tsql_temporal_cast (tsql_temporal_input_style value style) '("DATETIME2" 7) target)
		(if (has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR" "TEXT" "NTEXT") (car target))
			(tsql_character_cast (if (tsql_temporal_type? (car source)) (tsql_temporal_text value source (if (equal? style tsql_in1) tsql_i0 style))
				(if (tsql_exact_type? (car source)) (tsql_numeric_text value source (if (equal? style tsql_in1) tsql_i0 style)) (concat value))) target (sql_numeric_type? (car source)))
			(tsql_cast_bound value source target))))))
(define tsql_convert_value (lambda (value target style source_type)
	(tsql_convert_bound value (if source_type (tsql_spec source_type) (list (sql_runtime_value_type value))) target style)))
(define tsql_unit (lambda (unit) (coalesceNil (get_assoc '("yy" "year" "yyyy" "year" "qq" "quarter" "q" "quarter"
	"mm" "month" "m" "month" "dy" "dayofyear" "y" "dayofyear" "dd" "day" "d" "day"
	"wk" "week" "ww" "week" "dw" "weekday" "w" "weekday" "hh" "hour"
	"mi" "minute" "n" "minute" "ss" "second" "s" "second" "ms" "millisecond"
	"mcs" "microsecond" "ns" "nanosecond" "isowk" "iso_week" "isoww" "iso_week") (toLower unit)) (toLower unit))))
(define tsql_unit_ticks (lambda (unit) (coefficient_to_integer (integer_to_coefficient (get_assoc '("week" 6048000000000 "day" 864000000000 "dayofyear" 864000000000
	"weekday" 864000000000 "hour" 36000000000 "minute" 600000000 "second" 10000000 "millisecond" 10000 "microsecond" 10 "nanosecond" 1) unit)) 0 true)))
(define tsql_temporal_operand (lambda (value spec)
	(if (tsql_temporal_type? (car spec)) value
		(tsql_temporal_cast value spec (if (has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR") (car spec)) '("DATETIME2" 7) '("DATETIME"))))))
(define tsql_calendar_ticks (lambda (value spec)
	(if (equal? (car spec) "DATETIMEOFFSET") (+ (tsql_offset_ticks value) (* (tsql_offset_value_minutes value) tsql_i600000000))
		(if (equal? (car spec) "TIME") (+ value tsql_in22089888000000000) value))))
(define tsql_datepart_bound (lambda (unit value spec allow_missing) (if (nil? value) nil (begin
	(define unit (tsql_unit unit))
	(if (and (equal? (car spec) "TIME") (has? '("year" "quarter" "month" "day" "dayofyear" "week" "weekday" "iso_week") unit) (not allow_missing))
		(error "calendar DATEPART is unsupported for a declared TIME") true)
	(define ticks (tsql_calendar_ticks (tsql_temporal_operand value spec) spec))
	(define parts (unix_parts ticks tsql_ticks_per_second))
	(match unit "year" (car parts) "quarter" (+ (intdiv (- (nth parts tsql_i1) tsql_i1) tsql_i3) tsql_i1) "month" (nth parts tsql_i1) "day" (nth parts tsql_i2)
		"hour" (nth parts tsql_i3) "minute" (nth parts tsql_i4) "second" (nth parts tsql_i5) "millisecond" (intdiv (nth parts tsql_i6) tsql_i10000)
		"microsecond" (intdiv (nth parts tsql_i6) tsql_i10) "nanosecond" (* (nth parts tsql_i6) tsql_i100)
		"weekday" (+ (nth parts tsql_i7) tsql_i1) "dayofyear" (nth parts tsql_i8) "iso_week" (nth parts tsql_i10)
		"week" (+ tsql_i1 (intdiv (+ (- (nth parts tsql_i8) tsql_i1) (tsql_positive_mod (- (nth parts tsql_i7) (- (nth parts tsql_i8) tsql_i1)) tsql_i7)) tsql_i7))
		"tzoffset" (if (equal? (car spec) "DATETIMEOFFSET") (tsql_offset_value_minutes value) tsql_i0)
		_ (error "unsupported temporal unit"))))))
(define tsql_datepart (lambda (unit value allow_missing)
	(tsql_datepart_bound unit value (list (sql_runtime_value_type value)) (coalesceNil allow_missing false))))
(define tsql_month_shift (lambda (ticks months) (begin
	(define parts (unix_parts ticks tsql_ticks_per_second))
	(define absolute (+ (* (car parts) tsql_i12) (- (nth parts tsql_i1) tsql_i1) months))
	(define year (tsql_floor_divide absolute tsql_i12)) (define month (+ tsql_i1 (tsql_positive_mod absolute tsql_i12)))
	(if (or (< year tsql_i1) (> year tsql_i9999)) (error "calendar arithmetic out of range") true)
	(define nextmonth (+ (* year tsql_i12) month))
	(define last (unix_parts (- (unix_from_parts (tsql_floor_divide nextmonth tsql_i12) (+ tsql_i1 (tsql_positive_mod nextmonth tsql_i12)) tsql_i1 tsql_i0 tsql_i0 tsql_i0 tsql_i0 tsql_ticks_per_second) tsql_ticks_per_day) tsql_ticks_per_second))
	(unix_from_parts year month (min (nth parts tsql_i2) (nth last tsql_i2)) (nth parts tsql_i3) (nth parts tsql_i4) (nth parts tsql_i5) (nth parts tsql_i6) tsql_ticks_per_second))))
(define tsql_dateadd_bound (lambda (unit amount value spec) (if (or (nil? amount) (nil? value)) nil (begin
	(define unit (tsql_unit unit)) (define type (car spec))
	(define amount (tsql_integer_cast amount (list (sql_runtime_value_type amount)) '("INT")))
	(if (and (equal? type "DATE") (has? '("hour" "minute" "second" "millisecond" "microsecond" "nanosecond") unit)) (error "DATEADD clock unit is unsupported for DATE") true)
	(if (and (equal? type "TIME") (has? '("year" "quarter" "month" "week" "day" "dayofyear" "weekday") unit)) (error "DATEADD calendar unit is unsupported for TIME") true)
	(define native (if (tsql_temporal_type? type) value (tsql_temporal_cast value spec '("DATETIME"))))
	(define ticks (tsql_calendar_ticks native spec))
	(define offset (if (equal? type "DATETIMEOFFSET") (tsql_offset_value_minutes value) tsql_i0))
	(define moved (if (has? '("year" "quarter" "month") unit) (tsql_month_shift ticks (* amount (match unit "year" tsql_i12 "quarter" tsql_i3 _ tsql_i1)))
		(begin
			(define step (tsql_unit_ticks unit)) (if (nil? step) (error "unsupported DATEADD unit") true)
			(define adjusted (if (equal? unit "nanosecond") (if (< amount tsql_i0) (- tsql_i0 (intdiv (+ (- tsql_i0 amount) tsql_i50) tsql_i100)) (intdiv (+ amount tsql_i50) tsql_i100)) amount))
			(define delta (fixed_point_binary "*" (integer_to_coefficient adjusted) (integer_to_coefficient step) tsql_i0 tsql_i0 tsql_i0 true))
			(coefficient_to_integer (fixed_point_binary "+" (integer_to_coefficient ticks) delta tsql_i0 tsql_i0 tsql_i0 true) tsql_i0 true))))
	(if (equal? type "DATETIMEOFFSET") (tsql_temporal_validate (tsql_offset_value (- moved (* offset tsql_i600000000)) offset) spec)
		(tsql_temporal_round moved (if (tsql_temporal_type? type) spec '("DATETIME"))))))))
(define tsql_dateadd (lambda (unit amount value type)
	(tsql_dateadd_bound unit amount value (if type (tsql_spec type) (list (sql_runtime_value_type value))))))
(define tsql_temporal_boundary (lambda (unit value spec) (begin
	(define ticks (tsql_calendar_ticks (tsql_temporal_operand value spec) spec))
	(define parts (unix_parts ticks tsql_ticks_per_second))
	(match unit "year" (car parts) "quarter" (+ (* (car parts) tsql_i4) (intdiv (- (nth parts tsql_i1) tsql_i1) tsql_i3)) "month" (+ (* (car parts) tsql_i12) (- (nth parts tsql_i1) tsql_i1))
		"week" (tsql_floor_divide (+ (tsql_floor_divide ticks tsql_ticks_per_day) tsql_i4) tsql_i7)
		_ (begin (define step (tsql_unit_ticks unit)) (if step (tsql_floor_divide ticks step) (error "unsupported DATEDIFF unit")))))))
(define tsql_datediff_bound (lambda (unit start end start_spec end_spec) (if (or (nil? start) (nil? end)) nil (begin
	(define unit (tsql_unit unit))
	(define value (fixed_point_binary "-" (integer_to_coefficient (tsql_temporal_boundary unit end end_spec))
		(integer_to_coefficient (tsql_temporal_boundary unit start start_spec)) tsql_i0 tsql_i0 tsql_i0 true))
	(define scaled (if (equal? unit "nanosecond") (fixed_point_binary "*" value (integer_to_coefficient tsql_i100) tsql_i0 tsql_i0 tsql_i0 true) value))
	(tsql_integer_validate (coefficient_to_integer scaled tsql_i0 true) '("INT"))))))
(define tsql_datediff (lambda (unit start end)
	(tsql_datediff_bound unit start end (list (sql_runtime_value_type start)) (list (sql_runtime_value_type end)))))

(define tsql_sum_spec (lambda (spec) (match (car spec)
	"DECIMAL" (list "DECIMAL" tsql_i38 (tsql_spec_scale spec)) "NUMERIC" (list "DECIMAL" tsql_i38 (tsql_spec_scale spec))
	"SMALLMONEY" '("MONEY") "SMALLINT" '("INT") "TINYINT" '("INT") "REAL" '("FLOAT") _ spec)))
(define tsql_merge_specs (lambda (left right) (begin
	(define lt (car left)) (define rt (car right))
	(if (equal? lt "NULL") right (if (equal? rt "NULL") left
		(if (equal? left right) left
			(if (or (equal? lt "any") (equal? rt "any")) '("any")
				(if (or (has? '("FLOAT" "REAL" "DOUBLE") lt) (has? '("FLOAT" "REAL" "DOUBLE") rt)) '("FLOAT")
					(if (or (tsql_decimal_type? lt) (tsql_decimal_type? rt))
						(begin (define scale (max (tsql_spec_scale left) (tsql_spec_scale right)))
							(define integral (max (- (tsql_spec_precision left) (tsql_spec_scale left)) (- (tsql_spec_precision right) (tsql_spec_scale right))))
							(list "DECIMAL" (min tsql_i38 (+ integral scale)) (max tsql_i0 (min scale (- tsql_i38 integral)))))
						(if (or (tsql_money_type? lt) (tsql_money_type? rt)) (if (or (equal? lt "MONEY") (equal? rt "MONEY")) '("MONEY") '("SMALLMONEY"))
							(if (or (sql_numeric_type? lt) (sql_numeric_type? rt)) (if (or (equal? lt "BIGINT") (equal? rt "BIGINT")) '("BIGINT") '("INT"))
								(if (and (sql_text_type? lt) (sql_text_type? rt))
									(list (if (or (has? '("NVARCHAR" "NCHAR" "NTEXT") lt) (has? '("NVARCHAR" "NCHAR" "NTEXT") rt)) "NVARCHAR" "VARCHAR")
										(if (or (equal? (cadr left) tsql_in1) (equal? (cadr right) tsql_in1)) tsql_in1 (max (coalesceNil (cadr left) tsql_i0) (coalesceNil (cadr right) tsql_i0))))
									(if (equal? lt rt) left '("any"))))))))))))))
(define tsql_coerce_info (lambda (info target)
	(if (or (equal? (sql_info_type info) "NULL") (equal? (car target) "any") (equal? (sql_info_spec info) target)) (sql_info_formula info)
		(list 'tsql_cast_bound (sql_info_formula info) (list 'quote (sql_info_spec info)) (list 'quote target)))))
(define tsql_union_specs (lambda (query) (begin
	(define branches (union_branches query))
	(define descriptions (map branches (lambda (branch) (tsql_query_descriptions branch nil))))
	(if (empty_list? descriptions) '() (begin
		(define width (count (car descriptions)))
		(if (reduce descriptions (lambda (valid branch) (and valid (equal? (count branch) width))) true) true (error "UNION column counts do not match"))
		(map (produceN width) (lambda (ordinal)
			(reduce descriptions (lambda (spec branch) (tsql_merge_specs spec (tsql_description_spec (nth branch ordinal)))) '("NULL")))))))))
(define tsql_coerce_query_columns (lambda (query specs) (begin
	(define query (normalize_query_ast query))
	(if (query_block? query)
		(begin
			(define fields (expand_query_block_fields (qb_sources query) (qb_fields query)))
			(define casted (merge (mapIndex (extract_assoc fields (lambda (name expression) (list name expression))) (lambda (ordinal field)
				(list (car field) (tsql_coerce_info (sql_expr_info (qb_sources query) (cadr field) true) (nth specs ordinal)))))))
			(make_query_block (qb_schema query) (qb_sources query) casted (qb_where query) (qb_group query) (qb_having query)
				(qb_order query) (qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query)))
		(if (union_block? query) (make_union_block (union_mode query) (map (union_branches query) (lambda (branch) (tsql_coerce_query_columns branch specs)))
			(union_order query) (union_limit query) (union_offset query) (union_facts query)) (error "unsupported UNION branch"))))))
/* Equality keys are selected at the frontend boundary. Offset payloads retain
an original offset for transport; grouping uses their UTC integer projection. */
(define tsql_offset_key (lambda (value) (if (nil? value) nil (tsql_offset_ticks value))))
(define tsql_key_formula (lambda (sources expression bindings) (begin
	(define info (sql_expr_info sources expression true bindings))
	(if (equal? (sql_info_type info) "DATETIMEOFFSET") (list 'tsql_offset_key (sql_info_formula info)) (sql_info_formula info)))))
(define tsql_group_representative (lambda (previous value) (coalesceNil previous value)))
(define tsql_group_projection (lambda (expression bindings) (begin
	(define binding (find bindings (lambda (item) (equal? (car item) expression)) nil))
	(if binding (cadr binding) (match expression
		((symbol quote) _value) expression
		(cons (symbol aggregate) _args) expression
		(cons (symbol count_distinct) _args) expression
		(cons head tail) (cons head (map tail (lambda (part) (tsql_group_projection part bindings))))
		_ expression)))))
(define tsql_query_key_bindings (lambda (sources keys)
	(filter (map keys (lambda (expression) (begin
		(define info (sql_expr_info sources expression true)) (define spec (sql_info_spec info))
		(if (equal? (car spec) "DATETIMEOFFSET")
			(list (sql_info_formula info) (list 'tsql_cast_bound
				(list 'aggregate (sql_info_formula info) 'tsql_group_representative nil) (list 'quote spec) (list 'quote spec))) nil))))
		(lambda (binding) (not (nil? binding))))))
(define tsql_bind_query_block (lambda (query) (begin
	(define sources (map (qb_sources query) (lambda (source)
		(if (or (query_block? (source_relation source)) (union_block? (source_relation source)))
			(source_with_relation source (tsql_bind_query_types (source_relation source))) source))))
	(define bindings (tsql_query_key_bindings sources (coalesceNil (qb_group query) '())))
	(make_query_block (qb_schema query) sources
		(map_assoc (qb_fields query) (lambda (_name expression) (tsql_group_projection expression bindings)))
		(qb_where query) (if (nil? (qb_group query)) nil (map (qb_group query) (lambda (key) (tsql_key_formula sources key))))
		(if (nil? (qb_having query)) nil (tsql_group_projection (qb_having query) bindings))
		(map (coalesceNil (qb_order query) '()) (lambda (item)
			(list (tsql_group_projection (tsql_key_formula sources (car item)) bindings) (cadr item))))
		(qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query)))))
(define tsql_rename_query_columns (lambda (query names)
	(if (query_block? query) (make_query_block (qb_schema query) (qb_sources query)
		(merge (mapIndex (extract_assoc (expand_query_block_fields (qb_sources query) (qb_fields query)) (lambda (name expression) (list name expression)))
			(lambda (i field) (list (nth names i) (cadr field)))))
		(qb_where query) (qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query))
		(if (union_block? query) (make_union_block (union_mode query) (map (union_branches query) (lambda (branch) (tsql_rename_query_columns branch names)))
			(union_order query) (union_limit query) (union_offset query) (union_facts query)) (error "unsupported set branch")))))
(define tsql_offset_distinct_union (lambda (query specs) (begin
	(define descriptions (tsql_query_descriptions (car (union_branches query)) nil))
	(define names (map (produceN (count specs)) (lambda (i) (concat "__set_col_" i))))
	(define alias (concat "__set_" (stable_structural_hash query true)))
	(define input (make_union_block 'all (map (union_branches query) (lambda (branch) (tsql_rename_query_columns branch names))) '() nil nil (union_facts query)))
	(define fields (merge (mapIndex descriptions (lambda (i description) (list (description "name") (list 'get_column alias false (nth names i) false))))))
	(tsql_bind_query_block (make_query_block (qb_schema (car (union_branches query))) (list (list alias nil input false true))
		fields true (extract_assoc fields (lambda (_name expression) expression)) nil (union_order query) (union_limit query) (union_offset query) '() '()
		(list (list 'frontend "tsql") (list 'select_distinct true)))))))
(define tsql_bind_query_types (lambda (query) (begin
	(define normalized (normalize_query_ast query))
	(if (query_block? normalized) (tsql_bind_query_block normalized)
		(if (union_block? normalized) (begin
			(define recursively_bound (make_union_block (union_mode normalized) (map (union_branches normalized) tsql_bind_query_types)
				(union_order normalized) (union_limit normalized) (union_offset normalized) (union_facts normalized)))
			(define specs (tsql_union_specs recursively_bound))
			(define coerced (tsql_coerce_query_columns recursively_bound specs))
			(if (and (equal? (union_mode normalized) 'distinct) (reduce specs (lambda (offset spec) (or offset (equal? (car spec) "DATETIMEOFFSET"))) false))
				(tsql_offset_distinct_union coerced specs) coerced)) normalized)))))
(define tsql_average_spec (lambda (spec) (if (tsql_decimal_type? (car spec))
	(list "DECIMAL" tsql_i38 (max tsql_i6 (tsql_spec_scale spec))) (tsql_sum_spec spec))))
(define tsql_sum_reducer (lambda (spec)
	(if (tsql_decimal_type? (car spec))
		(lambda (a b) (if (nil? a) b (if (nil? b) a
			(tsql_decimal_validate (fixed_point_binary "+" a b (tsql_spec_scale spec) (tsql_spec_scale spec) (tsql_spec_scale spec) false) spec))))
		(lambda (a b) (if (nil? a) b (if (nil? b) a
			(if (has? '("FLOAT" "REAL" "DOUBLE") (car spec)) (+ a b)
				(tsql_integer_validate (coefficient_to_integer (fixed_point_binary "+" (integer_to_coefficient a) (integer_to_coefficient b) tsql_i0 tsql_i0 tsql_i0 true) tsql_i0 true) spec))))))))
(define tsql_average_bound (lambda (sum count spec result)
	(if (or (nil? sum) (nil? count) (equal? count tsql_i0)) nil
		(tsql_arithmetic_bound "/" sum count spec '("BIGINT") result))))
(define tsql_negate_bound (lambda (value spec) (if (nil? value) nil (begin
	(define type (car spec))
	(if (has? '("FLOAT" "REAL" "DOUBLE") type) (- tsql_i0 value)
		(if (tsql_decimal_type? type)
			(tsql_decimal_validate (fixed_point_binary "-" (coefficient_encode "0") value (tsql_spec_scale spec) (tsql_spec_scale spec) (tsql_spec_scale spec) false) spec)
			(tsql_integer_validate (coefficient_to_integer (fixed_point_binary "-" (coefficient_encode "0") (integer_to_coefficient value) tsql_i0 tsql_i0 tsql_i0 true) tsql_i0 true)
				(if (equal? type "TINYINT") '("SMALLINT") spec))))))))
(define tsql_math_spec (lambda (op spec)
	(if (equal? op "ROUND") (if (equal? (car spec) "SMALLMONEY") '("MONEY") spec)
		(if (tsql_decimal_type? (car spec)) (list "DECIMAL" tsql_i38 (tsql_spec_scale spec))
			(if (tsql_money_type? (car spec)) '("MONEY") spec)))))
(define tsql_math_bound (lambda (op value places truncate spec result) (if (nil? value) nil (begin
	(if (has? '("FLOAT" "REAL" "DOUBLE") (car spec))
		(match op "ABS" (sql_abs value) "FLOOR" (floor value) "CEILING" (ceil value)
			"ROUND" (begin (define factor (simplify (if (< places tsql_i0) (concat "0." (string_repeat "0" (- (- tsql_i0 places) tsql_i1)) "1") (concat "1" (string_repeat "0" places))))) (define scaled (* value factor))
				(/ (if truncate (if (< scaled tsql_i0) (ceil scaled) (floor scaled)) (round scaled)) factor))
			_ (error "unsupported numeric function"))
		(begin
			(define parts (tsql_value_coefficient value spec)) (define coefficient (car parts)) (define scale (cadr parts))
			(define zero (coefficient_encode "0"))
			(define rounded (match op
				"ABS" (if (< (coefficient_compare coefficient zero) tsql_i0) (fixed_point_binary "-" zero coefficient scale scale scale false) coefficient)
				"ROUND" (begin
					(define target (max tsql_i0 places))
					(if (< places tsql_in38) zero
						(if (< places tsql_i0)
							(begin (define divisor (coefficient_encode (concat "1" (string_repeat "0" (- tsql_i0 places)))))
								(define units (fixed_point_binary "/" coefficient divisor scale tsql_i0 tsql_i0 truncate))
								(fixed_point_binary "*" units divisor tsql_i0 tsql_i0 scale false))
							(fixed_point_rescale (fixed_point_rescale coefficient scale (min target scale) truncate) (min target scale) scale false))))
				_ (begin
					(define units (fixed_point_rescale coefficient scale tsql_i0 true))
					(define whole (fixed_point_rescale units tsql_i0 scale true))
					(define remainder (coefficient_compare coefficient whole))
					(define adjusted (if (or (and (equal? op "FLOOR") (< remainder tsql_i0)) (and (equal? op "CEILING") (> remainder tsql_i0)))
						(fixed_point_binary (if (< remainder tsql_i0) "-" "+") units (coefficient_encode "1") tsql_i0 tsql_i0 tsql_i0 true) units))
					(fixed_point_rescale adjusted tsql_i0 scale false))))
			(if (tsql_decimal_type? (car result)) (tsql_decimal_validate rounded result)
				(tsql_integer_validate (coefficient_to_integer rounded tsql_i0 true) result))))))))

(define tsql_decimal_math (lambda (op value places truncate) (begin
	(define spec (list (sql_runtime_value_type value)))
	(tsql_math_bound op value (coalesceNil places tsql_i0) (not (equal? (coalesceNil truncate tsql_i0) tsql_i0)) spec (tsql_math_spec op spec)))))

/* Comparison coercion is explicit; physical keys at a fixed declaration use
their ordinary integer or canonical bytewise ordering. */
(define tsql_compare_bound (lambda (op left right left_spec right_spec)
	(if (or (nil? left) (nil? right)) nil (begin
		(define floating (or (has? '("FLOAT" "REAL" "DOUBLE") (car left_spec)) (has? '("FLOAT" "REAL" "DOUBLE") (car right_spec))))
		(define comparison (if floating
			(begin (define a (tsql_cast_bound left left_spec '("FLOAT"))) (define b (tsql_cast_bound right right_spec '("FLOAT")))
				(if (< a b) tsql_in1 (if (> a b) tsql_i1 tsql_i0)))
			(if (or (tsql_exact_type? (car left_spec)) (tsql_exact_type? (car right_spec)))
				(begin (define a (tsql_value_coefficient left left_spec)) (define b (tsql_value_coefficient right right_spec))
					(define scale (max (cadr a) (cadr b)))
					(fixed_point_compare (car a) (car b) (cadr a) (cadr b)))
				(if (or (tsql_temporal_type? (car left_spec)) (tsql_temporal_type? (car right_spec)))
					(begin
						(define a (if (equal? (car left_spec) "DATETIMEOFFSET") (tsql_offset_ticks left)
							(if (tsql_temporal_type? (car left_spec)) left (tsql_temporal_cast left left_spec '("DATETIME2" 7)))))
						(define b (if (equal? (car right_spec) "DATETIMEOFFSET") (tsql_offset_ticks right)
							(if (tsql_temporal_type? (car right_spec)) right (tsql_temporal_cast right right_spec '("DATETIME2" 7)))))
						(if (< a b) tsql_in1 (if (> a b) tsql_i1 tsql_i0)))
					(if ((collate "bin" false) left right) tsql_in1 (if ((collate "bin" false) right left) tsql_i1 tsql_i0))))))
		(match op "<" (< comparison tsql_i0) ">" (> comparison tsql_i0) "<=" (<= comparison tsql_i0) ">=" (>= comparison tsql_i0)
			"equal??" (equal? comparison tsql_i0) "equal?" (equal? comparison tsql_i0) _ (error "unsupported comparison"))))))

/* One dialect extension of sql_expr_info. It reads the very same child infos
as the generic binder, then emits ordinary executable formulas. */
(define tsql_bind_expression (lambda (sources head args bindings)
	(match (cons head args)
		((symbol tsql_number_literal) text ((symbol quote) spec))
		(sql_declared_info (if (tsql_decimal_type? (car spec))
			(list 'tsql_decimal_validate (car (tsql_parse_coefficient text)) (list 'quote spec))
			(tsql_numeric_literal text)) spec nil)
		((symbol tsql_decimal_validate) value ((symbol quote) spec)) (sql_declared_info (cons head args) spec nil)
		((symbol tsql_integer_validate) value ((symbol quote) spec)) (sql_declared_info (cons head args) spec nil)
		((symbol tsql_temporal_validate) value ((symbol quote) spec)) (sql_declared_info (cons head args) spec nil)
		((symbol tsql_cast_value) value ((symbol quote) spec)) (begin
			(define spec (tsql_cast_spec spec))
			(define info (sql_expr_info sources value true bindings))
			(if (and (has? '("BINARY" "VARBINARY") (car spec)) (not (has? '("BINARY" "VARBINARY" "ROWVERSION" "TIMESTAMP" "NULL" "any") (sql_info_type info))))
				(error "binary conversion supports raw byte values and NULL") true)
			(sql_declared_info (list 'tsql_cast_bound (sql_info_formula info) (list 'quote (sql_info_spec info)) (list 'quote spec)) spec nil))
		((symbol tsql_cast_bound) value ((symbol quote) source) ((symbol quote) target))
		(sql_declared_info (cons head args) target nil)
		((symbol tsql_convert_value) value ((symbol quote) target) style) (begin
			(define target (tsql_cast_spec target))
			(define info (sql_expr_info sources value true bindings)) (define style_info (sql_expr_info sources style true bindings))
			(sql_declared_info (list 'tsql_convert_bound (sql_info_formula info) (list 'quote (sql_info_spec info))
				(list 'quote target) (sql_info_formula style_info)) target nil))
		((symbol tsql_convert_bound) _value _source ((symbol quote) target) _style) (sql_declared_info (cons head args) target nil)
		((symbol tsql_dateadd) unit amount value) (begin
			(define info (sql_expr_info sources value true bindings)) (define amount_info (sql_expr_info sources amount true bindings))
			(define spec (sql_info_spec info)) (define target (if (tsql_temporal_type? (car spec)) spec '("DATETIME")))
			(sql_declared_info (list 'tsql_dateadd_bound unit (sql_info_formula amount_info) (sql_info_formula info) (list 'quote spec)) target nil))
		((symbol tsql_dateadd_bound) _unit _amount _value ((symbol quote) spec))
		(sql_declared_info (cons head args) (if (tsql_temporal_type? (car spec)) spec '("DATETIME")) nil)
		((symbol tsql_datediff) unit start end) (begin
			(define a (sql_expr_info sources start true bindings)) (define b (sql_expr_info sources end true bindings))
			(sql_declared_info (list 'tsql_datediff_bound unit (sql_info_formula a) (sql_info_formula b)
				(list 'quote (sql_info_spec a)) (list 'quote (sql_info_spec b))) '("INT") nil))
		((symbol tsql_datediff_bound) _unit _start _end _start_spec _end_spec) (sql_declared_info (cons head args) '("INT") nil)
		(cons (symbol tsql_datepart) (cons unit (cons value flags))) (begin
			(define info (sql_expr_info sources value true bindings))
			(sql_declared_info (list 'tsql_datepart_bound unit (sql_info_formula info) (list 'quote (sql_info_spec info))
				(if (empty_list? flags) false (car flags))) '("INT") nil))
		((symbol tsql_datepart_bound) _unit _value _spec _allow_missing) (sql_declared_info (cons head args) '("INT") nil)
		((symbol tsql_nchar) value) (sql_declared_info (list head (sql_info_formula (sql_expr_info sources value true bindings))) '("NVARCHAR" 2) nil)
		((symbol tsql_binary_literal) value) (sql_declared_info (cons head args) (list "VARBINARY" (intdiv (+ (strlen value) tsql_i1) tsql_i2)) nil)
		((symbol window_func) fn values over) (begin
			(define infos (map values (lambda (value) (sql_expr_info sources value true bindings))))
			(define spec (if (empty_list? infos) '("BIGINT") (sql_info_spec (car infos))))
			(define result (match fn "SUM" (tsql_sum_spec spec) "COUNT" '("BIGINT") "ROW_NUMBER" '("BIGINT") "RANK" '("BIGINT") "DENSE_RANK" '("BIGINT") _ spec))
			(define bound_over (list (map (car over) (lambda (key) (tsql_key_formula sources key bindings)))
				(map (cadr over) (lambda (item) (list (tsql_key_formula sources (car item) bindings) (cadr item))))))
			(sql_declared_info (list head fn (map infos (lambda (info)
				(if (equal? fn "SUM") (match (sql_info_formula info)
					((symbol aggregate) _value _reducer _neutral) (sql_info_formula info)
					_ (list 'aggregate (list 'tsql_cast_bound (sql_info_formula info) (list 'quote (sql_info_spec info)) (list 'quote result)) (tsql_sum_reducer result) nil)) (sql_info_formula info)))) bound_over) result nil))
		((symbol tsql_compare_bound) _operator _left _right _left_spec _right_spec) (sql_info (cons head args) "BOOLEAN" nil)
		((symbol tsql_offset_key) value) (sql_declared_info (cons head args) '("BIGINT") nil)
		((symbol count_distinct) value) (sql_declared_info (list head (tsql_key_formula sources value bindings)) '("INT") nil)
		((symbol tsql_isnull) value replacement) (begin
			(define info (sql_expr_info sources value true bindings)) (define other (sql_expr_info sources replacement true bindings))
			(define result (if (equal? (sql_info_type info) "NULL") (sql_info_spec other) (sql_info_spec info)))
			(sql_declared_info (list 'coalesceNil (tsql_coerce_info info result) (tsql_coerce_info other result)) result (sql_info_collation info)))
		(cons (symbol coalesceNil) values) (begin
			(define infos (map values (lambda (value) (sql_expr_info sources value true bindings))))
			(if (reduce infos (lambda (declared info) (or declared (not (nil? (sql_info_declaration info))))) false)
				(begin (define spec (reduce (map infos sql_info_spec) tsql_merge_specs '("NULL")))
					(sql_declared_info (cons head (map infos (lambda (info) (tsql_coerce_info info spec)))) spec nil)) nil))
		(cons (symbol if) values) (begin
			(define infos (map values (lambda (value) (sql_expr_info sources value true bindings))))
			(define value_infos (sql_case_value_infos infos))
			(if (reduce value_infos (lambda (declared info) (or declared (not (nil? (sql_info_declaration info))))) false)
				(begin
					(define spec (reduce (map value_infos sql_info_spec) tsql_merge_specs '("NULL")))
					(define n (count infos))
					(sql_declared_info (cons head (mapIndex infos (lambda (i info)
						(if (or (equal? i (- n tsql_i1)) (equal? (tsql_positive_mod i tsql_i2) tsql_i1)) (tsql_coerce_info info spec) (sql_info_formula info))))) spec nil)) nil))
		((symbol tsql_clock_value) type) (sql_declared_info (list 'tsql_clock_at 'tsql_statement_values type) (if (equal? type "DATETIME2") '("DATETIME2" 7) (list type)) nil)
		((symbol tsql_clock_at) _context type) (sql_declared_info (cons head args) (if (equal? type "DATETIME2") '("DATETIME2" 7) (list type)) nil)
		((symbol tsql_parameter) value ((symbol quote) spec)) (sql_declared_info value spec nil)
		((symbol tsql_negate) value) (begin
			(define info (sql_expr_info sources value true bindings)) (define spec (sql_info_spec info))
			(if (or (has? '("any" "NULL" "FLOAT" "REAL" "DOUBLE" "INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") (car spec)) (tsql_exact_type? (car spec))) true
				(error "unary negation requires a numeric operand"))
			(sql_declared_info (list 'tsql_negate_bound (sql_info_formula info) (list 'quote spec)) (if (equal? (car spec) "TINYINT") '("SMALLINT") spec) nil))
		((symbol tsql_negate_bound) value ((symbol quote) spec)) (sql_declared_info (cons head args) (if (equal? (car spec) "TINYINT") '("SMALLINT") spec) nil)
		((symbol tsql_arithmetic_bound) _op _left _right _ls _rs ((symbol quote) result)) (sql_declared_info (cons head args) result nil)
		((symbol tsql_math_bound) _op _value _places _truncate _spec ((symbol quote) result)) (sql_declared_info (cons head args) result nil)
		((symbol tsql_average_bound) _sum _count _spec ((symbol quote) result)) (sql_declared_info (cons head args) result nil)
		((symbol tsql_sum_value) value) (begin
			(define info (sql_expr_info sources value true bindings)) (define target (tsql_sum_spec (sql_info_spec info)))
			(sql_declared_info (list 'tsql_cast_bound (sql_info_formula info) (list 'quote (sql_info_spec info)) (list 'quote target)) target nil))
		((symbol aggregate) value reducer neutral) (begin
			(define info (sql_expr_info sources value true bindings)) (define spec (sql_info_spec info))
			(define count_result (and (or (equal? reducer '+) (equal? reducer +)) (equal? neutral tsql_i0)))
			(sql_declared_info (list head (sql_info_formula info)
				(if (or (equal? reducer 'sql_sum_reduce) (equal? reducer sql_sum_reduce)) (tsql_sum_reducer spec) reducer) neutral)
				(if count_result '("INT") spec) (sql_info_collation info)))
		((symbol tsql_avg_value) input sum count) (begin
			(define info (sql_expr_info sources input true bindings)) (define sum_info (sql_expr_info sources sum true bindings)) (define count_info (sql_expr_info sources count true bindings))
			(define result (tsql_average_spec (sql_info_spec info)))
			(sql_declared_info (list 'tsql_average_bound (sql_info_formula sum_info) (sql_info_formula count_info)
				(list 'quote (tsql_sum_spec (sql_info_spec info))) (list 'quote result)) result nil))
		(cons (symbol tsql_decimal_math) (cons op (cons value rest))) (begin
			(define info (sql_expr_info sources value true bindings)) (define spec (sql_info_spec info)) (define result (tsql_math_spec op spec))
			(sql_declared_info (list 'tsql_math_bound op (sql_info_formula info)
				(if (empty_list? rest) tsql_i0 (sql_info_formula (sql_expr_info sources (car rest) true bindings)))
				(if (> (count rest) tsql_i1) (list 'not (list 'equal? (sql_info_formula (sql_expr_info sources (cadr rest) true bindings)) tsql_i0)) false)
				(list 'quote spec) (list 'quote result)) result nil))
		(cons op operands) (if (and (equal? (count operands) tsql_i2) (has? (list 'tsql_add 'tsql_subtract 'tsql_multiply 'tsql_divide 'tsql_remainder) op))
			(begin
				(define left (sql_expr_info sources (car operands) true bindings)) (define right (sql_expr_info sources (cadr operands) true bindings))
				(define operator (match op (symbol tsql_add) "+" (symbol tsql_subtract) "-" (symbol tsql_multiply) "*" (symbol tsql_divide) "/" _ "%"))
				(define ls (sql_info_spec left)) (define rs (sql_info_spec right))
				(define concatenated (tsql_concat_type operator (car ls) (car rs)))
				(define result (if concatenated (list concatenated) (tsql_arithmetic_spec operator ls rs)))
				(sql_declared_info (list 'tsql_arithmetic_bound operator (sql_info_formula left) (sql_info_formula right)
					(list 'quote ls) (list 'quote rs) (list 'quote result)) result nil))
			(if (and (equal? (count operands) tsql_i2) (has? (list 'equal?? 'equal? '< '> '<= '>=) op))
				(begin
					(define left (sql_expr_info sources (car operands) true bindings)) (define right (sql_expr_info sources (cadr operands) true bindings))
					(define ls (sql_info_spec left)) (define rs (sql_info_spec right))
					(if (or (sql_info_declaration left) (sql_info_declaration right))
						(if (and (equal? ls rs) (not (has? '("DATETIMEOFFSET" "BINARY" "VARBINARY" "ROWVERSION" "TIMESTAMP") (car ls))))
							(sql_comparison_info op left right)
							(sql_info (list 'tsql_compare_bound (string op) (sql_info_formula left) (sql_info_formula right)
								(list 'quote ls) (list 'quote rs)) "BOOLEAN" nil)) nil)) nil))
		_ nil)))

/* Protocols execute immutable recipes selected by this frontend. An empty
result and a NULL first row use precisely the same recipe as every other row. */
(define tsql_description_spec (lambda (description) (begin
	(define type (description "sql_type"))
	(cons type (if (tsql_decimal_type? type) (list (coalesceNil (description "precision") tsql_i18) (coalesceNil (description "scale") tsql_i0))
		(if (has? '("DATETIME2" "DATETIMEOFFSET" "TIME") type) (list (coalesceNil (description "scale") tsql_i7))
			(if (nil? (description "size")) '() (list (description "size")))))))))
(define tsql_temporal_wire (lambda (value spec) (begin
	(define type (car spec)) (define ticks (if (equal? type "DATETIMEOFFSET") (tsql_offset_ticks value) value))
	(define offset (if (equal? type "DATETIMEOFFSET") (tsql_offset_value_minutes value) tsql_i0))
	(define days (tsql_floor_divide ticks tsql_ticks_per_day))
	(define clock (tsql_positive_mod ticks tsql_ticks_per_day))
	(define origin (if (has? '("DATETIME" "SMALLDATETIME") type) tsql_in25567 (- tsql_i0 tsql_unix_epoch_days)))
	(list (- days origin)
		(match type "DATETIME" (intdiv (+ (* clock tsql_i300) tsql_i5000000) tsql_i10000000)
			"SMALLDATETIME" (intdiv clock tsql_i600000000)
			_ (intdiv clock (tsql_power_ten (- tsql_i7 (tsql_spec_scale spec))))) offset))))
(define tsql_export_procedure (lambda (spec)
	(if (tsql_exact_type? (car spec)) (lambda (value) (if (nil? value) nil
		(json_number (if (tsql_decimal_type? (car spec)) (coefficient_format value (tsql_spec_scale spec))
			(coefficient_format (integer_to_coefficient value) tsql_i4)))))
		(if (tsql_temporal_type? (car spec)) (lambda (value) (if (nil? value) nil (tsql_temporal_text value spec tsql_i0)))
			(lambda (value) value)))))
(define tsql_encode_description (lambda (spec) (begin
	(define type (car spec))
	(define kind (get_assoc '("TINYINT" 38 "SMALLINT" 38 "INT" 38 "INTEGER" 38 "BIGINT" 38
		"BIT" 104 "BOOLEAN" 104 "FLOAT" 109 "DOUBLE" 109 "REAL" 109
		"MONEY" 110 "SMALLMONEY" 110 "DECIMAL" 106 "NUMERIC" 108
		"DATE" 40 "TIME" 41 "DATETIME2" 42 "DATETIMEOFFSET" 43 "DATETIME" 111 "SMALLDATETIME" 111
		"VARCHAR" 167 "TEXT" 167 "CHAR" 175 "NVARCHAR" 231 "NTEXT" 231 "NCHAR" 239
		"BINARY" 173 "VARBINARY" 165 "ROWVERSION" 173 "TIMESTAMP" 173) type))
	(define scale (tsql_spec_scale spec))
	(define size (match type
		"TINYINT" tsql_i1 "SMALLINT" tsql_i2 "INT" tsql_i4 "INTEGER" tsql_i4 "BIGINT" tsql_i8 "BIT" tsql_i1 "BOOLEAN" tsql_i1
		"REAL" tsql_i4 "FLOAT" (if (and (not (nil? (tsql_spec_dimension spec tsql_i1))) (<= (tsql_spec_dimension spec tsql_i1) tsql_i24)) tsql_i4 tsql_i8) "DOUBLE" tsql_i8 "MONEY" tsql_i8 "SMALLMONEY" tsql_i4 "DATETIME" tsql_i8 "SMALLDATETIME" tsql_i4
		"ROWVERSION" tsql_i8 "TIMESTAMP" tsql_i8 "DATE" tsql_i3
		_ (if (tsql_decimal_type? type) (if (<= (tsql_spec_precision spec) tsql_i9) tsql_i5 (if (<= (tsql_spec_precision spec) tsql_i19) tsql_i9 (if (<= (tsql_spec_precision spec) tsql_i28) tsql_i13 tsql_i17)))
			(if (has? '("TIME" "DATETIME2" "DATETIMEOFFSET") type)
				(+ (if (<= scale tsql_i2) tsql_i3 (if (<= scale tsql_i4) tsql_i4 tsql_i5)) (match type "DATETIME2" tsql_i3 "DATETIMEOFFSET" tsql_i5 _ tsql_i0))
				(if (has? '("TEXT" "NTEXT") type) tsql_in1 (if (nil? (tsql_spec_dimension spec tsql_i1)) tsql_i8000
					(if (has? '("NVARCHAR" "NCHAR") type) (if (equal? (tsql_spec_dimension spec tsql_i1) tsql_in1) tsql_in1 (* tsql_i2 (tsql_spec_dimension spec tsql_i1))) (tsql_spec_dimension spec tsql_i1))))))))
	(define encode (if (tsql_temporal_type? type) (lambda (value) (tsql_temporal_wire value spec)) (lambda (value) value)))
	(if kind (list "wire_kind" (tsql_descriptor_integer kind) "wire_size" (if (equal? size tsql_in1) tsql_i65535 (tsql_descriptor_integer size)) "precision" (if (tsql_decimal_type? type) (tsql_descriptor_integer (tsql_spec_precision spec)) nil)
		"scale" (tsql_descriptor_integer scale) "encode" encode "export" (tsql_export_procedure spec)) (list "export" (lambda (value) value))))))

(define tsql_parse_declarations (lambda (text schema) (begin
	(define type_spec (tsql_type_specification schema nil))
	(define declaration (parser '((atom "@" false) (define name tsql_identifier)
		(define spec type_spec) (? (atom "OUTPUT" true)))
		(list (toLower name) (cons (car spec) (filter (cdr spec) (lambda (dimension) (not (nil? dimension))))))))
	(if (equal? (regexp_replace text "^\\s+|\\s+$" "") "") '()
		(begin (define declarations ((parser (* declaration ",")) text))
			(reduce declarations (lambda (seen item) (if (has? seen (car item)) (error "duplicate parameter declaration") (append seen (car item)))) '())
			declarations)))))
/* Resolve user declarations against the current frontend catalog before the
transport converts values. A declaration does not depend on its current value. */
(define tsql_bind_declarations (lambda (schema declarations)
	(map declarations (lambda (entry) (begin
		(define spec (cadr entry)) (define alias (sql_type_alias schema (car spec)))
		(if (and alias (not (empty_list? (cdr spec)))) (error "alias parameter types do not accept dimensions") true)
		(define target (if alias (cons (alias "BaseType") (alias "Dimensions")) spec))
		(list (car entry) (cons (car target) (tsql_declaration_dimensions target))))))))
(define tsql_wire_length (lambda (wire unicode) (begin
	(define width (wire "wire_size"))
	(if (equal? width tsql_i65535) tsql_in1 (if unicode (intdiv width tsql_i2) width)))))
(define tsql_wire_spec (lambda (wire) (match (wire "wire_kind")
	106 (list "DECIMAL" (wire "precision") (wire "scale")) 108 (list "NUMERIC" (wire "precision") (wire "scale"))
	110 (if (equal? (wire "wire_size") tsql_i4) '("SMALLMONEY") '("MONEY")) 111 (if (equal? (wire "wire_size") tsql_i4) '("SMALLDATETIME") '("DATETIME"))
	40 '("DATE") 41 (list "TIME" (wire "scale")) 42 (list "DATETIME2" (wire "scale")) 43 (list "DATETIMEOFFSET" (wire "scale"))
	38 (match (wire "wire_size") 1 '("TINYINT") 2 '("SMALLINT") 4 '("INT") _ '("BIGINT"))
	104 '("BIT") 109 (if (equal? (wire "wire_size") tsql_i4) '("REAL") '("FLOAT"))
	173 (list "BINARY" (tsql_wire_length wire false)) 165 (list "VARBINARY" (tsql_wire_length wire false))
	231 (list "NVARCHAR" (tsql_wire_length wire true)) 239 (list "NCHAR" (tsql_wire_length wire true))
	167 (list "VARCHAR" (tsql_wire_length wire false)) 175 (list "CHAR" (tsql_wire_length wire false))
	34 '("VARBINARY" -1) 35 '("TEXT") 99 '("NTEXT") _ '("VARCHAR" -1))))
/* Wire widths only establish a transport envelope. Declared precision, calendar
ranges and clock limits are validated here before converting to another type. */
(define tsql_parameter_temporal (lambda (raw source) (begin
	(define type (car source)) (define days (car raw)) (define units (cadr raw)) (define offset (coalesceNil (nth raw tsql_i2) tsql_i0))
	(define units_per_day (match type "DATE" tsql_i1 "DATETIME" tsql_i25920000 "SMALLDATETIME" tsql_i1440
		_ (* tsql_i86400 (tsql_power_ten (tsql_spec_scale source)))))
	(define minimum_day (if (equal? type "DATETIME") tsql_in53690 tsql_i0))
	(define maximum_day (match type "DATETIME" tsql_i2958463 "SMALLDATETIME" tsql_i65535 "TIME" tsql_i0 _ tsql_i3652058))
	(if (and (>= days minimum_day) (<= days maximum_day) (>= units tsql_i0) (< units units_per_day) (or (not (equal? type "DATE")) (equal? units tsql_i0))
		(or (not (equal? type "TIME")) (equal? days tsql_i0))
		(if (equal? type "DATETIMEOFFSET") (and (>= offset tsql_in840) (<= offset tsql_i840)) (equal? offset tsql_i0))) true
		(error "temporal parameter outside declaration range"))
	(define origin (if (has? '("DATETIME" "SMALLDATETIME") type) tsql_in25567 (- tsql_i0 tsql_unix_epoch_days)))
	(define clock (match type "DATETIME" (intdiv (+ (* units tsql_i10000000) tsql_i150) tsql_i300)
		"SMALLDATETIME" (* units tsql_i600000000)
		_ (* units (tsql_power_ten (- tsql_i7 (tsql_spec_scale source))))))
	(define ticks (if (equal? type "TIME") clock (+ (* (+ days origin) tsql_ticks_per_day) clock)))
	(tsql_temporal_validate (if (equal? type "DATETIMEOFFSET") (tsql_offset_value ticks offset) ticks) source))))
(define tsql_convert_parameter (lambda (raw wire target) (if (nil? raw) nil (begin
	(define source (tsql_wire_spec wire)) (define type (car source))
	(tsql_declaration_dimensions source)
	(define value (if (tsql_temporal_type? type) (tsql_parameter_temporal raw source)
		(if (tsql_decimal_type? type) (tsql_decimal_validate raw source)
			(if (or (tsql_money_type? type) (has? '("INT" "BIGINT" "SMALLINT" "TINYINT") type)) (tsql_integer_validate raw source) raw))))
	(tsql_cast_bound value source (coalesceNil target source))))))

(define tsql_count_insert_rows (lambda (action) (begin
	(define output (newsession))
	(collect_reports action tsql_i0 (lambda (total count) (+ total count)) (lambda (total success) (output "count" total)))
	(output "count"))))
(define tsql_insert_identity_values (lambda (session action)
	(collect_reports action nil (lambda (_previous value) value) (lambda (value success)
		(if (not (nil? value)) (begin
			(session "last_insert_id" value) (session "tsql_scope_identity" value) (session "tsql_last_identity" value)) true)))))
