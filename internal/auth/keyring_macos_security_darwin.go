//go:build darwin

package auth

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/99designs/keyring"
)

type macSecurityKeyring struct {
	serviceName string
}

func newMacSecurityKeyring(serviceName string) (keyring.Keyring, error) {
	if _, err := exec.LookPath("security"); err != nil {
		return nil, fmt.Errorf("security command not found: %w", err)
	}
	return &macSecurityKeyring{serviceName: serviceName}, nil
}

func (m *macSecurityKeyring) Get(key string) (keyring.Item, error) {
	out, err := runSecurity("find-generic-password", "-s", m.serviceName, "-a", key, "-w")
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return keyring.Item{}, err
		}
		return keyring.Item{}, fmt.Errorf("read key %q from macos keychain: %w", key, err)
	}

	return keyring.Item{
		Key:  key,
		Data: bytes.TrimSpace(out),
	}, nil
}

func (m *macSecurityKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, keyring.ErrMetadataNotSupported
}

func (m *macSecurityKeyring) Set(item keyring.Item) error {
	args := []string{"add-generic-password", "-s", m.serviceName, "-a", item.Key, "-w", string(item.Data)}
	if strings.TrimSpace(item.Label) != "" {
		args = append(args, "-l", item.Label)
	}
	args = append(args, "-U")
	if _, err := runSecurity(args...); err != nil {
		return fmt.Errorf("write key %q to macos keychain: %w", item.Key, err)
	}
	return nil
}

func (m *macSecurityKeyring) Remove(key string) error {
	if _, err := runSecurity("delete-generic-password", "-s", m.serviceName, "-a", key); err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return err
		}
		return fmt.Errorf("delete key %q from macos keychain: %w", key, err)
	}
	return nil
}

func (m *macSecurityKeyring) Keys() ([]string, error) {
	return nil, errors.New("listing macos keychain keys is not supported")
}

func runSecurity(args ...string) ([]byte, error) {
	cmd := exec.Command("security", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, nil
	}

	msg := strings.ToLower(string(bytes.TrimSpace(out)))
	if strings.Contains(msg, "could not be found") || strings.Contains(msg, "the specified item could not be found") {
		return nil, keyring.ErrKeyNotFound
	}
	return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
}
