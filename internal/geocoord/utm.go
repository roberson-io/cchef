package geocoord

import (
	"errors"
	"math"
)

// WGS84 conversion between latitude/longitude and Universal Transverse
// Mercator grid coordinates, using the standard Krüger series. Ported from
// the MIT-licensed utm Python package by Tobias Bieniek
// (https://github.com/Turbo87/utm).

// utmK0 is the UTM scale factor on the central meridian.
const utmK0 = 0.9996

// utmR is the WGS84 equatorial radius in metres.
const utmR = 6378137

// utmE is the WGS84 first eccentricity squared.
const utmE = 0.00669438

// utmFalseEasting places the central meridian at 500,000 m so eastings stay
// positive across a zone.
const utmFalseEasting = 500000

// utmFalseNorthing is added to southern-hemisphere northings so they count up
// from the south rather than negative from the equator.
const utmFalseNorthing = 10000000

// The remaining series coefficients all derive from the eccentricity.
var (
	utmE2  = utmE * utmE
	utmE3  = utmE2 * utmE
	utmEP2 = utmE / (1 - utmE)

	utmSqrtE = math.Sqrt(1 - utmE)
	utmN     = (1 - utmSqrtE) / (1 + utmSqrtE)
	utmN2    = utmN * utmN
	utmN3    = utmN2 * utmN
	utmN4    = utmN3 * utmN
	utmN5    = utmN4 * utmN

	utmM1 = 1 - utmE/4 - 3*utmE2/64 - 5*utmE3/256
	utmM2 = 3*utmE/8 + 3*utmE2/32 + 45*utmE3/1024
	utmM3 = 15*utmE2/256 + 45*utmE3/1024
	utmM4 = 35 * utmE3 / 3072

	utmP2 = 3.0/2*utmN - 27.0/32*utmN3 + 269.0/512*utmN5
	utmP3 = 21.0/16*utmN2 - 55.0/32*utmN4
	utmP4 = 151.0/96*utmN3 - 417.0/128*utmN5
	utmP5 = 1097.0 / 512 * utmN4
)

// utmModAngle wraps an angle in radians into [-pi, pi).
func utmModAngle(value float64) float64 {
	wrapped := math.Mod(value+math.Pi, 2*math.Pi)
	if wrapped < 0 {
		wrapped += 2 * math.Pi
	}
	return wrapped - math.Pi
}

// utmZoneNumber picks the UTM zone for a point, including the widened zone 32
// over south-west Norway and the even-numbered gaps around Svalbard.
func utmZoneNumber(lat, lon float64) int {
	lon = math.Mod(math.Mod(lon, 360)+540, 360) - 180

	if 56 <= lat && lat < 64 && 3 <= lon && lon < 12 {
		return 32
	}
	if 72 <= lat && lat <= 84 && lon >= 0 {
		switch {
		case lon < 9:
			return 31
		case lon < 21:
			return 33
		case lon < 33:
			return 35
		case lon < 42:
			return 37
		}
	}
	return int((lon+180)/6) + 1
}

// utmCentralLongitude is the central meridian of a zone in degrees.
func utmCentralLongitude(zoneNumber int) float64 {
	return float64((zoneNumber-1)*6 - 180 + 3)
}

// utmFromLatLon converts a WGS84 latitude/longitude to UTM easting, northing
// and zone number. Southern-hemisphere northings carry the false northing.
func utmFromLatLon(lat, lon float64) (easting, northing float64, zoneNumber int, err error) {
	if lat < -80 || lat > 84 {
		return 0, 0, 0, errors.New("latitude out of range (must be between 80 deg S and 84 deg N)")
	}
	if lon < -180 || lon > 180 {
		return 0, 0, 0, errors.New("longitude out of range (must be between 180 deg W and 180 deg E)")
	}

	latRad := lat * math.Pi / 180
	latSin, latCos := math.Sincos(latRad)
	latTan := latSin / latCos
	latTan2 := latTan * latTan
	latTan4 := latTan2 * latTan2

	zoneNumber = utmZoneNumber(lat, lon)
	lonRad := lon * math.Pi / 180
	centralRad := utmCentralLongitude(zoneNumber) * math.Pi / 180

	n := utmR / math.Sqrt(1-utmE*latSin*latSin)
	c := utmEP2 * latCos * latCos

	a := latCos * utmModAngle(lonRad-centralRad)
	a2 := a * a
	a3 := a2 * a
	a4 := a3 * a
	a5 := a4 * a
	a6 := a5 * a

	m := utmR * (utmM1*latRad -
		utmM2*math.Sin(2*latRad) +
		utmM3*math.Sin(4*latRad) -
		utmM4*math.Sin(6*latRad))

	easting = utmK0*n*(a+
		a3/6*(1-latTan2+c)+
		a5/120*(5-18*latTan2+latTan4+72*c-58*utmEP2)) + utmFalseEasting

	northing = utmK0 * (m + n*latTan*(a2/2+
		a4/24*(5-latTan2+9*c+4*c*c)+
		a6/720*(61-58*latTan2+latTan4+600*c-330*utmEP2)))
	if lat < 0 {
		northing += utmFalseNorthing
	}
	return easting, northing, zoneNumber, nil
}

// utmToLatLon converts UTM coordinates back to WGS84 latitude/longitude.
// northern selects the hemisphere; southern northings are counted down from
// the false northing.
func utmToLatLon(easting, northing float64, zoneNumber int, northern bool) (lat, lon float64, err error) {
	if easting < 100000 || easting >= 1000000 {
		return 0, 0, errors.New("easting out of range (must be between 100,000 m and 999,999 m)")
	}
	if northing < 0 || northing > utmFalseNorthing {
		return 0, 0, errors.New("northing out of range (must be between 0 m and 10,000,000 m)")
	}
	if zoneNumber < 1 || zoneNumber > 60 {
		return 0, 0, errors.New("zone number out of range (must be between 1 and 60)")
	}

	x := easting - utmFalseEasting
	y := northing
	if !northern {
		y -= utmFalseNorthing
	}

	m := y / utmK0
	mu := m / (utmR * utmM1)

	pRad := mu +
		utmP2*math.Sin(2*mu) +
		utmP3*math.Sin(4*mu) +
		utmP4*math.Sin(6*mu) +
		utmP5*math.Sin(8*mu)

	pSin, pCos := math.Sincos(pRad)
	pSin2 := pSin * pSin

	pTan := pSin / pCos
	pTan2 := pTan * pTan
	pTan4 := pTan2 * pTan2

	epSin := 1 - utmE*pSin2
	epSinSqrt := math.Sqrt(epSin)

	n := utmR / epSinSqrt
	r := (1 - utmE) / epSin

	c := utmEP2 * pCos * pCos
	c2 := c * c

	d := x / (n * utmK0)
	d2 := d * d
	d3 := d2 * d
	d4 := d3 * d
	d5 := d4 * d
	d6 := d5 * d

	lat = pRad - (pTan/r)*(d2/2-
		d4/24*(5+3*pTan2+10*c-4*c2-9*utmEP2)+
		d6/720*(61+90*pTan2+298*c+45*pTan4-252*utmEP2-3*c2))

	lon = (d -
		d3/6*(1+2*pTan2+c) +
		d5/120*(5-2*c+28*pTan2-3*c2+8*utmEP2+24*pTan4)) / pCos

	lon = utmModAngle(lon + utmCentralLongitude(zoneNumber)*math.Pi/180)

	return lat * 180 / math.Pi, lon * 180 / math.Pi, nil
}
