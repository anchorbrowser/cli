package backend

import (
	"strings"
	"testing"
)

func TestAssetForKnownPlatforms(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
	}{
		{goos: "darwin", goarch: "arm64"},
		{goos: "darwin", goarch: "amd64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm64"},
		{goos: "windows", goarch: "amd64"},
	}

	for _, tt := range tests {
		asset, err := AssetFor(tt.goos, tt.goarch)
		if err != nil {
			t.Fatalf("AssetFor(%s,%s): %v", tt.goos, tt.goarch, err)
		}
		if strings.TrimSpace(asset.Name) == "" {
			t.Fatalf("AssetFor(%s,%s) returned empty name", tt.goos, tt.goarch)
		}
		if !strings.HasPrefix(asset.Digest, "sha256:") {
			t.Fatalf("AssetFor(%s,%s) returned invalid digest %q", tt.goos, tt.goarch, asset.Digest)
		}
		if !strings.Contains(AssetURL(asset), PinnedVersion) {
			t.Fatalf("AssetURL(%s) does not include pinned version %s", asset.Name, PinnedVersion)
		}
	}
}

func TestAssetForUnsupportedPlatform(t *testing.T) {
	if _, err := AssetFor("freebsd", "amd64"); err == nil {
		t.Fatalf("expected unsupported platform error")
	}
}
