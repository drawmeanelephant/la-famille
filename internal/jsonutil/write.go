package jsonutil

import (
	"encoding/json"
	"os"
)

// WriteJSON writes the given data to the specified path as a formatted JSON file.
func WriteJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
