package ops

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/im7mortal/UTM"
	"github.com/klaus-tockloth/coco"
	"github.com/mmcloughlin/geohash"
	"github.com/wroge/wgs84"
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
	conv := math.Abs(degrees) + minutes/60 + seconds/3600
	if isNegativeZero(degrees) || degrees < 0 {
		conv = -conv
	}
	return conv
}

func convDDMToDD(degrees, minutes float64) float64 {
	conv := math.Abs(degrees) + minutes/60
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
	minutes := math.Floor(60 * (absDegrees - degrees))
	seconds := coordRound(3600*(absDegrees-degrees)-60*minutes, precision)
	out := jsNum(degrees) + "° " + jsNum(minutes) + "' " + jsNum(seconds) + "\""
	if isNegativeZero(decDegrees) || decDegrees < 0 {
		out = "-" + out
	}
	return out
}

func convDDToDDM(decDegrees float64, precision int) string {
	absDegrees := math.Abs(decDegrees)
	degrees := math.Floor(absDegrees)
	decMinutes := coordRound((absDegrees-degrees)*60, precision)
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
	lat, long, latDir, longDir := upper, "", "", ""
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
	} else {
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
	}
	if lat != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(lat), 64); err == nil && f < 0 {
			latDir = "S"
		} else {
			latDir = "N"
		}
	}
	if long != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(long), 64); err == nil && f < 0 {
			longDir = "W"
		} else {
			longDir = "E"
		}
	}
	return latDir, longDir
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
	var testData string
	hasTest := false
	input = strings.TrimSpace(input)

	if delim != "" && strings.Contains(delim, "Direction") {
		split := reCoordDirSpl.Split(input, -1)
		if len(split) > 1 {
			if split[0] == "" {
				testData = split[1]
			} else {
				testData = split[0]
			}
			hasTest = true
		}
	} else if delim != "" {
		if strings.Contains(input, delim) {
			split := strings.Split(input, delim)
			if len(split) > 1 {
				if split[0] == "" {
					testData = split[1]
				} else {
					testData = split[0]
				}
				hasTest = true
			}
		} else {
			testData = input
			hasTest = true
		}
	}

	if !reCoordDeg.MatchString(input) {
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
	}

	if hasTest {
		switch len(splitInput(testData)) {
		case 3:
			return "Degrees Minutes Seconds"
		case 2:
			return "Degrees Decimal Minutes"
		case 1:
			return "Decimal Degrees"
		}
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
	e100k := math.Floor(e / 100000)
	n100k := math.Floor(n / 100000)
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
	em := int(math.Floor(math.Mod(e, 100000) / scale))
	nm := int(math.Floor(math.Mod(n, 100000) / scale))
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
	return float64(e100k*100000 + ev), float64(n100k*100000 + nv), true
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

	var split []string
	isPair := false
	if !coordNoChange[inFormat] {
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
		if len(split) > 1 {
			isPair = true
		}
	} else {
		input = strings.Replace(input, inDelim, "", 1)
		isPair = true
	}

	var lat, lon float64
	switch inFormat {
	case "Geohash":
		lat, lon = geohash.DecodeCenter(reCoordNonAN.ReplaceAllString(input, ""))
	case "Military Grid Reference System":
		ll, _, err := coco.MGRS(reCoordNonAN.ReplaceAllString(input, "")).ToLL()
		if err != nil {
			return "", fmt.Errorf("invalid MGRS reference: %w", err)
		}
		lat, lon = ll.Lat, ll.Lon
	case "Ordnance Survey National Grid":
		e, n, ok := osgbParse(input)
		if !ok {
			return "", fmt.Errorf("invalid Ordnance Survey National Grid reference")
		}
		l, la, _ := wgs84.From(wgs84.OSGB36NationalGrid())(e, n, 0)
		lat, lon = la, l
	case "Universal Transverse Mercator":
		var err error
		lat, lon, err = utmParse(input)
		if err != nil {
			return "", err
		}
	case "Degrees Minutes Seconds":
		if isPair {
			sl, so := splitInput(split[0]), splitInput(split[1])
			if len(sl) < 3 || len(so) < 3 {
				return "", fmt.Errorf("invalid co-ordinate format for Degrees Minutes Seconds")
			}
			lat = convDMSToDD(sl[0], sl[1], sl[2])
			lon = convDMSToDD(so[0], so[1], so[2])
		} else {
			sl := splitInput(split[0])
			if len(sl) < 3 {
				return "", fmt.Errorf("invalid co-ordinate format for Degrees Minutes Seconds")
			}
			lat = convDMSToDD(sl[0], sl[1], sl[2])
			lon = lat
		}
	case "Degrees Decimal Minutes":
		if isPair {
			sl, so := splitInput(split[0]), splitInput(split[1])
			if len(sl) != 2 || len(so) != 2 {
				return "", fmt.Errorf("invalid co-ordinate format for Degrees Decimal Minutes")
			}
			lat = convDDMToDD(sl[0], sl[1])
			lon = convDDMToDD(so[0], so[1])
		} else {
			sl := splitInput(input)
			if len(sl) != 2 {
				return "", fmt.Errorf("invalid co-ordinate format for Degrees Decimal Minutes")
			}
			lat = convDDMToDD(sl[0], sl[1])
			lon = lat
		}
	case "Decimal Degrees":
		if isPair {
			sl, so := splitInput(split[0]), splitInput(split[1])
			if len(sl) != 1 || len(so) != 1 {
				return "", fmt.Errorf("invalid co-ordinate format for Decimal Degrees")
			}
			lat, lon = sl[0], so[0]
		} else {
			sl := splitInput(split[0])
			if len(sl) != 1 {
				return "", fmt.Errorf("invalid co-ordinate format for Decimal Degrees")
			}
			lat, lon = sl[0], sl[0]
		}
	default:
		return "", fmt.Errorf("unknown input format '%s'", inFormat)
	}

	// Negate lat/lon for S/W directions (CyberChef's faithful, quirky precedence).
	if strings.Contains(inFormat, "Degrees") {
		dirs := reCoordDirs.FindAllString(strings.ToUpper(input), -1)
		if len(dirs) >= 1 {
			if dirs[0] == "S" || (dirs[0] == "W" && lat > 0) {
				lat = -lat
			}
			if len(dirs) >= 2 {
				if dirs[1] == "S" || (dirs[1] == "W" && lon > 0) {
					lon = -lon
				}
			}
		}
	}

	latDir, longDir := findDirs(jsNum(lat)+","+jsNum(lon), ",")

	var convLat, convLon string
	switch outFormat {
	case "Decimal Degrees":
		convLat, convLon = convDDToDD(lat, precision), convDDToDD(lon, precision)
	case "Degrees Decimal Minutes":
		convLat, convLon = convDDToDDM(lat, precision), convDDToDDM(lon, precision)
	case "Degrees Minutes Seconds":
		convLat, convLon = convDDToDMS(lat, precision), convDDToDMS(lon, precision)
	case "Geohash":
		convLat = geohash.EncodeWithPrecision(lat, lon, uint(precision))
	case "Military Grid Reference System":
		p := precision
		if p%2 != 0 {
			p++
		}
		if p > 10 {
			p = 10
		}
		acc := int(math.Pow(10, float64(5-p/2)))
		m, err := coco.LL{Lat: lat, Lon: lon}.ToMGRS(acc)
		if err != nil {
			return "", fmt.Errorf("could not convert co-ordinates to MGRS: %w", err)
		}
		convLat = fmtMGRS(string(m))
	case "Ordnance Survey National Grid":
		e, n, _ := wgs84.To(wgs84.OSGB36NationalGrid())(lon, lat, 0)
		p := precision
		if p%2 != 0 {
			p++
		}
		if p > 10 {
			p = 10
		}
		convLat = osgbToGrid(e, n, p)
		if convLat == "" {
			return "", fmt.Errorf("could not convert co-ordinates to OS National Grid. Are the co-ordinates in range?")
		}
	case "Universal Transverse Mercator":
		e, n, zone, _, err := UTM.FromLatLon(lat, lon, lat >= 0)
		if err != nil {
			return "", fmt.Errorf("could not convert co-ordinates to UTM: %w", err)
		}
		hemi := "N"
		if lat < 0 {
			hemi = "S"
		}
		convLat = fmt.Sprintf("%d %s %s %s", zone, hemi,
			strconv.FormatFloat(e, 'f', precision, 64), strconv.FormatFloat(n, 'f', precision, 64))
	default:
		// outFormat is validated against coordFormats by the arg layer, so every
		// format has a case above; reaching here means an unvalidated value slipped
		// through, which is a programming error.
		panic(fmt.Sprintf("convertCoordinates: unhandled output format %q", outFormat))
	}
	// convLat is empty only for a zero-precision Geohash.
	if convLat == "" {
		return "", fmt.Errorf("error converting co-ordinates")
	}

	if strings.Contains(outFormat, "Degrees") {
		if latDir == "S" && includeDir != "None" {
			convLat = strings.Replace(convLat, "-", "", 1)
		}
		if longDir == "W" && includeDir != "None" {
			convLon = strings.Replace(convLon, "-", "", 1)
		}
		var out strings.Builder
		if includeDir == "Before" {
			out.WriteString(latDir + " ")
		}
		out.WriteString(convLat)
		if includeDir == "After" {
			out.WriteString(" " + latDir)
		}
		out.WriteString(outDelim)
		if isPair {
			if includeDir == "Before" {
				out.WriteString(longDir + " ")
			}
			out.WriteString(convLon)
			if includeDir == "After" {
				out.WriteString(" " + longDir)
			}
			out.WriteString(outDelim)
		}
		return out.String(), nil
	}
	return convLat + outDelim, nil
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
	lat, lon, err := UTM.ToLatLon(e, n, zone, "", hemi == "N")
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UTM co-ordinate: %w", err)
	}
	return lat, lon, nil
}
