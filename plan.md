# Plan: Generate template gallery at `content/showcase/`

1. Generate template gallery using bash script via `run_in_bash_session`:
   ```bash
   mkdir -p content/showcase
   rm -f content/showcase/*.md

   cat << 'INNER_EOF' > content/showcase/index.md
   ---
   title: "Template Gallery"
   description: "A visual index of every layout available."
   ---

   # Template Gallery

   Welcome to the template gallery. Here is a visual index of every layout available:

   <div class="not-prose grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
   INNER_EOF

   for tpl in templates/*.html; do
     base=$(basename "$tpl" .html)

     title=$(echo "$base" | sed -e 's/-/ /g' -e 's/_/ /g' | awk '{for(i=1;i<=NF;i++)sub(/./,toupper(substr($i,1,1)),$i)}1')

     cat << INNER_EOF2 > "content/showcase/${base}.md"
   ---
   title: "${title} Demo"
   layout: "${base}"
   ---

   ## ${title} Demo

   This page demonstrates the \`${base}\` layout with realistic content.

   Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

   Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

   ### Features
   - Responsive design
   - Clean typography
   - Elegant layout

   > "Design is not just what it looks like and feels like. Design is how it works."
   INNER_EOF2

     cat << INNER_EOF3 >> content/showcase/index.md
     <div class="card bg-base-100 shadow-xl border border-base-300">
       <div class="card-body">
         <h2 class="card-title text-xl font-bold m-0">${title}</h2>
         <p class="text-base-content/80 m-0">Demo page using the <code>${base}</code> layout.</p>
         <div class="card-actions justify-end mt-4">
           <a href="/showcase/${base}/" class="btn btn-primary btn-sm">View Demo</a>
         </div>
       </div>
     </div>
   INNER_EOF3

   done

   echo "</div>" >> content/showcase/index.md
   ```
2. Check that the script worked correctly by verifying generated files via `run_in_bash_session`:
   `ls -la content/showcase/` and `cat content/showcase/index.md`.
3. Run tests using `run_in_bash_session`:
   `go test ./...` and `go vet ./...` to ensure there are no regressions.
4. Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.
5. Submit the changes using the `submit` tool.
1. Write out this plan.md.
2. Update content/docs/generator.md to include the multi-pass pipeline, internal packages, and JSON outputs.
3. Build the static site using `go run ./cmd/la-famille build`.
4. Create a report of the work in content/jules/reports/generator-stub.md.
5. Run tests to ensure regressions have not been introduced.
