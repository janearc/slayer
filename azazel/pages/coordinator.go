package pages

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// TickEvery is how often the coordinator publishes while any job is live.
const TickEvery = 10 * time.Second // defined: agreed with an agent, 2026-11-15

// galleryEvery is the least time between two gallery publications while
// jobs run.
const galleryEvery = time.Minute // defined: latest-while.sh's sixty seconds

// encodeLimit is the longest one partial film may take.
// picked: a whole two-thousand-frame film at 480 lines encodes in under a
// minute on VideoToolbox; four thousand frames at 4K in a few.
const encodeLimit = 30 * time.Minute

// The notices the log is given once for a job.
const (
	noTotal = "pictures was not given, so barkeep shows no line for " +
		"this render"
	heldLine = "another render holds barkeep's line; waiting"
)

// live is one running job the coordinator publishes for, and what the
// log has been told of it.
type live struct {
	sequence uint64
	folder   Folder
	pictures int64
	inputs   []string
	started  time.Time
	said     map[string]bool
}

// say writes a message about a job into the coordinator's log, once.
func (c *Coordinator) say(job *live, message string) {
	if job.said[message] {
		return
	}
	job.said[message] = true
	fmt.Fprintf(c.log, "%s job %d (%s/%s): %s\n",
		c.publisher.now().Format(time.RFC3339), job.sequence, job.folder.Shelf,
		job.folder.Name, message)
}

// passOver writes, once for each job, why nothing is published for it.
func (c *Coordinator) passOver(sequence uint64, reason string) {
	c.state.Lock()
	defer c.state.Unlock()
	if c.passed[sequence] {
		return
	}
	c.passed[sequence] = true
	fmt.Fprintf(c.log, "%s job %d: nothing is published for it: %s\n",
		c.publisher.now().Format(time.RFC3339), sequence, reason)
}

// note writes a message about no one job into the log, when it is not
// the last one written.
func (c *Coordinator) note(message string) {
	if message == c.noted {
		return
	}
	c.noted = message
	fmt.Fprintf(c.log, "%s: %s\n", c.publisher.now().Format(time.RFC3339),
		message)
}

// tell gives the log a notice and a failure for every job of a group,
// where there is one.
func (c *Coordinator) tell(jobs []*live, notice string, err error) {
	for _, job := range jobs {
		if notice != "" {
			c.say(job, notice)
		}
		if err != nil {
			c.say(job, err.Error())
		}
	}
}

// reading is one render's progress at a tick, as barkeep's line says it.
type reading struct {
	done, total, perMinute, left int64
}

// Coordinator is the one publisher for every job of a queue. It gathers
// the jobs of a folder, so that no two write the same page, and it owns
// barkeep's line for one render at a time.
type Coordinator struct {
	publisher publisher
	every     time.Duration
	keeper    barkeep
	log       io.Writer
	noted     string
	looked    bool

	// state guards the live jobs, those passed over, the film locks and
	// the films' failures.
	state     sync.Mutex
	jobs      map[uint64]*live
	passed    map[uint64]bool
	ended     map[uint64]time.Time
	retryEnds map[uint64]bool
	films     map[Folder]*sync.Mutex
	failures  map[Folder]error
	encoding  sync.WaitGroup

	// publishing is held by one publication at a time, and guards the
	// rest.
	publishing sync.Mutex
	gallery    time.Time
	paces      map[Folder]*pace
	owner      Folder
	ownerTotal int64
}

// New is a coordinator for the settings, with no live jobs, writing what
// it has to say to log.
func New(settings Settings, log io.Writer) *Coordinator {
	made := publisherOf(settings)
	return &Coordinator{publisher: made, every: settings.Every, log: log,
		keeper: barkeep{path: filepath.Join(made.root, progressName)},
		jobs:   map[uint64]*live{}, passed: map[uint64]bool{}, ended: map[uint64]time.Time{}, retryEnds: map[uint64]bool{},
		films:    map[Folder]*sync.Mutex{},
		failures: map[Folder]error{}, paces: map[Folder]*pace{}}
}

// Start records a job as live, when it writes into a render folder, as
// old as the ledger says it is. Of a job with no pictures the log is told
// once that barkeep shows nothing for it.
func (c *Coordinator) Start(job queue.Job) {
	folder, reason := folderOf(c.publisher.root, job.Request)
	if reason != "" {
		c.passOver(job.Sequence, reason)
		return
	}
	record := &live{sequence: job.Sequence, folder: folder,
		inputs: job.Request.Inputs, started: job.Started,
		said: map[string]bool{}}
	if job.Request.Pictures == nil {
		c.say(record, noTotal)
	} else {
		record.pictures = *job.Request.Pictures
	}
	c.state.Lock()
	defer c.state.Unlock()
	c.jobs[job.Sequence] = record
}

// liveJobs is the live jobs, oldest first.
func (c *Coordinator) liveJobs() []*live {
	c.state.Lock()
	defer c.state.Unlock()
	jobs := make([]*live, 0, len(c.jobs))
	for _, job := range c.jobs {
		jobs = append(jobs, job)
	}
	slices.SortFunc(jobs, func(a, b *live) int {
		return cmp.Or(a.started.Compare(b.started),
			cmp.Compare(a.sequence, b.sequence))
	})
	return jobs
}

// tick is one publication: every live folder, the gallery when a minute
// has passed, and barkeep's line.
func (c *Coordinator) tick(ctx context.Context) {
	c.publishing.Lock()
	defer c.publishing.Unlock()
	jobs := c.liveJobs()
	if len(jobs) == 0 {
		return
	}
	c.sayFailures(jobs)
	counts, _ := c.publishFolders(ctx, jobs, true)
	now := c.publisher.now()
	if now.Sub(c.gallery) >= galleryEvery {
		c.publishGallery(jobs, now)
	}
	c.keepBarkeep(jobs, counts, now)
}

// byFolder groups jobs by their render folder, oldest first in each.
func byFolder(jobs []*live) map[Folder][]*live {
	groups := map[Folder][]*live{}
	for _, job := range jobs {
		groups[job.folder] = append(groups[job.folder], job)
	}
	return groups
}

// inputsOf is every declared input of a folder's jobs.
func inputsOf(jobs []*live) []string {
	var inputs []string
	for _, job := range jobs {
		inputs = append(inputs, job.inputs...)
	}
	return inputs
}

// publishFolders writes each live folder's page once, starts a partial
// film where one is due and films are still made, and returns how many
// pictures each folder holds.
func (c *Coordinator) publishFolders(ctx context.Context, jobs []*live,
	films bool) (map[Folder]int64, error) {
	counts := map[Folder]int64{}
	var failures []error
	for folder, group := range byFolder(jobs) {
		found, err := scanFolder(folder.path(c.publisher.root))
		if err != nil {
			c.tell(group, "", err)
			failures = append(failures, err)
			continue
		}
		counts[folder] = int64(len(found.pictures))
		look, err := c.publisher.view(folder, inputsOf(group))
		if films && look.due(false) {
			c.encodeLater(ctx, folder, inputsOf(group))
		}
		notice, wrong := c.publisher.publishFolder(folder, found, look)
		c.tell(group, notice, errors.Join(err, wrong))
		failures = append(failures, err, wrong)
	}
	return counts, errors.Join(failures...)
}

// filmLock is the lock one folder's film is made under.
func (c *Coordinator) filmLock(folder Folder) *sync.Mutex {
	c.state.Lock()
	defer c.state.Unlock()
	if c.films[folder] == nil {
		c.films[folder] = &sync.Mutex{}
	}
	return c.films[folder]
}

// encodeLater makes a folder's partial film away from the tick, so that
// barkeep's line keeps its time; a film already being made is left to
// finish. A failure is told at the next tick.
func (c *Coordinator) encodeLater(ctx context.Context, folder Folder,
	inputs []string) {
	lock := c.filmLock(folder)
	if !lock.TryLock() {
		return
	}
	c.encoding.Go(func() {
		defer lock.Unlock()
		look, err := c.publisher.view(folder, inputs)
		if err == nil && look.due(false) {
			limited, cancel := context.WithTimeout(ctx, encodeLimit)
			defer cancel()
			err = c.publisher.encode(limited,
				folder.path(c.publisher.root), look)
		}
		c.remember(folder, err)
	})
}

// remember keeps a film's failure for the next tick to tell.
func (c *Coordinator) remember(folder Folder, err error) {
	if err == nil {
		return
	}
	c.state.Lock()
	defer c.state.Unlock()
	c.failures[folder] = err
}

// sayFailures tells each folder's jobs the failures of its films since
// the last tick.
func (c *Coordinator) sayFailures(jobs []*live) {
	c.state.Lock()
	failures := c.failures
	c.failures = map[Folder]error{}
	c.state.Unlock()
	for folder, group := range byFolder(jobs) {
		c.tell(group, "", failures[folder])
	}
}

// publishGallery writes latest.html and the indexes, telling every job a
// failure.
func (c *Coordinator) publishGallery(jobs []*live, now time.Time) error {
	err := c.publisher.publishGallery()
	if err == nil {
		c.gallery = now
	}
	c.tell(jobs, "", err)
	return err
}

// readings takes every known render's count into its pace, and returns
// the owner's reading: the oldest live render with a known total.
func (c *Coordinator) readings(jobs []*live, counts map[Folder]int64,
	now time.Time) (Folder, reading, bool) {
	totals, owner := map[Folder]int64{}, Folder{}
	for _, job := range jobs {
		if job.pictures > 0 && owner == (Folder{}) {
			owner = job.folder
		}
		totals[job.folder] = max(totals[job.folder], job.pictures)
	}
	var mine reading
	for folder, total := range totals {
		done, counted := counts[folder]
		if total == 0 || !counted {
			continue
		}
		if c.paces[folder] == nil {
			c.paces[folder] = &pace{}
		}
		perMinute, left := c.paces[folder].add(now.Unix(), done, total)
		if folder == owner {
			mine = reading{done, total, perMinute, left}
		}
	}
	return owner, mine, mine.total > 0
}

// keepBarkeep writes the owner's line, unless someone else's live render
// holds it.
func (c *Coordinator) keepBarkeep(jobs []*live, counts map[Folder]int64,
	now time.Time) {
	owner, mine, found := c.readings(jobs, counts, now)
	if !found {
		return
	}
	group := byFolder(jobs)[owner]
	if !c.keeper.free(now.Unix()) {
		c.tell(group, heldLine, nil)
		return
	}
	err := c.keeper.write(mine.done, mine.total, mine.perMinute, mine.left,
		now.Unix())
	c.tell(group, "", err)
	c.owner, c.ownerTotal = owner, mine.total
}

// End publishes once more for a job that has ended: the last partial film
// when its folder has no other live job, the folder's page, the gallery,
// and barkeep's last line for the render it owned. The ledger is only
// read, so nothing here changes the job or its result.
func (c *Coordinator) End(job queue.Job, result queue.Result) error {
	return c.end(context.Background(), job, result)
}

// end handles a job that has ended: it writes the job's last pages and line.
func (c *Coordinator) end(ctx context.Context, job queue.Job, result queue.Result) error {
	record, last := c.remove(job.Sequence)
	if record == nil {
		return nil
	}
	var err error
	if last {
		err = c.finalFilm(ctx, record)
	}
	c.publishing.Lock()
	defer c.publishing.Unlock()
	jobs := []*live{record}
	c.tell(jobs, "", err)
	// Retry failures on the next ledger poll, without a second encode here.
	counts, folderErr := c.publishFolders(ctx, jobs, !last)
	now := c.publisher.now()
	galleryErr := c.publishGallery(jobs, now)
	var lineErr error
	if last {
		lineErr = c.endLine(record, counts, result, now)
	}
	return errors.Join(err, folderErr, galleryErr, lineErr)
}

// remove takes a job from the live jobs, and says whether it was its
// folder's last.
func (c *Coordinator) remove(sequence uint64) (*live, bool) {
	c.state.Lock()
	defer c.state.Unlock()
	record, found := c.jobs[sequence]
	if !found {
		return nil, false
	}
	delete(c.jobs, sequence)
	for _, other := range c.jobs {
		if other.folder == record.folder {
			return record, false
		}
	}
	return record, true
}

// finalFilm makes the last partial film of a folder whose jobs have all
// ended, when frames have come since the one before.
func (c *Coordinator) finalFilm(ctx context.Context, record *live) error {
	lock := c.filmLock(record.folder)
	lock.Lock()
	defer lock.Unlock()
	look, err := c.publisher.view(record.folder, record.inputs)
	if err != nil || !look.due(true) {
		return err
	}
	limited, cancel := context.WithTimeout(ctx, encodeLimit)
	defer cancel()
	return c.publisher.encode(limited, record.folder.path(c.publisher.root),
		look)
}

// endLine writes barkeep's last line for a render that owned it: done
// equal to the total and nought left when it succeeded, else the count it
// reached with the time left unknown.
func (c *Coordinator) endLine(record *live, counts map[Folder]int64,
	result queue.Result, now time.Time) error {
	progress := c.paces[record.folder]
	done, counted := counts[record.folder]
	if c.owner != record.folder || !counted || progress == nil {
		return nil
	}
	total := c.ownerTotal
	perMinute, _ := progress.add(now.Unix(), done, total)
	left := int64(unknownLeft)
	if result.State == queue.Succeeded {
		done, left = total, 0
	}
	if c.keeper.free(now.Unix()) {
		err := c.keeper.write(done, total, perMinute, left, now.Unix())
		c.tell([]*live{record}, "", err)
		if err != nil {
			return err
		}
	}
	delete(c.paces, record.folder)
	c.owner, c.ownerTotal = Folder{}, 0
	return nil
}
