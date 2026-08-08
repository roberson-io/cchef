// Package uregex compiles user-supplied regular expressions for the operations
// that accept them. It prefers Go's RE2 engine for its linear-time guarantee and
// falls back to regexp2 in ECMAScript (JavaScript-compatible) mode only when a
// pattern uses a feature RE2 lacks — lookahead, lookbehind, backreferences, or
// JavaScript named-group syntax. The regexp2 path is bounded by a match timeout,
// since it can backtrack.
//
// Both engines are presented through a small Regexp interface that mirrors the
// two shapes the operations need from the standard library: a list of matches
// with their capture-group text, and the same with byte offsets.
package uregex

import (
	"regexp"
	"time"

	"github.com/dlclark/regexp2"
)

// matchTimeout bounds a single regexp2 match. RE2 needs no such guard; this
// applies only to the backtracking fallback, so a pathological pattern fails
// with an error instead of hanging. It is a var only so tests can lower it.
var matchTimeout = 10 * time.Second

// Regexp is the behavior the operations need from a compiled user pattern.
type Regexp interface {
	// MatchString reports whether the pattern matches anywhere in s.
	MatchString(s string) bool
	// ReplaceAll replaces every match with repl, expanding $-group references.
	ReplaceAll(src, repl string) string
	// ReplaceFirst replaces only the first match with repl.
	ReplaceFirst(src, repl string) string
	// FindStringSubmatch returns the first match: element 0 is the whole match
	// and later elements are the capture groups (empty string if a group did not
	// participate). It returns nil when there is no match.
	FindStringSubmatch(s string) []string
	// FindAllStringSubmatch returns every non-overlapping match. In each result,
	// element 0 is the whole match and later elements are the capture groups; a
	// group that did not participate is the empty string.
	FindAllStringSubmatch(s string) [][]string
	// FindAllStringSubmatchIndex returns the same matches as byte-offset pairs:
	// [start,end] of the whole match followed by [start,end] of each group, with
	// -1,-1 for a group that did not participate.
	FindAllStringSubmatchIndex(s string) [][]int
}

// Compile builds a Regexp from a user pattern (which may already carry an inline
// flag prefix such as "(?ims)"). RE2 is tried first; a pattern it cannot compile
// is retried under regexp2's ECMAScript mode.
func Compile(pattern string) (Regexp, error) {
	if re, err := regexp.Compile(pattern); err == nil {
		return re2{re}, nil
	}
	re, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	re.MatchTimeout = matchTimeout
	return re2go{re}, nil
}

// re2 adapts the standard library's engine to the Regexp interface.
type re2 struct{ re *regexp.Regexp }

func (r re2) MatchString(s string) bool {
	return r.re.MatchString(s)
}

func (r re2) ReplaceAll(src, repl string) string {
	return r.re.ReplaceAllString(src, repl)
}

func (r re2) ReplaceFirst(src, repl string) string {
	m := r.re.FindStringSubmatchIndex(src)
	if m == nil {
		return src
	}
	out := []byte(src[:m[0]])
	out = r.re.ExpandString(out, repl, src, m)
	return string(append(out, src[m[1]:]...))
}

func (r re2) FindStringSubmatch(s string) []string {
	return r.re.FindStringSubmatch(s)
}

func (r re2) FindAllStringSubmatch(s string) [][]string {
	return r.re.FindAllStringSubmatch(s, -1)
}

func (r re2) FindAllStringSubmatchIndex(s string) [][]int {
	return r.re.FindAllStringSubmatchIndex(s, -1)
}

// re2go adapts regexp2, translating its rune-indexed matches into the byte
// offsets the operations slice the original string with. regexp2 can return an
// error (a match timeout); as elsewhere in the package that is treated as no
// match, matching how the extractor operations handle it.
type re2go struct{ re *regexp2.Regexp }

func (r re2go) MatchString(s string) bool {
	ok, err := r.re.MatchString(s)
	return err == nil && ok
}

func (r re2go) ReplaceAll(src, repl string) string {
	out, err := r.re.Replace(src, repl, -1, -1)
	if err != nil {
		return src
	}
	return out
}

func (r re2go) ReplaceFirst(src, repl string) string {
	out, err := r.re.Replace(src, repl, -1, 1)
	if err != nil {
		return src
	}
	return out
}

func (r re2go) FindStringSubmatch(s string) []string {
	m, err := r.re.FindStringMatch(s)
	if err != nil || m == nil {
		return nil
	}
	return groupStrings(m)
}

func (r re2go) FindAllStringSubmatch(s string) [][]string {
	var out [][]string
	m, err := r.re.FindStringMatch(s)
	for err == nil && m != nil {
		out = append(out, groupStrings(m))
		m, err = r.re.FindNextMatch(m)
	}
	return out
}

// groupStrings returns a match's whole text and capture groups, with the empty
// string for a group that did not participate.
func groupStrings(m *regexp2.Match) []string {
	groups := m.Groups()
	row := make([]string, len(groups))
	for i := range groups {
		if len(groups[i].Captures) > 0 {
			row[i] = groups[i].String()
		}
	}
	return row
}

func (r re2go) FindAllStringSubmatchIndex(s string) [][]int {
	runeToByte := byteOffsets(s)
	var out [][]int
	m, err := r.re.FindStringMatch(s)
	for err == nil && m != nil {
		groups := m.Groups()
		row := make([]int, 0, len(groups)*2)
		for i := range groups {
			if len(groups[i].Captures) == 0 {
				row = append(row, -1, -1)
				continue
			}
			start := runeToByte[groups[i].Index]
			end := runeToByte[groups[i].Index+groups[i].Length]
			row = append(row, start, end)
		}
		out = append(out, row)
		m, err = r.re.FindNextMatch(m)
	}
	return out
}

// byteOffsets maps a rune index to its byte offset in s; the final entry is
// len(s), so a match ending at the last rune resolves correctly.
func byteOffsets(s string) []int {
	offs := make([]int, 0, len(s)+1)
	for i := range s {
		offs = append(offs, i)
	}
	return append(offs, len(s))
}
