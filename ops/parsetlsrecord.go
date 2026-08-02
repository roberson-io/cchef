package ops

import (
	"encoding/hex"
	"strconv"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(ParseTLSRecord{})
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
func tlsReadBytesAsHex(s *bytestream.Stream, n int) string {
	b := s.GetBytes(n)
	if len(b) != n {
		return ""
	}
	return "0x" + hex.EncodeToString(b)
}

// tlsReadSizePrefixed reads a size-prefixed byte field as hex.
func tlsReadSizePrefixed(s *bytestream.Stream, sizePrefixLen int) string {
	length := s.ReadInt(sizePrefixLen)
	if length == 0 {
		return ""
	}
	return tlsReadBytesAsHex(s, length)
}

// tlsReadList reads a length-prefixed list of fixed-size or size-prefixed hex
// values into an omap {length, [truncated], values}. lengthBytes is the size of
// the list length field; readItem reads one item from the sub-stream.
func tlsReadList(s *bytestream.Stream, lengthBytes int, readItem func(*bytestream.Stream) string) *jsonval.OMap {
	length := s.ReadInt(lengthBytes)
	if length == 0 {
		return jsonval.NewOMap()
	}
	o := jsonval.NewOMap()
	o.Set("length", length)
	sub := bytestream.New(s.GetBytes(length))
	if sub.Length() < length {
		o.Set("truncated", true)
	}
	values := []string{}
	for sub.HasMore() {
		if v := readItem(sub); v != "" {
			values = append(values, v)
		}
	}
	o.Set("values", values)
	return o
}

// tlsReadExtensions reads the extensions field into {length, [truncated], values}.
func tlsReadExtensions(s *bytestream.Stream) *jsonval.OMap {
	length := s.ReadInt(2)
	if length == 0 {
		return jsonval.NewOMap()
	}
	o := jsonval.NewOMap()
	o.Set("length", length)
	sub := bytestream.New(s.GetBytes(length))
	if sub.Length() < length {
		o.Set("truncated", true)
	}
	exts := []*jsonval.OMap{}
	for sub.HasMore() {
		if e := tlsReadExtension(sub); e != nil {
			exts = append(exts, e)
		}
	}
	o.Set("values", exts)
	return o
}

// tlsReadExtension reads one Hello extension.
func tlsReadExtension(s *bytestream.Stream) *jsonval.OMap {
	if s.Pos+4 > len(s.Bytes) {
		s.MoveTo(len(s.Bytes))
		return nil
	}
	o := jsonval.NewOMap()
	o.Set("type", "0x"+hex.EncodeToString(s.GetBytes(2)))
	length := s.ReadInt(2)
	o.Set("length", length)
	if length == 0 {
		return o
	}
	value := s.GetBytes(length)
	if len(value) != length {
		o.Set("truncated", true)
	}
	if len(value) > 0 {
		o.Set("value", "0x"+hex.EncodeToString(value))
	}
	return o
}

func tlsParseClientHello(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	o.Set("clientVersion", tlsReadBytesAsHex(s, 2))
	o.Set("random", tlsReadBytesAsHex(s, 32))
	if sid := tlsReadSizePrefixed(s, 1); sid != "" {
		o.Set("sessionID", sid)
	}
	o.Set("cipherSuites", tlsReadList(s, 2, func(x *bytestream.Stream) string { return tlsReadBytesAsHex(x, 2) }))
	o.Set("compressionMethods", tlsReadList(s, 1, func(x *bytestream.Stream) string { return tlsReadBytesAsHex(x, 1) }))
	o.Set("extensions", tlsReadExtensions(s))
	return o
}

func tlsParseServerHello(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	o.Set("serverVersion", tlsReadBytesAsHex(s, 2))
	o.Set("random", tlsReadBytesAsHex(s, 32))
	if sid := tlsReadSizePrefixed(s, 1); sid != "" {
		o.Set("sessionID", sid)
	}
	o.Set("cipherSuite", tlsReadBytesAsHex(s, 2))
	o.Set("compressionMethod", tlsReadBytesAsHex(s, 1))
	o.Set("extensions", tlsReadExtensions(s))
	return o
}

func tlsParseNewSessionTicket(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	lifetime := ""
	if s.Pos+4 > len(s.Bytes) {
		s.MoveTo(len(s.Bytes))
	} else {
		lifetime = strconv.Itoa(s.ReadInt(4)) + "s"
	}
	o.Set("ticketLifetimeHint", lifetime)
	o.Set("ticket", tlsReadSizePrefixed(s, 2))
	return o
}

func tlsParseCertificate(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	list := jsonval.NewOMap()
	if s.Pos+3 > len(s.Bytes) {
		s.MoveTo(len(s.Bytes))
	} else {
		length := s.ReadInt(3)
		list.Set("length", length)
		if length != 0 {
			sub := bytestream.New(s.GetBytes(length))
			if sub.Length() < length {
				list.Set("truncated", true)
			}
			values := []string{}
			for sub.HasMore() {
				if c := tlsReadSizePrefixed(sub, 3); c != "" {
					values = append(values, c)
				}
			}
			list.Set("values", values)
		}
	}
	o.Set("certificateList", list)
	return o
}

func tlsParseCertificateRequest(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	o.Set("certificateTypes", tlsReadList(s, 1, func(x *bytestream.Stream) string { return tlsReadBytesAsHex(x, 1) }))
	o.Set("supportedSignatureAlgorithms", tlsReadList(s, 2, func(x *bytestream.Stream) string { return tlsReadBytesAsHex(x, 2) }))
	cas := tlsReadList(s, 2, func(x *bytestream.Stream) string { return tlsReadSizePrefixed(x, 2) })
	if l, ok := cas.Value("length").(int); ok && l > 0 {
		o.Set("certificateAuthorities", cas)
	}
	return o
}

func tlsParseCertificateVerify(s *bytestream.Stream) *jsonval.OMap {
	o := jsonval.NewOMap()
	o.Set("algorithmHash", tlsReadBytesAsHex(s, 1))
	o.Set("algorithmSignature", tlsReadBytesAsHex(s, 1))
	o.Set("signature", tlsReadSizePrefixed(s, 2))
	return o
}

// tlsParseHandshake parses a handshake message into the record omap.
func tlsParseHandshake(s *bytestream.Stream, rec *jsonval.OMap) *jsonval.OMap {
	if !s.HasMore() {
		return rec
	}
	handshakeType := s.ReadInt(1)
	htName := tlsHandshakeTypes[handshakeType]
	if htName == "" {
		htName = strconv.Itoa(handshakeType)
	}
	rec.Set("handshakeType", htName)

	if s.Pos+3 > len(s.Bytes) {
		s.MoveTo(len(s.Bytes))
		return rec
	}
	handshakeLength := s.ReadInt(3)
	if handshakeLength+4 != rec.Value("length").(int) {
		s.MoveTo(0)
		rec.Set("handshakeType", tlsHandshakeTypes[20]) // finished
		rec.Set("handshakeValue", "0x"+hex.EncodeToString(s.Bytes))
		return rec
	}
	content := s.GetBytes(handshakeLength)
	if len(content) == 0 {
		return rec
	}
	sub := bytestream.New(content)
	switch handshakeType {
	case 1:
		rec.Merge(tlsParseClientHello(sub))
	case 2:
		rec.Merge(tlsParseServerHello(sub))
	case 4:
		rec.Merge(tlsParseNewSessionTicket(sub))
	case 11:
		rec.Merge(tlsParseCertificate(sub))
	case 13:
		rec.Merge(tlsParseCertificateRequest(sub))
	case 15:
		rec.Merge(tlsParseCertificateVerify(sub))
	default:
		rec.Set("handshakeValue", "0x"+hex.EncodeToString(content))
	}
	return rec
}

// tlsReadRecord reads one TLS record from s.
func tlsReadRecord(s *bytestream.Stream) *jsonval.OMap {
	const recordHeaderLen = 5
	if s.Pos+recordHeaderLen > len(s.Bytes) {
		s.MoveTo(len(s.Bytes))
		return nil
	}
	typ := s.ReadInt(1)
	typeString := tlsContentTypes[typ]
	if typeString == "" {
		typeString = strconv.Itoa(typ)
	}
	version := "0x" + hex.EncodeToString(s.GetBytes(2))
	length := s.ReadInt(2)
	content := s.GetBytes(length)

	rec := jsonval.NewOMap()
	rec.Set("type", typeString)
	rec.Set("version", version)
	rec.Set("length", length)
	if len(content) < length {
		rec.Set("truncated", true)
	}
	if len(content) == 0 {
		return rec
	}
	if typ == 22 { // handshake
		return tlsParseHandshake(bytestream.New(content), rec)
	}
	rec.Set("value", "0x"+hex.EncodeToString(content))
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
	s := bytestream.New(in.Bytes())
	records := []*jsonval.OMap{}
	for s.HasMore() {
		if rec := tlsReadRecord(s); rec != nil {
			records = append(records, rec)
		}
	}
	out, err := jsonval.MarshalNoEscape(records)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
