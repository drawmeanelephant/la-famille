# Task plan: `luna/body-tags` (#582.2)

1. Parse body hashtags from Markdown text nodes, excluding headings, code, and
   autolinks, then normalize and deduplicate them with frontmatter tags.
2. Add parser and metadata tests plus a build fixture proving body-tag and
   frontmatter-tag pages share the same `/tags/` archive.
3. Document body-tag syntax and its taxonomy output alongside frontmatter tags.

**Static-output impact:** Body hashtags become page taxonomy tags. They will
appear in the existing tag archives, page tag links, search index, and
metadata/graph outputs wherever frontmatter tags already appear. Existing
frontmatter behavior remains unchanged.
