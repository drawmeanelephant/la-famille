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
func Watch(cfg config.Config) error {
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
					log.Printf("Detected change in %s, queuing rebuild...", event.Name)
					if buildTimer != nil {
						buildTimer.Stop()
					}
					buildTimer = time.AfterFunc(500*time.Millisecond, func() {
						log.Println("Rebuilding site...")
						if err := generator.Build(cfg); err != nil {
							log.Printf("Error rebuilding site: %v", err)
						} else {
							log.Println("Rebuild complete.")
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

	// Block forever
	<-make(chan struct{})
	return nil
}
