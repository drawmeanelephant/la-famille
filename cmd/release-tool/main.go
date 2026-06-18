package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	versionFile := flag.String("version-file", "VERSION", "Path to VERSION file")
	_ = flag.String("changelog", "CHANGELOG.md", "Path to CHANGELOG.md")
	albumFile := flag.String("album", "content/album_1_boom_bap.md", "Path to the current album file")
	releaseDocsFile := flag.String("release-docs", "content/docs/releases.md", "Path to release docs")
	trackTitle := flag.String("track-title", "", "Title of the new soundtrack track")
	trackLyrics := flag.String("track-lyrics", "", "Lyrics/Description of the new track")

	flag.Parse()

	versionBytes, err := ioutil.ReadFile(*versionFile)
	if err != nil {
		log.Fatalf("failed to read version file: %v", err)
	}
	version := strings.TrimSpace(string(versionBytes))

	if *trackTitle == "" {
		*trackTitle = fmt.Sprintf("Release %s", version)
	}

	// 1. Create soundtrack track file
	trackFilename := fmt.Sprintf("content/soundtrack/%s.md", strings.ReplaceAll(version, ".", "-"))
	trackContent := fmt.Sprintf("---\ntitle: \"%s\"\nauthor: \"Jules\"\ndate: \"%s\"\n---\n# %s\n\n%s\n",
		*trackTitle, time.Now().Format("2006-01-02"), *trackTitle, *trackLyrics)

	if err := ioutil.WriteFile(trackFilename, []byte(trackContent), 0644); err != nil {
		log.Fatalf("failed to write track file: %v", err)
	}
	fmt.Printf("Created track file: %s\n", trackFilename)

	// 2. Append to album
	if err := appendToAlbum(*albumFile, *trackTitle, version); err != nil {
		log.Printf("Warning: failed to append to album: %v", err)
	}

	// 3. Update Release Docs
	if err := updateReleaseDocs(*releaseDocsFile, version, *trackTitle); err != nil {
		log.Printf("Warning: failed to update release docs: %v", err)
	}

	fmt.Println("Release artifacts generated successfully.")
}

func appendToAlbum(path, title, version string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	foundTrackListing := false
	lastTrackNum := 0

	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.HasPrefix(line, "## Track Listing") {
			foundTrackListing = true
		}
		if foundTrackListing && strings.Contains(line, ". **\"") {
			fmt.Sscanf(line, "%d.", &lastTrackNum)
		}
	}

	if foundTrackListing {
		newTrackLine := fmt.Sprintf("%d. **\"%s\"** - The %s drop.", lastTrackNum+1, title, version)
		newLines = append(newLines, newTrackLine)
	}

	return ioutil.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

func updateReleaseDocs(path, version, title string) error {
	header := "---\ntitle: \"Releases\"\n---\n# Release History\n\n"
	newEntry := fmt.Sprintf("## [%s] - %s\n- **Featured Track:** [%s](../soundtrack/%s.md)\n\n",
		version, time.Now().Format("2006-01-02"), title, strings.ReplaceAll(version, ".", "-"))

	content, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return ioutil.WriteFile(path, []byte(header+newEntry), 0644)
	} else if err != nil {
		return err
	}

	body := string(content)
	if strings.Contains(body, "# Release History") {
		parts := strings.SplitN(body, "# Release History\n\n", 2)
		if len(parts) == 2 {
			newBody := parts[0] + "# Release History\n\n" + newEntry + parts[1]
			return ioutil.WriteFile(path, []byte(newBody), 0644)
		}
	}

	return ioutil.WriteFile(path, []byte(string(content)+"\n"+newEntry), 0644)
}
