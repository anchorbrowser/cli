package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	DefaultAppName    = "anchorbrowser"
	defaultConfigFile = "config.yaml"
)

// FileConfig contains non-secret local CLI configuration.
type FileConfig struct {
	ActiveKey     string             `mapstructure:"active_key" yaml:"active_key"`
	Keys          map[string]KeyMeta `mapstructure:"keys" yaml:"keys"`
	LastSessionID string             `mapstructure:"last_session_id,omitempty" yaml:"last_session_id,omitempty"`
}

// KeyMeta holds metadata for a named key profile. API keys are stored in OS keychain only.
type KeyMeta struct {
	CreatedAt string `mapstructure:"created_at,omitempty" yaml:"created_at,omitempty"`
}

// Manager reads/writes CLI config from the user's config directory.
type Manager struct {
	path string
}

func NewManager(appName string) (*Manager, error) {
	if appName == "" {
		appName = DefaultAppName
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config directory: %w", err)
	}
	cfgDir := filepath.Join(base, appName)
	return &Manager{path: filepath.Join(cfgDir, defaultConfigFile)}, nil
}

func NewManagerWithPath(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Load() (*FileConfig, error) {
	cfg := &FileConfig{Keys: map[string]KeyMeta{}}

	v := viper.New()
	v.SetConfigFile(m.path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFound) || errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", m.path, err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", m.path, err)
	}
	if cfg.Keys == nil {
		cfg.Keys = map[string]KeyMeta{}
	}
	return cfg, nil
}

func (m *Manager) Save(cfg *FileConfig) error {
	if cfg == nil {
		cfg = &FileConfig{}
	}
	if cfg.Keys == nil {
		cfg.Keys = map[string]KeyMeta{}
	}

	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(m.path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func ListKeyNames(cfg *FileConfig) []string {
	if cfg == nil || cfg.Keys == nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Keys))
	for k := range cfg.Keys {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
