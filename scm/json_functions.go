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

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
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

func jsonBSONDocumentArgument(value Scmer) (Scmer, bool) {
	if value.IsNil() {
		return NewNil(), false
	}
	if value.IsBSON() {
		return value, true
	}
	if value.IsString() || value.IsSymbol() {
		parsed, err := NewBSONFromJSON(value.String())
		jsonPanic(err)
		return parsed, true
	}
	parsed, err := bsonFromSQLScalar(value)
	jsonPanic(err)
	return parsed, true
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

func jsonRawChildren(value bson.RawValue) []bson.RawValue {
	switch value.Type {
	case bson.TypeEmbeddedDocument:
		elements, err := value.Document().Elements()
		jsonPanic(err)
		result := make([]bson.RawValue, len(elements))
		for i := range elements {
			result[i] = elements[i].Value()
		}
		return result
	case bson.TypeArray:
		values, err := value.Array().Values()
		jsonPanic(err)
		return values
	default:
		return nil
	}
}

func jsonRawPathMatches(values []bson.RawValue, legs []jsonPathLeg) []bson.RawValue {
	if len(legs) == 0 {
		return values
	}
	leg := legs[0]
	remaining := legs[1:]
	next := make([]bson.RawValue, 0, len(values))
	for _, value := range values {
		switch leg.kind {
		case jsonPathKey:
			if value.Type == bson.TypeEmbeddedDocument {
				child, err := value.Document().LookupErr(encodeBSONKey(leg.key))
				if err == nil {
					next = append(next, child)
				}
			}
		case jsonPathIndex:
			if value.Type == bson.TypeArray {
				array := value.Array()
				index := leg.index
				if index < 0 {
					values, err := array.Values()
					jsonPanic(err)
					index += len(values)
					if index >= 0 && index < len(values) {
						next = append(next, values[index])
					}
				} else if child, err := array.IndexErr(uint(index)); err == nil {
					next = append(next, child)
				}
			}
		case jsonPathWildcard:
			next = append(next, jsonRawChildren(value)...)
		case jsonPathRecursive:
			var visit func(bson.RawValue)
			visit = func(parent bson.RawValue) {
				next = append(next, parent)
				for _, child := range jsonRawChildren(parent) {
					visit(child)
				}
			}
			visit(value)
		}
	}
	return jsonRawPathMatches(next, remaining)
}

func jsonExtractBSONValues(document Scmer, paths []Scmer) ([]bson.RawValue, bool) {
	result := make([]bson.RawValue, 0, len(paths))
	multiple := len(paths) > 1
	root := bsonRawValue(document)
	for _, pathArg := range paths {
		if pathArg.IsNil() {
			return nil, false
		}
		legs, err := parseJSONPath(pathArg.String())
		jsonPanic(err)
		matches := jsonRawPathMatches([]bson.RawValue{root}, legs)
		if len(matches) > 1 {
			multiple = true
		}
		result = append(result, matches...)
	}
	return result, multiple
}

func jsonRawDepth(value bson.RawValue) int64 {
	depth := int64(1)
	for _, child := range jsonRawChildren(value) {
		if childDepth := 1 + jsonRawDepth(child); childDepth > depth {
			depth = childDepth
		}
	}
	return depth
}

func jsonRawTypeName(value bson.RawValue) string {
	switch value.Type {
	case bson.TypeNull:
		return "NULL"
	case bson.TypeBoolean:
		return "BOOLEAN"
	case bson.TypeInt32, bson.TypeInt64:
		return "INTEGER"
	case bson.TypeDouble:
		return "DOUBLE"
	case bson.TypeDecimal128:
		return "DECIMAL"
	case bson.TypeString:
		return "STRING"
	case bson.TypeArray:
		return "ARRAY"
	case bson.TypeEmbeddedDocument:
		return "OBJECT"
	default:
		return "OPAQUE"
	}
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

func jsonSchemaValidation(schemaArg, documentArg Scmer) (bool, *jsonschema.ValidationError) {
	schemaValue, ok := jsonDocumentArgument(schemaArg)
	if !ok {
		return false, nil
	}
	if _, objectOK := schemaValue.(bson.D); !objectOK {
		panic("JSON schema must be an object")
	}
	documentValue, ok := jsonDocumentArgument(documentArg)
	if !ok {
		return false, nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft4)
	if err := compiler.AddResource("memcp://mysql-json-schema", bsonToJSONGo(schemaValue)); err != nil {
		panic(err)
	}
	compiled, err := compiler.Compile("memcp://mysql-json-schema")
	if err != nil {
		panic(err)
	}
	if err := compiled.Validate(bsonToJSONGo(documentValue)); err != nil {
		validationError, ok := err.(*jsonschema.ValidationError)
		if !ok {
			panic(err)
		}
		return false, validationError
	}
	return true, nil
}

func deepestJSONSchemaError(err *jsonschema.ValidationError) *jsonschema.ValidationError {
	for len(err.Causes) > 0 {
		err = err.Causes[0]
	}
	return err
}

func jsonPointer(parts []string) string {
	escaped := make([]string, len(parts))
	for i, part := range parts {
		escaped[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1")
	}
	if len(escaped) == 0 {
		return "#"
	}
	return "#/" + strings.Join(escaped, "/")
}

func pgTextArray(value Scmer) []string {
	if value.IsBSON() {
		raw := bsonRawValue(value)
		if raw.Type == bson.TypeArray {
			array := mustBSONValues(raw)
			result := make([]string, len(array))
			for i := range array {
				if array[i].Type == bson.TypeString {
					result[i] = array[i].StringValue()
				} else {
					encoded, err := appendBSONJSON(nil, array[i], false, "", 0)
					jsonPanic(err)
					result[i] = string(encoded)
				}
			}
			return result
		}
	}
	text := strings.TrimSpace(value.String())
	if len(text) < 2 || text[0] != '{' || text[len(text)-1] != '}' {
		return []string{text}
	}
	text = text[1 : len(text)-1]
	if text == "" {
		return nil
	}
	parts := make([]string, 0, strings.Count(text, ",")+1)
	var current strings.Builder
	quoted, escaped := false, false
	for _, r := range text {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			quoted = !quoted
		case r == ',' && !quoted:
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	parts = append(parts, strings.TrimSpace(current.String()))
	return parts
}

func pgPathLegs(value Scmer) []jsonPathLeg {
	parts := pgTextArray(value)
	legs := make([]jsonPathLeg, len(parts))
	for i, part := range parts {
		if index, err := strconv.Atoi(part); err == nil {
			legs[i] = jsonPathLeg{kind: jsonPathIndex, index: index}
		} else {
			legs[i] = jsonPathLeg{kind: jsonPathKey, key: part}
		}
	}
	return legs
}

func pgExtractPath(document Scmer, paths []Scmer, textResult bool) Scmer {
	value, ok := jsonBSONDocumentArgument(document)
	if !ok {
		return NewNil()
	}
	if len(paths) == 1 && !paths[0].IsInt() && !paths[0].IsFloat() && strings.HasPrefix(strings.TrimSpace(paths[0].String()), "{") {
		parts := pgTextArray(paths[0])
		paths = make([]Scmer, len(parts))
		for i, part := range parts {
			paths[i] = NewString(part)
		}
	}
	legs := make([]jsonPathLeg, len(paths))
	for i, path := range paths {
		if path.IsInt() || path.IsFloat() {
			legs[i] = jsonPathLeg{kind: jsonPathIndex, index: int(path.Int())}
		} else if index, err := strconv.Atoi(path.String()); err == nil {
			legs[i] = jsonPathLeg{kind: jsonPathIndex, index: index}
		} else {
			legs[i] = jsonPathLeg{kind: jsonPathKey, key: path.String()}
		}
	}
	matches := jsonRawPathMatches([]bson.RawValue{bsonRawValue(value)}, legs)
	if len(matches) == 0 {
		return NewNil()
	}
	if textResult {
		if matches[0].Type == bson.TypeArray || matches[0].Type == bson.TypeEmbeddedDocument {
			encoded, err := appendBSONJSON(nil, matches[0], false, "", 0)
			jsonPanic(err)
			return NewString(string(encoded))
		}
		return jsonRawScalarResult(matches[0], "CHAR")
	}
	return NewBSONValue(matches[0].Type, matches[0].Value)
}

func pgStripJSONNulls(value any, arrays bool) any {
	switch value := value.(type) {
	case bson.D:
		result := make(bson.D, 0, len(value))
		for _, pair := range value {
			if pair.Value != nil {
				result = append(result, bson.E{Key: pair.Key, Value: pgStripJSONNulls(pair.Value, arrays)})
			}
		}
		return result
	case bson.A:
		result := make(bson.A, 0, len(value))
		for _, item := range value {
			if item != nil || !arrays {
				result = append(result, pgStripJSONNulls(item, arrays))
			}
		}
		return result
	default:
		return value
	}
}

func pgJSONBConcat(left, right any) any {
	if leftObject, ok := left.(bson.D); ok {
		if rightObject, rightOK := right.(bson.D); rightOK {
			result := append(bson.D(nil), leftObject...)
			for _, pair := range rightObject {
				result, _ = jsonObjectSet(result, pair.Key, pair.Value, false, false)
			}
			return result
		}
	}
	leftArray, leftOK := left.(bson.A)
	if !leftOK {
		leftArray = bson.A{left}
	}
	rightArray, rightOK := right.(bson.A)
	if !rightOK {
		rightArray = bson.A{right}
	}
	return append(append(bson.A(nil), leftArray...), rightArray...)
}

func pgJSONBDelete(value any, operand Scmer) any {
	if document, ok := value.(bson.D); ok {
		keys := pgTextArray(operand)
		remove := make(map[string]bool, len(keys))
		for _, key := range keys {
			remove[key] = true
		}
		result := make(bson.D, 0, len(document))
		for _, pair := range document {
			if !remove[pair.Key] {
				result = append(result, pair)
			}
		}
		return result
	}
	if array, ok := value.(bson.A); ok {
		if operand.IsInt() || operand.IsFloat() {
			index := int(operand.Int())
			if index < 0 {
				index += len(array)
			}
			if index < 0 || index >= len(array) {
				return array
			}
			return append(append(bson.A(nil), array[:index]...), array[index+1:]...)
		}
		remove := make(map[string]bool)
		for _, text := range pgTextArray(operand) {
			remove[text] = true
		}
		result := make(bson.A, 0, len(array))
		for _, item := range array {
			text, ok := item.(string)
			if !ok || !remove[text] {
				result = append(result, item)
			}
		}
		return result
	}
	panic("jsonb deletion requires an object or array")
}

func pgRawStripJSONNulls(value bson.RawValue, arrays bool) bson.RawValue {
	switch value.Type {
	case bson.TypeEmbeddedDocument:
		pairs := make([]bsonJSONPair, 0, len(mustBSONElements(value)))
		for _, element := range mustBSONElements(value) {
			if element.Value().Type != bson.TypeNull {
				pairs = append(pairs, bsonJSONPair{
					key: decodeBSONKey(element.Key()), value: pgRawStripJSONNulls(element.Value(), arrays),
				})
			}
		}
		return jsonRawDocument(pairs)
	case bson.TypeArray:
		result := make([]bson.RawValue, 0, len(mustBSONValues(value)))
		for _, item := range mustBSONValues(value) {
			if item.Type != bson.TypeNull || !arrays {
				result = append(result, pgRawStripJSONNulls(item, arrays))
			}
		}
		return jsonRawArray(result)
	default:
		return value
	}
}

func pgRawJSONBConcat(left, right bson.RawValue) bson.RawValue {
	if left.Type == bson.TypeEmbeddedDocument && right.Type == bson.TypeEmbeddedDocument {
		result := left
		for _, element := range mustBSONElements(right) {
			result, _ = jsonRawModify(result, []jsonPathLeg{{kind: jsonPathKey, key: decodeBSONKey(element.Key())}}, element.Value(), 's')
		}
		return result
	}
	leftValues := []bson.RawValue{left}
	if left.Type == bson.TypeArray {
		leftValues = mustBSONValues(left)
	}
	rightValues := []bson.RawValue{right}
	if right.Type == bson.TypeArray {
		rightValues = mustBSONValues(right)
	}
	return jsonRawArray(append(append([]bson.RawValue(nil), leftValues...), rightValues...))
}

func pgRawJSONBDelete(value bson.RawValue, operand Scmer) bson.RawValue {
	if value.Type == bson.TypeEmbeddedDocument {
		result := value
		for _, key := range pgTextArray(operand) {
			result, _ = jsonRawRemove(result, []jsonPathLeg{{kind: jsonPathKey, key: key}})
		}
		return result
	}
	if value.Type != bson.TypeArray {
		panic("jsonb deletion requires an object or array")
	}
	array := mustBSONValues(value)
	if operand.IsInt() || operand.IsFloat() {
		index := int(operand.Int())
		if index < 0 {
			index += len(array)
		}
		if index < 0 || index >= len(array) {
			return value
		}
		result := append([]bson.RawValue(nil), array[:index]...)
		return jsonRawArray(append(result, array[index+1:]...))
	}
	remove := make(map[string]bool)
	for _, text := range pgTextArray(operand) {
		remove[text] = true
	}
	result := make([]bson.RawValue, 0, len(array))
	for _, item := range array {
		if item.Type != bson.TypeString || !remove[item.StringValue()] {
			result = append(result, item)
		}
	}
	return jsonRawArray(result)
}

func pgJSONPathVariable(vars any, name string) (any, bool) {
	if document, ok := vars.(bson.D); ok {
		return jsonObjectLookup(document, name)
	}
	return nil, false
}

func pgJSONPathLiteral(text string, vars any) (any, bool) {
	text = strings.TrimSpace(strings.ReplaceAll(text, ".datetime()", ""))
	if strings.HasPrefix(text, "$") && len(text) > 1 {
		return pgJSONPathVariable(vars, text[1:])
	}
	if value, err := parseJSONText(text); err == nil {
		return normalizeBSONInput(value), true
	}
	return strings.Trim(text, `"`), true
}

func pgJSONPathCompare(left any, operator string, right any) bool {
	if operator == "==" {
		return jsonEqual(left, right)
	}
	if operator == "!=" || operator == "<>" {
		return !jsonEqual(left, right)
	}
	if jsonEqual(left, right) {
		return operator == ">=" || operator == "<="
	}
	less := jsonLess(left, right)
	switch operator {
	case "<", "<=":
		return less
	case ">", ">=":
		return !less
	default:
		return false
	}
}

func pgJSONPathPredicate(value any, predicate string, vars any) bool {
	predicate = strings.TrimSpace(predicate)
	if strings.HasPrefix(predicate, "!") {
		return !pgJSONPathPredicate(value, strings.TrimSpace(strings.TrimPrefix(predicate, "!")), vars)
	}
	if parts := strings.Split(predicate, "||"); len(parts) > 1 {
		for _, part := range parts {
			if pgJSONPathPredicate(value, part, vars) {
				return true
			}
		}
		return false
	}
	if parts := strings.Split(predicate, "&&"); len(parts) > 1 {
		for _, part := range parts {
			if !pgJSONPathPredicate(value, part, vars) {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(predicate, "(") && strings.HasSuffix(predicate, ")") {
		predicate = strings.TrimSpace(predicate[1 : len(predicate)-1])
	}
	predicate = strings.ReplaceAll(predicate, "@.datetime()", "@")
	comparison := regexp.MustCompile(`^@\s*(==|!=|<>|>=|<=|>|<)\s*(.+)$`).FindStringSubmatch(predicate)
	if comparison == nil {
		return false
	}
	right, ok := pgJSONPathLiteral(comparison[2], vars)
	return ok && pgJSONPathCompare(value, comparison[1], right)
}

func pgJSONPathQuery(document any, path string, vars any) ([]any, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(path, "lax "), "strict "))
	predicateResult := false
	if strings.HasPrefix(path, "exists(") && strings.HasSuffix(path, ")") {
		predicateResult = true
		path = strings.TrimSpace(path[len("exists(") : len(path)-1])
	}
	if match := regexp.MustCompile(`^(.*?)\s*\?\s*\((.*)\)$`).FindStringSubmatch(path); match != nil {
		legs, err := parseJSONPath(strings.TrimSpace(match[1]))
		if err != nil {
			panic(err)
		}
		candidates := jsonPathMatches([]any{document}, legs)
		result := make([]any, 0, len(candidates))
		for _, candidate := range candidates {
			if pgJSONPathPredicate(candidate, match[2], vars) {
				result = append(result, candidate)
			}
		}
		return result, predicateResult
	}
	if match := regexp.MustCompile(`^(\$.*?)(==|!=|<>|>=|<=|>|<)\s*(.+)$`).FindStringSubmatch(path); match != nil {
		legs, err := parseJSONPath(strings.TrimSpace(match[1]))
		if err != nil {
			panic(err)
		}
		right, ok := pgJSONPathLiteral(match[3], vars)
		if !ok {
			return nil, true
		}
		for _, candidate := range jsonPathMatches([]any{document}, legs) {
			if pgJSONPathCompare(candidate, match[2], right) {
				return []any{true}, true
			}
		}
		return []any{false}, true
	}
	legs, err := parseJSONPath(path)
	if err != nil {
		panic(err)
	}
	return jsonPathMatches([]any{document}, legs), predicateResult
}

func pgRawJSONPathLiteral(text string, vars bson.RawValue) (bson.RawValue, bool) {
	text = strings.TrimSpace(strings.ReplaceAll(text, ".datetime()", ""))
	if strings.HasPrefix(text, "$") && len(text) > 1 {
		if vars.Type != bson.TypeEmbeddedDocument {
			return bson.RawValue{}, false
		}
		value, err := vars.Document().LookupErr(encodeBSONKey(text[1:]))
		return value, err == nil
	}
	if value, err := parseBSONJSONText(text); err == nil {
		return value, true
	}
	value, _ := bsonFromSQLScalar(NewString(strings.Trim(text, `"`)))
	return bsonRawValue(value), true
}

func pgRawJSONPathCompare(left bson.RawValue, operator string, right bson.RawValue) bool {
	if operator == "==" {
		return bsonRawEqual(left, right)
	}
	if operator == "!=" || operator == "<>" {
		return !bsonRawEqual(left, right)
	}
	if bsonRawEqual(left, right) {
		return operator == ">=" || operator == "<="
	}
	less := bsonRawLess(left, right)
	switch operator {
	case "<", "<=":
		return less
	case ">", ">=":
		return !less
	default:
		return false
	}
}

func pgRawJSONPathPredicate(value bson.RawValue, predicate string, vars bson.RawValue) bool {
	predicate = strings.TrimSpace(predicate)
	if strings.HasPrefix(predicate, "!") {
		return !pgRawJSONPathPredicate(value, strings.TrimSpace(strings.TrimPrefix(predicate, "!")), vars)
	}
	if parts := strings.Split(predicate, "||"); len(parts) > 1 {
		for _, part := range parts {
			if pgRawJSONPathPredicate(value, part, vars) {
				return true
			}
		}
		return false
	}
	if parts := strings.Split(predicate, "&&"); len(parts) > 1 {
		for _, part := range parts {
			if !pgRawJSONPathPredicate(value, part, vars) {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(predicate, "(") && strings.HasSuffix(predicate, ")") {
		predicate = strings.TrimSpace(predicate[1 : len(predicate)-1])
	}
	predicate = strings.ReplaceAll(predicate, "@.datetime()", "@")
	comparison := regexp.MustCompile(`^@\s*(==|!=|<>|>=|<=|>|<)\s*(.+)$`).FindStringSubmatch(predicate)
	if comparison == nil {
		return false
	}
	right, ok := pgRawJSONPathLiteral(comparison[2], vars)
	return ok && pgRawJSONPathCompare(value, comparison[1], right)
}

func pgRawJSONPathQuery(document bson.RawValue, path string, vars bson.RawValue) ([]bson.RawValue, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(path, "lax "), "strict "))
	predicateResult := false
	if strings.HasPrefix(path, "exists(") && strings.HasSuffix(path, ")") {
		predicateResult = true
		path = strings.TrimSpace(path[len("exists(") : len(path)-1])
	}
	if match := regexp.MustCompile(`^(.*?)\s*\?\s*\((.*)\)$`).FindStringSubmatch(path); match != nil {
		legs, err := parseJSONPath(strings.TrimSpace(match[1]))
		jsonPanic(err)
		candidates := jsonRawPathMatches([]bson.RawValue{document}, legs)
		result := make([]bson.RawValue, 0, len(candidates))
		for _, candidate := range candidates {
			if pgRawJSONPathPredicate(candidate, match[2], vars) {
				result = append(result, candidate)
			}
		}
		return result, predicateResult
	}
	if match := regexp.MustCompile(`^(\$.*?)(==|!=|<>|>=|<=|>|<)\s*(.+)$`).FindStringSubmatch(path); match != nil {
		legs, err := parseJSONPath(strings.TrimSpace(match[1]))
		jsonPanic(err)
		right, ok := pgRawJSONPathLiteral(match[3], vars)
		if !ok {
			return nil, true
		}
		for _, candidate := range jsonRawPathMatches([]bson.RawValue{document}, legs) {
			if pgRawJSONPathCompare(candidate, match[2], right) {
				value, _ := bsonFromSQLScalar(NewBool(true))
				return []bson.RawValue{bsonRawValue(value)}, true
			}
		}
		value, _ := bsonFromSQLScalar(NewBool(false))
		return []bson.RawValue{bsonRawValue(value)}, true
	}
	legs, err := parseJSONPath(path)
	jsonPanic(err)
	return jsonRawPathMatches([]bson.RawValue{document}, legs), predicateResult
}

func pgJSONTableCell(value any, text bool) Scmer {
	if value == nil {
		return NewNil()
	}
	if text {
		if scalar := jsonScalarResult(value, "CHAR"); !scalar.IsBSON() {
			return scalar
		}
		encoded, err := appendOrdinaryJSON(nil, value, false, "")
		jsonPanic(err)
		return NewString(string(encoded))
	}
	return jsonResult(value)
}

func pgJSONRecordRow(document bson.D, columns []string) Scmer {
	row := make([]Scmer, len(columns))
	for i, column := range columns {
		value, _ := jsonObjectLookup(document, column)
		row[i] = pgJSONTableCell(value, false)
	}
	return NewSlice(row)
}

func pgRawJSONTableCell(value bson.RawValue, text bool) Scmer {
	if value.Type == bson.TypeNull {
		return NewNil()
	}
	if text {
		if value.Type != bson.TypeArray && value.Type != bson.TypeEmbeddedDocument {
			return jsonRawScalarResult(value, "CHAR")
		}
		encoded, err := appendBSONJSON(nil, value, false, "", 0)
		jsonPanic(err)
		return NewString(string(encoded))
	}
	return NewBSONValue(value.Type, value.Value)
}

func pgRawJSONRecordRow(document bson.RawValue, columns []string) Scmer {
	row := make([]Scmer, len(columns))
	for i, column := range columns {
		value, err := document.Document().LookupErr(encodeBSONKey(column))
		if err != nil {
			row[i] = NewNil()
		} else {
			row[i] = pgRawJSONTableCell(value, false)
		}
	}
	return NewSlice(row)
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

func jsonRawContainsValue(target, candidate bson.RawValue) bool {
	switch candidate.Type {
	case bson.TypeEmbeddedDocument:
		if target.Type != bson.TypeEmbeddedDocument {
			return false
		}
		for _, element := range mustBSONElements(candidate) {
			available, err := target.Document().LookupErr(element.Key())
			if err != nil || !jsonRawContainsValue(available, element.Value()) {
				return false
			}
		}
		return true
	case bson.TypeArray:
		if target.Type != bson.TypeArray {
			return false
		}
		available := mustBSONValues(target)
		for _, wanted := range mustBSONValues(candidate) {
			found := false
			for _, item := range available {
				if jsonRawContainsValue(item, wanted) {
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
		if target.Type == bson.TypeArray {
			for _, item := range mustBSONValues(target) {
				if bsonRawEqual(item, candidate) {
					return true
				}
			}
			return false
		}
		return bsonRawEqual(target, candidate)
	}
}

func mustBSONElements(value bson.RawValue) []bson.RawElement {
	elements, err := value.Document().Elements()
	jsonPanic(err)
	return elements
}

func mustBSONValues(value bson.RawValue) []bson.RawValue {
	values, err := value.Array().Values()
	jsonPanic(err)
	return values
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

func jsonRawDocument(pairs []bsonJSONPair) bson.RawValue {
	return bsonRawValue(bsonDocumentFromPairs(pairs, 0))
}

func jsonRawArray(values []bson.RawValue) bson.RawValue {
	return bsonRawValue(bsonArrayFromRawValues(values, 0))
}

func jsonRawModify(value bson.RawValue, legs []jsonPathLeg, replacement bson.RawValue, mode byte) (bson.RawValue, bool) {
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
	if leg.kind == jsonPathKey {
		if value.Type != bson.TypeEmbeddedDocument {
			return value, false
		}
		elements := mustBSONElements(value)
		pairs := make([]bsonJSONPair, 0, len(elements)+1)
		found := false
		for _, element := range elements {
			key := decodeBSONKey(element.Key())
			child := element.Value()
			if key == leg.key {
				found = true
				if len(legs) == 1 {
					if mode == 'i' {
						return value, false
					}
					child = replacement
				} else {
					modified, changed := jsonRawModify(child, legs[1:], replacement, mode)
					if !changed {
						return value, false
					}
					child = modified
				}
			}
			pairs = append(pairs, bsonJSONPair{key: key, value: child})
		}
		if !found {
			if len(legs) != 1 || mode == 'r' {
				return value, false
			}
			pairs = append(pairs, bsonJSONPair{key: leg.key, value: replacement})
		}
		return jsonRawDocument(pairs), true
	}
	if value.Type != bson.TypeArray {
		return value, false
	}
	values := mustBSONValues(value)
	index := leg.index
	if index < 0 {
		index += len(values)
	}
	if len(legs) == 1 {
		if index >= 0 && index < len(values) {
			if mode == 'i' {
				return value, false
			}
			result := append([]bson.RawValue(nil), values...)
			result[index] = replacement
			return jsonRawArray(result), true
		}
		if mode != 'r' && index >= len(values) {
			return jsonRawArray(append(append([]bson.RawValue(nil), values...), replacement)), true
		}
		return value, false
	}
	if index < 0 || index >= len(values) {
		return value, false
	}
	modified, changed := jsonRawModify(values[index], legs[1:], replacement, mode)
	if !changed {
		return value, false
	}
	result := append([]bson.RawValue(nil), values...)
	result[index] = modified
	return jsonRawArray(result), true
}

func jsonRawRemove(value bson.RawValue, legs []jsonPathLeg) (bson.RawValue, bool) {
	if len(legs) == 0 {
		panic("JSON_REMOVE cannot remove the root value")
	}
	leg := legs[0]
	if leg.kind != jsonPathKey && leg.kind != jsonPathIndex {
		panic("JSON_REMOVE paths cannot contain wildcards")
	}
	if len(legs) > 1 {
		matches := jsonRawPathMatches([]bson.RawValue{value}, legs[:1])
		if len(matches) == 0 {
			return value, false
		}
		child, changed := jsonRawRemove(matches[0], legs[1:])
		if !changed {
			return value, false
		}
		return jsonRawModify(value, legs[:1], child, 's')
	}
	if leg.kind == jsonPathKey {
		if value.Type != bson.TypeEmbeddedDocument {
			return value, false
		}
		pairs := make([]bsonJSONPair, 0, len(mustBSONElements(value)))
		removed := false
		for _, element := range mustBSONElements(value) {
			key := decodeBSONKey(element.Key())
			if key == leg.key {
				removed = true
				continue
			}
			pairs = append(pairs, bsonJSONPair{key: key, value: element.Value()})
		}
		if !removed {
			return value, false
		}
		return jsonRawDocument(pairs), true
	}
	if value.Type != bson.TypeArray {
		return value, false
	}
	values := mustBSONValues(value)
	index := leg.index
	if index < 0 {
		index += len(values)
	}
	if index < 0 || index >= len(values) {
		return value, false
	}
	result := append([]bson.RawValue(nil), values[:index]...)
	result = append(result, values[index+1:]...)
	return jsonRawArray(result), true
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

func jsonRawMergePatch(target, patch bson.RawValue) bson.RawValue {
	if patch.Type != bson.TypeEmbeddedDocument {
		return patch
	}
	if target.Type != bson.TypeEmbeddedDocument {
		target = jsonRawDocument(nil)
	}
	for _, element := range mustBSONElements(patch) {
		key := decodeBSONKey(element.Key())
		patchValue := element.Value()
		path := []jsonPathLeg{{kind: jsonPathKey, key: key}}
		if patchValue.Type == bson.TypeNull {
			target, _ = jsonRawRemove(target, path)
			continue
		}
		value := patchValue
		if old, err := target.Document().LookupErr(element.Key()); err == nil {
			value = jsonRawMergePatch(old, patchValue)
		}
		target, _ = jsonRawModify(target, path, value, 's')
	}
	return target
}

func jsonRawMergePreserve(a, b bson.RawValue) bson.RawValue {
	if a.Type == bson.TypeArray && b.Type == bson.TypeArray {
		values := append([]bson.RawValue(nil), mustBSONValues(a)...)
		return jsonRawArray(append(values, mustBSONValues(b)...))
	}
	if a.Type == bson.TypeEmbeddedDocument && b.Type == bson.TypeEmbeddedDocument {
		result := a
		for _, element := range mustBSONElements(b) {
			key := decodeBSONKey(element.Key())
			value := element.Value()
			if old, err := result.Document().LookupErr(element.Key()); err == nil {
				value = jsonRawMergePreserve(old, value)
			}
			result, _ = jsonRawModify(result, []jsonPathLeg{{kind: jsonPathKey, key: key}}, value, 's')
		}
		return result
	}
	left := []bson.RawValue{a}
	if a.Type == bson.TypeArray {
		left = mustBSONValues(a)
	}
	if b.Type == bson.TypeArray {
		return jsonRawArray(append(append([]bson.RawValue(nil), left...), mustBSONValues(b)...))
	}
	return jsonRawArray(append(append([]bson.RawValue(nil), left...), b))
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

func jsonRawSearchPaths(value bson.RawValue, pattern, escape, path string, result *[]string) {
	switch value.Type {
	case bson.TypeString:
		if jsonSQLLike(value.StringValue(), pattern, escape) {
			*result = append(*result, path)
		}
	case bson.TypeArray:
		for i, child := range mustBSONValues(value) {
			jsonRawSearchPaths(child, pattern, escape, path+"["+strconv.Itoa(i)+"]", result)
		}
	case bson.TypeEmbeddedDocument:
		for _, element := range mustBSONElements(value) {
			key := appendJSONString(nil, decodeBSONKey(element.Key()))
			jsonRawSearchPaths(element.Value(), pattern, escape, path+"."+string(key), result)
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

func jsonRawScalarResult(value bson.RawValue, returning string) Scmer {
	if value.Type == bson.TypeNull {
		return NewNil()
	}
	bsonValue := NewBSONValue(value.Type, value.Value)
	switch strings.ToUpper(returning) {
	case "SIGNED", "INTEGER", "INT", "BIGINT", "UNSIGNED":
		return NewInt(bsonValue.Int())
	case "FLOAT", "DOUBLE", "REAL", "DECIMAL", "NUMERIC":
		return NewFloat(bsonValue.Float())
	case "BOOLEAN", "BOOL":
		return NewBool(bsonValue.Bool())
	case "JSON":
		return bsonValue
	default:
		switch value.Type {
		case bson.TypeString:
			return NewString(value.StringValue())
		case bson.TypeBoolean:
			return NewString(strconv.FormatBool(value.Boolean()))
		case bson.TypeInt32:
			return NewString(strconv.FormatInt(int64(value.Int32()), 10))
		case bson.TypeInt64:
			return NewString(strconv.FormatInt(value.Int64(), 10))
		case bson.TypeDouble:
			return NewString(strconv.FormatFloat(value.Double(), 'g', -1, 64))
		case bson.TypeDecimal128:
			return NewString(value.Decimal128().String())
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
		return bsonArrayFromValues(args, 0)
	})
	declareJSON("json_object", "creates a MySQL-compatible JSON object", func(args ...Scmer) Scmer {
		if len(args)%2 != 0 {
			panic("JSON_OBJECT expects key/value pairs")
		}
		pairs := make([]bsonJSONPair, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if args[i].IsNil() {
				panic("JSON_OBJECT key cannot be NULL")
			}
			value, err := bsonFromSQLScalar(args[i+1])
			jsonPanic(err)
			pairs = append(pairs, bsonJSONPair{key: args[i].String(), value: bsonRawValue(value)})
		}
		return bsonDocumentFromPairs(pairs, 0)
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
			raw := bsonRawValue(args[0])
			if raw.Type == bson.TypeString {
				return NewString(raw.StringValue())
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
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		return NewString(jsonRawTypeName(bsonRawValue(value)))
	})
	declareJSON("json_depth", "returns maximum JSON document depth", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_DEPTH", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		return NewInt(jsonRawDepth(bsonRawValue(value)))
	})
	declareJSON("json_length", "returns JSON object or array length", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_LENGTH", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		raw := bsonRawValue(value)
		if len(args) > 1 {
			matches, _ := jsonExtractBSONValues(value, args[1:2])
			if len(matches) == 0 {
				return NewNil()
			}
			raw = matches[0]
		}
		switch raw.Type {
		case bson.TypeArray:
			values, err := raw.Array().Values()
			jsonPanic(err)
			return NewInt(int64(len(values)))
		case bson.TypeEmbeddedDocument:
			elements, err := raw.Document().Elements()
			jsonPanic(err)
			return NewInt(int64(len(elements)))
		default:
			return NewInt(1)
		}
	})
	declareJSON("json_extract", "extracts one or more JSON paths", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_EXTRACT", args, 2)
		document, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		matches, multiple := jsonExtractBSONValues(document, args[1:])
		if len(matches) == 0 {
			return NewNil()
		}
		if multiple {
			return bsonArrayFromRawValues(matches, 0)
		}
		return NewBSONValue(matches[0].Type, matches[0].Value)
	})
	declareJSON("json_keys", "returns object member names", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_KEYS", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		raw := bsonRawValue(value)
		if len(args) > 1 {
			matches, _ := jsonExtractBSONValues(value, args[1:2])
			if len(matches) == 0 {
				return NewNil()
			}
			raw = matches[0]
		}
		if raw.Type != bson.TypeEmbeddedDocument {
			return NewNil()
		}
		elements, err := raw.Document().Elements()
		jsonPanic(err)
		keys := make([]Scmer, len(elements))
		for i := range elements {
			key, err := elements[i].KeyErr()
			jsonPanic(err)
			keys[i] = NewString(decodeBSONKey(key))
		}
		return bsonArrayFromValues(keys, 0)
	})
	declareJSON("json_contains_path", "tests whether one or all JSON paths exist", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_CONTAINS_PATH", args, 3)
		document, ok := jsonBSONDocumentArgument(args[0])
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
			exists := len(jsonRawPathMatches([]bson.RawValue{bsonRawValue(document)}, legs)) > 0
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
		target, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		candidate, ok := jsonBSONDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		targetRaw := bsonRawValue(target)
		if len(args) > 2 {
			matches, _ := jsonExtractBSONValues(target, args[2:3])
			if len(matches) == 0 {
				return NewBool(false)
			}
			targetRaw = matches[0]
		}
		return NewBool(jsonRawContainsValue(targetRaw, bsonRawValue(candidate)))
	})
	declareJSON("json_overlaps", "tests whether JSON values overlap", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OVERLAPS", args, 2)
		left, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		right, ok := jsonBSONDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		leftRaw, rightRaw := bsonRawValue(left), bsonRawValue(right)
		if leftRaw.Type == bson.TypeArray {
			for _, lv := range mustBSONValues(leftRaw) {
				if rightRaw.Type == bson.TypeArray {
					for _, rv := range mustBSONValues(rightRaw) {
						if bsonRawEqual(lv, rv) {
							return NewBool(true)
						}
					}
				} else if bsonRawEqual(lv, rightRaw) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		}
		if leftRaw.Type == bson.TypeEmbeddedDocument {
			if rightRaw.Type != bson.TypeEmbeddedDocument {
				return NewBool(false)
			}
			for _, element := range mustBSONElements(leftRaw) {
				if rv, err := rightRaw.Document().LookupErr(element.Key()); err == nil && bsonRawEqual(element.Value(), rv) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		}
		return NewBool(bsonRawEqual(leftRaw, rightRaw))
	})
	declareJSON("json_member_of", "tests whether a JSON scalar is a member of an array", func(args ...Scmer) Scmer {
		requireJSONArgs("MEMBER OF", args, 2)
		left, err := bsonFromSQLScalar(args[0])
		jsonPanic(err)
		right, ok := jsonBSONDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		raw := bsonRawValue(right)
		if raw.Type != bson.TypeArray {
			panic("MEMBER OF requires a JSON array")
		}
		for _, item := range mustBSONValues(raw) {
			if bsonRawEqual(bsonRawValue(left), item) {
				return NewBool(true)
			}
		}
		return NewBool(false)
	})
	declareJSON("json_pretty", "pretty-prints JSON", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_PRETTY", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		encoded, err := appendBSONText(nil, value, true, "")
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
	declareJSON("json_schema_valid", "validates a document against a MySQL Draft 4 JSON schema", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_SCHEMA_VALID", args, 2)
		if args[0].IsNil() || args[1].IsNil() {
			return NewNil()
		}
		valid, _ := jsonSchemaValidation(args[0], args[1])
		return NewBool(valid)
	})
	declareJSON("json_schema_validation_report", "reports MySQL Draft 4 JSON schema validation", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_SCHEMA_VALIDATION_REPORT", args, 2)
		if args[0].IsNil() || args[1].IsNil() {
			return NewNil()
		}
		valid, validationError := jsonSchemaValidation(args[0], args[1])
		if valid {
			value, _ := bsonFromSQLScalar(NewBool(true))
			return bsonDocumentFromPairs([]bsonJSONPair{{key: "valid", value: bsonRawValue(value)}}, 0)
		}
		leaf := deepestJSONSchemaError(validationError)
		keywordPath := leaf.ErrorKind.KeywordPath()
		keyword := ""
		if len(keywordPath) > 0 {
			keyword = keywordPath[len(keywordPath)-1]
		}
		values := []Scmer{
			NewBool(false),
			NewString(leaf.Error()),
			NewString(jsonPointer(keywordPath[:max(0, len(keywordPath)-1)])),
			NewString(jsonPointer(leaf.InstanceLocation)),
			NewString(keyword),
		}
		keys := []string{"valid", "reason", "schema-location", "document-location", "schema-failed-keyword"}
		pairs := make([]bsonJSONPair, len(keys))
		for i := range keys {
			value, err := bsonFromSQLScalar(values[i])
			jsonPanic(err)
			pairs[i] = bsonJSONPair{key: keys[i], value: bsonRawValue(value)}
		}
		return bsonDocumentFromPairs(pairs, 0)
	})
	modify := func(mode byte, name string, args ...Scmer) Scmer {
		requireJSONArgs(name, args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON modification expects path/value pairs")
		}
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for i := 1; i < len(args); i += 2 {
			if args[i].IsNil() {
				return NewNil()
			}
			legs, err := parseJSONPath(args[i].String())
			jsonPanic(err)
			replacement, err := bsonFromSQLScalar(args[i+1])
			jsonPanic(err)
			modified, _ := jsonRawModify(bsonRawValue(value), legs, bsonRawValue(replacement), mode)
			value = NewBSONValue(modified.Type, modified.Value)
		}
		return value
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
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, path := range args[1:] {
			if path.IsNil() {
				return NewNil()
			}
			legs, err := parseJSONPath(path.String())
			jsonPanic(err)
			modified, _ := jsonRawRemove(bsonRawValue(value), legs)
			value = NewBSONValue(modified.Type, modified.Value)
		}
		return value
	})
	declareJSON("json_array_append", "appends values to JSON arrays", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAY_APPEND", args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON_ARRAY_APPEND expects path/value pairs")
		}
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for i := 1; i < len(args); i += 2 {
			legs, err := parseJSONPath(args[i].String())
			jsonPanic(err)
			matches := jsonRawPathMatches([]bson.RawValue{bsonRawValue(value)}, legs)
			if len(matches) == 0 {
				continue
			}
			array := []bson.RawValue{matches[0]}
			if matches[0].Type == bson.TypeArray {
				array = mustBSONValues(matches[0])
			}
			appended, err := bsonFromSQLScalar(args[i+1])
			jsonPanic(err)
			array = append(append([]bson.RawValue(nil), array...), bsonRawValue(appended))
			modified, _ := jsonRawModify(bsonRawValue(value), legs, jsonRawArray(array), 's')
			value = NewBSONValue(modified.Type, modified.Value)
		}
		return value
	})
	declareJSON("json_array_insert", "inserts values into JSON arrays", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAY_INSERT", args, 3)
		if (len(args)-1)%2 != 0 {
			panic("JSON_ARRAY_INSERT expects path/value pairs")
		}
		value, ok := jsonBSONDocumentArgument(args[0])
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
			matches := jsonRawPathMatches([]bson.RawValue{bsonRawValue(value)}, parentLegs)
			if len(matches) == 0 {
				continue
			}
			if matches[0].Type != bson.TypeArray {
				continue
			}
			array := mustBSONValues(matches[0])
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
			inserted := make([]bson.RawValue, 0, len(array)+1)
			inserted = append(inserted, array[:index]...)
			newValue, err := bsonFromSQLScalar(args[i+1])
			jsonPanic(err)
			inserted = append(inserted, bsonRawValue(newValue))
			inserted = append(inserted, array[index:]...)
			modified, _ := jsonRawModify(bsonRawValue(value), parentLegs, jsonRawArray(inserted), 's')
			value = NewBSONValue(modified.Type, modified.Value)
		}
		return value
	})
	declareJSON("json_merge_patch", "merges documents using RFC 7396 semantics", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_MERGE_PATCH", args, 2)
		result, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, arg := range args[1:] {
			patch, ok := jsonBSONDocumentArgument(arg)
			if !ok {
				return NewNil()
			}
			merged := jsonRawMergePatch(bsonRawValue(result), bsonRawValue(patch))
			result = NewBSONValue(merged.Type, merged.Value)
		}
		return result
	})
	declareJSON("json_merge_preserve", "merges documents while preserving values", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_MERGE_PRESERVE", args, 2)
		result, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		for _, arg := range args[1:] {
			next, ok := jsonBSONDocumentArgument(arg)
			if !ok {
				return NewNil()
			}
			merged := jsonRawMergePreserve(bsonRawValue(result), bsonRawValue(next))
			result = NewBSONValue(merged.Type, merged.Value)
		}
		return result
	})
	declareJSON("json_search", "searches JSON string values", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_SEARCH", args, 3)
		document, ok := jsonBSONDocumentArgument(args[0])
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
			value bson.RawValue
			path  string
		}{{bsonRawValue(document), "$"}}
		if len(args) > 4 {
			roots = roots[:0]
			for _, path := range args[4:] {
				legs, err := parseJSONPath(path.String())
				jsonPanic(err)
				for _, match := range jsonRawPathMatches([]bson.RawValue{bsonRawValue(document)}, legs) {
					roots = append(roots, struct {
						value bson.RawValue
						path  string
					}{match, path.String()})
				}
			}
		}
		paths := make([]string, 0)
		for _, root := range roots {
			jsonRawSearchPaths(root.value, args[2].String(), escape, root.path, &paths)
		}
		if len(paths) == 0 {
			return NewNil()
		}
		if mode == "one" {
			value, err := bsonFromSQLScalar(NewString(paths[0]))
			jsonPanic(err)
			return value
		}
		array := make([]Scmer, len(paths))
		for i := range paths {
			array[i] = NewString(paths[i])
		}
		return bsonArrayFromValues(array, 0)
	})
	declareJSON("json_value", "extracts a scalar JSON value", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_VALUE", args, 2)
		document, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		matches, _ := jsonExtractBSONValues(document, args[1:2])
		if len(matches) == 0 {
			return NewNil()
		}
		returning := "CHAR"
		if len(args) > 2 {
			returning = args[2].String()
		}
		return jsonRawScalarResult(matches[0], returning)
	})
	declareJSON("pg_to_json", "implements PostgreSQL to_json and to_jsonb", func(args ...Scmer) Scmer {
		requireJSONArgs("to_json", args, 1)
		value, err := bsonFromSQLScalar(args[0])
		jsonPanic(err)
		return value
	})
	declareJSON("pg_row_to_json", "implements PostgreSQL row_to_json", func(args ...Scmer) Scmer {
		requireJSONArgs("row_to_json", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		raw := bsonRawValue(value)
		if raw.Type != bson.TypeArray {
			return value
		}
		array := mustBSONValues(raw)
		pairs := make([]bsonJSONPair, len(array))
		for i := range array {
			pairs[i] = bsonJSONPair{key: fmt.Sprintf("f%d", i+1), value: array[i]}
		}
		return bsonDocumentFromPairs(pairs, 0)
	})
	declareJSON("pg_json_build_array", "implements PostgreSQL JSON array builders", func(args ...Scmer) Scmer {
		return bsonArrayFromValues(args, 0)
	})
	declareJSON("pg_json_array_absent", "implements SQL/JSON ARRAY ABSENT ON NULL", func(args ...Scmer) Scmer {
		array := make([]Scmer, 0, len(args))
		for _, arg := range args {
			if !arg.IsNil() {
				array = append(array, arg)
			}
		}
		return bsonArrayFromValues(array, 0)
	})
	declareJSON("pg_json_build_object", "implements PostgreSQL JSON object builders", func(args ...Scmer) Scmer {
		if len(args)%2 != 0 {
			panic("json_build_object expects key/value pairs")
		}
		pairs := make([]bsonJSONPair, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if args[i].IsNil() {
				panic("JSON object key cannot be NULL")
			}
			value, err := bsonFromSQLScalar(args[i+1])
			jsonPanic(err)
			pairs = append(pairs, bsonJSONPair{key: args[i].String(), value: bsonRawValue(value)})
		}
		return bsonDocumentFromPairs(pairs, 0)
	})
	declareJSON("pg_json_object", "implements PostgreSQL json_object and jsonb_object array forms", func(args ...Scmer) Scmer {
		requireJSONArgs("json_object", args, 1)
		keys := pgTextArray(args[0])
		values := []string(nil)
		if len(args) > 1 {
			values = pgTextArray(args[1])
		} else {
			if len(keys)%2 != 0 {
				panic("json_object array must contain an even number of values")
			}
			pairs := keys
			keys = make([]string, len(pairs)/2)
			values = make([]string, len(pairs)/2)
			for i := range keys {
				keys[i], values[i] = pairs[i*2], pairs[i*2+1]
			}
		}
		if len(keys) != len(values) {
			panic("json_object key and value arrays have different lengths")
		}
		pairs := make([]bsonJSONPair, len(keys))
		for i := range keys {
			value, err := bsonFromSQLScalar(NewString(values[i]))
			jsonPanic(err)
			pairs[i] = bsonJSONPair{key: keys[i], value: bsonRawValue(value)}
		}
		return bsonDocumentFromPairs(pairs, 0)
	})
	declareJSON("pg_json_serialize", "implements PostgreSQL json_serialize", func(args ...Scmer) Scmer {
		requireJSONArgs("json_serialize", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		encoded, err := appendBSONText(nil, value, false, "")
		jsonPanic(err)
		return NewString(string(encoded))
	})
	declareJSON("pg_json_array_length", "implements PostgreSQL json_array_length", func(args ...Scmer) Scmer {
		requireJSONArgs("json_array_length", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		raw := bsonRawValue(value)
		if raw.Type != bson.TypeArray {
			panic("json_array_length requires an array")
		}
		array, err := raw.Array().Values()
		jsonPanic(err)
		return NewInt(int64(len(array)))
	})
	declareJSON("pg_json_extract_path", "implements PostgreSQL json_extract_path", func(args ...Scmer) Scmer {
		requireJSONArgs("json_extract_path", args, 2)
		return pgExtractPath(args[0], args[1:], false)
	})
	declareJSON("pg_json_extract_path_text", "implements PostgreSQL json_extract_path_text", func(args ...Scmer) Scmer {
		requireJSONArgs("json_extract_path_text", args, 2)
		return pgExtractPath(args[0], args[1:], true)
	})
	declareJSON("pg_jsonb_set", "implements PostgreSQL jsonb_set", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb_set", args, 3)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		create := len(args) < 4 || args[3].Bool()
		mode := byte('s')
		if !create {
			mode = 'r'
		}
		replacement, replacementOK := jsonBSONDocumentArgument(args[2])
		if !replacementOK {
			return NewNil()
		}
		modified, _ := jsonRawModify(bsonRawValue(value), pgPathLegs(args[1]), bsonRawValue(replacement), mode)
		return NewBSONValue(modified.Type, modified.Value)
	})
	declareJSON("pg_jsonb_set_lax", "implements PostgreSQL jsonb_set_lax", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb_set_lax", args, 3)
		if !args[2].IsNil() {
			return Globalenv.Vars[Symbol("pg_jsonb_set")].Func()(args...)
		}
		treatment := "use_json_null"
		if len(args) > 4 {
			treatment = strings.ToLower(args[4].String())
		}
		switch treatment {
		case "raise_exception":
			panic("jsonb_set_lax new value is SQL NULL")
		case "return_target":
			return args[0]
		case "delete_key":
			value, ok := jsonBSONDocumentArgument(args[0])
			if !ok {
				return NewNil()
			}
			modified, _ := jsonRawRemove(bsonRawValue(value), pgPathLegs(args[1]))
			return NewBSONValue(modified.Type, modified.Value)
		case "use_json_null":
			replacement, _ := bsonFromGo(nil)
			forward := append([]Scmer(nil), args...)
			forward[2] = replacement
			return Globalenv.Vars[Symbol("pg_jsonb_set")].Func()(forward...)
		default:
			panic("invalid jsonb_set_lax null_value_treatment")
		}
	})
	declareJSON("pg_jsonb_insert", "implements PostgreSQL jsonb_insert", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb_insert", args, 3)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		replacement, replacementOK := jsonBSONDocumentArgument(args[2])
		if !replacementOK {
			return NewNil()
		}
		legs := pgPathLegs(args[1])
		if len(legs) == 0 {
			return args[0]
		}
		last, parentLegs := legs[len(legs)-1], legs[:len(legs)-1]
		parents := jsonRawPathMatches([]bson.RawValue{bsonRawValue(value)}, parentLegs)
		if len(parents) == 0 {
			return args[0]
		}
		if last.kind == jsonPathIndex {
			if parents[0].Type != bson.TypeArray {
				return args[0]
			}
			array := mustBSONValues(parents[0])
			index := last.index
			if index < 0 {
				index += len(array)
			}
			if len(args) > 3 && args[3].Bool() {
				index++
			}
			if index < 0 {
				index = 0
			}
			if index > len(array) {
				index = len(array)
			}
			result := append([]bson.RawValue(nil), array[:index]...)
			result = append(result, bsonRawValue(replacement))
			result = append(result, array[index:]...)
			modified, _ := jsonRawModify(bsonRawValue(value), parentLegs, jsonRawArray(result), 's')
			value = NewBSONValue(modified.Type, modified.Value)
		} else {
			modified, _ := jsonRawModify(bsonRawValue(value), legs, bsonRawValue(replacement), 'i')
			value = NewBSONValue(modified.Type, modified.Value)
		}
		return value
	})
	declareJSON("pg_json_strip_nulls", "implements PostgreSQL json_strip_nulls", func(args ...Scmer) Scmer {
		requireJSONArgs("json_strip_nulls", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		stripped := pgRawStripJSONNulls(bsonRawValue(value), len(args) > 1 && args[1].Bool())
		return NewBSONValue(stripped.Type, stripped.Value)
	})
	declareJSON("pg_json_typeof", "implements PostgreSQL json_typeof", func(args ...Scmer) Scmer {
		requireJSONArgs("json_typeof", args, 1)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		typeName := jsonRawTypeName(bsonRawValue(value))
		if typeName == "INTEGER" || typeName == "DOUBLE" || typeName == "DECIMAL" {
			typeName = "NUMBER"
		}
		return NewString(strings.ToLower(typeName))
	})
	declareJSON("pg_jsonb_populate_record_valid", "validates PostgreSQL jsonb_populate_record input shape", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb_populate_record_valid", args, 2)
		value, ok := jsonBSONDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		return NewBool(bsonRawValue(value).Type == bson.TypeEmbeddedDocument)
	})
	declareJSON("pg_jsonb_concat", "implements PostgreSQL jsonb concatenation", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb concatenation", args, 2)
		left, leftOK := jsonBSONDocumentArgument(args[0])
		right, rightOK := jsonBSONDocumentArgument(args[1])
		if !leftOK || !rightOK {
			return NewNil()
		}
		result := pgRawJSONBConcat(bsonRawValue(left), bsonRawValue(right))
		return NewBSONValue(result.Type, result.Value)
	})
	declareJSON("pg_jsonb_delete", "implements PostgreSQL jsonb deletion", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb deletion", args, 2)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		result := pgRawJSONBDelete(bsonRawValue(value), args[1])
		return NewBSONValue(result.Type, result.Value)
	})
	declareJSON("pg_jsonb_delete_path", "implements PostgreSQL jsonb path deletion", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb path deletion", args, 2)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		result, _ := jsonRawRemove(bsonRawValue(value), pgPathLegs(args[1]))
		return NewBSONValue(result.Type, result.Value)
	})
	declareJSON("pg_subtract", "dispatches PostgreSQL numeric and jsonb subtraction", func(args ...Scmer) Scmer {
		requireJSONArgs("subtraction", args, 2)
		if args[0].IsBSON() {
			result := pgRawJSONBDelete(bsonRawValue(args[0]), args[1])
			return NewBSONValue(result.Type, result.Value)
		}
		if args[0].IsInt() && args[1].IsInt() {
			return NewInt(args[0].Int() - args[1].Int())
		}
		return NewFloat(args[0].Float() - args[1].Float())
	})
	declareJSON("pg_jsonb_exists", "implements PostgreSQL jsonb top-level existence", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb existence", args, 2)
		value, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		keys := pgTextArray(args[1])
		mode := "one"
		if len(args) > 2 {
			mode = args[2].String()
		}
		matches := 0
		for _, key := range keys {
			found := false
			raw := bsonRawValue(value)
			if raw.Type == bson.TypeEmbeddedDocument {
				_, err := raw.Document().LookupErr(encodeBSONKey(key))
				found = err == nil
			} else if raw.Type == bson.TypeArray {
				for _, item := range mustBSONValues(raw) {
					if item.Type == bson.TypeString && item.StringValue() == key {
						found = true
						break
					}
				}
			}
			if found {
				matches++
			}
		}
		if mode == "all" {
			return NewBool(matches == len(keys))
		}
		return NewBool(matches > 0)
	})
	pathQuery := func(name string, args []Scmer) ([]bson.RawValue, bool, bool) {
		requireJSONArgs(name, args, 2)
		document, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return nil, false, false
		}
		vars := bsonDocumentFromPairs(nil, 0)
		if len(args) > 2 && !args[2].IsNil() {
			vars, _ = jsonBSONDocumentArgument(args[2])
		}
		values, predicate := pgRawJSONPathQuery(bsonRawValue(document), args[1].String(), bsonRawValue(vars))
		return values, predicate, true
	}
	declareJSON("pg_jsonb_path_exists", "implements PostgreSQL jsonb_path_exists", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("jsonb_path_exists", args)
		if !ok {
			return NewNil()
		}
		return NewBool(len(values) > 0)
	})
	declareJSON("pg_jsonb_path_match", "implements PostgreSQL jsonb_path_match", func(args ...Scmer) Scmer {
		values, predicate, ok := pathQuery("jsonb_path_match", args)
		if !ok {
			return NewNil()
		}
		if predicate {
			return NewBool(len(values) > 0)
		}
		if len(values) == 1 {
			if values[0].Type == bson.TypeBoolean {
				return NewBool(values[0].Boolean())
			}
		}
		return NewNil()
	})
	declareJSON("pg_jsonb_path_query_array", "implements PostgreSQL jsonb_path_query_array", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("jsonb_path_query_array", args)
		if !ok {
			return NewNil()
		}
		return bsonArrayFromRawValues(values, 0)
	})
	declareJSON("pg_jsonb_path_query_first", "implements PostgreSQL jsonb_path_query_first", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("jsonb_path_query_first", args)
		if !ok || len(values) == 0 {
			return NewNil()
		}
		return NewBSONValue(values[0].Type, values[0].Value)
	})
	declareJSON("pg_json_value", "implements PostgreSQL SQL/JSON JSON_VALUE", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("JSON_VALUE", args)
		if !ok || len(values) == 0 {
			if len(args) > 3 {
				return args[3]
			}
			return NewNil()
		}
		if len(values) != 1 {
			panic("JSON_VALUE path returned more than one item")
		}
		return jsonRawScalarResult(values[0], "CHAR")
	})
	declareJSON("json_get", "implements PostgreSQL JSON object and array extraction", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON extraction operator", args, 2)
		document, ok := jsonBSONDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		root := bsonRawValue(document)
		var value bson.RawValue
		var found bool
		if args[1].IsInt() || args[1].IsFloat() {
			if root.Type == bson.TypeArray {
				array := root.Array()
				index := int(args[1].Int())
				if index < 0 {
					values, err := array.Values()
					jsonPanic(err)
					index = len(values) + index
					if index >= 0 && index < len(values) {
						value, found = values[index], true
					}
				} else if child, err := array.IndexErr(uint(index)); err == nil {
					value, found = child, true
				}
			}
		} else if root.Type == bson.TypeEmbeddedDocument {
			var err error
			value, err = root.Document().LookupErr(encodeBSONKey(args[1].String()))
			found = err == nil
		}
		if !found {
			return NewNil()
		}
		if len(args) > 2 && args[2].Bool() {
			if value.Type == bson.TypeArray || value.Type == bson.TypeEmbeddedDocument {
				encoded, err := appendBSONJSON(nil, value, false, "", 0)
				jsonPanic(err)
				return NewString(string(encoded))
			}
			return jsonRawScalarResult(value, "CHAR")
		}
		return NewBSONValue(value.Type, value.Value)
	})
	declareJSON("json_arrayagg_entry", "wraps one JSON_ARRAYAGG input value", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAYAGG", args, 1)
		if len(args) > 1 && args[1].Bool() && args[0].IsNil() {
			return bsonArrayFromValues(nil, bsonFlagArrayAgg)
		}
		return bsonArrayFromValues(args[:1], bsonFlagArrayAgg)
	})
	declareJSON("json_arrayagg_reduce", "JSON_ARRAYAGG reducer", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_ARRAYAGG reducer", args, 2)
		leftIsState := bsonHasFlag(args[0], bsonFlagArrayAgg)
		rightIsState := bsonHasFlag(args[1], bsonFlagArrayAgg)
		switch {
		case args[0].IsNil() && rightIsState:
			return args[1]
		case args[1].IsNil() && leftIsState:
			return args[0]
		case args[0].IsNil():
			return bsonArrayFromValues(args[1:2], bsonFlagArrayAgg)
		case args[1].IsNil():
			return bsonArrayFromValues(args[:1], bsonFlagArrayAgg)
		}
		left := bson.RawValue{}
		if leftIsState {
			left = bsonRawValue(args[0])
		} else {
			value, err := bsonFromSQLScalar(args[0])
			jsonPanic(err)
			left = bsonRawValue(value)
		}
		right := bson.RawValue{}
		if rightIsState {
			right = bsonRawValue(args[1])
		} else {
			value, err := bsonFromSQLScalar(args[1])
			jsonPanic(err)
			right = bsonRawValue(value)
		}
		return bsonArrayFromRawParts(left, leftIsState, right, rightIsState, bsonFlagArrayAgg)
	})
	declareJSON("json_objectagg_entry", "constructs a JSON_OBJECTAGG key/value entry", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OBJECTAGG", args, 2)
		if args[0].IsNil() {
			panic("JSON_OBJECTAGG key cannot be NULL")
		}
		if len(args) > 2 && args[2].Bool() && args[1].IsNil() {
			return bsonDocumentFromPairs(nil, bsonFlagObjectAgg)
		}
		return bsonArrayFromValues([]Scmer{NewString(args[0].String()), args[1]}, 0)
	})
	declareJSON("json_objectagg_reduce", "JSON_OBJECTAGG reducer", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON_OBJECTAGG reducer", args, 2)
		if args[0].IsNil() && bsonHasFlag(args[1], bsonFlagObjectAgg) {
			return args[1]
		}
		if args[1].IsNil() && bsonHasFlag(args[0], bsonFlagObjectAgg) {
			return args[0]
		}
		leftState := bsonHasFlag(args[0], bsonFlagObjectAgg)
		rightState := bsonHasFlag(args[1], bsonFlagObjectAgg)
		document := jsonRawDocument(nil)
		if leftState {
			document = bsonRawValue(args[0])
		} else if rightState {
			document = bsonRawValue(args[1])
		}
		if leftState && rightState {
			for _, element := range mustBSONElements(bsonRawValue(args[1])) {
				document, _ = jsonRawModify(document, []jsonPathLeg{{kind: jsonPathKey, key: decodeBSONKey(element.Key())}}, element.Value(), 's')
			}
			return newBSONValueFlags(bson.TypeEmbeddedDocument, document.Value, bsonFlagObjectAgg)
		}
		entryValue := args[0]
		if args[0].IsNil() || leftState {
			entryValue = args[1]
		} else if args[1].IsNil() || rightState {
			entryValue = args[0]
		}
		if !entryValue.IsBSON() || bsonRawValue(entryValue).Type != bson.TypeArray {
			panic("invalid JSON_OBJECTAGG entry")
		}
		entry := mustBSONValues(bsonRawValue(entryValue))
		if len(entry) != 2 || entry[0].Type != bson.TypeString {
			panic("invalid JSON_OBJECTAGG entry")
		}
		document, _ = jsonRawModify(document, []jsonPathLeg{{kind: jsonPathKey, key: entry[0].StringValue()}}, entry[1], 's')
		return newBSONValueFlags(bson.TypeEmbeddedDocument, document.Value, bsonFlagObjectAgg)
	})
	declareJSON("pg_json_table_rows", "materializes a PostgreSQL JSON table function as rows", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON table function", args, 2)
		kind := args[0].String()
		document, ok := jsonBSONDocumentArgument(args[1])
		if !ok {
			return NewSlice(nil)
		}
		rows := make([]Scmer, 0)
		rawDocument := bsonRawValue(document)
		singleColumnRows := func(values []bson.RawValue, text bool) {
			for _, value := range values {
				rows = append(rows, NewSlice([]Scmer{pgRawJSONTableCell(value, text)}))
			}
		}
		switch kind {
		case "array", "array_text":
			if rawDocument.Type != bson.TypeArray {
				panic("json_array_elements requires an array")
			}
			singleColumnRows(mustBSONValues(rawDocument), kind == "array_text")
		case "each", "each_text":
			if rawDocument.Type != bson.TypeEmbeddedDocument {
				panic("json_each requires an object")
			}
			for _, element := range mustBSONElements(rawDocument) {
				rows = append(rows, NewSlice([]Scmer{NewString(decodeBSONKey(element.Key())), pgRawJSONTableCell(element.Value(), kind == "each_text")}))
			}
		case "keys":
			if rawDocument.Type != bson.TypeEmbeddedDocument {
				panic("json_object_keys requires an object")
			}
			for _, element := range mustBSONElements(rawDocument) {
				rows = append(rows, NewSlice([]Scmer{NewString(decodeBSONKey(element.Key()))}))
			}
		case "path":
			if len(args) < 3 {
				panic("jsonb_path_query requires a path")
			}
			values, _ := pgRawJSONPathQuery(rawDocument, args[2].String(), jsonRawDocument(nil))
			singleColumnRows(values, false)
		case "record", "recordset":
			if len(args) < 3 || !args[2].IsSlice() {
				panic("record JSON table function requires a column list")
			}
			columnValues := args[2].Slice()
			columns := make([]string, len(columnValues))
			for i := range columnValues {
				columns[i] = columnValues[i].String()
			}
			if kind == "record" {
				if rawDocument.Type != bson.TypeEmbeddedDocument {
					panic("JSON record input requires an object")
				}
				rows = append(rows, pgRawJSONRecordRow(rawDocument, columns))
			} else {
				if rawDocument.Type != bson.TypeArray {
					panic("JSON recordset input requires an array")
				}
				for _, value := range mustBSONValues(rawDocument) {
					if value.Type != bson.TypeEmbeddedDocument {
						panic("JSON recordset element requires an object")
					}
					rows = append(rows, pgRawJSONRecordRow(value, columns))
				}
			}
		case "json_table":
			if len(args) < 4 || !args[3].IsSlice() {
				panic("JSON_TABLE requires a row path and column definitions")
			}
			values, _ := pgRawJSONPathQuery(rawDocument, args[2].String(), jsonRawDocument(nil))
			definitions := args[3].Slice()
			for rowIndex, value := range values {
				row := make([]Scmer, len(definitions))
				for columnIndex, definitionValue := range definitions {
					definition := definitionValue.Slice()
					if definition[1].String() == "ordinality" {
						row[columnIndex] = NewInt(int64(rowIndex + 1))
						continue
					}
					matches, _ := pgRawJSONPathQuery(value, definition[2].String(), jsonRawDocument(nil))
					if len(matches) == 0 {
						row[columnIndex] = NewNil()
					} else {
						returning := "CHAR"
						if len(definition) > 3 {
							returning = definition[3].String()
						}
						row[columnIndex] = jsonRawScalarResult(matches[0], returning)
					}
				}
				rows = append(rows, NewSlice(row))
			}
		default:
			panic("unknown JSON table function: " + kind)
		}
		return NewSlice(rows)
	})
}
