package yara

import (
	"crypto/sha1" // #nosec G505 -- the thumbprint of a certificate is defined as a SHA-1
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

// The certificates a signed file carries. A file names a table of them, each
// entry holding a blob that says who signed the file, and the certificates
// belonging to those signers are what a rule can ask about.

// derOf writes one element: what it is, how long it is, then its contents.
func derOf(class byte, tag int, holds bool, body []byte) []byte {
	first := class<<6 | byte(tag)
	if holds {
		first |= 0x20
	}
	out := []byte{first}
	switch n := len(body); {
	case n < 0x80:
		out = append(out, byte(n))
	case n < 0x100:
		out = append(out, 0x81, byte(n))
	default:
		out = append(out, 0x82, byte(n>>8), byte(n))
	}
	return append(out, body...)
}

func derRun(parts ...[]byte) []byte {
	return derOf(derUniversal, derSequence, true, concat(parts...))
}

func derBag(parts ...[]byte) []byte {
	return derOf(derUniversal, derSet, true, concat(parts...))
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// derName writes an object's number in the run-length form these blobs use: the
// first two parts share a byte, and the rest follow seven bits at a time.
func derName(dotted string) []byte {
	var parts []int
	for piece := range strings.SplitSeq(dotted, ".") {
		n, err := strconv.Atoi(piece)
		if err != nil {
			panic("not a number in " + dotted)
		}
		parts = append(parts, n)
	}
	body := []byte{byte(parts[0]*40 + parts[1])}
	for _, n := range parts[2:] {
		var run []byte
		for {
			run = append([]byte{byte(n & 0x7F)}, run...)
			n >>= 7
			if n == 0 {
				break
			}
		}
		for i := range run[:len(run)-1] {
			run[i] |= 0x80
		}
		body = append(body, run...)
	}
	return derOf(derUniversal, derOID, false, body)
}

func derNumber(body []byte) []byte {
	return derOf(derUniversal, derInteger, false, body)
}

// derAttribute writes one part of a name: what it says about its holder, and
// what it says.
func derAttribute(oid string, tag int, text string) []byte {
	return derRun(derName(oid), derOf(derUniversal, tag, false, []byte(text)))
}

// derNameOf writes a whole name out of parts, each part its own group.
func derNameOf(parts ...[]byte) []byte {
	var groups [][]byte
	for _, p := range parts {
		groups = append(groups, derBag(p))
	}
	return derRun(groups...)
}

// The numbers of the shapes a signed blob is made of.
const (
	oidSignedData    = "1.2.840.113549.1.7.2"
	oidData          = "1.2.840.113549.1.7.1"
	oidSHA256RSA     = "1.2.840.113549.1.1.11"
	oidSHA1RSA       = "1.2.840.113549.1.1.5"
	oidCommonName    = "2.5.4.3"
	oidOrganization  = "2.5.4.10"
	oidCountry       = "2.5.4.6"
	oidNestedSigning = "1.3.6.1.4.1.311.2.4.1"
)

// testCertificate is a certificate being laid out for a test.
type testCertificate struct {
	// version is what the certificate says of itself, counting from nought, or
	// -1 for one so old it does not say.
	version   int
	serial    []byte
	issuer    []byte
	subject   []byte
	algorithm string
	// from and until are written the way a certificate writes a moment: two
	// digits of year and a Z, unless wide is set, which uses four.
	from, until string
	wide        bool
}

// der lays the certificate out.
func (c testCertificate) der() []byte {
	var head []byte
	if c.version >= 0 {
		head = derOf(derContext, 0, true, derNumber([]byte{byte(c.version)}))
	}
	when := func(s string) []byte {
		if c.wide {
			return derOf(derUniversal, derGeneralTime, false, []byte(s))
		}
		return derOf(derUniversal, derUTCTime, false, []byte(s))
	}
	algorithm := derRun(derName(c.algorithm), derOf(derUniversal, 5, false, nil))
	tbs := derRun(
		append(head, derNumber(c.serial)...),
		algorithm,
		c.issuer,
		derRun(when(c.from), when(c.until)),
		c.subject,
		// What follows is never read, so a shape standing in for the key is
		// enough to keep the certificate whole.
		derRun(algorithm, derOf(derUniversal, derBitString, false, []byte{0})),
	)
	return derRun(tbs, algorithm, derOf(derUniversal, derBitString, false, []byte{0, 1}))
}

// thumbprint is what a rule sees as the certificate's own mark.
func (c testCertificate) thumbprint() string {
	sum := sha1.Sum(c.der()) // #nosec G401 -- the mark is defined as a SHA-1
	return hex.EncodeToString(sum[:])
}

// signedBlob lays out a blob naming who signed something: the certificates it
// carries, which of them signed, and any signature nested inside it.
func signedBlob(certs []testCertificate, signers []testCertificate, nested []byte) []byte {
	var bag []byte
	for _, c := range certs {
		bag = append(bag, c.der()...)
	}
	var infos []byte
	for i, s := range signers {
		var extra []byte
		// Only the first signer is looked at for a nested signature.
		if i == 0 && nested != nil {
			extra = derOf(derContext, 1, true,
				derRun(derName(oidNestedSigning), derBag(nested)))
		}
		infos = append(infos, derRun(
			derNumber([]byte{1}),
			derRun(s.issuer, derNumber(s.serial)),
			derRun(derName(oidSHA256RSA)),
			derRun(derName(oidSHA256RSA)),
			derOf(derUniversal, derOctet, false, []byte{9}),
			extra,
		)...)
	}
	signed := derRun(
		derNumber([]byte{1}),
		derBag(derRun(derName(oidSHA256RSA))),
		derRun(derName(oidData)),
		derOf(derContext, 0, true, bag),
		derBag(infos),
	)
	return derRun(derName(oidSignedData), derOf(derContext, 0, true, signed))
}

// withCertificateTable writes a table of signed blobs onto the end of the file
// and names it in the table of tables.
func (b *peBuilder) withCertificateTable(entries [][]byte) {
	// The file has to claim the whole table of tables, or the entry naming the
	// certificates is not reached at all.
	b.put32(b.directoriesAt()-4, maxDataDirectories)
	b.pad((len(b.data) + 7) & ^7)
	at := len(b.data)
	for _, blob := range entries {
		head := make([]byte, certHeaderSize)
		binary.LittleEndian.PutUint32(head, uint32(certHeaderSize+len(blob)))
		binary.LittleEndian.PutUint16(head[4:], certRevision2)
		binary.LittleEndian.PutUint16(head[6:], certTypePKCS7)
		b.data = append(b.data, head...)
		b.data = append(b.data, blob...)
		b.pad((len(b.data) + 7) & ^7)
	}
	b.put32(b.directoriesAt()+directoryCertificates*dataDirectoryEntry, uint32(at))
	b.put32(b.directoriesAt()+directoryCertificates*dataDirectoryEntry+4,
		uint32(len(b.data)-at))
}

// aCertificate is a plain certificate to build the tests around.
func aCertificate() testCertificate {
	return testCertificate{
		version: 2,
		serial:  []byte{0x0C, 0x44, 0xF8, 0xC0},
		issuer: derNameOf(
			derAttribute(oidCountry, 19, "US"),
			derAttribute(oidOrganization, 19, "Example Roots"),
			derAttribute(oidCommonName, 19, "Example Signing CA"),
		),
		subject: derNameOf(
			derAttribute(oidCountry, 19, "GB"),
			derAttribute(oidCommonName, derUTF8, "Someone Or Other"),
		),
		algorithm: oidSHA256RSA,
		from:      "131210000000Z",
		until:     "170413120000Z",
	}
}

// signedPE lays out a file signed by the given certificates.
func signedPE(certs []testCertificate, signers []testCertificate, nested []byte) []byte {
	b := newPE(true)
	b.withCertificateTable([][]byte{signedBlob(certs, signers, nested)})
	return b.data
}

// wantString checks a field came to a given run of characters.
func wantString(t *testing.T, got value, want string, what string) {
	t.Helper()
	if got.kind != valueString || got.s != want {
		t.Errorf("%s came to %v %q, want the text %q", what, got.kind, got.s, want)
	}
}

// TestPESignatureFields covers everything a rule can ask about a certificate
// that signed a file.
func TestPESignatureFields(t *testing.T) {
	cert := aCertificate()
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)

	wantInt(t, scanPE(t, data, "pe.number_of_signatures"), 1, "how many signed it")
	wantString(t, scanPE(t, data, "pe.signatures[0].thumbprint"),
		cert.thumbprint(), "the certificate's own mark")
	wantString(t, scanPE(t, data, "pe.signatures[0].issuer"),
		"/C=US/O=Example Roots/CN=Example Signing CA", "who gave it out")
	wantString(t, scanPE(t, data, "pe.signatures[0].subject"),
		"/C=GB/CN=Someone Or Other", "who it belongs to")
	wantInt(t, scanPE(t, data, "pe.signatures[0].version"), 3, "which version it is")
	wantString(t, scanPE(t, data, "pe.signatures[0].algorithm"),
		"sha256WithRSAEncryption", "how it was signed")
	wantString(t, scanPE(t, data, "pe.signatures[0].algorithm_oid"),
		oidSHA256RSA, "the number of how it was signed")
	wantString(t, scanPE(t, data, "pe.signatures[0].serial"),
		"0c:44:f8:c0", "the number it was given")
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_before"), 1386633600, "when it begins")
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_after"), 1492084800, "when it ends")
}

// TestPESignatureValidOn covers asking whether a certificate held good at a
// given moment, which counts both of its ends as within.
func TestPESignatureValidOn(t *testing.T) {
	cert := aCertificate()
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	cases := []struct {
		when int64
		want int64
	}{
		{1386633599, 0},
		{1386633600, 1},
		{1400000000, 1},
		{1492084800, 1},
		{1492084801, 0},
	}
	for _, c := range cases {
		t.Run(strconv.FormatInt(c.when, 10), func(t *testing.T) {
			wantInt(t, scanPE(t, data, "pe.signatures[0].valid_on("+
				strconv.FormatInt(c.when, 10)+")"), c.want, "whether it held good")
		})
	}
}

// TestPESignatureValidOnUnanswerableMoment covers asking about a moment that
// itself has no value, which leaves the question unanswered rather than reading
// the missing moment as a number.
func TestPESignatureValidOnUnanswerableMoment(t *testing.T) {
	cert := aCertificate()
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	wantNothing(t, scanPE(t, data,
		"pe.signatures[0].valid_on(pe.sections[99].virtual_address)"),
		"whether it held good")
}

// TestPESignatureValidOnWithoutDates covers a certificate whose moments cannot
// be read, which leaves the question unanswerable.
func TestPESignatureValidOnWithoutDates(t *testing.T) {
	cert := aCertificate()
	cert.from, cert.until = "13", "17"
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	wantNothing(t, scanPE(t, data, "pe.signatures[0].not_before"), "when it begins")
	wantNothing(t, scanPE(t, data, "pe.signatures[0].valid_on(1400000000)"),
		"whether it held good")
}

// TestPESignatureWideYear covers a certificate writing its moments with the
// full year rather than the last two digits of it.
func TestPESignatureWideYear(t *testing.T) {
	cert := aCertificate()
	cert.wide = true
	cert.from, cert.until = "20131210000000Z", "20170413120000Z"
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_before"), 1386633600, "when it begins")
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_after"), 1492084800, "when it ends")
}

// TestPESignatureOldYear covers the two-digit year, where anything below
// seventy is read as being of this century rather than the last.
func TestPESignatureOldYear(t *testing.T) {
	cert := aCertificate()
	cert.from, cert.until = "691231235959Z", "700101000000Z"
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_before"), 3155759999, "when it begins")
	wantInt(t, scanPE(t, data, "pe.signatures[0].not_after"), 0, "when it ends")
}

// TestPESignatureVersionOne covers a certificate too old to say which version
// it is, which counts as the first.
func TestPESignatureVersionOne(t *testing.T) {
	cert := aCertificate()
	cert.version = -1
	data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
	wantInt(t, scanPE(t, data, "pe.signatures[0].version"), 1, "which version it is")
}

// TestPESignatureSerialLength covers which numbers are written out and which
// are passed over: a number is given only when it takes between one and twenty
// bytes to write down.
func TestPESignatureSerialLength(t *testing.T) {
	cases := []struct {
		name   string
		serial []byte
		want   string
	}{
		{"one byte", []byte{0x05}, "05"},
		{"a few", []byte{0x0C, 0x44, 0xF8}, "0c:44:f8"},
		{
			"twenty, the most there may be",
			append([]byte{0x01}, make([]byte, 19)...),
			"01" + strings.Repeat(":00", 19),
		},
		{
			"twenty-one, one too many",
			append([]byte{0x01}, make([]byte, 20)...), "",
		},
		// A number written with more room than it needs is trimmed back before
		// being counted, so what matters is its value rather than its writing.
		{"leading nothing is dropped", []byte{0x00, 0x7F}, "7f"},
		{"a number below nought keeps its sign", []byte{0xFF, 0x80}, "80"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cert := aCertificate()
			cert.serial = c.serial
			data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
			got := scanPE(t, data, "pe.signatures[0].serial")
			if c.want == "" {
				wantNothing(t, got, "the number it was given")
				return
			}
			wantString(t, got, c.want, "the number it was given")
		})
	}
}

// TestPESignatureNames covers writing a name out the way these fields do: each
// part opened by a slash, named by its short name, and anything outside plain
// printable text written as its value.
func TestPESignatureNames(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"the ordinary case", derNameOf(
			derAttribute(oidCountry, 19, "US"),
			derAttribute(oidCommonName, 19, "Someone"),
		), "/C=US/CN=Someone"},
		{"a slash within a part is marked", derNameOf(
			derAttribute(oidCommonName, 19, "a/b+c"),
		), `/CN=a\/b\+c`},
		{"anything not plain text is written as its value", derNameOf(
			derAttribute(oidCommonName, derUTF8, "Gerritstra\xC3\x9Fe"),
		), `/CN=Gerritstra\xC3\x9Fe`},
		{"a part with no name of its own is numbered", derNameOf(
			derAttribute("1.2.3.4", 19, "x"),
		), "/1.2.3.4=x"},
		{"parts of one group are joined rather than opened", derRun(
			derBag(derAttribute(oidCountry, 19, "US")),
			derBag(derAttribute(oidCommonName, 19, "A"),
				derAttribute(oidOrganization, 19, "B")),
		), "/C=US/CN=A+O=B"},
		{"an empty name comes to nothing", derRun(), ""},
		{"a part saying nothing", derNameOf(
			derAttribute(oidCommonName, 19, ""),
		), "/CN="},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, _, ok := readDER(c.in)
			if !ok {
				t.Fatal("the name did not read at all")
			}
			if got := onelineName(e.body); got != c.want {
				t.Errorf("the name reads as %q, want %q", got, c.want)
			}
		})
	}
}

// TestPESignatureNameTooLong covers a name longer than there is room for, where
// the parts that fit are kept whole and the rest are left off.
func TestPESignatureNameTooLong(t *testing.T) {
	long := strings.Repeat("x", 100)
	e, _, _ := readDER(derNameOf(
		derAttribute(oidCommonName, 19, long),
		derAttribute(oidCommonName, 19, long),
		// Each part above takes 105 characters, so a third would run past the
		// room set aside and is dropped whole.
		derAttribute(oidCommonName, 19, long),
	))
	got := onelineName(e.body)
	if want := "/CN=" + long + "/CN=" + long; got != want {
		t.Errorf("the name reads as %d characters, want the first two parts only", len(got))
	}

	// A first part that does not fit on its own leaves nothing at all.
	e, _, _ = readDER(derNameOf(derAttribute(oidCommonName, 19, strings.Repeat("y", 300))))
	if got := onelineName(e.body); got != "" {
		t.Errorf("the name reads as %q, want nothing", got)
	}
}

// TestPESignatureNameFourByteText covers the old way of writing text four bytes
// to a character, where the three empty bytes of each are left out unless some
// character actually uses them.
func TestPESignatureNameFourByteText(t *testing.T) {
	wide := "\x00\x00\x00A\x00\x00\x00B"
	e, _, _ := readDER(derNameOf(derAttribute(oidCommonName, derGeneralString, wide)))
	if got, want := onelineName(e.body), "/CN=AB"; got != want {
		t.Errorf("the name reads as %q, want %q", got, want)
	}

	// When any of the leading three is used, every byte is written out.
	used := "\x00\x00\x01A"
	e, _, _ = readDER(derNameOf(derAttribute(oidCommonName, derGeneralString, used)))
	if got, want := onelineName(e.body), `/CN=\x00\x00\x01A`; got != want {
		t.Errorf("the name reads as %q, want %q", got, want)
	}

	// A length not divisible by four is written out plainly.
	odd := "\x00A\x00"
	e, _, _ = readDER(derNameOf(derAttribute(oidCommonName, derGeneralString, odd)))
	if got, want := onelineName(e.body), `/CN=\x00A\x00`; got != want {
		t.Errorf("the name reads as %q, want %q", got, want)
	}
}

// TestPESignatureNameMalformed covers name parts that are not shaped as they
// should be, which are passed over rather than refused.
func TestPESignatureNameMalformed(t *testing.T) {
	name := derRun(
		// Something that is not a group at all.
		derNumber([]byte{1}),
		derBag(
			// A part naming nothing.
			derRun(derName(oidCommonName)),
			// A part named by something that is not a number.
			derRun(derNumber([]byte{1}), derOf(derUniversal, 19, false, []byte("x"))),
			// And one that is right.
			derAttribute(oidCountry, 19, "US"),
		),
	)
	e, _, _ := readDER(name)
	if got, want := onelineName(e.body), "/C=US"; got != want {
		t.Errorf("the name reads as %q, want %q", got, want)
	}
}

// TestPESignatureAlgorithms covers naming how a certificate was signed, and
// what happens when it was signed some way with no name.
func TestPESignatureAlgorithms(t *testing.T) {
	cases := []struct {
		oid       string
		algorithm string
		named     bool
	}{
		{oidSHA256RSA, "sha256WithRSAEncryption", true},
		{oidSHA1RSA, "sha1WithRSAEncryption", true},
		{"1.2.840.10045.4.3.2", "ecdsa-with-SHA256", true},
		{"1.2.3.4.5", "UNDEF", false},
	}
	for _, c := range cases {
		t.Run(c.oid, func(t *testing.T) {
			cert := aCertificate()
			cert.algorithm = c.oid
			data := signedPE([]testCertificate{cert}, []testCertificate{cert}, nil)
			wantString(t, scanPE(t, data, "pe.signatures[0].algorithm"),
				c.algorithm, "how it was signed")
			want := c.oid
			if !c.named {
				// One with no name has no number written out either.
				want = ""
			}
			wantString(t, scanPE(t, data, "pe.signatures[0].algorithm_oid"),
				want, "the number of how it was signed")
		})
	}
}

// TestPESignatureOnlySigners covers a blob carrying certificates that did not
// sign anything, which are not offered.
func TestPESignatureOnlySigners(t *testing.T) {
	signer := aCertificate()
	other := aCertificate()
	other.serial = []byte{0x77}
	other.subject = derNameOf(derAttribute(oidCommonName, 19, "Not The Signer"))

	data := signedPE([]testCertificate{other, signer}, []testCertificate{signer}, nil)
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"), 1, "how many signed it")
	wantString(t, scanPE(t, data, "pe.signatures[0].subject"),
		"/C=GB/CN=Someone Or Other", "who it belongs to")
}

// TestPESignatureSignerNotCarried covers a blob naming a signer whose
// certificate it does not carry, which leaves the whole blob unread.
func TestPESignatureSignerNotCarried(t *testing.T) {
	signer := aCertificate()
	missing := aCertificate()
	missing.serial = []byte{0x77}

	data := signedPE([]testCertificate{signer}, []testCertificate{signer, missing}, nil)
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"), 0, "how many signed it")
}

// TestPESignatureSeveralSigners covers a blob naming more than one signer,
// which are offered in the order the blob names them.
func TestPESignatureSeveralSigners(t *testing.T) {
	first := aCertificate()
	second := aCertificate()
	second.serial = []byte{0x77}
	second.subject = derNameOf(derAttribute(oidCommonName, 19, "The Other One"))

	data := signedPE([]testCertificate{first, second},
		[]testCertificate{second, first}, nil)
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"), 2, "how many signed it")
	wantString(t, scanPE(t, data, "pe.signatures[0].subject"),
		"/CN=The Other One", "who the first belongs to")
	wantString(t, scanPE(t, data, "pe.signatures[1].subject"),
		"/C=GB/CN=Someone Or Other", "who the second belongs to")
}

// TestPESignatureNested covers a signature carried inside another one, whose
// signers are counted alongside the outer ones.
func TestPESignatureNested(t *testing.T) {
	outer := aCertificate()
	inner := aCertificate()
	inner.serial = []byte{0x99}
	inner.subject = derNameOf(derAttribute(oidCommonName, 19, "The Inner One"))

	data := signedPE([]testCertificate{outer}, []testCertificate{outer},
		signedBlob([]testCertificate{inner}, []testCertificate{inner}, nil))
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"), 2, "how many signed it")
	wantString(t, scanPE(t, data, "pe.signatures[1].subject"),
		"/CN=The Inner One", "who the nested one belongs to")
}

// TestPESignatureTooMany covers a file carrying more certificates than are
// offered, which stops at the sixteenth.
func TestPESignatureTooMany(t *testing.T) {
	var certs []testCertificate
	for i := range maxCertificates + 4 {
		c := aCertificate()
		c.serial = []byte{byte(i + 1)}
		certs = append(certs, c)
	}
	data := signedPE(certs, certs, nil)
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"),
		maxCertificates, "how many signed it")
	wantNothing(t, scanPE(t, data, "pe.signatures[16].subject"),
		"who the seventeenth belongs to")
}

// TestPESignatureSeveralBlobs covers a table holding more than one blob, each
// read in turn.
func TestPESignatureSeveralBlobs(t *testing.T) {
	first := aCertificate()
	second := aCertificate()
	second.serial = []byte{0x77}
	second.subject = derNameOf(derAttribute(oidCommonName, 19, "The Second Blob"))

	b := newPE(true)
	b.withCertificateTable([][]byte{
		signedBlob([]testCertificate{first}, []testCertificate{first}, nil),
		signedBlob([]testCertificate{second}, []testCertificate{second}, nil),
	})
	wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 2, "how many signed it")
	wantString(t, scanPE(t, b.data, "pe.signatures[1].subject"),
		"/CN=The Second Blob", "who the second belongs to")
}

// TestPESignatureUnsigned covers a file with no certificates at all, which says
// plainly that none signed it.
func TestPESignatureUnsigned(t *testing.T) {
	b := newPE(true)
	b.put32(b.directoriesAt()-4, maxDataDirectories)
	wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 0, "how many signed it")
	wantNothing(t, scanPE(t, b.data, "pe.signatures[0].subject"), "who the first belongs to")
}

// TestPESignatureFewerDirectories covers a file claiming too few tables for the
// certificates to be among them, which leaves the question unanswered rather
// than answering none.
func TestPESignatureFewerDirectories(t *testing.T) {
	b := newPE(true)
	b.withCertificateTable([][]byte{signedBlob([]testCertificate{aCertificate()},
		[]testCertificate{aCertificate()}, nil)})
	b.put32(b.directoriesAt()-4, directoryCertificates)
	wantNothing(t, scanPE(t, b.data, "pe.number_of_signatures"), "how many signed it")
}

// TestPESignatureTableRefused covers tables that do not hold what they claim,
// which leave the file counted as signed by nobody.
func TestPESignatureTableRefused(t *testing.T) {
	blob := signedBlob([]testCertificate{aCertificate()},
		[]testCertificate{aCertificate()}, nil)

	cases := []struct {
		name  string
		spoil func(b *peBuilder)
	}{
		{"a table beginning past the end", func(b *peBuilder) {
			b.put32(b.directoriesAt()+directoryCertificates*dataDirectoryEntry, 1<<24)
		}},
		{"a table longer than the file", func(b *peBuilder) {
			b.put32(b.directoriesAt()+directoryCertificates*dataDirectoryEntry+4, 1<<24)
		}},
		{"an entry saying it is shorter than its own opening", func(b *peBuilder) {
			b.put32(len(b.data)-certHeaderSize-len(blob), certHeaderSize)
		}},
		{"an entry running past the table", func(b *peBuilder) {
			b.put32(len(b.data)-certHeaderSize-len(blob), 1<<20)
		}},
		{"an entry written in a form that is not read", func(b *peBuilder) {
			b.put16(len(b.data)-certHeaderSize-len(blob)+4, certRevision1)
		}},
		{"an entry written in no known form at all", func(b *peBuilder) {
			b.put16(len(b.data)-certHeaderSize-len(blob)+4, 0x0300)
		}},
		{"an entry holding something other than a signature", func(b *peBuilder) {
			b.put16(len(b.data)-certHeaderSize-len(blob)+6, 1)
		}},
		{"an entry holding bytes that are not a signature", func(b *peBuilder) {
			at := len(b.data) - len(blob)
			for i := range blob {
				b.data[at+i] = 0xFF
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newPE(true)
			b.withCertificateTable([][]byte{blob})
			c.spoil(b)
			wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 0,
				"how many signed it")
		})
	}
}

// TestPESignatureBlobRefused covers blobs whose shape is not the one a
// signature has, which are passed over.
func TestPESignatureBlobRefused(t *testing.T) {
	cert := aCertificate()
	cases := []struct {
		name string
		blob []byte
	}{
		{"not an element at all", []byte{0xFF, 0xFF}},
		{
			"holding something that is not a signature",
			derRun(derName(oidData), derOf(derContext, 0, true, derNumber([]byte{1}))),
		},
		{"naming nothing to hold", derRun(derName(oidSignedData))},
		{
			"holding too little to be a signature",
			derRun(derName(oidSignedData), derOf(derContext, 0, true, derRun())),
		},
		{"naming no signers at all", derRun(derName(oidSignedData),
			derOf(derContext, 0, true, derRun(
				derNumber([]byte{1}),
				derBag(),
				derRun(derName(oidData)),
				derOf(derContext, 0, true, cert.der()),
				derBag(),
			)))},
		{"a signer saying nothing of who it is", derRun(derName(oidSignedData),
			derOf(derContext, 0, true, derRun(
				derNumber([]byte{1}),
				derBag(),
				derRun(derName(oidData)),
				derOf(derContext, 0, true, cert.der()),
				derBag(derRun(derNumber([]byte{1}))),
			)))},
		{"carrying a certificate that is not one", derRun(derName(oidSignedData),
			derOf(derContext, 0, true, derRun(
				derNumber([]byte{1}),
				derBag(),
				derRun(derName(oidData)),
				derOf(derContext, 0, true, derNumber([]byte{1})),
				derBag(derRun(derNumber([]byte{1}),
					derRun(cert.issuer, derNumber(cert.serial)))),
			)))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newPE(true)
			b.withCertificateTable([][]byte{c.blob})
			wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 0,
				"how many signed it")
		})
	}
}

// TestPESignatureNestedWhenFull covers a nested signature reached only after as
// many certificates as are offered have already been found, which adds none.
func TestPESignatureNestedWhenFull(t *testing.T) {
	var certs []testCertificate
	for i := range maxCertificates {
		c := aCertificate()
		c.serial = []byte{byte(i + 1)}
		certs = append(certs, c)
	}
	inner := aCertificate()
	inner.serial = []byte{0x99}
	inner.subject = derNameOf(derAttribute(oidCommonName, 19, "The Inner One"))

	data := signedPE(certs, certs,
		signedBlob([]testCertificate{inner}, []testCertificate{inner}, nil))
	wantInt(t, scanPE(t, data, "pe.number_of_signatures"),
		maxCertificates, "how many signed it")
}

// TestPESignatureNestedOtherAccounts covers what a signer says of itself
// alongside a nested signature, which is passed over rather than read as one.
func TestPESignatureNestedOtherAccounts(t *testing.T) {
	outer := aCertificate()
	inner := aCertificate()
	inner.serial = []byte{0x99}
	inner.subject = derNameOf(derAttribute(oidCommonName, 19, "The Inner One"))

	// The account a signer gives of itself holds several things; only the one
	// numbered as a nested signature is read.
	nested := signedBlob([]testCertificate{inner}, []testCertificate{inner}, nil)
	extra := derOf(derContext, 1, true, concat(
		derRun(derName("1.2.3.4"), derBag(derNumber([]byte{7}))),
		// One that is not shaped like an account at all.
		derRun(derName(oidNestedSigning)),
		derRun(derName(oidNestedSigning), derBag(nested)),
	))

	signer := derRun(
		derNumber([]byte{1}),
		derRun(outer.issuer, derNumber(outer.serial)),
		derRun(derName(oidSHA256RSA)),
		derRun(derName(oidSHA256RSA)),
		derOf(derUniversal, derOctet, false, []byte{9}),
		extra,
	)
	blob := derRun(derName(oidSignedData), derOf(derContext, 0, true, derRun(
		derNumber([]byte{1}),
		derBag(derRun(derName(oidSHA256RSA))),
		derRun(derName(oidData)),
		derOf(derContext, 0, true, outer.der()),
		derBag(signer),
	)))

	b := newPE(true)
	b.withCertificateTable([][]byte{blob})
	wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 2, "how many signed it")
	wantString(t, scanPE(t, b.data, "pe.signatures[1].subject"),
		"/CN=The Inner One", "who the nested one belongs to")
}

// TestPESignatureMoreBlobsRefused covers further shapes a blob can take that
// are not a signature.
func TestPESignatureMoreBlobsRefused(t *testing.T) {
	cert := aCertificate()
	cases := []struct {
		name string
		blob []byte
	}{
		{
			"holding more than one account of who signed",
			derRun(derName(oidSignedData), derOf(derContext, 0, true,
				concat(derRun(), derRun()))),
		},
		{
			"a signer not saying who gave the certificate out",
			derRun(derName(oidSignedData), derOf(derContext, 0, true, derRun(
				derNumber([]byte{1}),
				derBag(),
				derRun(derName(oidData)),
				derOf(derContext, 0, true, cert.der()),
				derBag(derRun(derNumber([]byte{1}), derRun(cert.issuer))),
			))),
		},
		{
			"carrying something shaped like a certificate that is not one",
			derRun(derName(oidSignedData), derOf(derContext, 0, true, derRun(
				derNumber([]byte{1}),
				derBag(),
				derRun(derName(oidData)),
				derOf(derContext, 0, true, derRun(derNumber([]byte{1}))),
				derBag(derRun(derNumber([]byte{1}),
					derRun(cert.issuer, derNumber(cert.serial)))),
			))),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newPE(true)
			b.withCertificateTable([][]byte{c.blob})
			wantInt(t, scanPE(t, b.data, "pe.number_of_signatures"), 0,
				"how many signed it")
		})
	}
}

// TestReadCertificateRefused covers bytes that are not shaped like a
// certificate, which are turned away rather than half read.
func TestReadCertificateRefused(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{"not made of the three parts one is", derRun(derRun(), derRun())},
		{"saying too little of itself", derRun(
			derRun(derNumber([]byte{1}), derRun(derName(oidSHA256RSA))),
			derRun(derName(oidSHA256RSA)),
			derOf(derUniversal, derBitString, false, []byte{0}),
		)},
		{"saying nothing of itself at all", derRun(
			derRun(),
			derRun(derName(oidSHA256RSA)),
			derOf(derUniversal, derBitString, false, []byte{0}),
		)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, _, ok := readDER(c.in)
			if !ok {
				t.Fatal("the bytes did not read as an element")
			}
			if _, got := readCertificate(e); got {
				t.Error("the bytes read as a certificate, want them turned away")
			}
		})
	}
}

// TestPESignatureAlgorithmUnreadable covers a certificate saying how it was
// signed in a way that names nothing, which counts as signed by nothing known.
func TestPESignatureAlgorithmUnreadable(t *testing.T) {
	for _, in := range [][]byte{derRun(), derRun(derNumber([]byte{1}))} {
		e, _, _ := readDER(in)
		name, oid := algorithmNames(e)
		if name != unknownAlgorithm || oid != "" {
			t.Errorf("it reads as signed by %q, %q, want %q and nothing",
				name, oid, unknownAlgorithm)
		}
	}
}

// TestCertificateTimeRefused covers moments a certificate cannot be read as
// naming, which leave it without an answer.
func TestCertificateTimeRefused(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{
			"a full year cut short",
			derOf(derUniversal, derGeneralTime, false, []byte("2013121000")),
		},
		{
			"a short year cut short",
			derOf(derUniversal, derUTCTime, false, []byte("13121000")),
		},
		{
			"something that is not a moment",
			derOf(derUniversal, derUTF8, false, []byte("131210000000Z")),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, _, _ := readDER(c.in)
			if _, ok := certificateTime(e); ok {
				t.Error("the bytes read as a moment, want them turned away")
			}
		})
	}
}

// TestOIDText covers writing a numbered object out in the dotted form, and the
// writings that cannot be read as one.
func TestOIDText(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"two parts sharing a byte", []byte{0x55}, "2.5"},
		{"a part needing more than one byte", []byte{0x2A, 0x86, 0x48}, "1.2.840"},
		// A first part above two leaves the rest of the byte to the second,
		// which is how the numbers above eighty are written.
		{"a first part above the two there are", []byte{0x81, 0x04}, "2.49.4"},
		{"nothing at all", nil, ""},
		{"a part running on past the end", []byte{0x55, 0x81}, ""},
		{"a part too large to hold", []byte{
			0x55, 0x81, 0x81, 0x81, 0x81, 0x81,
			0x81, 0x81, 0x81, 0x81, 0x81, 0x01,
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := oidText(c.in); got != c.want {
				t.Errorf("it reads as %q, want %q", got, c.want)
			}
		})
	}
}

// TestPESignatureNoDirectory covers a file whose table of tables does not reach
// as far as the certificates, which leaves the question unanswered rather than
// answering none.
func TestPESignatureNoDirectory(t *testing.T) {
	b := newPE(true)
	b.put32(b.directoriesAt()-4, maxDataDirectories)
	// Cutting the optional header short takes the entry away with it, even
	// though the file still claims to have it.
	b.data = b.data[:b.directoriesAt()+directoryCertificates*dataDirectoryEntry+4]
	wantNothing(t, scanPE(t, b.data, "pe.number_of_signatures"), "how many signed it")
}
