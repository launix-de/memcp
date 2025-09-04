// Example Go-side helpers for feature building matching ai_training.py
// This file provides tokenizer + 34-d feature builder. The actual model
// loading via go-torch is shown as commented code to avoid adding deps.
package main

import (
	"math"
	"regexp"
	"strings"
)

const (
	MAX_TOKENS  = 512
	FEATURE_DIM = 16 + 2 + 1 + lenOP // must match ai_training.py
)

var tokenRe = regexp.MustCompile(`\(|\)|\?|>=|<=|<>|!=|=|>|<|\w+|\"(?:[^\"\\]|\\.)*\"`)

var opTokens = []string{
	"(", ")", "and", "or", "not",
	">", "<", ">=", "<=", "=", "!=", "<>",
	"equal?", "equal??", "?", "true", "false",
}

const lenOP = 17

func tokenizeFilterOrder(filter, order string) []string {
	toks := []string{}
	if filter != "" {
		toks = append(toks, tokenRe.FindAllString(filter, -1)...)
	}
	if order != "" {
		parts := strings.Split(order, "|")
		toks = append(toks, parts...)
	}
	return toks
}

func simpleHash16(s string) [16]float32 {
	var v [16]float32
	if s == "" {
		return v
	}
	for i := 0; i < len(s); i++ {
		v[i%16] += float32(s[i]&0xFF) / 128.0
	}
	norm := float32(len(s)) / 16.0
	if norm < 1.0 {
		norm = 1.0
	}
	for i := 0; i < 16; i++ {
		v[i] = v[i] / norm
	}
	return v
}

func buildTokenFeatures(tokens []string, maxTokens int) [][]float32 {
	if maxTokens <= 0 {
		maxTokens = MAX_TOKENS
	}
	out := make([][]float32, maxTokens)
	depth := 0
	for i := 0; i < maxTokens; i++ {
		out[i] = make([]float32, FEATURE_DIM)
	}
	for i, t := range tokens {
		if i >= maxTokens {
			break
		}
		// hash16
		h := simpleHash16(t)
		for k := 0; k < 16; k++ {
			out[i][k] = h[k]
		}
		// numeric
		var val float64
		neg := false
		tt := t
		if strings.HasPrefix(tt, "-") {
			neg = true
			tt = tt[1:]
		}
		isNum := tt != "" && strings.IndexFunc(tt, func(r rune) bool { return r < '0' || r > '9' }) == -1
		if isNum {
			// integer only
			for j := 0; j < len(tt); j++ {
				val = val*10 + float64(tt[j]-'0')
			}
			if neg {
				val = -val
			}
			out[i][16] = float32(val)
			out[i][17] = float32(math.Log1p(math.Abs(val)))
		}
		// paren depth (scaled)
		out[i][18] = float32(depth) / 16.0
		// operator flags
		lt := strings.ToLower(t)
		for j, op := range opTokens {
			if lt == op {
				out[i][19+j] = 1.0
			}
		}
		// update depth
		if t == "(" {
			depth++
		} else if t == ")" && depth > 0 {
			depth--
		}
	}
	return out
}

// The code below shows how to wire go-torch. Keep it commented to avoid
// adding external dependencies here.
/*
import (
    "bytes"
    "github.com/orktes/go-torch/torch"
)

func tensorFrom2D(data [][]float32, shape []int64) *torch.Tensor {
    flat := make([]float32, 0, len(data)*len(data[0]))
    for i := range data { flat = append(flat, data[i]...) }
    return torch.FromBlob(flat, shape, torch.Float32)
}

func runPredict(baseBytes, orderedBytes, headBytes, histBytes []byte, filter, order, schema, table string, inputCount float32, ordered bool) float64 {
    baseMod, _ := torch.Load(bytes.NewReader(baseBytes))
    ordMod,  _ := torch.Load(bytes.NewReader(orderedBytes))
    headMod, _ := torch.Load(bytes.NewReader(headBytes))
    histMod, _ := torch.Load(bytes.NewReader(histBytes))

    tokens := tokenizeFilterOrder(filter, order)
    feats := buildTokenFeatures(tokens, MAX_TOKENS)
    tok := tensorFrom2D(feats, []int64{1, MAX_TOKENS, FEATURE_DIM})
    sch := tensorFrom2D(buildTokenFeatures([]string{schema}, 1), []int64{1, FEATURE_DIM})
    tab := tensorFrom2D(buildTokenFeatures([]string{table}, 1),  []int64{1, FEATURE_DIM})
    inC := torch.FromBlob([]float32{inputCount}, []int64{1}, torch.Float32)

    baseVec, _ := baseMod.Forward(tok, sch, tab, inC)
    histVec, _ := histMod.Forward(baseVec)
    if ordered {
        histVec, _ = ordMod.Forward(histVec)
    }
    logp, _ := headMod.Forward(histVec)
    return math.Expm1(float64(logp.Item().(float32)))
}
*/
