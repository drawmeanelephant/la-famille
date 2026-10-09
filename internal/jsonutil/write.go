package jsonutil

import (
	"encoding/json"
	"os"
)

// WriteJSON writes the given data to the specified path as a formatted JSON
// file. Callers publish the result as part of the site artifact, so the mode
// matches every other generated file: 0644, subject to the process umask.
// Owner-only 0600 files 403 for a web server running as another user (#637).
func WriteJSON(path string, data interface{}) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
