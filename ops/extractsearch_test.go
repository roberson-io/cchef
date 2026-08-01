package ops

import (
	"reflect"
	"testing"

	"github.com/dlclark/regexp2"
)

// TestExtractSearch covers the walk the extractor operations share: every match
// of a pattern, optionally with some removed, sorted, and reduced to one of
// each.
func TestExtractSearch(t *testing.T) {
	words := regexp2.MustCompile(`[a-z]+`, regexp2.IgnoreCase)

	for _, tc := range []struct {
		name   string
		input  string
		remove *regexp2.Regexp
		less   func(a, b string) bool
		unique bool
		want   []string
	}{
		{"every match", "one two three", nil, nil, false, []string{"one", "two", "three"}},
		{"nothing to match", "1 2 3", nil, nil, false, nil},
		{
			"repeats kept", "b a b a", nil, nil, false,
			[]string{"b", "a", "b", "a"},
		},
		{
			"repeats removed", "b a b a", nil, nil, true,
			[]string{"b", "a"},
		},
		{
			"sorted", "b c a", nil, caseInsensitiveLess, false,
			[]string{"a", "b", "c"},
		},
		{
			"sorted and uniqued", "b a b", nil, caseInsensitiveLess, true,
			[]string{"a", "b"},
		},
		{
			"some removed", "keep drop keep", regexp2.MustCompile(`^drop$`, regexp2.None), nil, false,
			[]string{"keep", "keep"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSearch(tc.input, words, tc.remove, tc.less, tc.unique)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestExtractSearchEmptyMatches covers a pattern that can match nothing at all.
// The walk has to step forward when it does, or it would stand still forever.
func TestExtractSearchEmptyMatches(t *testing.T) {
	empty := regexp2.MustCompile(`a*`, regexp2.None)
	got := extractSearch("bab", empty, nil, nil, false)
	if len(got) == 0 {
		t.Fatal("no matches at all")
	}
	for _, m := range got {
		if m != "" && m != "a" {
			t.Errorf("unexpected match %q", m)
		}
	}
}

// TestExtractResult covers the two shapes the operations report their findings
// in: the matches alone, or a count before them.
func TestExtractResult(t *testing.T) {
	found := []string{"one", "two"}

	if got := extractResult(found, false); got != "one\ntwo" {
		t.Errorf("got %q", got)
	}
	if got := extractResult(found, true); got != "Total found: 2\n\none\ntwo" {
		t.Errorf("got %q", got)
	}
	if got := extractResult(nil, false); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := extractResult(nil, true); got != "Total found: 0\n\n" {
		t.Errorf("got %q", got)
	}
}

// TestCaseInsensitiveLess covers the ordering the text extractors sort by, which
// ignores case.
func TestCaseInsensitiveLess(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"apple", "Banana", true},
		{"Banana", "apple", false},
		{"a", "a", false},
		{"A", "a", false},
	} {
		if got := caseInsensitiveLess(tc.a, tc.b); got != tc.want {
			t.Errorf("caseInsensitiveLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestExtractIPLess covers the ordering IP addresses sort by, which reads each
// as the number its four parts make. Anything that does not read as one sorts
// after those that do, among themselves in text order.
func TestExtractIPLess(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b string
		want bool
	}{
		{"lower first", "1.2.3.4", "1.2.3.5", true},
		{"the first part carries the most weight", "2.0.0.0", "1.255.255.255", false},
		{"equal", "1.2.3.4", "1.2.3.4", false},
		{"a leading zero does not make it octal", "010.1.1.1", "9.1.1.1", false},
		{"text sorts after a number", "not.an.ip.here", "1.2.3.4", false},
		{"a number sorts before text", "1.2.3.4", "not.an.ip.here", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractIPLess(tc.a, tc.b); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestExtractIPValueRejects covers the addresses that do not read as a number,
// which sort after those that do.
func TestExtractIPValueRejects(t *testing.T) {
	for _, s := range []string{"1.2.3", "1.2.3.4.5", "a.b.c.d", "1.2.3.x", ""} {
		if _, ok := extractIPValue(s); ok {
			t.Errorf("%q was read as a number", s)
		}
	}
	if v, ok := extractIPValue("1.0.0.0"); !ok || v != 0x1000000 {
		t.Errorf("got %v, %v", v, ok)
	}
}

// TestExtractIPLessBothUnreadable covers two values that neither read as a
// number, which fall back to text order.
func TestExtractIPLessBothUnreadable(t *testing.T) {
	if !extractIPLess("a.b.c.d", "b.c.d.e") {
		t.Error("two unreadable addresses did not fall back to text order")
	}
	if extractIPLess("b.c.d.e", "a.b.c.d") {
		t.Error("text order was reversed")
	}
}
