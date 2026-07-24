package generator

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The watcher debounce cannot cancel a rebuild that already started, so two
// Build calls can reach the publish step at the same time. They must not fail
// each other or leave the intermediate directories behind.
//
// The lock is process-local: two separate la-famille processes publishing into
// one directory are still unserialized.
func TestBuild_ConcurrentBuildsShareOutputDirectory(t *testing.T) {
	cfg, _ := setupTestSite(t)

	const builders = 6
	var wg sync.WaitGroup
	errs := make([]error, builders)
	for i := 0; i < builders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = Build(cfg)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Build() %d error = %v, want nil", i, err)
		}
	}

	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "page1", "index.html")); err != nil {
		t.Errorf("published site is missing page1: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(cfg.OutputDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, ".staging-") || strings.Contains(name, ".previous-") {
			t.Errorf("build left an intermediate directory behind: %s", name)
		}
	}
}
