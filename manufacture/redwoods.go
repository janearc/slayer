package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
)

// These are authored specimen ranges, 2026-11-02, not field data.
// Coast redwoods have flat sprays; giant sequoias have scale-like foliage.
//
// Pinned OSU descriptions: osu-coast-redwood-foliage and
// osu-giant-sequoia-foliage in sources.yaml; see doc/conifer-foliage.md.
// No transpiration rate is inferred here; see doc/redwood-atmosphere.md.
func redwoodRow() conifer {
	row := norwayRow()
	row.tall, row.slender = span{18, 42}, span{0.018, 0.026}
	row.between, row.lowest = span{1.1, 1.8}, span{3, 5}
	row.hangs.start, row.hangs.reach = 0.30, 0.22
	// The spruce's minimum branch radius scales with the whole trunk.
	// On a tall redwood that swallowed the small upper foliage sleeves.
	// These authored proportions let the crown taper with its branches.
	row.hangs.least, row.hangs.girth = 0.0002, 0.22
	row.hangs.top = 0.995
	row.hangs.low, row.hangs.lift = measure.Degrees(86), measure.Degrees(22)
	row.grows.branch.droop = 0.06
	row.grows.shoots.per = span{
		shootValue("redwood-branchlet-minimum"),
		shootValue("redwood-branchlet-maximum"),
	}
	row.grows.shoots.shoot.droop = 0.08
	// The arrangement sizes children against one segment; the recorded
	// length ratio refers to the complete primary branch.
	row.grows.shoots.set.reach = shootValue("redwood-branchlet-length") *
		float64(row.grows.branch.pieces)
	row.grows.shoots.wear = sleeveFoliage{joints: lengths(0.05, 0.09, 0), sides: 4}
	row.foliageStyle, row.flare = 1, 1.6
	row.dressed = looks{
		body: plain("redwood bark", 0.31, 0.16, 0.10),
		leaf: plain("redwood foliage", 0.07, 0.24, 0.12),
	}
	return row
}

// sequoiaRow gives the authored family a stockier trunk and scaly detail.
func sequoiaRow() conifer {
	row := redwoodRow()
	row.tall, row.slender = span{20, 45}, span{0.045, 0.065}
	row.between = span{1.4, 2.4}
	row.hangs.start, row.hangs.reach = 0.25, 0.24
	row.hangs.low = measure.Degrees(72)
	row.grows.branch.droop = 0.12
	row.grows.shoots.set.turn = measure.Radians(math.Pi * (3 - math.Sqrt(5)))
	row.grows.shoots.shoot.droop = 0.18
	row.foliageStyle, row.flare = 2, 1.5
	row.dressed = looks{
		body: plain("sequoia bark", 0.37, 0.20, 0.12),
		leaf: plain("sequoia foliage", 0.10, 0.27, 0.13),
	}
	return row
}
