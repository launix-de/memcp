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

package NonLockingReadMap

import "encoding/json"
import "sort"

func (m NonLockingReadMap[T, TK]) MarshalJSON() ([]byte, error) {
	// serialize through map (inefficient but nobody cares for now)
	temp := make(map[TK]*T)
	for _, v := range m.GetAll() {
		temp[(*v).GetKey()] = v
	}
	return json.Marshal(temp)
}

func (m *NonLockingReadMap[T, TK]) UnmarshalJSON(b []byte) error {
	// deserialize through map (inefficient but nobody cares for now)
	newhandle := new([]*T)
	temp := make(map[TK]*T)
	err := json.Unmarshal(b, &temp)
	if err != nil {
		return err
	}
	*newhandle = make([]*T, len(temp))
	i := 0
	for _, v := range temp {
		(*newhandle)[i] = v
		i++
	}
	sort.Slice(*newhandle, func(i, j int) bool { // sort
		return (*(*newhandle)[i]).GetKey() < (*(*newhandle)[j]).GetKey()
	})
	m.p.Store(newhandle) // store since no value escaped yet
	return nil
}
