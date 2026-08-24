/*
Copyright (C) 2026  Carl-Philip Haensch

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
package storage

import (
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func nestedScanBatchRewriteCall(innerMapBody scm.Scmer) []scm.Scmer {
	outerID := scm.NewSymbol("outer_id")
	innerScan := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"), scm.NewNil(), scm.NewSymbol("inner_table"),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("inner_id")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("inner_id"), outerID}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("inner_id")}), innerMapBody,
		}),
		scm.NewNil(), scm.NewNil(), scm.NewNil(), scm.NewBool(false),
	})
	return []scm.Scmer{
		scm.NewSymbol("scan"), scm.NewNil(), scm.NewSymbol("outer_table"),
		scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(nil), scm.NewBool(true)}),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{outerID}), innerScan,
		}),
		scm.NewNil(), scm.NewNil(), scm.NewNil(), scm.NewBool(false),
	}
}

func TestScanBatchRewriteKeepsEffectfulInnerMapperStreaming(t *testing.T) {
	resultrow := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("resultrow"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewSymbol("inner_id")}),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(resultrow)); !rewritten.IsNil() {
		t.Fatalf("effectful nested scan was buffered: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanBatchRewriteKeepsDynamicCallbackStreaming(t *testing.T) {
	emit := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("emit"),
		scm.NewSymbol("inner_id"),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(emit)); !rewritten.IsNil() {
		t.Fatalf("dynamic callback was buffered: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanBatchRewriteKeepsDeclaredEffectStreaming(t *testing.T) {
	insert := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("insert"),
		scm.NewSymbol("target"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewSymbol("inner_id")}),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(insert)); !rewritten.IsNil() {
		t.Fatalf("declared effect was buffered: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanBatchRewriteStillBatchesPureInnerMapper(t *testing.T) {
	rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(scm.NewSymbol("inner_id")))
	if rewritten.IsNil() {
		t.Fatal("pure nested scan was not batched")
	}
	if plan := scm.SerializeToString(rewritten, &scm.Globalenv); !strings.Contains(plan, "scan_batch") {
		t.Fatalf("rewritten plan contains no scan_batch: %s", plan)
	}
}

func TestScanBatchRewriteStillBatchesDeclaredPureCall(t *testing.T) {
	pureCall := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewSymbol("inner_id"),
		scm.NewInt(1),
	})
	rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(pureCall))
	if rewritten.IsNil() {
		t.Fatal("declared pure callback was not batched")
	}
}

func TestScanBatchRewriteStillBatchesImmediatePureLambda(t *testing.T) {
	pureCall := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("value")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewSymbol("value")}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("var"), scm.NewInt(0)}),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(pureCall)); rewritten.IsNil() {
		t.Fatal("immediately invoked pure lambda was not batched")
	}
}

func TestScanBatchRewriteKeepsComputedCallbackStreaming(t *testing.T) {
	emit := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("var"), scm.NewInt(0)}),
		scm.NewSymbol("inner_id"),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(emit)); !rewritten.IsNil() {
		t.Fatalf("computed callback was buffered: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}
