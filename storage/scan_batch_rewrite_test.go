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
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), newScanAccessSchema(scanAccessConsumerScan, nil, -1)}),
		listAst(),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("inner_id")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("inner_id"), outerID}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("inner_acc"), scm.NewSymbol("inner_id")}), innerMapBody,
		}),
		scm.NewNil(), scm.NewNil(), scm.NewBool(false),
	})
	return []scm.Scmer{
		scm.NewSymbol("scan"), scm.NewNil(), scm.NewSymbol("outer_table"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), newScanAccessSchema(scanAccessConsumerScan, nil, -1)}),
		listAst(),
		scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(nil), scm.NewBool(true)}),
		scm.NewSlice([]scm.Scmer{scm.NewString("ID")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer_acc"), outerID}), innerScan,
		}),
		scm.NewNil(), scm.NewNil(), scm.NewBool(false),
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

func TestScanBatchRewriteKeepsStreamingEmitterOrdered(t *testing.T) {
	emit := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("stream_emit"),
		scm.NewSymbol("emit"),
		scm.NewSymbol("inner_id"),
	})
	if rewritten := tryScanBatchRewrite(nestedScanBatchRewriteCall(emit)); !rewritten.IsNil() {
		t.Fatalf("stream emitter was buffered: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
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

func TestScanBatchRewriteRecognizesNumberedOuterReference(t *testing.T) {
	call := nestedScanBatchRewriteCall(scm.NewNthLocalVar(1))
	outerReducer := call[8].Slice()
	inner := outerReducer[2].Slice()
	innerFilter := inner[6].Slice()
	innerFilter[2] = scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"), scm.NewNthLocalVar(0),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer"), scm.NewInt(1), scm.NewNthLocalVar(1)}),
	})
	rewritten := tryScanBatchRewrite(call)
	if rewritten.IsNil() {
		t.Fatal("numbered outer reference was not batched")
	}
	plan := scm.SerializeToString(rewritten, &scm.Globalenv)
	if !strings.Contains(plan, "scan_batch") || strings.Contains(plan, "(outer 1 (var 1))") {
		t.Fatalf("numbered outer reference was not replaced by a batch slot: %s", plan)
	}
}

func TestScanBatchOptimizerPreservesCompiledBatchAccess(t *testing.T) {
	Init(scm.Globalenv)
	filterColumns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("id"), scm.NewString("#0")})
	filterFn := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("id"), scm.NewSymbol("wanted_id")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("id"), scm.NewSymbol("wanted_id")}),
	})
	accessSchema, accessValues, ok := compileScanAccessMode(filterColumns, filterFn, true)
	if !ok {
		t.Fatal("failed to compile batch access fixture")
	}
	expr := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan_batch"), scm.NewNil(), scm.NewSymbol("table_value"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), accessSchema}),
		scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, accessValues...)),
		filterColumns,
		filterFn,
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("id")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("lambda"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc"), scm.NewSymbol("id")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewSymbol("id")}),
		}),
		scm.NewInt(1), scm.NewSymbol("batch_values"), scm.NewInt(0), scm.NewNil(), scm.NewBool(false),
	})
	optimized := scm.Optimize(expr, &scm.Globalenv, nil)
	plan := scm.SerializeToString(optimized, &scm.Globalenv)
	if !strings.Contains(plan, scanAccessSchemaName) {
		t.Fatalf("scan_batch plan contains no compiled access: %s", plan)
	}
}
