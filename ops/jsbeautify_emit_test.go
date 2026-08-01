package ops

// Direct unit tests for comment attach/emit guard helpers whose branches are
// defensive (unreachable through the normal beautify Run path because their
// callers already guard the same condition, or the input shape cannot arise
// from the scanner). Expected values are computed straight from the escodegen
// transliteration these helpers implement.

import "testing"

// jsNodeKey / jsNodeRange / jsRangeVal must fail closed when a node has a
// missing or malformed range field (attachment keys such nodes as unusable).
func TestJSNodeKeyRangeGuards(t *testing.T) {
	noRange := jsObject{{k: "type", v: "Identifier"}}
	badLen := jsObject{{k: "type", v: "Identifier"}, {k: "range", v: []any{float64(0)}}}

	if got := jsNodeKey(noRange); got != "" {
		t.Errorf("jsNodeKey(no range) = %q, want empty", got)
	}
	if got := jsNodeKey(badLen); got != "" {
		t.Errorf("jsNodeKey(bad range len) = %q, want empty", got)
	}

	if _, _, ok := jsNodeRange(noRange); ok {
		t.Errorf("jsNodeRange(no range) ok = true, want false")
	}
	if _, _, ok := jsNodeRange(badLen); ok {
		t.Errorf("jsNodeRange(bad range len) ok = true, want false")
	}

	// A well-formed range still resolves.
	good := jsObject{{k: "type", v: "Identifier"}, {k: "range", v: []any{float64(2), float64(5)}}}
	if got := jsNodeKey(good); got != "Identifier@2:5" {
		t.Errorf("jsNodeKey(good) = %q, want Identifier@2:5", got)
	}
	if s, e, ok := jsNodeRange(good); !ok || s != 2 || e != 5 {
		t.Errorf("jsNodeRange(good) = (%d,%d,%v), want (2,5,true)", s, e, ok)
	}
}

// jsRangeVal returns -1 for a non-float64 range endpoint (esprima only ever
// emits float64 offsets, so this is the fail-closed default).
func TestJSRangeVal(t *testing.T) {
	if got := jsRangeVal(float64(7)); got != 7 {
		t.Errorf("jsRangeVal(7.0) = %d, want 7", got)
	}
	if got := jsRangeVal("not a number"); got != -1 {
		t.Errorf("jsRangeVal(string) = %d, want -1", got)
	}
	if got := jsRangeVal(nil); got != -1 {
		t.Errorf("jsRangeVal(nil) = %d, want -1", got)
	}
}

// In non-comment mode g.comments is nil. addComments and jsNodeLeading/
// jsNodeTrailing must then be no-ops. generateStatement/generateExpression
// already guard g.comments != nil before calling addComments, and addComments
// guards before jsNodeTrailing, so these nil paths are only reachable directly.
func TestJSGenCommentNilGuards(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " "} // comments == nil
	node := jsObject{{k: "type", v: "Identifier"}, {k: "range", v: []any{float64(0), float64(1)}}}

	if got := g.addComments(node, "x"); got != "x" {
		t.Errorf("addComments(nil comments) = %q, want %q", got, "x")
	}
	if got := g.jsNodeLeading(node); got != nil {
		t.Errorf("jsNodeLeading(nil comments) = %v, want nil", got)
	}
	if got := g.jsNodeTrailing(node); got != nil {
		t.Errorf("jsNodeTrailing(nil comments) = %v, want nil", got)
	}
}

// addTrailingComments has a non-"tailing to statement" branch, taken when the
// node's already-generated result ends in a line terminator (its last child
// carried a trailing line comment). In that case each trailing comment is
// appended via addIndent rather than the indent/specialBase spacing. This shape
// is awkward to elicit through the full pipeline, so it is exercised directly.
func TestAddTrailingCommentsNonStatement(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " "} // base == ""
	trailing := []*jsComment{
		{typ: "Block", value: " t1 "},
		{typ: "Block", value: " t2 "},
	}
	// result ends with a line terminator -> tailingToStatement is false.
	got := g.addTrailingComments(trailing, "a + b\t// c\n")
	// addIndent prepends g.base (empty); a newline is inserted between the two
	// comments because the first does not end with a line terminator.
	want := "a + b\t// c\n/* t1 */\n/* t2 */"
	if got != want {
		t.Errorf("addTrailingComments(non-statement) =\n%q\nwant\n%q", got, want)
	}
}

// generateComment emits Line comments with a trailing newline and Block
// comments delimited by /* */ (the Line-value-ends-in-terminator branch was
// removed as dead: the scanner never includes the terminator in a Line value).
func TestGenerateComment(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " "}
	if got := g.generateComment(&jsComment{typ: "Line", value: " hi"}); got != "// hi\n" {
		t.Errorf("generateComment(Line) = %q, want %q", got, "// hi\n")
	}
	if got := g.generateComment(&jsComment{typ: "Block", value: " hi "}); got != "/* hi */" {
		t.Errorf("generateComment(Block) = %q, want %q", got, "/* hi */")
	}
}
