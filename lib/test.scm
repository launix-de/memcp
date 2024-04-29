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

(print "finished unit tests")
(print "test result: " (teststat "success") "/" (teststat "count"))
(if (< (teststat "success") (teststat "count")) (begin
	(print "")
	(print "---- !!! some test cases have failed !!! ----")
	(print "")
	(print " it is unsafe to run memcp in this configuration")
) (print "all tests succeeded."))
(print "")
