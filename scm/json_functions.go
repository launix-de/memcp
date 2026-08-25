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
		if array, ok := bsonDecoded(value).(bson.A); ok {
			result := make([]string, len(array))
			for i := range array {
				result[i] = fmt.Sprint(array[i])
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
	value, ok := jsonDocumentArgument(document)
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
	matches := jsonPathMatches([]any{value}, legs)
	if len(matches) == 0 {
		return NewNil()
	}
	if textResult {
		return jsonScalarResult(matches[0], "CHAR")
	}
	return jsonResult(matches[0])
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
			return jsonResult(bson.D{{Key: "valid", Value: true}})
		}
		leaf := deepestJSONSchemaError(validationError)
		keywordPath := leaf.ErrorKind.KeywordPath()
		keyword := ""
		if len(keywordPath) > 0 {
			keyword = keywordPath[len(keywordPath)-1]
		}
		return jsonResult(bson.D{
			{Key: "valid", Value: false},
			{Key: "reason", Value: leaf.Error()},
			{Key: "schema-location", Value: jsonPointer(keywordPath[:max(0, len(keywordPath)-1)])},
			{Key: "document-location", Value: jsonPointer(leaf.InstanceLocation)},
			{Key: "schema-failed-keyword", Value: keyword},
		})
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
	declareJSON("pg_to_json", "implements PostgreSQL to_json and to_jsonb", func(args ...Scmer) Scmer {
		requireJSONArgs("to_json", args, 1)
		return jsonResult(jsonConstructorArgument(args[0]))
	})
	declareJSON("pg_row_to_json", "implements PostgreSQL row_to_json", func(args ...Scmer) Scmer {
		requireJSONArgs("row_to_json", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		array, ok := value.(bson.A)
		if !ok {
			return jsonResult(value)
		}
		document := make(bson.D, len(array))
		for i := range array {
			document[i] = bson.E{Key: fmt.Sprintf("f%d", i+1), Value: array[i]}
		}
		return jsonResult(document)
	})
	declareJSON("pg_json_build_array", "implements PostgreSQL JSON array builders", func(args ...Scmer) Scmer {
		array := make(bson.A, len(args))
		for i := range args {
			array[i] = jsonConstructorArgument(args[i])
		}
		return jsonResult(array)
	})
	declareJSON("pg_json_array_absent", "implements SQL/JSON ARRAY ABSENT ON NULL", func(args ...Scmer) Scmer {
		array := make(bson.A, 0, len(args))
		for _, arg := range args {
			if !arg.IsNil() {
				array = append(array, jsonConstructorArgument(arg))
			}
		}
		return jsonResult(array)
	})
	declareJSON("pg_json_build_object", "implements PostgreSQL JSON object builders", func(args ...Scmer) Scmer {
		if len(args)%2 != 0 {
			panic("json_build_object expects key/value pairs")
		}
		document := bson.D{}
		for i := 0; i < len(args); i += 2 {
			if args[i].IsNil() {
				panic("JSON object key cannot be NULL")
			}
			document, _ = jsonObjectSet(document, args[i].String(), jsonConstructorArgument(args[i+1]), false, false)
		}
		return jsonResult(document)
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
		document := bson.D{}
		for i := range keys {
			document, _ = jsonObjectSet(document, keys[i], values[i], false, false)
		}
		return jsonResult(document)
	})
	declareJSON("pg_json_serialize", "implements PostgreSQL json_serialize", func(args ...Scmer) Scmer {
		requireJSONArgs("json_serialize", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		encoded, err := appendOrdinaryJSON(nil, value, false, "")
		jsonPanic(err)
		return NewString(string(encoded))
	})
	declareJSON("pg_json_array_length", "implements PostgreSQL json_array_length", func(args ...Scmer) Scmer {
		requireJSONArgs("json_array_length", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		array, ok := value.(bson.A)
		if !ok {
			panic("json_array_length requires an array")
		}
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
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		create := len(args) < 4 || args[3].Bool()
		mode := byte('s')
		if !create {
			mode = 'r'
		}
		replacement, replacementOK := jsonDocumentArgument(args[2])
		if !replacementOK {
			return NewNil()
		}
		value, _ = jsonModify(value, pgPathLegs(args[1]), replacement, mode)
		return jsonResult(value)
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
			value, ok := jsonDocumentArgument(args[0])
			if !ok {
				return NewNil()
			}
			value, _ = jsonRemoveValue(value, pgPathLegs(args[1]))
			return jsonResult(value)
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
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		replacement, replacementOK := jsonDocumentArgument(args[2])
		if !replacementOK {
			return NewNil()
		}
		legs := pgPathLegs(args[1])
		if len(legs) == 0 {
			return args[0]
		}
		last, parentLegs := legs[len(legs)-1], legs[:len(legs)-1]
		parents := jsonPathMatches([]any{value}, parentLegs)
		if len(parents) == 0 {
			return args[0]
		}
		if last.kind == jsonPathIndex {
			array, arrayOK := parents[0].(bson.A)
			if !arrayOK {
				return args[0]
			}
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
			result := append(append(append(bson.A(nil), array[:index]...), replacement), array[index:]...)
			value, _ = jsonModify(value, parentLegs, result, 's')
		} else {
			value, _ = jsonModify(value, legs, replacement, 'i')
		}
		return jsonResult(value)
	})
	declareJSON("pg_json_strip_nulls", "implements PostgreSQL json_strip_nulls", func(args ...Scmer) Scmer {
		requireJSONArgs("json_strip_nulls", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		return jsonResult(pgStripJSONNulls(value, len(args) > 1 && args[1].Bool()))
	})
	declareJSON("pg_json_typeof", "implements PostgreSQL json_typeof", func(args ...Scmer) Scmer {
		requireJSONArgs("json_typeof", args, 1)
		value, ok := jsonDocumentArgument(args[0])
		if !ok {
			return NewNil()
		}
		typeName := jsonTypeName(value)
		if typeName == "INTEGER" || typeName == "DOUBLE" || typeName == "DECIMAL" {
			typeName = "NUMBER"
		}
		return NewString(strings.ToLower(typeName))
	})
	declareJSON("pg_jsonb_populate_record_valid", "validates PostgreSQL jsonb_populate_record input shape", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb_populate_record_valid", args, 2)
		value, ok := jsonDocumentArgument(args[1])
		if !ok {
			return NewNil()
		}
		_, objectOK := value.(bson.D)
		return NewBool(objectOK)
	})
	declareJSON("pg_jsonb_concat", "implements PostgreSQL jsonb concatenation", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb concatenation", args, 2)
		left, leftOK := jsonDocumentArgument(args[0])
		right, rightOK := jsonDocumentArgument(args[1])
		if !leftOK || !rightOK {
			return NewNil()
		}
		return jsonResult(pgJSONBConcat(left, right))
	})
	declareJSON("pg_jsonb_delete", "implements PostgreSQL jsonb deletion", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb deletion", args, 2)
		value, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		return jsonResult(pgJSONBDelete(value, args[1]))
	})
	declareJSON("pg_jsonb_delete_path", "implements PostgreSQL jsonb path deletion", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb path deletion", args, 2)
		value, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return NewNil()
		}
		value, _ = jsonRemoveValue(value, pgPathLegs(args[1]))
		return jsonResult(value)
	})
	declareJSON("pg_subtract", "dispatches PostgreSQL numeric and jsonb subtraction", func(args ...Scmer) Scmer {
		requireJSONArgs("subtraction", args, 2)
		if args[0].IsBSON() {
			value, _ := jsonDocumentArgument(args[0])
			return jsonResult(pgJSONBDelete(value, args[1]))
		}
		if args[0].IsInt() && args[1].IsInt() {
			return NewInt(args[0].Int() - args[1].Int())
		}
		return NewFloat(args[0].Float() - args[1].Float())
	})
	declareJSON("pg_jsonb_exists", "implements PostgreSQL jsonb top-level existence", func(args ...Scmer) Scmer {
		requireJSONArgs("jsonb existence", args, 2)
		value, ok := jsonDocumentArgument(args[0])
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
			if object, objectOK := value.(bson.D); objectOK {
				_, found = jsonObjectLookup(object, key)
			} else if array, arrayOK := value.(bson.A); arrayOK {
				for _, item := range array {
					if text, textOK := item.(string); textOK && text == key {
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
	pathQuery := func(name string, args []Scmer) ([]any, bool, bool) {
		requireJSONArgs(name, args, 2)
		document, ok := jsonDocumentArgument(args[0])
		if !ok || args[1].IsNil() {
			return nil, false, false
		}
		var vars any = bson.D{}
		if len(args) > 2 && !args[2].IsNil() {
			vars, _ = jsonDocumentArgument(args[2])
		}
		values, predicate := pgJSONPathQuery(document, args[1].String(), vars)
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
			if result, resultOK := values[0].(bool); resultOK {
				return NewBool(result)
			}
		}
		return NewNil()
	})
	declareJSON("pg_jsonb_path_query_array", "implements PostgreSQL jsonb_path_query_array", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("jsonb_path_query_array", args)
		if !ok {
			return NewNil()
		}
		return jsonResult(bson.A(values))
	})
	declareJSON("pg_jsonb_path_query_first", "implements PostgreSQL jsonb_path_query_first", func(args ...Scmer) Scmer {
		values, _, ok := pathQuery("jsonb_path_query_first", args)
		if !ok || len(values) == 0 {
			return NewNil()
		}
		return jsonResult(values[0])
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
		return jsonScalarResult(values[0], "CHAR")
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
		if len(args) > 1 && args[1].Bool() && args[0].IsNil() {
			return jsonAggregateResult(bson.A{}, bsonFlagArrayAgg)
		}
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
		if len(args) > 2 && args[2].Bool() && args[1].IsNil() {
			return jsonAggregateResult(bson.D{}, bsonFlagObjectAgg)
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
	declareJSON("pg_json_table_rows", "materializes a PostgreSQL JSON table function as rows", func(args ...Scmer) Scmer {
		requireJSONArgs("JSON table function", args, 2)
		kind := args[0].String()
		document, ok := jsonDocumentArgument(args[1])
		if !ok {
			return NewSlice(nil)
		}
		rows := make([]Scmer, 0)
		singleColumnRows := func(values []any, text bool) {
			for _, value := range values {
				rows = append(rows, NewSlice([]Scmer{pgJSONTableCell(value, text)}))
			}
		}
		switch kind {
		case "array", "array_text":
			array, arrayOK := document.(bson.A)
			if !arrayOK {
				panic("json_array_elements requires an array")
			}
			singleColumnRows(array, kind == "array_text")
		case "each", "each_text":
			object, objectOK := document.(bson.D)
			if !objectOK {
				panic("json_each requires an object")
			}
			for _, pair := range object {
				rows = append(rows, NewSlice([]Scmer{NewString(pair.Key), pgJSONTableCell(pair.Value, kind == "each_text")}))
			}
		case "keys":
			object, objectOK := document.(bson.D)
			if !objectOK {
				panic("json_object_keys requires an object")
			}
			for _, pair := range object {
				rows = append(rows, NewSlice([]Scmer{NewString(pair.Key)}))
			}
		case "path":
			if len(args) < 3 {
				panic("jsonb_path_query requires a path")
			}
			values, _ := pgJSONPathQuery(document, args[2].String(), bson.D{})
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
				object, objectOK := document.(bson.D)
				if !objectOK {
					panic("JSON record input requires an object")
				}
				rows = append(rows, pgJSONRecordRow(object, columns))
			} else {
				array, arrayOK := document.(bson.A)
				if !arrayOK {
					panic("JSON recordset input requires an array")
				}
				for _, value := range array {
					object, objectOK := value.(bson.D)
					if !objectOK {
						panic("JSON recordset element requires an object")
					}
					rows = append(rows, pgJSONRecordRow(object, columns))
				}
			}
		case "json_table":
			if len(args) < 4 || !args[3].IsSlice() {
				panic("JSON_TABLE requires a row path and column definitions")
			}
			values, _ := pgJSONPathQuery(document, args[2].String(), bson.D{})
			definitions := args[3].Slice()
			for rowIndex, value := range values {
				row := make([]Scmer, len(definitions))
				for columnIndex, definitionValue := range definitions {
					definition := definitionValue.Slice()
					if definition[1].String() == "ordinality" {
						row[columnIndex] = NewInt(int64(rowIndex + 1))
						continue
					}
					matches, _ := pgJSONPathQuery(value, definition[2].String(), bson.D{})
					if len(matches) == 0 {
						row[columnIndex] = NewNil()
					} else {
						returning := "CHAR"
						if len(definition) > 3 {
							returning = definition[3].String()
						}
						row[columnIndex] = jsonScalarResult(matches[0], returning)
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
