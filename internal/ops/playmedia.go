package ops

import (
	"errors"
	"regexp"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(PlayMedia{})
}

// reAudioVideo matches audio/* and video/* mime types (CyberChef's /^(audio|video)/).
var reAudioVideo = regexp.MustCompile(`^(audio|video)`)

// PlayMedia validates that the input is an audio or video file and passes the
// bytes through. Ported from CyberChef PlayMedia.mjs, whose browser <audio>/
// <video> presentation is dropped; cchef offers Raw or base64 data-URI output.
type PlayMedia struct{}

// Meta returns the operation metadata.
func (PlayMedia) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Play Media",
		Module:      "Default",
		Description: "Validates that the input is audio or video and outputs it. Tags: sound, movie, mp3, mp4, mov, webm, wav, ogg.",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (PlayMedia) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Raw", "Base64", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Base64"}},
	}
}

// Run validates the media and renders it in the requested output form.
func (PlayMedia) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	if len(in.Bytes()) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	data := decodeImageInput(in, inputFormat)

	mime := isTypeMatch(reAudioVideo, data)
	if mime == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid or unrecognised file type")
	}

	return renderMedia(data, mime, outputFormat)
}
