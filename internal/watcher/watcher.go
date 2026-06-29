package watcher

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)


// Watch starts an fsnotify watcher on the given config's ContentDir and Templates dir.
// It will rebuild the site via generator.Build(cfg) whenever a file change is detected.
func Watch(cfg config.Config, onBuild func(generator.BuildResult)) error {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}
	defer watcher.Close()

	// Rate-limit rebuilds (debounce)
	var buildTimer *time.Timer

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Only rebuild on creation, modification, or deletion
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
					// If a new directory is created, add it to the watcher.
					if event.Has(fsnotify.Create) {
						stat, err := os.Stat(event.Name)
						if err == nil && stat.IsDir() {
							log.Printf("New directory detected, adding to watcher: %s", event.Name)
							watcher.Add(event.Name)
						}
					}
					log.Printf("Detected change in %s, queuing rebuild...", event.Name)
					if buildTimer != nil {
						buildTimer.Stop()
					}
					buildTimer = time.AfterFunc(500*time.Millisecond, func() {
						log.Println("Rebuilding site...")
						start := time.Now()
						if res, err := generator.Build(cfg); err != nil {
							if onBuild != nil { onBuild(res) }
							log.Printf("Error rebuilding site: %v", err)
						} else {
							log.Printf("Rebuild complete in %v.", time.Since(start))
							if onBuild != nil { onBuild(res) }
						}
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	// Watch content directory
	err = filepath.WalkDir(cfg.ContentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Watch templates directory if it exists
	templateDir := filepath.Dir(cfg.Template)
	if _, err := os.Stat(templateDir); err == nil {
		watcher.Add(templateDir)
	}

	// Watch assets directory if it exists
	assetsDir := "assets"
	if _, err := os.Stat(assetsDir); err == nil {
		filepath.WalkDir(assetsDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})
	}

	// Block forever
	<-make(chan struct{})
	return nil
}
