package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"

	"github.com/tasticolly/gitfame/internal/git"
	"github.com/tasticolly/gitfame/internal/out"
	"github.com/tasticolly/gitfame/internal/selection"
	"github.com/tasticolly/gitfame/internal/statistics"
	"github.com/tasticolly/gitfame/internal/validaton"
)

func handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	flagRepository   = flag.String("repository", ".", "path to git repository")
	flagRevision     = flag.String("revision", "HEAD", "revision to checkout")
	flagOrder        = flag.String("order-by", "lines", "key to sort result")
	flagUseCommitter = flag.Bool("use-committer", false, "calculate statistic for committer instead author")
	flagFormat       = flag.StringP("format", "f", "tabular", "output format")
	flagExtensions   = flag.StringSlice("extensions", nil, "list of file extensions for which statistics will be calculated")
	flagLanguages    = flag.StringSlice("languages", nil, "list of languages for which statistics will be calculated")
	flagExclude      = flag.StringSlice("exclude", nil, "list of Glob expressions such that files matching them will be excluded from statistics calculations")
	flagRestrictTo   = flag.StringSlice("restrict-to", nil, "list of Glob expressions such that if a file does not match any of them, then this file will be excluded from the statistics calculation")
	flagWorkers      = flag.Int("workers", statistics.DefaultWorkers(), "number of files blamed in parallel; defaults to GOMAXPROCS")
)

func main() {
	flag.Parse()

	// Ctrl-C has to reach the git processes, not just this one: without a
	// cancellable context the pool would keep several `git blame` runs alive
	// after the user has already given up on the command.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := validaton.Flags(
		ctx,
		*flagRepository, *flagRevision, *flagOrder, *flagFormat,
		*flagExtensions, *flagLanguages, *flagExclude, *flagRestrictTo,
	)
	handleError(err)

	files, err := git.GetFileNames(ctx, *flagRepository, *flagRevision)
	handleError(err)

	files, err = selection.SelectFiles(files, *flagExtensions, *flagLanguages, *flagExclude, *flagRestrictTo)
	handleError(err)

	statistic, err := statistics.CalculateStatistic(
		ctx, *flagRepository, *flagRevision, files, *flagUseCommitter, *flagWorkers,
	)
	handleError(err)

	err = out.WriteResult(statistic, *flagFormat, *flagOrder)
	handleError(err)
}
