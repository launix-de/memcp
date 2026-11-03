/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "strings"
import "unicode"

// TextSearchWords defines word boundaries for the generic weighted-text scorer.
// Frontends own query syntax and lower it to immutable weighted term tuples.
func TextSearchWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '\''
	})
}

// TextSearchScore evaluates tuples (weight, required, excluded, kind, body,
// prefix). The original query string is never parsed in the row loop.
func TextSearchScore(values, terms Scmer) Scmer {
	if terms.IsNil() {
		return NewNil()
	}
	columns := make([][]string, len(values.Slice()))
	for i, value := range values.Slice() {
		if !value.IsNil() {
			columns[i] = TextSearchWords(String(value))
		}
	}
	return NewFloat(textSearchGroup(columns, terms))
}

func textSearchGroup(columns [][]string, terms Scmer) float64 {
	var total float64
	positive := false
	for _, term := range terms.Slice() {
		t := term.Slice()
		var score float64
		switch t[3].String() {
		case "group":
			score = textSearchGroup(columns, t[4])
		case "word":
			needle := t[4].String()
			for _, words := range columns {
				for _, word := range words {
					if word == needle || t[5].Bool() && strings.HasPrefix(word, needle) {
						score++
					}
				}
			}
		case "phrase":
			phrase := t[4].Slice()
			if len(phrase) == 0 {
				break
			}
			for _, words := range columns {
				for start := 0; start+len(phrase) <= len(words); start++ {
					matches := true
					for i, word := range phrase {
						if words[start+i] != word.String() {
							matches = false
							break
						}
					}
					if matches {
						score++
					}
				}
			}
		default:
			panic("unknown text search term")
		}
		if t[2].Bool() && score > 0 || t[1].Bool() && score <= 0 {
			return 0
		}
		weight := ToFloat(t[0])
		if weight > 0 && score > 0 {
			positive = true
		}
		total += weight * score
	}
	if !positive {
		return 0
	}
	return total
}

func initTextSearch() {
	Declare(&Globalenv, &Declaration{
		Name: "text_search_score",
		Fn:   func(args ...Scmer) Scmer { return TextSearchScore(args[0], args[1]) },
		Type: &TypeDescriptor{
			Kind: "func",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "values"},
				{Kind: "list", Label: "terms"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,
		},
	})
}
