package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateRepo    = "NubiMa/hikari-cli"
	updateBinary  = "hikari"
	updateTimeout = 60 * time.Second
)

// updateCmd returns the "hikari update" subcommand.
func updateCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for a new Hikari release and update the binary",
		Long: `Checks GitHub for a newer release of Hikari.

If a new version is found it downloads the pre-built binary for your
platform, replaces the currently running executable, and prints the
new version. Requires internet access.

Examples:
  hikari update             # check and update
  hikari update --check     # only report available version, don't update`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(checkOnly)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only report if an update is available; do not download")
	return cmd
}

// ---------------------------------------------------------------------------
// GitHub release helpers
// ---------------------------------------------------------------------------

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease() (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", updateRepo)
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reaching GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &rel, nil
}

// ---------------------------------------------------------------------------
// Platform helpers
// ---------------------------------------------------------------------------

func platformSuffix() string {
	os_ := runtime.GOOS
	arch := runtime.GOARCH
	// normalise to the naming convention used by GoReleaser
	if arch == "amd64" {
		arch = "amd64"
	}
	return fmt.Sprintf("%s_%s", os_, arch)
}

// tarballName returns the expected release asset filename for the current
// platform, e.g. hikari_1.2.3_linux_amd64.tar.gz
func tarballName(tag string) string {
	v := strings.TrimPrefix(tag, "v")
	return fmt.Sprintf("%s_%s_%s.tar.gz", updateBinary, v, platformSuffix())
}

// ---------------------------------------------------------------------------
// Self-replace
// ---------------------------------------------------------------------------

// selfPath returns the absolute path of the currently running executable.
func selfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// downloadAndReplace fetches the tarball from url, extracts the binary, and
// replaces the running executable atomically using a rename.
func downloadAndReplace(assetURL string) error {
	client := &http.Client{Timeout: updateTimeout}

	fmt.Printf("  Downloading %s…\n", assetURL)
	resp, err := client.Get(assetURL) //nolint:noctx // intentional simple GET
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	// Write to a temp file first.
	tmpDir, err := os.MkdirTemp("", "hikari-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, "update.tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("writing download: %w", err)
	}
	f.Close()

	// Extract the binary from the tarball.
	binPath, err := extractBinary(tarPath, tmpDir)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Replace the running executable.
	exe, err := selfPath()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}

	// Rename current → current.old (so we can restore if needed), then move new binary in.
	oldPath := exe + ".old"
	if err := os.Rename(exe, oldPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}
	if err := os.Rename(binPath, exe); err != nil {
		// Try to restore
		_ = os.Rename(oldPath, exe)
		return fmt.Errorf("replacing binary: %w", err)
	}
	// Remove backup
	_ = os.Remove(oldPath)

	return nil
}

// extractBinary untars the archive at tarPath and returns the path of the
// extracted hikari binary inside destDir.
func extractBinary(tarPath, destDir string) (string, error) {
	// We use the system tar command to avoid importing a tar dependency.
	// This works on Linux, macOS, and Git Bash / WSL on Windows.
	outDir := filepath.Join(destDir, "extracted")
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return "", err
	}

	// #nosec G204 — tarPath and outDir are controlled by us
	cmd := tarCmd(tarPath, outDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar: %w\n%s", err, out)
	}

	// Find the binary inside the extracted directory.
	binName := updateBinary
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	candidates := []string{
		filepath.Join(outDir, binName),
		filepath.Join(outDir, updateBinary+"_*", binName),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			if err := os.Chmod(c, 0755); err != nil { //nolint:gosec // binary needs execute bit
				return "", err
			}
			return c, nil
		}
	}

	// Fall back: walk the extracted dir and find the binary by name.
	var found string
	_ = filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) == binName {
			found = path
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("binary %q not found in archive", binName)
	}
	if err := os.Chmod(found, 0755); err != nil { //nolint:gosec
		return "", err
	}
	return found, nil
}

// ---------------------------------------------------------------------------
// Update entry point
// ---------------------------------------------------------------------------

func runUpdate(checkOnly bool) error {
	fmt.Printf("Checking for updates (current: %s)…\n", version)

	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	latest := rel.TagName // e.g. "v1.2.3"
	latestPlain := strings.TrimPrefix(latest, "v")

	// The build-time version may be "dev", a short hash, or a semver tag.
	// We consider the user up-to-date only when version exactly matches the tag.
	if version != "dev" && (version == latestPlain || "v"+version == latest) {
		fmt.Printf("✓ Already on the latest version (%s). Nothing to do.\n", latest)
		return nil
	}

	fmt.Printf("  Latest release: %s\n", latest)
	if version == "dev" {
		fmt.Println("  (current build is a dev/source build)")
	} else {
		fmt.Printf("  Installed:      v%s\n", version)
	}

	if checkOnly {
		fmt.Println("\nRun 'hikari update' (without --check) to install the update.")
		return nil
	}

	// Find the asset for this platform.
	want := tarballName(latest)
	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == want {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		// No prebuilt binary — fall back to pointing the user at install.sh
		fmt.Printf("\nNo prebuilt binary found for %s (%s).\n", want, platformSuffix())
		fmt.Println("You can update from source by running:")
		fmt.Printf("  curl -fsSL https://raw.githubusercontent.com/%s/main/scripts/install.sh | sh\n", updateRepo)
		return nil
	}

	exe, err := selfPath()
	if err != nil {
		return fmt.Errorf("locating current binary: %w", err)
	}
	fmt.Printf("  Updating %s → %s…\n", exe, latest)

	if err := downloadAndReplace(downloadURL); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("\n✓ Hikari updated to %s\n", latest)
	fmt.Println("  Restart hikari to use the new version.")
	return nil
}
