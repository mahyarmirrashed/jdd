package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/farmergreg/rfsnotify"
	"github.com/gen2brain/beeep"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/jd"
	"github.com/mahyarmirrashed/jdd/internal/utils"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
	"gopkg.in/fsnotify.v1"
)

// Set at build time: go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	beeep.AppName = "Johnny Decimal Daemon"

	configFile := altsrc.StringSourcer(".jd.yaml")

	app := &cli.Command{
		Name:                  "jdd",
		Usage:                 "Johnny Decimal Daemon",
		Version:               version,
		EnableShellCompletion: true,
		Suggest:               true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "root",
				Usage:   "root directory to watch",
				Value:   ".",
				Sources: cli.NewValueSourceChain(yaml.YAML("root", configFile), cli.EnvVar("JDD_ROOT")),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "logging level: debug, info, warn, error",
				Value:   "info",
				Sources: cli.NewValueSourceChain(yaml.YAML("log_level", configFile), cli.EnvVar("JDD_LOG_LEVEL")),
			},
			&cli.BoolFlag{
				Name:    "daemonize",
				Usage:   "run as daemon",
				Value:   false,
				Sources: cli.NewValueSourceChain(yaml.YAML("daemonize", configFile), cli.EnvVar("JDD_DAEMONIZE")),
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "dry run mode",
				Value:   false,
				Sources: cli.NewValueSourceChain(yaml.YAML("dry_run", configFile), cli.EnvVar("JDD_DRY_RUN")),
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Usage:   "glob patterns to exclude (repeat or comma-separated)",
				Value:   []string{},
				Sources: cli.NewValueSourceChain(yaml.YAML("exclude", configFile), cli.EnvVar("JDD_EXCLUDE")),
			},
			&cli.DurationFlag{
				Name:    "delay",
				Usage:   "processing delay on files",
				Value:   0,
				Sources: cli.NewValueSourceChain(yaml.YAML("delay", configFile), cli.EnvVar("JDD_DELAY")),
			},
			&cli.BoolFlag{
				Name:    "notifications",
				Usage:   "send desktop notifications",
				Value:   false,
				Sources: cli.NewValueSourceChain(yaml.YAML("notifications", configFile), cli.EnvVar("JDD_NOTIFICATIONS")),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := &config.Config{
				Root:          cmd.String("root"),
				LogLevel:      strings.ToLower(cmd.String("log-level")),
				Daemonize:     cmd.Bool("daemonize"),
				DryRun:        cmd.Bool("dry-run"),
				Delay:         cmd.Duration("delay"),
				Notifications: cmd.Bool("notifications"),
			}

			excludes := cmd.StringSlice("exclude")
			var mergedExclude []string
			for _, e := range excludes {
				mergedExclude = append(mergedExclude, strings.Split(e, ",")...)
			}
			cfg.Exclude = mergedExclude

			// Set log level from config
			switch cfg.LogLevel {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info":
				log.SetLevel(log.InfoLevel)
			case "warn":
				log.SetLevel(log.WarnLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			// Display configuration
			log.Debugf("Config set as: %+v", cfg)

			// Only daemonize if config says so
			if cfg.Daemonize {
				daemonCtx := &daemon.Context{
					PidFileName: "jdd.pid",
					PidFilePerm: 0644,
					LogFileName: "jdd.log",
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"[jdd-daemon]"},
				}

				d, err := daemonCtx.Reborn()
				if err != nil {
					log.Fatalf("Unable to run: %s", err)
				}
				if d != nil {
					return nil // Parent process exits
				}
				defer daemonCtx.Release()
				log.Info("Daemon started")
			} else {
				log.Info("Running in foreground (not daemonized)")
			}

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

			select {}
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
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
