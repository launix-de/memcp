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

import "os"
import "bufio"
import "strings"
import "github.com/launix-de/memcp/scm"

func LoadCSV(table, filename, delimiter string) {
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

	t, ok := tables[table]
	if !ok {
		panic("table " + table + " does not exist")
	}
	for s := range(lines) {
		if s == "" {
			// ignore
		} else {
			arr := strings.Split(s, delimiter)
			x := make(map[string]scm.Scmer)
			i := 0
			// todo: dont rely on column order; add column idx
			for col := range t.columns {
				if i < len(arr) {
					x[col] = scm.Simplify(arr[i])
				}
				i++
			}
			t.Insert(x)
		}
	}
}

