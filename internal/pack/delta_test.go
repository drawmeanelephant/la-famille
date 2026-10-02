package pack

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDeltaMetadataStrictJSON(t *testing.T) {
	d := DeltaMetadata{
		SchemaVersion: 1, BaseSHA256: strings.Repeat("0", 64), TargetManifestSHA256: strings.Repeat("1", 64),
		Added: []string{}, Changed: []string{}, Removed: []string{},
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	valid := string(data)
	tests := []struct{ name, data, want string }{
		{"invalid JSON", "{", "malformed delta"},
		{"non UTF-8", "{\"\xff\":1}", "not UTF-8"},
		{"duplicate", strings.Replace(valid, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1), "duplicate"},
		{"case alias", strings.Replace(valid, `"added":[]`, `"added":[],"Added":[]`, 1), "unknown field"},
		{"unknown", strings.Replace(valid, `"added":[]`, `"added":[],"extra":1`, 1), "unknown field"},
		{"missing", strings.Replace(valid, `"added":[],`, "", 1), "missing field"},
		{"null", strings.Replace(valid, `"added":[]`, `"added":null`, 1), "null"},
		{"wrong type", strings.Replace(valid, `"added":[]`, `"added":{}`, 1), "malformed delta"},
		{"bad hash", strings.Replace(valid, d.BaseSHA256, "short", 1), "malformed delta SHA256"},
		{"reserved", strings.Replace(valid, `"added":[]`, `"added":["pack-manifest.json"]`, 1), "reserved"},
		{"trailing", valid + "{}", "trailing JSON"},
		{"nesting", strings.Repeat("[", 34) + "1" + strings.Repeat("]", 34), "nesting"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseDelta([]byte(tt.data)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parse error = %v, want %q", err, tt.want)
			}
		})
	}
	d.Added = make([]string, MaxMembers+1)
	if err := d.validate(); err == nil {
		t.Fatal("unbounded path array accepted")
	}
	d.Added = []string{}
	d.Changed = nil
	if err := d.validate(); err == nil {
		t.Fatal("nil array accepted")
	}
}

func TestLedgerReportingBoundsDoNotRejectOpaquePayload(t *testing.T) {
	for _, payload := range [][]byte{
		bytes.Repeat([]byte(" "), MaxManifestSize+1),
		[]byte(`{"version":2,"pages":[` + strings.Repeat("{},", MaxMembers) + `{}]}`),
	} {
		dir := t.TempDir()
		base, target, delta := dir+"/base", dir+"/target", dir+"/delta"
		writeTestPack(t, base, testEntry{"site-manifest.json", payload})
		writeTestPack(t, target, testEntry{"site-manifest.json", payload})
		report, err := Diff(base, target, delta)
		if err != nil || report.Ledger != nil || !strings.Contains(report.LedgerWarning, "report bound") {
			t.Fatalf("bounded Ledger report = %+v, %v", report, err)
		}
		if _, err := Apply(base, delta, dir+"/result"); err != nil {
			t.Fatal(err)
		}
	}
}
