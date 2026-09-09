package writers

import (
	"fmt"
	"github.com/tasticolly/gitfame/internal/statistics"
	"os"
	"text/tabwriter"
)

type TabularWriter struct {
	writer *tabwriter.Writer
}

func NewTabularWriter() *TabularWriter {
	return &TabularWriter{writer: tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)}
}

func (tw *TabularWriter) WriteFrames(data []statistics.EntityFrame) error {
	_, err := fmt.Fprint(tw.writer, "Name\tLines\tCommits\tFiles\n")
	if err != nil {
		return err
	}
	for _, entity := range data {
		_, err = fmt.Fprintf(tw.writer, "%s\t%d\t%d\t%d\n",
			entity.PersonName, entity.LinesCount, entity.CommitsCount, entity.FilesCount)
		if err != nil {
			return err
		}
	}
	return tw.writer.Flush()
}
