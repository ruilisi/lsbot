package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ruilisi/lsbot/internal/mcp"
	"github.com/spf13/cobra"
)

const manifestURL = "https://files.lingti.com/lsbot-version.json"

type upgradeManifest struct {
	Version string `json:"version"`
	Files   []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"files"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade lsbot to the latest version",
	Long:  `Download and install the latest lsbot binary from files.lingti.com.`,
	Run:   runUpgrade,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	if goos != "darwin" && goos != "linux" {
		fmt.Fprintf(os.Stderr, "upgrade is not supported on %s; download manually from files.lingti.com\n", goos)
		os.Exit(1)
	}

	fmt.Println("Fetching latest version info...")
	manifest, err := fetchUpgradeManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to fetch manifest: %v\n", err)
		os.Exit(1)
	}

	latest := manifest.Version
	current := mcp.ServerVersion

	if latest == current {
		fmt.Printf("lsbot is already up to date (%s)\n", current)
		return
	}

	fmt.Printf("Upgrading %s → %s\n", current, latest)

	target := fmt.Sprintf("lsbot-%s-%s-%s", latest, goos, goarch)
	downloadURL := ""
	for _, f := range manifest.Files {
		if f.Name == target {
			downloadURL = f.URL
			break
		}
	}
	if downloadURL == "" {
		fmt.Fprintf(os.Stderr, "Error: no binary available for %s/%s (looking for %q)\n", goos, goarch, target)
		os.Exit(1)
	}

	// Download to a temp file
	tmpFile, err := os.CreateTemp("", "lsbot-upgrade-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create temp file: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	fmt.Printf("Downloading %s...\n", downloadURL)
	if err := downloadUpgradeFile(downloadURL, tmpFile); err != nil {
		tmpFile.Close()
		fmt.Fprintf(os.Stderr, "Error: download failed: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to chmod: %v\n", err)
		os.Exit(1)
	}

	// Verify the downloaded binary works
	out, err := exec.Command(tmpPath, "version").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: downloaded binary failed self-check: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Verified: %s\n", strings.TrimSpace(string(out)))

	// Determine install path (where the current binary lives)
	installPath, err := resolveUpgradeInstallPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine install path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Installing to %s...\n", installPath)
	if err := upgradeInstallBinary(tmpPath, installPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: install failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("lsbot upgraded to %s successfully!\n", latest)
}

func fetchUpgradeManifest() (*upgradeManifest, error) {
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var m upgradeManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	if m.Version == "" {
		return nil, fmt.Errorf("manifest missing version field")
	}
	return &m, nil
}

func downloadUpgradeFile(url string, dst *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}

// resolveUpgradeInstallPath returns the absolute path of the running binary,
// following symlinks.
func resolveUpgradeInstallPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// upgradeInstallBinary moves src to dst, using sudo if the destination
// directory is not writable by the current user.
func upgradeInstallBinary(src, dst string) error {
	dir := filepath.Dir(dst)
	if upgradeIsWritable(dir) {
		return os.Rename(src, dst)
	}

	// Not writable — try sudo mv
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("cannot write to %s and sudo is not available", dir)
	}
	fmt.Printf("(sudo required to write to %s)\n", dir)
	out, err := exec.Command("sudo", "mv", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sudo mv failed: %v\n%s", err, string(out))
	}
	return nil
}

func upgradeIsWritable(dir string) bool {
	testFile := filepath.Join(dir, ".lsbot-write-test")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}
