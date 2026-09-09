package out

import (
	"github.com/tasticolly/gitfame/internal/out/preprocess"
	"github.com/tasticolly/gitfame/internal/out/writers"
	"github.com/tasticolly/gitfame/internal/statistics"
)

func WriteResult(data []statistics.EntityFrame, format, orderBy string) error {
	preprocess.Sort(data, orderBy)

	var writer statistics.EntityFrameWriter
	switch format {
	case "tabular":
		writer = writers.NewTabularWriter()
	case "csv":
		writer = writers.NewCSVWriter()
	case "json":
		writer = writers.NewJSONWriter()
	case "json-lines":
		writer = writers.NewJSONLinesWriter()
	default:
		panic("validation of formats don't give error" + format)
	}

	err := writer.WriteFrames(data)
	if err != nil {
		return err
	}

	return nil
}
