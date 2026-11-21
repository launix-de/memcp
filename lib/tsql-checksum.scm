/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* CHECKSUM's weights are SQL collation facts, not the native ordering key.
This bounded CP1 table was measured against SQL Server 2022 for every byte,
including prefix/suffix/interior positions; the three expansions consume two
rotation slots. Interior space has weight 32, whereas trailing U+0020 is trimmed.
https://dbfiddle.uk/bMdlywsT
https://learn.microsoft.com/sql/t-sql/functions/checksum-transact-sql */
(define tsql_checksum_cp1_weights '(
	0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
	16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31
	32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47
	132 133 134 135 136 137 138 139 140 141 48 49 50 51 52 53
	54 142 143 144 145 146 147 148 149 150 151 152 153 154 155 156
	157 158 159 160 161 162 163 164 165 166 167 55 56 57 58 59
	60 142 143 144 145 146 147 148 149 150 151 152 153 154 155 156
	157 158 159 160 161 162 163 164 165 166 167 61 62 63 64 65
	66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81
	82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97
	98 99 100 101 102 103 104 105 106 107 108 109 110 111 112 113
	114 115 116 117 118 119 120 121 122 123 124 125 126 127 128 129
	654 910 1166 1422 1678 1934 6514 400 658 914 1170 1426 406 662 918 1174
	168 411 412 668 924 1180 1436 130 1692 418 674 930 1186 422 169 7072
	654 910 1166 1422 1678 1934 6514 400 658 914 1170 1426 406 662 918 1174
	168 411 412 668 924 1180 1436 131 1692 418 674 930 1186 422 169 678))
(define tsql_checksum_rotate (close_procedure (lambda (value bits)
	(bit_and 4294967295 (bit_or (bit_shift_left value bits)
		(bit_shift_right value (intdiv (- 32 bits) 1)))))))
(define tsql_checksum_signed (close_procedure (lambda (value)
	(if (>= value 2147483648) (- value 4294967296) value))))
(define tsql_checksum_combine (close_procedure ((lambda (rotate signed)
	(lambda (values)
		(signed (reduce values (lambda (checksum value)
			(bit_xor (rotate checksum 4) value)) 0))))
	tsql_checksum_rotate tsql_checksum_signed)))
/* Integer width is selected by the declaration compiler. BIT has the native
Boolean carrier; BIGINT folds its two 32-bit words, not a floating conversion. */
(define tsql_checksum_integer (close_procedure (lambda (value wide boolean)
	(if (nil? value) 2147483647
		boolean (if value 1 0)
		wide (bit_xor (bit_and value 4294967295) (bit_shift_right value 32))
		(bit_and value 4294967295)))))
/* The byte string is an explicit frontend encoding, never a runtime SQL type.
The accumulator is one integer; reduce_range does not collect characters. */
(define tsql_checksum_encoded (close_procedure ((lambda (weights rotate)
	(lambda (bytes cp1 national)
		(reduce_range (intdiv (strlen bytes) (if national 2 1))
			(lambda (checksum index) (begin
				(define unit (if national
					(intdiv (+ (byte_value bytes (* index 2)) (bit_shift_left (byte_value bytes (+ (* index 2) 1)) 8)) 1)
					(byte_value bytes index)))
				(bit_xor (rotate checksum (if (and cp1 (has? '(198 223 230) unit)) 8 4))
					(if cp1 (nth weights unit) unit)))) 0)))
	tsql_checksum_cp1_weights tsql_checksum_rotate)))
(define tsql_checksum_text (close_procedure ((lambda (encoded)
	(lambda (value cp1 national)
		(if (nil? value) 2147483647 (begin
			(define text (regexp_replace value " +$" ""))
			(encoded (if national (utf16le_encode text)
				(single_byte_encode text "windows-1252")) cp1 national)))))
	tsql_checksum_encoded)))
/* Compile only proven declaration/weight contracts. Native UTF8 comparison
labels are not SQL CP1 identities. In particular this does not silently change
CREATE DATABASE, column, literal or server default-collation semantics. */
(define tsql_compile_checksum (lambda (infos) (begin
	(if (empty_list? infos) (error "CHECKSUM requires at least one expression") true)
	(define parts (map infos (lambda (info) (begin
		(define type (sql_info_type info))
		(define value (sql_info_formula info))
		/* BIT columns use the common Boolean declaration after binding, while
		explicit CAST keeps the frontend BIT name. Both choose the same kernel. */
		(if (has? '("NULL" "BIT" "BOOL" "BOOLEAN" "TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT") type)
			(list tsql_checksum_integer value (equal? type "BIGINT") (has? '("BIT" "BOOL" "BOOLEAN") type))
			(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "SYSNAME") type) (begin
				(define national (has? '("NCHAR" "NVARCHAR" "SYSNAME") type))
				(define collation (toLower (sql_collation_name info)))
				(define cp1 (equal? collation "sql_latin1_general_cp1_ci_as"))
				(if (or (equal? collation "latin1_general_100_bin2") (and cp1 (not national))) true
					(error "CHECKSUM has no verified weight kernel for declared collation/type: " collation "/" type))
				(list tsql_checksum_text value cp1 national))
			(error "CHECKSUM has no verified kernel for declared type: " type))))))
	(sql_info (list tsql_checksum_combine (cons 'list parts)) "INT" nil))))
