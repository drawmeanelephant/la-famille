package sitedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()

	metaData := map[string]map[string]interface{}{
		"index": {
			"title": "Home Page",
		},
	}

	err := Write(tempDir, metaData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 1. Check meta.json
	metaContent, err := os.ReadFile(filepath.Join(tempDir, "meta.json"))
	if err != nil {
		t.Fatalf("Failed to read meta.json: %v", err)
	}
	var readMeta map[string]map[string]interface{}
	if err := json.Unmarshal(metaContent, &readMeta); err != nil {
		t.Fatalf("Failed to parse meta.json: %v", err)
	}

	if readMeta["index"]["title"] != "Home Page" {
		t.Errorf("Unexpected meta content: %+v", readMeta)
	}
}
