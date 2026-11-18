package searching

import (
	"fmt"
	"image"
	"io"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/scene/thing"
)

// readFilmJob reads the one screenplay file that a heard-film job names.
func readFilmJob(with map[string]any) (*Edit, error) {
	if len(with) != 1 {
		return nil, fmt.Errorf("heard-film needs exactly one plan")
	}
	path, ok := with["plan"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("heard-film needs an explicit screenplay file")
	}
	return Read(path)
}

// registerSlayerFilm supplies only a world and camera clock. The ordinary
// Slayer session owns Metal, frame retention, print, sound and movie output.
func registerSlayerFilm() error {
	return named.RegisterAuthored(named.Authored{
		Name: "heard-film", About: "The checked heard. screenplay through the ordinary Slayer session", Unit: "m", FastLighting: true,
		Width: 480, Height: 270, Rays: 1, Wide: 65 * maths.Degree, Tone: colour.Tone{Exposure: 1.7, Knee: .7, Top: 8, Bloom: .22},
		Check: func(with map[string]any) error { _, err := readFilmJob(with); return err },
		Lens: func(with map[string]any) (func(time.Duration) float64, error) {
			edit, err := readFilmJob(with)
			if err != nil {
				return nil, err
			}
			return func(at time.Duration) float64 { return screenplayWide(edit, at.Seconds()) * maths.Degree }, nil
		},
		Build: func(with map[string]any, said io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error) {
			edit, err := readFilmJob(with)
			if err != nil {
				return nil, maths.Subspace{}, nil, err
			}
			player := &slayerFilm{edit: edit, world: thing.NewWorld(), active: -1, said: said}
			eye, err := player.at(0)
			return player.world, eye, player.at, err
		},
	})
}

// slayerFilm retains only the currently occupied stage. A cut replaces scene
// content at its musical boundary; camera and actor clocks remain absolute.
type slayerFilm struct {
	edit   *Edit
	world  *thing.World
	active int
	stage  *heardStage
	built  *named.Built
	said   io.Writer
	panel  *thing.Material
}

// at returns the eye for the film clock, switching stages when the cut moves.
func (film *slayerFilm) at(clock time.Duration) (maths.Subspace, error) {
	at := clock.Seconds()
	index := film.edit.shot(at)
	if index < 0 {
		return maths.Subspace{}, fmt.Errorf("heard-film has no shot at %.9g seconds", at)
	}
	shot := film.edit.Plan.Shots[index]
	span := film.edit.ShotSpans[index]
	if index != film.active {
		film.stage, film.built, film.panel = nil, nil, nil
		if shot.Scene == "heard" {
			stage, err := newHeardStage(shot.Stage, film.edit.Plan.Music)
			if err != nil {
				return maths.Subspace{}, err
			}
			film.stage = stage
			if shot.Stage.Kind == "listening" {
				if err = film.listeningPanel(); err != nil {
					return maths.Subspace{}, err
				}
			}
			*film.world = *stage.world
		} else {
			scene, err := named.Get(shot.Scene)
			if err != nil {
				return maths.Subspace{}, err
			}
			film.built, err = scene.Build(shot.With, film.said)
			if err != nil {
				return maths.Subspace{}, err
			}
			*film.world = *film.built.World()
		}
		film.active = index
		fmt.Fprintf(film.said, "heard-film shot %d/%d: %s, %.3f to %.3f s\n", index+1, len(film.edit.Plan.Shots), shot.Caption, span[0], span[1])
	}
	if film.stage != nil {
		if shot.Stage.Kind == "listening" {
			film.panel.Shine = spectrumPanel{film.stage.listening(480, 270, at), shot.Stage.Wide}
			return maths.NewSubspace(maths.V(0, 0, 10), maths.Identity, 1), nil
		}
		u := math.Max(0, math.Min(1, (at-span[0])/(span[1]-span[0])))
		eye, err := film.stage.update(shot.Stage, State{At: at, Beat: film.edit.Grid.BeatAt(at)}, u)
		if err != nil {
			return maths.Subspace{}, err
		}
		// The film uses one declared print; preserve each authored stage's field of view.
		*film.world = *film.stage.world
		return eye, nil
	}
	local := time.Duration((at - span[0]) * shot.TimeScale * float64(time.Second))
	if err := film.built.Advance(local); err != nil {
		return maths.Subspace{}, err
	}
	*film.world = *film.built.World()
	eye := film.built.Eye()
	if shot.Orbit != 0 {
		turn := maths.About(maths.YAxis, shot.Orbit*maths.Degree*(at-span[0])/(span[1]-span[0]))
		eye.Origin = turn.Out(eye.Origin)
		eye.Orientation = turn.Then(eye.Orientation)
	}
	return eye, nil
}

// listeningPanel makes the measured spectrogram an emissive scene surface.
// The data and title come from the existing WAV analysis, not invented signal.
func (film *slayerFilm) listeningPanel() error {
	halfW := 10 * math.Tan(screenplayWide(film.edit, 0)*maths.Degree/2)
	halfH := halfW * 9 / 16
	plate, err := mesh.New([]maths.Vec{maths.V(-halfW, -halfH, 0), maths.V(halfW, -halfH, 0), maths.V(halfW, halfH, 0), maths.V(-halfW, halfH, 0)}, [][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
	if err != nil {
		return err
	}
	film.panel = &thing.Material{Name: "measured heard soundtrack spectrogram", Shine: spectrumPanel{film.stage.listening(480, 270, 0), screenplayWide(film.edit, 0)}}
	film.stage.world.Add(thing.Thing{Shape: plate, Material: film.panel})
	return nil
}

type spectrumPanel struct {
	image *image.RGBA
	wide  float64
}

// At returns the light the spectrogram shows at a point on the panel.
func (panel spectrumPanel) At(p maths.Vec) maths.Light {
	halfW := 10 * math.Tan(panel.wide*maths.Degree/2)
	halfH := halfW * 9 / 16
	x := min(479, max(0, int((p.X/halfW+1)*240)))
	y := min(269, max(0, int((1-p.Y/halfH)*135)))
	pixel := panel.image.RGBAAt(x, y)
	linear := func(channel uint8) float64 {
		s := float64(channel) / 255
		if s <= .04045 {
			return s / 12.92
		}
		return math.Pow((s+.055)/1.055, 2.4)
	}
	return maths.RGB(linear(pixel.R), linear(pixel.G), linear(pixel.B))
}

// screenplayWide uses the same half-open musical cut as scene selection.
func screenplayWide(edit *Edit, at float64) float64 {
	index := edit.shot(at)
	if index < 0 {
		return 65
	}
	shot := edit.Plan.Shots[index]
	if shot.Stage != nil {
		return shot.Stage.Wide
	}
	return shot.Wide
}
