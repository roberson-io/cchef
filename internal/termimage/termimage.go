// Package termimage renders images inline in terminals that support it.
//
// Two protocols are covered:
//   - iTerm2 (also honoured by WezTerm): a single OSC 1337 escape carrying the
//     raw file bytes base64-encoded. Format-agnostic — any image the terminal
//     can decode works.
//   - kitty graphics protocol: only PNG can be transmitted directly without a
//     decoder, so non-PNG input is rejected.
package termimage

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Protocol identifies a detected terminal graphics protocol.
type Protocol int

const (
	// None means the terminal supports no inline-image protocol.
	None Protocol = iota
	// ITerm is the iTerm2 OSC 1337 protocol, also honoured by WezTerm.
	ITerm
	// Kitty is the kitty graphics protocol.
	Kitty
)

// kittyChunk is the maximum base64 payload per kitty escape (protocol limit).
const kittyChunk = 4096

// Detect picks a protocol from the current process environment.
func Detect() Protocol {
	return ProtocolFrom(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"), os.Getenv("KITTY_WINDOW_ID"))
}

// ProtocolFrom picks a protocol from terminal environment values
// (TERM_PROGRAM, TERM and KITTY_WINDOW_ID).
func ProtocolFrom(termProgram, term, kittyWindowID string) Protocol {
	switch termProgram {
	case "iTerm.app", "WezTerm":
		return ITerm
	}
	if kittyWindowID != "" || strings.Contains(term, "kitty") {
		return Kitty
	}
	return None
}

// Encode renders data as an inline-image escape sequence for proto.
func Encode(proto Protocol, mime string, data []byte) ([]byte, error) {
	b64 := base64.StdEncoding.EncodeToString(data)
	switch proto {
	case ITerm:
		return fmt.Appendf(nil, "\x1b]1337;File=inline=1;size=%d:%s\x07\n", len(data), b64), nil
	case Kitty:
		if mime != "image/png" {
			return nil, fmt.Errorf("terminal preview via the kitty protocol requires a PNG image (got %q)", mime)
		}
		return kittyFrames(b64), nil
	default:
		return nil, fmt.Errorf("could not detect a supported terminal graphics protocol (iTerm2 or kitty)")
	}
}

// kittyFrames splits a base64 PNG payload into kitty graphics escapes, chunked
// to the protocol's per-escape limit with m=1 continuation frames.
func kittyFrames(b64 string) []byte {
	var b strings.Builder
	for pos := 0; pos < len(b64); pos += kittyChunk {
		end := min(pos+kittyChunk, len(b64))
		last := end == len(b64)
		b.WriteString("\x1b_G")
		if pos == 0 {
			b.WriteString("f=100,a=T,")
		}
		if last {
			b.WriteString("m=0;")
		} else {
			b.WriteString("m=1;")
		}
		b.WriteString(b64[pos:end])
		b.WriteString("\x1b\\")
	}
	b.WriteString("\n")
	return []byte(b.String())
}
