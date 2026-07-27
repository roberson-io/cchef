package ops

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Terminal inline-image previews for the "Terminal" output option.
//
// Two protocols are supported:
//   - iTerm2 (also honoured by WezTerm): a single OSC 1337 escape carrying the
//     raw file bytes base64-encoded. Format-agnostic — any image the terminal
//     can decode works.
//   - kitty graphics protocol: only PNG can be transmitted directly without a
//     decoder, so non-PNG input is rejected.

// termProtocol identifies a detected terminal graphics protocol.
type termProtocol int

const (
	termNone termProtocol = iota
	termITerm
	termKitty
)

// kittyChunk is the maximum base64 payload per kitty escape (protocol limit).
const kittyChunk = 4096

// detectTermProtocol picks a protocol from the current process environment.
func detectTermProtocol() termProtocol {
	return termProtocolFrom(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"), os.Getenv("KITTY_WINDOW_ID"))
}

// termProtocolFrom picks a protocol from terminal environment values
// (TERM_PROGRAM, TERM and KITTY_WINDOW_ID).
func termProtocolFrom(termProgram, term, kittyWindowID string) termProtocol {
	switch termProgram {
	case "iTerm.app", "WezTerm":
		return termITerm
	}
	if kittyWindowID != "" || strings.Contains(term, "kitty") {
		return termKitty
	}
	return termNone
}

// encodeTerminalImage renders data as an inline-image escape sequence for proto.
func encodeTerminalImage(proto termProtocol, mime string, data []byte) ([]byte, error) {
	b64 := base64.StdEncoding.EncodeToString(data)
	switch proto {
	case termITerm:
		return fmt.Appendf(nil, "\x1b]1337;File=inline=1;size=%d:%s\x07\n", len(data), b64), nil
	case termKitty:
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
