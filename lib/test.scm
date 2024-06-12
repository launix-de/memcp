/*
Copyright (C) 2024  Carl-Philip Hänsch

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

/* Tests for scm package */
/* Tests for alu.go */

/* Test for number? */
(assert (number? 42) true "42 should be a number")
(assert (number? "42") false "\"42\" should not be a number")
(assert (number? `symbol) false "'symbol' should not be a number")

/* Test for + */
(assert (+ 1 2) 3 "1 + 2 should be 3")
(assert (+ 1 2 3) 6 "1 + 2 + 3 should be 6")
(assert (+ 0 0) 0 "0 + 0 should be 0")

/* Test for - */
(assert (- 5 3) 2 "5 - 3 should be 2")
(assert (- 5 3 1) 1 "5 - 3 - 1 should be 1")

/* Test for * */
(assert (* 2 3) 6 "2 * 3 should be 6")
(assert (* 2 3 4) 24 "2 * 3 * 4 should be 24")

/* Test for / */
(assert (/ 6 2) 3 "6 / 2 should be 3")
(assert (/ 12 2 2) 3 "12 / 2 / 2 should be 3")

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

(print "finished unit tests")
(print "test result: " (teststat "success") "/" (teststat "count"))
(if (< (teststat "success") (teststat "count")) (begin
	(print "")
	(print "---- !!! some test cases have failed !!! ----")
	(print "")
	(print " it is unsafe to run memcp in this configuration")
) (print "all tests succeeded."))
(print "")
