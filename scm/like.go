/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "strings"
import "unicode"
import "unicode/utf8"

const (
	LikeLiteral byte = iota
	LikeAny
	LikeMany
)

type LikeToken struct {
	Rune rune
	Kind byte
}

// DecodeLikePattern owns escaping for matching and index candidate selection.
// Escape characters are recognized before case folding, including Unicode.
func DecodeLikePattern(pattern, escape string) []LikeToken {
	if utf8.RuneCountInString(escape) > 1 {
		panic("LIKE ESCAPE must be empty or one character")
	}
	escapeRune, _ := utf8.DecodeRuneInString(escape)
	tokens := make([]LikeToken, 0, utf8.RuneCountInString(pattern))
	for pos := 0; pos < len(pattern); {
		r, size := utf8.DecodeRuneInString(pattern[pos:])
		pos += size
		if escape != "" && r == escapeRune {
			if pos < len(pattern) {
				r, size = utf8.DecodeRuneInString(pattern[pos:])
				pos += size
			}
			tokens = append(tokens, LikeToken{Rune: r})
			continue
		}
		kind := LikeLiteral
		if r == '%' {
			kind = LikeMany
		} else if r == '_' {
			kind = LikeAny
		}
		tokens = append(tokens, LikeToken{Rune: r, Kind: kind})
	}
	return tokens
}

// LikeLiteralRuns never includes an escape marker or a wildcard as text.
func LikeLiteralRuns(tokens []LikeToken) []string {
	var runs []string
	var run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			runs = append(runs, run.String())
			run.Reset()
		}
	}
	for _, token := range tokens {
		if token.Kind == LikeLiteral {
			run.WriteRune(token.Rune)
		} else {
			flush()
		}
	}
	flush()
	return runs
}

// MatchLikeTokens uses an invocation-local cursor. The caller supplies either
// a plain UTF-8 string or a compressed-text reader without decoding the column.
func MatchLikeTokens(tokens []LikeToken, fold bool, length int, at func(int) (rune, int)) bool {
	pos, p, star, retry := 0, 0, -1, 0
	for pos < length {
		if p < len(tokens) && tokens[p].Kind == LikeMany {
			star = p
			p++
			retry = pos
			continue
		}
		r, size := at(pos)
		if p < len(tokens) {
			token := tokens[p]
			literal := token.Rune
			if fold {
				r = unicode.ToLower(r)
				literal = unicode.ToLower(literal)
			}
			if token.Kind == LikeAny || token.Kind == LikeLiteral && r == literal {
				pos += size
				p++
				continue
			}
		}
		if star < 0 {
			return false
		}
		_, size = at(retry)
		retry += size
		pos = retry
		p = star + 1
	}
	for p < len(tokens) && tokens[p].Kind == LikeMany {
		p++
	}
	return p == len(tokens)
}

func matchLikeString(value string, tokens []LikeToken, fold bool) bool {
	return MatchLikeTokens(tokens, fold, len(value), func(pos int) (rune, int) { return utf8.DecodeRuneInString(value[pos:]) })
}
