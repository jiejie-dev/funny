package ast

import "fmt"

// TestBlock is a named unit test: `test "name":` followed by a body block.
type TestBlock struct {
	NodePos Pos
	Name    string
	Body    *Block
}

func (s *TestBlock) Pos() Pos    { return s.NodePos }
func (s *TestBlock) stmtMarker() {}
func (s *TestBlock) nodeMarker() {}
func (s *TestBlock) String() string {
	return fmt.Sprintf("test %q:\n%s", s.Name, s.Body.String())
}
