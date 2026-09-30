package content

// IsPublished reports whether a content file belongs to the published subset.
// The zero value preserves the historical behavior: pages are published unless
// their frontmatter explicitly sets publish: false.
func IsPublished(meta *FileMeta) bool {
	return meta != nil && (meta.Publish == nil || *meta.Publish)
}

// PublishedFiles returns the pages enabled for publishing while leaving the
// caller's source metadata map untouched.
func PublishedFiles(fileMap map[string]*FileMeta) map[string]*FileMeta {
	published := make(map[string]*FileMeta, len(fileMap))
	for path, meta := range fileMap {
		if IsPublished(meta) {
			published[path] = meta
		}
	}
	return published
}
