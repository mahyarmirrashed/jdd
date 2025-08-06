package main

import (
	"os"
	"os/exec"
	"sync"

	log "github.com/sirupsen/logrus"
)

// DaemonController manages the daemon subprocess lifecycle and state.
type DaemonController struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	onExit  func()
}

func NewDaemonController() *DaemonController {
	return &DaemonController{}
}

// Start the daemon if not running.
func (d *DaemonController) Start(executable, rootDir string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return nil // Already running
	}

	d.cmd = exec.Command(executable)
	d.cmd.Dir = rootDir
	d.cmd.Stdout = os.Stdout
	d.cmd.Stderr = os.Stderr

	err := d.cmd.Start()
	if err != nil {
		d.cmd = nil
		d.running = false
		return err
	}

	d.running = true
	go d.waitForExit()
	log.Printf("Daemon started with PID %d\n", d.cmd.Process.Pid)
	return nil
}

// Stop the daemon if running.
func (d *DaemonController) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running || d.cmd == nil || d.cmd.Process == nil {
		return nil // Already stopped
	}

	err := d.cmd.Process.Kill()
	if err != nil {
		return err
	}

	log.Println("Daemon instructed to stop")
	return err
}

// Toggle the daemon state (start if stopped, stop if running).
func (d *DaemonController) Toggle(executable, rootDir string) error {
	if d.IsRunning() {
		return d.Stop()
	}
	return d.Start(executable, rootDir)
}

// Check if daemon is running.
func (d *DaemonController) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.running
}

func (d *DaemonController) waitForExit() {
	err := d.cmd.Wait()
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cmd = nil
	d.running = false

	if err != nil {
		log.Println("Daemon exited with error:", err)
	} else {
		log.Println("Daemon exited")
	}

	if d.onExit != nil {
		d.onExit()
	}
}
