package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ConvertCoordinateFormat{})
}

var reCoordBlank = regexp.MustCompile(`[\s+]`)

// ConvertCoordinateFormat converts geographic coordinates between formats.
type ConvertCoordinateFormat struct{}

// Meta returns the operation metadata.
func (ConvertCoordinateFormat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Convert co-ordinate format",
		Module: "Hashing",
		Description: "Converts geographical coordinates between different formats.\n\n" +
			"Note: UTM and Ordnance Survey National Grid use different projection implementations than CyberChef, " +
			"so results may differ slightly at high precision (UTM by a sub-millimetre digit; OSGB-as-input by a few metres). " +
			"MGRS, Geohash and the lat/lon formats match exactly.",
		InfoURL:    "https://wikipedia.org/wiki/Geographic_coordinate_conversion",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ConvertCoordinateFormat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input Format", Type: core.ArgOption, Value: append([]string{"Auto"}, coordFormats...)},
		{Name: "Input Delimiter", Type: core.ArgOption, Value: []string{
			"Auto", "Direction Preceding", "Direction Following", "\\n", "Comma", "Semi-colon", "Colon",
		}},
		{Name: "Output Format", Type: core.ArgOption, Value: coordFormats},
		{Name: "Output Delimiter", Type: core.ArgOption, Value: []string{
			"Space", "\\n", "Comma", "Semi-colon", "Colon",
		}},
		{Name: "Include Compass Directions", Type: core.ArgOption, Value: []string{"None", "Before", "After"}},
		{Name: "Precision", Type: core.ArgNumber, Integer: true, Value: 3},
	}
}

// Run converts the coordinates. Ported from CyberChef ConvertCoordinateFormat.mjs.
func (ConvertCoordinateFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if reCoordBlank.ReplaceAllString(input, "") == "" {
		return core.NewDish([]byte(input), core.TypeString), nil
	}
	out, err := convertCoordinates(input,
		args[0].(string), args[1].(string), args[2].(string), args[3].(string),
		args[4].(string), int(args[5].(float64)))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
