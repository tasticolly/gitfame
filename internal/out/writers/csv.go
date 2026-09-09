package writers

import (
	"encoding/csv"
	"github.com/tasticolly/gitfame/internal/statistics"
	"os"
	"strconv"
)

type CSVWriter struct {
	writer *csv.Writer
}

func NewCSVWriter() *CSVWriter {
	return &CSVWriter{writer: csv.NewWriter(os.Stdout)}
}

func (cw *CSVWriter) WriteFrames(data []statistics.EntityFrame) error {
	allEntities := make([][]string, 0, len(data)+1)
	allEntities = append(allEntities,
		[]string{
			"Name",
			"Lines",
			"Commits",
			"Files",
		})
	for _, entity := range data {
		allEntities = append(allEntities,
			[]string{
				entity.PersonName,
				strconv.Itoa(entity.LinesCount),
				strconv.Itoa(entity.CommitsCount),
				strconv.Itoa(entity.FilesCount),
			})
	}
	err := cw.writer.WriteAll(allEntities)
	if err != nil {
		return err
	}

	return nil
}
