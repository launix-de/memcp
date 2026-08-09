/*
Copyright (C) 2026  Carl-Philip Hänsch

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
package scm

import "testing"

func TestKleeneParserWithoutMemoization(t *testing.T) {
	parserValue := Eval(Read("no-memo parser", `(parser '(
		(define values (* (regex "[0-9]+" false false) "," true))
		$
	) values "")`), &Globalenv)
	parser := parserValue.Parser()

	if got := parser.Execute("1,2,3", &Globalenv); String(got) != `(1 2 3)` {
		t.Fatalf("no-memo parser returned %s", String(got))
	}
	if got := parser.Execute("", &Globalenv); String(got) != `()` {
		t.Fatalf("empty no-memo parser returned %s", String(got))
	}
}
