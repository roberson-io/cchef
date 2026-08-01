package ops

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

// fileSig is one file-type signature. It matches if ANY of its alternative byte
// patterns (alts) fully matches the buffer.
type fileSig struct {
	name        string
	extension   string
	mime        string
	description string
	alts        [][]sigCheck
	// carver names the algorithm that can cut a file of this type out of a
	// larger buffer, or is empty when there is none. carvers maps it to the
	// function; the indirection keeps the generated table free of Go identifiers.
	carver string
}

// sigCategory groups file signatures as in CyberChef's FILE_SIGNATURES object.
type sigCategory struct {
	name  string
	types []fileSig
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

// foundFile is one signature match found by scanForFileTypes, at the offset the
// file it describes would start at.
type foundFile struct {
	offset  int
	details fileSig
}

// scanForFileTypes reports every position in buf at which a signature matches,
// in order of increasing offset. detectFileType only asks what the buffer starts
// with; this is what carving needs, since an embedded file can begin anywhere.
// categories filters to those category names; nil means all categories.
func scanForFileTypes(buf []byte, categories []string) []foundFile {
	if len(buf) < 2 {
		return nil
	}
	var found []foundFile
	for _, cat := range fileSignatures {
		if categories != nil && !slices.Contains(categories, cat.name) {
			continue
		}
		for _, ft := range cat.types {
			found = appendSigMatches(found, buf, ft)
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].offset < found[j].offset })
	return found
}

// appendSigMatches adds every position at which one file type's signature
// matches buf. Each alternative is walked separately, as CyberChef does, so a
// type whose alternatives overlap is reported once per alternative that holds.
func appendSigMatches(found []foundFile, buf []byte, ft fileSig) []foundFile {
	for _, alt := range ft.alts {
		for pos := 0; ; pos++ {
			pos = locatePotentialSig(buf, alt, pos)
			if pos < 0 {
				break
			}
			if altHoldsAt(alt, buf, pos) {
				found = append(found, foundFile{offset: pos, details: ft})
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
