package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

// Watch starts an fsnotify watcher on the given config's ContentDir, Templates, and Assets dir.
// It explicitly unbinds and tears down resources once the passed context registers Done.
func Watch(ctx context.Context, cfg config.Config, onBuild func(generator.BuildResult, error)) error {
	return watch(ctx, cfg, onBuild, generator.Build, 500*time.Millisecond)
}

type buildFunc func(config.Config) (generator.BuildResult, error)

// watch contains the event loop with injectable build and debounce behavior so
// lifecycle tests do not need to invoke the full generator or wait half a
// second for every assertion.
func watch(ctx context.Context, cfg config.Config, onBuild func(generator.BuildResult, error), build buildFunc, debounce time.Duration) error {
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

	// Timer.Stop is a no-op once the timer has fired, so debouncing alone
	// cannot cancel a pass that is already underway: every further event
	// scheduled another AfterFunc, and a burst arriving around a build boundary
	// produced several rebuilds where one was wanted.
	//
	// The timer therefore only signals, and a single builder goroutine does the
	// work. The channel holds one slot: signals raised while a build is running
	// collapse into exactly one follow-up pass, which is enough to pick up
	// every change that landed during the build. One goroutine also means two
	// builds can never overlap and race over the output directory swap.
	trigger := make(chan struct{}, 1)
	buildDone := make(chan struct{})

	// The builder is stopped through its own context rather than the caller's.
	// watch() returns on several paths that never cancel ctx — a failed
	// watcher.Add, a closed event channel, a WalkDir error — and waiting for a
	// builder that only exits on ctx.Done() would hang every one of them,
	// holding all registered watch descriptors open.
	buildCtx, stopBuilder := context.WithCancel(ctx)

	runBuild := func() {
		select {
		case <-buildCtx.Done():
			return
		default:
		}
		slog.Info("Executing pipeline rebuild...")
		start := time.Now()
		res, err := build(cfg)
		if err != nil {
			if onBuild != nil {
				onBuild(res, err)
			}
			// Deliberately no BroadcastReload here. The build failed, so the
			// output directory still holds the previous site; telling the
			// browser to reload made it refresh unchanged bytes and look as
			// though the edit had landed. The page now stays as it was and the
			// error is what the author sees.
			slog.Error("Pipeline compilation failed", "error", err)
			return
		}
		slog.Info("Rebuild complete", "duration", time.Since(start))
		if onBuild != nil {
			onBuild(res, nil)
		}
		BroadcastReload()
	}

	go func() {
		defer close(buildDone)
		for {
			select {
			case <-buildCtx.Done():
				return
			case <-trigger:
				runBuild()
			}
		}
	}()
	// Stop the builder first, then let it finish the pass it is on, so a watch
	// that ends for any reason still leaves a complete output directory and
	// still returns.
	defer func() {
		stopBuilder()
		<-buildDone
	}()

	// The build writes its own bookkeeping beside the output directory — the
	// cache file and the .<output>.staging-*/.<output>.previous-* swap trees —
	// and a flat layout (content_dir: ".") puts all of them inside the watched
	// root. Treating those writes as changes made every rebuild schedule the
	// next one forever (#633): a staging directory got watched on create, the
	// fsnotify watch followed the inode through the rename into public/, and
	// the cache write then retriggered the cycle.
	ignore := generator.BuildArtifactFilter(cfg)

	// Orchestrate directories to monitor
	dirsToWatch := []string{cfg.ContentDir}

	templateDir := filepath.Dir(cfg.Template)
	if _, err := os.Stat(templateDir); err == nil {
		dirsToWatch = append(dirsToWatch, templateDir)
	}
	if _, err := os.Stat(cfg.AssetDir); err == nil {
		dirsToWatch = append(dirsToWatch, cfg.AssetDir)
	}

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
			if ignore(path) {
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
				// Build bookkeeping writes — the cache file, the swap's
				// staging/previous trees, the output directory itself — are
				// consequences of a build, not content changes; acting on
				// them schedules rebuilds forever (#633).
				if ignore(event.Name) {
					continue
				}
				if event.Has(fsnotify.Create) {
					stat, err := os.Stat(event.Name)
					if err == nil && stat.IsDir() {
						slog.Info("Dynamic directory tracking added", "dir", event.Name)
						_ = filepath.WalkDir(event.Name, func(path string, d os.DirEntry, err error) error {
							if err != nil {
								return nil
							}
							if ignore(path) {
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
					}
				}

				slog.Info("Change caught, scheduling build pass", "file", event.Name)
				if buildTimer != nil {
					buildTimer.Stop()
				}

				buildTimer = time.AfterFunc(debounce, func() {
					// Non-blocking: if a pass is already queued it will pick up
					// this change too, so there is nothing to add.
					select {
					case trigger <- struct{}{}:
					default:
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
