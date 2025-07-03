package utils

import (
	"os"
	"path/filepath"

	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/jd"
	log "github.com/sirupsen/logrus"
)

// ProcessFile checks if the filename matches the Johnny Decimal pattern,
// ensures the correct folder structure, and moves the file if needed.
// Returns true if the file was processed.
func ProcessFile(fullPath string, root string, cfg *config.Config, ex *excluder.Excluder) bool {
	filename := filepath.Base(fullPath)

	if ex.IsExcluded(fullPath) {
		log.Debugf("Excluded: %s", fullPath)
		return false
	}

	if jd.JohnnyDecimalFilePattern.MatchString(filename) {
		jdObj, err := jd.Parse(filename)
		if err != nil {
			log.Warnf("Johnny Decimal parsing error: %v", err)
			return false
		}

		destDir, err := jdObj.EnsureFolders(root)
		if err != nil {
			log.Warnf("Error creating folders: %v", err)
			return false
		}

		oldPath := fullPath
		newPath := filepath.Join(destDir, filename)

		prettyPath := func(path string) string { return filepath.ToSlash(path) }

		if oldPath != newPath {
			if !cfg.DryRun {
				err = os.Rename(oldPath, newPath)
				if err != nil {
					log.Errorf("Error moving %s: %v", filename, err)
				} else {
					log.Infof("Moved %s -> %s", prettyPath(oldPath), prettyPath(newPath))
				}
			} else {
				log.Infof("[dry run] Would move %s -> %s", prettyPath(oldPath), prettyPath(newPath))
			}
		}
		return true
	}

	return false
}
