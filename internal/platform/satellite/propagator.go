package satellite

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// SGP4Elements holds parsed TLE orbital elements.
type SGP4Elements struct {
	EpochYear      int
	EpochDay       float64
	MeanMotion     float64 // revs/day
	Eccentricity   float64
	Inclination    float64 // degrees
	RAAN           float64 // Right Ascension of Ascending Node (degrees)
	ArgPerigee     float64 // degrees
	MeanAnomaly    float64 // degrees
	BStar          float64 // drag term
	MeanMotionDot  float64 // first derivative
	MeanMotionDDot float64 // second derivative
}

// ParseTLE extracts orbital elements from TLE lines.
func ParseTLE(line1, line2 string) (*SGP4Elements, error) {
	if len(line1) < 69 || len(line2) < 69 {
		return nil, fmt.Errorf("TLE lines too short")
	}

	elements := &SGP4Elements{}

	// Line 1 parsing
	epochYearStr := strings.TrimSpace(line1[18:20])
	epochDayStr := strings.TrimSpace(line1[20:32])

	epochYear, err := strconv.Atoi(epochYearStr)
	if err != nil {
		return nil, fmt.Errorf("parse epoch year: %w", err)
	}
	if epochYear >= 57 {
		elements.EpochYear = 1900 + epochYear
	} else {
		elements.EpochYear = 2000 + epochYear
	}

	elements.EpochDay, err = strconv.ParseFloat(epochDayStr, 64)
	if err != nil {
		return nil, fmt.Errorf("parse epoch day: %w", err)
	}

	// Mean motion derivatives
	mmDotStr := strings.TrimSpace(line1[33:43])
	elements.MeanMotionDot, _ = strconv.ParseFloat(mmDotStr, 64)

	// BStar drag term (in scientific notation format)
	bstarStr := strings.TrimSpace(line1[53:61])
	if len(bstarStr) >= 6 {
		mantissa := bstarStr[:6]
		expStr := bstarStr[6:]
		m, _ := strconv.ParseFloat("."+strings.TrimLeft(mantissa, " -+"), 64)
		if strings.HasPrefix(bstarStr, "-") {
			m = -m
		}
		exp, _ := strconv.Atoi(expStr)
		elements.BStar = m * math.Pow(10, float64(exp))
	}

	// Line 2 parsing
	incStr := strings.TrimSpace(line2[8:16])
	raanStr := strings.TrimSpace(line2[17:25])
	eccStr := strings.TrimSpace(line2[26:33])
	argpStr := strings.TrimSpace(line2[34:42])
	maStr := strings.TrimSpace(line2[43:51])
	mmStr := strings.TrimSpace(line2[52:63])

	elements.Inclination, _ = strconv.ParseFloat(incStr, 64)
	elements.RAAN, _ = strconv.ParseFloat(raanStr, 64)

	// Eccentricity is implied decimal
	if eccFloat, err := strconv.ParseFloat("0."+eccStr, 64); err == nil {
		elements.Eccentricity = eccFloat
	}

	elements.ArgPerigee, _ = strconv.ParseFloat(argpStr, 64)
	elements.MeanAnomaly, _ = strconv.ParseFloat(maStr, 64)
	elements.MeanMotion, _ = strconv.ParseFloat(mmStr, 64)

	return elements, nil
}

// Propagator computes satellite positions from TLE data using simplified SGP4.
type Propagator struct {
	elements *SGP4Elements
	epoch    time.Time
}

// NewPropagator creates a propagator from TLE data.
func NewPropagator(tle *TLE) (*Propagator, error) {
	elements, err := ParseTLE(tle.Line1, tle.Line2)
	if err != nil {
		return nil, err
	}

	// Calculate epoch time
	year := elements.EpochYear
	dayOfYear := elements.EpochDay
	days := int(dayOfYear)
	fraction := dayOfYear - float64(days)

	epochTime := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	epochTime = epochTime.AddDate(0, 0, days-1)
	epochTime = epochTime.Add(time.Duration(fraction * 24 * float64(time.Hour)))

	return &Propagator{
		elements: elements,
		epoch:    epochTime,
	}, nil
}

// Propagate computes the satellite position at a given time.
// Returns latitude, longitude (degrees) and altitude (km).
func (p *Propagator) Propagate(t time.Time) (lat, lon, alt float64) {
	// Earth constants
	const (
		mu = 398600.4418 // km^3/s^2 - Earth gravitational parameter
		Re = 6378.137    // km - Earth equatorial radius
		J2 = 0.00108263  // J2 perturbation coefficient
	)

	// Minutes since epoch
	minutesSinceEpoch := t.Sub(p.epoch).Minutes()

	// Convert mean motion from rev/day to rad/min
	n := p.elements.MeanMotion * 2 * math.Pi / 1440.0

	// Semi-major axis from mean motion (Kepler's third law)
	// n = sqrt(mu/a^3), so a = (mu/n^2)^(1/3)
	// But n here is in rad/min, need to convert: n_rad_s = n / 60
	nRadSec := n / 60.0
	a := math.Pow(mu/(nRadSec*nRadSec), 1.0/3.0)

	e := p.elements.Eccentricity
	i := p.elements.Inclination * math.Pi / 180.0
	omega0 := p.elements.RAAN * math.Pi / 180.0
	w0 := p.elements.ArgPerigee * math.Pi / 180.0
	M0 := p.elements.MeanAnomaly * math.Pi / 180.0

	// J2 secular perturbations (rates in rad/min)
	p0 := a * (1 - e*e)
	omegaDot := -1.5 * n * J2 * math.Pow(Re/p0, 2) * math.Cos(i)
	wDot := 0.75 * n * J2 * math.Pow(Re/p0, 2) * (5*math.Cos(i)*math.Cos(i) - 1)

	// Updated orbital elements
	omega := omega0 + omegaDot*minutesSinceEpoch
	w := w0 + wDot*minutesSinceEpoch

	// Mean anomaly at time t
	M := M0 + n*minutesSinceEpoch
	M = math.Mod(M, 2*math.Pi)
	if M < 0 {
		M += 2 * math.Pi
	}

	// Solve Kepler's equation for eccentric anomaly (Newton-Raphson)
	E := M
	for iter := 0; iter < 15; iter++ {
		dE := (E - e*math.Sin(E) - M) / (1 - e*math.Cos(E))
		E -= dE
		if math.Abs(dE) < 1e-12 {
			break
		}
	}

	// True anomaly
	sinNu := math.Sqrt(1-e*e) * math.Sin(E) / (1 - e*math.Cos(E))
	cosNu := (math.Cos(E) - e) / (1 - e*math.Cos(E))
	nu := math.Atan2(sinNu, cosNu)

	// Distance from Earth center
	r := a * (1 - e*math.Cos(E))

	// Argument of latitude
	u := w + nu

	// Position in orbital plane (perifocal coordinates)
	xPF := r * math.Cos(u)
	yPF := r * math.Sin(u)

	// Transform to ECI coordinates
	cosO := math.Cos(omega)
	sinO := math.Sin(omega)
	cosI := math.Cos(i)
	sinI := math.Sin(i)

	xECI := xPF*cosO - yPF*sinO*cosI
	yECI := xPF*sinO + yPF*cosO*cosI
	zECI := yPF * sinI

	// Greenwich Mean Sidereal Time
	jd := timeToJD(t)
	gmst := jdToGMST(jd)

	// Convert ECI to ECEF (rotate by GMST)
	cosGMST := math.Cos(gmst)
	sinGMST := math.Sin(gmst)
	xECEF := xECI*cosGMST + yECI*sinGMST
	yECEF := -xECI*sinGMST + yECI*cosGMST
	zECEF := zECI

	// Convert ECEF to geodetic coordinates (simplified spherical)
	rMag := math.Sqrt(xECEF*xECEF + yECEF*yECEF + zECEF*zECEF)
	lat = math.Asin(zECEF/rMag) * 180.0 / math.Pi
	lon = math.Atan2(yECEF, xECEF) * 180.0 / math.Pi
	alt = rMag - Re

	return lat, lon, alt
}

// PropagateRange returns positions over a time range.
func (p *Propagator) PropagateRange(start time.Time, duration time.Duration, step time.Duration) []PropagatedPosition {
	positions := make([]PropagatedPosition, 0)
	for t := start; t.Before(start.Add(duration)); t = t.Add(step) {
		lat, lon, alt := p.Propagate(t)
		positions = append(positions, PropagatedPosition{
			Time:      t,
			Latitude:  lat,
			Longitude: lon,
			Altitude:  alt,
		})
	}
	return positions
}

// PropagatedPosition represents a computed satellite position.
type PropagatedPosition struct {
	Time      time.Time `json:"time"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"` // km
}

// timeToJD converts time to Julian Date.
func timeToJD(t time.Time) float64 {
	y := float64(t.Year())
	m := float64(t.Month())
	d := float64(t.Day())
	h := float64(t.Hour()) + float64(t.Minute())/60 + float64(t.Second())/3600

	if m <= 2 {
		y--
		m += 12
	}

	A := math.Floor(y / 100)
	B := 2 - A + math.Floor(A/4)

	jd := math.Floor(365.25*(y+4716)) + math.Floor(30.6001*(m+1)) + d + h/24 + B - 1524.5
	return jd
}

// jdToGMST converts Julian Date to Greenwich Mean Sidereal Time (radians).
func jdToGMST(jd float64) float64 {
	// Julian centuries from J2000.0
	T := (jd - 2451545.0) / 36525.0

	// GMST in seconds
	gmstSec := 67310.54841 +
		(876600*3600+8640184.812866)*T +
		0.093104*T*T -
		6.2e-6*T*T*T

	// Convert to radians
	gmst := math.Mod(gmstSec*2*math.Pi/86400, 2*math.Pi)
	if gmst < 0 {
		gmst += 2 * math.Pi
	}
	return gmst
}
