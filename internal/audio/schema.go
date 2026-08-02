// Package audio reads metadata out of audio containers.
//
// [SniffContainer] works out the format from the file's first bytes, and
// [Parse] hands the data to that format's reader — MP3/ID3, FLAC, Ogg, WAVE,
// AIFF, AAC, AC-3, WMA/ASF, or MP4 — which fills in a [Report]: the common
// fields every format shares, the raw per-format tables, and any warnings the
// reader could not act on. Ported from CyberChef's lib/AudioParsers.mjs,
// AudioBytes.mjs and AudioMetaSchema.mjs.
package audio

import (
	"bytes"
	"fmt"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// Parse hands the file to the reader for its container, or notes that there
// is not one.
func Parse(b []byte, r *Report, container string, maxText int) {
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

// Container is what the opening bytes of a file say it is.
type Container struct {
	Typ   string
	mime  string
	brand string
}

// SniffContainer works out the container format from the file's first bytes.
// Some formats are named by a byte pattern at the very start, others by a
// four-character tag a little way in, so the two are looked for separately.
func SniffContainer(b []byte) Container {
	if c, ok := sniffByBytePattern(b); ok {
		return c
	}
	if c, ok := sniffByTag(b); ok {
		return c
	}
	return Container{Typ: "unknown"}
}

// audioMagics are the formats named by a fixed run of bytes at the very start.
// A format is only considered when the file is long enough to hold the header
// that follows the run, which is what minLength records.
var audioMagics = []struct {
	prefix    []byte
	minLength int
	container Container
}{
	{[]byte("ID3"), 3, Container{Typ: "mp3", mime: "audio/mpeg"}},
	{[]byte{0x0b, 0x77}, 8, Container{Typ: "ac3", mime: "audio/ac3"}},
	{
		[]byte{0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11},
		16,
		Container{Typ: "wma", mime: "audio/x-ms-wma"},
	},
}

// sniffByBytePattern names the formats whose first bytes are a fixed run, or the
// set bits that open an audio frame.
func sniffByBytePattern(b []byte) (Container, bool) {
	if c, ok := sniffFrameSync(b); ok {
		return c, true
	}
	for _, m := range audioMagics {
		if len(b) >= m.minLength && bytes.HasPrefix(b, m.prefix) {
			return m.container, true
		}
	}
	return Container{}, false
}

// sniffFrameSync names the two formats that open with a frame rather than a
// header. Eleven set bits start a frame of either; the layer bits say which.
func sniffFrameSync(b []byte) (Container, bool) {
	if len(b) < 2 || b[0] != 0xff || b[1]&0xe0 != 0xe0 {
		return Container{}, false
	}
	if b[1]&0x06 == 0x00 {
		return Container{Typ: "aac", mime: "audio/aac"}, true
	}
	return Container{Typ: "mp3", mime: "audio/mpeg"}, true
}

// sniffByTag names the formats identified by a four-character tag, either at the
// start of the file or just inside it.
func sniffByTag(b []byte) (Container, bool) {
	if len(b) >= 12 {
		if c, ok := sniffChunkedFamily(b); ok {
			return c, true
		}
	}
	if len(b) < 4 {
		return Container{}, false
	}
	switch ascii4(b, 0) {
	case "fLaC":
		return Container{Typ: "flac", mime: "audio/flac"}, true
	case "OggS":
		// An Ogg stream carries either Vorbis or Opus; only the Opus one names
		// itself in the first page.
		if indexOfASCII(b, "OpusHead", 0, min(len(b), 65536)) >= 0 {
			return Container{Typ: "opus", mime: "audio/ogg"}, true
		}
		return Container{Typ: "ogg", mime: "audio/ogg"}, true
	}
	return Container{}, false
}

// sniffChunkedFamily names the formats built out of chunks, which say what they
// hold in a second tag after the one that opens them.
func sniffChunkedFamily(b []byte) (Container, bool) {
	head, form := ascii4(b, 0), ascii4(b, 8)
	switch {
	case head == "RIFF" && form == "WAVE":
		return Container{Typ: "wav", mime: "audio/wav"}, true
	case head == "BW64" && form == "WAVE":
		return Container{Typ: "bw64", mime: "audio/wav"}, true
	case ascii4(b, 4) == "ftyp":
		return sniffMp4Brand(form), true
	case head == "FORM" && (form == "AIFF" || form == "AIFC"):
		return Container{Typ: "aiff", mime: "audio/aiff", brand: form}, true
	}
	return Container{}, false
}

// sniffMp4Brand tells an audio-only MP4 from the general one by its brand.
func sniffMp4Brand(brand string) Container {
	switch brand {
	case "M4A ", "M4B ", "M4P ":
		return Container{Typ: "m4a", mime: "audio/mp4", brand: brand}
	}
	return Container{Typ: "mp4", mime: "video/mp4", brand: brand}
}

// Report is the document the operation builds. The pieces a parser reaches
// into are held separately so that they can be added to without walking back
// down from the root, while the root keeps the order everything is written in.
type Report struct {
	Root     *jsonval.OMap
	common   *jsonval.OMap
	raw      *jsonval.OMap
	systems  []any
	embedded []any
	errs     []any

	detections *jsonval.OMap
	tags       *jsonval.OMap
	container  *jsonval.OMap
}

// NewReport builds the empty report a parser fills in.
func NewReport(filename string, byteLength int, c Container) *Report {
	r := &Report{
		systems:  []any{},
		embedded: []any{},
		errs:     []any{},
	}

	container := jsonval.NewOMap().
		Set("type", c.Typ).
		Set("brand", nullIfEmpty(c.brand)).
		Set("mime", nullIfEmpty(c.mime))

	artifact := jsonval.NewOMap().
		Set("filename", nullIfEmpty(filename)).
		Set("byte_length", byteLength).
		Set("container", container)

	r.detections = jsonval.NewOMap().
		Set("metadata_systems", r.systems).
		Set("provenance_systems", []any{})

	r.common = jsonval.NewOMap()
	for _, field := range audioCommonFields {
		r.common.Set(field, nil)
	}
	r.raw = jsonval.NewOMap()
	r.tags = jsonval.NewOMap().Set("common", r.common).Set("raw", r.raw)

	r.Root = jsonval.NewOMap().
		Set("schema_version", "audio-meta-1.0").
		Set("artifact", artifact).
		Set("detections", r.detections).
		Set("tags", r.tags).
		Set("embedded", r.embedded).
		Set("provenance", newAudioProvenance()).
		Set("errors", r.errs)

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
func newAudioProvenance() *jsonval.OMap {
	certificate := jsonval.NewOMap().
		Set("subject_cn", nil).
		Set("issuer_cn", nil).
		Set("serial_number", nil)

	c2pa := jsonval.NewOMap().
		Set("present", false).
		Set("embedding", []any{}).
		Set("manifest_store", jsonval.NewOMap().
			Set("active_manifest_urn", nil).
			Set("instance_id", nil).
			Set("claim_generator", nil)).
		Set("assertions", []any{}).
		Set("signature", jsonval.NewOMap().
			Set("algorithm", nil).
			Set("signing_time", nil).
			Set("certificate", certificate)).
		Set("validation", jsonval.NewOMap().
			Set("validation_state", "Unknown").
			Set("reasons", []any{}).
			Set("details_raw", nil))

	return jsonval.NewOMap().Set("c2pa", c2pa)
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
func (r *Report) addSystem(name string) {
	for _, s := range r.systems {
		if s == name {
			return
		}
	}
	r.systems = append(r.systems, name)
	r.detections.Set("metadata_systems", r.systems)
}

// dropSystem takes back a metadata system that turned out not to be there.
func (r *Report) dropSystem(name string) {
	kept := make([]any, 0, len(r.systems))
	for _, s := range r.systems {
		if s != name {
			kept = append(kept, s)
		}
	}
	r.systems = kept
	r.detections.Set("metadata_systems", r.systems)
}

// addEmbedded records a payload carried inside the file.
func (r *Report) addEmbedded(entry *jsonval.OMap) {
	r.embedded = append(r.embedded, entry)
	r.Root.Set("embedded", r.embedded)
}

// countEmbedded returns how many recorded payloads satisfy match, which is how
// the identifiers given to them are numbered.
func (r *Report) countEmbedded(match func(*jsonval.OMap) bool) int {
	n := 0
	for _, e := range r.embedded {
		if entry, ok := e.(*jsonval.OMap); ok && match(entry) {
			n++
		}
	}
	return n
}

// addError records something that went wrong while reading.
func (r *Report) addError(stage, message string) {
	r.errs = append(r.errs, jsonval.NewOMap().Set("stage", stage).Set("message", message))
	r.Root.Set("errors", r.errs)
}

// setCommonIfEmpty fills in one of the common tags, leaving a value that is
// already there alone.
func (r *Report) setCommonIfEmpty(field, value string) {
	r.setCommonIfEmptyAny(field, value)
}

// setCommonIfEmptyAny is setCommonIfEmpty for a value that need not be text. An
// ASF descriptor declares its own kind, and a tag taken from one keeps it, so a
// track number written as a number is reported as a number.
func (r *Report) setCommonIfEmptyAny(field string, value any) {
	if field == "" || !audioTruthy(value) {
		return
	}
	if existing, ok := r.common.Get(field); ok && existing != nil {
		return
	}
	r.common.Set(field, value)
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
func (r *Report) setContainerType(typ string) { r.container.Set("type", typ) }

// geobIdentifier names a payload found in an ID3 GEOB frame.
func geobIdentifier(n int) string { return fmt.Sprintf("geob_%d", n) }

// provenanceC2PA returns the section describing content credentials, which the
// MP3 reader writes to when it finds a manifest carried in a frame.
func (r *Report) provenanceC2PA() *jsonval.OMap {
	provenance, _ := r.Root.Value("provenance").(*jsonval.OMap)
	c2pa, _ := provenance.Value("c2pa").(*jsonval.OMap)
	return c2pa
}
