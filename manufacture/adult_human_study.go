package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/materials/fibre"
	"github.com/janearc/slayer/materials/tissue"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// adultHumanStudy places a reference adult in an authored studio pose.
// Its seed selects hair and pigment on the rest skin.
type adultHumanStudy struct {
	male, reaching, bareScalp bool
	widerStance               bool
	fibreScalp, bodyHair      bool
	facialHair, vellusHair    bool
	mediumScalp, skinMarks    bool
	scarRake                  bool
	thinSkin, auricles        bool
	crownLift                 float64
	scalpField                bool
	fieldWisps                bool
}

type posedHumanStudy struct {
	body *primates.Human
	pose *primates.HumanPose
}

func (row adultHumanStudy) body(from seed) (*primates.Human, error) {
	spec := primates.AdultHuman()
	if row.male {
		spec = primates.AdultMaleHuman().WithSkinTone(primates.SkinTone{
			Melanin:   .57,
			Vascular:  .49,
			Undertone: primates.Chroma{Gold: .16, Olive: .10},
		})
	}
	if row.skinMarks {
		marks, err := adultSkinMarks(from, row.scarRake)
		if err != nil {
			return nil, err
		}
		spec = spec.WithPaint(marks)
	}
	if row.auricles {
		spec = spec.WithAuricles(primates.AuricleForm{Asymmetry: .025})
	}
	if row.thinSkin {
		// The study uses illustrative layers in metres. The shorter
		// auricle cutoff applies across the thinner pinna walls.
		outer := tissue.Layer{Thickness: .001, Absorption: maths.RGB(85, 145, 220),
			Backscatter: maths.RGB(230, 300, 390)}
		inner := tissue.Layer{Thickness: .020, Absorption: maths.RGB(30, 65, 110),
			Backscatter: maths.RGB(130, 205, 290)}
		if row.auricles {
			inner.Thickness = .007
		}
		profile, err := thing.NewThinTissue([]tissue.Layer{outer, inner, outer}, .20)
		if err != nil {
			return nil, err
		}
		spec = spec.WithSkinTransmission(profile)
	}
	return primates.NewHuman(spec.WithSkinMottleSeed(from.number).
		WithSkinDiffusion(primates.AdultSkinDiffusion()))
}

func adultReachPose() skeleton.Pose {
	return skeleton.Pose{}.With("thorax", skeleton.NewTurn(maths.YAxis, .14)).
		With("head", skeleton.NewTurn(maths.YAxis, -.10)).
		With("left-clavicle", skeleton.NewTurn(maths.ZAxis, .10)).
		With("left-upper-arm", skeleton.NewTurn(maths.ZAxis, .40)).
		With("left-forearm", skeleton.NewTurn(maths.XAxis, -.25))
}

func (row adultHumanStudy) posed(from seed) (posedHumanStudy, error) {
	body, err := row.body(from)
	if err != nil {
		return posedHumanStudy{}, err
	}
	var pose skeleton.Pose
	if row.reaching {
		pose = adultReachPose()
	}
	if row.widerStance {
		const abduction = math.Pi / 36
		pose = pose.With("left-thigh", skeleton.NewTurn(maths.ZAxis, abduction)).
			With("right-thigh", skeleton.NewTurn(maths.ZAxis, -abduction))
	}
	posed, err := body.Pose(pose)
	return posedHumanStudy{body: body, pose: posed}, err
}

func (row adultHumanStudy) made(from seed) (good, error) {
	return row.withGaze(from, primates.Gaze{})
}

func (row adultHumanStudy) withGaze(from seed, gaze primates.Gaze) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	study, err := row.posed(from)
	if err != nil {
		return good{}, err
	}
	groom, err := row.groom(study.body, from)
	if err != nil {
		return good{}, err
	}
	return adultHumanFrame(study.pose, gaze, groom)
}

func (row adultHumanStudy) groom(body *primates.Human, from seed) (*hair.Groom, error) {
	style, err := row.hairStyle(from)
	if err != nil || style == (primates.HumanHair{}) {
		return nil, err
	}
	return body.Groom(style)
}

func adultHumanFrame(pose *primates.HumanPose, gaze primates.Gaze, groom *hair.Groom) (good, error) {
	skin, err := pose.Surface(nil)
	if err != nil {
		return good{}, err
	}
	eyes, err := pose.Eyes(gaze)
	if err != nil {
		return good{}, err
	}
	// The opaque eye exterior carries a narrower diffuse response than
	// skin. Surface reflections and the iris reflectance retain their
	// ordinary per-hit evaluation.
	eyeResponse, err := thing.NewSubsurface(thing.SubsurfaceProfile{
		Radius: [3]float64{.00115, .00086, .00066},
		Share:  .32,
	})
	if err != nil {
		return good{}, err
	}
	for i := range eyes {
		material := *eyes[i].Material
		material.Subsurface = eyeResponse
		eyes[i].Material = &material
	}
	parts := []thing.Thing{skin, eyes[0], eyes[1]}
	if groom == nil {
		return good{native: parts}, nil
	}
	fibres, err := groom.At(pose.Points())
	if err != nil {
		return good{}, err
	}
	return good{native: append(parts, fibres...)}, nil
}

const longestStudyStrand = .058

func (row adultHumanStudy) extent(from seed) (maths.Box, error) {
	study, err := row.posed(from)
	if err != nil {
		return maths.Box{}, err
	}
	points := study.pose.Points()
	bounds := geometry.GrowBox(maths.Box{Min: points[0], Max: points[0]}, points[1:])
	// Allow for shaft radius and the sampled scalp clearance.
	reach := longestStudyStrand + .0005
	if row.mediumScalp {
		reach = .32
	}
	margin := maths.V(reach, reach, reach)
	return maths.Box{Min: bounds.Min.Sub(margin), Max: bounds.Max.Add(margin)}, nil
}

func (row adultHumanStudy) scalpMaterial() (*thing.Material, error) {
	if !row.male && !row.fibreScalp {
		return &thing.Material{
			Name: "short scalp hair", Colour: maths.RGB(.044, .025, .018),
			Gloss: .11, Index: 1.54, Satin: .57,
		}, nil
	}
	model, err := fibre.New(fibre.Optics{
		Index: 1.55, Absorption: maths.RGB(.72, 1.19, 1.77),
		Longitudinal: .30, Azimuthal: .35, Cuticle: .04,
	})
	if err != nil {
		return nil, err
	}
	return &thing.Material{Name: "short scalp fibres", Fibre: model}, nil
}
