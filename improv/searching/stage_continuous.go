package searching

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"os"
)

// fieldMember retains one absolute position. Rebase every member by the
// same factor; changing scale never resets the camera or rearranges a field.
type fieldMember struct {
	part  *thing.Thing
	pose  maths.Subspace
	glow  maths.Light
	layer int
	phase float64
}
type continuousField struct {
	members     []fieldMember
	centre      maths.Vec
	reference   *flowSet
	plan        *FlowPlan
	fixed       map[*thing.Thing]maths.Subspace
	volumePoses []maths.Subspace
}

// fieldDistance eases one logarithmic retreat, ending before recognition.
// The displayed larger geography is authored; only nearby points are Gaia.
func fieldDistance(at float64) float64 {
	u := smooth((at - 52*240.0/66) / (7.5 * 240.0 / 66))
	return math.Exp(math.Log(62e6) * u)
}

// readStageCatalogue checks the same bounded, pinned data product as the
// standalone catalogue study, without claiming a second edit interval.
func readStageCatalogue(p *CataloguePlan) (*catalogue, error) {
	f, err := os.Open(p.File)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > 8*1024*1024 {
		return nil, fmt.Errorf("catalogue exceeds proof budget")
	}
	raw, err := os.ReadFile(p.File)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != p.SHA {
		return nil, fmt.Errorf("catalogue hash differs")
	}
	var data catalogue
	if err = json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, data.check()
}
