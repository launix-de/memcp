/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type jsonPathLeg struct {
	kind  byte
	key   string
	index int
}

const (
	jsonPathKey       = 'k'
	jsonPathIndex     = 'i'
	jsonPathWildcard  = '*'
	jsonPathRecursive = 'r'
)

func jsonPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func requireJSONArgs(name string, args []Scmer, min int) {
	if len(args) < min {
		panic(fmt.Sprintf("%s expects at least %d arguments", name, min))
	}
}

func jsonDocumentArgument(value Scmer) (any, bool) {
	if value.IsNil() {
		return nil, false
	}
	if value.IsBSON() {
		return bsonDecoded(value), true
	}
	if value.IsString() || value.IsSymbol() {
		parsed, err := parseJSONText(value.String())
		jsonPanic(err)
		return normalizeBSONInput(parsed), true
	}
	return jsonConstructorArgument(value), true
}

func jsonConstructorArgument(value Scmer) any {
	switch {
	case value.IsNil():
		return nil
	case value.IsBSON():
		return bsonDecoded(value)
	case value.IsBool():
		return value.Bool()
	case value.IsInt():
		return value.Int()
	case value.IsFloat():
		return value.Float()
	case value.IsString() || value.IsSymbol():
		return value.String()
	default:
		return value.String()
	}
}

func jsonResult(value any) Scmer {
	result, err := bsonFromGo(value)
	jsonPanic(err)
	return result
}

func jsonAggregateResult(value any, flag uint64) Scmer {
	result, err := bsonFromGoFlags(value, flag)
	jsonPanic(err)
	return result
}

func jsonObjectLookup(value any, key string) (any, bool) {
	document, ok := value.(bson.D)
	if !ok {
		return nil, false
	}
	for i := len(document) - 1; i >= 0; i-- {
		if document[i].Key == key {
			return document[i].Value, true
		}
	}
	return nil, false
}

func jsonObjectSet(value bson.D, key string, replacement any, insert, replace bool) (bson.D, bool) {
	copyValue := append(bson.D(nil), value...)
	for i := len(copyValue) - 1; i >= 0; i-- {
		if copyValue[i].Key == key {
			if insert {
				return copyValue, false
			}
			copyValue[i].Value = replacement
			return copyValue, true
		}
	}
	if replace {
		return copyValue, false
	}
	copyValue = append(copyValue, bson.E{Key: key, Value: replacement})
	sort.SliceStable(copyValue, func(i, j int) bool { return copyValue[i].Key < copyValue[j].Key })
	return copyValue, true
}

func parseJSONPath(path string) ([]jsonPathLeg, error) {
	if path == "" || path[0] != '$' {
		return nil, fmt.Errorf("invalid JSON path %q", path)
	}
	legs := make([]jsonPathLeg, 0, 4)
	for i := 1; i < len(path); {
		if strings.HasPrefix(path[i:], "**") {
			legs = append(legs, jsonPathLeg{kind: jsonPathRecursive})
			i += 2
			continue
		}
		switch path[i] {
		case '.':
			i++
			if i >= len(path) {
				return nil, fmt.Errorf("invalid JSON path %q", path)
			}
			if path[i] == '*' {
				legs = append(legs, jsonPathLeg{kind: jsonPathWildcard})
				i++
				continue
			}
			if path[i] == '"' {
				start := i
				i++
				for i < len(path) && path[i] != '"' {
					if path[i] == '\\' {
						i++
					}
					i++
				}
				if i >= len(path) {
					return nil, fmt.Errorf("unterminated JSON path key")
				}
				var key string
				if err := json.Unmarshal([]byte(path[start:i+1]), &key); err != nil {
					return nil, err
				}
				legs = append(legs, jsonPathLeg{kind: jsonPathKey, key: key})
				i++
				continue
			}
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' && path[i] != '*' {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("empty JSON path key")
			}
			legs = append(legs, jsonPathLeg{kind: jsonPathKey, key: path[start:i]})
		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unterminated JSON array subscript")
			}
			content := strings.TrimSpace(path[i+1 : i+end])
			i += end + 1
			switch {
			case content == "*":
				legs = append(legs, jsonPathLeg{kind: jsonPathWildcard})
			case content == "last":
				legs = append(legs, jsonPathLeg{kind: jsonPathIndex, index: -1})
			case strings.HasPrefix(content, "last-"):
				offset, err := strconv.Atoi(content[5:])
				if err != nil {
					return nil, fmt.Errorf("invalid JSON array subscript %q", content)
				}
				legs = append(legs, jsonPathLeg{kind: jsonPathIndex, index: -1 - offset})
			case strings.HasPrefix(content, "\"") || strings.HasPrefix(content, "'"):
				quoted := content
				if quoted[0] == '\'' {
					quoted = "\"" + strings.ReplaceAll(strings.Trim(quoted, "'"), "\"", "\\\"") + "\""
				}
				var key string
				if err := json.Unmarshal([]byte(quoted), &key); err != nil {
					return nil, err
				}
				legs = append(legs, jsonPathLeg{kind: jsonPathKey, key: key})
			default:
				index, err := strconv.Atoi(content)
				if err != nil || index < 0 {
					return nil, fmt.Errorf("invalid JSON array subscript %q", content)
				}
				legs = append(legs, jsonPathLeg{kind: jsonPathIndex, index: index})
			}
		default:
			return nil, fmt.Errorf("invalid JSON path %q at byte %d", path, i)
		}
	}
	return legs, nil
}

func jsonChildren(value any) []any {
	switch value := value.(type) {
	case bson.D:
		result := make([]any, len(value))
		for i := range value {
			result[i] = value[i].Value
		}
		return result
	case bson.A:
		return []any(value)
	default:
		return nil
	}
}

func jsonPathMatches(values []any, legs []jsonPathLeg) []any {
	if len(legs) == 0 {
		return values
	}
	leg := legs[0]
	remaining := legs[1:]
	next := make([]any, 0, len(values))
	for _, value := range values {
		switch leg.kind {
		case jsonPathKey:
			if child, ok := jsonObjectLookup(value, leg.key); ok {
				next = append(next, child)
			}
		case jsonPathIndex:
			if array, ok := value.(bson.A); ok {
				index := leg.index
				if index < 0 {
					index = len(array) + index
				}
				if index >= 0 && index < len(array) {
					next = append(next, array[index])
				}
			}
		case jsonPathWildcard:
			next = append(next, jsonChildren(value)...)
		case jsonPathRecursive:
			next = append(next, value)
			var visit func(any)
			visit = func(parent any) {
				for _, child := range jsonChildren(parent) {
					next = append(next, child)
					visit(child)
				}
			}
			visit(value)
		}
	}
	return jsonPathMatches(next, remaining)
}

func jsonExtractValues(document any, paths []Scmer) ([]any, bool) {
	result := make([]any, 0, len(paths))
	multiple := len(paths) > 1
	for _, pathArg := range paths {
		if pathArg.IsNil() {
			return nil, false
		}
		legs, err := parseJSONPath(pathArg.String())
		jsonPanic(err)
		matches := jsonPathMatches([]any{document}, legs)
		if len(matches) > 1 {
			multiple = true
		}
		result = append(result, matches...)
	}
	return result, multiple
}

func jsonNumeric(value any) (float64, bool) {
	switch value := value.(type) {
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case float64:
		return value, true
	case bson.Decimal128:
		parsed, err := strconv.ParseFloat(value.String(), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func jsonEqual(a, b any) bool {
	if an, ok := jsonNumeric(a); ok {
		bn, bok := jsonNumeric(b)
		return bok && an == bn
	}
	switch a := a.(type) {
	case nil:
		return b == nil
	case string:
		bv, ok := b.(string)
		return ok && a == bv
	case bool:
		bv, ok := b.(bool)
		return ok && a == bv
	case bson.A:
		bv, ok := b.(bson.A)
		if !ok || len(a) != len(bv) {
			return false
		}
		for i := range a {
			if !jsonEqual(a[i], bv[i]) {
				return false
			}
		}
		return true
	case bson.D:
		bv, ok := b.(bson.D)
		if !ok || len(a) != len(bv) {
			return false
		}
		for _, pair := range a {
			other, found := jsonObjectLookup(bv, pair.Key)
			if !found || !jsonEqual(pair.Value, other) {
				return false
			}
		}
		return true
	default:
		return fmt.Sprint(a) == fmt.Sprint(b)
	}
}

func jsonLess(a, b any) bool {
	if an, ok := jsonNumeric(a); ok {
		if bn, bok := jsonNumeric(b); bok {
			return an < bn
		}
	}
	rank := func(value any) int {
		switch value.(type) {
		case nil:
			return 0
		case bool:
			return 1
		case int32, int64, float64, bson.Decimal128:
			return 2
		case string:
			return 3
		case bson.A:
			return 4
		case bson.D:
			return 5
		default:
			return 6
		}
	}
	leftRank, rightRank := rank(a), rank(b)
	if leftRank != rightRank {
		return leftRank < rightRank
	}
	switch left := a.(type) {
	case bool:
		right, _ := b.(bool)
		return !left && right
	case string:
		right, _ := b.(string)
		return left < right
	default:
		leftJSON, _ := appendOrdinaryJSON(nil, a, false, "")
		rightJSON, _ := appendOrdinaryJSON(nil, b, false, "")
		return string(leftJSON) < string(rightJSON)
	}
}

func jsonContainsValue(target, candidate any) bool {
	switch candidate := candidate.(type) {
	case bson.D:
		targetObject, ok := target.(bson.D)
		if !ok {
			return false
		}
		for _, pair := range candidate {
			value, found := jsonObjectLookup(targetObject, pair.Key)
			if !found || !jsonContainsValue(value, pair.Value) {
				return false
			}
		}
		return true
	case bson.A:
		targetArray, ok := target.(bson.A)
		if !ok {
			return false
		}
		for _, wanted := range candidate {
			found := false
			for _, available := range targetArray {
				if jsonContainsValue(available, wanted) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		if targetArray, ok := target.(bson.A); ok {
			for _, available := range targetArray {
				if jsonEqual(available, candidate) {
					return true
				}
			}
			return false
		}
		return jsonEqual(target, candidate)
	}
}

func jsonDepthValue(value any) int64 {
	maxChild := int64(0)
	children := jsonChildren(value)
	for _, child := range children {
		if depth := jsonDepthValue(child); depth > maxChild {
			maxChild = depth
		}
	}
	return 1 + maxChild
}

func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "NULL"
	case bool:
		return "BOOLEAN"
	case int32, int64:
		return "INTEGER"
	case float64:
		return "DOUBLE"
	case bson.Decimal128:
		return "DECIMAL"
	case string:
		return "STRING"
	case bson.A:
		return "ARRAY"
	case bson.D:
		return "OBJECT"
	default:
		return "OPAQUE"
	}
}

func jsonModify(value any, legs []jsonPathLeg, replacement any, mode byte) (any, bool) {
	if len(legs) == 0 {
		switch mode {
		case 'i':
			return value, false
		case 's', 'r':
			return replacement, true
		}
	}
	leg := legs[0]
	if leg.kind != jsonPathKey && leg.kind != jsonPathIndex {
		panic("JSON modification paths cannot contain wildcards")
	}
	if len(legs) == 1 {
		switch leg.kind {
		case jsonPathKey:
			document, ok := value.(bson.D)
			if !ok {
				return value, false
			}
			return jsonObjectSet(document, leg.key, replacement, mode == 'i', mode == 'r')
		case jsonPathIndex:
			array, ok := value.(bson.A)
			if !ok {
				return value, false
			}
			index := leg.index
			if index < 0 {
				index = len(array) + index
			}
			copyValue := append(bson.A(nil), array...)
			if index >= 0 && index < len(array) {
				if mode == 'i' {
					return copyValue, false
				}
				copyValue[index] = replacement
				return copyValue, true
			}
			if mode != 'r' && index >= len(array) {
				return append(copyValue, replacement), true
			}
		}
		return value, false
	}
	switch leg.kind {
	case jsonPathKey:
		document, ok := value.(bson.D)
		if !ok {
			return value, false
		}
		child, found := jsonObjectLookup(document, leg.key)
		if !found {
			return value, false
		}
		modified, changed := jsonModify(child, legs[1:], replacement, mode)
		if !changed {
			return value, false
		}
		result, _ := jsonObjectSet(document, leg.key, modified, false, false)
		return result, true
	case jsonPathIndex:
		array, ok := value.(bson.A)
		if !ok {
			return value, false
		}
		index := leg.index
		if index < 0 {
			index = len(array) + index
		}
		if index < 0 || index >= len(array) {
			return value, false
		}
		modified, changed := jsonModify(array[index], legs[1:], replacement, mode)
		if !changed {
			return value, false
		}
		copyValue := append(bson.A(nil), array...)
		copyValue[index] = modified
		return copyValue, true
	}
	return value, false
}

func jsonRemoveValue(value any, legs []jsonPathLeg) (any, bool) {
	if len(legs) == 0 {
		panic("JSON_REMOVE cannot remove the root value")
	}
	leg := legs[0]
	if leg.kind != jsonPathKey && leg.kind != jsonPathIndex {
		panic("JSON_REMOVE paths cannot contain wildcards")
	}
	if len(legs) == 1 {
		if leg.kind == jsonPathKey {
			document, ok := value.(bson.D)
			if !ok {
				return value, false
			}
			result := make(bson.D, 0, len(document))
			removed := false
			for _, pair := range document {
				if pair.Key == leg.key {
					removed = true
					continue
				}
				result = append(result, pair)
			}
			return result, removed
		}
		array, ok := value.(bson.A)
		if !ok {
			return value, false
		}
		index := leg.index
		if index < 0 {
			index = len(array) + index
		}
		if index < 0 || index >= len(array) {
			return value, false
		}
		result := append(bson.A(nil), array[:index]...)
		result = append(result, array[index+1:]...)
		return result, true
	}
	child, found := any(nil), false
	if leg.kind == jsonPathKey {
		child, found = jsonObjectLookup(value, leg.key)
	} else if array, ok := value.(bson.A); ok {
		index := leg.index
		if index < 0 {
			index = len(array) + index
		}
		if index >= 0 && index < len(array) {
			child, found = array[index], true
		}
	}
	if !found {
		return value, false
	}
	modified, changed := jsonRemoveValue(child, legs[1:])
	if !changed {
		return value, false
	}
	return jsonModify(value, []jsonPathLeg{leg}, modified, 's')
}

func jsonMergePatch(target, patch any) any {
	patchObject, ok := patch.(bson.D)
	if !ok {
		return patch
	}
	targetObject, ok := target.(bson.D)
	if !ok {
		targetObject = bson.D{}
	}
	result := append(bson.D(nil), targetObject...)
	for _, pair := range patchObject {
		if pair.Value == nil {
			removed, _ := jsonRemoveValue(result, []jsonPathLeg{{kind: jsonPathKey, key: pair.Key}})
			result = removed.(bson.D)
			continue
		}
		old, found := jsonObjectLookup(result, pair.Key)
		value := pair.Value
		if found {
			value = jsonMergePatch(old, pair.Value)
		}
		result, _ = jsonObjectSet(result, pair.Key, value, false, false)
	}
	return result
}

func jsonMergePreserve(a, b any) any {
	aa, aArray := a.(bson.A)
	ba, bArray := b.(bson.A)
	if aArray && bArray {
		return append(append(bson.A(nil), aa...), ba...)
	}
	ad, aObject := a.(bson.D)
	bd, bObject := b.(bson.D)
	if aObject && bObject {
		result := append(bson.D(nil), ad...)
		for _, pair := range bd {
			if old, found := jsonObjectLookup(result, pair.Key); found {
				result, _ = jsonObjectSet(result, pair.Key, jsonMergePreserve(old, pair.Value), false, false)
			} else {
				result, _ = jsonObjectSet(result, pair.Key, pair.Value, false, false)
			}
		}
		return result
	}
	if !aArray {
		aa = bson.A{a}
	}
	if bArray {
		return append(aa, ba...)
	}
	return append(aa, b)
}

func jsonSQLLike(value, pattern, escape string) bool {
	var expression strings.Builder
	expression.WriteByte('^')
	escaping := false
	for _, character := range pattern {
		if !escaping && escape != "" && string(character) == escape {
			escaping = true
			continue
		}
		if !escaping && character == '%' {
			expression.WriteString(".*")
		} else if !escaping && character == '_' {
			expression.WriteByte('.')
		} else {
			expression.WriteString(regexp.QuoteMeta(string(character)))
		}
		escaping = false
	}
	expression.WriteByte('$')
	matched, err := regexp.MatchString(expression.String(), value)
	return err == nil && matched
}

func jsonSearchPaths(value any, pattern, escape, path string, result *[]string) {
	switch value := value.(type) {
	case string:
		if jsonSQLLike(value, pattern, escape) {
			*result = append(*result, path)
		}
	case bson.A:
		for i, child := range value {
			jsonSearchPaths(child, pattern, escape, path+"["+strconv.Itoa(i)+"]", result)
		}
	case bson.D:
		for _, pair := range value {
			key, _ := json.Marshal(pair.Key)
			jsonSearchPaths(pair.Value, pattern, escape, path+"."+string(key), result)
		}
	}
}

func jsonScalarResult(value any, returning string) Scmer {
	if value == nil {
		return NewNil()
	}
	returning = strings.ToUpper(returning)
	switch returning {
	case "SIGNED", "INTEGER", "INT", "BIGINT", "UNSIGNED":
		if number, ok := jsonNumeric(value); ok {
			return NewInt(int64(number))
		}
		return NewInt(numericPrefixInt(fmt.Sprint(value)))
	case "FLOAT", "DOUBLE", "REAL", "DECIMAL", "NUMERIC":
		if number, ok := jsonNumeric(value); ok {
			return NewFloat(number)
		}
		return NewFloat(numericPrefixFloat(fmt.Sprint(value)))
	case "BOOLEAN", "BOOL":
		if boolean, ok := value.(bool); ok {
			return NewBool(boolean)
		}
		return NewBool(fmt.Sprint(value) != "" && fmt.Sprint(value) != "0")
	case "JSON":
		return jsonResult(value)
	default:
		switch value := value.(type) {
		case string:
			return NewString(value)
		case bool:
			if value {
				return NewString("true")
			}
			return NewString("false")
		case int32:
			return NewString(strconv.FormatInt(int64(value), 10))
		case int64:
			return NewString(strconv.FormatInt(value, 10))
		case float64:
			return NewString(strconv.FormatFloat(value, 'g', -1, 64))
		case bson.Decimal128:
			return NewString(value.String())
		default:
			panic("JSON_VALUE result is not scalar")
		}
	}
}

func declareJSON(name, description string, fn func(...Scmer) Scmer) {
	Declare(&Globalenv, &Declaration{
		Name: name,
		Desc: description,
		Fn:   fn,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{{Kind: "any", ParamName: "arguments", Variadic: true}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,
		},
	})
}

func init_json_functions() {
	declareJSON("json_parse_bson", "validates JSON and returns a BSON-backed value", func(args ...Scmer) Scmer {
		requireJSONArgs("json_parse_bson", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		value, err := NewBSONFromJSON(args[0].String())
		jsonPanic(err)
		return value
	})
	declareJSON("json_array", "creates a MySQL-compatible JSON array", func(args ...Scmer) Scmer {
		array := make(bson.A, len(args))
		for i := range args {
			array[i] = jsonConstructorArgument(args[i])
		}
		return jsonResult(array)
	})
	declareJSON("json_object", "creates a MySQL-compatible JSON object", func(args ...Scmer) Scmer {
		if len(args)%2 != 0 {
			panic("JSON_OBJECT expects key/value pairs")
		}
		document := make(bson.D, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if args[i].IsNil() {
				panic("JSON_OBJECT key cannot be NULL")
			}
			document, _ = jsonObjectSet(document, args[i].String(), jsonConstructorArgument(args[i+1]), false, false)
		}
		return jsonResult(document)
	})
	declareJSON("json_quote", "quotes a string as JSON", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_QUOTE", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		encoded, _ := json.Marshal(args[0].String())
		return NewString(string(encoded))
	})
	declareJSON("json_unquote", "unquotes a JSON string", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_UNQUOTE", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		if args[0].IsBSON() {
			decoded := bsonDecoded(args[0])
			if text, ok := decoded.(string); ok {
				return NewString(text)
			}
			return NewString(bsonText(args[0]))
		}
		text := args[0].String()
		if len(text) > 0 && text[0] == '"' {
			var decoded string
			jsonPanic(json.Unmarshal([]byte(text), &decoded))
			return NewString(decoded)
		}
		return NewString(text)
	})
	declareJSON("json_valid", "tests whether a value contains valid JSON", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_VALID", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		if args[0].IsBSON() {
			return NewBool(bsonRawValue(args[0]).Validate() == nil)
		}
		_, err := parseJSONText(args[0].String())
		return NewBool(err == nil)
	})
	declareJSON("json_type", "returns the MySQL JSON type name", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_TYPE", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		return NewString(jsonTypeName(value))
	})
	declareJSON("json_depth", "returns maximum JSON document depth", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_DEPTH", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		return NewInt(jsonDepthValue(value))
	})
	declareJSON("json_length", "returns JSON object or array length", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_LENGTH", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		if len(args) > 1 {
			matches, _ := jsonExtractValues(value, args[1:2])
			if len(matches) == 0 {
				return NewNil()
			}
			value = matches[0]
		}
		switch value := value.(type) {
		case bson.A:
			return NewInt(int64(len(value)))
		case bson.D:
			return NewInt(int64(len(value)))
		default:
			return NewInt(1)
		}
	})
	declareJSON("json_extract", "extracts one or more JSON paths", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_EXTRACT", args, 2)
		document, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		matches, multiple := jsonExtractValues(document, args[1:])
		if len(matches) == 0 {
			return NewNil()
		}
		if multiple {
			return jsonResult(bson.A(matches))
		}
		return jsonResult(matches[0])
	})
	declareJSON("json_keys", "returns object member names", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_KEYS", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		if len(args) > 1 {
			matches, _ := jsonExtractValues(value, args[1:2])
			if len(matches) == 0 {
				return NewNil()
			}
			value = matches[0]
		}
		document, ok := value.(bson.D)
		if !ok {
			return NewNil()
		}
		keys := make(bson.A, len(document))
		for i := range document {
			keys[i] = document[i].Key
		}
		return jsonResult(keys)
	})
	declareJSON("json_contains_path", "tests whether one or all JSON paths exist", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_CONTAINS_PATH", args, 3)
		document, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		mode := strings.ToLower(args[1].String())
		if mode != "one" && mode != "all" {
			panic("JSON_CONTAINS_PATH mode must be 'one' or 'all'")
		}
		found := mode == "all"
		for _, path := range args[2:] {
			if path.IsNil() {
				return NewNil()
			}
			legs, err := parseJSONPath(path.String())
			jsonPanic(err)
			exists := len(jsonPathMatches([]any{document}, legs)) > 0
			if mode == "one" && exists {
				return NewBool(true)
			}
			if mode == "all" && !exists {
				return NewBool(false)
			}
		}
		return NewBool(found)
	})
	declareJSON("json_contains", "tests JSON containment", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_CONTAINS", args, 2)
		target, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		candidate, ok := jsonDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		if len(args) > 2 {
			matches, _ := jsonExtractValues(target, args[2:3])
			if len(matches) == 0 {
				return NewBool(false)
			}
			target = matches[0]
		}
		return NewBool(jsonContainsValue(target, candidate))
	})
	declareJSON("json_overlaps", "tests whether JSON values overlap", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OVERLAPS", args, 2)
		left, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		right, ok := jsonDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		if la, ok := left.(bson.A); ok {
			for _, lv := range la {
				if ra, rok := right.(bson.A); rok {
					for _, rv := range ra {
						if jsonEqual(lv, rv) {
							return NewBool(true)
						}
					}
				} else if jsonEqual(lv, right) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		}
		if ld, ok := left.(bson.D); ok {
			rd, rok := right.(bson.D)
			if !rok {
				return NewBool(false)
			}
			for _, pair := range ld {
				if rv, found := jsonObjectLookup(rd, pair.Key); found && jsonEqual(pair.Value, rv) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		}
		return NewBool(jsonEqual(left, right))
	})
	declareJSON("json_member_of", "tests whether a JSON scalar is a member of an array", func(args ...Scmer) Scmer {
		requireJSONArgs("MEMBER OF", args, 2)
		left := jsonConstructorArgument(args[0])
		right, ok := jsonDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		array, ok := right.(bson.A)
		if !ok {
			panic("MEMBER OF requires a JSON array")
		}
		for _, item := range array {
			if jsonEqual(left, item) {
				return NewBool(true)
			}
		}
		return NewBool(false)
	})
	declareJSON("json_pretty", "pretty-prints JSON", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_PRETTY", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		encoded, err := appendOrdinaryJSON(nil, value, true, "")
		jsonPanic(err)
		return NewString(string(encoded))
	})
	declareJSON("json_storage_size", "returns BSON payload size", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_STORAGE_SIZE", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		value := args[0]
		if !value.IsBSON() {
			parsed, err := NewBSONFromJSON(value.String())
			jsonPanic(err)
			value = parsed
		}
		_, payload := bsonTypeAndBytes(value)
		return NewInt(int64(len(payload) + 1))
	})
	declareJSON("json_storage_free", "returns unused BSON storage space", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_STORAGE_FREE", args, 1)
		if args[0].IsNil() {
			return NewNil()
		}
		return NewInt(0)
	})
	modify := func(mode byte, name string, args ...Scmer) Scmer {
		requireJSONArgs(name, args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON modification expects path/value pairs")
		}
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for i := 1; i < len(args); i += 2 {
			if args[i].IsNil() {
				return NewNil()
			}
			legs, err := parseJSONPath(args[i].String())
			jsonPanic(err)
			value, _ = jsonModify(value, legs, jsonConstructorArgument(args[i+1]), mode)
		}
		return jsonResult(value)
	}
	declareJSON("json_set", "sets or inserts JSON path values", func(args ...Scmer) Scmer {
		return modify('s', "JSON_SET", args...)
	})
	declareJSON("json_insert", "inserts missing JSON path values", func(args ...Scmer) Scmer {
		return modify('i', "JSON_INSERT", args...)
	})
	declareJSON("json_replace", "replaces existing JSON path values", func(args ...Scmer) Scmer {
		return modify('r', "JSON_REPLACE", args...)
	})
	declareJSON("json_remove", "removes JSON path values", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_REMOVE", args, 2)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, path := range args[1:] {
			if path.IsNil() {
				return NewNil()
			}
			legs, err := parseJSONPath(path.String())
			jsonPanic(err)
			value, _ = jsonRemoveValue(value, legs)
		}
		return jsonResult(value)
	})
	declareJSON("json_array_append", "appends values to JSON arrays", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAY_APPEND", args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON_ARRAY_APPEND expects path/value pairs")
		}
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for i := 1; i < len(args); i += 2 {
			legs, err := parseJSONPath(args[i].String())
			jsonPanic(err)
			matches := jsonPathMatches([]any{value}, legs)
			if len(matches) == 0 {
				continue
			}
			array, arrayOK := matches[0].(bson.A)
			if !arrayOK {
				array = bson.A{matches[0]}
			}
			array = append(array, jsonConstructorArgument(args[i+1]))
			value, _ = jsonModify(value, legs, array, 's')
		}
		return jsonResult(value)
	})
	declareJSON("json_array_insert", "inserts values into JSON arrays", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAY_INSERT", args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON_ARRAY_INSERT expects path/value pairs")
		}
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for i := 1; i < len(args); i += 2 {
			legs, err := parseJSONPath(args[i].String())
			jsonPanic(err)
			if len(legs) == 0 || legs[len(legs)-1].kind != jsonPathIndex {
				panic("JSON_ARRAY_INSERT path must end in an array index")
			}
			parentLegs := legs[:len(legs)-1]
			matches := jsonPathMatches([]any{value}, parentLegs)
			if len(matches) == 0 {
				continue
			}
			array, ok := matches[0].(bson.A)
			if !ok {
				continue
			}
			index := legs[len(legs)-1].index
			if index < 0 {
				index = len(array) + index
			}
			if index < 0 {
				index = 0
			}
			if index > len(array) {
				index = len(array)
			}
			inserted := make(bson.A, 0, len(array)+1)
			inserted = append(inserted, array[:index]...)
			inserted = append(inserted, jsonConstructorArgument(args[i+1]))
			inserted = append(inserted, array[index:]...)
			value, _ = jsonModify(value, parentLegs, inserted, 's')
		}
		return jsonResult(value)
	})
	declareJSON("json_merge_patch", "merges documents using RFC 7396 semantics", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_MERGE_PATCH", args, 2)
		result, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, arg := range args[1:] {
			patch, ok := jsonDocumentArgument(arg)
			if !ok {
				return NewNil()
			}
			result = jsonMergePatch(result, patch)
		}
		return jsonResult(result)
	})
	declareJSON("json_merge_preserve", "merges documents while preserving values", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_MERGE_PRESERVE", args, 2)
		result, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, arg := range args[1:] {
			next, ok := jsonDocumentArgument(arg)
			if !ok {
				return NewNil()
			}
			result = jsonMergePreserve(result, next)
		}
		return jsonResult(result)
	})
	declareJSON("json_search", "searches JSON string values", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_SEARCH", args, 3)
		document, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() || args[2].IsNil() {
			return NewNil()
		}
		mode := strings.ToLower(args[1].String())
		if mode != "one" && mode != "all" {
			panic("JSON_SEARCH mode must be 'one' or 'all'")
		}
		escape := "\\"
		if len(args) > 3 && !args[3].IsNil() {
			escape = args[3].String()
		}
		roots := []struct {
			value any
			path  string
		}{{document, "$"}}
		if len(args) > 4 {
			roots = roots[:0]
			for _, path := range args[4:] {
				legs, err := parseJSONPath(path.String())
				jsonPanic(err)
				for _, match := range jsonPathMatches([]any{document}, legs) {
					roots = append(roots, struct {
						value any
						path  string
					}{match, path.String()})
				}
			}
		}
		paths := make([]string, 0)
		for _, root := range roots {
			jsonSearchPaths(root.value, args[2].String(), escape, root.path, &paths)
		}
		if len(paths) == 0 {
			return NewNil()
		}
		if mode == "one" {
			return jsonResult(paths[0])
		}
		array := make(bson.A, len(paths))
		for i := range paths {
			array[i] = paths[i]
		}
		return jsonResult(array)
	})
	declareJSON("json_value", "extracts a scalar JSON value", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_VALUE", args, 2)
		document, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		matches, _ := jsonExtractValues(document, args[1:2])
		if len(matches) == 0 {
			return NewNil()
		}
		returning := "CHAR"
		if len(args) > 2 {
			returning = args[2].String()
		}
		return jsonScalarResult(matches[0], returning)
	})
	declareJSON("json_get", "implements PostgreSQL JSON object and array extraction", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON extraction operator", args, 2)
		document, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		var value any
		var found bool
		if args[1].IsInt() || args[1].IsFloat() {
			if array, arrayOK := document.(bson.A); arrayOK {
				index := int(args[1].Int())
				if index < 0 {
					index = len(array) + index
				}
				if index >= 0 && index < len(array) {
					value, found = array[index], true
				}
			}
		} else {
			value, found = jsonObjectLookup(document, args[1].String())
		}
		if !found {
			return NewNil()
		}
		if len(args) > 2 && args[2].Bool() {
			return jsonScalarResult(value, "CHAR")
		}
		return jsonResult(value)
	})
	declareJSON("json_arrayagg_entry", "wraps one JSON_ARRAYAGG input value", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAYAGG", args, 1)
		return jsonAggregateResult(bson.A{jsonConstructorArgument(args[0])}, bsonFlagArrayAgg)
	})
	declareJSON("json_arrayagg_reduce", "JSON_ARRAYAGG reducer", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAYAGG reducer", args, 2)
		var leftState, rightState bson.A
		leftIsState := bsonHasFlag(args[0], bsonFlagArrayAgg)
		rightIsState := bsonHasFlag(args[1], bsonFlagArrayAgg)
		if leftIsState {
			leftState, leftIsState = bsonDecoded(args[0]).(bson.A)
		}
		if rightIsState {
			rightState, rightIsState = bsonDecoded(args[1]).(bson.A)
		}
		switch {
		case args[0].IsNil() && rightIsState:
			return args[1]
		case args[1].IsNil() && leftIsState:
			return args[0]
		case args[0].IsNil():
			return jsonAggregateResult(bson.A{jsonConstructorArgument(args[1])}, bsonFlagArrayAgg)
		case args[1].IsNil():
			return jsonAggregateResult(bson.A{jsonConstructorArgument(args[0])}, bsonFlagArrayAgg)
		case leftIsState && rightIsState:
			return jsonAggregateResult(append(append(bson.A(nil), leftState...), rightState...), bsonFlagArrayAgg)
		case leftIsState:
			return jsonAggregateResult(append(append(bson.A(nil), leftState...), jsonConstructorArgument(args[1])), bsonFlagArrayAgg)
		case rightIsState:
			return jsonAggregateResult(append(append(bson.A(nil), rightState...), jsonConstructorArgument(args[0])), bsonFlagArrayAgg)
		default:
			return jsonAggregateResult(bson.A{jsonConstructorArgument(args[0]), jsonConstructorArgument(args[1])}, bsonFlagArrayAgg)
		}
	})
	declareJSON("json_objectagg_entry", "constructs a JSON_OBJECTAGG key/value entry", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OBJECTAGG", args, 2)
		if args[0].IsNil() {
			panic("JSON_OBJECTAGG key cannot be NULL")
		}
		return jsonResult(bson.A{args[0].String(), jsonConstructorArgument(args[1])})
	})
	declareJSON("json_objectagg_reduce", "JSON_OBJECTAGG reducer", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OBJECTAGG reducer", args, 2)
		if args[0].IsNil() && bsonHasFlag(args[1], bsonFlagObjectAgg) {
			return args[1]
		}
		if args[1].IsNil() && bsonHasFlag(args[0], bsonFlagObjectAgg) {
			return args[0]
		}
		var left, right any
		if bsonHasFlag(args[0], bsonFlagObjectAgg) || args[0].IsBSON() {
			left = bsonDecoded(args[0])
		}
		if bsonHasFlag(args[1], bsonFlagObjectAgg) || args[1].IsBSON() {
			right = bsonDecoded(args[1])
		}
		document, leftDocument := left.(bson.D)
		if !leftDocument {
			document, _ = right.(bson.D)
		}
		if document == nil {
			document = bson.D{}
		}
		if other, ok := right.(bson.D); ok && leftDocument && bsonHasFlag(args[0], bsonFlagObjectAgg) && bsonHasFlag(args[1], bsonFlagObjectAgg) {
			for _, pair := range other {
				document, _ = jsonObjectSet(document, pair.Key, pair.Value, false, false)
			}
			return jsonAggregateResult(document, bsonFlagObjectAgg)
		}
		entry, entryOK := left.(bson.A)
		if !entryOK {
			entry, entryOK = right.(bson.A)
		}
		if !entryOK || len(entry) != 2 {
			panic("invalid JSON_OBJECTAGG entry")
		}
		document, _ = jsonObjectSet(document, fmt.Sprint(entry[0]), entry[1], false, false)
		return jsonAggregateResult(document, bsonFlagObjectAgg)
	})
}
