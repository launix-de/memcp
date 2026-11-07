/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"github.com/launix-de/memcp/scm"
	"reflect"
	"testing"
)

func TestListScanAccessContracts(t *testing.T) {
	for _, c := range []struct {
		name     string
		boundary scm.Scmer
		values   []scm.Scmer
		rows     []scm.Scmer
		want     []bool
	}{
		{"equality", newScanBoundarySpec("id", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false), []scm.Scmer{scm.NewInt(2)}, []scm.Scmer{scm.NewInt(1), scm.NewInt(2), scm.NewNil()}, []bool{false, true, false}},
		{"open lower range", newScanBoundarySpec("id", RangeMatcher, 0, -1, false, false, "", false, -1, nil, nil, "", false), []scm.Scmer{scm.NewInt(2)}, []scm.Scmer{scm.NewInt(1), scm.NewInt(2), scm.NewInt(3)}, []bool{false, false, true}},
		{"inclusive upper range", newScanBoundarySpec("id", RangeMatcher, -1, 0, false, true, "", false, -1, nil, nil, "", false), []scm.Scmer{scm.NewInt(2)}, []scm.Scmer{scm.NewInt(1), scm.NewInt(2), scm.NewInt(3)}, []bool{true, true, false}},
		{"collation", newScanBoundarySpec("id", EqualMatcher, 0, 0, true, true, "utf8mb4_general_ci", false, -1, nil, nil, "", false), []scm.Scmer{scm.NewString("alpha")}, []scm.Scmer{scm.NewString("Alpha"), scm.NewString("beta")}, []bool{true, false}},
		{"NULL probe", newScanBoundarySpec("id", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false), []scm.Scmer{scm.NewNil()}, []scm.Scmer{scm.NewInt(0), scm.NewNil()}, []bool{false, false}},
		{"NULL-safe probe", newScanBoundarySpec("id", EqualMatcher, 0, 0, true, true, "", true, -1, nil, nil, "", false), []scm.Scmer{scm.NewNil()}, []scm.Scmer{scm.NewInt(0), scm.NewNil()}, []bool{false, true}},
		{"IS NULL", newScanBoundarySpec("id", EqualMatcher, -1, -1, true, true, "", false, -1, nil, nil, "", false), nil, []scm.Scmer{scm.NewInt(0), scm.NewNil()}, []bool{false, true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			schema := scm.NewSlice([]scm.Scmer{newScanAccessHeader(1, scanAccessConsumerScan, 0, -1), c.boundary})
			access := bindListScanAccess(schema, c.values)
			for i, value := range c.rows {
				if got := access.matches(dataset([]scm.Scmer{scm.NewString("id"), value})); got != c.want[i] {
					t.Fatalf("row%v got%v want%v", value, got, c.want[i])
				}
			}
		})
	}
}

func TestListScanOperatorsEnforceAccessBeforeLimit(t *testing.T) {
	Init(scm.Globalenv)
	rows := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(1)}),
		scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(2)}),
		scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(3)}),
	})
	schema, values := testEqualScanAccess("id", scm.NewInt(2))
	for _, ordered := range []bool{false, true} {
		var got []int64
		emit := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { got = append(got, a[1].Int()); return scm.NewNil() })
		filter := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
		columns := scm.NewSlice([]scm.Scmer{scm.NewString("id")})
		args := []scm.Scmer{scm.NewNil(), rows, schema, scm.NewSlice(values), scm.NewSlice(nil), filter}
		name := "scan"
		if ordered {
			name = "scan_order"
			args = append(args, columns, scm.NewSlice([]scm.Scmer{scm.Globalenv.Vars["<"]}), scm.NewInt(0), scm.NewInt(0), scm.NewInt(1), columns, emit, scm.NewNil(), scm.NewBool(false))
		} else {
			args = append(args, columns, emit, scm.NewNil(), scm.NewNil(), scm.NewBool(false))
		}
		scm.Apply(scm.Globalenv.Vars[scm.Symbol(name)], args...)
		if !reflect.DeepEqual(got, []int64{2}) {
			t.Fatalf("%s rows%v want[2]", name, got)
		}
	}
}

func TestListScanComputedAccess(t *testing.T) {
	mapper := scm.Eval(scm.Read(t.Name(), `(lambda (x) (+ x 1))`, true), &scm.Globalenv)
	schema := scm.NewSlice([]scm.Scmer{newScanAccessHeader(1, scanAccessConsumerScan, 0, -1), newScanBoundarySpec(".expression", EqualMatcher, 0, 0, true, true, "", false, 1, []string{"id"}, nil, "", false)})
	values := []scm.Scmer{scm.NewInt(3), scm.NewSlice([]scm.Scmer{scm.NewString(".expression"), mapper})}
	access := bindListScanAccess(schema, values)
	for _, c := range []struct {
		id   int64
		want bool
	}{{1, false}, {2, true}, {3, false}} {
		if got := access.matches(dataset([]scm.Scmer{scm.NewString("id"), scm.NewInt(c.id)})); got != c.want {
			t.Fatalf("computedrow%d got%v", c.id, got)
		}
	}
}
