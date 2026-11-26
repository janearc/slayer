package cities

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
)

// Composition is a structure assembled from named, connected parts.
// All coordinates and sizes are metres in the composition's local frame.
type Composition struct {
	Version int               `yaml:"version"`
	Name    string            `yaml:"name"`
	Parts   []CompositionPart `yaml:"parts"`
	Roads   []CompositionRoad `yaml:"roads"`
}

// CompositionPart describes a box or a fan of stays between two boxes.
type CompositionPart struct {
	ID          string     `yaml:"id"`
	Type        string     `yaml:"type"`
	At          [3]float64 `yaml:"at"`
	Size        [3]float64 `yaml:"size"`
	Ground      bool       `yaml:"ground"`
	RestsOn     []string   `yaml:"rests-on"`
	HangsFrom   []string   `yaml:"hangs-from"`
	SupportedBy []string   `yaml:"supported-by"`
	Tower       string     `yaml:"tower"`
	Deck        string     `yaml:"deck"`
	TowerFrom   [3]float64 `yaml:"tower-from"`
	TowerTo     [3]float64 `yaml:"tower-to"`
	DeckFrom    [3]float64 `yaml:"deck-from"`
	DeckTo      [3]float64 `yaml:"deck-to"`
	Count       int        `yaml:"count"`
	Thickness   float64    `yaml:"thickness"`
	From        [3]float64 `yaml:"from"`
	To          [3]float64 `yaml:"to"`
	Width       float64    `yaml:"width"`
	Centre      [3]float64 `yaml:"centre"`
	Radius      float64    `yaml:"radius"`
	Turns       float64    `yaml:"turns"`
	Rise        float64    `yaml:"rise"`
	StartAngle  float64    `yaml:"start-angle"`
	Clearance   float64    `yaml:"clearance"`
	Segments    int        `yaml:"segments"`
	FromSupport string     `yaml:"from-support"`
	ToSupport   string     `yaml:"to-support"`
	On          string     `yaml:"on"`
	Face        string     `yaml:"face"`
	BayWidth    float64    `yaml:"bay-width"`
	LitShare    float64    `yaml:"lit-share"`
	Heat        float64    `yaml:"heat"`
	Glow        float64    `yaml:"glow"`
}

// CompositionRoad describes a straight road on a deck.
type CompositionRoad struct {
	ID    string     `yaml:"id"`
	Type  string     `yaml:"type"`
	On    string     `yaml:"on"`
	Next  string     `yaml:"next"`
	From  [3]float64 `yaml:"from"`
	To    [3]float64 `yaml:"to"`
	Width float64    `yaml:"width"`
}

const contactTolerance = 1e-6

func finiteNumber(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func finitePoint(p [3]float64) bool {
	return finiteNumber(p[0]) && finiteNumber(p[1]) && finiteNumber(p[2])
}
func isBox(p CompositionPart) bool {
	return p.Type == "body" || p.Type == "leg" || p.Type == "deck" || p.Type == "storey-band"
}
func (p CompositionPart) contains(q [3]float64) bool {
	for axis := range 3 {
		if q[axis] < p.At[axis]-contactTolerance || q[axis] > p.At[axis]+p.Size[axis]+contactTolerance {
			return false
		}
	}
	return true
}
func (p CompositionPart) surface(q [3]float64) bool {
	if !p.contains(q) {
		return false
	}
	for axis := range 3 {
		if math.Abs(q[axis]-p.At[axis]) <= contactTolerance || math.Abs(q[axis]-p.At[axis]-p.Size[axis]) <= contactTolerance {
			return true
		}
	}
	return false
}
func (p CompositionPart) restsOn(q CompositionPart) bool {
	if math.Abs(p.At[1]-q.At[1]-q.Size[1]) > contactTolerance {
		return false
	}
	for _, axis := range []int{0, 2} {
		if math.Min(p.At[axis]+p.Size[axis], q.At[axis]+q.Size[axis])-math.Max(p.At[axis], q.At[axis]) <= contactTolerance {
			return false
		}
	}
	return true
}
func interpolate(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0] + t*(b[0]-a[0]), a[1] + t*(b[1]-a[1]), a[2] + t*(b[2]-a[2])}
}

// Validate checks grounding, actual contacts, stay attachments and roads.
// It is not an engineering calculation of load or stability.
func (c Composition) Validate() error {
	if c.Version != 1 || c.Name == "" {
		return fmt.Errorf("composition: version 1 and a name are required")
	}
	if len(c.Parts) == 0 || len(c.Parts) > 10000 || len(c.Roads) > 10000 {
		return fmt.Errorf("composition %s: invalid part or road count", c.Name)
	}
	parts := make(map[string]CompositionPart, len(c.Parts))
	pieces := 0
	for _, p := range c.Parts {
		if p.ID == "" {
			return fmt.Errorf("composition %s: part needs an id", c.Name)
		}
		if _, ok := parts[p.ID]; ok {
			return fmt.Errorf("composition %s: duplicate part %s", c.Name, p.ID)
		}
		parts[p.ID] = p
		pieces++
		if p.Type == "storey-band" || p.Type == "stay-fan" {
			pieces += max(0, p.Count-1)
		}
		if pieces > 100000 {
			return fmt.Errorf("composition %s: too many generated parts", c.Name)
		}
		if isBox(p) {
			if !finitePoint(p.At) || !finitePoint(p.Size) || p.Size[0] <= 0 || p.Size[1] <= 0 || p.Size[2] <= 0 {
				return fmt.Errorf("part %s: invalid box", p.ID)
			}
			if p.Type == "storey-band" && (p.Count < 1 || p.Count > 10000) {
				return fmt.Errorf("part %s: invalid storey count", p.ID)
			}
			for axis := range 3 {
				if !finiteNumber(p.At[axis] + p.Size[axis]) {
					return fmt.Errorf("part %s: box extent is not finite", p.ID)
				}
			}
			if p.Ground && math.Abs(p.At[1]) > contactTolerance {
				return fmt.Errorf("part %s: does not meet ground", p.ID)
			}
		} else if p.Type == "helix" {
			if err := p.helixCheck(); err != nil {
				return err
			}
			pieces += p.Segments*8 + 4
			if pieces > 100000 {
				return fmt.Errorf("composition %s: too many generated parts", c.Name)
			}
		} else if p.Type == "ramp" {
			if p.Ground || !finitePoint(p.From) || !finitePoint(p.To) || !finiteNumber(p.Width) || p.Width <= 0 || !finiteNumber(p.Thickness) || p.Thickness <= 0 || !finiteNumber(math.Hypot(p.To[0]-p.From[0], p.To[2]-p.From[2])) || math.Hypot(p.To[0]-p.From[0], p.To[2]-p.From[2]) <= 0 {
				return fmt.Errorf("part %s: invalid ramp", p.ID)
			}
		} else if p.Type == "window-ribbon" {
			if p.Ground || !finitePoint(p.At) || !finitePoint(p.Size) || p.Size[0] <= 0 || p.Size[1] <= 0 || p.Size[2] <= 0 || !finiteNumber(p.BayWidth) || p.BayWidth <= 0 || !finiteNumber(p.LitShare) || p.LitShare < 0 || p.LitShare > 1 || !finiteNumber(p.Heat) || p.Heat <= 0 || !finiteNumber(p.Glow) || p.Glow < 0 {
				return fmt.Errorf("part %s: invalid window ribbon", p.ID)
			}
		} else if p.Type == "stay-fan" {
			if p.Ground || p.Count < 2 || p.Count > 10000 || !finiteNumber(p.Thickness) || p.Thickness <= 0 || !finitePoint(p.TowerFrom) || !finitePoint(p.TowerTo) || !finitePoint(p.DeckFrom) || !finitePoint(p.DeckTo) {
				return fmt.Errorf("part %s: invalid stay fan", p.ID)
			}
		} else {
			return fmt.Errorf("part %s: unknown type %q", p.ID, p.Type)
		}
	}
	for _, p := range c.Parts {
		if p.Type == "helix" {
			for index, id := range []string{p.FromSupport, p.ToSupport} {
				host, ok := parts[id]
				if !ok || !isBox(host) {
					return fmt.Errorf("part %s: missing helix support %s", p.ID, id)
				}
				t := float64(index)
				for _, offset := range []float64{-p.Width / 2, 0, p.Width / 2} {
					if !host.surface(p.helixPoint(t, offset)) {
						return fmt.Errorf("part %s: helix width misses support %s", p.ID, id)
					}
				}
			}
		}
		if p.Type == "ramp" {
			a, ok := parts[p.FromSupport]
			if !ok || !isBox(a) || !a.surface(p.From) {
				return fmt.Errorf("part %s: ramp start misses %s", p.ID, p.FromSupport)
			}
			b, ok := parts[p.ToSupport]
			if !ok || !isBox(b) || !b.surface(p.To) {
				return fmt.Errorf("part %s: ramp end misses %s", p.ID, p.ToSupport)
			}
			run := math.Hypot(p.To[0]-p.From[0], p.To[2]-p.From[2])
			dx, dz := (p.To[0]-p.From[0])/run, (p.To[2]-p.From[2])/run
			for _, sign := range []float64{-1, 1} {
				start, end := p.From, p.To
				start[0] += sign * -dz * p.Width / 2
				start[2] += sign * dx * p.Width / 2
				end[0] += sign * -dz * p.Width / 2
				end[2] += sign * dx * p.Width / 2
				if !a.surface(start) || !b.surface(end) {
					return fmt.Errorf("part %s: ramp width misses supports", p.ID)
				}
			}
		}
		if p.Type == "window-ribbon" {
			host, ok := parts[p.On]
			if !ok || !isBox(host) {
				return fmt.Errorf("part %s: missing window host %s", p.ID, p.On)
			}
			axis, long, side := 2, 0, false
			switch p.Face {
			case "north":
			case "south":
				side = true
			case "west":
				axis, long = 0, 2
			case "east":
				axis, long, side = 0, 2, true
			default:
				return fmt.Errorf("part %s: unknown face", p.ID)
			}
			face := host.At[axis]
			ribbonFace := p.At[axis] + p.Size[axis]
			if side {
				face = host.At[axis] + host.Size[axis]
				ribbonFace = p.At[axis]
			}
			count := math.Ceil(p.Size[long] / p.BayWidth)
			pieces += int(math.Min(count, 10001)) - 1
			if pieces > 100000 {
				return fmt.Errorf("composition %s: too many generated parts", c.Name)
			}
			if math.Abs(face-ribbonFace) > contactTolerance || p.At[long] < host.At[long]-contactTolerance || p.At[long]+p.Size[long] > host.At[long]+host.Size[long]+contactTolerance || p.At[1] < host.At[1]-contactTolerance || p.At[1]+p.Size[1] > host.At[1]+host.Size[1]+contactTolerance || !finiteNumber(count) || count < 1 || count > 10000 {
				return fmt.Errorf("part %s: ribbon does not fit face", p.ID)
			}
		}
		if p.Type == "stay-fan" {
			tower, ok := parts[p.Tower]
			if !ok || !isBox(tower) {
				return fmt.Errorf("part %s: missing tower %s", p.ID, p.Tower)
			}
			deck, ok := parts[p.Deck]
			if !ok || deck.Type != "deck" {
				return fmt.Errorf("part %s: missing deck %s", p.ID, p.Deck)
			}
			for i := 0; i < p.Count; i++ {
				t := float64(i) / float64(p.Count-1)
				a, b := interpolate(p.TowerFrom, p.TowerTo, t), interpolate(p.DeckFrom, p.DeckTo, t)
				if !tower.surface(a) || !deck.surface(b) || a == b {
					return fmt.Errorf("part %s: stay %d does not meet both parts", p.ID, i)
				}
			}
		}
		for _, id := range p.RestsOn {
			q, ok := parts[id]
			if !ok || !isBox(p) || !isBox(q) || !p.restsOn(q) {
				return fmt.Errorf("part %s: invalid contact with %s", p.ID, id)
			}
		}
		for _, id := range p.HangsFrom {
			q, ok := parts[id]
			if !ok || !isBox(p) || !isBox(q) || !q.restsOn(p) {
				return fmt.Errorf("part %s: invalid hanging contact with %s", p.ID, id)
			}
		}
		for _, id := range p.SupportedBy {
			q, ok := parts[id]
			if !ok || q.Type != "stay-fan" || q.Deck != p.ID {
				return fmt.Errorf("part %s: invalid support %s", p.ID, id)
			}
		}
	}
	state := make(map[string]uint8, len(parts))
	var grounded func(string) bool
	grounded = func(id string) bool {
		if state[id] == 1 {
			return false
		}
		if state[id] == 2 {
			return true
		}
		if state[id] == 3 {
			return false
		}
		p := parts[id]
		state[id] = 1
		if p.Ground {
			state[id] = 2
			return true
		}
		deps := append(append(append([]string{}, p.RestsOn...), p.HangsFrom...), p.SupportedBy...)
		if p.Type == "ramp" || p.Type == "helix" {
			deps = []string{p.FromSupport, p.ToSupport}
		}
		if p.Type == "stay-fan" {
			deps = []string{p.Tower}
		}
		if p.Type == "window-ribbon" {
			deps = []string{p.On}
		}
		if len(deps) == 0 {
			state[id] = 3
			return false
		}
		for _, dep := range deps {
			if !grounded(dep) {
				state[id] = 3
				return false
			}
		}
		state[id] = 2
		return true
	}
	for _, p := range c.Parts {
		if !grounded(p.ID) {
			return fmt.Errorf("part %s: no grounded support", p.ID)
		}
	}
	for _, road := range c.Roads {
		if road.ID == "" {
			return fmt.Errorf("composition %s: road needs an id", c.Name)
		}
		if _, ok := parts[road.ID]; ok {
			return fmt.Errorf("road %s: duplicate id", road.ID)
		}
		parts[road.ID] = CompositionPart{}
		deck, ok := parts[road.On]
		if !ok || (deck.Type != "deck" && deck.Type != "ramp" && deck.Type != "helix") || (road.Type != "straight" && road.Type != "helix") || (road.Type == "helix") != (deck.Type == "helix") || !finitePoint(road.From) || !finitePoint(road.To) || !finiteNumber(road.Width) || road.Width <= 0 {
			return fmt.Errorf("road %s: invalid road or deck", road.ID)
		}
		if deck.Type == "helix" {
			start, end := deck.helixPoint(0, 0), deck.helixPoint(1, 0)
			same := func(a, b [3]float64) bool {
				for i := range 3 {
					if math.Abs(a[i]-b[i]) > contactTolerance {
						return false
					}
				}
				return true
			}
			if !((same(road.From, start) && same(road.To, end)) || (same(road.From, end) && same(road.To, start))) || road.Width > deck.Width {
				return fmt.Errorf("road %s: does not follow helix", road.ID)
			}
			continue
		}
		run := math.Hypot(road.To[0]-road.From[0], road.To[2]-road.From[2])
		if !finiteNumber(run) || run <= 0 || !finiteNumber(road.To[1]-road.From[1]) || math.Abs(road.To[1]-road.From[1])/run > 0.04 {
			return fmt.Errorf("road %s: grade exceeds 4%% or has no run", road.ID)
		}
		if deck.Type == "ramp" {
			if !((road.From == deck.From && road.To == deck.To) || (road.From == deck.To && road.To == deck.From)) || road.Width > deck.Width {
				return fmt.Errorf("road %s: does not follow ramp", road.ID)
			}
			continue
		}
		if !deck.contains(road.From) || !deck.contains(road.To) || math.Abs(road.From[1]-deck.At[1]-deck.Size[1]) > contactTolerance || math.Abs(road.To[1]-deck.At[1]-deck.Size[1]) > contactTolerance {
			return fmt.Errorf("road %s: not on deck", road.ID)
		}
		// A width extends perpendicular to the road, and all four corners must fit.
		dx, dz := (road.To[0]-road.From[0])/run, (road.To[2]-road.From[2])/run
		for _, point := range [][3]float64{road.From, road.To} {
			for _, sign := range []float64{-1, 1} {
				corner := point
				corner[0] += sign * -dz * road.Width / 2
				corner[2] += sign * dx * road.Width / 2
				if !deck.contains(corner) {
					return fmt.Errorf("road %s: width leaves deck", road.ID)
				}
			}
		}
	}
	roads := make(map[string]CompositionRoad, len(c.Roads))
	for _, road := range c.Roads {
		roads[road.ID] = road
	}
	incoming := make(map[string]bool, len(roads))
	for _, road := range c.Roads {
		if road.Next == "" {
			continue
		}
		next, ok := roads[road.Next]
		if !ok {
			return fmt.Errorf("road %s: missing next road %s", road.ID, road.Next)
		}
		if incoming[road.Next] {
			return fmt.Errorf("road %s: multiple incoming segments", road.Next)
		}
		incoming[road.Next] = true
		for axis := range 3 {
			if math.Abs(road.To[axis]-next.From[axis]) > contactTolerance {
				return fmt.Errorf("road %s: disconnected from %s", road.ID, road.Next)
			}
		}
		a := roadHeading(road, parts[road.On], true)
		b := roadHeading(next, parts[next.On], false)
		// Compare the actual left and right edge positions, not only the
		// headings: a tiny angular mismatch makes a large gap on a wide road.
		edgeGap := math.Hypot(road.To[0]-next.From[0]-(a[1]*road.Width-b[1]*next.Width)/2, road.To[2]-next.From[2]+(a[0]*road.Width-b[0]*next.Width)/2)
		otherGap := math.Hypot(road.To[0]-next.From[0]+(a[1]*road.Width-b[1]*next.Width)/2, road.To[2]-next.From[2]-(a[0]*road.Width-b[0]*next.Width)/2)
		if edgeGap > contactTolerance || otherGap > contactTolerance || a[0]*b[0]+a[1]*b[1] < 1-1e-9 {
			return fmt.Errorf("road %s: direction changes at %s", road.ID, road.Next)
		}
		if math.Abs(road.Width-next.Width) > contactTolerance {
			return fmt.Errorf("road %s: width changes at %s", road.ID, road.Next)
		}
	}
	visited := make(map[string]bool, len(roads))
	for _, road := range c.Roads {
		if incoming[road.ID] {
			continue
		}
		id := road.ID
		for id != "" {
			if visited[id] {
				return fmt.Errorf("road %s: cycle", id)
			}
			visited[id] = true
			id = roads[id].Next
		}
	}
	if len(visited) != len(roads) {
		return fmt.Errorf("composition %s: road cycle", c.Name)
	}
	return nil
}

// Design makes the checked parts into existing city elements.
func (c Composition) Design() (Design, error) { return c.DesignSeed(0) }

// DesignSeed makes a checked design, using seed and part ID for window lights.
func (c Composition) DesignSeed(seed Seed) (Design, error) {
	if err := c.Validate(); err != nil {
		return Design{}, err
	}
	look, err := material(roleBody)
	if err != nil {
		return Design{}, err
	}
	d := Design{Name: c.Name}
	for _, p := range c.Parts {
		if p.Type == "helix" {
			d.Elements = append(d.Elements, put(p.ID, helixForm{p}, look, maths.V(0, 0, 0)))
			continue
		}
		if p.Type == "ramp" {
			a, b := maths.V(p.From[0], p.From[1], p.From[2]), maths.V(p.To[0], p.To[1], p.To[2])
			along := b.Sub(a)
			axis := along.Unit()
			side := maths.V(-axis.Z, 0, axis.X).Unit()
			up := side.Cross(axis)
			origin := a.Sub(side.Scale(p.Width / 2)).Sub(up.Scale(p.Thickness))
			d.Elements = append(d.Elements, placedElement{Element: solid{Name: p.ID, Form: Box{along.Len(), p.Thickness, p.Width}, Material: look}, Place: maths.NewSubspace(origin, maths.Orientation{X: axis, Y: up, Z: side}, 0)})
			continue
		}
		if p.Type == "window-ribbon" {
			dark, err := material(roleGlazing)
			if err != nil {
				return Design{}, err
			}
			lit := *dark
			lit.Name = "cities composition lit window"
			lit.Glow = swatches.Light(swatch.Planckian(p.Heat)).Scale(p.Glow)
			axis := 0
			if p.Face == "east" || p.Face == "west" {
				axis = 2
			}
			n := int(math.Ceil(p.Size[axis] / p.BayWidth))
			hash := fnv.New64a()
			_, _ = hash.Write([]byte(p.ID))
			base := hash.Sum64() ^ uint64(seed)
			for i := 0; i < n; i++ {
				size := p.Size
				size[axis] = math.Min(p.BayWidth, p.Size[axis]-float64(i)*p.BayWidth)
				at := p.At
				at[axis] += float64(i) * p.BayWidth
				x := base + uint64(i) + 0x9e3779b97f4a7c15
				x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
				x = (x ^ (x >> 27)) * 0x94d049bb133111eb
				x ^= x >> 31
				look := dark
				if float64(x>>11)/float64(uint64(1)<<53) < p.LitShare {
					look = &lit
				}
				d.Elements = append(d.Elements, put(p.ID, Box{size[0], size[1], size[2]}, look, maths.V(at[0], at[1], at[2])))
			}
			continue
		}
		if p.Type == "storey-band" {
			height := p.Size[1] / float64(p.Count)
			for floor := range p.Count {
				d.Elements = append(d.Elements, put(p.ID, Box{p.Size[0], height, p.Size[2]}, look, maths.V(p.At[0], p.At[1]+float64(floor)*height, p.At[2])))
			}
			continue
		}
		if isBox(p) {
			d.Elements = append(d.Elements, put(p.ID, Box{p.Size[0], p.Size[1], p.Size[2]}, look, maths.V(p.At[0], p.At[1], p.At[2])))
			continue
		}
		for i := 0; i < p.Count; i++ {
			t := float64(i) / float64(p.Count-1)
			a, b := interpolate(p.TowerFrom, p.TowerTo, t), interpolate(p.DeckFrom, p.DeckTo, t)
			d.Elements = append(d.Elements, chord(p.ID, maths.V(a[0], a[1], a[2]), maths.V(b[0], b[1], b[2]), p.Thickness, look))
		}
	}
	return d, nil
}
