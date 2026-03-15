package auth

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/99designs/keyring"

	"github.com/anchorbrowser/cli/internal/config"
)

const (
	ServiceName = "anchorbrowser-cli"
	EnvVarName  = "ANCHORBROWSER_API_KEY"
)

var (
	ErrNoAPIKeyConfigured = errors.New("no API key configured")
	ErrKeyNotFound        = errors.New("named API key not found")
)

// ResolvedKey is the selected API key and metadata after precedence resolution.
type ResolvedKey struct {
	Name   string
	Value  string
	Source string
}

// Store persists key metadata in config and secret values in OS keychain.
type Store struct {
	cfgManager *config.Manager
	ring       keyring.Keyring
}

func NewStore(cfgManager *config.Manager) (*Store, error) {
	ring, err := openKeyring(ServiceName, allowedBackends())
	if err != nil {
		return nil, fmt.Errorf("open secure keychain failed: %w. in headless environments, use %s", err, EnvVarName)
	}
	return &Store{cfgManager: cfgManager, ring: ring}, nil
}

func NewStoreWithKeyring(cfgManager *config.Manager, ring keyring.Keyring) *Store {
	return &Store{cfgManager: cfgManager, ring: ring}
}

func (s *Store) Login(name, apiKey string) error {
	name = normalizeName(name)
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	if err := s.ring.Set(keyring.Item{
		Key:   keyringItemKey(name),
		Label: fmt.Sprintf("AnchorBrowser API key (%s)", name),
		Data:  []byte(apiKey),
	}); err != nil {
		return fmt.Errorf("save API key in keychain: %w", err)
	}

	cfg, err := s.cfgManager.Load()
	if err != nil {
		return err
	}
	if cfg.Keys == nil {
		cfg.Keys = map[string]config.KeyMeta{}
	}
	cfg.Keys[name] = config.KeyMeta{CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	cfg.ActiveKey = name
	return s.cfgManager.Save(cfg)
}

func (s *Store) List() ([]string, string, error) {
	cfg, err := s.cfgManager.Load()
	if err != nil {
		return nil, "", err
	}
	names := config.ListKeyNames(cfg)
	return names, cfg.ActiveKey, nil
}

func (s *Store) Use(name string) error {
	name = normalizeName(name)
	if _, err := s.getNamedKey(name); err != nil {
		return err
	}
	cfg, err := s.cfgManager.Load()
	if err != nil {
		return err
	}
	cfg.ActiveKey = name
	return s.cfgManager.Save(cfg)
}

func (s *Store) Remove(name string) error {
	name = normalizeName(name)
	if err := s.ring.Remove(keyringItemKey(name)); err != nil {
		if !errors.Is(err, keyring.ErrKeyNotFound) {
			return fmt.Errorf("remove key from keychain: %w", err)
		}
	}
	cfg, err := s.cfgManager.Load()
	if err != nil {
		return err
	}
	delete(cfg.Keys, name)
	if cfg.ActiveKey == name {
		cfg.ActiveKey = ""
		names := config.ListKeyNames(cfg)
		if len(names) > 0 {
			cfg.ActiveKey = names[0]
		}
	}
	return s.cfgManager.Save(cfg)
}

func (s *Store) Rename(oldName, newName string) error {
	oldName = normalizeName(oldName)
	newName = normalizeName(newName)
	if oldName == newName {
		return nil
	}
	key, err := s.getNamedKey(oldName)
	if err != nil {
		return err
	}

	if err := s.ring.Set(keyring.Item{
		Key:   keyringItemKey(newName),
		Label: fmt.Sprintf("AnchorBrowser API key (%s)", newName),
		Data:  []byte(key),
	}); err != nil {
		return fmt.Errorf("save renamed key: %w", err)
	}
	if err := s.ring.Remove(keyringItemKey(oldName)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("remove old key after rename: %w", err)
	}

	cfg, err := s.cfgManager.Load()
	if err != nil {
		return err
	}
	meta := cfg.Keys[oldName]
	delete(cfg.Keys, oldName)
	cfg.Keys[newName] = meta
	if cfg.ActiveKey == oldName {
		cfg.ActiveKey = newName
	}
	return s.cfgManager.Save(cfg)
}

func (s *Store) Current() (string, error) {
	cfg, err := s.cfgManager.Load()
	if err != nil {
		return "", err
	}
	return cfg.ActiveKey, nil
}

// Resolve returns the API key using precedence:
// explicit API key > named key flag > env var > active stored key.
func (s *Store) Resolve(explicitAPIKey, namedKey, envValue string) (*ResolvedKey, error) {
	if strings.TrimSpace(explicitAPIKey) != "" {
		return &ResolvedKey{Value: explicitAPIKey, Source: "flag:api-key"}, nil
	}

	if strings.TrimSpace(namedKey) != "" {
		key, err := s.getNamedKey(namedKey)
		if err != nil {
			return nil, err
		}
		return &ResolvedKey{Name: normalizeName(namedKey), Value: key, Source: "flag:key"}, nil
	}

	if strings.TrimSpace(envValue) != "" {
		return &ResolvedKey{Value: envValue, Source: "env:" + EnvVarName}, nil
	}

	cfg, err := s.cfgManager.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ActiveKey) == "" {
		return nil, ErrNoAPIKeyConfigured
	}
	key, err := s.getNamedKey(cfg.ActiveKey)
	if err != nil {
		return nil, err
	}
	return &ResolvedKey{Name: cfg.ActiveKey, Value: key, Source: "stored:active"}, nil
}

func (s *Store) getNamedKey(name string) (string, error) {
	name = normalizeName(name)
	item, err := s.ring.Get(keyringItemKey(name))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", fmt.Errorf("%w: %s", ErrKeyNotFound, name)
		}
		return "", fmt.Errorf("read key %q from keychain: %w", name, err)
	}
	if len(item.Data) == 0 {
		return "", fmt.Errorf("stored key %q is empty", name)
	}
	return string(item.Data), nil
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func keyringItemKey(name string) string {
	return "api-key:" + normalizeName(name)
}

func allowedBackends() []keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return []keyring.BackendType{keyring.KeychainBackend}
	case "windows":
		return []keyring.BackendType{keyring.WinCredBackend}
	default:
		backs := []keyring.BackendType{keyring.SecretServiceBackend, keyring.KWalletBackend, keyring.PassBackend}
		sort.SliceStable(backs, func(i, j int) bool { return i < j })
		return backs
	}
}
