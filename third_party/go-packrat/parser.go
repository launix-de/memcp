/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Parser[T any] interface {
	Match(s *Scanner[T]) (Node[T], bool)
}

func (s *Scanner[T]) applyRule(rule Parser[T]) (Node[T], bool) {
	startPosition := s.position

	memmap := s.memoization[startPosition]
	if memmap == nil {
		memmap = make(map[Parser[T]]*MemoEntry[T])
		s.memoization[startPosition] = memmap
	}

	m := s.Recall(rule, startPosition)
	if m == nil {
		lr := s.lrPool.Get().(*Lr[T])
		*lr = Lr[T]{seed: Node[T]{}, seedOk: false, rule: rule, head: nil, next: s.invocationStack}
		s.invocationStack = lr
		m := &MemoEntry[T]{Lr: lr, Position: startPosition}
		memmap[rule] = m
		ans, ok := rule.Match(s)
		s.invocationStack = s.invocationStack.next
		m.Position = s.position
		if lr.head != nil {
			lr.seed = ans
			lr.seedOk = ok
			result, resultOk := s.LrAnswer(rule, startPosition, m)
			s.lrPool.Put(lr)
			return result, resultOk
		}

		m.Lr = nil
		m.Ans = ans
		m.Ok = ok
		s.lrPool.Put(lr)
		return ans, ok
	}

	s.setPosition(m.Position)

	if m.Lr != nil {
		s.SetupLr(rule, m.Lr)
		return m.Lr.seed, m.Lr.seedOk
	}

	return m.Ans, m.Ok
}

var emptyString = ""

type Node[T any] struct {
	Payload  T
}

type ParserError[T any] struct {
	Parser        Parser[T]
	Line          int
	Column        int
	Position      int
	FailedParsers []Parser[T]
	Input         string
}

func (e *ParserError[T]) Error() string {
	linestartpos := e.Position - 30

	startpos := linestartpos
	if startpos < 0 {
		startpos = 0
	}

	endpos := e.Position + 10
	if endpos >= len(e.Input) {
		endpos = len(e.Input)
		if endpos < 0 {
			endpos = 0
		}
	}

	atomParsers := make(map[*AtomParser[T]]bool)
	regexParsers := make(map[*RegexParser[T]]bool)
	eofParser := false
	allskipws := true

	for _, p := range e.FailedParsers {
		switch pa := p.(type) {
		case *AtomParser[T]:
			atomParsers[pa] = true
			if !pa.skipWs {
				allskipws = false
			}
		case *RegexParser[T]:
			regexParsers[pa] = true
			if !pa.skipWs {
				allskipws = false
			}
		case *EndParser[T]:
			eofParser = true
		}
	}

	expected := strings.Builder{}
	count := 0
	for r := range atomParsers {
		if count >= 5 {
			break
		}
		expected.WriteString("- " + r.atom)
		if r.skipWs {
			expected.WriteString(" (with leading whitespace)")
		}
		expected.WriteString("\r\n")
		count++
	}
	for r := range regexParsers {
		if count >= 5 {
			break
		}
		expected.WriteString("- Regex: " + r.rs)
		if r.skipWs {
			expected.WriteString(" (with leading whitespace)")
		}
		expected.WriteString("\r\n")
		count++
	}
	if eofParser && count < 5 {
		expected.WriteString("- End of input\r\n")
	}

	epos := e.Position
	for allskipws && epos < len(e.Input)-1 && unicode.IsSpace(rune(e.Input[epos])) {
		epos++
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Parser failed at line %d, column %d (position %d of input string).", e.Line, e.Column, e.Position+1))
	replacer := strings.NewReplacer("\r\n", "\\n", "\n", "\\n", "\t", "  ")
	builder.WriteString("\r\n" + replacer.Replace(e.Input[startpos:endpos]))
	builder.WriteString("\r\n")
	runes := []rune(e.Input)
	for i := startpos; i < epos && i < len(runes); i++ {
		c := runes[i]
		switch c {
		case '\n':
			builder.WriteString("  ")
		case '\t':
			builder.WriteString("  ")
		default:
			builder.WriteRune(' ')
		}
	}
	builder.WriteString("^\r\n")
	builder.WriteString("Expected one of " + strconv.Itoa(len(atomParsers)+len(regexParsers)) + " alternatives:\r\n" + expected.String() + "Found: " + strings.ReplaceAll(e.Input[e.Position:endpos], "\n", "\\n"))

	return builder.String()
}

func ParsePartial[T any](p Parser[T], originalScanner *Scanner[T]) (Node[T], *ParserError[T]) {
	node, ok := originalScanner.applyRule(p)
	if ok {
		return node, nil
	}

	maxPos := 0
	var failedParsers []Parser[T]
	for index := len(originalScanner.input); index >= 0; index-- {
		m := originalScanner.memoization[index]
		if len(m) > 0 {
			maxPos = index
			for k := range m {
				failedParsers = append(failedParsers, k)
			}
			break
		}
	}

	consumed := originalScanner.input[:maxPos]
	line := strings.Count(consumed, "\n") + 1
	lastBreak := strings.LastIndex(consumed, "\n")
	if lastBreak < 0 {
		lastBreak = 0
	}
	column := maxPos - lastBreak + 1
	e := &ParserError[T]{FailedParsers: failedParsers, Parser: p, Line: line, Column: column, Position: maxPos, Input: originalScanner.input}
	return Node[T]{}, e
}

func Parse[T any](p Parser[T], originalScanner *Scanner[T]) (Node[T], *ParserError[T]) {
	node, ok := originalScanner.applyRule(p)
	if ok {
		originalScanner.Skip()
		if len(originalScanner.remainingInput) > 0 {
			consumed := originalScanner.input[:originalScanner.position]
			line := strings.Count(consumed, "\n") + 1
			column := originalScanner.position - strings.LastIndex(consumed, "\n") + 1
			e := &ParserError[T]{Parser: p, Line: line, Column: column, Position: originalScanner.position, Input: originalScanner.input}
			return Node[T]{}, e
		}

		return node, nil
	}

	maxPos := 0
	var failedParsers []Parser[T]
	for index := len(originalScanner.input); index >= 0; index-- {
		m := originalScanner.memoization[index]
		if len(m) > 0 {
			maxPos = index
			for k := range m {
				failedParsers = append(failedParsers, k)
			}
			break
		}
	}

	consumed := originalScanner.input[:maxPos]
	line := strings.Count(consumed, "\n") + 1
	lastBreak := strings.LastIndex(consumed, "\n")
	if lastBreak < 0 {
		lastBreak = 0
	}
	column := maxPos - lastBreak + 1
	e := &ParserError[T]{FailedParsers: failedParsers, Parser: p, Line: line, Column: column, Position: maxPos, Input: originalScanner.input}

	return Node[T]{}, e
}
