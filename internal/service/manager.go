package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

const (
	ServiceName = "com.lingti.bot"
)

// Paths returns the installation paths for the current platform
func Paths() (binaryPath, configPath string) {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/PrivilegedHelperTools/com.lingti.bot",
			"/Library/LaunchDaemons/com.lingti.bot.plist"
	case "linux":
		return "/usr/local/bin/lsbot",
			"/etc/systemd/system/lsbot.service"
	default:
		return "", ""
	}
}

// IsInstalled checks if the service is installed
func IsInstalled() bool {
	binaryPath, _ := Paths()
	if binaryPath == "" {
		return false
	}
	_, err := os.Stat(binaryPath)
	return err == nil
}

// IsRunning checks if the service is running
func IsRunning() bool {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("launchctl", "list", ServiceName)
		return cmd.Run() == nil
	case "linux":
		cmd := exec.Command("systemctl", "is-active", "--quiet", "lsbot")
		return cmd.Run() == nil
	default:
		return false
	}
}

// Install installs the service
func Install(sourceBinary string) error {
	binaryPath, configPath := Paths()
	if binaryPath == "" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Copy binary
	if err := copyBinary(sourceBinary, binaryPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Create service config
	if err := createServiceConfig(configPath, binaryPath); err != nil {
		return fmt.Errorf("failed to create service config: %w", err)
	}

	// Load/enable service
	if err := enableService(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	return nil
}

// Uninstall removes the service
func Uninstall() error {
	// Stop service first
	_ = Stop()

	binaryPath, configPath := Paths()

	switch runtime.GOOS {
	case "darwin":
		exec.Command("launchctl", "unload", configPath).Run()
	case "linux":
		exec.Command("systemctl", "disable", "lsbot").Run()
	}

	// Remove files
	os.Remove(configPath)
	os.Remove(binaryPath)

	return nil
}

// Start starts the service
func Start() error {
	switch runtime.GOOS {
	case "darwin":
		_, configPath := Paths()
		return exec.Command("launchctl", "load", configPath).Run()
	case "linux":
		return exec.Command("systemctl", "start", "lsbot").Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Stop stops the service
func Stop() error {
	switch runtime.GOOS {
	case "darwin":
		_, configPath := Paths()
		return exec.Command("launchctl", "unload", configPath).Run()
	case "linux":
		return exec.Command("systemctl", "stop", "lsbot").Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Restart restarts the service
func Restart() error {
	if err := Stop(); err != nil {
		// Ignore stop error, service might not be running
	}
	return Start()
}

func copyBinary(src, dst string) error {
	// Ensure directory exists
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Read source
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write destination
	if err := os.WriteFile(dst, data, 0755); err != nil {
		return err
	}

	return nil
}

func createServiceConfig(configPath, binaryPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return createLaunchdPlist(configPath, binaryPath)
	case "linux":
		return createSystemdUnit(configPath, binaryPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func enableService() error {
	switch runtime.GOOS {
	case "darwin":
		_, configPath := Paths()
		return exec.Command("launchctl", "load", configPath).Run()
	case "linux":
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return err
		}
		return exec.Command("systemctl", "enable", "lsbot").Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>serve</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/lsbot.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/lsbot.log</string>
</dict>
</plist>
`

func createLaunchdPlist(configPath, binaryPath string) error {
	tmpl, err := template.New("plist").Parse(launchdPlistTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]string{
		"Label":      ServiceName,
		"BinaryPath": binaryPath,
	})
}

const systemdUnitTemplate = `[Unit]
Description=Lingti Bot MCP Server
After=network.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve
Restart=always
RestartSec=5
StandardOutput=append:/tmp/lsbot.log
StandardError=append:/tmp/lsbot.log

[Install]
WantedBy=multi-user.target
`

func createSystemdUnit(configPath, binaryPath string) error {
	tmpl, err := template.New("unit").Parse(systemdUnitTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]string{
		"BinaryPath": binaryPath,
	})
}
