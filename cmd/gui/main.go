package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/mahyarmirrashed/jdd/internal/config"
	"github.com/mahyarmirrashed/jdd/internal/utils"
)

var iconResource = fyne.NewStaticResource("icon.png", utils.Icon)

func init() {
	// Configure logger to include timestamp and caller (file:line)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
		},
	})
	log.SetReportCaller(true)
}

func main() {
	a := app.New()
	w := a.NewWindow("Johnny Decimal Daemon")
	w.SetIcon(iconResource)
	w.Resize(fyne.NewSize(400, 500))

	rootEntry := widget.NewEntry()
	logLevelEntry := widget.NewEntry()
	excludePatterns := widget.NewMultiLineEntry()
	delayEntry := widget.NewEntry()
	dryRunCheck := widget.NewCheck("Dry Run Mode", nil)
	notificationsCheck := widget.NewCheck("Enable Notifications", nil)

	daemonExecutable := "jdd"

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
	logLevelEntry.SetText(cfg.LogLevel)
	excludePatterns.SetText(strings.Join(cfg.Exclude, "\n"))
	dryRunCheck.SetChecked(cfg.DryRun)
	delayEntry.SetText(cfg.Delay.String())
	notificationsCheck.SetChecked(cfg.Notifications)

	// Create daemon controller
	daemonCtrl := NewDaemonController()

	// Create toggle button
	toggleBtn := widget.NewButton("Start Daemon", nil)
	toggleBtn.OnTapped = func() {
		err := daemonCtrl.Toggle(daemonExecutable, cfgDir)
		if err != nil {
			dialog.ShowError(fmt.Errorf("daemon control failed: %v", err), w)
			return
		}
		updateToggleButton(toggleBtn, daemonCtrl)
	}

	daemonCtrl.onExit = func() {
		fyne.Do(func() {
			updateToggleButton(toggleBtn, daemonCtrl)
		})
	}

	saveBtn := widget.NewButton("Save Configuration", func() {
		logLevel := strings.ToLower(strings.TrimSpace(logLevelEntry.Text))
		switch logLevel {
		case "debug", "info", "warn", "error":
		default:
			dialog.ShowError(fmt.Errorf("invalid log level: %s", logLevel), w)
			return
		}

		parsedDelay, err := time.ParseDuration(strings.TrimSpace(delayEntry.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid delay duration: %v", err), w)
			return
		}

		newCfg := &config.Config{
			Root:          strings.TrimSpace(rootEntry.Text),
			LogLevel:      logLevel,
			Exclude:       parseExcludePatterns(excludePatterns.Text),
			DryRun:        dryRunCheck.Checked,
			Daemonize:     false,
			Delay:         parsedDelay,
			Notifications: notificationsCheck.Checked,
		}

		err = saveConfig(cfgPath, newCfg)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config: %v", err), w)
			return
		}

		// If daemon running, restart it to pick up new config
		if daemonCtrl.IsRunning() {
			err = daemonCtrl.Stop()
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to stop daemon for restart: %v", err), w)
				return
			}
			err = daemonCtrl.Start(daemonExecutable, cfgDir)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to restart daemon: %v", err), w)
				return
			}
		} else {
		}

		updateToggleButton(toggleBtn, daemonCtrl)
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Root Directory to Watch", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rootEntry,

		widget.NewLabelWithStyle("Log Level (debug, info, warn, error)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		logLevelEntry,

		widget.NewLabelWithStyle("Exclude Patterns (one per line)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		excludePatterns,

		widget.NewLabelWithStyle("Processing Delay (e.g., 3s, 500ms)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		delayEntry,

		container.NewHBox(
			dryRunCheck,
			notificationsCheck,
		),

		container.NewGridWithColumns(2,
			saveBtn,
			toggleBtn,
		),
	)

	scroll := container.NewScroll(form)
	content := container.NewPadded(scroll)
	content = container.NewPadded(content)

	w.SetContent(content)

	a.Lifecycle().SetOnStopped(func() {
		err := daemonCtrl.Stop()
		if err != nil {
			println("Error stopping daemon on app exit:", err.Error())
		}
	})

	updateToggleButton(toggleBtn, daemonCtrl)

	w.ShowAndRun()
}

// updateToggleButton updates the toggle button label and status label color based on daemon state
func updateToggleButton(btn *widget.Button, ctrl *DaemonController) {
	if ctrl.IsRunning() {
		btn.SetText("Stop Daemon")
	} else {
		btn.SetText("Start Daemon")
	}
	btn.Refresh()
}
