/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
		Name: "streamString",

		Fn: func(a ...Scmer) Scmer {
			return NewAny(a[0].Stream())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "creates a stream that contains a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "content", Description: "content to put into the stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "gzip",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compresses a stream with gzip. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "xz",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compresses a stream with xz. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "zcat",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("zcat expects a stream")
			}
			reader, err := gzip.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a compressed gzip stream into a stream of uncompressed data. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "xzcat",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("xzcat expects a stream")
			}
			reader, err := xz.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a compressed xz stream into a stream of uncompressed data. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: nil,
		},
	})
}
