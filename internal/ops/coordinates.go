package ops

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/klaus-tockloth/coco"
	"github.com/mmcloughlin/geohash"
	"github.com/wroge/wgs84"
)

// Coordinate conversion constants.
const (
	minutesPerDegree   = 60     // arc-minutes in a degree
	secondsPerDegree   = 3600   // arc-seconds in a degree
	osgbGridSquareSize = 100000 // size of an OS National Grid 100 km square, in metres
	maxGridPrecision   = 10     // maximum digits for MGRS / OS National Grid output
)

// coordFormats mirrors CyberChef's FORMATS (ConvertCoordinates.mjs).
var coordFormats = []string{
	"Degrees Minutes Seconds",
	"Degrees Decimal Minutes",
	"Decimal Degrees",
	"Geohash",
	"Military Grid Reference System",
	"Ordnance Survey National Grid",
	"Universal Transverse Mercator",
}

// coordNoChange are formats passed to the conversion module as-is.
var coordNoChange = map[string]bool{
	"Geohash":                        true,
	"Military Grid Reference System": true,
	"Ordnance Survey National Grid":  true,
	"Universal Transverse Mercator":  true,
}

var (
	reCoordMGRS    = regexp.MustCompile(`^[0-9]{2}\s?[C-HJ-NP-X]{1}\s?[A-HJ-NP-Z][A-HJ-NP-V]\s?[0-9\s]+`)
	reCoordOSNG    = regexp.MustCompile(`^[A-HJ-Z]{2}\s+[0-9\s]+$`)
	reCoordGeohash = regexp.MustCompile(`^[0123456789BCDEFGHJKMNPQRSTUVWXYZ]+$`)
	reCoordUTM     = regexp.MustCompile(`^[0-9]{2}\s?[C-HJ-NP-X]\s[0-9.]+\s?[0-9.]+$`)
	reCoordDeg     = regexp.MustCompile(`[°'"]`)
	reCoordDirs    = regexp.MustCompile(`[NESW]`)
	reCoordDirSpl  = regexp.MustCompile(`[NnEeSsWw]`)
	reCoordWS      = regexp.MustCompile(`\s+`)
	reCoordSym     = regexp.MustCompile(`[°˝´'"]`)
	reCoordNonNum  = regexp.MustCompile(`[^0-9.-]`)
	reCoordNonAN   = regexp.MustCompile(`[^A-Za-z0-9]`)
)

// jsMathRound rounds half towards +Infinity, like JavaScript's Math.round.
func jsMathRound(x float64) float64 { return math.Floor(x + 0.5) }

// coordRound rounds to precision decimal places, matching the lib's round().
func coordRound(input float64, precision int) float64 {
	p := math.Pow(10, float64(precision))
	return jsMathRound(input*p) / p
}

func isNegativeZero(x float64) bool { return x == 0 && math.Signbit(x) }

// splitInput splits on whitespace and parses each numeric chunk to a float.
func splitInput(input string) []float64 {
	var out []float64
	for _, item := range reCoordWS.Split(input, -1) {
		item = reCoordNonNum.ReplaceAllString(item, "")
		if len(item) > 0 {
			if f, err := strconv.ParseFloat(item, 64); err == nil {
				out = append(out, f)
			}
		}
	}
	return out
}

func convDMSToDD(degrees, minutes, seconds float64) float64 {
	conv := math.Abs(degrees) + minutes/minutesPerDegree + seconds/secondsPerDegree
	if isNegativeZero(degrees) || degrees < 0 {
		conv = -conv
	}
	return conv
}

func convDDMToDD(degrees, minutes float64) float64 {
	conv := math.Abs(degrees) + minutes/minutesPerDegree
	if isNegativeZero(degrees) || degrees < 0 {
		conv = -conv
	}
	return conv
}

func convDDToDD(degrees float64, precision int) string {
	return jsNum(coordRound(degrees, precision)) + "°"
}

func convDDToDMS(decDegrees float64, precision int) string {
	absDegrees := math.Abs(decDegrees)
	degrees := math.Floor(absDegrees)
	minutes := math.Floor(minutesPerDegree * (absDegrees - degrees))
	seconds := coordRound(secondsPerDegree*(absDegrees-degrees)-minutesPerDegree*minutes, precision)
	out := jsNum(degrees) + "° " + jsNum(minutes) + "' " + jsNum(seconds) + "\""
	if isNegativeZero(decDegrees) || decDegrees < 0 {
		out = "-" + out
	}
	return out
}

func convDDToDDM(decDegrees float64, precision int) string {
	absDegrees := math.Abs(decDegrees)
	degrees := math.Floor(absDegrees)
	decMinutes := coordRound((absDegrees-degrees)*minutesPerDegree, precision)
	out := jsNum(degrees) + "° " + jsNum(decMinutes) + "'"
	if decDegrees < 0 || isNegativeZero(decDegrees) {
		out = "-" + out
	}
	return out
}

// findDirs finds the compass directions of an input, ported from ConvertCoordinates.mjs.
func findDirs(input, delim string) (string, string) {
	upper := strings.ToUpper(input)
	if dirs := reCoordDirs.FindAllString(upper, -1); dirs != nil {
		if len(dirs) <= 2 && len(dirs) >= 1 {
			if len(dirs) == 2 {
				return dirs[0], dirs[1]
			}
			return dirs[0], ""
		}
	}
	lat, long := splitLatLong(upper, delim)
	return dirFromValue(lat, "S", "N"), dirFromValue(long, "W", "E")
}

// splitLatLong splits the (uppercased) input into its latitude and longitude
// substrings, using the delimiter, or the direction markers for a
// "Direction"-style delimiter.
func splitLatLong(upper, delim string) (lat, long string) {
	lat = upper
	if !strings.Contains(delim, "Direction") {
		if strings.Contains(upper, delim) {
			split := strings.Split(upper, delim)
			if len(split) >= 1 {
				if split[0] != "" {
					lat = split[0]
				}
				if len(split) >= 2 && split[1] != "" {
					long = split[1]
				}
			}
		}
		return lat, long
	}
	split := reCoordDirSpl.Split(upper, -1)
	if len(split) > 1 {
		if split[0] == "" {
			lat = split[1]
		} else {
			lat = split[0]
		}
		if len(split) > 2 && split[2] != "" {
			long = split[2]
		}
	}
	return lat, long
}

// dirFromValue returns neg when s parses to a negative number, pos otherwise;
// an empty string yields "".
func dirFromValue(s, neg, pos string) string {
	if s == "" {
		return ""
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil && f < 0 {
		return neg
	}
	return pos
}

// findDelim auto-detects the input delimiter. Returns "" if none found.
func findDelim(input string) string {
	input = strings.TrimSpace(input)
	if testDir := reCoordDirSpl.FindAllString(input, -1); len(testDir) > 0 && len(testDir) < 3 {
		split := reCoordDirSpl.Split(input, -1)
		if len(split) <= 3 && len(split) > 0 {
			if split[0] == "" {
				return "Direction Preceding"
			} else if split[len(split)-1] == "" {
				return "Direction Following"
			}
		}
	}
	for _, delim := range []string{",", ";", ":"} {
		if strings.Contains(input, delim) {
			split := strings.Split(input, delim)
			if len(split) <= 3 && len(split) > 0 {
				return delim
			}
		}
	}
	return ""
}

// findFormat auto-detects the input format. Returns "" if none found.
func findFormat(input, delim string) string {
	input = strings.TrimSpace(input)
	testData, hasTest := findTestData(input, delim)

	if grid := detectGridFormat(input, delim); grid != "" {
		return grid
	}
	if hasTest {
		return detectDegreeFormat(testData)
	}
	return ""
}

// firstNonEmpty returns the first of split[0]/split[1] that is non-empty (used
// to pick the latitude token when a leading direction marker leaves split[0]
// empty).
func firstNonEmpty(split []string) string {
	if split[0] == "" {
		return split[1]
	}
	return split[0]
}

// findTestData extracts the first coordinate token used to classify the degree
// format, and whether a token was found.
func findTestData(input, delim string) (string, bool) {
	if delim == "" {
		return "", false
	}
	if strings.Contains(delim, "Direction") {
		if split := reCoordDirSpl.Split(input, -1); len(split) > 1 {
			return firstNonEmpty(split), true
		}
		return "", false
	}
	if !strings.Contains(input, delim) {
		return input, true
	}
	if split := strings.Split(input, delim); len(split) > 1 {
		return firstNonEmpty(split), true
	}
	return "", false
}

// detectGridFormat classifies grid-style coordinates (UTM/MGRS/OSNG/Geohash) by
// regex, returning "" for degree-style input (which contains °/'/") or no match.
func detectGridFormat(input, delim string) string {
	if reCoordDeg.MatchString(input) {
		return ""
	}
	filtered := strings.Replace(strings.ToUpper(input), delim, "", 1)
	switch {
	case reCoordUTM.MatchString(filtered):
		return "Universal Transverse Mercator"
	case reCoordMGRS.MatchString(filtered):
		return "Military Grid Reference System"
	case reCoordOSNG.MatchString(filtered):
		return "Ordnance Survey National Grid"
	case reCoordGeohash.MatchString(filtered):
		return "Geohash"
	}
	return ""
}

// detectDegreeFormat classifies a degree coordinate by how many numeric
// components its test token has (3=DMS, 2=DDM, 1=DD).
func detectDegreeFormat(testData string) string {
	switch len(splitInput(testData)) {
	case 3:
		return "Degrees Minutes Seconds"
	case 2:
		return "Degrees Decimal Minutes"
	case 1:
		return "Decimal Degrees"
	}
	return ""
}

// realDelim maps a delimiter name to its character.
func realDelim(delim string) string {
	return map[string]string{
		"Auto": "Auto", "Space": " ", "\\n": "\n",
		"Comma": ",", "Semi-colon": ";", "Colon": ":",
	}[delim]
}

// osgbToGrid formats OSGB easting/northing as a grid reference (e.g. "TQ 30028 80380").
func osgbToGrid(e, n float64, digits int) string {
	e100k := math.Floor(e / osgbGridSquareSize)
	n100k := math.Floor(n / osgbGridSquareSize)
	if e100k < 0 || e100k > 6 || n100k < 0 || n100k > 12 {
		return ""
	}
	l1 := (19 - int(n100k)) - (19-int(n100k))%5 + int(math.Floor((e100k+10)/5))
	l2 := (19-int(n100k))*5%25 + int(e100k)%5
	if l1 > 7 {
		l1++
	}
	if l2 > 7 {
		l2++
	}
	letters := string(rune('A'+l1)) + string(rune('A'+l2)) // #nosec G115 -- grid letter index is small and bounded
	d := digits / 2
	scale := math.Pow(10, float64(5-d))
	em := int(math.Floor(math.Mod(e, osgbGridSquareSize) / scale))
	nm := int(math.Floor(math.Mod(n, osgbGridSquareSize) / scale))
	return fmt.Sprintf("%s %0*d %0*d", letters, d, em, d, nm)
}

// osgbParse parses an OSGB grid reference into easting/northing.
func osgbParse(ref string) (float64, float64, bool) {
	ref = strings.ToUpper(reCoordNonAN.ReplaceAllString(ref, ""))
	if len(ref) < 2 {
		return 0, 0, false
	}
	l1 := int(ref[0] - 'A')
	l2 := int(ref[1] - 'A')
	if l1 > 7 {
		l1--
	}
	if l2 > 7 {
		l2--
	}
	e100k := ((l1-2)%5)*5 + (l2 % 5)
	n100k := (19 - (l1/5)*5) - l2/5
	en := ref[2:]
	if len(en)%2 != 0 {
		return 0, 0, false
	}
	half := len(en) / 2
	eStr := (en[:half] + "00000")[:5]
	nStr := (en[half:] + "00000")[:5]
	ev, _ := strconv.Atoi(eStr)
	nv, _ := strconv.Atoi(nStr)
	return float64(e100k*osgbGridSquareSize + ev), float64(n100k*osgbGridSquareSize + nv), true
}

// fmtMGRS reformats a coco MGRS string (e.g. "30UXC9931610163" -> "30U XC 99316 10163").
func fmtMGRS(m string) string {
	i := 0
	for i < len(m) && m[i] >= '0' && m[i] <= '9' {
		i++
	}
	if i+3 > len(m) {
		return m
	}
	zone, band, sq, rest := m[:i], string(m[i]), m[i+1:i+3], m[i+3:]
	h := len(rest) / 2
	return fmt.Sprintf("%s%s %s %s %s", zone, band, sq, rest[:h], rest[h:])
}

// convertCoordinates converts a coordinate string between formats.
// Ported from CyberChef ConvertCoordinates.mjs (geodesy/ngeohash replaced by Go libs).
func convertCoordinates(input, inFormat, inDelim, outFormat, outDelim string, includeDir string, precision int) (string, error) {
	if precision < 0 {
		precision = 0
	}

	if inDelim == "Auto" {
		inDelim = findDelim(input)
		if inDelim == "" {
			return "", fmt.Errorf("unable to detect the input delimiter automatically")
		}
	} else if !strings.Contains(inDelim, "Direction") {
		inDelim = realDelim(inDelim)
	}

	if inFormat == "Auto" {
		inFormat = findFormat(input, inDelim)
		if inFormat == "" {
			return "", fmt.Errorf("unable to detect the input format automatically")
		}
	}

	outDelim = realDelim(outDelim)

	input, split, isPair := tokeniseCoordInput(input, inFormat, inDelim)

	lat, lon, err := parseCoordinateInput(inFormat, input, split, isPair)
	if err != nil {
		return "", err
	}

	lat, lon = applyInputDirections(inFormat, input, lat, lon)
	latDir, longDir := findDirs(jsNum(lat)+","+jsNum(lon), ",")

	convLat, convLon, err := formatCoordinateOutput(outFormat, lat, lon, precision)
	if err != nil {
		return "", err
	}
	// convLat is empty only for a zero-precision Geohash.
	if convLat == "" {
		return "", fmt.Errorf("error converting co-ordinates")
	}

	if strings.Contains(outFormat, "Degrees") {
		return assembleDirectionalOutput(convLat, convLon, latDir, longDir, includeDir, outDelim, isPair), nil
	}
	return convLat + outDelim, nil
}

// applyInputDirections negates lat/lon for S/W direction markers found in a
// degree-based input, preserving CyberChef's faithful, quirky precedence.
func applyInputDirections(inFormat, input string, lat, lon float64) (float64, float64) {
	if !strings.Contains(inFormat, "Degrees") {
		return lat, lon
	}
	dirs := reCoordDirs.FindAllString(strings.ToUpper(input), -1)
	if len(dirs) == 0 {
		return lat, lon
	}
	if dirs[0] == "S" || (dirs[0] == "W" && lat > 0) {
		lat = -lat
	}
	if len(dirs) >= 2 && (dirs[1] == "S" || (dirs[1] == "W" && lon > 0)) {
		lon = -lon
	}
	return lat, lon
}

// assembleDirectionalOutput builds the final string for a degree-based output
// format: it strips the sign when a direction marker is shown and places the
// marker before/after each coordinate per includeDir.
func assembleDirectionalOutput(convLat, convLon, latDir, longDir, includeDir, outDelim string, isPair bool) string {
	if latDir == "S" && includeDir != "None" {
		convLat = strings.Replace(convLat, "-", "", 1)
	}
	if longDir == "W" && includeDir != "None" {
		convLon = strings.Replace(convLon, "-", "", 1)
	}

	writeCoord := func(out *strings.Builder, value, dir string) {
		if includeDir == "Before" {
			out.WriteString(dir + " ")
		}
		out.WriteString(value)
		if includeDir == "After" {
			out.WriteString(" " + dir)
		}
		out.WriteString(outDelim)
	}

	var out strings.Builder
	writeCoord(&out, convLat, latDir)
	if isPair {
		writeCoord(&out, convLon, longDir)
	}
	return out.String()
}

// tokeniseCoordInput splits the input into per-coordinate tokens for the
// degree-based formats (normalising symbols to spaces) and reports whether the
// input holds a lat/lon pair. Grid-style formats (coordNoChange) are not split;
// their delimiter is stripped and the whole string is treated as one pair value.
// The (possibly delimiter-stripped) input is returned for the caller to reuse.
func tokeniseCoordInput(input, inFormat, inDelim string) (string, []string, bool) {
	if coordNoChange[inFormat] {
		return strings.Replace(input, inDelim, "", 1), nil, true
	}

	var split []string
	if strings.Contains(inDelim, "Direction") {
		split = reCoordDirSpl.Split(input, -1)
		if len(split) > 0 && split[0] == "" {
			split = split[1:]
		}
	} else {
		split = strings.Split(input, inDelim)
	}
	for i := range split {
		split[i] = reCoordSym.ReplaceAllString(split[i], " ")
	}
	return input, split, len(split) > 1
}

// parseCoordinateInput converts a coordinate string in the given format to
// decimal-degree latitude and longitude. split/isPair carry the pre-tokenised
// input for the degree-based formats.
func parseCoordinateInput(inFormat, input string, split []string, isPair bool) (lat, lon float64, err error) {
	switch inFormat {
	case "Geohash":
		lat, lon = geohash.DecodeCenter(reCoordNonAN.ReplaceAllString(input, ""))
	case "Military Grid Reference System":
		ll, _, err := coco.MGRS(reCoordNonAN.ReplaceAllString(input, "")).ToLL()
		if err != nil {
			return 0, 0, fmt.Errorf("invalid MGRS reference: %w", err)
		}
		lat, lon = ll.Lat, ll.Lon
	case "Ordnance Survey National Grid":
		e, n, ok := osgbParse(input)
		if !ok {
			return 0, 0, fmt.Errorf("invalid Ordnance Survey National Grid reference")
		}
		l, la, _ := wgs84.From(wgs84.OSGB36NationalGrid())(e, n, 0)
		lat, lon = la, l
	case "Universal Transverse Mercator":
		return utmParse(input)
	case "Degrees Minutes Seconds":
		return parseDegreeInput(inFormat, split, input, isPair, 3, func(f []float64) float64 {
			return convDMSToDD(f[0], f[1], f[2])
		})
	case "Degrees Decimal Minutes":
		return parseDegreeInput(inFormat, split, input, isPair, 2, func(f []float64) float64 {
			return convDDMToDD(f[0], f[1])
		})
	case "Decimal Degrees":
		return parseDegreeInput(inFormat, split, input, isPair, 1, func(f []float64) float64 {
			return f[0]
		})
	default:
		return 0, 0, fmt.Errorf("unknown input format '%s'", inFormat)
	}
	return lat, lon, nil
}

// parseDegreeInput handles the three degree-based input formats, which share the
// same pair/single structure: each coordinate is want fields long and folded to
// decimal degrees by conv. Degrees Minutes Seconds accepts extra fields (its
// original check is >=3, not ==3), so it passes want=3 with a >= comparison.
func parseDegreeInput(inFormat string, split []string, input string, isPair bool, want int, conv func([]float64) float64) (lat, lon float64, err error) {
	// Degrees Minutes Seconds keeps its historical ">= 3 fields" leniency; the
	// other two require an exact field count.
	enough := func(n int) bool {
		if want == 3 {
			return n >= want
		}
		return n == want
	}
	badFormat := fmt.Errorf("invalid co-ordinate format for %s", inFormat)

	if isPair {
		sl, so := splitInput(split[0]), splitInput(split[1])
		if !enough(len(sl)) || !enough(len(so)) {
			return 0, 0, badFormat
		}
		return conv(sl), conv(so), nil
	}
	// Single value: Degrees Decimal Minutes historically re-split the raw input
	// rather than split[0]; the others use split[0].
	src := split[0]
	if want == 2 {
		src = input
	}
	sl := splitInput(src)
	if !enough(len(sl)) {
		return 0, 0, badFormat
	}
	return conv(sl), conv(sl), nil
}

// formatCoordinateOutput renders decimal-degree lat/lon into the requested
// output format at the given precision. convLon is empty for single-value
// formats (Geohash, MGRS, OSGB, UTM).
func formatCoordinateOutput(outFormat string, lat, lon float64, precision int) (convLat, convLon string, err error) {
	switch outFormat {
	case "Decimal Degrees":
		return convDDToDD(lat, precision), convDDToDD(lon, precision), nil
	case "Degrees Decimal Minutes":
		return convDDToDDM(lat, precision), convDDToDDM(lon, precision), nil
	case "Degrees Minutes Seconds":
		return convDDToDMS(lat, precision), convDDToDMS(lon, precision), nil
	case "Geohash":
		return geohash.EncodeWithPrecision(lat, lon, uint(precision)), "", nil
	case "Military Grid Reference System":
		p := clampGridPrecision(precision)
		acc := int(math.Pow(10, float64(5-p/2)))
		m, err := coco.LL{Lat: lat, Lon: lon}.ToMGRS(acc)
		if err != nil {
			return "", "", fmt.Errorf("could not convert co-ordinates to MGRS: %w", err)
		}
		return fmtMGRS(string(m)), "", nil
	case "Ordnance Survey National Grid":
		e, n, _ := wgs84.To(wgs84.OSGB36NationalGrid())(lon, lat, 0)
		grid := osgbToGrid(e, n, clampGridPrecision(precision))
		if grid == "" {
			return "", "", fmt.Errorf("could not convert co-ordinates to OS National Grid. Are the co-ordinates in range?")
		}
		return grid, "", nil
	case "Universal Transverse Mercator":
		e, n, zone, err := utmFromLatLon(lat, lon)
		if err != nil {
			return "", "", fmt.Errorf("could not convert co-ordinates to UTM: %w", err)
		}
		hemi := "N"
		if lat < 0 {
			hemi = "S"
		}
		return fmt.Sprintf("%02d %s %s %s", zone, hemi,
			strconv.FormatFloat(e, 'f', precision, 64), strconv.FormatFloat(n, 'f', precision, 64)), "", nil
	default:
		// outFormat is validated against coordFormats by the arg layer, so every
		// format has a case above; reaching here means an unvalidated value slipped
		// through, which is a programming error.
		panic(fmt.Sprintf("formatCoordinateOutput: unhandled output format %q", outFormat))
	}
}

// clampGridPrecision rounds a precision up to the next even number and caps it at
// 10, as the MGRS and OS National Grid encoders require.
func clampGridPrecision(precision int) int {
	if precision%2 != 0 {
		precision++
	}
	if precision > maxGridPrecision {
		precision = maxGridPrecision
	}
	return precision
}

// utmParse parses a UTM string ("30 U 699316.234 5710163.758") to lat/lon.
func utmParse(input string) (float64, float64, error) {
	if regexp.MustCompile(`^[\d]{2}[A-Za-z]`).MatchString(input) {
		input = input[:2] + " " + input[2:]
	}
	fields := strings.Fields(input)
	if len(fields) < 4 {
		return 0, 0, fmt.Errorf("invalid UTM co-ordinate")
	}
	zone, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UTM zone")
	}
	// geodesy's Utm.parse expects an N/S hemisphere, not the MGRS band letter.
	hemi := strings.ToUpper(fields[1])
	if hemi != "N" && hemi != "S" {
		return 0, 0, fmt.Errorf("invalid UTM hemisphere %s", fields[1])
	}
	e, err1 := strconv.ParseFloat(fields[2], 64)
	n, err2 := strconv.ParseFloat(fields[3], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("invalid UTM easting/northing")
	}
	lat, lon, err := utmToLatLon(e, n, zone, hemi == "N")
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UTM co-ordinate: %w", err)
	}
	return lat, lon, nil
}
