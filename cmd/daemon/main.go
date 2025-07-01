package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/farmergreg/rfsnotify"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/sevlyar/go-daemon"
	"gopkg.in/fsnotify.v1"
)

func main() {
	// Daemon context setup
	ctx := &daemon.Context{
		PidFileName: "jdd.pid",
		PidFilePerm: 0644,
		LogFileName: "jdd.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[jdd-daemon]"},
	}

	d, err := ctx.Reborn()
	if err != nil {
		log.Fatalf("Unable to run: %s", err)
	}
	if d != nil {
		return // Parent process exits
	}
	defer ctx.Release()

	log.Println("Daemon started")

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

	ex, err := excluder.New(cfg.Exclude)
	if err != nil {
		log.Fatalf("Failed to compile exclude patterns: %v", err)
	}

	// Signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Printf("Received signal: %s, shutting down...", sig)

		// Close watcher
		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Printf("Error closing watcher: %v", err)
			}
		}
		// Remove PID file
		if err := os.Remove("jdd.pid"); err != nil && !os.IsNotExist(err) {
			log.Printf("Error removing PID file: %v", err)
		}

		log.Println("Cleanup complete. Exiting.")
		os.Exit(0)
	}()

	// Initial scan
	log.Println("Starting initial scan...")
	if err := initialScan(dir, cfg, ex); err != nil {
		log.Fatalf("Initial scan failed: %v", err)
	}
	log.Println("Initial scan complete.")

	// Main event handler loop
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op == fsnotify.Create {
					processFile(event.Name, dir, cfg, ex)
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

		if d.IsDir() {
			return nil
		}

		processFile(path, root, cfg, ex)
		return nil
	})
}
