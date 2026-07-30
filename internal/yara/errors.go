package yara

import "fmt"

// compileError is a fault in the rules themselves, carrying the line it was
// found on so it can be reported the way libyara reports one.
type compileError struct {
	line int
	msg  string
}

// Error renders the fault as libyara writes it.
func (e *compileError) Error() string {
	return fmt.Sprintf("Error on line %d: %s", e.line, e.msg)
}

// Line is where in the rules the fault was found.
func (e *compileError) Line() int { return e.line }

// Message is the fault on its own, without the line.
func (e *compileError) Message() string { return e.msg }

// Warning is something libyara notes about a rule without refusing it.
type Warning struct {
	Line    int
	Message string
}

// String renders a warning the way libyara writes one.
func (w Warning) String() string {
	return fmt.Sprintf("Warning on line %d: %s", w.Line, w.Message)
}
