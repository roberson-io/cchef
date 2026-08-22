package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/internal/bytestream"
)

// ja4Ext is one parsed TLS extension: its type number, the raw type bytes, and
// the raw extension value bytes.
type ja4Ext struct {
	typ      int
	typeData []byte
	value    []byte
}

// ja4Hello holds the TLS Client/Server Hello fields JA4/JA4S need.
type ja4Hello struct {
	handshakeType int
	helloVersion  int
	cipherData    [][]byte // client: each entry is a 2-byte cipher suite
	cipherSuite   []byte   // server: the chosen 2-byte cipher suite
	extensions    []ja4Ext
}

// parseJA4Hello parses a single TLS handshake record (Client or Server Hello),
// mirroring lib/TLS.mjs's parseTLSRecord/parseHandshake framing.
func parseJA4Hello(data []byte) (*ja4Hello, error) {
	s := bytestream.New(data)
	if s.ReadInt(1) != 0x16 {
		return nil, fingerprintError("not handshake data")
	}
	s.ReadInt(2) // record version
	recLen := s.ReadInt(2)
	if s.Length() != recLen+5 {
		return nil, fingerprintError("incorrect handshake length")
	}
	hs := bytestream.New(s.GetBytes(recLen))

	h := &ja4Hello{}
	h.handshakeType = hs.ReadInt(1)
	hsLen := hs.ReadInt(3)
	if hs.Length() != hsLen+4 {
		return nil, fingerprintError("not enough data in handshake message")
	}
	h.helloVersion = hs.ReadInt(2)
	hs.GetBytes(32)            // random
	hs.GetBytes(hs.ReadInt(1)) // session ID

	switch h.handshakeType {
	case 0x01: // Client Hello
		cs := bytestream.New(hs.GetBytes(hs.ReadInt(2)))
		for cs.HasMore() {
			h.cipherData = append(h.cipherData, cs.GetBytes(2))
		}
		hs.GetBytes(hs.ReadInt(1)) // compression methods
	case 0x02: // Server Hello
		h.cipherSuite = hs.GetBytes(2)
		hs.GetBytes(1) // compression method
	default:
		return nil, fingerprintError("not a known handshake message")
	}

	es := bytestream.New(hs.GetBytes(hs.ReadInt(2)))
	for es.HasMore() {
		typeData := es.GetBytes(2)
		if len(typeData) < 2 {
			break
		}
		typ := int(typeData[0])<<8 | int(typeData[1])
		val := es.GetBytes(es.ReadInt(2))
		h.extensions = append(h.extensions, ja4Ext{typ: typ, typeData: typeData, value: val})
	}
	return h, nil
}

// tlsVersionMapper maps a TLS version number to its JA4 two-character code.
func tlsVersionMapper(version int) string {
	switch version {
	case 0x0304:
		return "13"
	case 0x0303:
		return "12"
	case 0x0302:
		return "11"
	case 0x0301:
		return "10"
	case 0x0300:
		return "s3"
	case 0x0200:
		return "s2"
	case 0x0100:
		return "s1"
	default:
		return "00"
	}
}

func isAlphanumericByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// alpnFingerprint returns the 2-character ALPN code for the first ALPN value.
func alpnFingerprint(raw []byte) string {
	if len(raw) == 0 {
		return "00"
	}
	first, last := raw[0], raw[len(raw)-1]
	if isAlphanumericByte(first) && isAlphanumericByte(last) {
		return string([]byte{first, last})
	}
	firstHex := fmt.Sprintf("%02x", first)
	lastHex := fmt.Sprintf("%02x", last)
	return string(firstHex[0]) + string(lastHex[1])
}

// parseHighestSupportedVersion reads the supported_versions extension value and
// returns the highest non-GREASE version (or the single value in a Server Hello).
func parseHighestSupportedVersion(b []byte) int {
	s := bytestream.New(b)
	if s.Length() == 2 {
		return s.ReadInt(2)
	}
	i := s.ReadInt(1)
	highest := 0
	for s.HasMore() && i > 0 {
		i--
		v := s.ReadInt(2)
		if greaseCipherSuites[v] {
			continue
		}
		if v > highest {
			highest = v
		}
	}
	return highest
}

// parseFirstALPNValue returns the first ALPN protocol value as raw bytes.
func parseFirstALPNValue(b []byte) []byte {
	s := bytestream.New(b)
	if s.ReadInt(2) < 2 {
		return nil
	}
	strLen := s.ReadInt(1)
	if strLen < 1 {
		return nil
	}
	return s.GetBytes(strLen)
}

func sha256Trunc12(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)[:12]
}

func pad2Count(n int) string {
	if n > 99 {
		return "99"
	}
	return fmt.Sprintf("%02d", n)
}

// commaEvery4 inserts a comma after each group of 4 characters (used for the
// signature-algorithms list, whose entries are 4 hex characters each).
func commaEvery4(s string) string {
	var parts []string
	for i := 0; i < len(s); i += 4 {
		end := min(i+4, len(s))
		parts = append(parts, s[i:end])
	}
	return strings.Join(parts, ",")
}

// toJA4 computes the JA4 fingerprints from a TLS Client Hello.
func toJA4(data []byte) (map[string]string, error) {
	h, err := parseJA4Hello(data)
	if err != nil || h.handshakeType != 0x01 {
		return nil, fingerprintError("data is not a valid TLS Client Hello (QUIC is not yet supported)")
	}

	version, sni, alpn := ja4ClientSummary(h)
	ver := tlsVersionMapper(version)

	cipherLen, sortedCiphersRaw, origCiphersRaw := ja4Ciphers(h)
	extLen, sortedExtsRaw, origExtsRaw := ja4Extensions(h)

	prefix := "t" + ver + sni + cipherLen + extLen + alpn
	return map[string]string{
		"JA4":    prefix + "_" + sha256Trunc12(sortedCiphersRaw) + "_" + sha256Trunc12(sortedExtsRaw),
		"JA4_o":  prefix + "_" + sha256Trunc12(origCiphersRaw) + "_" + sha256Trunc12(origExtsRaw),
		"JA4_r":  prefix + "_" + sortedCiphersRaw + "_" + sortedExtsRaw,
		"JA4_ro": prefix + "_" + origCiphersRaw + "_" + origExtsRaw,
	}, nil
}

// ja4ClientSummary scans the Client Hello extensions for the negotiated version,
// whether a server_name is present ("d"/"i"), and the ALPN fingerprint.
func ja4ClientSummary(h *ja4Hello) (version int, sni, alpn string) {
	version = h.helloVersion
	sni = "i"
	alpn = "00"
	for _, ext := range h.extensions {
		switch ext.typ {
		case 0x002b: // supported_versions
			version = parseHighestSupportedVersion(ext.value)
		case 0x0000: // server_name
			sni = "d"
		case 0x0010: // ALPN
			if alpn == "00" {
				alpn = alpnFingerprint(parseFirstALPNValue(ext.value))
			}
		}
	}
	return version, sni, alpn
}

// ja4Ciphers collects the non-GREASE cipher suites as hex, returning the padded
// count and both the sorted and original-order comma-joined lists.
func ja4Ciphers(h *ja4Hello) (cipherLen, sortedRaw, origRaw string) {
	var origCiphers []string
	for _, cd := range h.cipherData {
		v := int(cd[0])<<8 | int(cd[1])
		if !greaseCipherSuites[v] {
			origCiphers = append(origCiphers, hex.EncodeToString(cd))
		}
	}
	cipherLen = pad2Count(len(origCiphers))
	sortedCiphers := append([]string(nil), origCiphers...)
	sort.Strings(sortedCiphers)
	return cipherLen, strings.Join(sortedCiphers, ","), strings.Join(origCiphers, ",")
}

// ja4Extensions collects the non-GREASE extensions as hex and the
// signature_algorithms list, returning the padded count and both the sorted and
// original-order raw strings (each suffixed with "_" + signature algorithms).
func ja4Extensions(h *ja4Hello) (extLen, sortedRaw, origRaw string) {
	var origExts []string
	signatureAlgorithms := ""
	extCount := 0
	for _, ext := range h.extensions {
		if !greaseCipherSuites[ext.typ] {
			origExts = append(origExts, hex.EncodeToString(ext.typeData))
			extCount++
		}
		if ext.typ == 0x000d { // signature_algorithms
			sa := ext.value
			if len(sa) >= 2 {
				sa = sa[2:]
			} else {
				sa = nil
			}
			signatureAlgorithms = commaEvery4(hex.EncodeToString(sa))
		}
	}
	extLen = pad2Count(extCount)

	var sortedExts []string
	for _, e := range origExts {
		if e != "0000" && e != "0010" {
			sortedExts = append(sortedExts, e)
		}
	}
	sort.Strings(sortedExts)
	sortedRaw = strings.Join(sortedExts, ",") + "_" + signatureAlgorithms
	origRaw = strings.Join(origExts, ",") + "_" + signatureAlgorithms
	return extLen, sortedRaw, origRaw
}

// toJA4S computes the JA4S fingerprints from a TLS Server Hello.
func toJA4S(data []byte) (map[string]string, error) {
	h, err := parseJA4Hello(data)
	if err != nil || h.handshakeType != 0x02 {
		return nil, fingerprintError("data is not a valid TLS Server Hello (QUIC is not yet supported)")
	}

	version := h.helloVersion
	alpn := "00"
	for _, ext := range h.extensions {
		switch ext.typ {
		case 0x002b:
			version = parseHighestSupportedVersion(ext.value)
		case 0x0010:
			if alpn == "00" {
				alpn = alpnFingerprint(parseFirstALPNValue(ext.value))
			}
		}
	}
	ver := tlsVersionMapper(version)

	var extList []string
	for _, ext := range h.extensions {
		extList = append(extList, hex.EncodeToString(ext.typeData))
	}
	extRaw := strings.Join(extList, ",")
	cipher := hex.EncodeToString(h.cipherSuite)
	prefix := "t" + ver + pad2Count(len(h.extensions)) + alpn
	return map[string]string{
		"JA4S":   prefix + "_" + cipher + "_" + sha256Trunc12(extRaw),
		"JA4S_r": prefix + "_" + cipher + "_" + extRaw,
	}, nil
}
