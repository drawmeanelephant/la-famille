with open("internal/generator/generator.go", "r") as f:
    content = f.read()

import re

# Update sort logic for edges and backlinks to be deterministic based on original loop order.
# Or simpler: just sort edges! Wait, I tried sorting edges before and it failed the first test.
# Why? Because in the original loop, it iterates files in ALPHABETICAL order (since we do sort.Strings(keys)).
# Inside the file, it walks AST and appends edges.
# So edges are appended in order of: file1 links, file2 links...
# So if we just sort edges by the SourceId (edge[0]), but keep the original relative order within the same SourceId (stable sort)?
# Wait, if we sort by SourceId, that IS the alphabetical order of files!
# Let's use sort.SliceStable on g.Edges based on edge[0].
# What about backlinks? We can sort each array in backlinks by string.

sort_logic = """	// Sort searchIndex, edges, and other outputs to ensure deterministic output
	sort.SliceStable(g.Edges, func(i, j int) bool {
		return g.Edges[i][0] < g.Edges[j][0]
	})

	for k := range backlinks {
		sort.Strings(backlinks[k])
	}
"""

start_idx = content.find("\t// Sort errs for deterministic order")
if start_idx == -1:
    print("Could not find line")
    import sys
    sys.exit(1)

new_content = content[:start_idx] + sort_logic + "\n" + content[start_idx:]

with open("internal/generator/generator.go", "w") as f:
    f.write(new_content)
print("Updated generator.go to sort outputs")
