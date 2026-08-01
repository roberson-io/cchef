package ops

import (
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseTLSRecord{})
}

// merge appends all of src's keys (in order) into o.
func (o *omap) merge(src *omap) {
	for _, k := range src.keys {
		o.set(k, src.vals[k])
	}
}

var tlsContentTypes = map[int]string{
	20: "change_cipher_spec", 21: "alert", 22: "handshake", 23: "application_data",
}

var tlsHandshakeTypes = map[int]string{
	0: "hello_request", 1: "client_hello", 2: "server_hello", 4: "new_session_ticket",
	11: "certificate", 12: "server_key_exchange", 13: "certificate_request",
	14: "server_hello_done", 15: "certificate_verify", 16: "client_key_exchange", 20: "finished",
}

// tlsReadBytesAsHex reads n bytes as "0x…" hex, or "" if fewer than n remain.
func tlsReadBytesAsHex(s *byteStream, n int) string {
	b := s.getBytes(n)
	if len(b) != n {
		return ""
	}
	return "0x" + toHexFast(b)
}

// tlsReadSizePrefixed reads a size-prefixed byte field as hex.
func tlsReadSizePrefixed(s *byteStream, sizePrefixLen int) string {
	length := s.readInt(sizePrefixLen)
	if length == 0 {
		return ""
	}
	return tlsReadBytesAsHex(s, length)
}

// tlsReadList reads a length-prefixed list of fixed-size or size-prefixed hex
// values into an omap {length, [truncated], values}. lengthBytes is the size of
// the list length field; readItem reads one item from the sub-stream.
func tlsReadList(s *byteStream, lengthBytes int, readItem func(*byteStream) string) *omap {
	length := s.readInt(lengthBytes)
	if length == 0 {
		return newOMap()
	}
	o := newOMap()
	o.set("length", length)
	sub := newByteStream(s.getBytes(length))
	if sub.length() < length {
		o.set("truncated", true)
	}
	values := []string{}
	for sub.hasMore() {
		if v := readItem(sub); v != "" {
			values = append(values, v)
		}
	}
	o.set("values", values)
	return o
}

// tlsReadExtensions reads the extensions field into {length, [truncated], values}.
func tlsReadExtensions(s *byteStream) *omap {
	length := s.readInt(2)
	if length == 0 {
		return newOMap()
	}
	o := newOMap()
	o.set("length", length)
	sub := newByteStream(s.getBytes(length))
	if sub.length() < length {
		o.set("truncated", true)
	}
	exts := []*omap{}
	for sub.hasMore() {
		if e := tlsReadExtension(sub); e != nil {
			exts = append(exts, e)
		}
	}
	o.set("values", exts)
	return o
}

// tlsReadExtension reads one Hello extension.
func tlsReadExtension(s *byteStream) *omap {
	if s.pos+4 > len(s.bytes) {
		s.moveTo(len(s.bytes))
		return nil
	}
	o := newOMap()
	o.set("type", "0x"+toHexFast(s.getBytes(2)))
	length := s.readInt(2)
	o.set("length", length)
	if length == 0 {
		return o
	}
	value := s.getBytes(length)
	if len(value) != length {
		o.set("truncated", true)
	}
	if len(value) > 0 {
		o.set("value", "0x"+toHexFast(value))
	}
	return o
}

func tlsParseClientHello(s *byteStream) *omap {
	o := newOMap()
	o.set("clientVersion", tlsReadBytesAsHex(s, 2))
	o.set("random", tlsReadBytesAsHex(s, 32))
	if sid := tlsReadSizePrefixed(s, 1); sid != "" {
		o.set("sessionID", sid)
	}
	o.set("cipherSuites", tlsReadList(s, 2, func(x *byteStream) string { return tlsReadBytesAsHex(x, 2) }))
	o.set("compressionMethods", tlsReadList(s, 1, func(x *byteStream) string { return tlsReadBytesAsHex(x, 1) }))
	o.set("extensions", tlsReadExtensions(s))
	return o
}

func tlsParseServerHello(s *byteStream) *omap {
	o := newOMap()
	o.set("serverVersion", tlsReadBytesAsHex(s, 2))
	o.set("random", tlsReadBytesAsHex(s, 32))
	if sid := tlsReadSizePrefixed(s, 1); sid != "" {
		o.set("sessionID", sid)
	}
	o.set("cipherSuite", tlsReadBytesAsHex(s, 2))
	o.set("compressionMethod", tlsReadBytesAsHex(s, 1))
	o.set("extensions", tlsReadExtensions(s))
	return o
}

func tlsParseNewSessionTicket(s *byteStream) *omap {
	o := newOMap()
	lifetime := ""
	if s.pos+4 > len(s.bytes) {
		s.moveTo(len(s.bytes))
	} else {
		lifetime = strconv.Itoa(s.readInt(4)) + "s"
	}
	o.set("ticketLifetimeHint", lifetime)
	o.set("ticket", tlsReadSizePrefixed(s, 2))
	return o
}

func tlsParseCertificate(s *byteStream) *omap {
	o := newOMap()
	list := newOMap()
	if s.pos+3 > len(s.bytes) {
		s.moveTo(len(s.bytes))
	} else {
		length := s.readInt(3)
		list.set("length", length)
		if length != 0 {
			sub := newByteStream(s.getBytes(length))
			if sub.length() < length {
				list.set("truncated", true)
			}
			values := []string{}
			for sub.hasMore() {
				if c := tlsReadSizePrefixed(sub, 3); c != "" {
					values = append(values, c)
				}
			}
			list.set("values", values)
		}
	}
	o.set("certificateList", list)
	return o
}

func tlsParseCertificateRequest(s *byteStream) *omap {
	o := newOMap()
	o.set("certificateTypes", tlsReadList(s, 1, func(x *byteStream) string { return tlsReadBytesAsHex(x, 1) }))
	o.set("supportedSignatureAlgorithms", tlsReadList(s, 2, func(x *byteStream) string { return tlsReadBytesAsHex(x, 2) }))
	cas := tlsReadList(s, 2, func(x *byteStream) string { return tlsReadSizePrefixed(x, 2) })
	if l, ok := cas.vals["length"].(int); ok && l > 0 {
		o.set("certificateAuthorities", cas)
	}
	return o
}

func tlsParseCertificateVerify(s *byteStream) *omap {
	o := newOMap()
	o.set("algorithmHash", tlsReadBytesAsHex(s, 1))
	o.set("algorithmSignature", tlsReadBytesAsHex(s, 1))
	o.set("signature", tlsReadSizePrefixed(s, 2))
	return o
}

// tlsParseHandshake parses a handshake message into the record omap.
func tlsParseHandshake(s *byteStream, rec *omap) *omap {
	if !s.hasMore() {
		return rec
	}
	handshakeType := s.readInt(1)
	htName := tlsHandshakeTypes[handshakeType]
	if htName == "" {
		htName = strconv.Itoa(handshakeType)
	}
	rec.set("handshakeType", htName)

	if s.pos+3 > len(s.bytes) {
		s.moveTo(len(s.bytes))
		return rec
	}
	handshakeLength := s.readInt(3)
	if handshakeLength+4 != rec.vals["length"].(int) {
		s.moveTo(0)
		rec.set("handshakeType", tlsHandshakeTypes[20]) // finished
		rec.set("handshakeValue", "0x"+toHexFast(s.bytes))
		return rec
	}
	content := s.getBytes(handshakeLength)
	if len(content) == 0 {
		return rec
	}
	sub := newByteStream(content)
	switch handshakeType {
	case 1:
		rec.merge(tlsParseClientHello(sub))
	case 2:
		rec.merge(tlsParseServerHello(sub))
	case 4:
		rec.merge(tlsParseNewSessionTicket(sub))
	case 11:
		rec.merge(tlsParseCertificate(sub))
	case 13:
		rec.merge(tlsParseCertificateRequest(sub))
	case 15:
		rec.merge(tlsParseCertificateVerify(sub))
	default:
		rec.set("handshakeValue", "0x"+toHexFast(content))
	}
	return rec
}

// tlsReadRecord reads one TLS record from s.
func tlsReadRecord(s *byteStream) *omap {
	const recordHeaderLen = 5
	if s.pos+recordHeaderLen > len(s.bytes) {
		s.moveTo(len(s.bytes))
		return nil
	}
	typ := s.readInt(1)
	typeString := tlsContentTypes[typ]
	if typeString == "" {
		typeString = strconv.Itoa(typ)
	}
	version := "0x" + toHexFast(s.getBytes(2))
	length := s.readInt(2)
	content := s.getBytes(length)

	rec := newOMap()
	rec.set("type", typeString)
	rec.set("version", version)
	rec.set("length", length)
	if len(content) < length {
		rec.set("truncated", true)
	}
	if len(content) == 0 {
		return rec
	}
	if typ == 22 { // handshake
		return tlsParseHandshake(newByteStream(content), rec)
	}
	rec.set("value", "0x"+toHexFast(content))
	return rec
}

// ParseTLSRecord parses one or more raw TLS records into structured JSON.
type ParseTLSRecord struct{}

// Meta returns the operation metadata.
func (ParseTLSRecord) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse TLS record",
		Module:      "Default",
		Description: "Parses one or more TLS records",
		InfoURL:     "https://wikipedia.org/wiki/Transport_Layer_Security",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ParseTLSRecord) Args() []core.ArgDef { return nil }

// Run parses the records. Ported from CyberChef ParseTLSRecord.mjs.
func (ParseTLSRecord) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := newByteStream(in.Bytes())
	records := []*omap{}
	for s.hasMore() {
		if rec := tlsReadRecord(s); rec != nil {
			records = append(records, rec)
		}
	}
	out, err := jsonNoEscape(records)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
