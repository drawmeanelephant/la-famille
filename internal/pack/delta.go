package pack

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"
)

const (
	DeltaVersion = 1
	DeltaName    = "pack-delta.json"
)

// DeltaMetadata binds a delta to the exact base archive, including its manifest
// and headers, and to the complete target manifest bytes.
type DeltaMetadata struct {
	SchemaVersion        int      `json:"schema_version"`
	BaseSHA256           string   `json:"base_sha256"`
	TargetManifestSHA256 string   `json:"target_manifest_sha256"`
	Added                []string `json:"added"`
	Changed              []string `json:"changed"`
	Removed              []string `json:"removed"`
}

func (d DeltaMetadata) validate() error {
	if d.SchemaVersion != DeltaVersion {
		return fmt.Errorf("unsupported delta schema version %d", d.SchemaVersion)
	}
	if !validHash(d.BaseSHA256) || !validHash(d.TargetManifestSHA256) {
		return fmt.Errorf("malformed delta SHA256")
	}
	seen := make(map[string]bool)
	for _, paths := range [][]string{d.Added, d.Changed, d.Removed} {
		if paths == nil || len(paths) > MaxMembers {
			return fmt.Errorf("delta paths must be arrays of at most %d members", MaxMembers)
		}
		for i, name := range paths {
			if err := validatePath(name); err != nil {
				return err
			}
			if name == ManifestName || seen[name] {
				return fmt.Errorf("duplicate or reserved delta path %q", name)
			}
			if i > 0 && paths[i-1] >= name {
				return fmt.Errorf("delta paths must be sorted")
			}
			seen[name] = true
		}
	}
	if len(d.Added)+len(d.Changed) > MaxMembers {
		return fmt.Errorf("delta payload exceeds %d members", MaxMembers)
	}
	return nil
}

func parseDelta(data []byte) (DeltaMetadata, error) {
	var d DeltaMetadata
	if !utf8.Valid(data) {
		return d, fmt.Errorf("delta metadata is not UTF-8")
	}
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSON(tokens, 0); err != nil {
		return d, fmt.Errorf("malformed delta metadata: %w", err)
	}
	if _, err := tokens.Token(); err != io.EOF {
		return d, fmt.Errorf("malformed delta metadata: trailing JSON data")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return d, fmt.Errorf("malformed delta metadata: %w", err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if err := checkFields(fields, []string{
		"schema_version", "base_sha256", "target_manifest_sha256", "added", "changed", "removed",
	}); err != nil {
		return d, fmt.Errorf("delta metadata: %w", err)
	}
	return d, d.validate()
}

func readDeltaJSON(r io.Reader, name string) ([]byte, error) {
	header, err := nextHeader(r)
	if err != nil {
		return nil, fmt.Errorf("read delta %s header: %w", name, err)
	}
	if header.Name != name || header.Size > MaxManifestSize {
		return nil, fmt.Errorf("expected %s, at most %d bytes", name, MaxManifestSize)
	}
	data, err := io.ReadAll(io.LimitReader(r, header.Size))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != header.Size {
		return nil, fmt.Errorf("truncated delta %s", name)
	}
	if err := readPadding(r, header.Size); err != nil {
		return nil, err
	}
	return data, nil
}

// Payload names are indices into the sorted union of added and changed paths.
// This leaves all valid v1 member names available, even pack-delta.json.
func deltaPayloadName(i int) string { return fmt.Sprintf("payload/%04d", i) }

func deltaPayloadMembers(d DeltaMetadata, target Manifest) ([]Member, error) {
	paths := append(append([]string{}, d.Added...), d.Changed...)
	sort.Strings(paths)
	index := memberIndex(target)
	payload := make([]Member, 0, len(paths))
	for i, name := range paths {
		member, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("delta payload %q is absent from target manifest", name)
		}
		member.Path = deltaPayloadName(i)
		payload = append(payload, member)
	}
	for _, name := range d.Removed {
		if _, ok := index[name]; ok {
			return nil, fmt.Errorf("removed delta path %q remains in target manifest", name)
		}
	}
	return payload, nil
}

func verifyDelta(input io.Reader) (DeltaMetadata, Manifest, []byte, map[string]int64, error) {
	r := &countingReader{Reader: io.LimitReader(input, MaxArchiveSize+1)}
	data, err := readDeltaJSON(r, DeltaName)
	if err != nil {
		return DeltaMetadata{}, Manifest{}, nil, nil, err
	}
	d, err := parseDelta(data)
	if err != nil {
		return d, Manifest{}, nil, nil, err
	}
	targetData, err := readDeltaJSON(r, ManifestName)
	if err != nil {
		return d, Manifest{}, nil, nil, err
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(targetData))
	if actual != d.TargetManifestSHA256 {
		return d, Manifest{}, nil, nil, fmt.Errorf("target manifest SHA256 mismatch: expected %s, actual %s", d.TargetManifestSHA256, actual)
	}
	target, err := parseManifest(targetData)
	if err != nil {
		return d, target, nil, nil, err
	}
	payload, err := deltaPayloadMembers(d, target)
	if err != nil {
		return d, target, nil, nil, err
	}
	offsets, err := verifyMembers(r, payload, DeltaName, ManifestName)
	if err != nil {
		return d, target, nil, nil, err
	}
	paths := append(append([]string{}, d.Added...), d.Changed...)
	sort.Strings(paths)
	named := make(map[string]int64, len(paths))
	for i, name := range paths {
		named[name] = offsets[deltaPayloadName(i)]
	}
	return d, target, targetData, named, nil
}

func loadDelta(name string) (*snapshot, DeltaMetadata, error) {
	var d DeltaMetadata
	s, err := captureSnapshot(name, func(r io.Reader, s *snapshot) error {
		var err error
		d, s.manifest, s.manifestJSON, s.offsets, err = verifyDelta(r)
		return err
	})
	return s, d, err
}
