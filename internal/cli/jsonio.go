package cli

import (
	"encoding/json"
	"io"
	"os"
)

func writeJSON(path string, value interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSONTo(file, value)
}

func writeJSONTo(w io.Writer, value interface{}) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	_, err = w.Write(bytes)
	return err
}
