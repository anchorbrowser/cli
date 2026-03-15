package auth

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"

	"github.com/anchorbrowser/cli/internal/config"
)

type memoryKeyring struct {
	items map[string]keyring.Item
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{items: map[string]keyring.Item{}}
}

func (m *memoryKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *memoryKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	item, err := m.Get(key)
	if err != nil {
		return keyring.Metadata{}, err
	}
	item.Data = nil
	return keyring.Metadata{Item: &item}, nil
}

func (m *memoryKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

func (m *memoryKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *memoryKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	cfg := config.NewManagerWithPath(filepath.Join(t.TempDir(), "config.yaml"))
	return NewStoreWithKeyring(cfg, newMemoryKeyring())
}

func TestResolvePrecedence(t *testing.T) {
	store := newTestStore(t)
	if err := store.Login("prod", "prod-key"); err != nil {
		t.Fatalf("login prod: %v", err)
	}
	if err := store.Login("staging", "staging-key"); err != nil {
		t.Fatalf("login staging: %v", err)
	}
	if err := store.Use("prod"); err != nil {
		t.Fatalf("use prod: %v", err)
	}

	resolved, err := store.Resolve("explicit-key", "", "env-key")
	if err != nil {
		t.Fatalf("resolve explicit: %v", err)
	}
	if resolved.Value != "explicit-key" || resolved.Source != "flag:api-key" {
		t.Fatalf("unexpected explicit result: %+v", resolved)
	}

	resolved, err = store.Resolve("", "staging", "env-key")
	if err != nil {
		t.Fatalf("resolve named: %v", err)
	}
	if resolved.Value != "staging-key" || resolved.Name != "staging" || resolved.Source != "flag:key" {
		t.Fatalf("unexpected named result: %+v", resolved)
	}

	resolved, err = store.Resolve("", "", "env-key")
	if err != nil {
		t.Fatalf("resolve env: %v", err)
	}
	if resolved.Value != "env-key" || resolved.Source != "env:"+EnvVarName {
		t.Fatalf("unexpected env result: %+v", resolved)
	}

	resolved, err = store.Resolve("", "", "")
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	if resolved.Value != "prod-key" || resolved.Name != "prod" || resolved.Source != "stored:active" {
		t.Fatalf("unexpected active result: %+v", resolved)
	}
}

func TestResolveMissingKey(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Resolve("", "", "")
	if !errors.Is(err, ErrNoAPIKeyConfigured) {
		t.Fatalf("expected ErrNoAPIKeyConfigured, got: %v", err)
	}
}

func TestRenameAndRemove(t *testing.T) {
	store := newTestStore(t)
	if err := store.Login("old", "secret"); err != nil {
		t.Fatalf("login old: %v", err)
	}
	if err := store.Rename("old", "new"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	names, active, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "new" {
		t.Fatalf("unexpected names after rename: %#v", names)
	}
	if active != "new" {
		t.Fatalf("unexpected active after rename: %s", active)
	}

	if err := store.Remove("new"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	names, active, err = store.List()
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(names) != 0 || active != "" {
		t.Fatalf("expected no keys after remove, got names=%v active=%q", names, active)
	}
}
