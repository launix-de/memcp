/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* XML is a declared SQL scalar. Native codecs receive Unicode data and QName
steps only; SQL conversion, XQuery spelling and context-node policy stay here.
https://learn.microsoft.com/sql/t-sql/xml/nodes-method-xml-data-type
https://learn.microsoft.com/sql/t-sql/xml/value-method-xml-data-type */
/* XML method identifiers are case-sensitive, unlike ordinary T-SQL words.
Their continuation boundary and leading trivia still use the common lexer. */
(define tsql_xml_method_parser (lambda (word)
	(eval (list 'parser
		(list 'define 'token (list 'capture (list 'not
			(list 'atom word false false)
			(list 'regex (concat word tsql_identifier_continue_pattern) false false))))
		(list 'cadr 'token)))))
(define tsql_xml_nodes_method (tsql_xml_method_parser "nodes"))
(define tsql_xml_value_method (tsql_xml_method_parser "value"))
(define tsql_xml_error (close_procedure ((lambda (raise)
	(lambda (failure) (match (failure_value failure)
		'("xml-codec-error" "mismatched-end" message) (raise 9436 message 1)
		'("xml-codec-error" "duplicate-attribute" message) (raise 9437 message 1)
		'("xml-codec-error" "namespace-unbound" message) (raise 9459 message 1)
		'("xml-codec-error" "namespace-reserved" message) (raise 9464 message 1)
		'("xml-codec-error" "unexpected-end" message) (raise 9400 message 1)
		'("xml-codec-error" "document-type" message) (raise 6359 message 1)
		_ (error failure)))) tsql_raise_error)))
(define tsql_xml_cast_value (close_procedure ((lambda (parse encoding raise xml_error)
	(lambda (value preserve source_encoding scope) (if (nil? value) nil
		(try (lambda () (begin
			(define declared (encoding value scope))
			(if (nil? declared) true (begin
				(define declared (toLower declared))
				(if (equal? source_encoding "unicode")
					(if (has? '("utf-16" "utf-16le" "ucs-2") declared) true
						(raise 9402 "unable to switch the XML encoding" 1))
					(or (equal?? declared source_encoding)
						(and (equal? source_encoding "windows-1252") (has? '("windows-1252" "cp1252" "iso-8859-1") declared))) true
					(error "XML source encoding conversion is not implemented for " declared))))
			(parse value preserve scope))) xml_error)))) xml_parse xml_declared_encoding tsql_raise_error tsql_xml_error)))
(define tsql_xml_cast_compiler (lambda (preserve) (lambda (infos) (begin
	(define input (car infos))
	(define source (sql_info_type input))
	(if (equal? source "XML") input
		(equal? source "NULL") (sql_info nil "XML" nil)
		(begin
			(if (tsql_binary_type? source)
				(error "binary XML conversion requires a byte-stream XML codec") true)
			(if (has? '("CHAR" "VARCHAR" "TEXT" "NCHAR" "NVARCHAR" "NTEXT") source) true
				(tsql_raise_error 529 (concat "explicit conversion from " source " to XML is not allowed") 2))
			(sql_info (list tsql_xml_cast_value (sql_info_formula input) preserve
				(if (has? '("NCHAR" "NVARCHAR" "NTEXT") source) "unicode" (tsql_character_encoding input)) 'tx)
				"XML" nil)))))))
(define tsql_xml_cast_expr (lambda (value preserve)
	(list 'sql_compile_expr (list 'tsql_xml_cast_compiler preserve) value)))
(define tsql_xml_value_type (parser '((define type tsql_declared_type)
	(define dimensions tsql_type_dimensions) (regex "\\s*\\z" false false))
	(cons type (tsql_decl_dimensions type dimensions))))
(define tsql_xml_value_compiler (lambda (path type) (lambda (infos) (begin
	(if (equal? (sql_trim path) ".") true
		(error "XML value paths other than the singleton context require XQuery lowering"))
	(define input (car infos))
	(if (has? '("XML" "XML_NODE") (sql_info_type input)) true
		(error "XML value requires a declared XML input"))
	(define contract (tsql_xml_value_type type))
	(if (has? '("XML" "IMAGE" "TEXT" "NTEXT" "SQL_VARIANT") (car contract))
		(error "invalid XML value return type") true)
	(define value (list 'if (list 'nil? (sql_info_formula input)) nil
		(list 'xml_string_value (sql_info_formula input) 'tx)))
	(sql_expr_info '() (tsql_cast_expr (list 'sql_type_expr value (list "NVARCHAR" '(-1)) nil)
		(car contract) (cdr contract)))))))
/* The first path profile is arbitrary absolute or relative child-element chains. No
regular expression parses XML data. QName spelling is an XQuery frontend rule;
the codec matches expanded names, including the empty namespace. */
(define tsql_xml_child_name (parser '((regex "[ \t\r\n]*" false false)
	(define name (regex "[\\p{L}_][\\p{L}\\p{N}\\p{Mn}\\p{Mc}_.·-]*" false false))
	(regex "[ \t\r\n]*" false false)) name))
(define tsql_xml_child_path (parser '((regex "[ \t\r\n]*" false false)
	(? (atom "/" false false))
	(define names (+ tsql_xml_child_name (atom "/" false false)))
	(regex "\\z" false false)) (map names (lambda (name) (list "" name)))))

(define tsql_xml_nodes_query (lambda (database document path alias column) (begin
	(define info (sql_expr_info '() document))
	(if (equal? (sql_info_type info) "XML") true
		(error "XML nodes requires an independent declared XML scalar"))
	(define steps (tsql_xml_child_path path))
	(define relation (symbol (concat "__xml_relation:" alias)))
	(define native (list (list "column" column "any" '() '())))
	(define columns (list (list "Field" column "Type" "XML_NODE" "RawType" "XML_NODE"
		"Dimensions" '() "Null" false "Collation" "")))
	(define source (list alias database (make_local_relation_reference relation columns) false nil
		(list "lifetime" "invocation" "columns" columns "constraints" '("Unique" () "ForeignKeys" ()))))
	(define query (make_query_block database (list source)
		(list column (list 'get_column alias false column false)) true nil nil nil nil nil nil nil nil))
	(define input (make_query_block database '()
		(list "document" document "steps" (list 'quote steps))
		(list 'not (list 'nil? document)) nil nil nil nil nil nil nil nil))
	(define initialize (make_local_relation_emit input relation (list column) 'xml_children_each '()))
	(make_invocation_relation_query query (list (list relation database native columns))
		(car (qb_stages initialize))))))

/* SQL Server excludes XML methods directly inside a GROUP BY expression.
Derived scalar outputs are ordinary columns and retain their usual binding. */
(define tsql_xml_validate_group_key (lambda (expression) (match expression
	((symbol sql_compile_expr) ((symbol tsql_xml_value_compiler) _path _type) _receiver)
	(tsql_raise_error 4148 "XML methods are not allowed in GROUP BY" 1)
	((symbol quote) _value) true
	(cons _head arguments) (begin (map arguments tsql_xml_validate_group_key) true)
	_ true)))
