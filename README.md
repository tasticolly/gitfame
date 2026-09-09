# gitfame

Attributes every line of a Git repository to the person who last touched it, at a given
revision, and prints the totals per person.

```
$ gitfame --repository=. --extensions='.go,.md' --order-by=lines

Name            Lines Commits Files
Nikolai Krekhov 22036 11      115
```

Three numbers per person, all measured at one revision rather than over the history:

- **Lines** — lines in the tree at that revision whose last change is theirs.
- **Commits** — distinct commits of theirs that survive in that tree.
- **Files** — files they appear in at all.

## Origin

This started as an exercise from the Yandex Go course, which supplies the specification, the
flag set and the integration suite in `test/integration`. That is the part I did not write.
The implementation under `internal/` and `cmd/` is mine, and so is everything described in
the next section, which the exercise did not ask for.

The point of keeping it public is the concurrency, not the flags.

## Why it is not one `git blame` at a time

The work is one `git blame --incremental` process per file, and process start-up dominates:
on a repository of a hundred-odd files the utility spends its time waiting on `fork`/`exec`,
not computing anything. The obvious version — a loop over the file list — leaves every core
but one idle.

Files are therefore blamed on a pool of workers (`--workers`, defaulting to `GOMAXPROCS`).
On this repository's own `splitr` sibling, 115 files:

```
--workers=1   1.45s
--workers=8   0.22s
```

Two decisions are worth spelling out, because they are what keeps the result trustworthy:

**The fold is not concurrent.** Workers only run `git blame`; the aggregate map is read and
written on the calling goroutine alone, fed by a channel. There is no mutex in
`internal/statistics` because there is nothing to guard. This is also why the output is
stable: the three quantities being folded are commutative, so the order in which files
finish cannot change the answer. `TestFoldDoesNotDependOnWorkerCount` asserts exactly that,
comparing the pool against a plain sequential fold for worker counts from 1 to 1000.

**The first error stops the rest.** Workers share a context that the first failure cancels,
which kills the `git blame` processes still running instead of blaming the remaining files
only to throw the work away. The same context is wired to SIGINT, so Ctrl-C does not leave
child processes behind.

## Flags

- `--repository` — path to the Git repository; current directory by default.
- `--revision` — commit to measure; `HEAD` by default.
- `--order-by` — sort key: `lines` (default), `commits`, `files`. Sorting is descending by
  the key, then by the remaining keys in that order, then lexicographically by name.
- `--use-committer` — attribute to the committer instead of the author.
- `--format`, `-f` — `tabular` (default), `csv`, `json`, `json-lines`.
- `--extensions` — restrict to these file extensions, e.g. `.go,.md`.
- `--languages` — restrict by language rather than extension, e.g. `go,markdown`.
- `--exclude` — Glob patterns to drop, e.g. `vendor/*,testdata/*`.
- `--restrict-to` — Glob patterns; anything matching none of them is dropped.
- `--workers` — files blamed in parallel; defaults to `GOMAXPROCS`. `--workers=1` gives the
  sequential behaviour back.

## Build and test

```
go install github.com/tasticolly/gitfame/cmd/gitfame@latest

go test -race ./...
go test -run '^$' -bench BenchmarkFold ./internal/statistics/
```

The integration suite unbundles the repositories in `test/integration/testdata/bundles` and
runs the built binary against them, comparing stdout with the recorded expectations.

## License

MIT, see [LICENSE](LICENSE).
