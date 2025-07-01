package excluder

import (
	"github.com/gobwas/glob"
)

// Excluder matches file paths against a list of glob patterns.
type Excluder struct {
	globs []glob.Glob
}

// New creates an Excluder from a list of glob patterns.
// Patterns use '/' as the path separator.
func New(patterns []string) (*Excluder, error) {
	var globs []glob.Glob
	for _, pat := range patterns {
		g, err := glob.Compile(pat, '/')
		if err != nil {
			return nil, err
		}
		globs = append(globs, g)
	}

	return &Excluder{globs: globs}, nil
}

// IsExcluded returns true if the given path matches any exclude pattern.
func (e *Excluder) IsExcluded(path string) bool {
	for _, g := range e.globs {
		if g.Match(path) {
			return true
		}
	}

	return false
}
