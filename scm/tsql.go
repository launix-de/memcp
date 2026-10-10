/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ReadTSQLScript streams bounded statements, preserving quoted GO/semicolons.
// Export scripts commonly omit semicolons between line-start INSERT/SET/DDL.
// Only those unambiguous boundaries are recognized; procedural SQL fails in
// the parser rather than being skipped by the importer.
func ReadTSQLScript(input io.Reader, consume func(string), beginBatch func()) int {
	if beginBatch != nil {
		beginBatch()
	}
	r := bufio.NewReader(input)
	header, _ := r.Peek(3)
	if len(header) >= 3 && string(header[:3]) == "\xef\xbb\xbf" {
		_, _ = r.Discard(3)
	} else if len(header) >= 2 && (string(header[:2]) == "\xff\xfe" || string(header[:2]) == "\xfe\xff") {
		little := header[0] == 0xff
		_, _ = r.Discard(2)
		r = bufio.NewReader(&tsqlUTF16Reader{source: r, little: little})
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), tdsMaxMessage)
	// ScanLines would discard CR bytes, including CRLF inside text literals.
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF && len(data) != 0 {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	var sql strings.Builder
	quote, comments, parentheses := byte(0), 0, 0
	count, lineNumber := 0, 0
	emit := func() {
		statement := strings.TrimSpace(tsqlStripComments(sql.String()))
		sql.Reset()
		if statement != "" {
			count++
			consume(statement)
		}
	}
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text() + "\n"
		if !utf8.ValidString(line) {
			panic(fmt.Sprintf("invalid UTF-8 in t-sql script at line %d", lineNumber))
		}
		if quote == 0 && comments == 0 && parentheses == 0 {
			control := strings.TrimSpace(strings.SplitN(line, "--", 2)[0])
			words := strings.Fields(control)
			if len(words) > 0 && strings.EqualFold(words[0], "GO") {
				if len(words) != 1 {
					panic("GO repeat counts are unsupported")
				}
				emit()
				if beginBatch != nil {
					beginBatch()
				}
				continue
			}
			if len(words) > 0 {
				switch strings.ToUpper(words[0]) {
				case "IF", "INSERT", "SET", "USE", "CREATE", "ALTER", "DROP", "COMMIT", "ROLLBACK", "BEGIN", "EXEC", "EXECUTE":
					if !tsqlConditionalNeedsLeaf(sql.String()) {
						emit()
					}
				}
			}
		}
		start := 0
		for i := 0; i < len(line); i++ {
			ch := line[i]
			if comments != 0 {
				if i+1 < len(line) && line[i:i+2] == "/*" {
					comments++
					i++
				} else if i+1 < len(line) && line[i:i+2] == "*/" {
					comments--
					i++
				}
				continue
			}
			if quote != 0 {
				if ch == quote {
					if i+1 < len(line) && line[i+1] == quote {
						i++
					} else {
						quote = 0
					}
				}
				continue
			}
			if i+1 < len(line) && line[i:i+2] == "--" {
				break
			}
			if i+1 < len(line) && line[i:i+2] == "/*" {
				comments++
				i++
				continue
			}
			switch ch {
			case '[':
				quote = ']'
			case '\'', '"':
				quote = ch
			case '(':
				parentheses++
			case ')':
				parentheses--
			case ';':
				if parentheses == 0 {
					sql.WriteString(line[start:i])
					emit()
					start = i + 1
				}
			}
		}
		sql.WriteString(line[start:])
		if sql.Len() > tdsMaxMessage {
			panic("t-sql import statement exceeds 16 MiB")
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if quote != 0 || comments != 0 || parentheses != 0 {
		panic("unterminated t-sql import statement")
	}
	emit()
	return count
}

// Only the line-boundary heuristic needs this lexical state. IF and ELSE must
// stay with their following leaf; the normal parser validates and executes the
// completed statement. Quoted words, parameters and CASE ELSE are not control
// boundaries. No SQL expression or relational planning is performed here.
func tsqlConditionalNeedsLeaf(statement string) bool {
	statement = strings.TrimSpace(statement)
	if len(statement) < 2 || (!strings.EqualFold(statement[:2], "IF") &&
		!strings.HasPrefix(statement, "/*") && !strings.HasPrefix(statement, "--")) {
		return false
	}
	statement = tsqlStripComments(statement)
	quote := byte(0)
	parentheses, cases := 0, 0
	conditional, leaf := false, false
	for i := 0; i < len(statement); i++ {
		ch := statement[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < len(statement) && statement[i+1] == quote {
					i++
				} else {
					quote = 0
				}
			}
			continue
		}
		switch ch {
		case '[':
			quote = ']'
		case '\'', '"':
			quote = ch
		case '(':
			parentheses++
		case ')':
			parentheses--
		}
		if parentheses != 0 || !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_') {
			continue
		}
		start := i
		for i+1 < len(statement) {
			c := statement[i+1]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
				break
			}
			i++
		}
		if start > 0 && statement[start-1] == '@' {
			continue
		}
		word := strings.ToUpper(statement[start : i+1])
		if !conditional {
			if word != "IF" {
				return false
			}
			conditional = true
			continue
		}
		switch word {
		case "CASE":
			cases++
		case "END":
			cases--
		case "ELSE":
			if cases == 0 {
				leaf = false
			}
		case "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "SET":
			if cases == 0 {
				leaf = true
			}
		}
	}
	return conditional && !leaf
}

func tsqlStripComments(sql string) string {
	var out strings.Builder
	quote, depth := byte(0), 0
	lineComment := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if lineComment {
			if ch == '\n' {
				lineComment = false
				out.WriteByte(ch)
			}
			continue
		}
		if depth != 0 {
			if i+1 < len(sql) && sql[i:i+2] == "/*" {
				depth++
				i++
			} else if i+1 < len(sql) && sql[i:i+2] == "*/" {
				depth--
				i++
			} else if ch == '\n' {
				out.WriteByte(ch)
			}
			continue
		}
		if quote != 0 {
			out.WriteByte(ch)
			if ch == quote {
				if i+1 < len(sql) && sql[i+1] == quote {
					i++
					out.WriteByte(ch)
				} else {
					quote = 0
				}
			}
			continue
		}
		if i+1 < len(sql) && sql[i:i+2] == "--" {
			lineComment = true
			i++
			out.WriteByte(' ')
			continue
		}
		if i+1 < len(sql) && sql[i:i+2] == "/*" {
			depth = 1
			i++
			out.WriteByte(' ')
			continue
		}
		if ch == '[' {
			quote = ']'
		} else if ch == '\'' || ch == '"' {
			quote = ch
		}
		out.WriteByte(ch)
	}
	return out.String()
}

// Decode BOM-marked UTF-16 strictly: replacement characters must never hide
// malformed input in an imported value. The buffered source amortizes reads.
type tsqlUTF16Reader struct {
	source         *bufio.Reader
	little         bool
	pending        [utf8.UTFMax]byte
	position, size int
}

func (r *tsqlUTF16Reader) unit() (uint16, error) {
	first, err := r.source.ReadByte()
	if err != nil {
		return 0, err
	}
	second, err := r.source.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	if r.little {
		return uint16(first) | uint16(second)<<8, nil
	}
	return uint16(second) | uint16(first)<<8, nil
}

func (r *tsqlUTF16Reader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if r.position == r.size {
			v, err := r.unit()
			if err != nil {
				return n, err
			}
			ch := rune(v)
			if v >= 0xd800 && v <= 0xdbff {
				low, err := r.unit()
				if err != nil || low < 0xdc00 || low > 0xdfff {
					return n, fmt.Errorf("invalid UTF-16 surrogate in t-sql import")
				}
				ch = 0x10000 + (rune(v)-0xd800)*1024 + rune(low) - 0xdc00
			} else if v >= 0xdc00 && v <= 0xdfff {
				return n, fmt.Errorf("unpaired UTF-16 surrogate in t-sql import")
			}
			r.position = 0
			r.size = len(utf8.AppendRune(r.pending[:0], ch))
		}
		used := copy(p[n:], r.pending[r.position:r.size])
		n += used
		r.position += used
	}
	return n, nil
}
