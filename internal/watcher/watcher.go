package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

// Watch starts an fsnotify watcher on the given config's ContentDir, Templates, and Assets dir.
// It explicitly unbinds and tears down resources once the passed context registers Done.
func Watch(ctx context.Context, cfg config.Config, onBuild func(generator.BuildResult)) error {
	return watch(ctx, cfg, onBuild, generator.Build, 500*time.Millisecond)
}

type buildFunc func(config.Config) (generator.BuildResult, error)

// watch contains the event loop with injectable build and debounce behavior so
// lifecycle tests do not need to invoke the full generator or wait half a
// second for every assertion.
func watch(ctx context.Context, cfg config.Config, onBuild func(generator.BuildResult), build buildFunc, debounce time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Debounce timer management
	var buildTimer *time.Timer
	defer func() {
		if buildTimer != nil {
			buildTimer.Stop()
		}
	}()

	// Orchestrate directories to monitor
	dirsToWatch := []string{cfg.ContentDir}

	templateDir := filepath.Dir(cfg.Template)
	if _, err := os.Stat(templateDir); err == nil {
		dirsToWatch = append(dirsToWatch, templateDir)
	}
	if _, err := os.Stat(cfg.AssetDir); err == nil {
		dirsToWatch = append(dirsToWatch, cfg.AssetDir)
	}

	outDirClean := filepath.Clean(cfg.OutputDir)
	for _, dir := range dirsToWatch {
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			cleanPath := filepath.Clean(path)
			if cleanPath == outDirClean || strings.HasPrefix(cleanPath, outDirClean+string(filepath.Separator)) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	slog.Info("Context-aware file watcher initialized successfully.")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Halting file watcher: Context canceled.")
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				if event.Has(fsnotify.Create) {
					stat, err := os.Stat(event.Name)
					if err == nil && stat.IsDir() {
						cleanName := filepath.Clean(event.Name)
						if !(cleanName == outDirClean || strings.HasPrefix(cleanName, outDirClean+string(filepath.Separator))) {
							slog.Info("Dynamic directory tracking added", "dir", event.Name)
							_ = watcher.Add(event.Name)
						}
					}
				}

				slog.Info("Change caught, scheduling build pass", "file", event.Name)
				if buildTimer != nil {
					buildTimer.Stop()
				}

				buildTimer = time.AfterFunc(debounce, func() {
					select {
					case <-ctx.Done():
						return
					default:
					}
					slog.Info("Executing pipeline rebuild...")
					start := time.Now()
					if res, err := build(cfg); err != nil {
						if onBuild != nil {
							onBuild(res)
						}
						BroadcastReload()
						slog.Error("Pipeline compilation failed", "error", err)
					} else {
						slog.Info("Rebuild complete", "duration", time.Since(start))
						if onBuild != nil {
							onBuild(res)
						}
						BroadcastReload()
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Error("Watcher filesystem interruption error", "error", err)
		}
	}
}
