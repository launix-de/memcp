/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"io"
	"math"
	"strings"
	"sync/atomic"
)

const tdsCursorLimit = 32
const tdsCursorBytesLimit = 16 << 20
const tdsCursorRowsLimit = 100000
const tdsCursorFetchLimit = 10000

// A STATIC cursor owns an immutable, bounded snapshot of logical scalar rows.
// Only the connection execution goroutine accesses these handles/positions.
// No transaction, query session, planner, shard or physical row identity survives.
type tdsCursor struct {
	columns []tdsColumn
	rows    [][]Scmer
	bytes   int
	start   int // 1-based first row of the last buffer; 0/beyond end are sentinels
	fetched int
	name    string
}

func tdsCursorInteger(argument tdsRPCParameter, output bool, nullable bool) int32 {
	flags := byte(0)
	if output {
		flags = 1
	}
	if argument.flags != flags || (!argument.value.IsInt() && !(nullable && argument.value.IsNil())) {
		panic("invalid TDS cursor integer parameter")
	}
	if argument.value.IsNil() {
		return 0
	}
	n := argument.value.Int()
	if n < math.MinInt32 || n > math.MaxInt32 {
		panic("TDS cursor integer parameter overflow")
	}
	return int32(n)
}

func (c *tdsConnection) removeCursor(handle int32) {
	if cursor, found := c.cursors[handle]; found {
		c.cursorBytes -= cursor.bytes
		delete(c.cursors, handle)
	}
}

func (c *tdsConnection) openCursor(arguments []tdsRPCParameter, request *tdsRequest, wires ...[]Scmer) (*tdsCursor, int32) {
	if len(arguments) < 5 || c.server.describe.IsNil() {
		panic("STATIC cursor requires a pure SELECT describe callback")
	}
	if len(c.cursors) >= tdsCursorLimit || c.cursorBytes >= tdsCursorBytesLimit || c.nextCursor == math.MaxInt32 {
		panic("TDS cursor resource limit exceeded")
	}
	if tdsCursorInteger(arguments[0], true, true) != 0 {
		panic("cursor output handle must be unassigned")
	}
	query := tdsRPCText(arguments[1], false)
	scroll := tdsCursorInteger(arguments[2], true, false)
	if scroll != 8 && scroll != 0x1008 {
		panic("only STATIC TDS cursors are supported")
	}
	if scroll == 0x1008 && len(arguments) < 6 {
		panic("parameterized cursor requires declarations")
	}
	concurrency := tdsCursorInteger(arguments[3], true, false)
	if concurrency != 1 && concurrency != 0x2001 {
		panic("only READ_ONLY TDS cursors are supported")
	}
	tdsCursorInteger(arguments[4], true, true)
	var parameters map[string]Scmer
	var declarations []tdsDeclaration
	declarationText := ""
	if len(arguments) > 5 {
		declarationText = tdsRPCText(arguments[5], true)
		var err error
		declarations, err = c.declarations(declarationText)
		if err != nil {
			panic(err)
		}
		var descriptors []Scmer
		if len(wires) > 0 && len(wires[0]) > 0 {
			if len(wires[0]) != len(arguments) {
				panic("invalid cursor parameter descriptor carrier")
			}
			descriptors = wires[0][6:]
		}
		parameters = c.bindParameters(declarations, arguments[6:], descriptors)
	}
	statements, err := splitTDSBatch(query)
	if err != nil || len(statements) != 1 {
		panic("cursor requires exactly one SELECT statement")
	}
	atomic.AddInt64(&TotalHTTPRequests, 1)
	seq := c.ss.BeginQuery("Query", query)
	ctx, cancel := context.WithCancel(request.ctx)
	c.ss.SetCancel(seq, cancel)
	c.ss.SetQueryContext(seq, ctx)
	c.seq.Store(seq)
	defer cancel()
	defer c.seq.Store(0)
	defer c.ss.EndQuery(seq, "Sleep", "")
	// Describe validates the logical SELECT without executing DML. Flags and
	// placeholder values live in a disposable clone, never the connection map.
	Apply(c.server.describe, NewString(c.database), NewString(query), NewString(declarationText), c.descriptionSession(declarations))
	if err := request.ctx.Err(); err != nil {
		panic(err)
	}
	sink := &tdsResult{w: io.Discard, bytes: 512, snapshotLimit: tdsCursorBytesLimit - c.cursorBytes}
	Apply(c.server.query, NewString(c.database), NewString(query), NewFunc(sink.row), NewFunc(sink.fields),
		c.parameterSession(c.session, parameters, declarations), NewAny(c.ss), NewInt(int64(seq)))
	if err := request.ctx.Err(); err != nil {
		panic(err)
	}
	if sink.columns == nil {
		panic("cursor query did not return a result set")
	}
	if len(sink.columns) > 254 {
		panic("TDS cursor supports at most 254 visible columns")
	}
	for i := range sink.columns {
		sink.columns[i].name = strings.Clone(sink.columns[i].name)
		sink.bytes += 64 + len(sink.columns[i].name)
	}
	if sink.bytes > sink.snapshotLimit {
		panic("TDS cursor memory limit exceeded")
	}
	// Validate the entire snapshot before publishing a handle. Wire codecs
	// operate on private copies; retained frontend scalars remain unchanged.
	rows := sink.rows
	sink.rows = nil
	sink.publish()
	for _, row := range rows {
		sink.emit(row)
	}
	cursor := &tdsCursor{columns: sink.columns, rows: rows, bytes: sink.bytes}
	c.nextCursor++
	return cursor, c.nextCursor
}

func (cursor *tdsCursor) rangeForFetch(kind, rownum, nrows int32) (start, end, position int, status uint32) {
	count, n := len(cursor.rows), int(nrows)
	if n < 0 || n > tdsCursorFetchLimit || (n == 0 && kind != 1 && kind != 8) {
		panic("invalid or excessive TDS cursor fetch size")
	}
	position = cursor.start
	switch kind {
	case 1:
		position = 1
		if n == 0 {
			return 0, 0, 0, 0
		}
	case 2:
		position += cursor.fetched
		if position == 0 {
			position = 1
		}
	case 4:
		position -= n
		if position < 1 && cursor.start > 0 {
			position, status = 1, 2
		}
	case 8:
		position = count - n + 1
		if n == 0 {
			return 0, 0, count + 1, 0
		}
		if position < 1 {
			position = 1
		}
	case 16:
		position = int(rownum)
		if position < 0 {
			position += count + 1
		}
	case 32:
		position += int(rownum)
	default:
		panic("unsupported TDS cursor fetch mode")
	}
	if position <= 0 {
		return 0, 0, 0, status
	}
	if position > count {
		return count, count, count + 1, status
	}
	start = position - 1
	end = start + n
	if end > count {
		end = count
	}
	return
}

func (c *tdsConnection) cursorRPC(rpc tdsRPC, request *tdsRequest, response *tdsResponse) (err error) {
	var tokens bytes.Buffer
	var openedHandle int32
	completed := false
	defer func() {
		if openedHandle != 0 && !completed {
			c.removeCursor(openedHandle)
		}
	}()
	status := uint32(0)
	info := false
	var positionCursor *tdsCursor
	position, fetched := 0, 0
	switch rpc.procedure {
	case 2:
		cursor, handle := c.openCursor(rpc.arguments, request, rpc.wireDescriptors)
		if c.cursors == nil {
			c.cursors = make(map[int32]*tdsCursor)
		}
		c.cursors[handle] = cursor
		c.cursorBytes += cursor.bytes
		openedHandle = handle
		sink := &tdsResult{w: response, columns: cursor.wireColumns(), browseMetadata: c.browseMetadata}
		sink.publish()
		tdsDone(&tokens, 0xff, 1, 0)
		tokens.WriteByte(0x79)
		tdsU32(&tokens, 0)
		for _, output := range []struct {
			ordinal uint16
			value   int32
		}{{0, handle}, {2, 8}, {3, 1}, {4, int32(len(cursor.rows))}} {
			tdsReturnInteger(&tokens, rpc.arguments[output.ordinal].name, output.ordinal, output.value)
		}
	case 8:
		if len(rpc.arguments) != 3 {
			panic("cursor option requires handle, code and value")
		}
		handle := tdsRPCHandle(rpc.arguments[0])
		cursor, found := c.cursors[handle]
		if !found {
			panic("unknown TDS cursor handle")
		}
		code := tdsCursorInteger(rpc.arguments[1], false, false)
		switch code {
		case 2:
			name := tdsRPCText(rpc.arguments[2], false)
			if len(name) > 512 || len(tdsUTF16(name)) > 256 {
				panic("TDS cursor name exceeds 128 characters")
			}
			for id, candidate := range c.cursors {
				if name != "" && id != handle && strings.EqualFold(candidate.name, name) {
					panic("duplicate TDS cursor name")
				}
			}
			oldSize, newSize := 0, 0
			if cursor.name != "" {
				oldSize = 16 + int(align8(uint(len(cursor.name))))
			}
			if name != "" {
				newSize = 16 + int(align8(uint(len(name))))
			}
			if c.cursorBytes+newSize-oldSize > tdsCursorBytesLimit {
				panic("TDS cursor memory limit exceeded")
			}
			cursor.name = strings.Clone(name)
			cursor.bytes += newSize - oldSize
			c.cursorBytes += newSize - oldSize
		case 3:
			column := tdsCursorInteger(rpc.arguments[2], false, false)
			if column < 0 || int(column) > len(cursor.columns) {
				panic("invalid TDS cursor TEXTDATA column")
			}
			// Snapshots always return scalar data; text pointers are unsupported.
		case 4, 5, 6:
			tdsCursorInteger(rpc.arguments[2], true, true)
			value := int32(8)
			if code == 5 {
				value = 1
			}
			if code == 6 {
				value = int32(len(cursor.rows))
			}
			tokens.WriteByte(0x79)
			tdsU32(&tokens, 0)
			tdsReturnInteger(&tokens, rpc.arguments[2].name, 2, value)
			info = true
		default:
			panic("unsupported TDS cursor option")
		}
		if !info {
			tokens.WriteByte(0x79)
			tdsU32(&tokens, 0)
		}
	case 7, 9:
		if len(rpc.arguments) == 0 {
			panic("missing TDS cursor handle")
		}
		handle := tdsRPCHandle(rpc.arguments[0])
		cursor, found := c.cursors[handle]
		if !found {
			panic("unknown TDS cursor handle")
		}
		if rpc.procedure == 9 {
			if len(rpc.arguments) != 1 {
				panic("cursor close requires exactly one handle")
			}
			c.removeCursor(handle)
		} else {
			if len(rpc.arguments) > 4 {
				panic("too many cursor fetch parameters")
			}
			kind, rownum, nrows := int32(2), int32(0), int32(20)
			if len(rpc.arguments) > 1 {
				kind = tdsCursorInteger(rpc.arguments[1], false, false)
			}
			if kind == 256 {
				info = true
				if len(rpc.arguments) != 4 {
					panic("cursor INFO requires rownum/nrows outputs")
				}
				tdsCursorInteger(rpc.arguments[2], true, true)
				tdsCursorInteger(rpc.arguments[3], true, true)
				rownum = int32(cursor.start)
				if cursor.start > len(cursor.rows) {
					rownum = -1
				}
				tokens.WriteByte(0x79)
				tdsU32(&tokens, 0)
				tdsReturnInteger(&tokens, rpc.arguments[2].name, 2, rownum)
				tdsReturnInteger(&tokens, rpc.arguments[3].name, 3, int32(len(cursor.rows)))
				break
			}
			if len(rpc.arguments) > 2 {
				rownum = tdsCursorInteger(rpc.arguments[2], false, kind == 1 || kind == 2 || kind == 4 || kind == 8)
			}
			if len(rpc.arguments) > 3 {
				nrows = tdsCursorInteger(rpc.arguments[3], false, false)
			}
			start, end, next, fetchStatus := cursor.rangeForFetch(kind, rownum, nrows)
			positionCursor, position, fetched, status = cursor, next, end-start, fetchStatus
			sink := &tdsResult{w: response, columns: cursor.wireColumns(), browseMetadata: c.browseMetadata}
			if rpc.options&2 != 0 {
				// The cursor's immutable schema was already advertised by open.
				sink.write([]byte{0x81, 255, 255})
				sink.published = true
			} else {
				sink.publish()
			}
			for _, row := range cursor.rows[start:end] {
				if err := request.ctx.Err(); err != nil {
					panic(err)
				}
				values := append(append([]Scmer(nil), row...), NewInt(1))
				sink.emit(values)
			}
			tdsDone(&tokens, 0xfd, 1, 0)
		}
		if !info {
			tokens.WriteByte(0x79)
			tdsU32(&tokens, status)
		}
	}
	if request.finish() {
		tokens.Reset()
		tdsDone(&tokens, 0xfd, 0x20, 0)
	} else {
		tdsDone(&tokens, 0xfe, 0, 0)
	}
	if _, err = response.Write(tokens.Bytes()); err != nil {
		return err
	}
	err = response.flush(true)
	if err == nil && request.ctx.Err() == nil {
		completed = true
		if positionCursor != nil {
			positionCursor.start, positionCursor.fetched = position, fetched
		}
	}
	return
}

func (cursor *tdsCursor) wireColumns() []tdsColumn {
	columns := append([]tdsColumn(nil), cursor.columns...)
	for i := range columns {
		columns[i].updatable = false // implemented cursors are static read-only snapshots
		columns[i].source = [4]string{}
		columns[i].key = false
	}
	return append(columns, tdsColumn{name: "rowstat", kind: 0x26, size: 4, declared: true, hidden: true})
}
