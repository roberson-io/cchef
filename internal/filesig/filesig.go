// Package filesig identifies file types from their magic bytes.
//
// A signature constrains particular byte offsets rather than matching a fixed
// prefix, so a type with several variants is one entry with several
// alternatives. [Detect] answers what a buffer starts with; [Scan] finds every
// position a signature holds at, which is what carving an embedded file needs.
package filesig

import (
	"regexp"
	"slices"
	"sort"
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

// Sig is one file-type signature. It matches if ANY of its alternative byte
// patterns (alts) fully matches the buffer.
type Sig struct {
	Name        string
	Extension   string
	MIME        string
	Description string
	alts        [][]sigCheck
	// Carver names the algorithm that can cut a file of this type out of a
	// larger buffer, or is empty when there is none. carvers maps it to the
	// function; the indirection keeps the generated table free of Go identifiers.
	Carver string
}

// Category groups file signatures as in CyberChef's FILE_SIGNATURES object.
type Category struct {
	Name  string
	Types []Sig
}

// matchAt reports whether the byte at c.off satisfies the constraint, with the
// whole signature shifted along by offset. An offset outside the buffer never
// matches (mirrors the JS undefined checks).
func (c sigCheck) matchAt(buf []byte, offset int) bool {
	pos := c.off + offset
	if pos < 0 || pos >= len(buf) {
		return false
	}
	return c.holds(buf[pos])
}

// holds reports whether one byte satisfies the constraint.
func (c sigCheck) holds(b byte) bool {
	if c.set != nil {
		return slices.Contains(c.set, b)
	}
	return int(b) >= c.lo && int(b) <= c.hi
}

// sigMatches reports whether any alternative in alts fully matches buf.
func sigMatches(alts [][]sigCheck, buf []byte) bool {
	return altMatchesAt(alts, buf, 0) >= 0
}

// altMatchesAt returns the index of the first alternative in alts that matches
// buf at offset, or -1 if none does.
func altMatchesAt(alts [][]sigCheck, buf []byte, offset int) int {
	for i, alt := range alts {
		if altHoldsAt(alt, buf, offset) {
			return i
		}
	}
	return -1
}

// altHoldsAt reports whether every check of one alternative holds at offset.
func altHoldsAt(alt []sigCheck, buf []byte, offset int) bool {
	for _, c := range alt {
		if !c.matchAt(buf, offset) {
			return false
		}
	}
	return true
}

// Detect returns every file signature that matches buf. categories
// filters to those category names; nil means all categories.
func Detect(buf []byte, categories []string) []Sig {
	if len(buf) < 2 {
		return nil
	}
	var out []Sig
	for _, cat := range Signatures {
		if categories != nil && !slices.Contains(categories, cat.Name) {
			continue
		}
		for _, ft := range cat.Types {
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
	for _, t := range Detect(buf, nil) {
		if strings.HasPrefix(t.MIME, prefix) {
			return t.MIME
		}
	}
	return ""
}

// IsTypeMatch returns the mime of the first detected type whose mime matches re,
// or "" if none.
func IsTypeMatch(re *regexp.Regexp, buf []byte) string {
	for _, t := range Detect(buf, nil) {
		if re.MatchString(t.MIME) {
			return t.MIME
		}
	}
	return ""
}

// IsImage returns the image mime of buf, or "" if buf is not a known image.
func IsImage(buf []byte) string { return isTypeString("image", buf) }

// Match is one signature match found by scanForFileTypes, at the offset the
// file it describes would start at.
type Match struct {
	Offset  int
	Details Sig
}

// Scan reports every position in buf at which a signature matches,
// in order of increasing offset. detectFileType only asks what the buffer starts
// with; this is what carving needs, since an embedded file can begin anywhere.
// categories filters to those category names; nil means all categories.
func Scan(buf []byte, categories []string) []Match {
	if len(buf) < 2 {
		return nil
	}
	var found []Match
	for _, cat := range Signatures {
		if categories != nil && !slices.Contains(categories, cat.Name) {
			continue
		}
		for _, ft := range cat.Types {
			found = appendSigMatches(found, buf, ft)
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].Offset < found[j].Offset })
	return found
}

// appendSigMatches adds every position at which one file type's signature
// matches buf. Each alternative is walked separately, as CyberChef does, so a
// type whose alternatives overlap is reported once per alternative that holds.
func appendSigMatches(found []Match, buf []byte, ft Sig) []Match {
	for _, alt := range ft.alts {
		for pos := 0; ; pos++ {
			pos = locatePotentialSig(buf, alt, pos)
			if pos < 0 {
				break
			}
			if altHoldsAt(alt, buf, pos) {
				found = append(found, Match{Offset: pos, Details: ft})
			}
		}
	}
	return found
}

// locatePotentialSig returns the next offset at or after from where the
// alternative's first check could hold, or -1 if there is none. The checks are
// ordered by the byte they constrain, so finding a candidate for the first one
// and stepping back by its offset gives the position the file would start at.
func locatePotentialSig(buf []byte, alt []sigCheck, from int) int {
	first := alt[0]
	for i := from + first.off; i < len(buf); i++ {
		if i >= 0 && first.holds(buf[i]) {
			return i - first.off
		}
	}
	return -1
}
