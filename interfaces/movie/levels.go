package movie

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"image"
	"strconv"
	"strings"
)

// levelTable is ITU-T H.264 (08/2021) table A-1, from level 3.0, the
// lowest VideoToolbox names for High profile, to 6.2: the most
// macroblocks a second and a frame, and the base MaxBR in kbit/s.
//
//go:embed levels.csv
var levelTable string

// levelColumns is how many columns each row of the table has. Defined
// by its heading.
const levelColumns = 4

// anyFields lets rows of the table hold any count of fields, so that
// readLevel can refuse a short or long row by its number. Defined by
// encoding/csv's FieldsPerRecord.
const anyFields = -1

// level is one row of table A-1.
type level struct {
	name      string // as the standard writes it, "4.1"
	idc       int    // level_idc, as the stream carries it, 41
	perSecond int64  // MaxMBPS
	perFrame  int64  // MaxFS
	kbits     int64  // MaxBR for Baseline and Main, kbit/s
}

// demand is what a movie asks of a level.
type demand struct {
	across, down int64 // the coded picture's sides, in macroblocks
	rate         int64 // frames a second
	bitRate      int64 // bits a second asked of the encoder
}

// newDemand is what a coded picture of that size, at that rate, asks.
func newDemand(coded image.Point, rate int) demand {
	pixels := float64(coded.X) * float64(coded.Y) * float64(rate)
	return demand{
		across:  macroblocks(coded.X),
		down:    macroblocks(coded.Y),
		rate:    int64(rate),
		bitRate: int64(pixels * bitsPerPixel),
	}
}

// macroblocks is how many macroblocks cover a side of that many pixels.
func macroblocks(pixels int) int64 {
	return (int64(pixels) + macroblock - one) / macroblock
}

// fits is whether the level allows what the movie asks.
func (row level) fits(asked demand) bool {
	frame := asked.across * asked.down
	side := frameAreaRoot * row.perFrame
	allowed := row.kbits * highBitRate
	return frame <= row.perFrame &&
		frame*asked.rate <= row.perSecond &&
		asked.bitRate <= allowed &&
		asked.across*asked.across <= side &&
		asked.down*asked.down <= side
}

// levelFor is the lowest level the movie fits, refused by name when
// that is above 5.2 or when no level holds it. It is nil on a refusal.
func levelFor(asked demand) (*level, error) {
	rows, err := readLevels(levelTable)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if !row.fits(asked) {
			continue
		}
		if row.idc > highestLevel {
			return nil, fmt.Errorf("movie: this size at this rate "+
				"needs H.264 level %s, and QuickTime shows above 5.2 "+
				"as blocks: make the picture smaller or the rate lower",
				row.name)
		}
		return &row, nil
	}
	return nil, fmt.Errorf("movie: this size at this rate is "+
		"beyond every H.264 level, the last being %s: make the picture "+
		"smaller or the rate lower", rows[len(rows)-one].name)
}

// readLevels reads the level table, refusing a malformed one by row, and
// one whose levels do not rise, since levelFor takes the first that fits.
func readLevels(text string) ([]level, error) {
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = anyFields
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("movie: the level table: %w", err)
	}
	if len(records) == zero {
		return nil, fmt.Errorf("movie: the level table has no heading")
	}
	var rows []level
	for place, record := range records[headingRows:] {
		row, err := readLevel(record)
		if err != nil {
			return nil, fmt.Errorf("movie: the level table, row %d: %w",
				place+one, err)
		}
		if len(rows) > zero && row.idc <= rows[len(rows)-one].idc {
			return nil, fmt.Errorf("movie: the level table, row %d: "+
				"level %s is not above the one before it: the levels "+
				"must be in order", place+one, row.name)
		}
		rows = append(rows, *row)
	}
	if len(rows) == zero {
		return nil, fmt.Errorf("movie: the level table has no rows")
	}
	return rows, nil
}

// readLevel reads one row: a level written as 4.1, then three counts. It
// is nil on a refusal.
func readLevel(record []string) (*level, error) {
	if len(record) != levelColumns {
		return nil, fmt.Errorf("%d columns, want %d", len(record),
			levelColumns)
	}
	name := record[nameColumn]
	majorText, minorText, found := strings.Cut(name, ".")
	major, majorErr := strconv.Atoi(majorText)
	minor, minorErr := strconv.Atoi(minorText)
	if !found || majorErr != nil || minorErr != nil {
		return nil, fmt.Errorf("%q is not a level such as 4.1", name)
	}
	row := level{name: name, idc: major*levelTenths + minor}
	counts := []*int64{&row.perSecond, &row.perFrame, &row.kbits}
	for place, count := range counts {
		text := record[place+firstCount]
		read, err := strconv.ParseInt(text, decimal, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a count", text)
		}
		*count = read
	}
	return &row, nil
}
