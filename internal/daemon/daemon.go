package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/farmergreg/rfsnotify"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/jd"
	"github.com/mahyarmirrashed/jdd/internal/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

// RunDaemon runs the main daemon process; it blocks until stopped or context cancellation.
func RunDaemon(ctx context.Context, cfg *config.Config) error {
	dir := utils.ExpandTilde(cfg.Root)

	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.AddRecursive(dir)
	if err != nil {
		log.Fatal(err)
	}

	ex, err := excluder.New(cfg.Exclude, cfg.Root)
	if err != nil {
		log.Fatalf("Failed to compile exclude patterns: %v", err)
	}

	// Signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Infof("Received signal: %s, shutting down...", sig)

		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Warnf("Error closing watcher: %v", err)
			}
		}
		if cfg.Daemonize {
			if err := os.Remove("jdd.pid"); err != nil && !os.IsNotExist(err) {
				log.Warnf("Error removing PID file: %v", err)
			}
		}

		log.Info("Cleanup complete. Exiting.")
		os.Exit(0)
	}()

	// Initial scan
	log.Info("Starting initial scan...")
	if err := initialScan(dir, cfg, ex); err != nil {
		log.Fatalf("Initial scan failed: %v", err)
	}
	log.Info("Initial scan complete.")

	// Main event handler loop
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Create {
					// Delay addresses an issue with Windows File Explorer
					if cfg.Delay > 0 {
						time.Sleep(cfg.Delay)
					}

					processFile(event.Name, dir, cfg, ex)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error("error:", err)
			}
		}
	}()

	// Wait for signal or context cancellation
	select {
	case sig := <-signals:
		log.Infof("Received signal: %s, shutting down...", sig)
	case <-ctx.Done():
		return ctx.Err()
	}

	log.Info("Daemon stopping")
	return nil
}

// processFile checks if the filename matches the Johnny Decimal pattern,
// ensures the correct folder structure, and moves the file if needed.
// Returns true if the file was processed.
func processFile(fullPath string, root string, cfg *config.Config, ex *excluder.Excluder) bool {
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
					utils.SendNotification(cfg.Notifications, "JDD", out)
				} else {
					out := fmt.Sprintf("Moved %s -> %s", prettyPath(oldPath), prettyPath(newPath))
					// Log and send notification
					log.Info(out)
					utils.SendNotification(cfg.Notifications, "JDD", out)
				}
			} else {
				out := fmt.Sprintf("[dry run] Would move %s -> %s", prettyPath(oldPath), prettyPath(newPath))
				// Log and send notification
				log.Info(out)
				utils.SendNotification(cfg.Notifications, "JDD", out)
			}
		}
		return true
	}

	return false
}

// initialScan walks the entire directory and ensures Johnny Decimal adherence.
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
