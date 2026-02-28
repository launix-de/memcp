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

import (
	"bufio"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

func init_streams() {
	// string functions
	DeclareTitle("Streams")

	Declare(&Globalenv, &Declaration{
		"streamString", "creates a stream that contains a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"content", "string", "content to put into the stream", nil},
		}, "stream",
		func(a ...Scmer) Scmer {
			reader, writer := io.Pipe()
			go func() {
				io.WriteString(writer, String(a[0]))
				writer.Close()
			}()
			return NewAny(io.Reader(reader))
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"gzip", "compresses a stream with gzip. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream", nil},
		}, "stream",
		func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("gzip expects a stream")
			}
			reader, writer := io.Pipe()
			bwriter := bufio.NewWriterSize(writer, 16*1024)
			zip := gzip.NewWriter(bwriter)
			go func() {
				io.Copy(zip, stream)
				zip.Close()
				bwriter.Flush()
				writer.Close()
			}()
			return NewAny(io.Reader(reader))
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"xz", "compresses a stream with xz. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream", nil},
		}, "stream",
		func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("xz expects a stream")
			}
			reader, writer := io.Pipe()
			bwriter := bufio.NewWriterSize(writer, 16*1024)
			zip, err := xz.NewWriter(bwriter)
			go func() {
				io.Copy(zip, stream)
				zip.Close()
				bwriter.Flush()
				writer.Close()
			}()
			if err != nil {
				panic(err)
			}
			return NewAny(io.Reader(reader))
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"zcat", "turns a compressed gzip stream into a stream of uncompressed data. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream", nil},
		}, "stream",
		func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("zcat expects a stream")
			}
			reader, err := gzip.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"xzcat", "turns a compressed xz stream into a stream of uncompressed data. Create streams with (stream filename)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"stream", "stream", "input stream", nil},
		}, "stream",
		func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("xzcat expects a stream")
			}
			reader, err := xz.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		}, false, false, nil,
		nil,
	})
}
