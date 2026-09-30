package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
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
	goos := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		// already correct
	case "arm64":
		// already correct
	case "arm":
		arch = "armv7"
	case "386":
		arch = "386"
	}
	return fmt.Sprintf("%s_%s", goos, arch)
}

// archiveExt returns the file extension used for release archives.
// Windows releases use .zip; everything else uses .tar.gz.
func archiveExt() string {
	if runtime.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// assetName returns the expected release asset filename for the current platform,
// e.g. hikari_1.2.3_linux_amd64.tar.gz or hikari_1.2.3_windows_amd64.zip
func assetName(tag string) string {
	v := strings.TrimPrefix(tag, "v")
	return fmt.Sprintf("%s_%s_%s%s", updateBinary, v, platformSuffix(), archiveExt())
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

	archive := archiveExt()
	archivePath := filepath.Join(tmpDir, "update"+archive)
	f, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("writing download: %w", err)
	}
	f.Close()

	// Extract the binary from the archive.
	binPath, err := extractBinary(archivePath, tmpDir)
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
	if err := moveOrCopy(binPath, exe); err != nil {
		// Try to restore
		_ = os.Rename(oldPath, exe)
		return fmt.Errorf("replacing binary: %w", err)
	}
	// Remove backup
	_ = os.Remove(oldPath)

	return nil
}

// moveOrCopy moves src to dst. If src and dst reside on different filesystems,
// os.Rename fails with EXDEV ("invalid cross-device link"). In that case, it
// falls back to copying file contents and permissions, then removes src.
func moveOrCopy(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	fi, err := sf.Stat()
	if err != nil {
		return err
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}

	if _, err := io.Copy(df, sf); err != nil {
		df.Close()
		_ = os.Remove(dst)
		return err
	}

	if err := df.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}

	if err := os.Chmod(dst, fi.Mode().Perm()); err != nil {
		return err
	}

	_ = os.Remove(src)
	return nil
}

// extractBinary unpacks the archive at archivePath and returns the path of
// the extracted hikari binary inside destDir.
func extractBinary(archivePath, destDir string) (string, error) {
	outDir := filepath.Join(destDir, "extracted")
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return "", err
	}

	if filepath.Ext(archivePath) == ".zip" {
		// Windows: use PowerShell to extract the zip
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", //nolint:gosec
			"-Command", fmt.Sprintf("Expand-Archive -Force -Path '%s' -DestinationPath '%s'", archivePath, outDir))
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("powershell unzip: %w\n%s", err, out)
		}
	} else {
		// Linux / macOS: use system tar
		cmd := tarCmd(archivePath, outDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("tar: %w\n%s", err, out)
		}
	}

	binName := updateBinary
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	// Walk the extracted directory to find the binary.
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
	want := assetName(latest)
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
		if runtime.GOOS == "windows" {
			fmt.Println("Please download the .zip from:")
			fmt.Printf("  https://github.com/%s/releases/latest\n", updateRepo)
		} else {
			fmt.Println("You can update from source by running:")
			fmt.Printf("  curl -fsSL https://raw.githubusercontent.com/%s/main/scripts/install.sh | sh\n", updateRepo)
		}
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
