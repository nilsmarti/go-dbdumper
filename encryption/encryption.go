// Package encryption provides streaming asymmetric encryption and decryption
// of backup payloads using age (https://age-encryption.org).
//
// The application holds only the public key (age recipient) and can therefore
// encrypt backups without ever being able to decrypt them. Decryption requires
// the corresponding private key (age identity), which should live on a
// separate, hardened restore host — never on the app server.
package encryption

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
)

// ParseRecipients parses one or more age public keys (recipients). Each key
// string must be an age recipient in Bech32 form starting with "age1".
// Multiple keys may be provided as separate lines or as a single
// newline/comma-separated string.
func ParseRecipients(keys ...string) ([]age.Recipient, error) {
	var recipients []age.Recipient
	for _, key := range keys {
		for _, line := range strings.FieldsFunc(key, func(r rune) bool { return r == '\n' || r == ',' }) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			r, err := age.ParseX25519Recipient(line)
			if err != nil {
				return nil, fmt.Errorf("invalid age recipient %q: %w", line, err)
			}
			recipients = append(recipients, r)
		}
	}
	if len(recipients) == 0 {
		return nil, errors.New("no age recipients provided")
	}
	return recipients, nil
}

// LoadRecipientsFromFile reads age recipients from a file. Each non-empty,
// non-comment line is treated as a recipient key. Lines starting with "#" are
// ignored.
func LoadRecipientsFromFile(path string) ([]age.Recipient, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipients file: %w", err)
	}

	var keys []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keys = append(keys, line)
	}

	return ParseRecipients(keys...)
}

// Encrypt wraps the provided writer in an age encryptor. Plaintext written to
// the returned io.WriteCloser is streamed as ciphertext to dst. The returned
// writer MUST be closed to flush the final encrypted chunk.
func Encrypt(dst io.Writer, recipients []age.Recipient) (io.WriteCloser, error) {
	if len(recipients) == 0 {
		return nil, errors.New("no recipients provided")
	}
	return age.Encrypt(dst, recipients...)
}

// Decrypt wraps the provided reader in an age decryptor. Ciphertext read from
// src is streamed as plaintext through the returned io.Reader. One of the
// provided identities must match a recipient the file was encrypted to.
func Decrypt(src io.Reader, identities []age.Identity) (io.Reader, error) {
	if len(identities) == 0 {
		return nil, errors.New("no identities provided")
	}
	return age.Decrypt(src, identities...)
}

// ParseIdentity parses a single age private key (identity) in Bech32 form
// starting with "AGE-SECRET-KEY-1".
func ParseIdentity(key string) (age.Identity, error) {
	key = strings.TrimSpace(key)
	id, err := age.ParseX25519Identity(key)
	if err != nil {
		return nil, fmt.Errorf("invalid age identity: %w", err)
	}
	return id, nil
}

// LoadIdentityFromFile reads a single age identity from a file.
func LoadIdentityFromFile(path string) (age.Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read identity file: %w", err)
	}
	return ParseIdentity(string(data))
}
