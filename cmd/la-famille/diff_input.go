package main

import (
	"fmt"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

func loadDiffInput(input string, cfg config.Config) (sitedata.Manifest, error) {
	return sitediff.LoadInputWithBuilder(input, cfg.OutputDir, cfg.ProjectRoot, func(root string) (sitedata.Manifest, error) {
		configRel := "config.yaml"
		if cfg.ConfigPath != "" {
			var err error
			configRel, err = filepath.Rel(cfg.ProjectRoot, cfg.ConfigPath)
			if err != nil || !filepath.IsLocal(configRel) {
				return sitedata.Manifest{}, fmt.Errorf("Git snapshot config must be inside the project root")
			}
		}
		raw, err := config.Load(filepath.Join(root, configRel))
		if err != nil {
			return sitedata.Manifest{}, err
		}
		snapshotCfg, err := raw.ResolvePaths(root)
		if err != nil {
			return sitedata.Manifest{}, err
		}
		// Never read external paths named by archived configuration, or write
		// to its configured output. Everything lives in the disposable archive.
		for _, path := range []string{snapshotCfg.ContentDir, snapshotCfg.AssetDir, snapshotCfg.Template} {
			rel, err := filepath.Rel(root, path)
			if err != nil || !filepath.IsLocal(rel) {
				return sitedata.Manifest{}, fmt.Errorf("Git snapshot path must be inside the project root: %q", path)
			}
		}
		snapshotCfg.OutputDir = filepath.Join(root, ".ledger-output")
		if err := snapshotCfg.ValidateResolved(); err != nil {
			return sitedata.Manifest{}, err
		}
		if _, err := generator.Build(snapshotCfg); err != nil {
			return sitedata.Manifest{}, fmt.Errorf("build Git snapshot: %w", err)
		}
		return sitedata.ReadManifest(filepath.Join(snapshotCfg.OutputDir, sitedata.ManifestFileName))
	})
}
