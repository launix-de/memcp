package packrat

import "testing"

func TestBreaksWordBoundaries(t *testing.T) {
	s := NewScanner[int]("hello world", nil)
	// Position 0: start of input, always a break
	if !s.breaks[0] {
		t.Error("Position 0 should be a break")
	}
	// Position 5: space after 'hello' (non-word), should be break
	if !s.breaks[5] {
		t.Error("Position 5 (space) should be a break")
	}
	// Position 6: 'w' after space (word start), should be break
	if !s.breaks[6] {
		t.Error("Position 6 (word start) should be a break")
	}
	// Position 1: 'e' after 'h' (word-to-word), NOT a break
	if s.breaks[1] {
		t.Error("Position 1 (mid-word) should not be a break")
	}
	// Position len: end of input, always a break
	if !s.breaks[len("hello world")] {
		t.Error("End position should be a break")
	}
}

func TestBreaksUnicode(t *testing.T) {
	s := NewScanner[int]("abc_123 über", nil)
	// 'a' at 0: always break
	if !s.breaks[0] {
		t.Error("Position 0 should be a break")
	}
	// '_' (Pc category) is a word char; position within abc_ should not be break
	if s.breaks[2] {
		t.Error("Position 2 (mid-word 'c') should not be a break")
	}
	// Position 3 ('_'): word after word, no break
	if s.breaks[3] {
		t.Error("Position 3 ('_') should not be a break (word continues)")
	}
	// Space at position 7 is non-word, so it's a break
	if !s.breaks[7] {
		t.Error("Position 7 (space) should be a break")
	}
}

func TestBreaksEmptyInput(t *testing.T) {
	s := NewScanner[int]("", nil)
	if len(s.breaks) != 1 {
		t.Errorf("Empty input should have breaks slice of length 1, got %d", len(s.breaks))
	}
	if !s.breaks[0] {
		t.Error("Position 0 of empty input should be a break")
	}
}

func TestBreaksDigits(t *testing.T) {
	s := NewScanner[int]("42+7", nil)
	// '4' at 0: always break
	if !s.breaks[0] {
		t.Error("Position 0 should be a break")
	}
	// '2' at 1: digit after digit, no break
	if s.breaks[1] {
		t.Error("Position 1 (digit after digit) should not be a break")
	}
	// '+' at 2: non-word after word, break
	if !s.breaks[2] {
		t.Error("Position 2 ('+' after digit) should be a break")
	}
	// '7' at 3: word after non-word, break
	if !s.breaks[3] {
		t.Error("Position 3 ('7' after '+') should be a break")
	}
}
