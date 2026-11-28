package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

type scalpEdge struct {
	depth, stagger, ear float64
}

// edge follows the cranial chart around the hairline. Noise varies
// along the perimeter at two scales, so follicles at different depths
// in the same region inherit the same irregular boundary.
func (line Hairline) edge(point maths.Vec, seed uint64) scalpEdge {
	angle := math.Atan2(point.X, point.Z-.010)
	fromFront := math.Abs(angle)
	sine, forward := math.Sincos(angle)
	forehead := maths.Smooth(0, .88, forward)
	back := maths.Smooth(.04, .94, -forward)
	recess := bell((fromFront - .80) / .39)
	peak := bell(angle / .18)
	temple := bell((fromFront - 1.18) / .33)
	boundary := 1.589 + (.032+line.FrontRise)*forehead -
		(.046-line.NapeRise)*back + line.TempleRecession*recess -
		line.Peak*peak + line.Asymmetry*sine + .006*temple
	mounds := noise.Gradient(seed+13, 11*sine, 11*forward, .43)
	fine := noise.Gradient(seed+17, 35*sine, 35*forward, .73)
	nape := noise.Gradient(seed+19, 19*sine, 19*forward, .51)
	depth := point.Y - boundary + .0058*mounds + .0028*fine + .007*back*nape
	pinna := math.Hypot((fromFront-1.65)/.23, (point.Y-1.552)/.036)
	return scalpEdge{depth: depth, stagger: .58*mounds + .28*fine + .14*back*nape,
		ear: maths.Smooth(.80, 1.20, pinna)}
}

func (edge scalpEdge) coverage(population scalpPopulation) float64 {
	d := edge.depth
	delay := .0048 + .0044*edge.stagger
	switch population {
	case scalpFringe:
		return edge.ear * maths.Smooth(-.005, .0015, d) *
			(1 - maths.Smooth(.003, .011, d)) * (.81 + .19*edge.stagger)
	case scalpMargin:
		return edge.ear * maths.Smooth(-.0015, .0052, d) *
			(1 - maths.Smooth(delay+.003, delay+.014, d))
	default:
		return edge.ear * maths.Smooth(delay-.002, delay+.009, d)
	}
}

func (spec ScalpHair) strand(site hair.Site, population scalpPopulation, clumps hair.Clumps) hair.Strand {
	point := site.Chart
	grain := noise.Gradient(spec.Seed, 391*point.X, 391*point.Y, 391*point.Z)
	edge := spec.Line.edge(point, spec.Seed)
	inner := maths.Smooth(.0005, .024, edge.depth)
	high := maths.Smooth(1.600, 1.674, point.Y)
	crown := math.Pow(min(1, max(0, site.ChartNormal.Y)), .72) * maths.Smooth(1.563, 1.669, point.Y)
	part := spec.partDistance(point)
	partWeight := maths.Smooth(-.027, .016, point.Z) * maths.Smooth(1.598, 1.649, point.Y)
	closeToPart := 1 - maths.Smooth(.0005, .0016, math.Abs(part))
	coverage := edge.coverage(population)
	coverage *= (1 - .09*partWeight*closeToPart) * (.84 + .16*high) * (.95 + .05*grain)
	if population == scalpLoose {
		coverage *= maths.Smooth(.22, .66, high) * inner * (.22 + .78*crown)
	}
	if coverage <= 0 {
		return hair.Strand{}
	}

	sweep := noise.Gradient(spec.Seed+43, 43*point.X, 43*point.Y, 43*point.Z)
	turn := (.045+.13*(1-inner))*grain + .16*inner*sweep
	authored := spec.flow(site, high, part, partWeight, turn)
	flow := site.ChartFlow(authored).Unit()
	anchor := clumps.BlendTarget(point, .0063)
	lock := noise.Gradient(spec.Seed+39, 39*anchor.X, 39*anchor.Y, 39*anchor.Z)
	nape := maths.Smooth(.030, .078, -point.Z) * (1 - maths.Smooth(1.565, 1.648, point.Y))
	length := mix(mix(spec.SideLength, spec.NapeLength, nape), spec.Length, crown) *
		(.46 + .54*inner) * (.86 + .07*grain + .07*lock)
	radius := spec.Radius * (.61 + .31*inner + .08*grain)
	crest := .78 + .22*bell((point.X-spec.Part*.8)/.042)
	volume := (.22 + .78*crown) * crest * (.22 + .78*inner)
	transverse := site.ChartNormal.Cross(authored).Unit()
	pull := max(-.11, min(.11, (.29+.31*crown)*anchor.Sub(point).Dot(transverse)/length))
	pull *= .36 + .64*inner
	guide := spec.shaftGuide(volume, grain, lock, pull, population)
	switch population {
	case scalpFringe:
		length *= .34
		radius *= .49
	case scalpMargin:
		length *= .63
		radius *= .72
	case scalpLoose:
		radius *= .65
	}
	if population == scalpMain || population == scalpLoose {
		guide.Wave = spec.guideWave(length, volume, grain, lock)
	}
	return hair.Strand{
		Coverage: min(1, coverage), Flow: flow, Length: length, Radius: radius, Guide: guide,
		SkinFollow: min(.98, .41+.55*(1-crown)+.10*(1-inner)),
	}
}

func (spec ScalpHair) shaftGuide(volume, grain, lock, pull float64, population scalpPopulation) hair.Guide {
	lift := .061 + .22*volume + .025*grain
	wave := (.033 + .10*spec.Curl) * grain
	switch population {
	case scalpFringe:
		return hair.Guide{
			Lead: maths.V(.23, .053, .055*grain),
			Bend: maths.V(.59, .025, .22*pull+.080*grain),
			Tip:  maths.V(.90, -.080, .30*pull+.080*grain),
		}
	case scalpMargin:
		return hair.Guide{
			Lead: maths.V(.22, .77*lift, .033*grain),
			Bend: maths.V(.59, -.044+.23*volume, .40*pull+wave),
			Tip:  maths.V(.92, -.12+.12*volume, .70*pull+.060*grain),
		}
	case scalpLoose:
		return hair.Guide{
			Lead: maths.V(.19, lift+.07, .056*grain),
			Bend: maths.V(.56, .29+.19*volume, .085*grain),
			Tip:  maths.V(.84, .063+.22*volume, .15*grain),
		}
	default:
		return hair.Guide{
			Lead: maths.V(.19, lift, .024*grain+.085*pull),
			Bend: maths.V(.61, -.067+.39*volume+.070*lock*volume, .70*pull+wave),
			Tip:  maths.V(.92, -.17+.18*volume+.047*lock, pull+.056*grain),
		}
	}
}

func (spec ScalpHair) guideWave(length, volume, grain, lock float64) hair.GuideWave {
	width := .11 * spec.Curl * (.25 + .75*volume)
	return hair.GuideWave{
		Width: width, Height: .38 * width,
		Turns: min(2.4, length/spec.Wavelength),
		Phase: .61 + 1.6*lock + .45*grain,
	}
}
