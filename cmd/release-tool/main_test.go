package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestAppendToAlbum(t *testing.T) {
	content := "## Track Listing\n\n1. **\"Track 1\"** - Drop 1."
	tmpfile, err := ioutil.TempFile("", "album")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	err = appendToAlbum(tmpfile.Name(), "Track 2", "0.2.0")
	if err != nil {
		t.Fatalf("appendToAlbum failed: %v", err)
	}

	newContent, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "2. **\"Track 2\"** - The 0.2.0 drop."
	if !strings.Contains(string(newContent), expected) {
		t.Errorf("Expected content to contain %q, but got %q", expected, string(newContent))
	}
}

func TestUpdateReleaseDocs(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "releases")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	os.Remove(tmpfile.Name()) // Start with non-existent file

	err = updateReleaseDocs(tmpfile.Name(), "0.1.0", "Release 0.1.0")
	if err != nil {
		t.Fatalf("updateReleaseDocs failed: %v", err)
	}

	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "## [0.1.0]") {
		t.Errorf("Expected content to contain release entry, but got %q", string(content))
	}
}
