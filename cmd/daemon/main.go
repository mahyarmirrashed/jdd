package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/mahyarmirrashed/jdd/pkg/jd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: daemon <directory>")
		os.Exit(1)
	}
	dir := os.Args[1]

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

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
						fmt.Println(johnnyDecimalFile.String())

						destinationDir, err := johnnyDecimalFile.EnsureFolders(dir)
						if err != nil {
							log.Println("Error creating folders:", err)
							return
						}

						oldPath := filepath.Join(dir, filename)
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
