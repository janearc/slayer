package show

// state is the lights' state once every cue up to the moment has
// spoken: the defaults are a dark rig over the resting mist.
type state struct {
	sweepBars, duty, decay              float64
	spots, spotBars, search, searchBars float64
	beams, hot, hotter                  int
	palette                             []string
	strobes, pulse, floods              string
	split                               bool
	// the fading levels
	rig, mist, master, overhead, exposure, fire ramp
	moon, moonAlt                               ramp
	// fireFrom is where on the grid the fires' bursts are counted from,
	// in beats, or below nought when no cue has lit them
	fireFrom float64
	// moonPulse is how deep the moon pulses on the beat
	moonPulse float64
	// the lasers' figures and banks, each a list taken in turn so many
	// bars apiece from where on the grid its cue stands
	figures, banks         []string
	figureBars, banksBars  float64
	figuresFrom, banksFrom float64
	// the searchlights' moves, taken in turn the same way, and how deep
	// the beat strikes them
	searchMoves                     []string
	searchMoveBars, searchMovesFrom float64
	searchBeat                      float64
}

// resting is the state before any cue.
func resting() state {
	return state{beams: restBeams, sweepBars: restSweepBars,
		duty: restDuty, palette: []string{restPalette},
		strobes: restStrobes, hot: restHot, hotter: restHotter,
		pulse: restPulse, decay: restDecay, floods: restFloods,
		spotBars: restSpotBars, searchBars: restSearchBars,
		rig: held(0), mist: held(restMist), master: held(1),
		overhead: held(0), exposure: held(0), fire: held(0),
		fireFrom: noGrid, moon: held(1), moonAlt: held(setMoonAlt)}
}

// fades are the levels a cue glides over its fade: the field it names
// and the ramp it moves.
var fades = []struct {
	said  func(cue) *float64
	level func(*state) *ramp
}{
	{func(c cue) *float64 { return c.Rig }, func(s *state) *ramp { return &s.rig }},
	{func(c cue) *float64 { return c.Mist }, func(s *state) *ramp { return &s.mist }},
	{func(c cue) *float64 { return c.Master }, func(s *state) *ramp { return &s.master }},
	{func(c cue) *float64 { return c.Overhead }, func(s *state) *ramp { return &s.overhead }},
	{func(c cue) *float64 { return c.Exposure }, func(s *state) *ramp { return &s.exposure }},
	{func(c cue) *float64 { return c.Fire }, func(s *state) *ramp { return &s.fire }},
	{func(c cue) *float64 { return c.Moon }, func(s *state) *ramp { return &s.moon }},
	{func(c cue) *float64 { return c.MoonAlt }, func(s *state) *ramp { return &s.moonAlt }},
}

// numbers are the plain numbers a cue sets outright.
var numbers = []struct {
	said  func(cue) *float64
	value func(*state) *float64
}{
	{func(c cue) *float64 { return c.SweepBars }, func(s *state) *float64 { return &s.sweepBars }},
	{func(c cue) *float64 { return c.Duty }, func(s *state) *float64 { return &s.duty }},
	{func(c cue) *float64 { return c.Decay }, func(s *state) *float64 { return &s.decay }},
	{func(c cue) *float64 { return c.Spots }, func(s *state) *float64 { return &s.spots }},
	{func(c cue) *float64 { return c.SpotBars }, func(s *state) *float64 { return &s.spotBars }},
	{func(c cue) *float64 { return c.Search }, func(s *state) *float64 { return &s.search }},
	{func(c cue) *float64 { return c.SearchBars }, func(s *state) *float64 { return &s.searchBars }},
	{func(c cue) *float64 { return c.MoonPulse }, func(s *state) *float64 { return &s.moonPulse }},
	{func(c cue) *float64 { return c.FigureBars }, func(s *state) *float64 { return &s.figureBars }},
	{func(c cue) *float64 { return c.BanksBars }, func(s *state) *float64 { return &s.banksBars }},
	{func(c cue) *float64 { return c.SearchMoveBars }, func(s *state) *float64 { return &s.searchMoveBars }},
	{func(c cue) *float64 { return c.SearchBeat }, func(s *state) *float64 { return &s.searchBeat }},
}

// counts and words are the whole numbers and the words a cue sets.
var (
	counts = []struct {
		said  func(cue) *int
		value func(*state) *int
	}{
		{func(c cue) *int { return c.Beams }, func(s *state) *int { return &s.beams }},
		{func(c cue) *int { return c.Hot }, func(s *state) *int { return &s.hot }},
		{func(c cue) *int { return c.Hotter }, func(s *state) *int { return &s.hotter }},
	}
	words = []struct {
		said  func(cue) *string
		value func(*state) *string
	}{
		{func(c cue) *string { return c.Strobes }, func(s *state) *string { return &s.strobes }},
		{func(c cue) *string { return c.Pulse }, func(s *state) *string { return &s.pulse }},
		{func(c cue) *string { return c.Floods }, func(s *state) *string { return &s.floods }},
	}
)

// fold is the state once every cue at or before a moment has spoken.
func (of *Sheet) fold(seconds float64) state {
	now := resting()
	for _, each := range of.cues {
		if each.at > seconds {
			continue
		}
		now.apply(each)
	}
	return now
}

// apply is one cue speaking: each field it names changes, and only
// those.
func (now *state) apply(each placed) {
	line := each.line
	fade := 0.0
	if line.Fade != nil {
		fade = *line.Fade
	}
	for _, field := range fades {
		if said := field.said(line); said != nil {
			field.level(now).glide(*said, each.at, fade)
		}
	}
	for _, field := range numbers {
		if said := field.said(line); said != nil {
			*field.value(now) = *said
		}
	}
	for _, field := range counts {
		if said := field.said(line); said != nil {
			*field.value(now) = *said
		}
	}
	for _, field := range words {
		if said := field.said(line); said != nil {
			*field.value(now) = *said
		}
	}
	if line.Split != nil {
		now.split = *line.Split
	}
	now.lists(each)
}

// lists is a cue's lists, each taken in turn from where the cue stands
// on the grid. The fires burst from their cue's beat the same way: where
// the sheet puts it is the bursts' phase.
func (now *state) lists(each placed) {
	line := each.line
	if len(line.Palette) > 0 {
		now.palette = line.Palette
	}
	if line.Fire != nil {
		now.fireFrom = each.grid
	}
	if len(line.Figures) > 0 {
		now.figures, now.figuresFrom = line.Figures, each.grid
	}
	if len(line.Banks) > 0 {
		now.banks, now.banksFrom = line.Banks, each.grid
	}
	if len(line.SearchMoves) > 0 {
		now.searchMoves, now.searchMovesFrom = line.SearchMoves, each.grid
	}
}
