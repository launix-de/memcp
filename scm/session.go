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

package scm

import "sync"

/* threadsafe session storage */

type session struct {
	Mu sync.RWMutex
	Map map[string]Scmer
}

// build this function into your SCM environment to offer http server capabilities
func NewSession(a ...Scmer) Scmer {
	// params: port, authcallback, schemacallback, querycallback
	sess := new(session)
	sess.Map = make(map[string]Scmer)
	return func (a ...Scmer) (result Scmer) {
		if len(a) == 2 {
			// set
			sess.Mu.Lock()
			defer sess.Mu.Unlock()
			sess.Map[String(a[0])] = a[1]
			return a[1] // reflect the value as of mysql semantics
		} else if len(a) == 1 {
			// get
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			result, _ = sess.Map[String(a[0])]
			return
		} else if len(a) == 0 {
			// list keys
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			l := make([]Scmer, 0, len(sess.Map))
			for k, _ := range sess.Map {
				l = append(l, k)
			}
			result = l
			return
		} else {
			panic("wrong number of parameters provided to session: 0, 1 or 2 required")
		}
	}
}
