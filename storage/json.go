/*
Copyright (C) 2023  Carl-Philip Hänsch

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

/*

JSON storage on disk for persistence:
 - each node has its own data folder
 - each db/table.jsonl is a jsonl file
 - the first line is #table so it can be read by a simple .jsonl
 - a line can also say #delete <recordid>
 - a line can also say #update <recordid> json
 - on rewrite, db/_table.jsonl is rebuild and replaced (maybe once a week)

*/

import "os"
import "bufio"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func LoadJSON(schema, filename string) {
	f, _ := os.Open(filename)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	lines := make(chan string, 64)

	go func () {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	var t *table
	for s := range(lines) {
		if s == "" {
			// ignore
		} else if s[0:7] == "#table " {
			var ok bool
			db := GetDatabase(schema)
			if db == nil {
				panic("database " + schema + " does not exist")
			}
			t, ok = db.Tables[s[7:]]
			if !ok {
				// new table
				t = CreateTable(schema, s[7:], Safe)
			}
		} else if s[0] == '#' {
			// comment
		} else {
			if t == nil {
				panic("no table set")
			} else {
				if len(t.Columns) == 0 {
					// JSON with an unknown table format -> create dummy cols
					var x map[string]scm.Scmer
					json.Unmarshal([]byte(s), &x) // parse JSON
					for c, _ := range x {
						// create column with dummy storage for next rebuild
						t.CreateColumn(c, "ANY", []int{}, "AUTO CREATED")
					}
				}
				func (t *table, s string) {
					var y map[string]scm.Scmer
					json.Unmarshal([]byte(s), &y) // parse JSON
					x := make([]scm.Scmer, 2*len(y))
					i := 0
					for k, v := range y {
						x[i] = k
						x[i+1] = v
						i += 2
					}
					t.Insert(x) // put into table
				}(t, s)
			}
		}
	}
}

