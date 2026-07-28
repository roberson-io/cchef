package ops

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// audioMaxTextFloor is the smallest embedded-text limit the operation will work
// to, however small a figure it is given.
const audioMaxTextFloor = 1024

// audioMaxTextDefault is how much of an embedded text payload is kept by default.
const audioMaxTextDefault = 1024 * 512

// errAudioNoInput is what an empty input gets.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errAudioNoInput = errors.New(
	"No input data. Load an audio file (drag/drop or use the open file button).")

// ExtractAudioMetadata reads the metadata out of an audio file.
type ExtractAudioMetadata struct{}

// Meta returns the operation metadata.
func (ExtractAudioMetadata) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract Audio Metadata",
		Module: "Default",
		Description: "Extract common audio metadata across MP3 (ID3v2/ID3v1/GEOB), " +
			"WAV/BWF/BW64 (INFO/bext/iXML/axml), FLAC (Vorbis Comment/Picture), OGG " +
			"(Vorbis/OpusTags), AAC (ADTS), AC3 (Dolby Digital), WMA (ASF), plus " +
			"best-effort MP4/M4A and AIFF scanning. Outputs normalized JSON.",
		InfoURL:    "https://wikipedia.org/wiki/Audio_file_format",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ExtractAudioMetadata) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Filename (optional)", Type: core.ArgString, Value: ""},
		{
			Name:  "Max embedded text bytes (iXML/axml/etc)",
			Type:  core.ArgNumber,
			Value: float64(audioMaxTextDefault),
		},
	}
}

// Run reads the file.
func (ExtractAudioMetadata) Run(in *core.Dish, args []any) (*core.Dish, error) {
	filename, _ := args[0].(string)
	filename = strings.TrimSpace(filename)

	maxText := audioMaxTextDefault
	if given, ok := args[1].(float64); ok && !isNaNOrInf(given) {
		maxText = max(int(given), audioMaxTextFloor)
	}

	data := in.Bytes()
	if len(data) == 0 {
		return nil, errAudioNoInput
	}

	container := sniffContainer(data)
	report := newAudioReport(filename, len(data), container)
	audioParse(data, report, container.typ, maxText)

	encoded, err := marshalOMap(report.root)
	if err != nil {
		return nil, err
	}
	return core.NewDish(encoded, core.TypeJSON), nil
}

// isNaNOrInf reports whether f is one of the values JavaScript's isFinite turns
// away, which the argument is checked for before being used as a limit.
func isNaNOrInf(f float64) bool { return f != f || f > 1e308 || f < -1e308 }

// audioParse hands the file to the reader for its container, or notes that there
// is not one.
func audioParse(b []byte, r *audioReport, container string, maxText int) {
	switch container {
	case "mp3":
		parseMp3(b, r)
	case "wav", "bw64":
		parseRiffWave(b, r, maxText)
	case "flac":
		parseFlac(b, r, maxText)
	case "ogg", "opus":
		parseOgg(b, r)
	case "mp4", "m4a":
		parseMp4BestEffort(b, r)
	case "aiff":
		parseAiffBestEffort(b, r, maxText)
	case "aac":
		parseAacAdts(b, r)
	case "ac3":
		parseAc3(b, r)
	case "wma":
		parseWmaAsf(b, r)
	default:
		r.addError("sniff", "Unknown/unsupported container (best-effort scan not implemented).")
	}
}

// audioContainer is what the opening bytes of a file say it is.
type audioContainer struct {
	typ   string
	mime  string
	brand string
}

// sniffContainer works out the container format from the file's first bytes.
// Some formats are named by a byte pattern at the very start, others by a
// four-character tag a little way in, so the two are looked for separately.
func sniffContainer(b []byte) audioContainer {
	if c, ok := sniffByBytePattern(b); ok {
		return c
	}
	if c, ok := sniffByTag(b); ok {
		return c
	}
	return audioContainer{typ: "unknown"}
}

// audioMagics are the formats named by a fixed run of bytes at the very start.
// A format is only considered when the file is long enough to hold the header
// that follows the run, which is what minLength records.
var audioMagics = []struct {
	prefix    []byte
	minLength int
	container audioContainer
}{
	{[]byte("ID3"), 3, audioContainer{typ: "mp3", mime: "audio/mpeg"}},
	{[]byte{0x0b, 0x77}, 8, audioContainer{typ: "ac3", mime: "audio/ac3"}},
	{
		[]byte{0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11},
		16,
		audioContainer{typ: "wma", mime: "audio/x-ms-wma"},
	},
}

// sniffByBytePattern names the formats whose first bytes are a fixed run, or the
// set bits that open an audio frame.
func sniffByBytePattern(b []byte) (audioContainer, bool) {
	if c, ok := sniffFrameSync(b); ok {
		return c, true
	}
	for _, m := range audioMagics {
		if len(b) >= m.minLength && bytes.HasPrefix(b, m.prefix) {
			return m.container, true
		}
	}
	return audioContainer{}, false
}

// sniffFrameSync names the two formats that open with a frame rather than a
// header. Eleven set bits start a frame of either; the layer bits say which.
func sniffFrameSync(b []byte) (audioContainer, bool) {
	if len(b) < 2 || b[0] != 0xff || b[1]&0xe0 != 0xe0 {
		return audioContainer{}, false
	}
	if b[1]&0x06 == 0x00 {
		return audioContainer{typ: "aac", mime: "audio/aac"}, true
	}
	return audioContainer{typ: "mp3", mime: "audio/mpeg"}, true
}

// sniffByTag names the formats identified by a four-character tag, either at the
// start of the file or just inside it.
func sniffByTag(b []byte) (audioContainer, bool) {
	if len(b) >= 12 {
		if c, ok := sniffChunkedFamily(b); ok {
			return c, true
		}
	}
	if len(b) < 4 {
		return audioContainer{}, false
	}
	switch ascii4(b, 0) {
	case "fLaC":
		return audioContainer{typ: "flac", mime: "audio/flac"}, true
	case "OggS":
		// An Ogg stream carries either Vorbis or Opus; only the Opus one names
		// itself in the first page.
		if indexOfASCII(b, "OpusHead", 0, min(len(b), 65536)) >= 0 {
			return audioContainer{typ: "opus", mime: "audio/ogg"}, true
		}
		return audioContainer{typ: "ogg", mime: "audio/ogg"}, true
	}
	return audioContainer{}, false
}

// sniffChunkedFamily names the formats built out of chunks, which say what they
// hold in a second tag after the one that opens them.
func sniffChunkedFamily(b []byte) (audioContainer, bool) {
	head, form := ascii4(b, 0), ascii4(b, 8)
	switch {
	case head == "RIFF" && form == "WAVE":
		return audioContainer{typ: "wav", mime: "audio/wav"}, true
	case head == "BW64" && form == "WAVE":
		return audioContainer{typ: "bw64", mime: "audio/wav"}, true
	case ascii4(b, 4) == "ftyp":
		return sniffMp4Brand(form), true
	case head == "FORM" && (form == "AIFF" || form == "AIFC"):
		return audioContainer{typ: "aiff", mime: "audio/aiff", brand: form}, true
	}
	return audioContainer{}, false
}

// sniffMp4Brand tells an audio-only MP4 from the general one by its brand.
func sniffMp4Brand(brand string) audioContainer {
	switch brand {
	case "M4A ", "M4B ", "M4P ":
		return audioContainer{typ: "m4a", mime: "audio/mp4", brand: brand}
	}
	return audioContainer{typ: "mp4", mime: "video/mp4", brand: brand}
}

// audioReport is the document the operation builds. The pieces a parser reaches
// into are held separately so that they can be added to without walking back
// down from the root, while the root keeps the order everything is written in.
type audioReport struct {
	root     *omap
	common   *omap
	raw      *omap
	systems  []any
	embedded []any
	errs     []any

	detections *omap
	tags       *omap
	container  *omap
}

// newAudioReport builds the empty report a parser fills in.
func newAudioReport(filename string, byteLength int, c audioContainer) *audioReport {
	r := &audioReport{
		systems:  []any{},
		embedded: []any{},
		errs:     []any{},
	}

	container := newOMap().
		set("type", c.typ).
		set("brand", nullIfEmpty(c.brand)).
		set("mime", nullIfEmpty(c.mime))

	artifact := newOMap().
		set("filename", nullIfEmpty(filename)).
		set("byte_length", byteLength).
		set("container", container)

	r.detections = newOMap().
		set("metadata_systems", r.systems).
		set("provenance_systems", []any{})

	r.common = newOMap()
	for _, field := range audioCommonFields {
		r.common.set(field, nil)
	}
	r.raw = newOMap()
	r.tags = newOMap().set("common", r.common).set("raw", r.raw)

	r.root = newOMap().
		set("schema_version", "audio-meta-1.0").
		set("artifact", artifact).
		set("detections", r.detections).
		set("tags", r.tags).
		set("embedded", r.embedded).
		set("provenance", newAudioProvenance()).
		set("errors", r.errs)

	r.container = container
	return r
}

// audioCommonFields are the tags every container is boiled down to, in the order
// the report lists them.
var audioCommonFields = []string{
	"title", "artist", "album", "date", "track",
	"genre", "comment", "composer", "copyright", "language",
}

// newAudioProvenance builds the empty provenance section, which only the MP3
// reader ever writes to.
func newAudioProvenance() *omap {
	certificate := newOMap().
		set("subject_cn", nil).
		set("issuer_cn", nil).
		set("serial_number", nil)

	c2pa := newOMap().
		set("present", false).
		set("embedding", []any{}).
		set("manifest_store", newOMap().
			set("active_manifest_urn", nil).
			set("instance_id", nil).
			set("claim_generator", nil)).
		set("assertions", []any{}).
		set("signature", newOMap().
			set("algorithm", nil).
			set("signing_time", nil).
			set("certificate", certificate)).
		set("validation", newOMap().
			set("validation_state", "Unknown").
			set("reasons", []any{}).
			set("details_raw", nil))

	return newOMap().set("c2pa", c2pa)
}

// nullIfEmpty maps the empty string to the absent value, which is how the report
// writes a field that was not found.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// addSystem records that a metadata system was found, without repeating one.
func (r *audioReport) addSystem(name string) {
	for _, s := range r.systems {
		if s == name {
			return
		}
	}
	r.systems = append(r.systems, name)
	r.detections.set("metadata_systems", r.systems)
}

// dropSystem takes back a metadata system that turned out not to be there.
func (r *audioReport) dropSystem(name string) {
	kept := make([]any, 0, len(r.systems))
	for _, s := range r.systems {
		if s != name {
			kept = append(kept, s)
		}
	}
	r.systems = kept
	r.detections.set("metadata_systems", r.systems)
}

// addEmbedded records a payload carried inside the file.
func (r *audioReport) addEmbedded(entry *omap) {
	r.embedded = append(r.embedded, entry)
	r.root.set("embedded", r.embedded)
}

// countEmbedded returns how many recorded payloads satisfy match, which is how
// the identifiers given to them are numbered.
func (r *audioReport) countEmbedded(match func(*omap) bool) int {
	n := 0
	for _, e := range r.embedded {
		if entry, ok := e.(*omap); ok && match(entry) {
			n++
		}
	}
	return n
}

// addError records something that went wrong while reading.
func (r *audioReport) addError(stage, message string) {
	r.errs = append(r.errs, newOMap().set("stage", stage).set("message", message))
	r.root.set("errors", r.errs)
}

// setCommonIfEmpty fills in one of the common tags, leaving a value that is
// already there alone.
func (r *audioReport) setCommonIfEmpty(field, value string) {
	r.setCommonIfEmptyAny(field, value)
}

// setCommonIfEmptyAny is setCommonIfEmpty for a value that need not be text. An
// ASF descriptor declares its own kind, and a tag taken from one keeps it, so a
// track number written as a number is reported as a number.
func (r *audioReport) setCommonIfEmptyAny(field string, value any) {
	if field == "" || !audioTruthy(value) {
		return
	}
	if existing, ok := r.common.vals[field]; ok && existing != nil {
		return
	}
	r.common.set(field, value)
}

// audioTruthy reports whether a value counts as present. The empty string, zero
// and false do not, matching the test the JavaScript makes before mapping a
// descriptor onto a common tag.
func audioTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case int:
		return v != 0
	case bool:
		return v
	default:
		return true
	}
}

// setContainerType changes the recorded container format, which the WAV reader
// does when it finds the chunk marking the larger variant of the format.
func (r *audioReport) setContainerType(typ string) { r.container.set("type", typ) }

// geobIdentifier names a payload found in an ID3 GEOB frame.
func geobIdentifier(n int) string { return fmt.Sprintf("geob_%d", n) }

func init() { core.Register(ExtractAudioMetadata{}) }

// provenanceC2PA returns the section describing content credentials, which the
// MP3 reader writes to when it finds a manifest carried in a frame.
func (r *audioReport) provenanceC2PA() *omap {
	provenance, _ := r.root.vals["provenance"].(*omap)
	c2pa, _ := provenance.vals["c2pa"].(*omap)
	return c2pa
}
