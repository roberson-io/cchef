package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Operations are fed data cchef did not produce: a file off disk, the output
// of another tool, something pulled out of a capture. A malformed input must
// come back as an error, never a panic or a hang.

// fuzzSkip lists operations a fuzzer must not drive: the ones that would reach
// the network or another process, and the deliberately slow key-derivation
// functions, which would spend the whole budget on a handful of executions.
var fuzzSkip = map[string]bool{
	"DNS over HTTPS":                true,
	"HTTP request":                  true,
	"Optical Character Recognition": true,
	"Show on map":                   true,
	"Argon2":                        true,
	"Argon2 compare":                true,
	"Bcrypt":                        true,
	"Bcrypt compare":                true,
	"Scrypt":                        true,
	"Derive PBKDF2 key":             true,
	"Sleep":                         true,
	// Its output doubles with every character, so a fuzzer drives it straight
	// into the length limit and learns nothing.
	"Get All Casings":                true,
	"Pseudo-Random Number Generator": true,
	"Generate RSA Key Pair":          true,
	"Generate PGP Key Pair":          true,
	"Generate ECDSA Key Pair":        true,
	"Pseudo-Random Prime Generator":  true,
}

// fuzzableOps is every registered operation a fuzzer may drive, in a fixed
// order so a failing input keeps naming the same operation.
func fuzzableOps() []core.Operation {
	all := core.Default.All()
	out := make([]core.Operation, 0, len(all))
	for _, op := range all {
		if !fuzzSkip[op.Meta().Name] {
			out = append(out, op)
		}
	}
	return out
}

// FuzzOperation runs one operation over arbitrary input with its default
// arguments. The operation is chosen by an index the fuzzer controls, so over
// a run it reaches every parser in the package.
func FuzzOperation(f *testing.F) {
	ops := fuzzableOps()
	if len(ops) == 0 {
		f.Fatal("no operations to fuzz")
	}

	// Seeds pair a few operations with input shaped like what they parse, so
	// the fuzzer starts inside the interesting code rather than at its guards.
	for _, seed := range []struct {
		op    int
		input string
	}{
		{0, ""},
		{1, "hello world"},
		{2, "\x7fELF\x02\x01\x01\x00"},
		{3, "\x1f\x8b\x08\x00\x00\x00\x00\x00"},
		{4, "\x50\x4b\x03\x04"},
		{5, "%PDF-1.4"},
		{6, "\x89PNG\r\n\x1a\n"},
		{7, "30 82 01 0a 02 82 01 01"},
		{8, "{\"a\":[1,2,{\"b\":null}]}"},
		{9, "<a b='c'><d/></a>"},
		{10, "function f(x) { return x + 1; }"},
		{11, "a1b2c3d4e5f6"},
		{12, "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----"},
	} {
		f.Add(seed.op, seed.input)
	}

	f.Fuzz(func(t *testing.T, which int, input string) {
		op := ops[((which%len(ops))+len(ops))%len(ops)]
		args, err := core.CoerceArgs(op.Args(), nil)
		if err != nil {
			t.Fatalf("%s: default arguments do not coerce: %v", op.Meta().Name, err)
		}
		// A panic here fails the test with the operation named and the input
		// saved, which is the whole point of the target.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s panicked on %q: %v", op.Meta().Name, input, r)
			}
		}()
		_, _ = op.Run(core.NewDish([]byte(input), op.Meta().InputType), args)
	})
}

// FuzzDecoders drives the decoding half of the reciprocal operations, where
// hostile input is most likely to arrive: every one of these reads a format
// somebody else produced.
// fuzzDecoderNames is the decoder list FuzzDecoders drives, in order.
func fuzzDecoderNames() []string {
	return []string{
		"From Base64", "From Base32", "From Base58", "From Base62", "From Base85",
		"From Base92", "From Base45", "From Hex", "From Binary", "From Octal",
		"From Decimal", "From Charcode", "From Hexdump", "From Braille",
		"From Punycode", "From COBS", "From MessagePack", "From Quoted Printable",
		"From Morse Code", "From Modhex", "From Case Insensitive Regex",
		"URL Decode", "Unescape string", "From HTML Entity", "From UNIX Timestamp",
		"Parse ASN.1 hex string", "CBOR Decode", "Protobuf Decode", "BSON deserialise",
		"Gunzip", "Zlib Inflate", "Raw Inflate", "Bzip2 Decompress", "LZMA Decompress",
		"LZ4 Decompress", "LZNT1 Decompress", "LZString Decompress", "Untar", "Unzip",
		"Detect File Type", "Scan for Embedded Files", "ELF Info", "Parse TLV",
		"Parse UDP", "Parse TCP", "Parse IPv4 header", "Parse Ethernet frame",
		"Parse TLS record", "Parse X.509 certificate", "PEM to Hex", "JWT Decode",
		"Avro to JSON", "AMF Decode", "JSON Beautify", "XML Beautify",
		"JavaScript Parser", "Disassemble x86", "Disassemble ARM", "YARA Rules",
		"Extract Files", "Extract EXIF", "Extract ID3", "Parse QR Code",
	}
}

func FuzzDecoders(f *testing.F) {
	names := fuzzDecoderNames()
	ops := make([]core.Operation, 0, len(names))
	for _, n := range names {
		if op, ok := core.Default.Get(n); ok {
			ops = append(ops, op)
		} else {
			f.Fatalf("operation %q is not registered; fix the name in this list", n)
		}
	}

	for _, seed := range []struct {
		op    int
		input string
	}{
		{0, "aGVsbG8="},
		{7, "48 65 6c 6c 6f"},
		{29, "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\x03"},
		{38, "\x50\x4b\x03\x04\x14\x00"},
		{41, "\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00"},
		{53, "{\"a\":1}"},
		{55, "let x = 1;"},
	} {
		f.Add(seed.op, seed.input)
	}

	f.Fuzz(func(t *testing.T, which int, input string) {
		op := ops[((which%len(ops))+len(ops))%len(ops)]
		args, err := core.CoerceArgs(op.Args(), nil)
		if err != nil {
			t.Fatalf("%s: default arguments do not coerce: %v", op.Meta().Name, err)
		}
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s panicked on %q: %v", op.Meta().Name, input, r)
			}
		}()
		_, _ = op.Run(core.NewDish([]byte(input), op.Meta().InputType), args)
	})
}

// FuzzRoundTrip checks the reciprocal operations against each other: whatever
// the encoder produces, the decoder must read back byte for byte.
func FuzzRoundTrip(f *testing.F) {
	// Only byte-level codecs belong here. Morse code and Braille carry no
	// letter case, MessagePack encodes JSON values rather than bytes, and
	// Everything here reads and writes bytes. The text codecs are deliberately
	// absent, because for them encode-then-decode is not meant to return the
	// input and CyberChef behaves the same way: Morse code and Braille carry
	// no letter case; MessagePack encodes JSON values; Base62 reads the input
	// as one big number, dropping leading zero bytes; Quoted Printable
	// normalises line endings to CRLF; and Charcode and LZString count
	// characters rather than bytes, so a two-byte character becomes one.
	pairs := []struct{ enc, dec string }{
		{"To Base64", "From Base64"},
		{"To Base32", "From Base32"},
		{"To Base58", "From Base58"},
		{"To Base85", "From Base85"},
		{"To Base92", "From Base92"},
		{"To Base45", "From Base45"},
		{"To Hex", "From Hex"},
		{"To Binary", "From Binary"},
		{"To Octal", "From Octal"},
		{"To Decimal", "From Decimal"},
		{"To Hexdump", "From Hexdump"},
		{"To COBS", "From COBS"},
		{"To Modhex", "From Modhex"},
		{"Gzip", "Gunzip"},
		{"Zlib Deflate", "Zlib Inflate"},
		{"Raw Deflate", "Raw Inflate"},
		{"Bzip2 Compress", "Bzip2 Decompress"},
		{"LZ4 Compress", "LZ4 Decompress"},
	}
	for _, p := range pairs {
		for _, n := range []string{p.enc, p.dec} {
			if _, ok := core.Default.Get(n); !ok {
				f.Fatalf("operation %q is not registered; fix the name in this list", n)
			}
		}
	}

	f.Add(0, "hello")
	f.Add(7, "")
	f.Add(14, "\x00\x00\x01\x00")
	f.Add(19, strings.Repeat("A", 300))

	f.Fuzz(func(t *testing.T, which int, input string) {
		p := pairs[((which%len(pairs))+len(pairs))%len(pairs)]
		enc, _ := core.Default.Get(p.enc)
		dec, _ := core.Default.Get(p.dec)

		encoded, err := runFuzzOp(enc, []byte(input))
		if err != nil {
			// An encoder refusing an input is a decision, not a defect.
			return
		}
		decoded, err := runFuzzOp(dec, encoded)
		if err != nil {
			t.Fatalf("%s produced output %s could not read: %v\ninput %q\nencoded %q",
				p.enc, p.dec, err, input, encoded)
		}
		if string(decoded) != input {
			t.Fatalf("%s -> %s changed the data:\n input   %q\n encoded %q\n decoded %q",
				p.enc, p.dec, input, encoded, decoded)
		}
	})
}

// runFuzzOp runs one operation with its default arguments, turning a panic
// into an error so a round-trip failure reports which half broke.
func runFuzzOp(op core.Operation, in []byte) (out []byte, err error) {
	args, err := core.CoerceArgs(op.Args(), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = &fuzzPanic{op: op.Meta().Name, value: r}
		}
	}()
	dish, err := op.Run(core.NewDish(in, op.Meta().InputType), args)
	if err != nil {
		return nil, err
	}
	return dish.Bytes(), nil
}

// fuzzPanic reports a panic as an error, naming the operation that raised it.
type fuzzPanic struct {
	op    string
	value any
}

func (p *fuzzPanic) Error() string {
	return p.op + " panicked: " + strings.TrimSpace(strings.SplitN(sprint(p.value), "\n", 2)[0])
}

func sprint(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return "non-string panic value"
}
