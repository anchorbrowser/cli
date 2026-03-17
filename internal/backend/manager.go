package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Manager struct {
	baseDir string
	client  *http.Client
}

type Status struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Path      string `json:"path,omitempty"`
	Asset     string `json:"asset,omitempty"`
	Supported bool   `json:"supported"`
}

type Doctor struct {
	Platform        string `json:"platform"`
	Supported       bool   `json:"supported"`
	Installed       bool   `json:"installed"`
	ExecutablePath  string `json:"executable_path,omitempty"`
	ExecutableWorks bool   `json:"executable_works"`
	VersionOutput   string `json:"version_output,omitempty"`
}

func NewManager(appName string) (*Manager, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config dir: %w", err)
	}
	if strings.TrimSpace(appName) == "" {
		appName = "anchorbrowser"
	}
	return NewManagerWithBaseDir(filepath.Join(base, appName, "backend")), nil
}

func NewManagerWithBaseDir(baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (m *Manager) BinaryPath() (string, error) {
	_, err := CurrentAsset()
	if err != nil {
		return "", err
	}
	return filepath.Join(m.baseDir, "agent-browser", PinnedVersion, BinaryFileName()), nil
}

func (m *Manager) EnsureInstalled(ctx context.Context) (string, error) {
	path, err := m.BinaryPath()
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return path, nil
	}
	return m.Install(ctx)
}

func (m *Manager) Install(ctx context.Context) (string, error) {
	asset, err := CurrentAsset()
	if err != nil {
		return "", err
	}
	targetPath, err := m.BinaryPath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("create backend directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(targetPath), "agent-browser-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AssetURL(asset), nil)
	if err != nil {
		return "", fmt.Errorf("build backend request: %w", err)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download backend: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return "", fmt.Errorf("download backend failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("write backend binary: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("sync backend binary: %w", err)
	}
	if err := verifyChecksum(tmpPath, asset.Digest); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close backend binary: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil && !os.IsPermission(err) {
		return "", fmt.Errorf("chmod backend binary: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return "", fmt.Errorf("install backend binary: %w", err)
	}
	return targetPath, nil
}

func verifyChecksum(path, expectedDigest string) error {
	expected := strings.TrimSpace(expectedDigest)
	expected = strings.TrimPrefix(strings.ToLower(expected), "sha256:")
	if len(expected) == 0 {
		return fmt.Errorf("missing checksum for backend asset")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open backend binary for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash backend binary: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("backend checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func (m *Manager) Status() (Status, error) {
	asset, err := CurrentAsset()
	if err != nil {
		return Status{
			Installed: false,
			Version:   PinnedVersion,
			Supported: false,
		}, nil
	}
	path, err := m.BinaryPath()
	if err != nil {
		return Status{}, err
	}
	_, statErr := os.Stat(path)
	return Status{
		Installed: statErr == nil,
		Version:   PinnedVersion,
		Path:      path,
		Asset:     asset.Name,
		Supported: true,
	}, nil
}

func (m *Manager) Uninstall() error {
	root := filepath.Join(m.baseDir, "agent-browser")
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("remove backend directory: %w", err)
	}
	return nil
}

func (m *Manager) Doctor(ctx context.Context) (Doctor, error) {
	platformAsset, assetErr := CurrentAsset()
	platform := PlatformKey(runtime.GOOS, runtime.GOARCH)
	report := Doctor{
		Platform:  platform,
		Supported: assetErr == nil,
	}
	if assetErr != nil {
		return report, nil
	}

	path, err := m.BinaryPath()
	if err != nil {
		return report, err
	}
	report.ExecutablePath = path
	if _, err := os.Stat(path); err != nil {
		return report, nil
	}
	report.Installed = true

	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.CombinedOutput()
	report.VersionOutput = strings.TrimSpace(string(output))
	report.ExecutableWorks = err == nil
	if report.VersionOutput == "" {
		report.VersionOutput = platformAsset.Name
	}
	return report, nil
}
