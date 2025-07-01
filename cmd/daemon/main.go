package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/farmergreg/rfsnotify"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/jd"
	"gopkg.in/fsnotify.v1"
)

func main() {
	configPath := ".jd.yaml"

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dir := cfg.Root

	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.AddRecursive(dir)
	if err != nil {
		log.Fatal(err)
	}

	ex, err := excluder.NewExcluder(cfg.Exclude)
	if err != nil {
		log.Fatalf("Failed to compile exclude patterns: %v", err)
	}

	// Initial scan
	log.Println("Starting initial scan...")
	if err := initialScan(dir, cfg, ex); err != nil {
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

					if ex.IsExcluded(event.Name) {
						continue
					}

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

						if !cfg.DryRun {
							err = os.Rename(oldPath, newPath)
							if err != nil {
								log.Println("Error moving file:", err)
							}
						} else {
							log.Printf("[dry run] Would move %s -> %s", oldPath, newPath)
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

func initialScan(root string, cfg *config.Config, ex *excluder.Excluder) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || ex.IsExcluded(d.Name()) {
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

			if oldPath != newPath {
				if !cfg.DryRun {
					err = os.Rename(oldPath, newPath)
					if err != nil {
						log.Printf("Error moving %s: %v", filename, err)
					} else {
						log.Printf("Moved %s -> %s", oldPath, newPath)
					}
				} else {
					log.Printf("[dry run] Would move %s -> %s", oldPath, newPath)
				}
			}
		}
		return nil
	})
}
