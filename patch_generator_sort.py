with open("internal/generator/generator.go", "r") as f:
    content = f.read()

import re

# Sort searchIndex, edges, and missingFiles before writing
# To ensure output is deterministic

sort_logic = """	// Sort searchIndex, edges, and other outputs to ensure deterministic output
	sort.Slice(searchIndex, func(i, j int) bool {
		if searchIndex[i].Title == searchIndex[j].Title {
			return searchIndex[i].URL < searchIndex[j].URL
		}
		return searchIndex[i].Title < searchIndex[j].Title
	})

	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i][0] == g.Edges[j][0] {
			return g.Edges[i][1] < g.Edges[j][1]
		}
		return g.Edges[i][0] < g.Edges[j][0]
	})

	for k := range backlinks {
		sort.Strings(backlinks[k])
	}
"""

start_idx = content.find("\t// 3. Generate stubs for missing files in deterministic order")
if start_idx == -1:
    print("Could not find line")
    import sys
    sys.exit(1)

new_content = content[:start_idx] + sort_logic + "\n" + content[start_idx:]

with open("internal/generator/generator.go", "w") as f:
    f.write(new_content)
print("Updated generator.go to sort outputs")
