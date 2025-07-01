package excluder

import (
	"path/filepath"

	"github.com/gobwas/glob"
)

// Excluder matches file paths against a list of glob patterns.
type Excluder struct {
	globs []glob.Glob
	root  string
}

// New creates an Excluder from a list of glob patterns and the root directory.
// Patterns use '/' as the path separator.
func New(patterns []string, root string) (*Excluder, error) {
	var globs []glob.Glob
	for _, pat := range patterns {
		g, err := glob.Compile(pat, '/')
		if err != nil {
			return nil, err
		}
		globs = append(globs, g)
	}

	return &Excluder{globs: globs, root: root}, nil
}

// IsExcluded returns true if the given path matches any exclude pattern.
// The path is made relative to the root before matching.
func (e *Excluder) IsExcluded(path string) bool {
	rel, err := filepath.Rel(e.root, path)
	if err != nil {
		// fallback: just use the original path
		rel = path
	}
	rel = filepath.ToSlash(rel) // Ensure '/' as separator

	for _, g := range e.globs {
		if g.Match(rel) {
			return true
		}
	}
	return false
}
