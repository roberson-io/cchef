package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// audioGolden is one sample and the report CyberChef produces for it. The
// samples are the ones CyberChef's own tests are built on
// (CyberChef's tests/samples/Audio.mjs); the reports came from the
// CyberChef-server oracle, which returns the operation's value rather than the
// HTML the upstream fixtures match against.
type audioGolden struct {
	Name     string          `json:"name"`
	Filename string          `json:"filename"`
	Hex      string          `json:"hex"`
	Want     json.RawMessage `json:"want"`
}

// readAudioGoldens loads the recorded reports.

func readAudioGoldens(t *testing.T) []audioGolden {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "audiometa.jsonl"))
	if err != nil {
		t.Fatalf("opening the goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []audioGolden
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var g audioGolden
		if err := json.Unmarshal([]byte(line), &g); err != nil {
			t.Fatalf("reading a golden: %v", err)
		}
		out = append(out, g)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading the goldens: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no goldens were read")
	}
	return out
}

// runExtractAudioMetadata runs the operation over one sample.

// runExtractAudioMetadata runs the operation over one sample.
func runExtractAudioMetadata(t *testing.T, data []byte, filename string, maxText float64) (string, error) {
	t.Helper()
	op, ok := core.Default.Get("Extract Audio Metadata")
	if !ok {
		t.Fatal("Extract Audio Metadata is not registered")
	}
	coerced, err := core.CoerceArgs(op.Args(), []any{filename, maxText})
	if err != nil {
		t.Fatalf("arguments: %v", err)
	}
	out, err := op.Run(core.NewDish(data, core.TypeArrayBuffer), coerced)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// TestExtractAudioMetadataFixtures covers every container the operation knows,
// against the report CyberChef produces for the same bytes — field for field and
// in the same order.

// TestExtractAudioMetadataFixtures covers every container the operation knows,
// against the report CyberChef produces for the same bytes — field for field and
// in the same order.
func TestExtractAudioMetadataFixtures(t *testing.T) {
	for _, g := range readAudioGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			data, err := hex.DecodeString(g.Hex)
			if err != nil {
				t.Fatalf("the sample is not hex: %v", err)
			}
			got, err := runExtractAudioMetadata(t, data, g.Filename, 524288)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != string(g.Want) {
				t.Errorf("got  %s\nwant %s", got, string(g.Want))
			}
		})
	}
}

// TestExtractAudioMetadataRejectsEmpty covers being given nothing, which cannot
// be a file of any kind.

// TestExtractAudioMetadataRejectsEmpty covers being given nothing, which cannot
// be a file of any kind.
func TestExtractAudioMetadataRejectsEmpty(t *testing.T) {
	_, err := runExtractAudioMetadata(t, nil, "", 524288)
	if err == nil {
		t.Fatal("an empty input was read as audio")
	}
	const want = "No input data. Load an audio file (drag/drop or use the open file button)."
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

// TestExtractAudioMetadataUnknownContainer covers bytes that are not audio at
// all: a report is still produced, saying so.

// TestExtractAudioMetadataUnknownContainer covers bytes that are not audio at
// all: a report is still produced, saying so.
func TestExtractAudioMetadataUnknownContainer(t *testing.T) {
	got, err := runExtractAudioMetadata(t, []byte("this is not audio"), "", 524288)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{
		`"container":{"type":"unknown","brand":null,"mime":null}`,
		`"metadata_systems":[]`,
		`"Unknown/unsupported container (best-effort scan not implemented)."`,
		`"filename":null`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not contain %s:\n%s", want, got)
		}
	}
}

// TestExtractAudioMetadataFilename covers the filename argument, which is
// recorded in the report and otherwise plays no part.

// TestExtractAudioMetadataFilename covers the filename argument, which is
// recorded in the report and otherwise plays no part.
func TestExtractAudioMetadataFilename(t *testing.T) {
	for _, tc := range []struct{ name, given, want string }{
		{"a name", "song.mp3", `"filename":"song.mp3"`},
		{"a name with spaces around it", "  song.mp3  ", `"filename":"song.mp3"`},
		{"nothing", "", `"filename":null`},
		{"only spaces", "   ", `"filename":null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runExtractAudioMetadata(t, []byte("not audio"), tc.given, 524288)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("the report does not contain %s:\n%.120s", tc.want, got)
			}
		})
	}
}

// TestExtractAudioMetadataByteLength covers the length being reported as given,
// which is how a truncated file shows itself.

// TestExtractAudioMetadataByteLength covers the length being reported as given,
// which is how a truncated file shows itself.
func TestExtractAudioMetadataByteLength(t *testing.T) {
	got, err := runExtractAudioMetadata(t, []byte("not audio at all"), "", 524288)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, `"byte_length":16`) {
		t.Errorf("the length was not reported as 16:\n%.150s", got)
	}
}

// TestExtractAudioMetadataDefaults covers the arguments the operation starts
// from.

// TestExtractAudioMetadataDefaults covers the arguments the operation starts
// from.
func TestExtractAudioMetadataDefaults(t *testing.T) {
	op, ok := core.Default.Get("Extract Audio Metadata")
	if !ok {
		t.Fatal("Extract Audio Metadata is not registered")
	}
	args := op.Args()
	if len(args) != 2 {
		t.Fatalf("declares %d arguments, want 2", len(args))
	}
	if args[0].Name != "Filename (optional)" || args[0].Value != "" {
		t.Errorf("first argument is %q = %v", args[0].Name, args[0].Value)
	}
	if args[1].Name != "Max embedded text bytes (iXML/axml/etc)" || args[1].Value != float64(1024*512) {
		t.Errorf("second argument is %q = %v", args[1].Name, args[1].Value)
	}
	if _, err := core.CoerceArgs(args, nil); err != nil {
		t.Errorf("the operation's own defaults were turned away: %v", err)
	}
}

// TestAudioByteHelpers covers the readers the container parsers are built from,
// including the reads that run off the end of the buffer. The JavaScript gets
// undefined there, which its arithmetic turns into zero and its string building
// into nothing.

// TestAudioRunIgnoresAnUnusableLimit covers the embedded-text limit being given
// a value that is not a number, which falls back to the default.
func TestAudioRunIgnoresAnUnusableLimit(t *testing.T) {
	if !isNaNOrInf(math.NaN()) || !isNaNOrInf(math.Inf(1)) || !isNaNOrInf(math.Inf(-1)) {
		t.Error("isNaNOrInf does not recognise the values JavaScript calls non-finite")
	}
	if isNaNOrInf(1024) {
		t.Error("an ordinary number was called non-finite")
	}

	got, err := runExtractAudioMetadata(t, []byte("not audio"), "", math.NaN())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, `"schema_version":"audio-meta-1.0"`) {
		t.Error("a report was not produced")
	}
}

// TestAudioNullTerminatedWideRunsOut covers a UTF-16 field with no terminator
// before the end of the frame. The scan steps two bytes at a time, so it stops
// at the last whole character pair rather than taking the odd byte after it.
