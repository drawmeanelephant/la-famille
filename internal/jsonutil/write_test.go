package jsonutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.json")

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestData{
		Name:  "Test",
		Value: 123,
	}

	err := WriteJSON(tempFile, data)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	var readData TestData
	err = json.Unmarshal(fileContent, &readData)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if readData.Name != data.Name || readData.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, readData)
	}
}
