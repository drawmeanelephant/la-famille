import re

with open("internal/transform/link_transformer.go", "r") as f:
    content = f.read()

# Make sure we only have one set of replacements
# Restore from git just to be sure we are clean
import subprocess
subprocess.run(["git", "checkout", "internal/transform/link_transformer.go"])

with open("internal/transform/link_transformer.go", "r") as f:
    content = f.read()

if '"sync"' not in content:
    content = content.replace('"strings"\n', '"strings"\n\t"sync"\n')

if "Mu           *sync.Mutex" not in content:
    content = content.replace("Graph        *graph.Graph\n}", "Graph        *graph.Graph\n\tMu           *sync.Mutex\n}")

# Fix Graph and Backlinks access (only exact string match)
old_access = """			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceId, targetId})
			t.Backlinks[targetId] = append(t.Backlinks[targetId], sourceId)"""
new_access = """			if t.Mu != nil {
				t.Mu.Lock()
			}
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceId, targetId})
			t.Backlinks[targetId] = append(t.Backlinks[targetId], sourceId)
			if t.Mu != nil {
				t.Mu.Unlock()
			}"""
content = content.replace(old_access, new_access)

# Fix MissingFiles access (only exact string match)
old_missing = """				parents := t.MissingFiles[targetRelPath]
				found := false
				for _, p := range parents {
					if p == t.CurrentFile {
						found = true
						break
					}
				}
				if !found {
					t.MissingFiles[targetRelPath] = append(parents, t.CurrentFile)
				}"""
new_missing = """				if t.Mu != nil {
					t.Mu.Lock()
				}
				parents := t.MissingFiles[targetRelPath]
				found := false
				for _, p := range parents {
					if p == t.CurrentFile {
						found = true
						break
					}
				}
				if !found {
					t.MissingFiles[targetRelPath] = append(parents, t.CurrentFile)
				}
				if t.Mu != nil {
					t.Mu.Unlock()
				}"""
content = content.replace(old_missing, new_missing)

with open("internal/transform/link_transformer.go", "w") as f:
    f.write(content)
print("Updated link_transformer.go")
