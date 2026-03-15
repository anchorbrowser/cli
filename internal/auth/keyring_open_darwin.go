//go:build darwin

package auth

import (
	"errors"
	"fmt"

	"github.com/99designs/keyring"
)

func openKeyring(serviceName string, backends []keyring.BackendType) (keyring.Keyring, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     serviceName,
		AllowedBackends: backends,
	})
	if err == nil {
		return ring, nil
	}
	if !errors.Is(err, keyring.ErrNoAvailImpl) {
		return nil, err
	}

	fallback, fallbackErr := newMacSecurityKeyring(serviceName)
	if fallbackErr != nil {
		return nil, fmt.Errorf("%w; macos fallback failed: %v", err, fallbackErr)
	}
	return fallback, nil
}
