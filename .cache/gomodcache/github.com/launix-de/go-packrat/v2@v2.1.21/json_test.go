/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"
import "strconv"

func TestJSON(t *testing.T) {
	input := `{"menu": {
		"header": "SVG Viewer",
		"items": [
			{"id": "Open"},
			{"id": "OpenNew", "label": "Open New"},
			null,
			{"id": "ZoomIn", "label": "Zoom In"},
			{"id": "ZoomOut", "label": "Zoom Out"},
			{"id": "OriginalView", "label": "Original View"},
			null,
			{"id": "Quality"},
			{"id": "Pause"},
			{"id": "Mute"},
			null,
			{"id": "Find", "label": "Find..."},
			{"id": "FindAgain", "label": "Find Again"},
			{"id": "Copy"},
			{"id": "CopyAgain", "label": "Copy Again"},
			{"id": "CopySVG", "label": "Copy SVG"},
			{"id": "ViewSVG", "label": "View SVG"},
			{"id": "ViewSource", "label": "View Source"},
			{"id": "SaveAs", "label": "Save As"},
			null,
			{"id": "Help"},
			{"id": "About", "label": "About Adobe CVG Viewer..."}
		]
	}}`
	scanner := NewScanner[any](input, SkipWhitespaceRegex)

	stringParser := NewAndParser[any](func (s string, a ...any) any {return a[1]}, NewAtomParser[any](nil, `"`, false, true), NewRegexParser(func (s string) any {return s }, `(?:[^"\\]|\\.)*`, false, false), NewAtomParser[any](nil, `"`, false, false))
	valueParser := NewOrParser[any](nil)
	propParser := NewAndParser[any](func (s string, a ...any) any { return []any{a[0], a[2]} }, stringParser, NewAtomParser[any](nil, ":", false, true), valueParser)

	objParser := NewAndParser(func (s string, a ...any) any {return a[1]}, NewAtomParser[any](nil, "{", false, true), NewKleeneParser(func (s string, a ...any) any {
		result := make(map[string]any)
		for _, v := range a {
			vx := v.([]any)
			result[vx[0].(string)] = vx[1]
		}
		return result
	}, propParser, NewAtomParser[any](nil, ",", false, true)), NewAtomParser[any](nil, "}", false, true))
	nullParser := NewAtomParser[any](nil, "null", false, true)
	numParser := NewRegexParser(func (s string) any {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}, `-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?`, false, true)
	boolParser := NewRegexParser(func (s string) any { return s != "false" }, "(true|false)", false, true)
	arrayParser := NewAndParser(func (s string, a ...any) any { return a[1] }, NewAtomParser[any](nil, "[", false, true), NewKleeneParser(func (s string, a ...any) any { return a }, valueParser, NewAtomParser[any](nil, ",", false, true)), NewAtomParser[any](nil, "]", false, true))
	valueParser.Set(nullParser, objParser, stringParser, numParser, boolParser, arrayParser)

	json, err := Parse(valueParser, scanner)
	if err != nil {
		t.Error(err)
	}
	if json.Payload.(map[string]any)["menu"].(map[string]any)["header"] != "SVG Viewer" {
		t.Error("json object payload fail")
	}
	if len(json.Payload.(map[string]any)["menu"].(map[string]any)["items"].([]any)) != 22 {
		t.Error("json array payload fail")
	}
	if json.Payload.(map[string]any)["menu"].(map[string]any)["items"].([]any)[2] != nil {
		t.Error("json nil fail")
	}
}
