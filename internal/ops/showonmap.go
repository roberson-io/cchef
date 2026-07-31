package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ShowOnMap{})
}

var (
	reMapBlank   = regexp.MustCompile(`\s+`)
	reMapTrailer = regexp.MustCompile(`,$`)
)

// ShowOnMap converts coordinates to a latitude/longitude pair for display on a map.
type ShowOnMap struct{}

// Meta returns the operation metadata.
func (ShowOnMap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Show on map",
		Module:      "Hashing",
		Description: "Displays co-ordinates on a slippy map. In CyberChef the GUI renders an interactive map; here the operation returns the parsed 'latitude,longitude' pair.",
		InfoURL:     "https://wikipedia.org/wiki/Geographic_coordinate_system",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ShowOnMap) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Zoom Level", Type: core.ArgNumber, Integer: true, Value: 13},
		{Name: "Input Format", Type: core.ArgOption, Value: append([]string{"Auto"}, coordFormats...)},
		{Name: "Input Delimiter", Type: core.ArgOption, Value: []string{
			"Auto", "Direction Preceding", "Direction Following", "\\n", "Comma", "Semi-colon", "Colon",
		}},
	}
}

// Run parses the coordinates. Ported from CyberChef ShowOnMap.mjs (run, not the GUI present).
func (ShowOnMap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if reMapBlank.ReplaceAllString(input, "") == "" {
		return core.NewDish([]byte(input), core.TypeString), nil
	}
	latLong, err := convertCoordinates(input, args[1].(string), args[2].(string), "Decimal Degrees", "Comma", "None", 5)
	if err != nil {
		return nil, err
	}
	latLong = reMapTrailer.ReplaceAllString(latLong, "")
	latLong = strings.ReplaceAll(latLong, "°", "")

	coords := strings.Split(latLong, ",")
	if len(coords) != 2 {
		return nil, mapCoordErr(latLong)
	}
	for _, c := range coords {
		c = strings.TrimSpace(c)
		if _, err := strconv.ParseFloat(c, 64); c == "" || err != nil {
			return nil, mapCoordErr(latLong)
		}
	}
	return core.NewDish([]byte(latLong), core.TypeString), nil
}

func mapCoordErr(latLong string) error {
	return fmt.Errorf("could not show coordinates '%s' on the map. Expected a latitude and longitude pair - check that the input format and delimiter are correct", latLong)
}
