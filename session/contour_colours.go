package session

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// A row of a contour stand is several terraces end to end. Each terrace
// keeps its culture's rule that no two neighbours share a colour, and
// cannot see the terrace beside it, so where two meet the stand draws the
// next from a number whose first bay differs in colour from the last bay
// of the one before. A culture that does not say its neighbours differ is
// left as it was, and so is the first terrace of every row.

// joinShift lifts a retry above the bits of a run's own number. A pick is
// the stand's seed shifted up by gridPickShift bits, added to the run's
// number, so a retry's count shifted by this many bits cannot reach either.
const joinShift = 40

// mostJoins is the most numbers tried for one run. A try fails when the
// first bay's colour is the one before it, which one palette colour in as
// many as the palette has does, so five colours fail sixty-four tries in a
// row once in five to the power of sixty-four. The bound is there so that
// a palette of one colour ends.
const mostJoins = 64

// contourRuns is the room each run of a contour stand takes, by its number
// among the sites the habit lays.
type contourRuns struct {
	stage *staging
	each  set.Stand
}

// room is the room of the terrace that will stand at a run's number, from
// the number the stage will make it from.
func (runs contourRuns) room(number int) (cities.Run, error) {
	pick, err := runs.stage.runPick(runs.each, number)
	if err != nil {
		return cities.Run{}, err
	}
	return runs.stage.runFrom(runs.each, pick)
}

// runFrom is the room and the end colours of the terrace of a stand's kind
// that the stage makes from a pick.
func (stage *staging) runFrom(each set.Stand, pick uint64) (cities.Run,
	error) {
	return manufacture.RunOf(each.Kind(),
		placement.Seed(stage.given.seed, int(pick)))
}

// pickOf is the number a site's thing is drawn from: the contour's run
// number for a run, the frontage's own number for each of its sites, and
// the stand's own pick for any other habit.
func (stage *staging) pickOf(each set.Stand, seed uint64, index,
	number int) (uint64, error) {
	switch each.Habit() {
	case cities.HabitContour:
		return stage.runPick(each, number)
	case cities.HabitFrontage:
		return standPick(each, seed, number), nil
	}
	return standPick(each, seed, index), nil
}

// runPick is the number the run of a contour stand with a number among its
// sites is drawn from. Runs are numbered row by row, so the one before a
// run in its row is the run numbered one less, unless it is the first of
// its row. The pick of every run up to the one asked for is kept, so a
// run asked for twice has the same pick.
func (stage *staging) runPick(each set.Stand, number int) (uint64, error) {
	if stage.joined == nil {
		stage.joined = map[string][]uint64{}
	}
	picks := stage.joined[each.Name()]
	_, seed := each.Count()
	for len(picks) <= number {
		next, err := stage.joinedPick(each, seed, picks)
		if err != nil {
			return 0, err
		}
		picks = append(picks, next)
	}
	stage.joined[each.Name()] = picks
	return picks[number], nil
}

// joinedPick is the pick of the run that follows the runs already picked:
// its own, if it starts a row or its culture lets neighbours match, and
// otherwise the first retry of it whose first bay is not the colour of the
// last bay of the run before.
func (stage *staging) joinedPick(each set.Stand, seed uint64,
	before []uint64) (uint64, error) {
	number := len(before)
	pick := standPick(each, seed, number)
	climb, _ := each.Climb()
	if number%climb.Runs == 0 {
		return pick, nil
	}
	last, err := stage.runFrom(each, before[number-1])
	if err != nil || !last.Differ {
		return pick, err
	}
	for try := range uint64(mostJoins) {
		candidate := pick + try<<joinShift
		run, err := stage.runFrom(each, candidate)
		if err != nil || run.First != last.Last {
			return candidate, err
		}
	}
	return pick, nil
}
