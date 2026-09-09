package statistics

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tasticolly/gitfame/internal/git"
)

// fakeBlame stands in for git.GetInfo. Every file is attributed to two of three
// authors, so the folded result mixes contributions from several files per
// person, which is the part the worker pool could plausibly get wrong.
func fakeBlame(_ context.Context, _, _, filename string, _ bool) (map[string]*git.CommitInfo, error) {
	n := len(filename)
	return map[string]*git.CommitInfo{
		fmt.Sprintf("commit-%d", n%7):  {Person: "alice", NumOfLines: n},
		fmt.Sprintf("commit-%d", n%11): {Person: "bob", NumOfLines: 2 * n},
	}, nil
}

func manyFiles(count int) []string {
	files := make([]string, count)
	for i := range files {
		files[i] = strings.Repeat("f", i%37+1) + fmt.Sprintf("/%d.go", i)
	}
	return files
}

func sorted(frames []EntityFrame) []EntityFrame {
	out := append([]EntityFrame(nil), frames...)
	sort.Slice(out, func(i, j int) bool { return out[i].PersonName < out[j].PersonName })
	return out
}

// sequentialFold is the obvious single-goroutine implementation, kept here as
// the reference the concurrent one has to agree with.
func sequentialFold(t *testing.T, files []string) []EntityFrame {
	t.Helper()
	perPerson := map[string]*internalStats{}
	for _, f := range files {
		info, err := fakeBlame(context.Background(), "", "", f, false)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]struct{}{}
		for hash, ci := range info {
			seen[ci.Person] = struct{}{}
			st, ok := perPerson[ci.Person]
			if !ok {
				st = newInternalStats()
				perPerson[ci.Person] = st
			}
			st.CommitSet[hash] = struct{}{}
			st.LinesCount += ci.NumOfLines
		}
		for p := range seen {
			perPerson[p].FilesCount++
		}
	}
	out := make([]EntityFrame, 0, len(perPerson))
	for p, st := range perPerson {
		out = append(out, EntityFrame{p, st.LinesCount, len(st.CommitSet), st.FilesCount})
	}
	return sorted(out)
}

func TestFoldDoesNotDependOnWorkerCount(t *testing.T) {
	files := manyFiles(500)
	want := sequentialFold(t, files)

	for _, workers := range []int{1, 2, 3, 8, 64, 1000, 0, -1} {
		got, err := calculate(context.Background(), fakeBlame, "repo", "HEAD", files, false, workers)
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}
		if !reflect.DeepEqual(sorted(got), want) {
			t.Fatalf("workers=%d: got %v, want %v", workers, sorted(got), want)
		}
	}
}

func TestEmptyFileListYieldsNoFrames(t *testing.T) {
	got, err := calculate(context.Background(), fakeBlame, "repo", "HEAD", nil, false, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d frames, want 0", len(got))
	}
}

func TestFirstBlameErrorIsReturnedAndNamesTheFile(t *testing.T) {
	boom := errors.New("blame exploded")
	blame := func(ctx context.Context, repo, rev, filename string, useCommitter bool) (map[string]*git.CommitInfo, error) {
		if filename == "ffff/3.go" {
			return nil, boom
		}
		return fakeBlame(ctx, repo, rev, filename, useCommitter)
	}

	_, err := calculate(context.Background(), blame, "repo", "HEAD", manyFiles(500), false, 8)
	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want it to wrap %v", err, boom)
	}
	if !strings.Contains(err.Error(), "ffff/3.go") {
		t.Fatalf("error %q does not name the failing file", err)
	}
}

// A failing file must stop the remaining work rather than let the pool blame
// every other file first.
func TestErrorCancelsRemainingBlames(t *testing.T) {
	var started, cancelledMidflight atomic.Int64
	boom := errors.New("blame exploded")

	blame := func(ctx context.Context, repo, rev, filename string, useCommitter bool) (map[string]*git.CommitInfo, error) {
		if started.Add(1) == 1 {
			return nil, boom
		}
		select {
		case <-ctx.Done():
			cancelledMidflight.Add(1)
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return fakeBlame(ctx, repo, rev, filename, useCommitter)
		}
	}

	files := manyFiles(2000)
	_, err := calculate(context.Background(), blame, "repo", "HEAD", files, false, 4)
	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want it to wrap %v", err, boom)
	}
	if got := started.Load(); got >= int64(len(files)) {
		t.Fatalf("started %d blames out of %d; the pool did not stop early", got, len(files))
	}
}

func TestCancelledContextStopsTheFold(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := calculate(ctx, fakeBlame, "repo", "HEAD", manyFiles(500), false, 4)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want nil or context.Canceled", err)
	}
}

// slowBlame models the real cost: one `git blame` process per file, dominated by
// process start-up rather than by anything this package computes.
func slowBlame(ctx context.Context, repo, rev, filename string, useCommitter bool) (map[string]*git.CommitInfo, error) {
	time.Sleep(200 * time.Microsecond)
	return fakeBlame(ctx, repo, rev, filename, useCommitter)
}

func BenchmarkFold(b *testing.B) {
	files := manyFiles(400)
	for _, workers := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := calculate(context.Background(), slowBlame, "repo", "HEAD", files, false, workers); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
