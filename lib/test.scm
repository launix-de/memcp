/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

(begin /* own enclosure */
	(print "performing unit tests ...")

	(set teststat (newsession))
	(teststat "count" 0)
	(teststat "success" 0)
	(define assert (lambda (val1 val2 errormsg) (begin
		(teststat "count" (+ (teststat "count") 1))
		(if (equal? val1 val2) (teststat "success" (+ (teststat "success") 1)) (print "failed test "(teststat "count")": " errormsg))
	)))

	/* equal? */
	(assert (equal? "a" "a") true "equality check")
	(assert (equal? "a" "b") false "inequality check")
	/* equal? cross-type coverage (compare.go:Equal) */
	(assert (equal? nil nil) true "equal? nil nil")
	(assert (equal? nil 0) true "equal? nil vs 0 (falsy)")
	(assert (equal? nil 1) false "equal? nil vs 1 (truthy)")
	(assert (equal? 0 nil) true "equal? 0 vs nil (falsy)")
	(assert (equal? 1 nil) false "equal? 1 vs nil (truthy)")
	(assert (equal? nil false) true "equal? nil vs false")
	(assert (equal? nil "") true "equal? nil vs empty string")
	(assert (equal? true true) true "equal? bool same true")
	(assert (equal? false false) true "equal? bool same false")
	(assert (equal? true false) false "equal? bool different")
	(assert (equal? 3.14 3.14) true "equal? float same")
	(assert (equal? 3.14 2.71) false "equal? float different")
	(assert (equal? 3 3.0) true "equal? int vs float")
	(assert (equal? 3.0 3) true "equal? float vs int")
	(assert (equal? 42 "42") true "equal? int vs string")
	(assert (equal? "42" 42) true "equal? string vs int")
	(assert (equal? 3.14 "3.14") true "equal? float vs string")
	(assert (equal? "3.14" 3.14) true "equal? string vs float")
	(assert (equal? 1 true) true "equal? int 1 vs true")
	(assert (equal? 0 false) true "equal? int 0 vs false")
	(assert (equal? 1.0 true) true "equal? float 1.0 vs true")
	(assert (equal? "true" true) true "equal? string vs bool true")
	(assert (equal? '(1 2) '(1 2 3)) false "equal? unequal length lists")
	(assert (equal? '(1 2) '(1 3)) false "equal? lists different elements")
	(assert (equal? '() false) true "equal? empty list vs false")
	(assert (equal? + +) true "equal? same native function")
	/* equal?? (EqualSQL) cross-type coverage (compare.go:EqualSQL) */
	(assert (nil? (equal?? nil nil)) true "equal?? nil nil returns nil")
	(assert (nil? (equal?? nil 42)) true "equal?? nil int returns nil")
	(assert (nil? (equal?? 42 nil)) true "equal?? int nil returns nil")
	(assert (equal?? 1 1.0) true "equal?? int vs float")
	(assert (equal?? 1.0 1) true "equal?? float vs int")
	(assert (equal?? "42" 42) true "equal?? string vs int")
	(assert (equal?? 42 "42") true "equal?? int vs string")
	(assert (equal?? 3.14 "3.14") true "equal?? float vs string")
	(assert (equal?? "3.14" 3.14) true "equal?? string vs float")
	(assert (equal?? "true" true) true "equal?? string vs bool")
	(assert (equal?? true true) true "equal?? bool same")
	(assert (equal?? + +) true "equal?? same func")
	(assert (equal?? + -) false "equal?? different func")
	/* Less cross-type coverage (compare.go:Less) */
	(assert (< nil nil) false "less nil nil")
	(assert (< nil 5) true "less nil vs value")
	(assert (< 5 nil) false "less value vs nil")
	(assert (< 1.5 2.5) true "less float vs float")
	(assert (< 2.5 1.5) false "less float vs float reverse")
	(assert (< "apple" "banana") true "less string vs string")
	(assert (< "banana" "apple") false "less string vs string reverse")
	(assert (< false true) true "less bool false < true")
	(assert (< "1" 2) true "less string vs int coerced")

	/* strlike */
	(assert (strlike "a" "a") true "strlike simple")
	(assert (strlike "a" "_") true "strlike single")
	(assert (strlike "a" "_5") false "strlike overlap")
	(assert (strlike "asdf" "asdf") true "strlike complex")
	(assert (strlike "asdf" "as%") true "strlike prefix")
	(assert (strlike "asdf" "%df") true "strlike postfix")
	(assert (strlike "asdf" "a%f") true "strlike infix")
	(assert (strlike "acdf" "asdf") false "!strlike complex")
	(assert (strlike "abdf" "as%") false "!strlike prefix")
	(assert (strlike "asdfm" "%df") false "!strlike postfix")
	(assert (strlike "masdf" "a%f") false "!strlike infix")
	(assert (strlike "asd whatever mif" "a%ever%f") true "two infix")

	/* match */
	(assert (match '(1 2 3 5 6) (merge '(a b) rest) (concat "a=" a ", b=" b ", rest=" rest)) "a=1, b=2, rest=(3 5 6)" "match merge")

	/* Lists */
	/* count / nth / append / append_unique */
	(assert (equal? (count '(1 2 3)) 3) true "count on list")
	(assert (equal? (nth '(10 20 30) 1) 20) true "nth returns element")
	(assert (equal? (append '(1 2) 3 4) '(1 2 3 4)) true "append extends list")
	(assert (equal? (append_unique '(1 2 2) 2 3) '(1 2 2 3)) true "append_unique keeps first duplicates only")

	/* cons / car / cdr */
	(assert (equal? (cons 1 '(2 3)) '(1 2 3)) true "cons builds list")
	(assert (equal? (car '(9 8 7)) 9) true "car head")
	(assert (equal? (cdr '(9 8 7)) '(8 7)) true "cdr tail")

	/* zip / merge / merge_unique */
	(assert (equal? (zip (list (list 1 2) (list 3 4))) (list (list 1 3) (list 2 4))) true "zip list of lists")
	(assert (equal? (merge (list (list 1 2) (list 3))) '(1 2 3)) true "merge flattens")
	(assert (equal? (merge_unique (list (list 1 2) (list 2 3))) '(1 2 3)) true "merge_unique removes duplicates")

	/* has? / filter / map / mapIndex / reduce */
	(assert (has? '("a" "b" "c") "b") true "has? finds element")
	(assert (equal? (filter '(1 2 3 4) (lambda (x) (> x 2))) '(3 4)) true "filter keeps >2")
	(assert (equal? (map '(1 2 3) (lambda (x) (+ x x))) '(2 4 6)) true "map doubles")
	(assert (equal? (mapIndex '(10 20) (lambda (i v) (string i))) '("0" "1")) true "mapIndex uses index (stringified)")
	(assert (equal? (reduce '(1 2 3 4) (lambda (acc x) (+ acc x)) 0) 10) true "reduce sums")

	/* list? / contains? */
	(assert (list? '(1 2)) true "list? true on list")
	(assert (list? "x") false "list? false on string")
	(assert (contains? '("a" "b") "b") true "contains? present")
	(assert (contains? '("a" "b") "c") false "contains? absent")

	/* queryplan.scm pattern compatibility (with SourceInfo wrapping) */
	(print "testing queryplan pattern matching ...")
	(define tblx "t")
	(define expr_gc (list 'get_column "t" false "id" false))
	(assert (match expr_gc '((symbol get_column) (eval tblx) _ col _) col "no") "id" "match get_column for alias t -> id")
	(define expr_gc_src (source "unit" 1 1 expr_gc))
	(assert (match expr_gc_src '((symbol get_column) (eval tblx) _ col _) col "no") "id" "match get_column with SourceInfo wrapper")

	/* nil tblvar */
	(define expr_gc_nil (list 'get_column nil false "foo" false))
	(assert (match expr_gc_nil '((symbol get_column) nil _ col _) col "no") "foo" "match get_column with nil tblvar -> foo")

	/* ORDER mapping: o = ((get_column t.col) dir) -> extract col */
	(define order1 (list (list 'get_column "t" false "id" false) true))
	(assert (equal? (match order1 '(((symbol get_column) (eval tblx) _ col _) dir) (list col) '()) '("id")) true "match order key extraction")

	/* aggregate head detection */
	(define expr_agg (list 'aggregate 1 '+ 0))
	(assert (match expr_agg (cons (symbol aggregate) args) args "no") '(1 '+ 0) "match aggregate captures args")

	/* star expansion head (tbl.*) */
	(define expr_star (list 'get_column "t" true "*" false))
	(assert (match expr_star '((symbol get_column) (eval tblx) ignorecase "*" _) "ok" "no") "ok" "match tbl.* with case-insensitive flag")

	/* match/matchConcat patterns (scm/match.go coverage) */
	(print "testing match/matchConcat patterns ...")

	/* matchConcat: prefix + variable */
	(assert (match "hello_world" (concat "hello_" rest) rest "no") "world" "matchConcat prefix+var")
	/* matchConcat: variable + suffix */
	(assert (match "foo_bar" (concat prefix "_bar") prefix "no") "foo" "matchConcat var+suffix")
	/* matchConcat: prefix + middle + suffix */
	(assert (match "pre_mid_suf" (concat "pre_" middle "_suf") middle "no") "mid" "matchConcat 3-part split")
	/* matchConcat: var + delim + var (infix) */
	(assert (equal? (match "key.value" (concat mc_left "." mc_right) (concat mc_left ":" mc_right) "no") "key:value") true "matchConcat infix split")
	/* matchConcat: single var captures all */
	(assert (match "everything" (concat mc_x) mc_x "no") "everything" "matchConcat single var")
	/* matchConcat: prefix mismatch */
	(assert (match "xyz" (concat "abc" mc_rest) mc_rest "no") "no" "matchConcat prefix mismatch")
	/* matchConcat: suffix mismatch */
	(assert (match "xyz" (concat mc_pfx "abc") mc_pfx "no") "no" "matchConcat suffix mismatch")
	/* matchConcat: delimiter not found */
	(assert (match "nocolon" (concat mc_l ":" mc_r) "yes" "no") "no" "matchConcat delim not found")
	/* matchConcat: non-string input */
	(assert (match 42 (concat mc_x) mc_x "no") "no" "matchConcat rejects non-string")

	/* match: type-checking patterns */
	(assert (match "hello" (string? mc_s) mc_s "no") "hello" "match string? captures")
	(assert (match 42 (string? mc_s) mc_s "no") "no" "match string? rejects number")
	(assert (match 3.14 (number? mc_n) mc_n "no") 3.14 "match number? captures")
	(assert (match "abc" (number? mc_n) mc_n "no") "no" "match number? rejects string")
	(assert (match (list 10 20 30) (list? mc_l) (count mc_l) "no") 3 "match list? captures list")
	(assert (match "abc" (list? mc_l) mc_l "no") "no" "match list? rejects string")

	/* match: ignorecase */
	(assert (match "Hello" (ignorecase "hello") "yes" "no") "yes" "match ignorecase match")
	(assert (match "Hello" (ignorecase "world") "yes" "no") "no" "match ignorecase mismatch")

	/* match: regex */
	(assert (match "v=5" (regex "^v=(.*)" _ mc_v) mc_v "no") "5" "match regex single capture")
	(assert (match "abc" (regex "^v=(.*)" _ mc_v) mc_v "no") "no" "match regex no match")
	(assert (equal? (match "key=val" (regex "^(.*)=(.*)$" _ mc_k mc_v) (concat mc_k ":" mc_v) "no") "key:val") true "match regex multi-capture")

	/* match: list destructuring */
	(assert (match (list 10 20 30) (list mc_a mc_b mc_c) (+ mc_a mc_b mc_c) "no") 60 "match list destructure")
	(assert (match (list 10 20) (list mc_a mc_b mc_c) "yes" "no") "no" "match list length mismatch")
	(assert (match "notalist" (list mc_a) "yes" "no") "no" "match list rejects non-list")

	/* match: quote/symbol literal */
	(assert (match 'foo (quote foo) "yes" "no") "yes" "match quote literal match")
	(assert (match 'foo (quote bar) "yes" "no") "no" "match quote literal mismatch")
	(assert (match 'bar (symbol bar) "yes" "no") "yes" "match symbol literal match")
	(assert (match 'bar (symbol baz) "yes" "no") "no" "match symbol literal mismatch")
	(assert (match "str" (quote foo) "yes" "no") "no" "match quote rejects non-symbol")
	(assert (match "str" (symbol foo) "yes" "no") "no" "match symbol rejects non-symbol")

	/* match: cons */
	(assert (match (list 10 20 30) (cons mc_h mc_t) mc_h "no") 10 "match cons head")
	(assert (match (list 10 20 30) (cons mc_h mc_t) (count mc_t) "no") 2 "match cons tail length")
	(assert (match (list) (cons mc_h mc_t) "yes" "no") "no" "match cons empty list")

	/* match: literal value matching */
	(assert (match 42 42 "yes" "no") "yes" "match literal int")
	(assert (match "abc" "abc" "yes" "no") "yes" "match literal string")
	(assert (match nil nil "yes" "no") "yes" "match nil symbol")
	(assert (match true true "yes" "no") "yes" "match true symbol")
	(assert (match false false "yes" "no") "yes" "match false symbol")

	/* Tests for scm package */
	/* Tests for alu.go */

	/* Test for number? */
	(assert (number? 42) true "42 should be a number")
	(assert (number? "42") false "\"42\" should not be a number")
	(assert (number? `symbol) false "'symbol' should not be a number")

	/* Test for int? (requires int64-producing builtin like size/now) */
	(assert (int? (size "abc")) true "size returns an int")
	(assert (int? 42) false "literal 42 is not an int (parsed as number)")

	/* Test for + */
	(assert (+ 1 2) 3 "1 + 2 should be 3")
	(assert (+ 1 2 3) 6 "1 + 2 + 3 should be 6")
	(assert (+ 1 2 3.5) 6.5 "int + float promotes to float")
	(assert (nil? (+ 1 2 nil)) true "+ with nil returns nil")
	(assert (+ 0 0) 0 "0 + 0 should be 0")

	/* Test for - */
	(assert (- 5 3) 2 "5 - 3 should be 2")
	(assert (- 5 3 1) 1 "5 - 3 - 1 should be 1")
	(assert (equal? (- 5 3.5) 1.5) true "int - float promotes to float")
	(assert (nil? (- 5 nil)) true "- with nil returns nil")

	/* Test for * */
	(assert (* 2 3) 6 "2 * 3 should be 6")
	(assert (* 2 3 4) 24 "2 * 3 * 4 should be 24")
	(assert (equal? (* 2 3.5) 7.0) true "int * float promotes to float")
	/* regression: first arg float must not be truncated to int */
	(assert (equal? (* 2.5 2) 5.0) true "float * int (float first) -> 5.0")
	(assert (equal? (* 2.5 2 2) 10.0) true "float-first multiply across ints -> 10.0")
	/* additional: begin with integers, only integers -> stays int */
	(assert (int? (* 2 3 4 5)) true "int*int*int*int stays int type")
	(assert (* 1 2 3 4 5) 120 "many ints multiply correctly")
	/* additional: mixed starting with integers -> promote when float appears */
	(assert (equal? (* 2 3 0.5) 3.0) true "int*int*float -> 3.0 (float)")
	(assert (equal? (* 2 0.5 3) 3.0) true "int*float*int -> 3.0 (float)")
	(assert (equal? (* 2 3 4.0) 24.0) true "int*int*float -> 24.0 (float)")
	/* nil propagation */
	(assert (nil? (* 2 nil)) true "* with nil at end returns nil")
	(assert (nil? (* nil 2 3)) true "* with nil at beginning returns nil")
	(assert (nil? (* 2 3 nil)) true "* with nil at end (3-args) returns nil")
	(assert (nil? (* 2 3.5 nil)) true "* with int->float change then nil returns nil")

	/* Test for / */
	(assert (/ 6 2) 3 "6 / 2 should be 3")
	(assert (/ 12 2 2) 3 "12 / 2 / 2 should be 3")
	(assert (equal? (/ 7 2) 3.5) true "int / int yields float when needed")
	(assert (nil? (/ 5 nil)) true "/ with nil returns nil")

	/* Test for < */
	(assert (< 1 2) true "1 < 2 should be true")
	(assert (< 2 1) false "2 < 1 should be false")

	/* Test for <= */
	(assert (<= 1 2) true "1 <= 2 should be true")
	(assert (<= 2 2) true "2 <= 2 should be true")
	(assert (<= 3 2) false "3 <= 2 should be false")

	/* Test for > */
	(assert (> 2 1) true "2 > 1 should be true")
	(assert (> 1 2) false "1 > 2 should be false")

	/* Test for >= */
	(assert (>= 2 1) true "2 >= 1 should be true")
	(assert (>= 2 2) true "2 >= 2 should be true")
	(assert (>= 1 2) false "1 >= 2 should be false")

	/* Test for equal? */
	(assert (equal? 2 2) true "2 equal? 2 should be true")
	(assert (equal? 2 3) false "2 equal? 3 should be false")

	/* Test for equal?? */
	(assert (equal?? 42 42) true "42 equal?? 42 should be true")
	(assert (equal?? 42 43) false "42 equal?? 43 should be false")
	(assert (equal?? "hello" "HELLO") true "\"hello\" equal?? \"HELLO\" should be true")
	(assert (equal?? "hello" "world") false "\"hello\" equal?? \"world\" should be false")
	(assert (equal?? true true) true "true equal?? true should be true")
	(assert (equal?? true false) false "true equal?? false should be false")

	/* Test for ! */
	(assert (! true) false "not true should be false")
	(assert (! false) true "not false should be true")

	/* Test for not */
	(assert (not true) false "not true should be false")
	(assert (not false) true "not false should be true")

	/* Truthiness of quoted symbols in special forms (no type-enforced not) */
	(assert (if false 1 2) 2 "false treated as falsy in if")
	(assert (if nil 1 2) 2 "'nil treated as falsy in if")

	/* Test for nil? */
	(assert (nil? nil) true "nil? of nil should be true")
	(assert (nil? 0) false "nil? of 0 should be false")

	/* Test for min */
	(assert (equal? (min 1 2 3) 1) true "min of 1, 2, 3 should be 1")
	(assert (equal? (min 5 3 1) 1) true "min of 5, 3, 1 should be 1")

	/* Test for max */
	(assert (equal? (max 1 2 3) 3) true "max of 1, 2, 3 should be 3")
	(assert (equal? (max 5 3 1) 5) true "max of 5, 3, 1 should be 5")

	/* Test for floor */
	(assert (equal? (floor 3.7) 3) true "floor of 3.7 should be 3")
	(assert (equal? (floor 3.2) 3) true "floor of 3.2 should be 3")

	/* Test for ceil */
	(assert (equal? (ceil 3.7) 4) true "ceil of 3.7 should be 4")
	(assert (equal? (ceil 3.2) 4) true "ceil of 3.2 should be 4")

	/* Test for round */
	(assert (equal? (round 3.7) 4) true "round of 3.7 should be 4")
	(assert (equal? (round 3.2) 3) true "round of 3.2 should be 3")

	/* Dictionaries / Assoc lists (with FastDict auto-upgrade) */
	(print "testing dictionaries ...")

	/* small assoc basic ops */
	(define d '())
	(set d (set_assoc d "a" 1))
	(set d (set_assoc d "b" 2))
	(assert (has_assoc? d "a") true "assoc has a")
	(assert (has_assoc? d "x") false "assoc no x")
	(assert (equal? (reduce_assoc d (lambda (acc k v) (+ acc v)) 0) 3) true "reduce sum small")
	(define la (list "a" 1 "b" 2))
	(assert (equal? (la "a") 1) true "call assoc as func(list)")
	(assert (equal? (d "b") 2) true "call assoc as func(dict)")

	/* overwrite should not grow list length */
	(set d (set_assoc d "a" 11))
	(assert (equal? (d "a") 11) true "overwrite list assoc value")
	(assert (equal? (count d) 4) true "list length unchanged on overwrite")

	/* merge + map + filter */
	(define d1 (list "x" 10 "y" 20))
	(define d2 (list "y" 5  "z" 7))
	(define dm (merge_assoc d1 d2))
	(assert (equal? (dm "y") 5) true "merge overwrites second wins")
	(define dmap (map_assoc dm (lambda (k v) (+ v 1))))
	(assert (equal? (dmap "z") 8) true "map increments values")
	(define df (filter_assoc dmap (lambda (k v) (> v 10))))
	(assert (has_assoc? df "x") true "filter keeps x")
	(assert (has_assoc? df "z") false "filter drops z")

	/* big assoc to test auto switch to FastDict */
	(define big (reduce (produceN 2000) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? (reduce_assoc big (lambda (acc k v) (+ acc v)) 0) 1999000) true "reduce sum big (0..1999)")

	/* FastDict getter correctness on many keys */
	(assert (has_assoc? big "k0") true "fastdict has k0")
	(assert (has_assoc? big "k1234") true "fastdict has k1234")
	(assert (equal? (big "k1999") 1999) true "fastdict getter last key")
	(assert (equal? (big "k1") 1) true "fastdict getter small key")

	/* Overwrite existing key in FastDict and get updated value */
	(set big (set_assoc big "k100" 555))
	(assert (equal? (big "k100") 555) true "fastdict overwrite value")

	/* extract_assoc produces all keys (sanity: count) */
	(define countkeys (reduce (extract_assoc big (lambda (k v) 1)) (lambda (a b) (+ a b)) 0))
	(assert (equal? countkeys 2000) true "fastdict extract returns all keys (2000)")

	/* map_assoc and filter_assoc over FastDict */
	(define biginc (map_assoc big (lambda (k v) (+ v 1))))
	(assert (equal? (biginc "k0") 1) true "map fastdict increments")
	(define bigf (filter_assoc biginc (lambda (k v) (> v 1000))))
	(assert (has_assoc? bigf "k1500") true "filter keeps large values")
	(assert (has_assoc? bigf "k1") false "filter drops small values")

	/* set_assoc immutability: original must not be modified */
	(define orig '("a" 1 "b" 2))
	(define modified (set_assoc orig "a" 99))
	(assert (orig "a") 1 "set_assoc immutable: original unchanged on slice path")
	(assert (modified "a") 99 "set_assoc immutable: modified has new value")
	(define orig_fd (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(define modified_fd (set_assoc orig_fd "k5" 999))
	(assert (orig_fd "k5") 5 "set_assoc immutable: original FastDict unchanged")
	(assert (modified_fd "k5") 999 "set_assoc immutable: modified FastDict has new value")

	/* Strings / JSON */
	(print "testing strings ...")
	(assert (equal? (strlen "abc") 3) true "strlen counts bytes")
	(assert (equal? (replace "a-b-c" "-" ":") "a:b:c") true "replace replaces all")
	(assert (equal? (split "a,b,c" ",") '("a" "b" "c")) true "split splits on sep")
	(assert (strlike (htmlentities "<tag>") "&lt;tag&gt;") true "htmlentities encodes angle brackets")
	(assert (equal? (urldecode (urlencode "a b")) "a b") true "url roundtrip")
	(assert (strlike (json_encode_assoc (list "x" 1)) "%\"x\":1%") true "json_encode_assoc contains key and value")

	/* string? / substr / simplify / case conversion */
	(assert (string? "foo") true "string? on string")
	(assert (string? 123) false "string? on number")
	(assert (equal? (substr "hello" 1 3) "ell") true "substr with length")
	(assert (equal? (substr "hello" 1) "ello") true "substr to end")
	(assert (equal? (simplify "3.14") 3.14) true "simplify numeric string")
	(assert (equal? (simplify "abc") "abc") true "simplify keeps non-numeric")
	(assert (equal? (toLower "ÄBCd") "äbcd") true "toLower handles letters")
	(assert (equal? (toUpper "ÄBCd") "ÄBCD") true "toUpper handles letters")

	/* collate comparator */
	(define less_bin (collate "bin"))
	(assert (less_bin "a" "b") true "bin collation: a<b")
	(assert ((collate "bin" true) "a" "b") false "bin reverse: a<b -> false")
	/* general_ci heuristic places ASCII before non-ASCII class like leading 'aa' */
	(assert ((collate "general_ci") "z" "aa") true "general_ci: ASCII first")

	/* SQL unescape */
	(assert (equal? (bin2hex (sql_unescape "a\\nb")) "610a62") true "sql_unescape newline")
	(assert (equal? (sql_unescape "a\\'b") "a'b") true "sql_unescape quote")
	(assert (equal? (bin2hex (sql_unescape "a\\0b")) "610062") true "sql_unescape NUL byte present")

	/* json_encode vs json_encode_assoc semantics (master-compatible) */
	(assert (equal? (json_encode '(1 2 3)) "[1,2,3]") true "json_encode lists as arrays")
	(assert (strlike (json_encode_assoc (list "x" 1 "y" 2)) "%\"x\":1%") true "json_encode_assoc has x:1")
	(assert (strlike (json_encode_assoc (list "x" 1 "y" 2)) "%\"y\":2%") true "json_encode_assoc has y:2")

	/* symbol encoding must preserve type marker */
	(assert (equal? (json_encode (symbol "alpha")) "{\"symbol\":\"alpha\"}") true "json_encode(symbol) -> {symbol:\"alpha\"}")
	(assert (strlike (json_encode (lambda (a b) (+ a b))) "%\"symbol\":\"lambda\"%") true "json_encode(lambda ...) contains lambda symbol header")
	(assert (strlike (json_encode_assoc (list "s" (symbol "S"))) "%\"s\":{\"symbol\":\"S\"}%") true "json_encode_assoc preserves symbol values")

	/* json_decode builds assoc list with functional access */
	(assert (equal? ((json_decode "{\"a\":2}") "a") 2) true "json_decode object -> assoc callable by key")
	(assert (equal? (nth (json_decode "[1,2,3]") 1) 2) true "json_decode array -> list indexable with nth")

	/* Optimizer safeguards for eval/import and aliasing in begin */
	/* Case: preserve old binding while overloading after an eval barrier */
	(define http_test_begin (begin
		(define http_handler (lambda (req res) 1))
		(define old_handler http_handler)
		(eval '(print "optimizer eval barrier test"))
		(define http_handler (lambda (req res) (+ (old_handler req res) 1)))
		(http_handler 0 0)
	))
	(assert http_test_begin 2 "handler layering with eval barrier")

	/* Case: forbid inlining an alias to a symbol redefined later in the same begin */
	(define alias_cycle_guard (begin
		(define a 10)
		(define old_a a)
		(define a (+ old_a 5))
		(equal? a 15)
	))
	(assert alias_cycle_guard true "no self-referential aliasing inlining")

	/* hex/bin encode-decode */
	(assert (equal? (bin2hex "AB") "4142") true "bin2hex encodes bytes to hex")
	(assert (equal? (hex2bin "414243") "ABC") true "hex2bin decodes hex to bytes")
	(assert (equal? (hex2bin (bin2hex "Hello")) "Hello") true "hex/bin roundtrip")

	/* base64 encode/decode */
	(assert (equal? (base64_encode "foo") "Zm9v") true "base64_encode encodes correctly")
	(assert (equal? (base64_decode "Zm9v") "foo") true "base64_decode decodes correctly")
	(assert (equal? (base64_decode (base64_encode "Hello, world!")) "Hello, world!") true "base64 roundtrip")

	/* randomBytes properties */
	(assert (equal? (strlen (randomBytes 0)) 0) true "randomBytes 0 length")
	(assert (equal? (strlen (randomBytes 16)) 16) true "randomBytes length 16")
	/* two independently generated strings should differ (overwhelmingly likely) */
	(assert (equal? (randomBytes 32) (randomBytes 32)) false "two random strings must be unequal")

	/* Streams */
	(print "testing streams ...")
	(assert (equal? (concat (streamString "abc")) "abc") true "streamString -> concat")
	(assert (equal? (concat (zcat (gzip (streamString "hello")))) "hello") true "gzip+zcat roundtrip")
	(assert (equal? (concat (xzcat (xz (streamString "xyz")))) "xyz") true "xz+xzcat roundtrip")

	/* Eval and Parser (Any-wrapped) semantics */
	(print "testing eval and parser semantics ...")
	(assert (eval '(+ 2 3)) 5 "eval executes quoted code")
	/* eval of computed code (list-built call) */
	(assert (equal? (eval (list + 1 2 3)) 6) true "eval applies computed list")
	/* eval on parsed code */
	(assert (equal? (eval (scheme "(+ 2 5)" "eval1.scm")) 7) true "eval scheme AST")
	/* serialize -> scheme -> eval roundtrip */
	(assert (equal? (eval (scheme (serialize (scheme "(+ 3 4)" "ser.scm")))) 7) true "serialize/scheme roundtrip")
	/* quote returns literal data */
	(assert (equal? (quote a) 'a) true "quote symbol")
	(assert (equal? '(1 2) (list 1 2)) true "literal list equals built list")
	/* if multi-branch and else */
	(assert (equal? (if false 1 true 2 3) 2) true "if selects first true branch")
	(assert (equal? (if false 1 false 2 3) 3) true "if else branch")
	/* and/or short-circuit evaluation with side-effect check via session */
	(define sc (newsession))
	(sc "a" 0)
	(and false (sc "a" 1))
	(assert (equal? (sc "a") 0) true "and short-circuits second arg")
	(or true (sc "a" 1))
	(assert (equal? (sc "a") 0) true "or short-circuits second arg")
	/* coalesce and coalesceNil */
	(assert (equal? (coalesce nil "" 0 '()) '()) true "coalesce takes last even if falsy")
	(assert (equal? (coalesceNil nil "" 0) "") true "coalesceNil returns first non-nil")
	/* outer evaluates expression in outer environment */
	(define ox_eval 10)
	(assert (equal? (begin (define ox_eval 20) (eval (list 'outer 'ox_eval))) 10) true "outer reads outer var")
	/* begin creates new scope; final value is last expression */
	(assert (equal? (begin (define p 1) (define q (+ p 1)) q) 2) true "begin uses new scope and returns last")
	/* !begin executes in parent env */
	(define xb 10)
	(define rb (begin (define xb 1) (!begin (define xb 2)) xb))
	(assert (equal? rb 2) true "!begin does not create new scope; updates same begin env")
	(assert (equal? xb 10) true "outer env unchanged by !begin inside begin")
	/* undefined symbol lookup yields nil */
	(assert (nil? unknown_var_12345) true "reading unknown symbol yields nil")
	/* simple regex parser returns captured value */
	(define p_regex (parser (define x (regex "[A-Za-z]+" true)) x))
	(assert (equal? (p_regex "Hello") "Hello") true "parser returns regex capture")
	/* atom parser with constant generator returns constant */
	(define p_atom (parser (atom "FOO" true) "ok"))
	(assert (equal? (p_atom "FOO") "ok") true "atom parser returns constant generator")

	/* error cases intentionally omitted in unit tests to avoid compile-time constant folding side-effects */

	/* Lambda / apply_assoc */
	(print "testing lambdas and apply_assoc ...")
	(assert (equal? ((lambda (x y) (+ x y)) 2 3) 5) true "lambda call")
	(assert (equal? (apply_assoc (lambda (x y) (+ x y)) (list "x" 2 "y" 3)) 5) true "apply_assoc maps assoc args")

	/* for loop (functional) */
	(print "testing for loop ...")
	/* Count to 10 with single state var */
	(assert (equal? (for '(0) (lambda (x) (< x 10)) (lambda (x) (list (+ x 1)))) '(10)) true "for increments to 10")
	/* Sum 0..9 with (x sum) state */
	(define for_res (for '(0 0) (lambda (x sum) (< x 10)) (lambda (x sum) (list (+ x 1) (+ sum x)))))
	(assert (equal? (nth for_res 0) 10) true "for final x=10")
	(assert (equal? (nth for_res 1) 45) true "for sum 0..9=45")
	/* condition false initially -> returns init unchanged */
	(assert (equal? (for '(5) (lambda (x) (< x 0)) (lambda (x) (list (+ x 1)))) '(5)) true "for returns init when condition false")

	/* Assoc merge with custom merge function */
	(print "testing assoc merge ...")
	(define m1 (list "x" 1))
	(set m1 (set_assoc m1 "x" 2 (lambda (old new) (+ old new))))
	(assert (equal? (m1 "x") 3) true "set_assoc merge function")
	(define m2 (merge_assoc (list "a" 1) (list "a" 5) (lambda (old new) (+ old new))))
	(assert (equal? (m2 "a") 6) true "merge_assoc merge function")

	/* FastDict vs assoc equality */
	(print "testing dict equality ...")
	(define ld '("k0" 0 "k1" 1 "k2" 2 "k3" 3 "k4" 4 "k5" 5))
	(define dd (reduce (produceN 6) (lambda (acc i) (set_assoc acc (concat "k" i) i)) ()))
	(assert (equal? ld dd) true "list vs dict equal content")

	/* FastDict unwrapping in match patterns */
	(print "testing FastDict match unwrapping ...")
	(define fd2 (reduce (produceN 10) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	/* list? should unwrap FastDict to a flat pair list; size = 20 */
	(assert (match fd2 (list? xs) (reduce xs (lambda (acc _i) (+ acc 1)) 0) "no") 20 "FastDict list? unwraps to 20 elements")
	/* cons over FastDict-as-list should take first element "k0" */
	(assert (match fd2 (cons first rest) first "no") "k0" "FastDict cons head extracts first key")
	/* verify next key via cons on rest -> "k1" */
	(assert (match fd2 (cons _ rest) (match rest (cons k1 _) k1 "no") "no") "k1" "FastDict cons rest begins with k1")

	/* Optimizer semantics (constant folding, shadowing, set behavior) */
	(print "testing optimizer semantics ...")

	/* Constant folding candidates (boolean/arithmetic inside lambdas) */
	(assert ((lambda () (and (and true) true))) true "const and true -> true")
	(assert ((lambda () (and true false))) false "const and false -> false")
	(assert ((lambda () (+ 1 2 3))) 6 "const arith folds to 6")
	(assert ((lambda () (if (and true (equal? 1 1)) 1 2))) true "const condition -> true")

	/* Setting and calling lambdas via set */
	(assert (begin (set fn (lambda (x) (+ x 1))) (fn 4)) 5 "set lambda then call")
	(assert (begin (set add2 (lambda (a b) (+ a b))) (add2 2 5)) 7 "set 2-param lambda then call")

	/* Optimize should fold constants */
	(assert (optimize '(+ 1 2)) 3 "optimize folds +")
	(assert (optimize '('concat "a" 2)) "a2" "optimize folds string concat")
	(assert (optimize '('and true '(equal? 2 2))) true "optimize folds and/equal")
	(assert (optimize '('begin '('define 'x 4) '(+ 'x 1))) 5 "optimize inlines define use-once")
	(assert (optimize '('and '('and '('and '(> 'LINEITEM.L_QUANTITY 10)) true) '('equal? 1 '('outer 1)))) '(> 'LINEITEM.L_QUANTITY 10) "SQL filter optimization")

	/* Flatten nested + and * (associative operators) */
	(assert (optimize '('+ 1 '('+ 2 3))) 6 "optimize flattens nested + constants")
	(assert (optimize '('* 2 '('* 3 4))) 24 "optimize flattens nested * constants")
	(assert (optimize '('+ 1 '('+ 2 '('+ 3 4)))) 10 "optimize flattens deeply nested +")
	(assert (optimize '('* 1 '('* 2 '('* 3 4)))) 24 "optimize flattens deeply nested *")
	/* - and / must NOT be flattened (non-associative) */
	(assert (- 10 (- 3 1)) 8 "subtraction not flattened: 10-(3-1)=8")
	(assert (/ 100 (/ 10 2)) 20 "division not flattened: 100/(10/2)=20")

	/* Flatten nested !begin blocks */
	(assert (begin (define xb1 1) (!begin (!begin (set xb1 2) (set xb1 (+ xb1 3))) (set xb1 (* xb1 10))) xb1) 50 "nested !begin flattens correctly")

	/* Lambda params overshadow outer variables */
	(define y 10)
	(assert ((lambda (y) (+ y 1)) 5) 6 "param shadows outer value")
	(assert y 10 "outer y unchanged after lambda")

	/* set should affect current scope/param, not outer */
	(define sx 1)
	(assert ((lambda (sx) (begin (set sx 3) sx)) 7) 3 "set updates local param")
	(assert sx 1 "outer sx unchanged after local set")

	/* Shadowing via inner define does not leak */
	(define dz 2)
	(define dz_inner (begin (define dz 9) dz))
	(assert dz_inner 9 "inner define returns its value")
	(assert dz 2 "outer dz unchanged after inner define")

	/* Use-once inlining safety: begin with unused define should not change result */
	(assert (begin (define tmp_unused 42) 7) 7 "unused define eliminated")

	/* !list optimization: (list ...) passed to NoEscape parameter uses stack allocation */
	(assert ((eval (optimize '('lambda '('a 'b) '(count '(list 'a 'b))))) 10 20) 2 "!list count")
	(assert ((eval (optimize '('lambda '('a 'b) '(car '(list 'a 'b))))) 5 6) 5 "!list car")
	(assert ((eval (optimize '('lambda '('a 'b 'c) '(nth '(list 'a 'b 'c) 1)))) 10 20 30) 20 "!list nth")
	(assert ((eval (optimize '('lambda '('a 'b) '(has? '(list 'a 'b) 'a)))) 5 6) true "!list has?")
	(assert ((eval (optimize '('lambda '('a 'b) '(get_assoc '(list "x" 'a "y" 'b) "x")))) 42 99) 42 "!list get_assoc")
	(assert ((eval (optimize '('lambda '('a 'b) '(cdr '(list 'a 'b))))) 5 6) '(6) "!list cdr")
	(assert ((eval (optimize '('lambda '('a 'b) '(reduce '(list 'a 'b) '('lambda '('acc 'x) '(+ 'acc 'x)) 0)))) 10 20) 30 "!list reduce")

	/* _mut optimization: freshly constructed list triggers in-place operations */
	(assert ((eval (optimize '('lambda '('a 'b 'c) '(map '(list 'a 'b 'c) '('lambda '('x) '(+ 'x 1)))))) 10 20 30) '(11 21 31) "_mut map on fresh list")
	(assert ((eval (optimize '('lambda '('a 'b 'c 'd) '(filter '(list 'a 'b 'c 'd) '('lambda '('x) '(> 'x 2)))))) 1 2 3 4) '(3 4) "_mut filter on fresh list")

	/* Declaration-driven optimizer hooks */
	/* and hook: short-circuit on constant-false, remove constant-true */
	(assert (optimize '('and '(equal? 1 1) '(equal? 1 1) '(equal? 1 2))) false "and hook: short-circuit false")
	(assert (optimize '('and '(equal? 1 1) 'x)) 'x "and hook: removes true constant")
	(assert (optimize '('and '(equal? 1 1) '(equal? 1 1))) true "and hook: all true constants fold")
	(assert (serialize (optimize '('and 'x 'y))) "(and x y)" "and hook: non-const args preserved")

	/* +/* hooks: associative flattening with symbolic args */
	(assert (serialize (optimize '('+ 'a '('+ 'b 'c)))) "(+ a b c)" "+ hook: flattens nested +")
	(assert (serialize (optimize '('* 'a '('* 'b 'c)))) "(* a b c)" "* hook: flattens nested *")
	(assert (serialize (optimize '('+ 'a '('+ 'b '('+ 'c 'd))))) "(+ a b c d)" "+ hook: deeply nested flatten")

	/* _mut hook via FirstParameterMutable: set_assoc on filter result */
	(assert (serialize (optimize '('set_assoc '('filter '('list) '('lambda '('x) true)) "k" "v"))) "(set_assoc_mut (filter '() (lambda (x) true 1)) \"k\" \"v\")" "_mut hook: set_assoc -> set_assoc_mut")
	/* _mut on append with fresh list arg */
	(assert ((eval (optimize '('lambda '('a 'b) '(append '(list 'a) 'b)))) 10 20) '(10 20) "_mut append on fresh list")
	/* scan callback ownership: reduce accumulator enables _mut inside reduce body */
	(assert (serialize (optimize '('scan "db" "tbl" '("x") '('lambda '('x) true) '("x") '('lambda '('x) 'x) '('lambda '('acc 'row) '(set_assoc 'acc 'row true)) '(list) nil false))) "(scan \"db\" \"tbl\" (\"x\") (lambda (x) true 1) (\"x\") (lambda (x) (var 0) 1) (lambda (acc row) (set_assoc_mut (var 0) (var 1) true) 2) (list) nil false)" "scan hook: reduce acc enables set_assoc_mut")

	/* REGEXP_REPLACE precompilation: constant pattern gets precompiled */
	(assert ((eval (optimize '('lambda '('s) '(regexp_replace 's "[^0-9]" "")))) "abc123def") "123" "regexp_replace precompilation works")
	(assert ((eval (optimize '('lambda '('s) '(regexp_replace 's "^0+" "")))) "000042") "42" "regexp_replace precompiled strips leading zeros")

	/* Numbered parameter semantics (NthLocalVar / NumVars) */
	(print "testing numbered parameters ...")

	/* Correct case: body uses (var i), NumVars covers indices */
	(define lam_ok '('lambda '('a 'b) '('+ '('var 0) '('var 1)) 2))
	(assert ((eval (optimize lam_ok)) 2 3) 5 "numbered params add correctly (NumVars=2)")

	/* Broken case: body references (var 1) but NumVars too small -> must raise error */
	(define lam_bad '('lambda '('a 'b) '('+ '('var 0) '('var 1)) 1))
	(define panicked (newsession))
	(try (lambda () ((eval (optimize lam_bad)) 2 3)) (lambda (e) (panicked "panic" true)))
	(assert (panicked "panic") true "insufficient NumVars must panic (guards optimizer bug)")

	/* Optimizer regression: lambdas with eval must keep symbol-bound params (no NthLocalVar-only params). */
	(define lam_eval_scope (list
		'lambda
		(list 'session)
		(list
			'begin
			(list 'eval (list 'quote (list 'session "probe" 1)))
			(list 'session "probe")
		)
	))
	(assert (list? lam_eval_scope) true "lambda eval scope fixture must be list AST")
	(define eval_scope_state (newsession))
	(try
		(lambda () (eval_scope_state "opt" (optimize lam_eval_scope)))
		(lambda (e) (panicked "eval-opt-panic" true))
	)
	(assert (panicked "eval-opt-panic") nil "optimizer must not panic on lambda containing eval")
	(define lam_eval_scope_opt (eval_scope_state "opt"))
	(if (panicked "eval-opt-panic") true (begin
		(assert (list? lam_eval_scope_opt) true "optimizer output for lambda AST must remain list AST")
		(if (list? lam_eval_scope_opt)
			(assert (count lam_eval_scope_opt) 3 "lambdas with eval must not be auto-numbered (no NumVars append)")
			true
		)

		(define lam_eval_scope_serialized "")
		(try (lambda () (set lam_eval_scope_serialized (serialize lam_eval_scope_opt))) (lambda (e) (panicked "eval-serialize-panic" true)))
		(assert (panicked "eval-serialize-panic") nil "serialize on optimized lambda must not panic")
		(assert (match lam_eval_scope_serialized (regex "\\(var 0\\)" _) true false) false "lambda with eval must keep session symbol, not (var 0)")
	))

	/* cascade override */
	(print "testing more lambda functions ...")
	(define lam_nested1 (lambda (req res) (+ req res)))
	(define lam_nested2 (lambda (req res) (+ 1 (lam_nested1 req res))))
	(define lam_nested3 (lambda (req res) (lam_nested2 req res)))
	(define lam_nested4 lam_nested3)
	(define lam_nested3 (lambda (req res) (+ 3 (lam_nested4 req res))))
	(assert (lam_nested3 4 7) 15 "nested lambda scope calling")

	/* cascade overrides with same variable name -> value must be drawn into inner scope */
	(set lam_handler (newsession))
	(lam_handler "handler" (lambda (req res) (+ req res)))
	(lam_handler "handler" (begin (set old_handler (lam_handler "handler")) (lambda (req res) (+ 1 (old_handler req res)))))
	(assert ((lam_handler "handler") 4 7) 12 "nested lambda scope overriding")
	(lam_handler "handler" (begin (set old_handler (lam_handler "handler")) (set mid_handler (lambda (req res) (+ 1 (old_handler req res)))) mid_handler))
	(assert ((lam_handler "handler") 4 7) 13 "nested lambda scope overriding with inner variables")


	/* Mixed-type comparison: be forgiving (SQL) — no panic */
	(try (lambda () (< "x" 1)) (lambda (e) (panicked "cmp-panic" true)))
	(assert (panicked "cmp-panic") nil "mixed-type comparison should not panic")

	/* Sync / Context */
	(print "testing sync/context ...")
	/* newsession key listing */
	(define sess (newsession))
	(sess "a" 1)
	(sess "b" 2)
	(define keys (sess))
	(assert (contains? keys "a") true "session lists key a")
	(assert (contains? keys "b") true "session lists key b")
	/* context + sleep */
	(assert (context (lambda () (sleep 0.005))) true "sleep inside context")
	/* context session */
	(assert (context (lambda () (begin (define s (context "session")) (s "k" 7) (equal? (s "k") 7)))) true "context session set/get")
	/* context check inside valid context */
	(assert (context (lambda () (context "check"))) true "context check returns true in valid context")
	/* once */
	(define once_calls (newsession))
	(once_calls "n" 0)
	(define once_fn (once (lambda (x) (begin (once_calls "n" (+ (once_calls "n") 1)) (+ x 1)))))

	(assert (equal? (once_fn 2) 3) true "once first call computes")
	(assert (equal? (once_fn 99) 3) true "once second call returns cached")
	(assert (equal? (once_calls "n") 1) true "once executes only once")
	/* mutex */
	(define mtx (mutex))
	(assert (equal? (mtx (lambda () 42)) 42) true "mutex executes inner function")

	/* Scheduler */
	(print "testing scheduler ...")
	(define sched (newsession))
	(sched "done" false)
	(setTimeout (lambda () (sched "done" true)) 1)
	(context (lambda () (sleep 0.01)))
	(assert (sched "done") true "setTimeout fires callback")
	/* clearTimeout */
	(sched "done" false)
	(define tid (setTimeout (lambda () (sched "done" true)) 50))
	(clearTimeout tid)
	(context (lambda () (sleep 0.02)))
	(assert (sched "done") false "clearTimeout cancels callback")
	/* setTimeout with negative delay fires immediately */
	(sched "done" false)
	(setTimeout (lambda () (sched "done" true)) -50)
	(context (lambda () (sleep 0.01)))
	(assert (sched "done") true "setTimeout negative delay fires")
	/* clearTimeout on non-existent ID */
	(assert (clearTimeout 999999) false "clearTimeout non-existent ID returns false")

	/* Date */
	(print "testing date helpers ...")
	(assert (number? (now)) true "now returns number")
	(assert (>= (now) 1000000000) true "now >= 1e9 epoch")

	/* Vectors */
	(print "testing vectors ...")
	(assert (equal? (dot '(1 2 3) '(4 5 6)) 32) true "dot product")
	(assert (equal? (dot '(3 4) '(3 4) "COSINE") 1) true "cosine of identical vectors = 1")
	(assert (equal? (round (* 1000 (dot '(3 4) '(3 4) "EUCLIDEAN"))) 5000) true "euclidean length sqrt(sum) *1000")

	/* JIT compilation */
	(print "testing JIT compilation ...")

	/* Basic arithmetic with single parameter */
	(assert ((jit (lambda (x) (+ x 1))) 4) 5 "jit: x + 1")
	(assert ((jit (lambda (x) (- x 3))) 10) 7 "jit: x - 3")
	(assert ((jit (lambda (x) (* x 2))) 5) 10 "jit: x * 2")

	/* Two parameters */
	(assert ((jit (lambda (a b) (+ a b))) 3 4) 7 "jit: a + b")
	(assert ((jit (lambda (a b) (* a b))) 3 4) 12 "jit: a * b")
	(assert ((jit (lambda (a b) (- a b))) 10 3) 7 "jit: a - b")

	/* Nested operations */
	(assert ((jit (lambda (x) (* (+ x 1) 2))) 4) 10 "jit: (x+1)*2")
	(assert ((jit (lambda (x) (+ (* x 2) 1))) 4) 9 "jit: x*2+1")
	(assert ((jit (lambda (a b c) (+ (* a b) c))) 3 4 5) 17 "jit: a*b+c")
	(assert ((jit (lambda (a b c d) (+ (* a b) (* c d)))) 2 3 4 5) 26 "jit: a*b + c*d")
	(assert ((jit (lambda (x) (+ (+ (+ x 1) 2) 3))) 10) 16 "jit: deeply nested +")
	(assert ((jit (lambda (x) (* (* (* x 2) 2) 2))) 3) 24 "jit: deeply nested *")
	(assert ((jit (lambda (a b) (- (* a a) (* b b)))) 5 3) 16 "jit: a²-b²")
	(assert ((jit (lambda (a b c) (+ a (- b c)))) 10 7 3) 14 "jit: a+(b-c)")

	/* Comparisons */
	(assert ((jit (lambda (x) (< x 10))) 5) true "jit: x < 10 (true)")
	(assert ((jit (lambda (x) (< x 10))) 15) false "jit: x < 10 (false)")
	(assert ((jit (lambda (x) (> x 0))) 5) true "jit: x > 0")
	(assert ((jit (lambda (a b) (equal? a b))) 5 5) true "jit: a == b (true)")
	(assert ((jit (lambda (a b) (equal? a b))) 5 6) false "jit: a == b (false)")

	/* Conditionals */
	(assert ((jit (lambda (x) (if (< x 5) 1 0))) 3) 1 "jit: if x<5 then 1 else 0 (true)")
	(assert ((jit (lambda (x) (if (< x 5) 1 0))) 7) 0 "jit: if x<5 then 1 else 0 (false)")

	/* Constants */
	(assert ((jit (lambda () 42))) 42 "jit: constant return")
	(assert ((jit (lambda (x) 99)) 5) 99 "jit: ignore param, return constant")

	/* Boolean logic */
	(assert ((jit (lambda (x) (and (> x 0) (< x 10)))) 5) true "jit: and (true)")
	(assert ((jit (lambda (x) (and (> x 0) (< x 10)))) 15) false "jit: and (false)")
	(assert ((jit (lambda (x) (or (< x 0) (> x 10)))) 5) false "jit: or (false)")
	(assert ((jit (lambda (x) (or (< x 0) (> x 10)))) 15) true "jit: or (true)")
	(assert ((jit (lambda (x) (not (< x 5)))) 3) false "jit: not (false)")
	(assert ((jit (lambda (x) (not (< x 5)))) 7) true "jit: not (true)")

	/* String operations */
	(assert ((jit (lambda (s) (strlen s))) "hello") 5 "jit: strlen")
	(assert ((jit (lambda (s) (strlen s))) "") 0 "jit: strlen empty")
	(assert ((jit (lambda (s) (strlen s))) "äöü") 6 "jit: strlen utf8 bytes")
	(assert ((jit (lambda (a b) (concat a b))) "foo" "bar") "foobar" "jit: concat")
	(assert ((jit (lambda (s) (substr s 1 3))) "hello") "ell" "jit: substr")
	(assert ((jit (lambda (s) (substr s 2))) "hello") "llo" "jit: substr to end")
	(assert ((jit (lambda (s) (toUpper s))) "hello") "HELLO" "jit: toUpper")
	(assert ((jit (lambda (s) (toLower s))) "HELLO") "hello" "jit: toLower")

	/* Pattern matching (strlike) */
	(assert ((jit (lambda (s) (strlike s "hello"))) "hello") true "jit: strlike exact")
	(assert ((jit (lambda (s) (strlike s "hello"))) "world") false "jit: strlike no match")
	(assert ((jit (lambda (s) (strlike s "h%"))) "hello") true "jit: strlike prefix %")
	(assert ((jit (lambda (s) (strlike s "%o"))) "hello") true "jit: strlike suffix %")
	(assert ((jit (lambda (s) (strlike s "h_llo"))) "hello") true "jit: strlike single _")
	(assert ((jit (lambda (s) (strlike s "%ll%"))) "hello") true "jit: strlike infix")
	(assert ((jit (lambda (s p) (strlike s p))) "hello" "h%") true "jit: strlike dynamic pattern")

	/* Mixed types and nil handling */
	(assert (nil? ((jit (lambda (x) (+ x nil))) 5)) true "jit: + with nil returns nil")
	(assert (nil? ((jit (lambda (x) (* x nil))) 5)) true "jit: * with nil returns nil")
	(assert ((jit (lambda (x) (if (nil? x) 0 x))) nil) 0 "jit: nil? check true")
	(assert ((jit (lambda (x) (if (nil? x) 0 x))) 42) 42 "jit: nil? check false")

	/* alu.go: sql_abs */
	(print "testing sql_abs ...")
	(assert (equal? (sql_abs -5) 5) true "sql_abs of -5 = 5")
	(assert (equal? (sql_abs 3) 3) true "sql_abs of 3 = 3")
	(assert (equal? (sql_abs 0) 0) true "sql_abs of 0 = 0")
	(assert (nil? (sql_abs nil)) true "sql_abs of nil = nil")
	(assert (equal? (sql_abs -3.7) 3.7) true "sql_abs of -3.7 = 3.7")

	/* alu.go: equal_collate / notequal_collate */
	(print "testing collation equality ...")
	(assert (equal_collate "hello" "HELLO" "utf8_general_ci") true "equal_collate ci: hello=HELLO")
	(assert (equal_collate "hello" "HELLO" "utf8_bin") false "equal_collate bin: hello!=HELLO")
	(assert (equal_collate "hello" "hello" "utf8_bin") true "equal_collate bin: hello=hello")
	(assert (nil? (equal_collate nil "x" "utf8_bin")) true "equal_collate nil returns nil")
	(assert (notequal_collate "hello" "HELLO" "utf8_bin") true "notequal_collate bin: hello!=HELLO")
	(assert (notequal_collate "hello" "HELLO" "utf8_general_ci") false "notequal_collate ci: hello=HELLO -> false")
	(assert (nil? (notequal_collate nil "x" "utf8_bin")) true "notequal_collate nil returns nil")

	/* date.go: full coverage */
	(print "testing date functions ...")
	/* helper: identity function returning "any" type to bypass static int validation */
	(define _i (lambda (x) x))
	/* current_date returns a number <= now */
	(assert (number? (current_date)) true "current_date returns number")
	(assert (<= (current_date) (now)) true "current_date <= now")
	/* parse_date */
	(define dt1 (parse_date "2024-06-15"))
	(assert (number? dt1) true "parse_date returns number")
	(assert (nil? (parse_date nil)) true "parse_date nil returns nil")
	/* format_date */
	(assert (equal? (format_date dt1 "%Y-%m-%d") "2024-06-15") true "format_date Y-m-d")
	(assert (equal? (format_date dt1 "%Y") "2024") true "format_date year only")
	(assert (nil? (format_date nil "%Y")) true "format_date nil returns nil")
	/* extract_date */
	(assert (equal? (extract_date dt1 "YEAR") 2024) true "extract_date YEAR")
	(assert (equal? (extract_date dt1 "MONTH") 6) true "extract_date MONTH")
	(assert (equal? (extract_date dt1 "DAY") 15) true "extract_date DAY")
	(assert (nil? (extract_date nil "YEAR")) true "extract_date nil returns nil")
	/* date_add / date_sub (use _i to bypass static type validator for int param) */
	(define dt2 (date_add dt1 (_i 1) "DAY"))
	(assert (equal? (format_date dt2 "%Y-%m-%d") "2024-06-16") true "date_add 1 DAY")
	(define dt3 (date_add dt1 (_i 1) "MONTH"))
	(assert (equal? (format_date dt3 "%Y-%m-%d") "2024-07-15") true "date_add 1 MONTH")
	(define dt4 (date_add dt1 (_i 1) "YEAR"))
	(assert (equal? (format_date dt4 "%Y-%m-%d") "2025-06-15") true "date_add 1 YEAR")
	(define dt5 (date_sub dt1 (_i 1) "DAY"))
	(assert (equal? (format_date dt5 "%Y-%m-%d") "2024-06-14") true "date_sub 1 DAY")
	(assert (nil? (date_add nil (_i 1) "DAY")) true "date_add nil returns nil")
	(assert (nil? (date_sub nil (_i 1) "DAY")) true "date_sub nil returns nil")
	/* date_add units: HOUR, MINUTE, SECOND, WEEK */
	(define dt_h (date_add dt1 (_i 2) "HOUR"))
	(assert (equal? (format_date dt_h "%H") "02") true "date_add 2 HOUR")
	(define dt_m (date_add dt1 (_i 30) "MINUTE"))
	(assert (equal? (format_date dt_m "%i") "30") true "date_add 30 MINUTE")
	(define dt_s (date_add dt1 (_i 45) "SECOND"))
	(assert (equal? (format_date dt_s "%s") "45") true "date_add 45 SECOND")
	(define dt_w (date_add dt1 (_i 1) "WEEK"))
	(assert (equal? (format_date dt_w "%Y-%m-%d") "2024-06-22") true "date_add 1 WEEK")
	/* date_sub units: HOUR, MINUTE, SECOND, WEEK */
	(define dt_sh (date_sub (date_add dt1 (_i 5) "HOUR") (_i 2) "HOUR"))
	(assert (equal? (format_date dt_sh "%H") "03") true "date_sub 2 HOUR")
	/* extract_date: HOUR, MINUTE, SECOND */
	(define dt6 (parse_date "2024-06-15 14:30:45"))
	(assert (equal? (extract_date dt6 "HOUR") 14) true "extract_date HOUR")
	(assert (equal? (extract_date dt6 "MINUTE") 30) true "extract_date MINUTE")
	(assert (equal? (extract_date dt6 "SECOND") 45) true "extract_date SECOND")
	/* date_trunc_day */
	(assert (equal? (format_date (date_trunc_day dt6) "%H:%i:%s") "00:00:00") true "date_trunc_day zeroes time")
	(assert (nil? (date_trunc_day nil)) true "date_trunc_day nil returns nil")
	/* str_to_date */
	(define dt7 (str_to_date "15/06/2024" "%d/%m/%Y"))
	(assert (equal? (format_date dt7 "%Y-%m-%d") "2024-06-15") true "str_to_date custom format")
	(assert (nil? (str_to_date nil "%Y-%m-%d")) true "str_to_date nil returns nil")
	(assert (nil? (str_to_date "invalid" "%Y-%m-%d")) true "str_to_date invalid returns nil")
	/* date_sub MINUTE, SECOND, WEEK */
	(define dt_sm (date_sub (date_add dt1 (_i 30) "MINUTE") (_i 15) "MINUTE"))
	(assert (equal? (format_date dt_sm "%i") "15") true "date_sub MINUTE")
	(define dt_ss (date_sub (date_add dt1 (_i 45) "SECOND") (_i 20) "SECOND"))
	(assert (equal? (format_date dt_ss "%s") "25") true "date_sub SECOND")
	(define dt_sw (date_sub (date_add dt1 (_i 2) "WEEK") (_i 1) "WEEK"))
	(assert (equal? (format_date dt_sw "%Y-%m-%d") "2024-06-22") true "date_sub WEEK")
	/* format_date with float timestamp (toTime tagFloat) - use _i to bypass type check */
	(assert (equal? (format_date (_i 0.0) "%Y") "1970") true "format_date float timestamp epoch")
	/* format_date with invalid string returns nil (toTime tagString fail) */
	(assert (nil? (format_date (_i "not-a-date") "%Y")) true "format_date bad string returns nil")
	/* format_date %% literal percent */
	(assert (equal? (format_date dt1 "100%%") "100%") true "format_date %% literal percent")
	/* str_to_date with time components %H %i %s */
	(define dt_hms (str_to_date "14:30:45" "%H:%i:%s"))
	(assert (equal? (format_date dt_hms "%H:%i:%s") "14:30:45") true "str_to_date time %H:%i:%s")
	/* parse_date on already-parsed date (tagDate passthrough) */
	(assert (equal? (parse_date dt1) dt1) true "parse_date on date is identity")
	/* parse_date on integer (tagInt) - use _i to bypass static type check */
	(assert (number? (parse_date (_i 1718451045))) true "parse_date on int returns date")

	/* list.go: produce */
	(print "testing produce ...")
	(assert (equal? (produce 0 (lambda (x) (< x 5)) (lambda (x) (+ x 1))) '(0 1 2 3 4)) true "produce 0..4")
	(assert (equal? (produce 10 (lambda (x) false) (lambda (x) x)) '()) true "produce empty on false init")
	/* list.go: edge cases */
	(assert (equal? (cdr '()) '()) true "cdr on empty list returns empty")
	(assert (equal? (reduce '(10 20 30) (lambda (acc x) (+ acc x)) 0) 60) true "reduce with neutral")
	(assert (nil? (reduce '() (lambda (acc x) (+ acc x)))) true "reduce empty list no neutral returns nil")
	(assert (equal? (merge '(1 2) '(3 4)) '(1 2 3 4)) true "merge multi-arg")
	(assert (equal? (merge_unique '(1 2) '(2 3)) '(1 2 3)) true "merge_unique multi-arg")
	(assert (has_assoc? nil "key") false "has_assoc? on nil returns false")
	(assert (nil? (get_assoc nil "key")) true "get_assoc on nil returns nil")

	/* list.go: get_assoc */
	(print "testing get_assoc ...")
	(assert (equal? (get_assoc (list "a" 1 "b" 2) "a") 1) true "get_assoc finds key")
	(assert (nil? (get_assoc (list "a" 1) "z")) true "get_assoc missing key returns nil")
	(assert (equal? (get_assoc (list "a" 1) "z" 99) 99) true "get_assoc missing key returns default")
	/* get_assoc on FastDict */
	(define big_ga (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? (get_assoc big_ga "k5") 5) true "get_assoc on FastDict")
	(assert (equal? (get_assoc big_ga "missing" -1) -1) true "get_assoc FastDict missing key with default")

	/* strings.go: sql_substr (1-based) */
	(print "testing additional string functions ...")
	(assert (equal? (sql_substr "hello" 2 3) "ell") true "sql_substr 1-based with length")
	(assert (equal? (sql_substr "hello" 2) "ello") true "sql_substr 1-based to end")
	(assert (equal? (sql_substr "hello" 10) "") true "sql_substr out of bounds returns empty")
	(assert (nil? (sql_substr nil 1 3)) true "sql_substr nil returns nil")

	/* strings.go: strlike_cs (case-sensitive) */
	(assert (strlike_cs "Hello" "H%") true "strlike_cs case-sensitive prefix match")
	(assert (strlike_cs "Hello" "h%") false "strlike_cs case-sensitive no match")
	(assert (strlike_cs "abc" "a_c") true "strlike_cs single char wildcard")

	/* strings.go: trim functions */
	(assert (equal? (strtrim "  hello  ") "hello") true "strtrim both ends")
	(assert (equal? (strltrim "  hello  ") "hello  ") true "strltrim left only")
	(assert (equal? (strrtrim "  hello  ") "  hello") true "strrtrim right only")
	(assert (equal? (sql_trim "  hello  ") "hello") true "sql_trim both ends")
	(assert (equal? (sql_ltrim "  hello  ") "hello  ") true "sql_ltrim left only")
	(assert (equal? (sql_rtrim "  hello  ") "  hello") true "sql_rtrim right only")
	(assert (nil? (sql_trim nil)) true "sql_trim nil returns nil")
	(assert (nil? (sql_ltrim nil)) true "sql_ltrim nil returns nil")
	(assert (nil? (sql_rtrim nil)) true "sql_rtrim nil returns nil")

	/* strings.go: string_repeat */
	(assert (equal? (string_repeat "ab" 3) "ababab") true "string_repeat 3x")
	(assert (equal? (string_repeat "x" 0) "") true "string_repeat 0 = empty")
	(assert (nil? (string_repeat nil 3)) true "string_repeat nil returns nil")
	/* strings.go: strlike with explicit collation */
	(assert (strlike "Hello" "h%" "utf8_general_ci") true "strlike explicit ci collation")
	(assert (strlike "Hello" "h%" "utf8_bin") false "strlike explicit bin collation is CS")
	/* strings.go: sql_substr edge cases */
	(assert (equal? (sql_substr "hello" 0 3) "hel") true "sql_substr start=0 clamped")
	(assert (equal? (sql_substr "hello" 4 10) "lo") true "sql_substr length exceeds remaining")
	/* strings.go: regexp_replace nil + direct */
	(assert (nil? (regexp_replace nil "foo" "bar")) true "regexp_replace nil returns nil")
	/* strings.go: collate with language aliases */
	(define less_en (collate "en"))
	(assert (less_en "a" "b") true "english collation a<b")
	(define less_rev (collate "en" true))
	(assert (less_rev "b" "a") true "english reverse b before a")

	/* scm.go: string (type conversion) / printer.go coverage */
	(print "testing type conversion and apply ...")
	(assert (equal? (string 42) "42") true "string converts number to string")
	(assert (equal? (string "abc") "abc") true "string on string is identity")
	(assert (equal? (string nil) "nil") true "string of nil")
	(assert (equal? (string true) "true") true "string of true")
	(assert (equal? (string false) "false") true "string of false")
	(assert (equal? (string 3.14) "3.14") true "string of float")
	(assert (equal? (string +) "[native func]") true "string of native func")
	/* serialize coverage (printer.go:SerializeEx) - use _i to bypass type checks */
	(assert (equal? (serialize (_i true)) "true") true "serialize true")
	(assert (equal? (serialize (_i false)) "false") true "serialize false")
	(assert (equal? (serialize (_i nil)) "nil") true "serialize nil")
	(assert (equal? (serialize (_i 42)) "42") true "serialize int")
	(assert (equal? (serialize (_i 3.14)) "3.14") true "serialize float")
	(assert (equal? (serialize (_i "hello")) "\"hello\"") true "serialize string")

	/* scm.go: apply */
	(assert (equal? (apply + '(1 2 3)) 6) true "apply + to list")
	(assert (equal? (apply concat '("a" "b" "c")) "abc") true "apply concat to list")

	/* scm.go: error (via try) */
	(define err_caught (newsession))
	(try (lambda () (error "test error")) (lambda (e) (err_caught "msg" e)))
	(assert (strlike (err_caught "msg") "%test error%") true "error throws and try catches")

	/* scm.go: time */
	(define time_result (time (+ 2 3)))
	(assert (equal? time_result 5) true "time returns the computed value")

	/* scm.go: parallel */
	(define par_done (newsession))
	(par_done "a" false)
	(par_done "b" false)
	(parallel (par_done "a" true) (par_done "b" true))
	(assert (par_done "a") true "parallel executed branch a")
	(assert (par_done "b") true "parallel executed branch b")

	/* scm.go: for_mut (optimizer-internal but callable) */
	(assert (equal? (for_mut (list 0) (lambda (x) (< x 5)) (lambda (x) (list (+ x 1)))) '(5)) true "for_mut counts to 5")

	/* sync.go: numcpu / memstats */
	(print "testing system info ...")
	(assert (> (numcpu) 0) true "numcpu > 0")
	(define ms (memstats))
	(assert (> (ms "alloc") 0) true "memstats alloc > 0")
	(assert (> (ms "sys") 0) true "memstats sys > 0")
	(assert (has_assoc? ms "heap_alloc") true "memstats has heap_alloc")
	(assert (has_assoc? ms "heap_sys") true "memstats has heap_sys")
	(assert (has_assoc? ms "total_alloc") true "memstats has total_alloc")

	/* _mut variants (optimizer internal, but test directly for coverage) */
	(print "testing _mut variants ...")
	(assert (equal? (map_mut '(1 2 3) (lambda (x) (* x 2))) '(2 4 6)) true "map_mut doubles")
	(assert (equal? (mapIndex_mut '(10 20) (lambda (i v) (string i))) '("0" "1")) true "mapIndex_mut")
	(assert (equal? (filter_mut '(1 2 3 4) (lambda (x) (> x 2))) '(3 4)) true "filter_mut keeps >2")
	(assert (equal? (append_mut '(1 2) 3 4) '(1 2 3 4)) true "append_mut extends list")
	(assert (equal? (append_unique_mut '(1 2 2) 2 3) '(1 2 2 3)) true "append_unique_mut deduplicates")
	(define d_mut (list "a" 1))
	(set d_mut (set_assoc_mut d_mut "b" 2))
	(assert (equal? (get_assoc d_mut "b") 2) true "set_assoc_mut adds key")
	(define d_mut2 (merge_assoc_mut (list "x" 1) (list "y" 2)))
	(assert (equal? (get_assoc d_mut2 "y") 2) true "merge_assoc_mut merges")
	(define d_mut3 (map_assoc_mut (list "a" 1 "b" 2) (lambda (k v) (+ v 10))))
	(assert (equal? (get_assoc d_mut3 "a") 11) true "map_assoc_mut increments")
	(define d_mut4 (filter_assoc_mut (list "a" 1 "b" 20) (lambda (k v) (> v 5))))
	(assert (has_assoc? d_mut4 "b") true "filter_assoc_mut keeps b")
	(assert (has_assoc? d_mut4 "a") false "filter_assoc_mut drops a")
	(define d_mut5 (extract_assoc_mut (list "a" 1 "b" 2) (lambda (k v) v)))
	(assert (equal? d_mut5 '(1 2)) true "extract_assoc_mut extracts values")

	/* window_mut / window_flush */
	(print "testing window_mut / window_flush ...")
	/* LAG(col1, 1): window_size=2, stride=1, skip=0 */
	(define _win_results (newsession))
	(_win_results "items" '())
	(define _win_lag1 (list 0 0 1 nil nil))
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) 10))
	(assert (equal? (_win_results "items") '(nil)) true "window_mut LAG stride=1 row1 emits nil")
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) 20))
	(assert (equal? (_win_results "items") '(nil 10)) true "window_mut LAG stride=1 row2 emits 10")
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) 30))
	(assert (equal? (_win_results "items") '(nil 10 20)) true "window_mut LAG stride=1 row3 emits 20")

	/* LEAD(col1, 1): window_size=2, stride=1, skip=1 */
	(define _win_results2 (newsession))
	(_win_results2 "items" '())
	(define _win_lead1 (list 1 0 1 nil nil))
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) 10))
	(assert (equal? (_win_results2 "items") '()) true "window_mut LEAD skip=1 row1 no emit")
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) 20))
	(assert (equal? (_win_results2 "items") '(20)) true "window_mut LEAD row2 emits newest=20")
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) 30))
	(assert (equal? (_win_results2 "items") '(20 30)) true "window_mut LEAD row3 emits newest=30")
	/* flush remaining */
	(window_flush _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) 1)
	(assert (equal? (_win_results2 "items") '(20 30 nil)) true "window_flush LEAD emits nil for last row")

	/* LEAD(col1, 2): window_size=3, stride=1, skip=2 */
	(define _win_results3 (newsession))
	(_win_results3 "items" '())
	(define _win_lead2 (list 2 0 1 nil nil nil))
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 10))
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 20))
	(assert (equal? (_win_results3 "items") '()) true "window_mut LEAD(2) skips first 2 rows")
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 30))
	(assert (equal? (_win_results3 "items") '(30)) true "window_mut LEAD(2) row3 emits newest=30")
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 40))
	(assert (equal? (_win_results3 "items") '(30 40)) true "window_mut LEAD(2) row4 emits 40")
	(window_flush _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 2)
	(assert (equal? (_win_results3 "items") '(30 40 nil nil)) true "window_flush LEAD(2) flushes 2 nils")

	/* stride=2: tracking two columns, LAG(col1,1) + LAG(col2,1)
	   window_size=2, stride=2 -> 4 slots, emit gets 4 flat args: old_c1 old_c2 new_c1 new_c2 */
	(define _win_results4 (newsession))
	(_win_results4 "items" '())
	(define _win_stride2 (list 0 0 2 nil nil nil nil))
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) 10 100))
	(assert (equal? (_win_results4 "items") '(nil nil)) true "window_mut stride=2 row1 emits old=(nil nil)")
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) 20 200))
	(assert (equal? (_win_results4 "items") '(nil nil 10 100)) true "window_mut stride=2 row2 emits old=(10 100)")
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) 30 300))
	(assert (equal? (_win_results4 "items") '(nil nil 10 100 20 200)) true "window_mut stride=2 row3 emits old=(20 200)")

	(print "finished unit tests")
	(print "test result: " (teststat "success") "/" (teststat "count"))
	(if (< (teststat "success") (teststat "count")) (begin
		(print "")
		(print "---- !!! some test cases have failed !!! ----")
		(print "")
		(print " it is unsafe to run memcp in this configuration")
	) (print "all tests succeeded."))
	(print "")
) /* end enclosure */
