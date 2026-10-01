package diff

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

var runGit = func(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), bytes.TrimSpace(output), err)
	}
	return output, nil
}

// LoadInput reads a manifest from an output directory, directly from a
// manifest file, or from a Git revision. Relative filesystem inputs are
// resolved from projectRoot. A ref can be forced with the "ref:" prefix when
// it has the same name as a local path.
func LoadInput(input, outputDir, projectRoot string) (sitedata.Manifest, error) {
	return LoadInputWithBuilder(input, outputDir, projectRoot, nil)
}

// LoadInputWithBuilder additionally builds source-only Git revisions in a
// disposable archive. The callback must not run source scripts.
func LoadInputWithBuilder(input, outputDir, projectRoot string, builder func(string) (sitedata.Manifest, error)) (sitedata.Manifest, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return sitedata.Manifest{}, fmt.Errorf("input is empty")
	}

	ref := strings.TrimPrefix(input, "ref:")
	forceRef := ref != input
	if !forceRef {
		path := input
		if !filepath.IsAbs(path) && projectRoot != "" {
			path = filepath.Join(projectRoot, path)
		}
		info, err := os.Stat(path)
		switch {
		case err == nil && info.IsDir():
			return readManifest(filepath.Join(path, sitedata.ManifestFileName))
		case err == nil:
			return readManifest(path)
		case !os.IsNotExist(err):
			return sitedata.Manifest{}, fmt.Errorf("inspect diff input %q: %w", input, err)
		}
		ref = input
	}

	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			return sitedata.Manifest{}, fmt.Errorf("get current directory: %w", err)
		}
	}
	manifestPath, err := manifestPathForOutput(projectRoot, outputDir)
	if err != nil {
		return sitedata.Manifest{}, err
	}
	revision, err := runGit(projectRoot, "rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return sitedata.Manifest{}, fmt.Errorf("resolve Git ref %q: %w", ref, err)
	}
	revision = []byte(strings.TrimSpace(string(revision)))
	data, err := runGit(projectRoot, "show", string(revision)+":./"+manifestPath)
	if err != nil {
		if builder != nil {
			return buildRevision(projectRoot, string(revision), builder)
		}
		return sitedata.Manifest{}, fmt.Errorf("read %s from Git ref %q: %w", manifestPath, ref, err)
	}
	manifest, err := sitedata.ParseManifest(data)
	if err != nil {
		return sitedata.Manifest{}, fmt.Errorf("read manifest from Git ref %q: %w", ref, err)
	}
	return manifest, nil
}

func readManifest(path string) (sitedata.Manifest, error) {
	manifest, err := sitedata.ReadManifest(path)
	if err != nil {
		return sitedata.Manifest{}, fmt.Errorf("read manifest %q: %w", path, err)
	}
	return manifest, nil
}

func manifestPathForOutput(projectRoot, outputDir string) (string, error) {
	if outputDir == "" {
		outputDir = "public"
	}
	var err error
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root for Git ref: %w", err)
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(projectRoot, outputDir)
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory for Git ref: %w", err)
	}
	relPath, err := filepath.Rel(projectRoot, outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory for Git ref: %w", err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("configured output directory %q is outside project root; pass a manifest path instead", outputDir)
	}
	return filepath.ToSlash(filepath.Join(relPath, sitedata.ManifestFileName)), nil
}
