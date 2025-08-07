package utils

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/beeep"
	log "github.com/sirupsen/logrus"
)

var icon []byte
//go:embed icon.png

// ExpandTilde will resolve to the correct location on disk.
func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

func SendNotification(enabled bool, title string, message string) {
	if enabled {
		if err := beeep.Notify(title, message, icon); err != nil {
			log.Warnf("Notification failed: %v", err)
		}
	}
}
