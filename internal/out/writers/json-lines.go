package writers

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tasticolly/gitfame/internal/statistics"
	"io"
	"os"
)

type JSONLinesWriter struct {
	io.Writer
}

func NewJSONLinesWriter() *JSONLinesWriter {
	return &JSONLinesWriter{os.Stdout}
}

func (jw *JSONLinesWriter) WriteFrames(data []statistics.EntityFrame) error {

	for _, entity := range data {
		encodedJSON, err := json.Marshal(entity)
		if err != nil {
			return errors.Wrap(err, "failed to encode json data in output")
		}

		_, err = fmt.Fprint(jw, string(encodedJSON), "\n")
		if err != nil {
			return errors.Wrap(err, "failed to write json data in output")
		}
	}
	return nil
}
