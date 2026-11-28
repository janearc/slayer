package manufacture

// The proportions of the pterosaur, in metres and radians. Each says
// where it came from, in one of three words.
//
//	documented: the number is written in this repository's own Hatz
//	  documents, which cite the paper named.
//	recalled: the number is the author's memory of a published figure.
//	  The paper is not on this machine and was not re-read. Check it.
//	authored: a choice made for the look of the animal, with no source.
//
// None is a measurement of one individual. The Hatz holotype is a
// fragmentary skull and a proximal humerus (Averianov 2014, PMC4141157).
// The wing chain's four bone lengths are not here: the rig keeps the
// ones in hatz_source_next.go, authored in the earlier study.

// Skull and neck.
const (
	// pterosaurSkullShareOfSpan is the skull's length as a share of the
	// wingspan. Documented: a skull of nearly 3 m against a span of at
	// least 12 m, extrapolated in the original description (Buffetaut
	// et al. 2002, https://pubmed.ncbi.nlm.nih.gov/12061403/, quoted in
	// doc/bioforms/hatzegopteryx/01-plan.md), a quarter.
	pterosaurSkullShareOfSpan = .25
	// pterosaurNeckLength is the neck from the chest to the occiput.
	// Documented: about 1.5 m as an illustrative control length, from
	// the neck biomechanics of a referred cervical (Naish and Witton
	// 2017, PMC5248582; doc/bioforms/hatzegopteryx/01-plan.md). Next to
	// a 2.3 m skull it reads short; the paper's figure is not on this
	// machine, so it is kept as written and flagged in the report.
	pterosaurNeckLength = 1.5
	// pterosaurNeckJoints is how many vertebral control points carry the
	// neck's spine. Authored.
	pterosaurNeckJoints = 9
	// pterosaurNeckBaseHalfWidth and the head end are the neck's half
	// width where it leaves the chest and where it meets the skull.
	// Authored, from a robust short neck (Naish and Witton 2017).
	pterosaurNeckBaseHalfWidth = .21
	pterosaurNeckHeadHalfWidth = .125
	// pterosaurNeckHeightShare is the neck's half height as a share of
	// its half width. Authored.
	pterosaurNeckHeightShare = 1.12
	// pterosaurVertebrae is the count of heavy cervical swellings drawn
	// along the neck. Recalled: azhdarchid necks have elongate
	// cervicals 3 to 7 (Witton and Naish 2008, PLoS ONE 3(5): e2271);
	// seven swellings are drawn.
	pterosaurVertebrae = 7
)

// Skull, in shares of the skull length from the occiput (x) and metres.
const (
	// pterosaurSkullBackWidth and the following are the skull's half
	// widths and half heights at its stations. Authored, after the
	// wide, robust skull and long toothless rostrum of azhdarchids
	// (Witton and Naish 2008; Naish and Witton 2017).
	pterosaurSkullBackWidth   = .17
	pterosaurSkullCheekWidth  = .205
	pterosaurSkullBackHeight  = .16
	pterosaurSkullCheekHeight = .20
	// pterosaurJawJoint is where the jaw hinges, as a share of the skull
	// length from the occiput. Authored.
	pterosaurJawJoint = .085
	// pterosaurBeakSheath is where the bare keratin beak begins, as a
	// share of the skull length. Authored.
	pterosaurBeakSheath = .34
	// pterosaurCrestStart, pterosaurCrestEnd and pterosaurCrestHeight
	// set the low blade crest on the skull's top. Authored: the
	// holotype has no crest preserved; some azhdarchids carry a low one.
	pterosaurCrestStart  = .10
	pterosaurCrestEnd    = .58
	pterosaurCrestHeight = .13
	// pterosaurEyeRadius is the eyeball's radius. Authored.
	pterosaurEyeRadius = .042
)

// Trunk.
const (
	// pterosaurTailEnd, pterosaurHipAt and pterosaurChestAt are the
	// trunk's stations along x in the flight pose. Authored: the old
	// study's trunk ran from x = -.93 to .18.
	pterosaurTailEnd = -1.15
	pterosaurHipAt   = -.72
	pterosaurChestAt = .10
	// pterosaurTrunkHalfWidth and pterosaurTrunkHalfHeight are the
	// trunk's half sizes at the shoulders. Authored.
	pterosaurTrunkHalfWidth  = .30
	pterosaurTrunkHalfHeight = .31
)

// Limbs.
const (
	// pterosaurThigh, pterosaurShank and pterosaurFoot are the hind
	// limb's segment lengths. Recalled: azhdarchid hind limbs are long,
	// the femur about the length of the humerus and the tibia longer
	// (Witton and Naish 2008). The humerus here is .85 m.
	pterosaurThigh = .75
	pterosaurShank = 1.0
	pterosaurFoot  = .35
	// pterosaurPteroidLength is the pteroid bone, which props the
	// forward membrane. Authored: .45 of the humerus. Palmer and Dyke
	// 2010 (PMC2842769) describe the pteroid; its length in Hatz is
	// unknown.
	pterosaurPteroidLength = .38
	// pterosaurStandLift is how high the trunk stands above the ground
	// in the all-fours pose. It follows from the leg and arm lengths:
	// a straight arm is .85 + 1.02 = 1.87 m and a hind limb 1.75 m.
	pterosaurStandLift = 1.75
)

// The membrane.
const (
	// pterosaurChordSlack is the extra length of each chord of the main
	// membrane over its distance in the glide pose, so the glide skin is
	// held with a slight sag and not stretched. Authored: 1 per cent
	// more length sags a 1.5 m chord by about a tenth of a metre.
	pterosaurChordSlack = .012
	// pterosaurEdgeSlack is the same for the free trailing edge.
	// Authored.
	pterosaurEdgeSlack = .004
	// pterosaurPleatBase and pterosaurPleatGain set how many pleats a
	// chord forms: the base plus the gain times the slack share up to
	// pterosaurPleatSaturation. Authored.
	pterosaurPleatBase       = .5
	pterosaurPleatGain       = 5.5
	pterosaurPleatSaturation = 1.5
	// pterosaurFibreEvery is how many chord lines apart the stiffening
	// fibres are drawn. Recalled: fibre layers run in different
	// directions in the membrane of Jeholopterus (Kellner et al. 2010,
	// PMC2842671); their spacing in Hatz is unknown.
	pterosaurFibreEvery = 4
)
