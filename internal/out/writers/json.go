package writers

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/tasticolly/gitfame/internal/statistics"
	"io"
	"os"
)

type JSONWriter struct {
	io.Writer
}

func NewJSONWriter() *JSONWriter {
	return &JSONWriter{os.Stdout}
}

func (jw *JSONWriter) WriteFrames(data []statistics.EntityFrame) error {
	encodedJSON, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to encode json data in output")
	}
	_, err = jw.Write(encodedJSON)
	if err != nil {
		return errors.Wrap(err, "failed to write json data in output")
	}
	return nil
}
