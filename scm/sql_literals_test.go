/*
Copyright (C) 2026  MemCP Contributors

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

func TestParameterizeSQLSelectLiterals(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		normalized string
		bindings   []Scmer
	}{
		{
			name:       "conditions and bounds",
			query:      "SELECT id FROM items WHERE id >= 12 AND label = 'open' ORDER BY id LIMIT 5 OFFSET 2",
			normalized: "SELECT id FROM items WHERE id >= ? AND label = ? ORDER BY id LIMIT ? OFFSET ?",
			bindings:   []Scmer{Simplify("12"), NewString("open"), Simplify("5"), Simplify("2")},
		},
		{
			name:       "comments and quoted identifiers",
			query:      "SELECT `value2` FROM `items7` /* 99 ? */ WHERE note = 'line\\nvalue' -- 42\nLIMIT 3",
			normalized: "SELECT `value2` FROM `items7` /* 99 ? */ WHERE note = ? -- 42\nLIMIT ?",
			bindings:   []Scmer{NewString("line\nvalue"), Simplify("3")},
		},
		{
			name:       "order ordinal and cast width",
			query:      "SELECT CAST(score AS DECIMAL(10,2)) AS 'amount' FROM items WHERE score > 4 ORDER BY 1 LIMIT 6",
			normalized: "SELECT CAST(score AS DECIMAL(10,2)) AS 'amount' FROM items WHERE score > ? ORDER BY 1 LIMIT ?",
			bindings:   []Scmer{Simplify("4"), Simplify("6")},
		},
		{
			name:       "escaped string",
			query:      "SELECT id FROM items WHERE note = 'it\\'s\\\\ok'",
			normalized: "SELECT id FROM items WHERE note = ?",
			bindings:   []Scmer{NewString("it's\\ok")},
		},
		{
			name:       "projection metadata and negative condition",
			query:      "SELECT 1, 'fixed' AS label, score - 2 AS adjusted FROM items WHERE score >= -4",
			normalized: "SELECT 1, 'fixed' AS label, score - 2 AS adjusted FROM items WHERE score >= ?",
			bindings:   []Scmer{Simplify("-4")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, bindings, shapeHash := parameterizeSQLSelectLiterals(tt.query)
			if normalized != tt.normalized {
				t.Fatalf("normalized = %q, want %q", normalized, tt.normalized)
			}
			if shapeHash != fnvHashString(normalized) {
				t.Fatalf("shape hash = %q, want hash of %q", shapeHash, normalized)
			}
			if !Equal(NewSlice(bindings), NewSlice(tt.bindings)) {
				t.Fatalf("bindings = %v, want %v", NewSlice(bindings), NewSlice(tt.bindings))
			}
		})
	}
}

func TestParameterizeSQLSelectLiteralsKeepsUnsafeShapesExact(t *testing.T) {
	queries := []string{
		"UPDATE items SET score = 2 WHERE id = 1",
		"SELECT id FROM items WHERE id = ?",
		"SELECT id FROM items WHERE created_at >= DATE '2026-01-01'",
		"SELECT category, COUNT(*) FROM items WHERE state = 'open' GROUP BY category",
		"SELECT id FROM items WHERE id IN (SELECT item_id FROM links WHERE kind = 2)",
		"SELECT id FROM items WHERE state = 'open' OR score = 2",
		"SELECT COUNT(*) FROM items WHERE state = 'open'",
		"SELECT DISTINCT state FROM items WHERE score >= 2",
		"SELECT id FROM items WHERE id = 1 UNION SELECT id FROM items WHERE id = 2",
		"EXPLAIN COMPILE SELECT id FROM items WHERE id = 1",
	}
	for _, query := range queries {
		normalized, bindings, shapeHash := parameterizeSQLSelectLiterals(query)
		if normalized != query || len(bindings) != 0 {
			t.Errorf("unsafe query changed: %q -> %q, %v", query, normalized, bindings)
		}
		if shapeHash != fnvHashString(query) {
			t.Errorf("unsafe query hash = %q, want hash of exact query", shapeHash)
		}
	}
}

func BenchmarkParameterizeSQLSelectLiterals(b *testing.B) {
	query := "SELECT id, label, score FROM items WHERE tenant_id = 481 AND state = 'open' AND score >= 12.5 ORDER BY created_at DESC LIMIT 50 OFFSET 100"
	b.ReportAllocs()
	b.SetBytes(int64(len(query)))
	for b.Loop() {
		parameterizeSQLSelectLiterals(query)
	}
}
