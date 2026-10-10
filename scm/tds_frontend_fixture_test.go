/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"fmt"
	"strings"
)

// These protocol tests supply a mock frontend. Production declaration parsing,
// type decisions, casts and output recipes belong to lib/, not these adapters.
func tdsInstallTestFrontend(server *tdsServer) {
	server.resolveMetadata = NewFunc(func(a ...Scmer) Scmer {
		fixtures := map[string]string{"sp_tables": "sp_tables", "dbo.sp_tables": "sp_tables", "[fixture]..sp_tables": "sp_tables", "[fixture]..[sp_tables]": "sp_tables", "sp_pkeys": "sp_pkeys", "sp_statistics": "sp_statistics", "sp_special_columns": "sp_special_columns", "sp_fkeys": "sp_fkeys"}
		procedure, ok := fixtures[a[0].String()]
		if !ok {
			panic("mock frontend rejected metadata route")
		}
		return NewSlice([]Scmer{a[1], NewString(procedure)})
	})

	server.declarations = NewFunc(func(a ...Scmer) Scmer {
		text := a[1].String()
		for _, invalid := range []string{"@v int(4)", "@v nvarchar(0)", "@v nvarchar(max),@V int", "@v nchar(max)", "@v time(8)", "@v decimal(10,20)", "@v int output"} {
			if text == invalid {
				panic("mock frontend rejected fixture declaration")
			}
		}
		declarations := []Scmer{}
		start, depth := 0, 0
		for i := 0; i <= len(text); i++ {
			if i < len(text) {
				if text[i] == '(' {
					depth++
				}
				if text[i] == ')' {
					depth--
				}
				if text[i] != ',' || depth != 0 {
					continue
				}
			}
			pieces := strings.Fields(text[start:i])
			if len(pieces) != 2 || !strings.HasPrefix(pieces[0], "@") {
				panic("invalid fixture declaration")
			}
			name := strings.ToLower(pieces[0][1:])
			kind := strings.ToUpper(strings.SplitN(pieces[1], "(", 2)[0])
			declarations = append(declarations, NewSlice([]Scmer{NewString(name), NewSlice([]Scmer{NewString(kind)})}))
			start = i + 1
		}
		return NewSlice(declarations)
	})
	server.convertParameter = NewFunc(func(a ...Scmer) Scmer {
		name := a[4].String()
		if a[3].String() != "" {
			name = strings.ToLower(strings.TrimPrefix(a[3].String(), "@"))
		}
		return NewSlice([]Scmer{NewString(name), a[0]})
	})
	server.bindSession = NewFunc(func(a ...Scmer) Scmer {
		base, params, specs := a[0].Func(), a[1], a[2]
		values := map[string]Scmer{}
		types := map[string]Scmer{}
		for i := 0; i < len(params.Slice()); i += 2 {
			values[params.Slice()[i].String()] = params.Slice()[i+1]
		}
		for i := 0; i < len(specs.Slice()); i += 2 {
			types[specs.Slice()[i].String()] = specs.Slice()[i+1]
		}
		return NewFunc(func(args ...Scmer) Scmer {
			if len(args) == 0 {
				keys := append([]Scmer(nil), base().Slice()...)
				for name := range values {
					keys = append(keys, NewString("tsql_bound:"+name), NewString("tsql_param:"+name))
				}
				for name := range types {
					keys = append(keys, NewString("tsql_param_type:"+name))
				}
				return NewSlice(keys)
			}
			if args[0].IsString() {
				key := args[0].String()
				mode, name, _ := strings.Cut(key, ":")
				name = strings.ToLower(name)
				switch mode {
				case "tsql_bound", "tsql_param", "tsql_param_type":
					if len(args) != 1 {
						panic("input bindings are read-only")
					}
					if mode == "tsql_bound" {
						_, ok := values[name]
						return NewBool(ok)
					}
					if mode == "tsql_param_type" {
						return types[name]
					}
					return values[name]
				}
			}
			return base(args...)
		})
	})
	server.event = NewFunc(func(a ...Scmer) Scmer {
		session := a[1].Func()
		switch a[0].String() {
		case "initialize":
			session(NewString("syntax"), NewString("tsql"))
			return session(NewString("tsql_nocount"), NewBool(false))
		case "begin":
			return session(NewString("tsql_scope_identity"), NewNil())
		case "describe":
			return session(NewString("tsql_prepare_describe"), NewBool(true))
		case "done":
			session(NewString("tsql_rowcount"), a[2])
			return session(NewString("tsql_nocount"))
		}
		return NewNil()
	})
}

// A fixed fixture table supplies wire metadata, rather than testing Go SQL
// type inference. Empty/all-NULL results use precisely these descriptors.
func tdsTestDescription(fields []Scmer) Scmer {
	input := NewSlice(fields)
	name := tdsDescriptorValue(input, "sql_type").String()
	layouts := map[string]struct {
		kind byte
		size int
	}{
		"INT": {0x26, 4}, "BIGINT": {0x26, 8}, "BIT": {0x68, 1}, "VARCHAR": {0xa7, 0xffff}, "NVARCHAR": {0xe7, 0xffff}, "VARBINARY": {0xa5, 0xffff}, "ROWVERSION": {0xad, 8}, "DATE": {0x28, 0}, "TIME": {0x29, 0}, "DATETIME2": {0x2a, 0}, "DECIMAL": {0x6a, 17},
	}
	layout, ok := layouts[name]
	if !ok {
		panic(fmt.Sprintf("unknown result fixture %q", name))
	}
	result := []Scmer{NewString("wire_kind"), NewInt(int64(layout.kind))}
	size := layout.size
	if (layout.kind == 0xa7 || layout.kind == 0xe7 || layout.kind == 0xa5) && !tdsDescriptorValue(input, "size").IsNil() {
		size = int(tdsDescriptorValue(input, "size").Int())
		if size < 0 {
			size = 0xffff
		} else if layout.kind == 0xe7 {
			size *= 2
		}
	}
	result = append(result, NewString("wire_size"), NewInt(int64(size)))
	if layout.kind == 0x29 || layout.kind == 0x2a {
		if tdsDescriptorValue(input, "scale").IsNil() {
			result = append(result, NewString("scale"), NewInt(7))
		}
	}
	for i := 0; i < len(fields); i += 2 {
		if fields[i].String() != "sql_type" && fields[i].String() != "size" {
			result = append(result, fields[i], fields[i+1])
		}
	}
	return NewSlice(result)
}
