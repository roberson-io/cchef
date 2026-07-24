package ops

import (
	"regexp"
	"slices"
	"strings"
)

// sigCheck constrains one byte offset within a candidate buffer: the byte must
// fall in [lo, hi], or be one of set when set is non-nil.
type sigCheck struct {
	off int
	lo  int
	hi  int
	set []byte
}

// fileSig is one file-type signature. It matches if ANY of its alternative byte
// patterns (alts) fully matches the buffer.
type fileSig struct {
	name        string
	extension   string
	mime        string
	description string
	alts        [][]sigCheck
}

// sigCategory groups file signatures as in CyberChef's FILE_SIGNATURES object.
type sigCategory struct {
	name  string
	types []fileSig
}

// match reports whether the byte at c.off satisfies the constraint. An offset
// past the end of the buffer never matches (mirrors the JS undefined checks).
func (c sigCheck) match(buf []byte) bool {
	if c.off < 0 || c.off >= len(buf) {
		return false
	}
	b := buf[c.off]
	if c.set != nil {
		return slices.Contains(c.set, b)
	}
	return int(b) >= c.lo && int(b) <= c.hi
}

// sigMatches reports whether any alternative in alts fully matches buf.
func sigMatches(alts [][]sigCheck, buf []byte) bool {
	for _, alt := range alts {
		matched := true
		for _, c := range alt {
			if !c.match(buf) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// detectFileType returns every file signature that matches buf. categories
// filters to those category names; nil means all categories.
func detectFileType(buf []byte, categories []string) []fileSig {
	if len(buf) < 2 {
		return nil
	}
	var out []fileSig
	for _, cat := range fileSignatures {
		if categories != nil && !slices.Contains(categories, cat.name) {
			continue
		}
		for _, ft := range cat.types {
			if sigMatches(ft.alts, buf) {
				out = append(out, ft)
			}
		}
	}
	return out
}

// isTypeString returns the mime of the first detected type whose mime starts
// with prefix, or "" if none.
func isTypeString(prefix string, buf []byte) string {
	for _, t := range detectFileType(buf, nil) {
		if strings.HasPrefix(t.mime, prefix) {
			return t.mime
		}
	}
	return ""
}

// isTypeMatch returns the mime of the first detected type whose mime matches re,
// or "" if none.
func isTypeMatch(re *regexp.Regexp, buf []byte) string {
	for _, t := range detectFileType(buf, nil) {
		if re.MatchString(t.mime) {
			return t.mime
		}
	}
	return ""
}

// isImage returns the image mime of buf, or "" if buf is not a known image.
func isImage(buf []byte) string { return isTypeString("image", buf) }
