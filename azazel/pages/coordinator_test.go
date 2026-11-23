package pages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// picturesEach is how many pictures each job of a concurrent test makes.
const picturesEach = 8 // picked: enough to cross several looks

// wholeLine is a whole barkeep line, as its reader takes one.
var wholeLine = regexp.MustCompile(`^\d+ [1-9]\d* \d+ (-1|\d+) \d+\n$`)

// renders is an empty renders root with its two shelves.
func renders(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "renders")
	for _, shelf := range []string{framesShelf, animationsShelf} {
		if err := os.MkdirAll(filepath.Join(root, shelf), folders); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// stills lays count pictures, from the number first, in a folder of
// stills.
func stills(t *testing.T, root string, folder Folder, first, count int) {
	t.Helper()
	for index := first; index < first+count; index++ {
		lay(t, root, fixtureNoon.Unix(), fmt.Sprintf("%s/%s/still-%03d.png",
			folder.Shelf, folder.Name, index))
	}
}

// jobFor is a running job that writes into a render folder, started at
// the clock's start plus its sequence in seconds, with its pictures when
// there are any.
func jobFor(sequence uint64, root string, folder Folder,
	pictures int64) queue.Job {
	request := queue.Request{Agent: "studio", Key: fmt.Sprint(sequence),
		Command: "/usr/bin/true", Directory: root, Lane: "gpu",
		Args: []string{"draw", "set.yaml", "--out",
			filepath.Join(folder.path(root), "frame.png")}}
	if pictures > 0 {
		request.Pictures = &pictures
	}
	started := fixtureNoon.Add(time.Duration(sequence) * time.Second)
	return queue.Job{Sequence: sequence, Request: request,
		State: queue.Running, Started: started}
}

// finished is a job as the ledger shows it once it has ended.
func finished(job queue.Job, result queue.Result, at time.Time) queue.Job {
	job.State, job.Result, job.Finished = result.State, result, at
	return job
}

// lineFieldsOf is barkeep's line as its five words.
func lineFieldsOf(t *testing.T, root string) []string {
	t.Helper()
	progressPath := filepath.Join(root, progressName)
	return strings.Fields(read(t, progressPath))
}

// pair is two render folders of stills.
var pair = []Folder{
	{Shelf: framesShelf, Name: "2026-11-15-studio-first"},
	{Shelf: framesShelf, Name: "2026-11-15-studio-second"},
}

// succeeded is a job's success.
var succeeded = queue.Result{State: queue.Succeeded}

// TestTwoJobsInOneFolderShareItsPageAndLine: the folder's total is the
// larger stated, and its last job's end says done equal to the total.
func TestTwoJobsInOneFolderShareItsPageAndLine(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 3)
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	coordinator.Start(jobFor(1, root, pair[0], 10))
	coordinator.Start(jobFor(2, root, pair[0], 12))
	coordinator.tick(t.Context())
	line := lineFieldsOf(t, root)
	pagePath := filepath.Join(pair[0].path(root), pageName)
	page := read(t, pagePath)
	if line[0] != "3" || line[1] != "12" || !strings.Contains(page,
		"3 pictures.") {
		t.Fatal(line, page)
	}
	coordinator.End(jobFor(1, root, pair[0], 10), succeeded)
	hands.advance(TickEvery)
	stills(t, root, pair[0], 3, 9)
	coordinator.End(jobFor(2, root, pair[0], 12), succeeded)
	line = lineFieldsOf(t, root)
	if line[0] != "12" || line[1] != "12" || line[3] != "0" {
		t.Fatal("the last end did not finish the line", line)
	}
	if log.Len() != 0 {
		t.Fatal("the log was told of failures:", log.String())
	}
}

// TestUnknownTotalWritesNoLine is the anti-test: a job with no pictures
// gets no line, and the log says so once.
func TestUnknownTotalWritesNoLine(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 3)
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	coordinator.Start(jobFor(1, root, pair[0], 0))
	coordinator.tick(t.Context())
	hands.advance(TickEvery)
	coordinator.tick(t.Context())
	if _, err := os.Lstat(filepath.Join(root, progressName)); err == nil {
		t.Fatal("a line was written without a total")
	}
	if strings.Count(log.String(), noTotal) != 1 {
		t.Fatal("the log did not say it once:", log.String())
	}
}

// startPair starts a job in each folder of the pair, the first older.
func startPair(root string, coordinator *Coordinator, pictures []int64) {
	for index, folder := range pair {
		number := uint64(index + 1)
		coordinator.Start(jobFor(number, root, folder,
			pictures[index]))
	}
}

// TestOldestKnownRenderOwnsTheLine: of two folders, the one started
// first holds barkeep's line.
func TestOldestKnownRenderOwnsTheLine(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	stills(t, root, pair[1], 0, 2)
	coordinator := New(settingsFor(root, nil, hands), io.Discard)
	startPair(root, coordinator, []int64{5, 7})
	coordinator.tick(t.Context())
	if line := lineFieldsOf(t, root); line[0] != "1" || line[1] != "5" {
		t.Fatal("the younger render took the line", line)
	}
}

// TestOldestWithoutATotalLeavesTheLineToTheNext is the anti-test: the
// older render without a total gives the line to the younger.
func TestOldestWithoutATotalLeavesTheLineToTheNext(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	stills(t, root, pair[1], 0, 2)
	coordinator := New(settingsFor(root, nil, hands), io.Discard)
	startPair(root, coordinator, []int64{0, 7})
	coordinator.tick(t.Context())
	if line := lineFieldsOf(t, root); line[0] != "2" || line[1] != "7" {
		t.Fatal("the render with a total did not take the line", line)
	}
}

// TestLiveForeignLineIsNeverOverwritten: a line someone else wrote five
// seconds ago is left, and the log says so once.
func TestLiveForeignLineIsNeverOverwritten(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	path := filepath.Join(root, progressName)
	foreign := lineAt(t, path, fixtureNoon.Unix()-5)
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	coordinator.Start(jobFor(1, root, pair[0], 10))
	coordinator.tick(t.Context())
	hands.advance(TickEvery)
	coordinator.tick(t.Context())
	if read(t, path) != foreign || strings.Count(log.String(), heldLine) != 1 {
		t.Fatal("a live line was written over", read(t, path), log.String())
	}
}

// TestStaleForeignLineIsTaken is the anti-test: a line two minutes old is
// no one's live render, and is written over.
func TestStaleForeignLineIsTaken(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	path := filepath.Join(root, progressName)
	lineAt(t, path, fixtureNoon.Unix()-liveLine)
	coordinator := New(settingsFor(root, nil, hands), io.Discard)
	coordinator.Start(jobFor(1, root, pair[0], 10))
	coordinator.tick(t.Context())
	if line := lineFieldsOf(t, root); line[1] != "10" {
		t.Fatal("a stale line was kept", line)
	}
}

// ledgerOf is a mock ledger that shows each snapshot in turn, then the
// last one for good.
func ledgerOf(t *testing.T, snapshots ...[]queue.Job) *MockLedger {
	t.Helper()
	ledger := NewMockLedger(gomock.NewController(t))
	var calls []any
	for _, jobs := range snapshots {
		calls = append(calls, ledger.EXPECT().Status(gomock.Any()).
			Return(queue.Snapshot{Jobs: jobs}, nil))
	}
	gomock.InOrder(calls...)
	return ledger
}

// filmJob is the one film job of the publication tests.
func filmJob(root string) queue.Job {
	return jobFor(1, root, Folder{Shelf: animationsShelf, Name: filmName}, 12)
}

// TestPublicationFailuresAreLoggedAndServingGoesOn: an encoder that fails
// and a renders root that cannot be written are told in the log, and the
// next look still publishes the job's end.
func TestPublicationFailuresAreLoggedAndServingGoesOn(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "6")
	encoder := NewMockEncoder(gomock.NewController(t))
	// Once at the tick, and once more for the last film at the end.
	encoder.EXPECT().Encode(gomock.Any(), gomock.Any()).
		Return(errors.New("the encoder failed")).Times(2)
	if err := os.Chmod(root, 0o555); err != nil { // defined: read-only
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, folders) })
	job := filmJob(root)
	ledger := ledgerOf(t, nil, []queue.Job{job},
		[]queue.Job{finished(job, succeeded, fixtureNoon)})
	var log bytes.Buffer
	coordinator := New(settingsFor(root, encoder, &clock{at: fixtureNoon}),
		&log)
	for range 3 { // defined: one look for each snapshot
		coordinator.look(t.Context(), ledger)
	}
	coordinator.encoding.Wait()
	page := filepath.Join(folder.path(root), pageName)
	if !strings.Contains(log.String(), "the encoder failed") ||
		!strings.Contains(log.String(), latestName) || !regularFile(page) {
		t.Fatal("a failure went untold or stopped publication", log.String())
	}
}

// TestPublicationThatSucceedsTellsTheLogNothing is the anti-test.
func TestPublicationThatSucceedsTellsTheLogNothing(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "6")
	err := os.MkdirAll(filepath.Join(root, framesShelf), folders)
	if err != nil {
		t.Fatal(err)
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 12)
	job := filmJob(root)
	ledger := ledgerOf(t, []queue.Job{job},
		[]queue.Job{finished(job, succeeded, fixtureNoon)})
	var log bytes.Buffer
	coordinator := New(settingsFor(root, encoder, &clock{at: fixtureNoon}),
		&log)
	coordinator.look(t.Context(), ledger)
	coordinator.encoding.Wait()
	coordinator.look(t.Context(), ledger)
	if log.Len() != 0 {
		t.Fatal(log.String())
	}
}

// publishFilmJob runs a film job's start, a tick and its end on a new
// coordinator, and returns the folder's page.
func publishFilmJob(t *testing.T, root string, sequence uint64,
	encoder Encoder) string {
	t.Helper()
	hands := &clock{at: fixtureNoon}
	coordinator := New(settingsFor(root, encoder, hands), io.Discard)
	folder := Folder{Shelf: animationsShelf, Name: filmName}
	job := jobFor(sequence, root, folder, 12)
	coordinator.Start(job)
	coordinator.tick(t.Context())
	coordinator.encoding.Wait()
	coordinator.End(job, succeeded)
	pagePath := filepath.Join(folder.path(root), pageName)
	return read(t, pagePath)
}

// restartFolder is a film folder with a hand page and a first
// coordinator's work done: one partial film of twelve frames.
func restartFolder(t *testing.T) (string, Folder, string) {
	t.Helper()
	root, folder := filmFolder(t, upTo(12), "6")
	err := os.MkdirAll(filepath.Join(root, framesShelf), folders)
	if err != nil {
		t.Fatal(err)
	}
	hand := filepath.Join(folder.path(root), pageName)
	if err := os.WriteFile(hand, []byte("<p>hers</p>\n"), listing); err != nil {
		t.Fatal(err)
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 12).Times(1)
	return root, folder, publishFilmJob(t, root, 1, encoder)
}

// TestRestartPublishesFromDiskAlone: a new coordinator over the folder
// writes the same page, keeps the hand page aside and makes no film.
func TestRestartPublishesFromDiskAlone(t *testing.T) {
	root, folder, before := restartFolder(t)
	if !strings.Contains(before, "the first 12 frames") ||
		!strings.Contains(before, handPageName) {
		t.Fatal("the first coordinator's page:", before)
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	if after := publishFilmJob(t, root, 2, encoder); after != before {
		t.Fatal("the page changed on restart:", after)
	}
	handPath := filepath.Join(folder.path(root), handPageName)
	aside := read(t, handPath)
	if aside != "<p>hers</p>\n" || !regularFile(filepath.Join(root,
		latestName)) {
		t.Fatal("the restart lost the hand page or the gallery")
	}
}

// TestRestartWithoutItsRecordMakesTheFilmAgain is the anti-test: with
// partial.json gone, the new coordinator cannot know the film, and
// makes it again.
func TestRestartWithoutItsRecordMakesTheFilmAgain(t *testing.T) {
	root, folder, _ := restartFolder(t)
	record := filepath.Join(folder.path(root), partialRecord)
	if err := os.Rename(record, record+".moved"); err != nil {
		t.Fatal(err)
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	expectFilm(encoder, root, folder, 12).Times(1)
	publishFilmJob(t, root, 2, encoder)
}

// firstLookAt has a new coordinator look once at a ledger holding one
func TestFinalFilmRetriesOnTheNextPoll(t *testing.T) {
	root, folder := filmFolder(t, upTo(12), "6")
	encoder := NewMockEncoder(gomock.NewController(t))
	encoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return(errors.New("temporary encoder failure")).Times(1)
	expectFilm(encoder, root, folder, 12).Times(1)
	job := finished(filmJob(root), succeeded, fixtureNoon)
	c := New(settingsFor(root, encoder, &clock{at: fixtureNoon}), io.Discard)
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	partialPath := filepath.Join(folder.path(root), partialName)
	if regularFile(partialPath) {
		t.Fatal("failed encode published a movie")
	}
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	if !regularFile(partialPath) {
		t.Fatal("completed job never retried publication")
	}
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
}

func TestFinalPageRetriesAfterWriteFailure(t *testing.T) {
	root := renders(t)
	stills(t, root, pair[0], 0, 1)
	dir := pair[0].path(root)
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })
	job := finished(jobFor(1, root, pair[0], 1), succeeded, fixtureNoon)
	c := New(settingsFor(root, nil, &clock{at: fixtureNoon}), io.Discard)
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	if regularFile(filepath.Join(dir, pageName)) {
		t.Fatal("write failure was not exercised")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	if !regularFile(filepath.Join(dir, pageName)) {
		t.Fatal("completed page was not retried")
	}
}

func TestPublisherFlushesCompletedJobOnShutdown(t *testing.T) {
	root := renders(t)
	stills(t, root, pair[0], 0, 1)
	job := finished(jobFor(1, root, pair[0], 1), succeeded, fixtureNoon)
	ledger := NewMockLedger(gomock.NewController(t))
	seen := make(chan struct{})
	first := ledger.EXPECT().Status(gomock.Any()).DoAndReturn(func(context.Context) (queue.Snapshot, error) { close(seen); return queue.Snapshot{}, nil })
	ledger.EXPECT().Status(gomock.Any()).DoAndReturn(func(ctx context.Context) (queue.Snapshot, error) {
		if ctx.Err() != nil {
			t.Error("shutdown flush used cancelled context")
		}
		return queue.Snapshot{Jobs: []queue.Job{job}}, nil
	}).After(first)
	settings := settingsFor(root, nil, &clock{at: fixtureNoon})
	settings.Every = time.Hour
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { New(settings, io.Discard).serve(ctx, ledger); close(done) }()
	<-seen
	cancel()
	<-done
	pagePath := filepath.Join(pair[0].path(root), pageName)
	if !regularFile(pagePath) {
		t.Fatal("shutdown missed the completed job")
	}
}

// firstLookAt has a new coordinator look once at a ledger holding one
// stills job that finished at the time given, and says whether its
// folder page was written.
func firstLookAt(t *testing.T, ended time.Time) bool {
	t.Helper()
	root := renders(t)
	stills(t, root, pair[0], 0, 2)
	job := finished(jobFor(1, root, pair[0], 2), succeeded, ended)
	coordinator := New(settingsFor(root, nil, &clock{at: fixtureNoon}),
		io.Discard)
	coordinator.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	pagePath := filepath.Join(pair[0].path(root), pageName)
	return regularFile(pagePath)
}

// TestFirstLookPublishesAnEndItMissed: a job that ended a minute before a
// restart has its end published from the ledger alone.
func TestFirstLookPublishesAnEndItMissed(t *testing.T) {
	if !firstLookAt(t, fixtureNoon.Add(-time.Minute)) {
		t.Fatal("a missed end was not published")
	}
}

// TestFirstLookLeavesOldEnds is the anti-test: an end eleven minutes old
// is left as it was.
func TestPublishesJobThatFinishesBetweenPollsOnce(t *testing.T) {
	root := renders(t)
	stills(t, root, pair[0], 0, 1)
	c := New(settingsFor(root, nil, &clock{at: fixtureNoon}), io.Discard)
	c.look(t.Context(), ledgerOf(t, nil))
	job := finished(jobFor(1, root, pair[0], 1), succeeded, fixtureNoon)
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	pagePath := filepath.Join(pair[0].path(root), pageName)
	if !regularFile(pagePath) {
		t.Fatal("job completed between polls was missed")
	}
	path := staleGallery(t, root)
	c.look(t.Context(), ledgerOf(t, []queue.Job{job}))
	if read(t, path) != "stale\n" {
		t.Fatal("same terminal job was republished")
	}
}

func TestFirstLookLeavesOldEnds(t *testing.T) {
	if firstLookAt(t, fixtureNoon.Add(-justFinished-time.Minute)) {
		t.Fatal("an old end was published again")
	}
}

// staleGallery writes over latest.html, so a test can see whether it is
// written again.
func staleGallery(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, latestName)
	if err := os.WriteFile(path, []byte("stale\n"), listing); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestGalleryWaitsAMinuteWhileJobsRun: ten seconds after the gallery was
// written, it is not written again.
func TestGalleryWaitsAMinuteWhileJobsRun(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	coordinator := New(settingsFor(root, nil, hands), io.Discard)
	coordinator.Start(jobFor(1, root, pair[0], 10))
	coordinator.tick(t.Context())
	path := staleGallery(t, root)
	hands.advance(TickEvery)
	coordinator.tick(t.Context())
	if read(t, path) != "stale\n" {
		t.Fatal("the gallery was written twice in ten seconds")
	}
}

// TestGalleryIsWrittenAfterAMinuteAndAtTheEnd is the anti-test.
func TestGalleryIsWrittenAfterAMinuteAndAtTheEnd(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 1)
	coordinator := New(settingsFor(root, nil, hands), io.Discard)
	job := jobFor(1, root, pair[0], 10)
	coordinator.Start(job)
	coordinator.tick(t.Context())
	path := staleGallery(t, root)
	hands.advance(galleryEvery)
	coordinator.tick(t.Context())
	if read(t, path) == "stale\n" {
		t.Fatal("the gallery was not written after a minute")
	}
	staleGallery(t, root)
	coordinator.End(job, succeeded)
	if read(t, path) == "stale\n" {
		t.Fatal("the gallery was not written at the end")
	}
}

// changingLedger is a ledger whose jobs a test changes while the
// coordinator reads it, behind a mock.
type changingLedger struct {
	lock sync.Mutex
	jobs []queue.Job
}

// set replaces one job in the ledger.
func (ledger *changingLedger) set(job queue.Job) {
	ledger.lock.Lock()
	defer ledger.lock.Unlock()
	ledger.jobs[job.Sequence-1] = job
}

// snapshot is the ledger now.
func (ledger *changingLedger) snapshot(context.Context) (queue.Snapshot,
	error) {
	ledger.lock.Lock()
	defer ledger.lock.Unlock()
	return queue.Snapshot{Jobs: append([]queue.Job{}, ledger.jobs...)}, nil
}

// runJobs makes four jobs' pictures, two jobs in each folder of the pair,
// at once, then marks each finished in the ledger.
func runJobs(root string, ledger *changingLedger, hands *clock) {
	var jobs sync.WaitGroup
	for sequence := range uint64(4) {
		folder := pair[sequence%2]
		jobs.Go(func() {
			for index := range picturesEach {
				hands.advance(TickEvery)
				name := fmt.Sprintf("%d-%d.png", sequence, index)
				target := filepath.Join(folder.path(root), name)
				os.WriteFile(target, []byte("picture"), listing)
				time.Sleep(time.Millisecond)
			}
			job := jobFor(sequence+1, root, folder, 2*picturesEach)
			done := finished(job, succeeded, hands.now())
			ledger.set(done)
		})
	}
	jobs.Wait()
}

// whole says whether a page or line read while it was being written is
// whole: absent, or complete.
func whole(path string, line bool) bool {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if line {
		return err == nil && wholeLine.Match(data)
	}
	return err == nil && strings.HasSuffix(string(data), "</html>\n")
}

// tornReads reads every page and the line until done closes, and counts
// the reads that were not whole.
func tornReads(root string, done <-chan struct{}) int {
	torn := 0
	for {
		select {
		case <-done:
			return torn
		default:
		}
		for _, path := range []string{filepath.Join(root, latestName),
			filepath.Join(pair[0].path(root), pageName)} {
			if !whole(path, false) {
				torn++
			}
		}
		if !whole(filepath.Join(root, progressName), true) {
			torn++
		}
	}
}

// TestServingFourJobsKeepsWholePages serves a ledger of four jobs in two
// folders, running at once, while a reader reads every page: no read is
// part of a page, and each folder's page ends with every picture.
func TestServingFourJobsKeepsWholePages(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	ledger := &changingLedger{}
	for sequence := range uint64(4) {
		folder := pair[sequence%2]
		stills(t, root, folder, 0, 1)
		ledger.jobs = append(ledger.jobs, jobFor(sequence+1, root, folder,
			2*picturesEach))
	}
	reading := NewMockLedger(gomock.NewController(t))
	reading.EXPECT().Status(gomock.Any()).DoAndReturn(ledger.snapshot).
		AnyTimes()
	settings := settingsFor(root, nil, hands)
	settings.Every = time.Millisecond
	ctx, stop := context.WithCancel(t.Context())
	served := make(chan struct{})
	go func() { New(settings, io.Discard).serve(ctx, reading); close(served) }()
	ran := make(chan struct{})
	go func() { runJobs(root, ledger, hands); close(ran) }()
	torn := tornReads(root, ran)
	want := fmt.Sprintf("%d pictures.", 2*picturesEach+1)
	deadline := time.Now().Add(5 * time.Second) // picked: many looks' time
	for _, folder := range pair {
		page := filepath.Join(folder.path(root), pageName)
		for !holds(page, want) && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !holds(page, want) {
			t.Errorf("%s does not say %s", folder.Name, want)
		}
	}
	stop()
	<-served
	if torn > 0 {
		t.Fatalf("a reader saw part of a page %d times", torn)
	}
}

// holds says whether a file there now holds the words.
func holds(path, words string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), words)
}

// TestASecondCoordinatorIsRefused: while one holds a queue's lock, a
// second is refused and told who holds it.
func TestASecondCoordinatorIsRefused(t *testing.T) {
	queueRoot := t.TempDir()
	first, err := lock(queueRoot, fixtureNoon)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	_, err = lock(queueRoot, fixtureNoon)
	holder := fmt.Sprintf("process %d", os.Getpid())
	if err == nil || !strings.Contains(err.Error(), holder) {
		t.Fatal("a second coordinator was not refused:", err)
	}
	serveSettings := settingsFor(renders(t), nil, &clock{at: fixtureNoon})
	if Serve(t.Context(), queueRoot, serveSettings, io.Discard) == nil {
		t.Fatal("Serve ran beside a held lock")
	}
}

// TestALockLetGoIsTaken is the anti-test: once the first lets go, the
// next takes the lock.
func TestALockLetGoIsTaken(t *testing.T) {
	queueRoot := t.TempDir()
	first, err := lock(queueRoot, fixtureNoon)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	second, err := lock(queueRoot, fixtureNoon)
	if err != nil {
		t.Fatal(err)
	}
	second.Close()
}
