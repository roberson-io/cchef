package ops

import (
	"errors"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/audio"
	"github.com/roberson-io/cchef/internal/jsonval"
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
			Name: "Max embedded text bytes (iXML/axml/etc)", Flag: "max-embedded-text-bytes",
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

	container := audio.SniffContainer(data)
	report := audio.NewReport(filename, len(data), container)
	audio.Parse(data, report, container.Typ, maxText)

	encoded, err := jsonval.MarshalOMap(report.Root)
	if err != nil {
		return nil, err
	}
	return core.NewDish(encoded, core.TypeJSON), nil
}

// isNaNOrInf reports whether f is one of the values JavaScript's isFinite turns
// away, which the argument is checked for before being used as a limit.
func isNaNOrInf(f float64) bool { return f != f || f > 1e308 || f < -1e308 }

func init() { core.Register(ExtractAudioMetadata{}) }
