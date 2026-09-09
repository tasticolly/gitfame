package statistics

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/tasticolly/gitfame/internal/git"
)

type EntityFrame struct {
	PersonName   string `json:"name"`
	LinesCount   int    `json:"lines"`
	CommitsCount int    `json:"commits"`
	FilesCount   int    `json:"files"`
}

type EntityFrameWriter interface {
	WriteFrames(data []EntityFrame) error
}

type internalStats struct {
	FilesCount int
	LinesCount int
	CommitSet  map[string]struct{}
}

func newInternalStats() *internalStats {
	return &internalStats{CommitSet: make(map[string]struct{})}
}

// DefaultWorkers is the fan-out used when the caller does not pick one.
// The work is one `git blame` process per file, so the useful degree of
// parallelism follows the number of cores available to this process.
func DefaultWorkers() int {
	return runtime.GOMAXPROCS(0)
}

// CalculateStatistic blames every file in fileNames and folds the result into
// one frame per person.
//
// Each file is blamed by a separate `git blame` process and those processes are
// independent, so they run on a pool of workers. Folding, on the other hand,
// stays on the calling goroutine: the aggregate is read and written on one
// goroutine only, which is why there is no lock on it anywhere below. The three
// quantities being folded (lines, the set of commits, the file count) are all
// commutative, so the result does not depend on the order in which files finish
// and repeated runs over the same revision produce identical output.
//
// The first error cancels ctx, which kills the `git blame` processes still
// running instead of waiting for them.
func CalculateStatistic(
	ctx context.Context,
	repository, revision string,
	fileNames []string,
	useCommitter bool,
	workers int,
) ([]EntityFrame, error) {
	return calculate(ctx, git.GetInfo, repository, revision, fileNames, useCommitter, workers)
}

// blamer is the shape of git.GetInfo. Naming it lets the pool below be tested
// without a repository on disk and without spawning processes.
type blamer func(ctx context.Context, repository, revision, filename string, useCommitter bool) (map[string]*git.CommitInfo, error)

func calculate(
	ctx context.Context,
	blame blamer,
	repository, revision string,
	fileNames []string,
	useCommitter bool,
	workers int,
) ([]EntityFrame, error) {
	personToStats := make(map[string]*internalStats)

	if len(fileNames) == 0 {
		return []EntityFrame{}, nil
	}

	if workers <= 0 {
		workers = DefaultWorkers()
	}
	if workers > len(fileNames) {
		workers = len(fileNames)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan string)
	results := make(chan map[string]*git.CommitInfo, workers)

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	fail := func(err error) {
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filename := range jobs {
				commitsInfoFromFile, err := blame(ctx, repository, revision, filename, useCommitter)
				if err != nil {
					// A cancelled context means some other worker already
					// failed and reported the real error; do not bury it.
					if ctx.Err() == nil {
						fail(fmt.Errorf("%s: %w", filename, err))
					}
					return
				}
				select {
				case results <- commitsInfoFromFile:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, filename := range fileNames {
			select {
			case jobs <- filename:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Draining results to completion is what makes reading firstErr safe: the
	// channel closes only after every worker has returned.
	for commitsInfoFromFile := range results {
		currentPersons := make(map[string]struct{})
		for commitHash, currentCommitInfo := range commitsInfoFromFile {
			currentPersons[currentCommitInfo.Person] = struct{}{}

			stats, ok := personToStats[currentCommitInfo.Person]
			if !ok {
				stats = newInternalStats()
				personToStats[currentCommitInfo.Person] = stats
			}
			stats.CommitSet[commitHash] = struct{}{}
			stats.LinesCount += currentCommitInfo.NumOfLines
		}
		for person := range currentPersons {
			personToStats[person].FilesCount++
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	result := make([]EntityFrame, 0, len(personToStats))
	for person, internalStatPtr := range personToStats {
		result = append(result, EntityFrame{
			PersonName:   person,
			LinesCount:   internalStatPtr.LinesCount,
			CommitsCount: len(internalStatPtr.CommitSet),
			FilesCount:   internalStatPtr.FilesCount,
		})
	}
	return result, nil
}
