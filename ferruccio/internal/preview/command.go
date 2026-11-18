package preview

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// arguments selects the renderer that owns this document format.
func (plan *Plan) arguments(tools Tools) ([]string, error) {
	if plan == nil {
		return nil, fmt.Errorf("no prepared preview")
	}
	if plan.Kind == "catalogue" {
		if plan.Profile.Full {
			return nil, fmt.Errorf("full render is not supported for catalogue plans")
		}
		if tools.Catalogue == "" {
			return nil, fmt.Errorf("configure the catalogue executable before rendering specimen settings")
		}
		if err := validSpecimen(plan.Specimen); err != nil {
			return nil, err
		}
		return []string{tools.Catalogue, "render", plan.Specimen, "--plan", plan.Input, "--out", filepath.Join(plan.Directory, "book")}, nil
	}
	if tools.Slayer == "" {
		return nil, fmt.Errorf("configure the Slayer executable before rendering")
	}
	profile, err := plan.Profile.resolved()
	if err != nil {
		return nil, err
	}
	arguments := []string{tools.Slayer, "draw", plan.Input, "--out", plan.Output}
	if !profile.Full {
		arguments = append(arguments, "--width", strconv.Itoa(profile.Width), "--height", strconv.Itoa(profile.Height), "--rays", "1")
	}
	arguments = append(arguments, "--quality", profile.quality())
	if profile.CPURenderer {
		arguments = append(arguments, "--renderer", "cpu")
	}
	return arguments, nil
}
