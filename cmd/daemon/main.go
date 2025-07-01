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
							log.Println("jd parsing error:", err)
							continue
						}

						destinationDir, err := johnnyDecimalFile.EnsureFolders(dir)
						if err != nil {
							log.Println("error creating folders:", err)
							return
						}

						oldPath := event.Name
						newPath := filepath.Join(destinationDir, filename)

						err = os.Rename(oldPath, newPath)
						if err != nil {
							log.Println("error moving file:", err)
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
