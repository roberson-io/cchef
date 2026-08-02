package ops

import (
	"encoding/binary"
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(RemoveEXIF{})
}

// errExifNotFound signals that no EXIF APP1 segment was present; CyberChef
// swallows this and returns the input unchanged. The text is never surfaced.
var errExifNotFound = errors.New("exif not found")

// RemoveEXIF removes EXIF metadata from a JPEG.
type RemoveEXIF struct{}

// Meta returns the operation metadata.
func (RemoveEXIF) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove EXIF",
		Module:      "Image",
		Description: "Removes EXIF data from a JPEG image.",
		InfoURL:     "https://wikipedia.org/wiki/Exif",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RemoveEXIF) Args() []core.ArgDef { return nil }

// Run removes EXIF data, or returns the input unchanged if none is found.
func (RemoveEXIF) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) == 0 {
		return core.NewDish(data, core.TypeByteArray), nil
	}

	out, err := removeEXIF(data)
	if err != nil {
		if errors.Is(err, errExifNotFound) {
			return core.NewDish(data, core.TypeByteArray), nil
		}
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Could not remove EXIF data from image: " + err.Error())
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// exifAPP1Prefix is the marker (FFE1) plus, at offset 4, "Exif\0\0".
var exifIdentifier = []byte("Exif\x00\x00")

// removeEXIF strips the APP1 EXIF segment from a JPEG byte stream.
func removeEXIF(jpeg []byte) ([]byte, error) {
	if len(jpeg) < 2 || jpeg[0] != 0xff || jpeg[1] != 0xd8 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Given data is not jpeg.")
	}

	segments, err := splitJPEGSegments(jpeg)
	if err != nil {
		return nil, err
	}

	switch {
	case len(segments) > 1 && isExifSegment(segments[1]):
		out := segments[0]
		for _, s := range segments[2:] {
			out = append(out, s...)
		}
		return out, nil
	case len(segments) > 2 && isExifSegment(segments[2]):
		out := append(append([]byte{}, segments[0]...), segments[1]...)
		for _, s := range segments[3:] {
			out = append(out, s...)
		}
		return out, nil
	default:
		return nil, errExifNotFound
	}
}

// isExifSegment reports whether seg is an APP1 (FFE1) segment carrying EXIF.
func isExifSegment(seg []byte) bool {
	return len(seg) >= 10 && seg[0] == 0xff && seg[1] == 0xe1 &&
		string(seg[4:10]) == string(exifIdentifier)
}

// splitJPEGSegments splits a JPEG into its SOI, marker segments and the final
// scan (from SOS onwards), mirroring piexifjs' splitIntoSegments.
func splitJPEGSegments(data []byte) ([][]byte, error) {
	segments := [][]byte{data[0:2]}
	head := 2
	for {
		if head+2 > len(data) {
			//nolint:staticcheck,revive // CyberChef's verbatim error text
			return nil, errors.New("Wrong JPEG data.")
		}
		if data[head] == 0xff && data[head+1] == 0xda { // Start of Scan
			segments = append(segments, data[head:])
			break
		}
		if head+4 > len(data) {
			//nolint:staticcheck,revive // CyberChef's verbatim error text
			return nil, errors.New("Wrong JPEG data.")
		}
		length := int(binary.BigEndian.Uint16(data[head+2 : head+4]))
		endPoint := min(head+length+2, len(data))
		segments = append(segments, data[head:endPoint])
		head = endPoint
		if head >= len(data) {
			//nolint:staticcheck,revive // CyberChef's verbatim error text
			return nil, errors.New("Wrong JPEG data.")
		}
	}
	return segments, nil
}
