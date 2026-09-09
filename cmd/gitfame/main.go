package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"github.com/tasticolly/gitfame/internal/git"
	"github.com/tasticolly/gitfame/internal/out"
	"github.com/tasticolly/gitfame/internal/selection"
	"github.com/tasticolly/gitfame/internal/statistics"
	"github.com/tasticolly/gitfame/internal/validaton"
	"os"
	"sync/atomic"
	"time"
)

func handleError(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

var finished atomic.Bool

func spinningLine() {
	box := `|/-\`
	delay := time.Millisecond * 50

	for pos := 0; !finished.Load(); pos = (pos + 1) % len(box) {
		fmt.Fprint(os.Stderr, string(box[pos]))
		time.Sleep(delay)
		fmt.Fprint(os.Stderr, "\b")
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
)

func main() {

	//go spinningLine()

	flag.Parse()

	err := validaton.Flags(
		*flagRepository, *flagRevision, *flagOrder, *flagFormat,
		*flagExtensions, *flagLanguages, *flagExclude, *flagRestrictTo,
	)
	handleError(err)

	files, err := git.GetFileNames(*flagRepository, *flagRevision)
	handleError(err)

	files, err = selection.SelectFiles(files, *flagExtensions, *flagLanguages, *flagExclude, *flagRestrictTo)
	handleError(err)

	statistic, err := statistics.CalculateStatistic(*flagRepository, *flagRevision, files, *flagUseCommitter)
	handleError(err)
	finished.Store(true)

	err = out.WriteResult(statistic, *flagFormat, *flagOrder)
	handleError(err)

}
