package yara

import (
	"crypto/sha1" // #nosec G505 -- the mark of a certificate is defined as a SHA-1
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The certificates a signed file carries. A file names a table of blobs, each
// saying who signed it and carrying the certificates of those signers, and it
// is those signers' certificates a rule can ask about.

const (
	// directoryCertificates names the table of certificates. Alone among the
	// entries of the table of tables, what it holds is a place in the file
	// rather than an address in memory.
	directoryCertificates = 4
	// certHeaderSize is what opens each entry of that table: how long the entry
	// is, which form it is written in, and what kind of thing it holds.
	certHeaderSize = 8
	// The forms an entry may be written in. Only the second is read; the first
	// is stepped over and anything else ends the walk.
	certRevision1 = 0x0100
	certRevision2 = 0x0200
	// certTypePKCS7 is the only kind of thing read out of an entry.
	certTypePKCS7 = 2
	// certAlignment is what the next entry begins on.
	certAlignment = 8
	// maxCertificates is as many as are offered, however many a file holds.
	maxCertificates = 16
	// onelineMax is how long a written-out name may grow. It is one short of
	// the room set aside for it, which the name is closed off within.
	onelineMax = 255
	// serialMax is the most bytes a certificate's number may take before it is
	// passed over rather than written out.
	serialMax = 20
	// unknownAlgorithm is what a certificate signed some way with no name is
	// said to be signed by.
	unknownAlgorithm = "UNDEF"
	// oidSignedBlob names the one shape of blob that is read: the one saying
	// who signed something.
	oidSignedBlob = "1.2.840.113549.1.7.2"
	// oidNestedSignature names the part of a signer's own account of itself
	// that may carry another whole signature.
	oidNestedSignature = "1.3.6.1.4.1.311.2.4.1"
)

// signatureSchema adds to what the pe module declares the parts to do with the
// certificates that signed the file.
func signatureSchema(members map[string]*modDecl) {
	members["number_of_signatures"] = decInt()
	members["signatures"] = decArray(decStruct(map[string]*modDecl{
		"thumbprint": decString(), "issuer": decString(), "subject": decString(),
		"version": decInt(), "algorithm": decString(),
		"algorithm_oid": decString(), "serial": decString(),
		"not_before": decInt(), "not_after": decInt(),
		"valid_on": decFunc(modInt, "i"),
	}))
}

// peCertificate is one certificate belonging to somebody who signed the file.
type peCertificate struct {
	thumbprint              string
	issuer, subject         string
	version                 int64
	algorithm, algorithmOID string
	serial                  string
	hasSerial               bool
	notBefore, notAfter     int64
	knowsBefore, knowsAfter bool
}

// addSignatures reads the certificates that signed the file and offers what a
// rule may ask about each.
func (f *peFile) addSignatures(fields map[string]modValue) {
	at, size, ok := f.certificateTable()
	if !ok {
		return
	}
	// From here the question has an answer, even if the table turns out to hold
	// nothing worth reading.
	var certs []peCertificate
	for _, entry := range f.certificateEntries(at, size) {
		readSigned(entry, &certs)
		if len(certs) >= maxCertificates {
			break
		}
	}

	items := make([]modValue, 0, len(certs))
	for _, c := range certs {
		items = append(items, c.value())
	}
	fields["signatures"] = listOf(items)
	fields["number_of_signatures"] = valueOf(intValue(int64(len(certs))))
}

// value is what a rule sees of one certificate. A part the certificate does not
// say is left out, so that asking after it gets no answer.
func (c peCertificate) value() modValue {
	member := map[string]modValue{
		"thumbprint": valueOf(stringValue(c.thumbprint)),
		"issuer":     valueOf(stringValue(c.issuer)),
		"subject":    valueOf(stringValue(c.subject)),
		"version":    valueOf(intValue(c.version)),
		"algorithm":  valueOf(stringValue(c.algorithm)),
		// The number of how it was signed is written out only for a way that
		// has a name, since it is looked up by that name rather than kept.
		"algorithm_oid": valueOf(stringValue(c.algorithmOID)),
		"valid_on":      c.validOn(),
	}
	if c.hasSerial {
		member["serial"] = valueOf(stringValue(c.serial))
	}
	if c.knowsBefore {
		member["not_before"] = valueOf(intValue(c.notBefore))
	}
	if c.knowsAfter {
		member["not_after"] = valueOf(intValue(c.notAfter))
	}
	return structOf(member)
}

// validOn answers whether the certificate held good at a given moment, counting
// both of its ends as within. A certificate whose ends are not known, or a
// moment that is not itself a number, leaves the question unanswered.
func (c peCertificate) validOn() modValue {
	return funcOf(func(_ *evaluator, args []value) (value, error) {
		if !c.knowsBefore || !c.knowsAfter ||
			len(args) != 1 || args[0].kind != valueInt {
			return undefined, nil
		}
		// The answer is given as a number rather than as a plain yes or no,
		// which is what a rule comparing it against one expects.
		when := args[0].i
		if when >= c.notBefore && when <= c.notAfter {
			return intValue(1), nil
		}
		return intValue(0), nil
	})
}

// certificateTable finds the table of certificates: where in the file it begins
// and how far it runs. A file whose table of tables does not reach that far
// leaves the whole question unanswered.
func (f *peFile) certificateTable() (at, size int, ok bool) {
	begins, length, named := f.directoryEntry(directoryCertificates)
	if !named && !f.namesDirectory(directoryCertificates) {
		return 0, 0, false
	}
	// A table that begins nowhere, or that does not lie within the file, is
	// read as holding nothing rather than as not being there at all.
	if begins == 0 || int64(begins) > int64(len(f.data)) ||
		int64(length) > int64(len(f.data)) ||
		int64(begins)+int64(length) > int64(len(f.data)) {
		return 0, 0, true
	}
	return int(begins), int(length), true
}

// namesDirectory says whether the file claims a given entry of the table of
// tables and has room for it, whatever that entry then holds.
func (f *peFile) namesDirectory(which uint32) bool {
	count, ok := f.u32(f.dataDirectoriesAt() - 4)
	if !ok || which >= min(count, maxDataDirectories) {
		return false
	}
	at := f.dataDirectoriesAt() + int(which)*dataDirectoryEntry
	_, endOK := f.u32(at + 4)
	return endOK
}

// certificateEntries walks the table and gives back the blob each entry holds.
// An entry written in the older form, or holding something other than a
// signature, is stepped over; anything the walk cannot make sense of ends it.
func (f *peFile) certificateEntries(at, size int) [][]byte {
	var out [][]byte
	for end := at + size; ; {
		if at < 0 || at+certHeaderSize > len(f.data) {
			return out
		}
		length, _ := f.u32(at)
		if int64(length) <= certHeaderSize ||
			int64(at)+int64(length) > int64(len(f.data)) ||
			at+certHeaderSize >= end || int64(at)+int64(length) > int64(end) {
			return out
		}
		revision, _ := f.u16(at + 4)
		kind, _ := f.u16(at + 6)
		if revision != certRevision1 && revision != certRevision2 {
			return out
		}
		if revision == certRevision2 && kind == certTypePKCS7 {
			out = append(out, f.data[at+certHeaderSize:at+int(length)])
		}
		at = (at + int(length) + certAlignment - 1) & ^(certAlignment - 1)
	}
}

// readSigned reads the signers out of one entry, which may hold more than one
// blob laid one after another.
func readSigned(entry []byte, out *[]peCertificate) {
	for len(entry) > 0 && len(*out) < maxCertificates {
		blob, next, ok := readDER(entry)
		if !ok {
			return
		}
		readSigners(blob, out)
		entry = next
	}
}

// readSigners adds the certificates of everyone who signed a blob, and of
// everyone who signed any signature nested within it. A blob naming a signer
// whose certificate it does not carry is left unread altogether.
func readSigners(blob derElement, out *[]peCertificate) {
	if len(*out) >= maxCertificates {
		return
	}
	signed, ok := signedBody(blob)
	if !ok {
		return
	}
	carried, signers, ok := signedParts(signed)
	if !ok {
		return
	}
	// Every signer has to be accounted for before any of them is offered: a
	// blob naming one whose certificate it does not carry is left unread.
	found := make([]peCertificate, 0, len(signers))
	for _, signer := range signers {
		cert, ok := matchingCertificate(carried, signer)
		if !ok {
			return
		}
		found = append(found, cert)
	}
	for _, cert := range found {
		if len(*out) == maxCertificates {
			break
		}
		*out = append(*out, cert)
	}
	// Only the first signer's own account of itself is looked in for another
	// whole signature.
	if len(signers) > 0 {
		for _, inner := range nestedSignatures(signers[0]) {
			readSigners(inner, out)
		}
	}
}

// signedBody picks the account of who signed something out of a blob that
// names what it holds and then holds it.
func signedBody(blob derElement) (derElement, bool) {
	parts := derParts(blob.body)
	if len(parts) < 2 || !parts[0].named(derUniversal, derOID) ||
		oidText(parts[0].body) != oidSignedBlob {
		return derElement{}, false
	}
	inner := derParts(parts[1].body)
	if len(inner) != 1 {
		return derElement{}, false
	}
	return inner[0], true
}

// signedParts picks out the certificates a signed account carries and the
// signers it names. The certificates are written where nothing else can be,
// and the signers are the last thing it holds.
func signedParts(signed derElement) (carried, signers []derElement, ok bool) {
	parts := derParts(signed.body)
	// A version, the ways things were reduced, what was signed, and the
	// signers; anything shorter is not one of these at all.
	const leastParts = 4
	if len(parts) < leastParts {
		return nil, nil, false
	}
	for _, p := range parts {
		if p.named(derContext, 0) {
			carried = derParts(p.body)
		}
	}
	return carried, derParts(parts[len(parts)-1].body), true
}

// matchingCertificate finds the certificate a signer points at, which it names
// by who gave it out and the number it was given.
func matchingCertificate(carried []derElement, signer derElement) (peCertificate, bool) {
	parts := derParts(signer.body)
	if len(parts) < 2 {
		return peCertificate{}, false
	}
	named := derParts(parts[1].body)
	if len(named) < 2 {
		return peCertificate{}, false
	}
	issuer, serial := named[0].raw, trimNumber(named[1].body)

	for _, held := range carried {
		if !held.named(derUniversal, derSequence) {
			continue
		}
		cert, ok := readCertificate(held)
		if !ok {
			continue
		}
		if string(cert.issuerRaw) == string(issuer) &&
			string(cert.serialRaw) == string(serial) {
			return cert.peCertificate, true
		}
	}
	return peCertificate{}, false
}

// nestedSignatures picks any whole signatures out of a signer's own account of
// itself, where they are kept alongside what it says of the thing it signed.
func nestedSignatures(signer derElement) []derElement {
	var out []derElement
	for _, part := range derParts(signer.body) {
		if !part.named(derContext, 1) {
			continue
		}
		for _, attribute := range derParts(part.body) {
			held := derParts(attribute.body)
			if len(held) != 2 || !held[0].named(derUniversal, derOID) ||
				oidText(held[0].body) != oidNestedSignature {
				continue
			}
			// However many are carried here, reading them stops of its own
			// accord once as many certificates as are offered have been found.
			out = append(out, derParts(held[1].body)...)
		}
	}
	return out
}

// readCertificate is a certificate read out, kept alongside the bytes a signer
// points at it by.
type readCertificateResult struct {
	peCertificate
	issuerRaw, serialRaw []byte
}

// readCertificate reads what a rule can ask about out of one certificate.
func readCertificate(cert derElement) (readCertificateResult, bool) {
	outer := derParts(cert.body)
	// What it says of itself, how it was signed, and the signing.
	const outerParts = 3
	if len(outer) != outerParts {
		return readCertificateResult{}, false
	}
	said := derParts(outer[0].body)
	// The version is written where nothing else can be, and one too old to say
	// which it is counts as the first.
	var version int64
	if len(said) > 0 && said[0].named(derContext, 0) {
		if inner := derParts(said[0].body); len(inner) == 1 {
			version = numberValue(inner[0].body)
		}
		said = said[1:]
	}
	// Its number, how it was signed, who gave it out, how long it holds good,
	// and who it belongs to.
	const leastSaid = 5
	if len(said) < leastSaid {
		return readCertificateResult{}, false
	}

	sum := sha1.Sum(cert.raw) // #nosec G401 -- the mark is defined as a SHA-1
	out := readCertificateResult{
		peCertificate: peCertificate{
			thumbprint: hex.EncodeToString(sum[:]),
			// A version counts from nought, so it is said one higher.
			version: version + 1,
			issuer:  onelineName(said[2].body),
			subject: onelineName(said[4].body),
		},
		issuerRaw: said[2].raw,
		serialRaw: trimNumber(said[0].body),
	}
	out.serial, out.hasSerial = serialText(out.serialRaw)
	out.algorithm, out.algorithmOID = algorithmNames(outer[1])
	if when := derParts(said[3].body); len(when) == 2 {
		out.notBefore, out.knowsBefore = certificateTime(when[0])
		out.notAfter, out.knowsAfter = certificateTime(when[1])
	}
	return out, true
}

// algorithmNames says how a certificate was signed, by the full name of that
// way and by its number. A way with no name has no number written out either,
// since the number is looked up by the name rather than kept.
func algorithmNames(algorithm derElement) (name, oid string) {
	parts := derParts(algorithm.body)
	if len(parts) == 0 || !parts[0].named(derUniversal, derOID) {
		return unknownAlgorithm, ""
	}
	dotted := oidText(parts[0].body)
	if full, known := signatureAlgorithms[dotted]; known {
		return full, dotted
	}
	return unknownAlgorithm, ""
}

// certificateTime reads a moment as a certificate writes one: digits running
// year to second, with the year given either in full or by its last two digits.
// Anything said after the digits about where in the world it is meant is passed
// over, so every moment is read as being at Greenwich.
func certificateTime(when derElement) (int64, bool) {
	text := when.body
	var year, at int
	switch {
	case when.named(derUniversal, derUTCTime):
		if len(text) < 12 {
			return 0, false
		}
		year, at = digits(text, 0, 2), 2
		// A year below seventy is of this century rather than the last.
		if year < 70 {
			year += 100
		}
		year += 1900
	case when.named(derUniversal, derGeneralTime):
		if len(text) < 14 {
			return 0, false
		}
		year, at = digits(text, 0, 4), 4
	default:
		return 0, false
	}
	moment := time.Date(year, time.Month(digits(text, at, 2)), digits(text, at+2, 2),
		digits(text, at+4, 2), digits(text, at+6, 2), digits(text, at+8, 2), 0, time.UTC)
	return moment.Unix(), true
}

// digits reads a number written out as characters. A character that is not a
// digit is taken at what it is worth against nought, which lets a moment
// written oddly still come to something rather than nothing.
func digits(text []byte, at, count int) int {
	n := 0
	for i := at; i < at+count; i++ {
		n = n*10 + int(text[i]) - '0'
	}
	return n
}

// numberValue reads a number written as bytes, highest first, as a count.
func numberValue(b []byte) int64 {
	var n int64
	for _, c := range b {
		n = n<<8 | int64(c)
	}
	return n
}

// trimNumber takes off the leading bytes a number does not need, so that two
// writings of the same number come to the same bytes.
func trimNumber(b []byte) []byte {
	for len(b) > 1 {
		switch {
		case b[0] == 0x00 && b[1]&0x80 == 0:
		case b[0] == 0xFF && b[1]&0x80 != 0:
		default:
			return b
		}
		b = b[1:]
	}
	return b
}

// serialText writes the number a certificate was given as pairs of characters
// with colons between. A number needing no bytes at all, or more than there may
// be, is passed over.
func serialText(b []byte) (string, bool) {
	if len(b) == 0 || len(b) > serialMax {
		return "", false
	}
	var out strings.Builder
	for i, c := range b {
		if i > 0 {
			out.WriteByte(':')
		}
		fmt.Fprintf(&out, "%02x", c)
	}
	return out.String(), true
}

// oidText writes a numbered object out in the dotted form. The first two parts
// share a byte and the rest follow seven bits at a time.
func oidText(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// The first byte holds the first part times forty plus the second, and the
	// first part is never more than two.
	const firstMax, perFirst = 2, 40
	first, second := int(b[0])/perFirst, int(b[0])%perFirst
	if first > firstMax {
		first, second = firstMax, int(b[0])-firstMax*perFirst
	}
	var out strings.Builder
	out.WriteString(strconv.Itoa(first))
	out.WriteByte('.')
	out.WriteString(strconv.Itoa(second))

	var part uint64
	var carrying bool
	for _, c := range b[1:] {
		// A part is not read past what can be held without losing bits.
		const roomLeft = 1 << 56
		if part >= roomLeft {
			return ""
		}
		part = part<<7 | uint64(c&0x7F)
		carrying = c&0x80 != 0
		if !carrying {
			out.WriteByte('.')
			out.WriteString(strconv.FormatUint(part, 10))
			part = 0
		}
	}
	if carrying {
		return ""
	}
	return out.String()
}

// Writing out the name of whoever a certificate belongs to, or of whoever gave
// it out. A name is made of groups, each group of one or more parts, and each
// part says one thing about its holder: a country, an organisation, a common
// name. They are written on one line, each part opened by a slash, named by its
// short name, and given as it was written down.

// onelineName writes a name out that way. Parts are added whole while there is
// room for them, and the first that would not fit ends the name, so that what
// comes back is never cut off part way through a part.
func onelineName(name []byte) string {
	var out strings.Builder
	length, written := 0, -1
	for group, element := range derParts(name) {
		if !element.named(derUniversal, derSet) {
			continue
		}
		for _, attribute := range derParts(element.body) {
			label, text, ok := nameAttribute(attribute)
			if !ok {
				continue
			}
			value := onelineValue(text)
			// What opens the part, its short name, the equals, and the value.
			length += 1 + len(label) + 1 + len(value)
			if length > onelineMax {
				return out.String()
			}
			// Parts of one group are joined to each other; a new group opens.
			if written == group {
				out.WriteByte('+')
			} else {
				out.WriteByte('/')
			}
			out.WriteString(label)
			out.WriteByte('=')
			out.WriteString(value)
			written = group
		}
	}
	return out.String()
}

// nameAttribute reads one part of a name: what it says about its holder, and
// what it says. A part with no short name of its own is named by its number
// instead. Anything not shaped like a part at all is passed over.
func nameAttribute(attribute derElement) (label string, text derElement, ok bool) {
	parts := derParts(attribute.body)
	if len(parts) != 2 || !parts[0].named(derUniversal, derOID) {
		return "", derElement{}, false
	}
	dotted := oidText(parts[0].body)
	if short, known := nameAttributes[dotted]; known {
		return short, parts[1], true
	}
	return dotted, parts[1], true
}

// onelineValue writes what a part says. Plain printable characters are given as
// they are, a slash or a plus is marked so it is not read as opening a part,
// and anything else is written as its value.
func onelineValue(text derElement) string {
	keep := usedBytes(text)
	var out strings.Builder
	for i, c := range text.body {
		if !keep[i&3] {
			continue
		}
		switch {
		case c < ' ' || c > '~':
			fmt.Fprintf(&out, `\x%02X`, c)
		case c == '/' || c == '+':
			out.WriteByte('\\')
			out.WriteByte(c)
		default:
			out.WriteByte(c)
		}
	}
	return out.String()
}

// usedBytes says which of every four bytes to write out. Text written the old
// way gives four bytes to a character, nearly always with the first three empty;
// where that holds, only the last of each four is written, and where any
// character actually uses the others, all four are.
func usedBytes(text derElement) [4]bool {
	const wide = 4
	all := [4]bool{true, true, true, true}
	if !text.named(derUniversal, derGeneralString) || len(text.body)%wide != 0 {
		return all
	}
	var used [4]bool
	for i, c := range text.body {
		if c != 0 {
			used[i%wide] = true
		}
	}
	if used[0] || used[1] || used[2] {
		return all
	}
	return [4]bool{false, false, false, true}
}

// The names given to the numbered objects that turn up in a certificate. Both
// tables follow the naming used by the library these fields were first written
// against, since a rule comparing against them is comparing against those exact
// words.

// nameAttributes gives the short name of each part a name can be made of. A
// part with no name here is written out as its number instead.
var nameAttributes = map[string]string{
	"2.5.4.3":  "CN",
	"2.5.4.4":  "SN",
	"2.5.4.5":  "serialNumber",
	"2.5.4.6":  "C",
	"2.5.4.7":  "L",
	"2.5.4.8":  "ST",
	"2.5.4.9":  "street",
	"2.5.4.10": "O",
	"2.5.4.11": "OU",
	"2.5.4.12": "title",
	"2.5.4.13": "description",
	"2.5.4.14": "searchGuide",
	"2.5.4.15": "businessCategory",
	"2.5.4.16": "postalAddress",
	"2.5.4.17": "postalCode",
	"2.5.4.18": "postOfficeBox",
	"2.5.4.19": "physicalDeliveryOfficeName",
	"2.5.4.20": "telephoneNumber",
	"2.5.4.21": "telexNumber",
	"2.5.4.22": "teletexTerminalIdentifier",
	"2.5.4.23": "facsimileTelephoneNumber",
	"2.5.4.24": "x121Address",
	"2.5.4.25": "internationaliSDNNumber",
	"2.5.4.26": "registeredAddress",
	"2.5.4.27": "destinationIndicator",
	"2.5.4.28": "preferredDeliveryMethod",
	"2.5.4.29": "presentationAddress",
	"2.5.4.30": "supportedApplicationContext",
	"2.5.4.31": "member",
	"2.5.4.32": "owner",
	"2.5.4.33": "roleOccupant",
	"2.5.4.34": "seeAlso",
	"2.5.4.35": "userPassword",
	"2.5.4.36": "userCertificate",
	"2.5.4.37": "cACertificate",
	"2.5.4.38": "authorityRevocationList",
	"2.5.4.39": "certificateRevocationList",
	"2.5.4.40": "crossCertificatePair",
	"2.5.4.41": "name",
	"2.5.4.42": "GN",
	"2.5.4.43": "initials",
	"2.5.4.44": "generationQualifier",
	"2.5.4.45": "x500UniqueIdentifier",
	"2.5.4.46": "dnQualifier",
	"2.5.4.47": "enhancedSearchGuide",
	"2.5.4.48": "protocolInformation",
	"2.5.4.49": "distinguishedName",
	"2.5.4.50": "uniqueMember",
	"2.5.4.51": "houseIdentifier",
	"2.5.4.52": "supportedAlgorithms",
	"2.5.4.53": "deltaRevocationList",
	"2.5.4.54": "dmdName",
	"2.5.4.65": "pseudonym",
	"2.5.4.72": "role",

	"1.2.840.113549.1.9.1": "emailAddress",
	"1.2.840.113549.1.9.2": "unstructuredName",
	"1.2.840.113549.1.9.7": "challengePassword",
	"1.2.840.113549.1.9.8": "unstructuredAddress",

	"0.9.2342.19200300.100.1.1":  "UID",
	"0.9.2342.19200300.100.1.25": "DC",

	// The parts naming where a holder is registered, which certificates given
	// out after a fuller check carry.
	"1.3.6.1.4.1.311.60.2.1.1": "jurisdictionL",
	"1.3.6.1.4.1.311.60.2.1.2": "jurisdictionST",
	"1.3.6.1.4.1.311.60.2.1.3": "jurisdictionC",
}

// signatureAlgorithms gives the full name of each way a certificate can be
// signed. One signed some way not named here is said to be signed by nothing
// known.
var signatureAlgorithms = map[string]string{
	"1.2.840.113549.1.1.2":  "md2WithRSAEncryption",
	"1.2.840.113549.1.1.3":  "md4WithRSAEncryption",
	"1.2.840.113549.1.1.4":  "md5WithRSAEncryption",
	"1.2.840.113549.1.1.5":  "sha1WithRSAEncryption",
	"1.2.840.113549.1.1.10": "rsassaPss",
	"1.2.840.113549.1.1.11": "sha256WithRSAEncryption",
	"1.2.840.113549.1.1.12": "sha384WithRSAEncryption",
	"1.2.840.113549.1.1.13": "sha512WithRSAEncryption",
	"1.2.840.113549.1.1.14": "sha224WithRSAEncryption",
	"1.2.840.113549.1.1.15": "sha512-224WithRSAEncryption",
	"1.2.840.113549.1.1.16": "sha512-256WithRSAEncryption",

	"1.2.840.10040.4.3": "dsaWithSHA1",
	"1.3.14.3.2.3":      "md5WithRSA",
	"1.3.14.3.2.27":     "dsaWithSHA1-old",
	"1.3.14.3.2.29":     "sha1WithRSA",

	"1.2.840.10045.4.1":   "ecdsa-with-SHA1",
	"1.2.840.10045.4.3.1": "ecdsa-with-SHA224",
	"1.2.840.10045.4.3.2": "ecdsa-with-SHA256",
	"1.2.840.10045.4.3.3": "ecdsa-with-SHA384",
	"1.2.840.10045.4.3.4": "ecdsa-with-SHA512",

	"2.16.840.1.101.3.4.3.1":  "dsa_with_SHA224",
	"2.16.840.1.101.3.4.3.2":  "dsa_with_SHA256",
	"2.16.840.1.101.3.4.3.5":  "id-dsa-with-sha384",
	"2.16.840.1.101.3.4.3.6":  "id-dsa-with-sha512",
	"2.16.840.1.101.3.4.3.9":  "ecdsa_with_SHA3-224",
	"2.16.840.1.101.3.4.3.10": "ecdsa_with_SHA3-256",
	"2.16.840.1.101.3.4.3.11": "ecdsa_with_SHA3-384",
	"2.16.840.1.101.3.4.3.12": "ecdsa_with_SHA3-512",
	"2.16.840.1.101.3.4.3.13": "RSA-SHA3-224",
	"2.16.840.1.101.3.4.3.14": "RSA-SHA3-256",
	"2.16.840.1.101.3.4.3.15": "RSA-SHA3-384",
	"2.16.840.1.101.3.4.3.16": "RSA-SHA3-512",

	"1.3.101.112": "ED25519",
	"1.3.101.113": "ED448",
}
