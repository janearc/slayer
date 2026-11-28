package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The head is drawn in its own frame, x forward from the occiput, y up,
// z across, and then placed on the neck and pitched nose down. The skull
// is a skin-covered rear and a bare horn beak over a long rostrum, with a
// low crest. The lower jaw is a separate part that hinges: its angle is
// the pose's gape. After the wide, robust skull and long toothless beak
// of azhdarchids (Witton and Naish 2008; Naish and Witton 2017). The
// holotype preserves only the posterior skull (Averianov 2014), so the
// rostrum, the crest and the lower jaw are authored.

// The skull's stations, as shares of its length from the occiput, and its
// half width and half height at each, in metres.
var (
	skullShares = []float64{0, .04, .085, .15, .26, .42, .62, .82, 1}
	skullWidths = []float64{
		pterosaurSkullBackWidth, .19, pterosaurSkullCheekWidth, .17,
		.11, .07, .045, .03, .016}
	skullHeights = []float64{
		pterosaurSkullBackHeight, .185, pterosaurSkullCheekHeight, .17,
		.125, .085, .058, .036, .02}
)

// The numbers the head's surface uses.
const (
	// headSides and headSpacing are the rings' size on the skull.
	headSides   = 40
	headSpacing = .02
	// sheathGrow is how far the horn sheath stands off the skull, in
	// metres.
	sheathGrow = .004
	// mouthLine is where the jaws meet, as a share of the beak's half
	// height below its axis. Authored.
	mouthLine = .55
	// jawHeightShare and jawWidthShare are the lower jaw's half height
	// and half width as shares of the upper beak's. Authored.
	jawHeightShare = .42
	jawWidthShare  = .78
	// jawBack is where the lower jaw's rear end is, as a share of the
	// skull's length, behind the hinge so that it tucks into the skull
	// when the jaw opens. jawRearSize is its size there as a share of the
	// full jaw's, and jawTaperReach how far along it grows to full size.
	jawBack       = .03
	jawRearSize   = .45
	jawTaperReach = .14
	// jawEnd is where the lower jaw ends, as a share of the skull's
	// length: a little short of the upper beak's tip. Authored.
	jawEnd = .965
	// cheekRelief and cheekReach shape the jaw muscle's swelling, as a
	// share of the radius and of the skull's length. Authored.
	cheekRelief = .07
	cheekReach  = .09
	// browRelief shapes the ridge over the eye. Authored.
	browRelief = .05
	// grooveRelief is the depth of the fine grooves along the horn, as a
	// share of the radius, and grooveCount how many ring the beak.
	grooveRelief = .01
	grooveCount  = 24
	// eyeSocketHeight and eyeSocketAt place the eye: its height above
	// the skull's axis and its place along, as a share of the length.
	eyeSocketHeight = .05
	eyeSocketAt     = .085
)

// skullSection is the skull's half width and half height at a share of
// its length, interpolated between the stations.
func skullSection(share float64) (float64, float64) {
	share = clamp(share, 0, 1)
	i := 0
	for i < len(skullShares)-2 && skullShares[i+1] < share {
		i++
	}
	span := skullShares[i+1] - skullShares[i]
	local := (share - skullShares[i]) / span
	width := skullWidths[i] + (skullWidths[i+1]-skullWidths[i])*local
	height := skullHeights[i] + (skullHeights[i+1]-skullHeights[i])*local
	return width, height
}

// skullKeys are control rings from one share of the skull's length to
// another, each grown by grow metres, on the skull's axis.
func skullKeys(from, to, grow, length float64) []tubeKey {
	shares := []float64{from}
	for _, share := range skullShares {
		if share > from && share < to {
			shares = append(shares, share)
		}
	}
	shares = append(shares, to)
	keys := make([]tubeKey, len(shares))
	for i, share := range shares {
		width, height := skullSection(share)
		keys[i] = tubeKey{maths.V(share*length, 0, 0), width + grow,
			height + grow, 2.3}
	}
	return keys
}

// gauss is a bell of unit height and the given width around centre.
func gauss(at, centre, width float64) float64 {
	offset := (at - centre) / width
	return math.Exp(-offset * offset)
}

// skullRelief is the rear skull's outline multiplier: the jaw muscle's
// swelling on each cheek and the brow over the eye.
func skullRelief(along, angle, length float64) float64 {
	cheek := cheekRelief * gauss(along/length, .11, cheekReach) *
		math.Cos(angle) * math.Cos(angle)
	brow := browRelief * gauss(along/length, eyeSocketAt, cheekReach) *
		gauss(angle, math.Pi/2, .5)
	return 1 + cheek + brow
}

// sheathRelief is the horn's outline multiplier: a keel along the top and
// fine grooves.
func sheathRelief(_, angle float64) float64 {
	keel := .03 * gauss(angle, math.Pi/2, .2)
	return 1 + keel + grooveRelief*math.Cos(grooveCount*angle)
}

// crestKeys are the control rings of the crest: a thin blade standing on
// the skull's top from pterosaurCrestStart to pterosaurCrestEnd of its
// length, rising and falling smoothly.
func crestKeys(length float64) []tubeKey {
	const count = 7
	keys := make([]tubeKey, count)
	for i := range keys {
		share := float64(i) / float64(count-1)
		reach := pterosaurCrestEnd - pterosaurCrestStart
		at := pterosaurCrestStart + reach*share
		_, base := skullSection(at)
		bell := math.Pow(math.Sin(math.Pi*share), .7)
		rise := pterosaurCrestHeight * bell
		keys[i] = tubeKey{maths.V(at*length, base*.8+rise/2, 0), .012,
			math.Max(tubeFloor, rise/2), 2.6}
	}
	return keys
}

// jawKeys are the lower jaw's control rings from one share of the skull's
// length to another, below the mouth line, before it is opened.
func jawKeys(from, to, length float64) []tubeKey {
	keys := skullKeys(from, to, 0, length)
	for i, key := range keys {
		share := key.centre.X / length
		_, height := skullSection(share)
		taper := jawRearSize + (1-jawRearSize)*
			smoothShare((share-jawBack)/jawTaperReach)
		keys[i].halfHeight = jawHeightShare * height * taper
		keys[i].centre.Y = -mouthLine*height - keys[i].halfHeight
		keys[i].halfWidth = jawWidthShare * key.halfWidth * taper
	}
	return keys
}

// headFrame places a head's own space on the neck: its origin at the
// occiput, turned nose down by pitch about the span axis.
type headFrame struct {
	origin maths.Vec
	pitch  float64
}

// place is the point of the head's own space in the animal's space.
func (frame headFrame) place(point maths.Vec) maths.Vec {
	turned := turnedDown(point, maths.V(0, 0, 0), frame.pitch)
	return frame.origin.Add(turned)
}

// turnedDown is the point turned nose down by angle about the span axis
// through pivot.
func turnedDown(point, pivot maths.Vec, angle float64) maths.Vec {
	cosine, sine := math.Cos(angle), math.Sin(angle)
	offset := point.Sub(pivot)
	return pivot.Add(maths.V(offset.X*cosine+offset.Y*sine,
		-offset.X*sine+offset.Y*cosine, offset.Z))
}

// jawPivot is where the lower jaw hinges, in the head's own space.
func jawPivot(length float64) maths.Vec {
	_, height := skullSection(pterosaurJawJoint)
	return maths.V(pterosaurJawJoint*length, -mouthLine*height, 0)
}
