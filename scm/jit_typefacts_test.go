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

func TestJITTypeFactsConst(t *testing.T) {
	f := JITFactsConst(NewInt(42))
	if !f.HasConst {
		t.Fatalf("expected constant facts")
	}
	if !f.IsSingleTag() {
		t.Fatalf("expected single tag for constant")
	}
	if !f.MayBeTag(tagInt) {
		t.Fatalf("expected int tag")
	}
	if f.MayBeTag(tagString) {
		t.Fatalf("did not expect string tag")
	}
}

func TestJITTypeFactsRefineBranches(t *testing.T) {
	base := JITFactsUnknown()
	intOnly := base.RefineToTag(tagInt)
	if !intOnly.IsSingleTag() || !intOnly.MayBeTag(tagInt) {
		t.Fatalf("expected int-only facts after positive refine: %+v", intOnly)
	}
	notInt := base.RefineNotTag(tagInt)
	if notInt.MayBeTag(tagInt) {
		t.Fatalf("expected int to be removed from negative branch: %+v", notInt)
	}
}

func TestJITTypeFactsJoin(t *testing.T) {
	left := JITFactsKnownTag(tagInt)
	right := JITFactsKnownTag(tagFloat)
	join := left.Join(right)
	if !join.MayBeTag(tagInt) || !join.MayBeTag(tagFloat) {
		t.Fatalf("expected int|float after join: %+v", join)
	}
	if join.IsSingleTag() {
		t.Fatalf("expected non-single tag after join")
	}
}
