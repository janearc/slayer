package workspace

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/job"
)

func inheritedStillCameraPath(path, source string) (string, time.Duration, error) {
	wanted, err := job.ReadSource(path, []byte(source))
	if err != nil {
		return "", 0, err
	}
	recipe, err := wanted.StillCameraRecipe()
	if err != nil {
		return "", 0, err
	}
	at, _ := wanted.Clock().At()
	// Slayer's --at override writes seconds through a float64. A scene
	// camera switch must keep exactly the same instant as the path recipe.
	if time.Duration(at.Seconds()*float64(time.Second)) != at {
		return "", 0, fmt.Errorf("camera path authoring cannot preserve this still time when switching cameras")
	}
	return recipe, at, nil
}

// inheritedStillWide samples the same first lens value as Slayer's still clock.
func inheritedStillWide(path, source, name string) (float64, error) {
	wanted, err := job.ReadSource(path, []byte(source))
	if err != nil {
		return 0, err
	}
	if name == "" || !wanted.Clock().Still() {
		return 0, fmt.Errorf("open a named still job to edit a scene with inherited zoom")
	}
	lens, _ := wanted.Camera()
	zoom, _ := lens.Zoom()
	return zoom.At(0)
}
