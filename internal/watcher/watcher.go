package watcher

import (
	"context"
	"log"
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
	if _, err := os.Stat("assets"); err == nil {
		dirsToWatch = append(dirsToWatch, "assets")
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

	log.Println("Context-aware file watcher initialized successfully.")

	for {
		select {
		case <-ctx.Done():
			log.Println("Halting file watcher: Context canceled.")
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
							log.Printf("Dynamic directory tracking added: %s", event.Name)
							_ = watcher.Add(event.Name)
						}
					}
				}

				log.Printf("Change caught in %s, scheduling build pass...", event.Name)
				if buildTimer != nil {
					buildTimer.Stop()
				}

				buildTimer = time.AfterFunc(500*time.Millisecond, func() {
					log.Println("Executing pipeline rebuild...")
					start := time.Now()
					if res, err := generator.Build(cfg); err != nil {
						if onBuild != nil {
							onBuild(res)
						}
						BroadcastReload()
						log.Printf("Pipeline compilation failed: %v", err)
					} else {
						log.Printf("Rebuild complete in %v.", time.Since(start))
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
			log.Printf("Watcher filesystem interruption error: %v", err)
		}
	}
}
