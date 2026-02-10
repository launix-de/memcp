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
