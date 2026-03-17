package backend

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	PinnedVersion = "v0.20.13"
	releaseBase   = "https://github.com/vercel-labs/agent-browser/releases/download/" + PinnedVersion + "/"
)

type Asset struct {
	Name   string
	Digest string
}

var assetByPlatform = map[string]Asset{
	"darwin/arm64": {Name: "agent-browser-darwin-arm64", Digest: "sha256:85bc226bacfb4cd80f0fde0e8c54e4c8b01d4a2d3a27d422c9b0598c6d4e42dc"},
	"darwin/amd64": {Name: "agent-browser-darwin-x64", Digest: "sha256:9b05e425708eec2bea907b78d51d0e6674c0b47db288624f39f630981b5d9270"},
	"linux/amd64":  {Name: "agent-browser-linux-x64", Digest: "sha256:35c4ba45c2b45fc7da19f3f10d330d328e7755cfc7c873b6201a083ad6d0ad99"},
	"linux/arm64":  {Name: "agent-browser-linux-arm64", Digest: "sha256:f9db79671c0c84a65d258a823f232c3438943db977c724b5c44dd5181392a886"},
	"windows/amd64": {
		Name:   "agent-browser-win32-x64.exe",
		Digest: "sha256:228cfbee4bd3be8556ab176c2d3e2e71569bc6436c93a5b40f4999da92498aa3",
	},
}

func PlatformKey(goos, goarch string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSpace(goos), strings.TrimSpace(goarch))
}

func CurrentAsset() (Asset, error) {
	return AssetFor(runtime.GOOS, runtime.GOARCH)
}

func AssetFor(goos, goarch string) (Asset, error) {
	key := PlatformKey(goos, goarch)
	asset, ok := assetByPlatform[key]
	if !ok {
		return Asset{}, fmt.Errorf("agent-browser backend is unsupported on %s", key)
	}
	return asset, nil
}

func AssetURL(asset Asset) string {
	return releaseBase + asset.Name
}

func BinaryFileName() string {
	if runtime.GOOS == "windows" {
		return "agent-browser.exe"
	}
	return "agent-browser"
}
