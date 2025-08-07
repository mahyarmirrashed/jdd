package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/daemon"
	"github.com/mahyarmirrashed/jdd/internal/utils"
)

var iconResource = fyne.NewStaticResource("icon.png", utils.Icon)

func main() {
	a := app.New()
	w := a.NewWindow("Johnny Decimal Daemon")
	w.SetIcon(iconResource)
	w.Resize(fyne.NewSize(400, 400))

	rootEntry := widget.NewEntry()
	excludePatterns := widget.NewMultiLineEntry()
	delayEntry := widget.NewEntry()
	notificationsCheck := widget.NewCheck("Enable Notifications", nil)
	toggleBtn := widget.NewButton("Start Daemon", nil)

	homeDir, _ := os.UserHomeDir()
	cfgDir := homeDir
	cfgPath := filepath.Join(cfgDir, config.DefaultConfigFilename)

	// Load config or defaults
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		cfg = &config.Config{
			Root:          ".",
			LogLevel:      "info",
			Exclude:       []string{},
			DryRun:        false,
			Daemonize:     false,
			Delay:         0,
			Notifications: false,
		}
	}

	// Populate fields from config
	rootEntry.SetText(cfg.Root)
	excludePatterns.SetText(strings.Join(cfg.Exclude, "\n"))
	delayEntry.SetText(cfg.Delay.String())
	notificationsCheck.SetChecked(cfg.Notifications)

	var (
		daemonCtx     context.Context
		daemonCancel  context.CancelFunc
		daemonMu      sync.Mutex
		daemonRunning bool
	)

	startDaemon := func() error {
		daemonMu.Lock()
		defer daemonMu.Unlock()

		if daemonRunning {
			return nil // already running
		}

		daemonCtx, daemonCancel = context.WithCancel(context.Background())

		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Johnny Decimal Daemon",
			Content: "Daemon started.",
		})

		go func() {
			err := daemon.RunDaemon(daemonCtx, cfg)

			if err != nil && err != context.Canceled {
				log.Errorf("Daemon error: %v", err)

				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Johnny Decimal Daemon",
					Content: fmt.Sprintf("Daemon error: %v", err),
				})
			}

			daemonMu.Lock()
			daemonRunning = false
			daemonMu.Unlock()

			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Johnny Decimal Daemon",
				Content: "Daemon stopped.",
			})

			fyne.Do(func() {
				updateToggleButton(toggleBtn, false)
			})
		}()

		daemonRunning = true

		return nil
	}

	stopDaemon := func() error {
		daemonMu.Lock()
		defer daemonMu.Unlock()

		if !daemonRunning {
			return nil
		}
		if daemonCancel != nil {
			daemonCancel()
		}
		daemonRunning = false
		return nil
	}

	toggleDaemon := func() error {
		daemonMu.Lock()
		running := daemonRunning
		daemonMu.Unlock()

		if running {
			return stopDaemon()
		}
		return startDaemon()
	}

	toggleBtn.OnTapped = func() {
		err := toggleDaemon()
		if err != nil {
			dialog.ShowError(fmt.Errorf("daemon control failed: %v", err), w)
			return
		}

		daemonMu.Lock()
		running := daemonRunning
		daemonMu.Unlock()
		updateToggleButton(toggleBtn, running)
	}

	saveBtn := widget.NewButton("Save Configuration", func() {
		parsedDelay, err := time.ParseDuration(strings.TrimSpace(delayEntry.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid delay duration: %v", err), w)
			return
		}

		newCfg := &config.Config{
			Root:          strings.TrimSpace(rootEntry.Text),
			LogLevel:      "info",
			Exclude:       parseExcludePatterns(excludePatterns.Text),
			DryRun:        false,
			Daemonize:     false,
			Delay:         parsedDelay,
			Notifications: notificationsCheck.Checked,
		}

		err = saveConfig(cfgPath, newCfg)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config: %v", err), w)
			return
		}

		cfg = newCfg

		daemonMu.Lock()
		running := daemonRunning
		daemonMu.Unlock()
		if running {
			if err := stopDaemon(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to stop daemon for restart: %v", err), w)
				return
			}
			if err := startDaemon(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to restart daemon: %v", err), w)
				return
			}
		}

		updateToggleButton(toggleBtn, running)
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Root Directory to Watch", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rootEntry,

		widget.NewLabelWithStyle("Exclude Patterns (one per line)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		excludePatterns,

		widget.NewLabelWithStyle("Processing Delay (e.g., 3s, 500ms)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		delayEntry,

		notificationsCheck,

		container.NewGridWithColumns(2,
			saveBtn,
			toggleBtn,
		),
	)

	scroll := container.NewScroll(form)
	content := container.NewPadded(scroll)
	content = container.NewPadded(content)

	w.SetContent(content)

	// Ensure daemon is stopped on app exit
	a.Lifecycle().SetOnStopped(func() {
		_ = stopDaemon()
	})

	updateToggleButton(toggleBtn, false)

	w.ShowAndRun()
}

func updateToggleButton(btn *widget.Button, running bool) {
	if running {
		btn.SetText("Stop Daemon")
	} else {
		btn.SetText("Start Daemon")
	}
	btn.Refresh()
}
