/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Terminal XML AUTO serialization consumes the ordinary planned row stream.
Only a processed-row count and its SQL text scalar survive the reduction.
https://learn.microsoft.com/sql/relational-databases/xml/use-auto-mode-with-for-xml
https://learn.microsoft.com/sql/t-sql/statements/set-rowcount-transact-sql
SQL Server 2022: https://dbfiddle.uk/YyoMWOWC https://dbfiddle.uk/21t1BoQ7 */
(define tsql_set_rowcount (close_procedure ((lambda (raise)
	(lambda (value session) (begin
		(if (nil? value) (raise 507 "Invalid value for ROWCOUNT" 1)
			(< value 0) (raise 507 "Invalid value for ROWCOUNT" 1) true)
		(session "__tsql_rowcount_limit" value)
		0))) tsql_raise_error)))
(define tsql_rowcount_setting_compiler (lambda (infos) (begin
	(define input (car infos))
	(if (has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "NULL") (sql_info_type input)) input
		(error "ROWCOUNT requires a declared integral setting")))))
(define tsql_rowcount_limit_formula (list 'coalesceNil (list 'session "__tsql_rowcount_limit") 0))
(define tsql_rowcount_bound (lambda (limit)
	(if (nil? limit) tsql_rowcount_limit_formula
		(list 'min limit tsql_rowcount_limit_formula))))
(define tsql_apply_rowcount_limit (lambda (query) (match query
	((symbol tsql_pending_with) definitions body)
	(list 'tsql_pending_with definitions (tsql_apply_rowcount_limit body))
	((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
	(make_query_block db sources fields condition group having order (tsql_rowcount_bound limit) offset hidden stages facts)
	((symbol union-block) mode branches order limit offset facts)
	(make_union_block mode branches order (tsql_rowcount_bound limit) offset facts)
	_ (error "ROWCOUNT requires an ordinary query root"))))
(define tsql_for_xml_mode (lambda (query) (match query
	((symbol tsql_pending_with) _definitions body) (tsql_for_xml_mode body)
	((symbol query-block) _db _sources _fields _where _group _having _order _limit _offset _hidden _stages facts)
	(qassoc_get facts 'tsql_parser_for_xml nil)
	((symbol union-block) _mode _branches _order _limit _offset facts) (qassoc_get facts 'tsql_parser_for_xml nil)
	_ nil)))
(define tsql_for_xml_input (lambda (query) (match query
	((symbol tsql_pending_with) definitions body)
	(list 'tsql_pending_with definitions (tsql_for_xml_input body))
	((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
	(make_query_block db sources fields condition group having order limit offset hidden stages
		(filter facts (lambda (entry) (not (equal? (car entry) 'tsql_parser_for_xml)))))
	_ (error "XML AUTO set-operation roots require an XML hierarchy producer"))))
(define tsql_for_xml_alias (lambda (query) (match query
	((symbol tsql_pending_with) _definitions body) (tsql_for_xml_alias body)
	((symbol query-block) _db sources _fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(if (equal? (count sources) 1) (begin
		(define relation (source_relation (car sources)))
		(match relation
			((symbol tsql_pending_full_join) _left _right _predicate) (error "XML AUTO joined operands require a hierarchy")
			((symbol tsql_pending_join_operand) _left _right) (error "XML AUTO joined operands require a hierarchy")
			((symbol tsql_pending_nullable_join) _left _right) (error "XML AUTO joined operands require a hierarchy")
			_ true)
		(coalesceNil (get_assoc (coalesceNil (source_declarations (car sources)) '()) "tsql_xml_auto_name" nil)
			(source_alias (car sources))))
		(error "XML AUTO over several sources requires the hierarchical serializer"))
	_ (error "XML AUTO requires a query block"))))
(define tsql_xml_hex (close_procedure (lambda (value width) (begin
	(define bytes (map (produceN width) (lambda (index)
		(byte_char (mod (bit_shift_right (intdiv value 1) (intdiv (* 8 (- width index 1)) 1)) 256)))))
	(toUpper (bin2hex (apply concat bytes)))))))
(define tsql_xml_codepoint (close_procedure (lambda (character) (begin
	(define first (byte_value character 0))
	(if (< first 128) first
		(< first 224) (+ (* (- first 192) 64) (- (byte_value character 1) 128))
		(< first 240) (+ (* (- first 224) 4096) (* (- (byte_value character 1) 128) 64) (- (byte_value character 2) 128))
		(+ (* (- first 240) 262144) (* (- (byte_value character 1) 128) 4096)
			(* (- (byte_value character 2) 128) 64) (- (byte_value character 3) 128)))))))
/* This is SQL identifier encoding, not XML document parsing. AUTO's untyped
string form preserves colons and encodes its reserved lower-case _x marker. */
(define tsql_xml_name (lambda (name) (begin
	(if (equal? name "") (error "XML AUTO requires named output columns") true)
	(define characters (split name ""))
	(apply concat (mapIndex characters (lambda (index character) (begin
		(define reserved (and (equal? character "_")
			(regexp_test (apply concat (slice characters index (count characters))) "^_x(?:[0-9A-Fa-f]{4}|[0-9A-Fa-f]{8})_")))
		(if (or reserved (not (regexp_test character (if (equal? index 0) "^[:_\\p{L}]$" "^[:_.·\\p{L}\\p{N}\\p{Mn}\\p{Mc}-]$"))))
			(begin (define code (tsql_xml_codepoint character))
				(concat "_x" (tsql_xml_hex code (if (> code 65535) 4 2)) "_")) character))))))))
(define tsql_xml_attribute_replacements (map (merge (list (produceN 32) '(65534 65535))) (lambda (code)
	(list (unicode_char code) (concat "&#x" (tsql_xml_hex code (if (> code 255) 2 1)) ";")))))
(define tsql_xml_escape_attribute (close_procedure ((lambda (replacements)
	(lambda (text) (reduce replacements (lambda (result pair) (replace result (car pair) (cadr pair)))
		(replace (replace (replace (replace text "&" "&amp;") "\"" "&quot;") "<" "&lt;") ">" "&gt;")))) tsql_xml_attribute_replacements)))
(define tsql_for_xml_attribute (close_procedure ((lambda (escape)
	(lambda (name text) (if (nil? text) "" (concat " " name "=\"" (escape text) "\"")))) tsql_xml_escape_attribute)))
(define tsql_for_xml_value_formula (lambda (info value) (begin
	(define type (sql_info_type info))
	(define typed (list 'sql_type_expr value (sql_info_datatype info) (sql_info_collation info)))
	(if (or (tsql_temporal_type? type)
		(has? '("NULL" "TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "BIT" "BOOLEAN" "BOOL" "DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY"
			"CHAR" "VARCHAR" "TEXT" "NCHAR" "NVARCHAR" "NTEXT" "UNIQUEIDENTIFIER") type)) true
		(error "XML AUTO serialization requires a declared supported scalar contract: " type))
	(if (has? '("FLOAT" "REAL" "XML" "XML_NODE" "BINARY" "VARBINARY" "IMAGE" "ROWVERSION" "SQL_VARIANT") type)
		(error "XML AUTO scalar serialization requires a distinct contract for " type) true)
	(if (tsql_temporal_type? type) (tsql_temporal_style_expr (sql_info value (sql_info_datatype info) (sql_info_collation info)) 126 false)
		(has? '("MONEY" "SMALLMONEY") type) (list tsql_decimal_text value 4)
		(begin
			(define typed (if (has? '("CHAR" "NCHAR") type)
				(tsql_cast_expr typed type (sql_info_dimensions info)) typed))
			(sql_type_formula '() (tsql_cast_expr typed "NVARCHAR" '(-1))))))))
(define tsql_for_xml_reduce (close_procedure (lambda (state fragment)
	(list (+ (car state) 1) (concat (cadr state) fragment)))))
(define tsql_for_xml_combine (close_procedure (lambda (left right)
	(list (+ (car left) (car right)) (concat (cadr left) (cadr right))))))
(define tsql_for_xml_column "XML_F52E2B61-18A1-11d1-B105-00805F49916B")
(define tsql_for_xml_descriptors (lambda (database parameters)
	(tsql_result_columns (make_query_block database '()
		(list tsql_for_xml_column (list 'sql_type_expr nil "NTEXT" nil)) true nil nil nil nil nil nil nil nil) parameters)))
(define tsql_for_xml_plan (lambda (query alias parameters planning tx) (begin
	(define titles (queryplan_result_titles query))
	(define row '__xml_auto_row)
	(define formatter (list 'lambda (list row) (cons 'concat (merge (list
		(list (concat "<" (tsql_xml_name alias)))
		(mapIndex titles (lambda (index title) (begin
			(define info (sql_projection_info_at query index '()))
			(list tsql_for_xml_attribute (tsql_xml_name title)
				(tsql_for_xml_value_formula info (list 'nth row (+ (* index 2) 1)))))))
		(list "/>"))))))
	(define consumer (make_result_reduction tsql_for_xml_reduce '(0 "") tsql_for_xml_combine))
	(define input (list (list 'lambda (list 'resultrow)
		(build_reducing_queryplan_term query consumer planning tx nil)) formatter))
	(define descriptors (tsql_for_xml_descriptors (qb_schema query) parameters))
	(define state '__xml_auto_result)
	(list '!begin
		(list 'resultfields (list 'quote (list tsql_for_xml_column)) (list 'quote descriptors))
		(list (list 'lambda (list state) (list '!begin
			(list 'if (list '> (list 'car state) 0)
				(list 'resultrow (list 'list tsql_for_xml_column (list 'cadr state))) true)
			(list 'car state))) input)))))
