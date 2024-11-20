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

import "io"
import "compress/gzip"
import "github.com/ulikunitz/xz"


func init_streams() {
	// string functions
	DeclareTitle("Streams")

	// TODO: add support for writers
	Declare(&Globalenv, &Declaration{
		"gzip", "turns a compressed gzip stream into a stream of uncompressed data. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream"},
		}, "stream",
		func(a ...Scmer) (result Scmer) {
			stream := a[0].(io.Reader)
			result, err := gzip.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"xz", "turns a compressed xz stream into a stream of uncompressed data. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream"},
		}, "stream",
		func(a ...Scmer) (result Scmer) {
			stream := a[0].(io.Reader)
			result, err := xz.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return result
		},
	})
}


