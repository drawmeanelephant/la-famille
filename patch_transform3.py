with open("internal/transform/link_transformer.go", "r") as f:
    content = f.read()

import re

# Add sync to imports if needed
if '"sync"' not in content:
    content = re.sub(r'("strings"\n)', r'\1\t"sync"\n', content)

# Add Mu *sync.Mutex
if 'Mu           *sync.Mutex' not in content:
    content = re.sub(r'(Graph\s+\*graph\.Graph)\n', r'\1\n\tMu           *sync.Mutex\n', content)

# Replace Edges and Backlinks logic
old_block1 = """			targetId := strings.TrimSuffix(targetRelPath, ".md")
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceId, targetId})
			t.Backlinks[targetId] = append(t.Backlinks[targetId], sourceId)

			// Check file map"""
new_block1 = """			targetId := strings.TrimSuffix(targetRelPath, ".md")
			if t.Mu != nil {
				t.Mu.Lock()
			}
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceId, targetId})
			t.Backlinks[targetId] = append(t.Backlinks[targetId], sourceId)
			if t.Mu != nil {
				t.Mu.Unlock()
			}

			// Check file map"""
content = content.replace(old_block1, new_block1)

# Replace MissingFiles logic
old_block2 = """			if !exists {
				// record target as missing so we can generate stub
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
			}"""
new_block2 = """			if !exists {
				// record target as missing so we can generate stub
				if t.Mu != nil {
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
				}
			}"""
content = content.replace(old_block2, new_block2)

with open("internal/transform/link_transformer.go", "w") as f:
    f.write(content)
