//go:build !darwin

package auth

import "github.com/99designs/keyring"

func openKeyring(serviceName string, backends []keyring.BackendType) (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName:     serviceName,
		AllowedBackends: backends,
	})
}
