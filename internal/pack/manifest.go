// Package pack builds and verifies content-only Corpus Packs without extraction.
package pack

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	SchemaVersion   = 1
	ManifestName    = "pack-manifest.json"
	MaxManifestSize = 1 << 20
	MaxMemberSize   = 64 << 20
	MaxTotalSize    = 256 << 20
	MaxArchiveSize  = 272 << 20
	MaxMembers      = 4096
	MaxPathBytes    = 240
)

type Site struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Provenance identifies the pack producer, not a claimed source revision.
// Unknown release fields are omitted; no wall-clock packing time is recorded.
type Provenance struct {
	Generator string `json:"generator"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	Target    string `json:"target,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
}

type Member struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	SchemaVersion int        `json:"schema_version"`
	Site          Site       `json:"site"`
	Provenance    Provenance `json:"provenance"`
	Members       []Member   `json:"members"`
	ContentRoot   string     `json:"content_root"`
}

// contentRoot hashes the compact JSON member array in path order, without a
// newline. Site identity and producer provenance are intentionally not payload.
func contentRoot(members []Member) string {
	data, _ := json.Marshal(members)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func validatePath(name string) error {
	if name == "" || len(name) > MaxPathBytes || !utf8.ValidString(name) ||
		path.IsAbs(name) || path.Clean(name) != name || name == "." ||
		strings.ContainsAny(name, "\\:") {
		return fmt.Errorf("unsafe member path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." || part == "" {
			return fmt.Errorf("unsafe member path %q", name)
		}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("unsafe member path %q", name)
		}
	}
	return nil
}

func validHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (m Manifest) validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported pack schema version %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Site.Name) == "" || m.Provenance.Generator == "" || m.Provenance.Version == "" {
		return fmt.Errorf("manifest requires site name and producer generator/version")
	}
	if len(m.Members) == 0 || len(m.Members) > MaxMembers {
		return fmt.Errorf("manifest member count must be 1..%d", MaxMembers)
	}
	seen := make(map[string]bool, len(m.Members))
	var total int64
	for i, member := range m.Members {
		if err := validatePath(member.Path); err != nil {
			return err
		}
		if member.Path == ManifestName || seen[member.Path] {
			return fmt.Errorf("duplicate or reserved manifest member %q", member.Path)
		}
		seen[member.Path] = true
		if i > 0 && m.Members[i-1].Path > member.Path {
			return fmt.Errorf("manifest members must be sorted by path")
		}
		if member.Size < 0 || member.Size > MaxMemberSize {
			return fmt.Errorf("member %q exceeds size bound 0..%d", member.Path, MaxMemberSize)
		}
		total += member.Size
		if total > MaxTotalSize {
			return fmt.Errorf("payload exceeds total size bound %d", MaxTotalSize)
		}
		if !validHash(member.SHA256) {
			return fmt.Errorf("member %q has malformed SHA256", member.Path)
		}
	}
	if !validHash(m.ContentRoot) || m.ContentRoot != contentRoot(m.Members) {
		return fmt.Errorf("content root mismatch: expected %s, actual %s", m.ContentRoot, contentRoot(m.Members))
	}
	return nil
}

func parseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if !utf8.Valid(data) {
		return m, fmt.Errorf("manifest is not UTF-8")
	}
	// encoding/json otherwise accepts repeated keys and null scalar fields.
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSON(tokens, 0); err != nil {
		return m, fmt.Errorf("malformed manifest: %w", err)
	}
	if _, err := tokens.Token(); err != io.EOF {
		return m, fmt.Errorf("malformed manifest: trailing JSON data")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return m, fmt.Errorf("malformed manifest: %w", err)
	}
	// Required zero-valued fields (url and size) need a presence check.
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if err := checkFields(fields, []string{"schema_version", "site", "provenance", "members", "content_root"}); err != nil {
		return m, err
	}
	var site map[string]json.RawMessage
	_ = json.Unmarshal(fields["site"], &site)
	if err := checkFields(site, []string{"name", "url"}); err != nil {
		return m, err
	}
	var provenance map[string]json.RawMessage
	_ = json.Unmarshal(fields["provenance"], &provenance)
	if err := checkFields(provenance, []string{"generator", "version"}, "commit", "build_date", "target", "go_version"); err != nil {
		return m, err
	}
	var members []map[string]json.RawMessage
	_ = json.Unmarshal(fields["members"], &members)
	for _, member := range members {
		if err := checkFields(member, []string{"path", "size", "sha256"}); err != nil {
			return m, err
		}
	}
	return m, m.validate()
}

func checkFields(fields map[string]json.RawMessage, required []string, optional ...string) error {
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, name := range required {
		if _, ok := fields[name]; !ok {
			return fmt.Errorf("malformed manifest: missing field %q", name)
		}
		allowed[name] = true
	}
	for _, name := range optional {
		allowed[name] = true
	}
	// encoding/json matches struct fields case-insensitively. The wire schema
	// does not, so also reject aliases such as "size" plus "Size".
	for name := range fields {
		if !allowed[name] {
			return fmt.Errorf("malformed manifest: unknown field %q", name)
		}
	}
	return nil
}

func checkJSON(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting exceeds 32")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("null is not allowed")
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= 32 {
		return fmt.Errorf("JSON nesting exceeds 32")
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid JSON key %q", key)
			}
			seen[name] = true
			if err := checkJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := checkJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	_, err = decoder.Token()
	return err
}
