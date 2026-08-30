package report

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type JSONWriter struct{}

func NewJSONWriter() *JSONWriter { return &JSONWriter{} }

func (w *JSONWriter) Write(r Report, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(destination, "report.json"), data, 0o644)
}
