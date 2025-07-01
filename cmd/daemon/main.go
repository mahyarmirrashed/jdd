package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/farmergreg/rfsnotify"
	"github.com/mahyarmirrashed/jdd/pkg/jd"
	"gopkg.in/fsnotify.v1"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: daemon <directory>")
		os.Exit(1)
	}
	dir := os.Args[1]

	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.AddRecursive(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Initial scan
	log.Println("Starting initial scan...")
	if err := initialScan(dir); err != nil {
		log.Fatalf("Initial scan failed: %v", err)
	}
	log.Println("Initial scan complete.")

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op == fsnotify.Create {
					filename := filepath.Base(event.Name)

					if jd.JohnnyDecimalFilePattern.MatchString(filename) {
						johnnyDecimalFile, err := jd.Parse(filename)
						if err != nil {
							log.Println("Johnny Decimal parsing error:", err)
							continue
						}

						destinationDir, err := johnnyDecimalFile.EnsureFolders(dir)
						if err != nil {
							log.Println("Error creating folders:", err)
							return
						}

						oldPath := event.Name
						newPath := filepath.Join(destinationDir, filename)

						err = os.Rename(oldPath, newPath)
						if err != nil {
							log.Println("Error moving file:", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

func initialScan(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		filename := d.Name()

		if jd.JohnnyDecimalFilePattern.MatchString(filename) {
			jdObj, err := jd.Parse(filename)
			if err != nil {
				log.Printf("JD parse error for %s: %v", filename, err)
				return nil
			}

			destDir, err := jdObj.EnsureFolders(root)
			if err != nil {
				log.Printf("Error creating folders for %s: %v", filename, err)
				return nil
			}

			oldPath := path
			newPath := filepath.Join(destDir, filename)

			// Only move if not already in the correct place
			if oldPath != newPath {
				err = os.Rename(oldPath, newPath)
				if err != nil {
					log.Printf("Error moving %s: %v", filename, err)
				} else {
					log.Printf("Moved %s -> %s", oldPath, newPath)
				}
			}
		}
		return nil
	})
}
