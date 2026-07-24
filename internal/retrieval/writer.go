package retrieval

import "os"

// writeArtifact persists a string to disk, used by chunker_test.go to keep
// the test fixtures readable. The helper lives here (not in the test file)
// only because the test file already imports the package's surface and
// adding yet another file keeps the import set minimal. Permissions are
// conservative so CI never accidentally creates world-writable fixtures.
func writeArtifact(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o600)
}
