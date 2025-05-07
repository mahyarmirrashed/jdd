package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add all subdirectories to the watcher (recursive)
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				log.Printf("WARNING: Failed to watch %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`^(\d{2})\.(\d{2})\s+(.+)$`)

	log.Println("Watching for Johnny Decimal files...")

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Rename == fsnotify.Rename {
				filename := filepath.Base(event.Name)

				if matches := re.FindStringSubmatch(filename); matches != nil {
					area, project, _ := matches[1], matches[2], matches[3]
					areaDir := fmt.Sprintf("%s-area", area)
					projectDir := fmt.Sprintf("%s/%s.%s-project", areaDir, area, project)
					newPath := filepath.Join(projectDir, filename)

					if err := os.MkdirAll(projectDir, 0755); err != nil {
						log.Printf("ERROR creating directory: %v", err)
						continue
					}

					if err := os.Rename(event.Name, newPath); err != nil {
						log.Printf("ERROR moving file: %v", err)
					} else {
						log.Printf("MOVED: %s -> %s", filename, newPath)
					}
				}
			}

		case err := <-watcher.Errors:
			log.Println("ERROR:", err)
		}
	}
}
