package pack

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

// Diff verifies both complete packs and writes a deterministic member delta.
func Diff(beforeFile, afterFile, destination string) (Comparison, error) {
	before, err := loadPack(beforeFile)
	if err != nil {
		return Comparison{}, fmt.Errorf("before pack: %w", err)
	}
	defer before.close()
	after, err := loadPack(afterFile)
	if err != nil {
		return Comparison{}, fmt.Errorf("after pack: %w", err)
	}
	defer after.close()
	report := compareMembers(before.manifest, after.manifest)
	report.BaseSHA256 = before.hash
	addLedger(&report, before, after)
	d := DeltaMetadata{
		SchemaVersion: DeltaVersion, BaseSHA256: before.hash,
		TargetManifestSHA256: fmt.Sprintf("%x", sha256.Sum256(after.manifestJSON)),
		Added:                report.Added, Changed: report.Changed, Removed: report.Removed,
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return Comparison{}, err
	}
	data = append(data, '\n')
	if len(data) > MaxManifestSize {
		return Comparison{}, fmt.Errorf("delta metadata exceeds %d bytes", MaxManifestSize)
	}
	old := memberIndex(before.manifest)
	err = publishArchive(destination, func(w io.Writer) error {
		writer := tar.NewWriter(w)
		if err := writeJSONMember(writer, DeltaName, data); err != nil {
			return err
		}
		if err := writeJSONMember(writer, ManifestName, after.manifestJSON); err != nil {
			return err
		}
		i := 0
		for _, member := range after.manifest.Members {
			if base, included := old[member.Path]; included && base == member {
				continue
			}
			if err := copyVerifiedMember(writer, deltaPayloadName(i), member, after.member(member)); err != nil {
				return err
			}
			i++
		}
		return writer.Close()
	}, func(r io.Reader) error {
		_, _, _, _, err := verifyDelta(r)
		return err
	})
	if err != nil {
		return Comparison{}, err
	}
	return report, nil
}

// Apply never alters the base and only publishes a newly verified destination.
// Unchanged members come from the verified base snapshot, including unknown
// members; replacements come from the verified delta snapshot.
func Apply(baseFile, deltaFile, destination string) (Manifest, error) {
	base, err := loadPack(baseFile)
	if err != nil {
		return Manifest{}, fmt.Errorf("base pack: %w", err)
	}
	defer base.close()
	delta, d, err := loadDelta(deltaFile)
	if err != nil {
		return Manifest{}, fmt.Errorf("delta: %w", err)
	}
	defer delta.close()
	return applySnapshots(base, delta, d, destination, "")
}

func applySnapshots(base, delta *snapshot, d DeltaMetadata, destination, targetSHA256 string) (Manifest, error) {
	if base.hash != d.BaseSHA256 {
		return Manifest{}, fmt.Errorf("wrong base SHA256: expected %s, actual %s", d.BaseSHA256, base.hash)
	}
	changes := compareMembers(base.manifest, delta.manifest)
	if !slices.Equal(changes.Added, d.Added) || !slices.Equal(changes.Changed, d.Changed) ||
		!slices.Equal(changes.Removed, d.Removed) {
		return Manifest{}, fmt.Errorf("delta inventory does not match base and target manifests")
	}
	err := publishArchive(destination, func(w io.Writer) error {
		writer := tar.NewWriter(w)
		if err := writeJSONMember(writer, ManifestName, delta.manifestJSON); err != nil {
			return err
		}
		for _, member := range delta.manifest.Members {
			src := base
			if _, replaced := delta.offsets[member.Path]; replaced {
				src = delta
			}
			if err := copyVerifiedMember(writer, member.Path, member, src.member(member)); err != nil {
				return err
			}
		}
		return writer.Close()
	}, func(r io.Reader) error {
		return verifyTargetArchive(r, targetSHA256)
	})
	if err != nil {
		return Manifest{}, err
	}
	return delta.manifest, nil
}

// The advertised identity is checked on the same bytes that were verified,
// before publication. Local apply has no advertised archive digest.
func verifyTargetArchive(r io.Reader, expected string) error {
	if expected == "" {
		_, err := Verify(r)
		return err
	}
	hash := sha256.New()
	if _, err := Verify(io.TeeReader(r, hash)); err != nil {
		return err
	}
	actual := fmt.Sprintf("%x", hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("target archive SHA256 mismatch: expected %s, actual %s", expected, actual)
	}
	return nil
}

func writeJSONMember(w *tar.Writer, name string, data []byte) error {
	if err := writeHeader(w, name, int64(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// Hash the bytes actually written as a second check on snapshot consumption.
func copyVerifiedMember(w *tar.Writer, name string, member Member, input io.Reader) error {
	if err := writeHeader(w, name, member.Size); err != nil {
		return err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(w, hash), io.LimitReader(input, member.Size+1))
	if err != nil {
		return fmt.Errorf("copy member %q: %w", member.Path, err)
	}
	actual := fmt.Sprintf("%x", hash.Sum(nil))
	if size != member.Size || actual != member.SHA256 {
		return fmt.Errorf("member %q changed while copying: expected size %d SHA256 %s, actual size %d SHA256 %s",
			member.Path, member.Size, member.SHA256, size, actual)
	}
	return nil
}
