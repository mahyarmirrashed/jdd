package utils

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/jd"
	log "github.com/sirupsen/logrus"
)

//go:embed logo.png
var icon []byte

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
					out := fmt.Sprintf("Error moving %s: %v", filename, err)
					// Log and send notification
					log.Error(out)
					sendNotification(cfg.Notifications, "JDD", out)
				} else {
					out := fmt.Sprintf("Moved %s -> %s", prettyPath(oldPath), prettyPath(newPath))
					// Log and send notification
					log.Info(out)
					sendNotification(cfg.Notifications, "JDD", out)
				}
			} else {
				out := fmt.Sprintf("[dry run] Would move %s -> %s", prettyPath(oldPath), prettyPath(newPath))
				// Log and send notification
				log.Info(out)
				sendNotification(cfg.Notifications, "JDD", out)
			}
		}
		return true
	}

	return false
}

func sendNotification(enabled bool, title string, message string) {
	if enabled {
		if err := beeep.Notify(title, message, icon); err != nil {
			log.Warnf("Notification failed: %v", err)
		}
	}
}
