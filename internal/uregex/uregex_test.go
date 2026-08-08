package uregex

import (
	"reflect"
	"testing"
	"time"
)

// A pattern RE2 accepts uses the standard-library engine; one it rejects (here a
// lookbehind) uses the regexp2 fallback. Both must present the same shapes.
func TestCompileEnginePaths(t *testing.T) {
	direct, err := Compile(`\d+`)
	if err != nil {
		t.Fatalf("RE2 compile: %v", err)
	}
	if _, ok := direct.(re2); !ok {
		t.Errorf("plain pattern should use the RE2 engine, got %T", direct)
	}
	fallback, err := Compile(`(?<=x)\d+`)
	if err != nil {
		t.Fatalf("fallback compile: %v", err)
	}
	if _, ok := fallback.(re2go); !ok {
		t.Errorf("lookbehind should use the regexp2 fallback, got %T", fallback)
	}
}

func TestFindAllStringSubmatch(t *testing.T) {
	cases := []struct {
		name, pattern, input string
		want                 [][]string
	}{
		{"re2 groups", `(\w)(\d)`, "a1 b2", [][]string{{"a1", "a", "1"}, {"b2", "b", "2"}}},
		{"lookahead", `foo(?=bar)`, "foobar foobaz", [][]string{{"foo"}}},
		{"lookbehind group", `(?<=\{)(\d+)(?=\})`, "{12}{34}", [][]string{{"12", "12"}, {"34", "34"}}},
		{"backreference", `(\w)\1`, "aabb", [][]string{{"aa", "a"}, {"bb", "b"}}},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern)
		if err != nil {
			t.Fatalf("%s: compile: %v", c.name, err)
		}
		if got := re.FindAllStringSubmatch(c.input); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: FindAllStringSubmatch = %v; want %v", c.name, got, c.want)
		}
	}
}

func TestFindStringSubmatch(t *testing.T) {
	cases := []struct {
		name, pattern, input string
		want                 []string
	}{
		{"re2 first match", `(\w)(\d)`, "a1 b2", []string{"a1", "a", "1"}},
		{"re2 no match", `z(\d)`, "a1", nil},
		{"fallback first match", `(?<=#)(\w+)`, "x #abc #def", []string{"abc", "abc"}},
		{"fallback no match", `(?<=#)(\w+)`, "no hash here", nil},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern)
		if err != nil {
			t.Fatalf("%s: compile: %v", c.name, err)
		}
		if got := re.FindStringSubmatch(c.input); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: FindStringSubmatch = %v; want %v", c.name, got, c.want)
		}
	}
}

// The fallback reports byte offsets even though regexp2 matches on runes, and a
// group that does not participate is reported as -1,-1.
func TestFindAllStringSubmatchIndexFallback(t *testing.T) {
	re, err := Compile(`(?<=€)(\d+)`) // € is three bytes
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got := re.FindAllStringSubmatchIndex("€100 €25")
	// "€100" -> digits at bytes 3..6; " €25" -> € starts at byte 7, digits 10..12.
	want := [][]int{{3, 6, 3, 6}, {10, 12, 10, 12}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAllStringSubmatchIndex = %v; want %v", got, want)
	}
	// Slicing the input with the reported offsets returns the matched text.
	in := "€100 €25"
	if s := in[got[0][2]:got[0][3]]; s != "100" {
		t.Errorf("offset slice = %q; want 100", s)
	}
}

func TestFindAllStringSubmatchIndexNonParticipatingGroup(t *testing.T) {
	// An alternation where only one branch's group participates per match.
	re, err := Compile(`(?<=#)(?:(a)|(b))`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got := re.FindAllStringSubmatch("#a #b")
	want := [][]string{{"a", "a", ""}, {"b", "", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAllStringSubmatch = %v; want %v", got, want)
	}
	idx := re.FindAllStringSubmatchIndex("#a #b")
	// First match: group 2 absent (-1,-1); second match: group 1 absent.
	if idx[0][4] != -1 || idx[0][5] != -1 {
		t.Errorf("first match group 2 = %v; want -1,-1", idx[0][4:6])
	}
	if idx[1][2] != -1 || idx[1][3] != -1 {
		t.Errorf("second match group 1 = %v; want -1,-1", idx[1][2:4])
	}
}

// The RE2 path reports byte offsets directly from the standard library.
func TestFindAllStringSubmatchIndexRE2(t *testing.T) {
	re, err := Compile(`\d+`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got := re.FindAllStringSubmatchIndex("a1 b22")
	want := [][]int{{1, 2}, {4, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAllStringSubmatchIndex = %v; want %v", got, want)
	}
}

func TestMatchString(t *testing.T) {
	cases := []struct {
		pattern, input string
		want           bool
	}{
		{`\d+`, "abc123", true},     // RE2 path
		{`\d+`, "abc", false},       // RE2 path, no match
		{`(?<=x)\d+`, "x42", true},  // fallback path
		{`(?<=x)\d+`, "y42", false}, // fallback path, no match
	}
	for _, c := range cases {
		re, err := Compile(c.pattern)
		if err != nil {
			t.Fatalf("compile %q: %v", c.pattern, err)
		}
		if got := re.MatchString(c.input); got != c.want {
			t.Errorf("MatchString(%q) with %q = %v; want %v", c.input, c.pattern, got, c.want)
		}
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		name, pattern, input, repl, all, first string
	}{
		{"re2", `(\w)(\d)`, "a1 b2", "$2$1", "1a 2b", "1a b2"},
		{"re2 no match", `z\d`, "abc", "X", "abc", "abc"},
		{"fallback lookahead", `(\w+)(?=@)`, "a@x b@y", "<$1>", "<a>@x <b>@y", "<a>@x b@y"},
		{"fallback no match", `(?<=z)\d`, "abc", "X", "abc", "abc"},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern)
		if err != nil {
			t.Fatalf("%s: compile: %v", c.name, err)
		}
		if got := re.ReplaceAll(c.input, c.repl); got != c.all {
			t.Errorf("%s: ReplaceAll = %q; want %q", c.name, got, c.all)
		}
		if got := re.ReplaceFirst(c.input, c.repl); got != c.first {
			t.Errorf("%s: ReplaceFirst = %q; want %q", c.name, got, c.first)
		}
	}
}

// TestMatchTimeout checks the ReDoS guard: a catastrophically backtracking
// fallback pattern is stopped by the match timeout, and each method degrades
// safely (no match / input unchanged) rather than hanging.
func TestMatchTimeout(t *testing.T) {
	old := matchTimeout
	matchTimeout = time.Millisecond
	defer func() { matchTimeout = old }()

	// A lookahead forces the regexp2 fallback; the inner (a+)+ backtracks
	// forever against an input with no "b".
	re, err := Compile(`(?=(a+)+b)`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if re.MatchString(input) {
		t.Error("MatchString: want false on timeout")
	}
	if got := re.ReplaceAll(input, "X"); got != input {
		t.Errorf("ReplaceAll on timeout = %q; want input unchanged", got)
	}
	if got := re.ReplaceFirst(input, "X"); got != input {
		t.Errorf("ReplaceFirst on timeout = %q; want input unchanged", got)
	}
}

func TestCompileInvalidInBothEngines(t *testing.T) {
	if _, err := Compile(`(?<=`); err == nil {
		t.Error("expected an error for a pattern invalid in both engines")
	}
}
