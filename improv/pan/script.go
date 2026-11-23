package pan

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/job"
	yaml "gopkg.in/yaml.v3"
)

// script is the YAML a cut is read from.
type script struct {
	Size  sizeWire   `yaml:"size"`
	FPS   int        `yaml:"fps"`
	Sound soundWire  `yaml:"sound"`
	Title *titleWire `yaml:"title"`
	Shots []shotWire `yaml:"shots"`
}

// sizeWire is a preset's name or a width and a height.
type sizeWire struct {
	Preset        string
	Width, Height int
}

// UnmarshalYAML reads a size written as a preset's name or as
// {width, height}.
func (size *sizeWire) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&size.Preset)
	}
	var given struct {
		Width  int `yaml:"width"`
		Height int `yaml:"height"`
	}
	err := node.Decode(&given)
	size.Width, size.Height = given.Width, given.Height
	return err
}

type soundWire struct {
	Score   string `yaml:"score"`
	Voicing string `yaml:"voicing"`
	Bars    []int  `yaml:"bars"`
}

type titleWire struct {
	Words string  `yaml:"words"`
	Until float64 `yaml:"until"`
	Rows  int     `yaml:"rows"`
}

type shotWire struct {
	Still string    `yaml:"still"`
	Bars  int       `yaml:"bars"`
	From  []float64 `yaml:"from"`
	To    []float64 `yaml:"to"`
	Fade  fadeWire  `yaml:"fade"`
}

type fadeWire struct {
	In  float64 `yaml:"in"`
	Out float64 `yaml:"out"`
}

// Cut is a film read from a script and checked: its size and rate, its
// shots with their stills read, the tune's piece and the bars it plays,
// and its title.
type Cut struct {
	size   image.Point
	rate   int
	shots  []shot
	piece  *cheeptune.Piece
	first  int // the first bar played, from one
	last   int // the last bar played
	title  *titleWire
	script string
}

// shot is one still, its span of bars, and its window's way and fades.
type shot struct {
	still    *image.RGBA
	size     image.Point // the film's
	name     string
	bars     int
	from, to image.Point // the window's corner at either end, in pixels
	fadeIn   time.Duration
	fadeOut  time.Duration
}

// Read is the cut a script says, checked: a size, a rate, a tune and the
// bars it plays, and shots whose bars add up to them, each with a still
// at least the film's size and a window's way inside it. Each refusal
// names what was wrong.
func Read(path string) (Cut, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Cut{}, err
	}
	var said script
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&said); err != nil {
		return Cut{}, fmt.Errorf("%s: %w", path, err)
	}
	made, err := said.cut(filepath.Dir(path))
	if err != nil {
		return Cut{}, fmt.Errorf("%s: %w", path, err)
	}
	made.script = path
	return made, nil
}

// cut is the script checked, its paths under base.
func (said script) cut(base string) (Cut, error) {
	size, err := said.Size.point()
	if err != nil {
		return Cut{}, err
	}
	if said.FPS < one {
		return Cut{}, fmt.Errorf("fps is %d; give frames a second", said.FPS)
	}
	made := Cut{size: size, rate: said.FPS, title: said.Title}
	if err := made.tune(said.Sound, base); err != nil {
		return Cut{}, err
	}
	bars := nothing
	for i, each := range said.Shots {
		taken, err := each.shot(base, size)
		if err != nil {
			return Cut{}, fmt.Errorf("shot %d: %w", i+one, err)
		}
		made.shots = append(made.shots, taken)
		bars += each.Bars
	}
	if played := made.last - made.first + one; bars != played {
		return Cut{}, fmt.Errorf("the shots take %d bars and the sound "+
			"plays %d (bars %d to %d); make them agree", bars, played,
			made.first, made.last)
	}
	return made, nil
}

// point is the film's size: a job preset's, or the width and height.
func (size sizeWire) point() (image.Point, error) {
	if size.Preset == "" {
		if size.Width < one || size.Height < one {
			return image.Point{}, errors.New("size says no width and " +
				"height, and no preset")
		}
		return image.Pt(size.Width, size.Height), nil
	}
	for _, each := range job.Presets() {
		if each.Name() == size.Preset {
			width, height := each.Size()
			return image.Pt(width, height), nil
		}
	}
	return image.Point{}, fmt.Errorf("size %q is no job preset",
		size.Preset)
}

// tune reads the piece and the bars it plays.
func (made *Cut) tune(said soundWire, base string) error {
	if said.Score == "" || said.Voicing == "" || len(said.Bars) != ends {
		return errors.New("sound needs a score, a voicing and bars: " +
			"[first, last]")
	}
	score := said.Score
	if score[:one] != "@" && !filepath.IsAbs(score) {
		score = filepath.Join(base, score)
	}
	piece, err := cheeptune.Open(score, said.Voicing)
	if err != nil {
		return err
	}
	made.piece, made.first, made.last = piece, said.Bars[0], said.Bars[1]
	if made.first < one || made.last < made.first ||
		made.last > piece.Score.Bars() {
		return fmt.Errorf("bars %d to %d: the score has bars 1 to %d",
			made.first, made.last, piece.Score.Bars())
	}
	return nil
}

// shot is the shot checked: its still read and at least the film's size,
// its bars at least one, its window's ends inside the still.
func (said shotWire) shot(base string, size image.Point) (shot, error) {
	if said.Bars < one {
		return shot{}, fmt.Errorf("%s takes %d bars; give one or more",
			said.Still, said.Bars)
	}
	path := said.Still
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	still, err := readStill(path)
	if err != nil {
		return shot{}, err
	}
	slack := still.Bounds().Size().Sub(size)
	if slack.X < nothing || slack.Y < nothing {
		return shot{}, fmt.Errorf("%s is %v and the film %v: a still is "+
			"at least the film's size, since nothing is scaled up",
			said.Still, still.Bounds().Size(), size)
	}
	from, err := corner(said.From, slack)
	if err != nil {
		return shot{}, fmt.Errorf("%s: from %w", said.Still, err)
	}
	to, err := corner(said.To, slack)
	if err != nil {
		return shot{}, fmt.Errorf("%s: to %w", said.Still, err)
	}
	return shot{still: still, size: size, name: said.Still, bars: said.Bars,
		from: from, to: to, fadeIn: seconds(said.Fade.In),
		fadeOut: seconds(said.Fade.Out)}, nil
}

// corner is a window's corner in pixels from its shares of the slack;
// none said is the middle.
func corner(shares []float64, slack image.Point) (image.Point, error) {
	if len(shares) == nothing {
		shares = []float64{half, half}
	}
	if len(shares) != ends {
		return image.Point{}, fmt.Errorf("is %v; give [across, down]",
			shares)
	}
	for _, share := range shares {
		if share < nothing || share > whole {
			return image.Point{}, fmt.Errorf("is %v; each share is 0 to 1",
				shares)
		}
	}
	return image.Pt(int(shares[0]*float64(slack.X)),
		int(shares[1]*float64(slack.Y))), nil
}

// readStill is a png's picture as RGBA.
func readStill(path string) (*image.RGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	made := image.NewRGBA(image.Rectangle{Max: picture.Bounds().Size()})
	draw.Draw(made, made.Bounds(), picture, picture.Bounds().Min, draw.Src)
	return made, nil
}

// seconds is a number of seconds as a duration.
func seconds(count float64) time.Duration {
	return time.Duration(count * float64(time.Second))
}
