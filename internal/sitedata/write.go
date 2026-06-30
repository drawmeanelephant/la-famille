package sitedata

import (
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/jsonutil"
)

// Write writes the meta data to the output directory.
func Write(outputDir string, metaData map[string]map[string]interface{}) error {
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "meta.json"), metaData); err != nil {
		return err
	}

	return nil
}
