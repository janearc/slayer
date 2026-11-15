package sky

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/slayer/maths"
)

// The Yale Bright Star Catalogue, fifth revised edition, as the file
// ybsc5 writes it: one star a line, in fixed columns. The columns here
// were checked against the file itself on 2026-11-12: Sirius, HR 2491,
// reads 06 45 08.9, -16 42 58 and magnitude -1.46, and Vega, HR 7001,
// reads 18 36 56.3, +38 47 01 and 0.03. They count from one, as the
// catalogue's own description does, and are both ends of a field.
const (
	brightHoursFrom, brightHoursTo     = 76, 77   // read: right ascension, hours, equinox J2000
	brightMinutesFrom, brightMinutesTo = 78, 79   // read: its minutes
	brightSecondsFrom, brightSecondsTo = 80, 83   // read: its seconds, to a tenth
	brightSign                         = 84       // read: the declination's sign
	brightDegreesFrom, brightDegreesTo = 85, 86   // read: declination, degrees
	brightArcminFrom, brightArcminTo   = 87, 88   // read: its minutes of arc
	brightArcsecFrom, brightArcsecTo   = 89, 90   // read: its seconds of arc
	brightVisualFrom, brightVisualTo   = 103, 107 // read: visual magnitude
	brightWide                         = 107      // defined: a line shorter than this holds no magnitude

	hoursToDegrees = 15.0 // defined: an hour of right ascension is fifteen degrees
	sixty          = 60.0 // defined: minutes in an hour or a degree, seconds in a minute
	perMagnitude   = -0.4 // defined: a magnitude is a factor of ten to the minus two fifths in light
)

// field is the columns from one to another of a line, both counted from
// one, with the blanks round it taken off.
func field(line string, from, to int) string {
	return strings.TrimSpace(line[from-1 : to])
}

// sexagesimal is three fields of a line as one number: whole ones,
// sixtieths, and sixtieths of those.
func sexagesimal(line string, columns [6]int) (float64, error) {
	var total float64
	share := 1.0
	for part := 0; part < len(columns); part += 2 {
		read, err := strconv.ParseFloat(field(line, columns[part], columns[part+1]), 64)
		if err != nil {
			return 0, err
		}
		total += read * share
		share /= sixty
	}
	return total, nil
}

// brightStar is one line of the catalogue as a star, and whether the
// line holds one: a line with no magnitude is an entry the catalogue
// keeps for a thing that is no star, and is no star here either.
func brightStar(line string, zero maths.Light) (Star, bool, error) {
	if len(line) < brightWide || field(line, brightVisualFrom, brightVisualTo) == "" {
		return Star{}, false, nil
	}
	hours, err := sexagesimal(line, [6]int{brightHoursFrom, brightHoursTo,
		brightMinutesFrom, brightMinutesTo, brightSecondsFrom, brightSecondsTo})
	if err != nil {
		return Star{}, false, err
	}
	degrees, err := sexagesimal(line, [6]int{brightDegreesFrom, brightDegreesTo,
		brightArcminFrom, brightArcminTo, brightArcsecFrom, brightArcsecTo})
	if err != nil {
		return Star{}, false, err
	}
	if line[brightSign-1] == '-' {
		degrees = -degrees
	}
	visual, err := strconv.ParseFloat(field(line, brightVisualFrom, brightVisualTo), 64)
	if err != nil {
		return Star{}, false, err
	}
	along, up := hours*hoursToDegrees*maths.Degree, degrees*maths.Degree
	return Star{
		Toward: maths.V(math.Cos(up)*math.Cos(along),
			math.Cos(up)*math.Sin(along), math.Sin(up)),
		Light: zero.Scale(math.Pow(10, perMagnitude*visual)),
	}, true, nil
}

// BrightStars reads the Bright Star Catalogue's stars from its file,
// unpacked: each where the catalogue puts it at equinox J2000, as a unit
// vector with x toward right ascension nought on the equator and z
// toward the north pole, and each as bright as its visual magnitude
// makes it beside a star of magnitude nought, whose light is zero's.
//
// Every star is zero's colour: the catalogue's colours are not read. A
// line that is no star is passed over, and one that cannot be read is
// refused, by its number.
func BrightStars(from io.Reader, zero maths.Light) ([]Star, error) {
	var stars []Star
	lines := bufio.NewScanner(from)
	for number := 1; lines.Scan(); number++ {
		star, is, err := brightStar(lines.Text(), zero)
		if err != nil {
			return nil, fmt.Errorf("sky: the bright star catalogue's line %d: %w",
				number, err)
		}
		if is {
			stars = append(stars, star)
		}
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(stars) == 0 {
		return nil, fmt.Errorf("sky: the bright star catalogue held no stars")
	}
	return stars, nil
}
