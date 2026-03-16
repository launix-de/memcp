package packrat

import "testing"

func TestSkipAtNonWhitespace(t *testing.T) {
	s := NewScanner[int]("SELECT", SkipWhitespaceRegex)
	posBefore := s.position
	s.Skip()
	if s.position != posBefore {
		t.Error("Skip at non-whitespace position should be a no-op")
	}
}

func TestSkipWithLeadingWhitespace(t *testing.T) {
	s := NewScanner[int]("  hello", SkipWhitespaceRegex)
	s.Skip()
	if s.position != 2 {
		t.Errorf("Skip should advance past whitespace, got position %d", s.position)
	}
}

func TestSkipWithComment(t *testing.T) {
	s := NewScanner[int]("/* comment */ hello", SkipWhitespaceAndCommentsRegex)
	s.Skip()
	if s.position != 14 {
		t.Errorf("Skip should advance past comment and following space, got position %d", s.position)
	}
}

func TestSkipAtEndOfInput(t *testing.T) {
	s := NewScanner[int]("x", SkipWhitespaceRegex)
	s.move(1) // move to end
	posBefore := s.position
	s.Skip()
	if s.position != posBefore {
		t.Error("Skip at end of input should be a no-op")
	}
}

func TestSkipNilRegex(t *testing.T) {
	s := NewScanner[int]("  hello", nil)
	posBefore := s.position
	s.Skip()
	if s.position != posBefore {
		t.Error("Skip with nil regex should be a no-op")
	}
}
