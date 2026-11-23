package session

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/set"
)

// Where scrooge is asked for, the things the set names are made ahead of
// the loop that stands them, on as many goroutines as the program is
// allowed processors, and the loop takes each from a table where it would
// have made it.
//
// The loop stays the only writer of the world, the places, the radii, the
// stock and the counters, and stands things in set order.
//
// What is made ahead is only what the loop is about to ask the makers
// for, found by the loop's own questions. A table is keyed by everything
// the making depends on, so a thing the table has for other arguments is
// not taken, and a guess that is wrong costs time and changes nothing.
//
// Everything of the stage a making reads is worked out before the workers
// start, on the loop's goroutine: the camera, the facts of each design,
// each row constructed and each ladder's leveller. A worker holds a job
// that carries its own inputs and writes its own result, and nothing else.

// fireKind is the kind of a thing that is a fire, which is not made from
// the stock.
const fireKind = "fire"

// askProcessors asks GOMAXPROCS how many processors the program is
// allowed, without changing it.
const askProcessors = 0

// turnedAt is a place turned about the vertical by a turn in radians.
func turnedAt(place maths.Vec, turn float64) maths.Subspace {
	return maths.Subspace{Origin: place,
		Orientation: maths.About(maths.YAxis, turn)}
}

// warmKey is everything the making of a thing part by part depends on:
// the arguments levelledStock is given, the complexity being the one
// memberDetail gave and the place carrying its turn.
type warmKey struct {
	kind, recipe, detail string
	seed                 uint64
	name                 string
	at                   maths.Subspace
}

// wholeKey is everything the making of a whole thing depends on: what the
// stock is asked for, and how much taller and whether branched the thing's
// name has it grown, which is all the name has to do with it.
type wholeKey struct {
	kind, recipe, detail string
	seed                 uint64
	taller               float64
	branched             bool
}

// warmed is the things made ahead of the loop, by what they depend on,
// and how many of them the loop has taken. It is written once, before the
// loop, and then read and counted by the loop alone. A nil table has
// nothing.
type warmed struct {
	composites map[warmKey]kept
	wholes     map[wholeKey]kept
	taken      int
}

// composite is the thing part by part made ahead for a key, if there is
// one, and counts it taken.
func (table *warmed) composite(key warmKey) (kept, bool) {
	if table == nil {
		return kept{}, false
	}
	one, made := table.composites[key]
	if made {
		table.taken++
	}
	return one, made
}

// whole is the whole thing made ahead for a key, if there is one, and
// counts it taken.
func (table *warmed) whole(key wholeKey) (kept, bool) {
	if table == nil {
		return kept{}, false
	}
	one, made := table.wholes[key]
	if made {
		table.taken++
	}
	return one, made
}

// wholeMade is a whole thing of a key: the table's, or made now.
func (stage *staging) wholeMade(key wholeKey) (kept, error) {
	if one, taken := stage.warm.whole(key); taken {
		return one, nil
	}
	fresh, err := stage.constructedTaller(key.kind, key.recipe, key.detail,
		key.seed, key.taller, key.branched)
	if err != nil {
		return kept{}, err
	}
	return *fresh, nil
}

// warmAsk is a thing of the set as the loop will ask the makers for it,
// before the complexity scrooge gives it: where it stands, and which of
// the set's things it is, which its turn may depend on.
type warmAsk struct {
	kind, recipe, detail, name string
	seed, index                uint64
	burning                    bool
	place                      maths.Vec
}

// warmAsks is the things of the set that can be made ahead, in the set's
// order. A fire is not made from the stock; a village or a house of one is
// made for the ground under it, which a worker cannot read.
//
// A thing the trial leaves out is not made, and a thing that has no place,
// which the loop will refuse, is left to the loop.
func (stage *staging) warmAsks(things []set.Thing) []warmAsk {
	var asks []warmAsk
	for index, each := range things {
		if each.Kind() == fireKind || stage.villagey(each) ||
			stage.trialUnseen(each.Name()) {
			continue
		}
		place, err := stage.thingPlace(each)
		if err != nil {
			continue
		}
		asks = append(asks, warmAsk{kind: each.Kind(), recipe: each.Recipe(),
			detail: each.Complexity(), name: each.Name(), seed: each.Seed(),
			index: uint64(index), burning: each.Alight(), place: place})
	}
	return asks
}

// warmThings is the table of what the set's things are made ahead as,
// where scrooge is asked for, and nil where it is not.
func (stage *staging) warmThings(things []set.Thing) *warmed {
	if !scrooge.Asked() {
		return nil
	}
	return stage.warmUp(stage.warmAsks(things))
}

// warmJob is one thing to make ahead, how to make it from what it holds,
// and where it goes in the table. A worker writes one and its own result
// fields, and the loop's goroutine reads them once the workers are done.
type warmJob struct {
	build func() (kept, error)
	store func(table *warmed, one kept)
	one   kept
	err   error
}

// warmWorkers is how many goroutines make things ahead: as many as the
// processors the program is allowed.
func warmWorkers() int { return runtime.GOMAXPROCS(askProcessors) }

// warmUp makes what the asks call for on the workers, and keeps what was
// made in a table. A thing whose making is refused is not kept, so the
// loop makes it and is refused as it would have been, in its order.
func (stage *staging) warmUp(asks []warmAsk) *warmed {
	composites, wholes := stage.warmPlan(asks)
	jobs := append(composites, wholes...)
	warmRun(jobs, warmWorkers())
	table := &warmed{composites: map[warmKey]kept{},
		wholes: map[wholeKey]kept{}}
	for _, job := range jobs {
		if job.err == nil {
			job.store(table, job.one)
		}
	}
	return table
}

// warmRun runs the jobs on at most workers goroutines, each taking the
// next that is not taken, and waits for them all.
func warmRun(jobs []*warmJob, workers int) {
	var next atomic.Int64
	var done sync.WaitGroup
	for range min(workers, len(jobs)) {
		done.Add(1)
		go func() {
			defer done.Done()
			for {
				at := int(next.Add(1)) - 1
				if at >= len(jobs) {
					return
				}
				jobs[at].one, jobs[at].err = jobs[at].build()
			}
		}()
	}
	done.Wait()
}

// warmPlan is the jobs for the asks, on the loop's goroutine: those made
// part by part, which cost most and so start first, and the whole ones.
// Where the camera is refused there are none, and the loop is refused.
func (stage *staging) warmPlan(asks []warmAsk) (composites,
	wholes []*warmJob) {
	eye, err := stage.seen()
	if err != nil {
		return nil, nil
	}
	planned := map[any]bool{}
	for _, each := range asks {
		job, composite := stage.warmJobFor(each, eye, planned)
		if job == nil {
			continue
		}
		if composite {
			composites = append(composites, job)
		} else {
			wholes = append(wholes, job)
		}
	}
	return composites, wholes
}

// warmJobFor is the job for an ask, and whether it is made part by part,
// as putRecipeFacing and levelledStock go about it: the turn, the
// complexity memberDetail gives, then a thing part by part where
// sparingly would make one, and a whole one from the stock where it
// would not.
//
// A thing the set levels its own way is not made ahead, since its level
// is worked out as it is made and kept in the stage. The job is nil where
// the ask cannot be made ahead or is already planned.
func (stage *staging) warmJobFor(each warmAsk, eye strategies.Eye,
	planned map[any]bool) (*warmJob, bool) {
	turn := stage.placementTurn(each.kind, each.name, each.seed, each.index)
	at := turnedAt(each.place, turn)
	chosen, err := stage.rungChosen(each.kind, each.recipe, each.detail,
		each.seed, each.name, at, each.burning)
	if err != nil || chosen.rung == manufacture.Levelled {
		return nil, false
	}
	row, rung, sparing := stage.sparingRung(each.kind, each.recipe,
		chosen.rung)
	if row == nil {
		return nil, false
	}
	if sparing && each.name != "" {
		key := warmKey{kind: each.kind, recipe: each.recipe,
			detail: chosen.rung, seed: each.seed, name: each.name, at: at}
		return stage.compositeJob(key, *row, rung, eye, planned), true
	}
	taller, branched := stage.growth(each.name)
	key := wholeKey{kind: each.kind, recipe: each.recipe,
		detail: chosen.rung, seed: each.seed, taller: taller,
		branched: branched}
	return stage.wholeJob(key, *row, planned), false
}

// compositeJob is the job that makes a thing part by part, for a key, from
// a row constructed and the ladder's leveller, both of which the stage
// keeps and a worker must not be the first to ask for.
func (stage *staging) compositeJob(key warmKey, row manufacture.Kind,
	rung int, eye strategies.Eye, planned map[any]bool) *warmJob {
	leveller, ladder, err := stage.levellerOf(row)
	if err != nil || planned[key] {
		return nil
	}
	planned[key] = true
	with := composing{row: row, leveller: leveller, ladder: ladder, eye: eye}
	given := stage.given
	return &warmJob{
		build: func() (kept, error) {
			return with.made(given, key.seed, key.at, rung)
		},
		store: func(table *warmed, one kept) { table.composites[key] = one },
	}
}

// wholeJob is the job that makes a whole thing, for a key, from a row
// constructed.
func (stage *staging) wholeJob(key wholeKey, row manufacture.Kind,
	planned map[any]bool) *warmJob {
	if planned[key] {
		return nil
	}
	planned[key] = true
	given := stage.given
	return &warmJob{
		build: func() (kept, error) {
			one, err := madeTaller(given, row, key.detail, key.seed,
				key.taller, key.branched)
			if err != nil {
				return kept{}, err
			}
			return *one, nil
		},
		store: func(table *warmed, one kept) { table.wholes[key] = one },
	}
}

// warmNote says, among the session's notes, how many things were made
// ahead and how many of them the loop took. A thing made and not taken
// was a wrong guess, which cost time and nothing else.
func (stage *staging) warmNote() {
	if stage.warm == nil {
		return
	}
	made := len(stage.warm.composites) + len(stage.warm.wholes)
	stage.given.cached = append(stage.given.cached, cacheNote{told: true,
		what: fmt.Sprintf("scrooge: %d things made ahead on %d workers, "+
			"%d taken by the loop", made, warmWorkers(),
			stage.warm.taken)})
}
