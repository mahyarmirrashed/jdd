package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/farmergreg/rfsnotify"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/excluder"
	"github.com/mahyarmirrashed/jdd/internal/utils"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"gopkg.in/fsnotify.v1"
)

// Set at build time: go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	app := &cli.Command{
		Name:    "jdd",
		Usage:   "Johnny Decimal Daemon",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to config file",
				Sources: cli.EnvVars("JDD_CONFIG"),
				Value:   ".jd.yaml",
			},
			&cli.StringFlag{
				Name:    "root",
				Usage:   "root directory to watch",
				Sources: cli.EnvVars("JDD_ROOT"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "logging level: debug, info, warn, error",
				Sources: cli.EnvVars("JDD_LOG_LEVEL"),
			},
			&cli.BoolFlag{
				Name:    "daemonize",
				Usage:   "run as daemon",
				Sources: cli.EnvVars("JDD_DAEMONIZE"),
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "dry run mode",
				Sources: cli.EnvVars("JDD_DRY_RUN"),
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Usage:   "glob patterns to exclude (repeat or comma-separated)",
				Sources: cli.EnvVars("JDD_EXCLUDE"),
			},
			&cli.DurationFlag{
				Name:    "delay",
				Usage:   "processing delay on files",
				Sources: cli.EnvVars("JDD_DELAY"),
				Value:   0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var cfg *config.Config
			configPath := cmd.String("config")

			// Only load config if the file exists
			if _, err := os.Stat(configPath); err == nil {
				cfg, err = config.LoadConfig(configPath)
				if err != nil {
					log.Fatalf("Failed to load config: %v", err)
				}
			} else {
				// Use defaults if no config file
				cfg = &config.Config{
					Root:     ".",
					LogLevel: "info",
				}
			}

			// Override config with flags if set
			if cmd.IsSet("root") {
				cfg.Root = cmd.String("root")
			}
			if cmd.IsSet("log-level") {
				cfg.LogLevel = cmd.String("log-level")
			}
			if cmd.IsSet("daemonize") {
				cfg.Daemonize = cmd.Bool("daemonize")
			}
			if cmd.IsSet("dry-run") {
				cfg.DryRun = cmd.Bool("dry-run")
			}
			if cmd.IsSet("exclude") {
				exclude := cmd.StringSlice("exclude")
				var merged []string
				for _, e := range exclude {
					merged = append(merged, strings.Split(e, ",")...)
				}
				cfg.Exclude = merged
			}
			if cmd.IsSet("delay") {
				cfg.Delay = cmd.Duration("delay")
			}

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

							utils.ProcessFile(event.Name, dir, cfg, ex)
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

// initialScan walks the entire directory and ensures Johnny Decimal adherence.
func initialScan(root string, cfg *config.Config, ex *excluder.Excluder) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		utils.ProcessFile(path, root, cfg, ex)
		return nil
	})
}
