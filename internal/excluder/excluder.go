package excluder

import (
	"github.com/gobwas/glob"
)

type Excluder struct {
	globs []glob.Glob
}

func NewExcluder(patterns []string) (*Excluder, error) {
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

func (e *Excluder) IsExcluded(path string) bool {
	for _, g := range e.globs {
		if g.Match(path) {
			return true
		}
	}
	return false
}
