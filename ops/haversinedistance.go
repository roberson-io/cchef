package ops

import (
	"errors"
	"math"
	"regexp"
	"strconv"

	"github.com/roberson-io/cchef/core"
)

// haversinePositions is the shape the input must take: four numbers separated
// by commas, each of which may carry a sign and a fractional part, and each
// comma optionally followed by one space. Nothing else is allowed, not even
// space at either end.
var haversinePositions = regexp.MustCompile(
	`^(-?\d+(\.\d+)?), ?(-?\d+(\.\d+)?), ?(-?\d+(\.\d+)?), ?(-?\d+(\.\d+)?)$`)

// The groups of that shape holding the four numbers.
const (
	haversineLat1 = 1
	haversineLng1 = 3
	haversineLat2 = 5
	haversineLng2 = 7
)

// earthRadiusMetres is the mean radius the distance is worked out on.
const earthRadiusMetres = 6371000

// errHaversineFormat is what input of any other shape gets.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errHaversineFormat = errors.New("Input must in the format lat1, lng1, lat2, lng2")

// HaversineDistance measures the distance between two points on the Earth.
type HaversineDistance struct{}

// Meta returns the operation metadata.
func (HaversineDistance) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Haversine distance",
		Module: "Default",
		Description: "Returns the distance between two pairs of GPS latitude and " +
			"longitude co-ordinates in metres.<br><br>e.g. <code>51.487263,-0.124323, " +
			"38.9517,-77.1467</code>",
		InfoURL:    "https://wikipedia.org/wiki/Haversine_formula",
		InputType:  core.TypeString,
		OutputType: core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (HaversineDistance) Args() []core.ArgDef { return nil }

// Run measures the distance.
func (HaversineDistance) Run(in *core.Dish, args []any) (*core.Dish, error) {
	found := haversinePositions.FindStringSubmatch(in.String())
	if found == nil {
		return nil, errHaversineFormat
	}

	// The shape has already been checked, so each of the four reads as a number.
	lat1, _ := strconv.ParseFloat(found[haversineLat1], 64)
	lng1, _ := strconv.ParseFloat(found[haversineLng1], 64)
	lat2, _ := strconv.ParseFloat(found[haversineLat2], 64)
	lng2, _ := strconv.ParseFloat(found[haversineLng2], 64)

	metres := haversineMetres(lat1, lng1, lat2, lng2)
	return core.NewDish([]byte(jsNum(metres)), core.TypeNumber), nil
}

// haversineMetres is the haversine formula: the square of half the chord
// between the two points, turned back into the angle between them and then into
// a distance along the surface.
func haversineMetres(lat1, lng1, lat2, lng2 float64) float64 {
	const toRadians = math.Pi / 180

	dLat := (lat2 - lat1) * toRadians
	dLng := (lng2 - lng1) * toRadians

	// Every product is rounded on its own before the two halves are added. Go
	// may otherwise fuse a multiply and an add into one operation that rounds
	// once, which lands a place or two away from the value JavaScript works
	// out, and the second half is a chain of three multiplications that has to
	// be rounded at each step.
	halfLat := math.Sin(dLat / 2)
	halfLng := math.Sin(dLng / 2)

	between := float64(math.Cos(lat1*toRadians) * math.Cos(lat2*toRadians))
	between = float64(between * halfLng)
	between = float64(between * halfLng)

	chord := float64(halfLat*halfLat) + between

	return earthRadiusMetres * 2 * math.Atan2(math.Sqrt(chord), math.Sqrt(1-chord))
}

func init() { core.Register(HaversineDistance{}) }
