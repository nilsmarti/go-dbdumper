package encryption

import (
	"bytes"
	"io"
	"testing"

	"filippo.io/age"
)

// generateTestIdentity creates an age X25519 identity and its corresponding
// recipient for testing.
func generateTestIdentity(t *testing.T) (*age.X25519Identity, *age.X25519Recipient) {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	return identity, identity.Recipient()
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	identity, recipient := generateTestIdentity(t)

	plaintext := []byte("sensitive PII data that needs to be encrypted at rest")

	// Encrypt
	var ciphertext bytes.Buffer
	encWriter, err := Encrypt(&ciphertext, []age.Recipient{recipient})
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := encWriter.Write(plaintext); err != nil {
		t.Fatalf("Write to encryptor failed: %v", err)
	}
	if err := encWriter.Close(); err != nil {
		t.Fatalf("Close encryptor failed: %v", err)
	}

	// The ciphertext must not contain the plaintext.
	if bytes.Contains(ciphertext.Bytes(), plaintext) {
		t.Error("ciphertext contains plaintext — encryption did not work")
	}

	// Decrypt
	decReader, err := Decrypt(&ciphertext, []age.Identity{identity})
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	decrypted, err := io.ReadAll(decReader)
	if err != nil {
		t.Fatalf("ReadAll from decryptor failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted data does not match plaintext:\nwant %q\ngot  %q", plaintext, decrypted)
	}
}

func TestEncryptStreamingRoundTrip(t *testing.T) {
	identity, recipient := generateTestIdentity(t)

	// Use a larger payload to exercise the streaming chunked encryption.
	plaintext := bytes.Repeat([]byte("ABCDEF123456"), 10000) // ~120 KB

	// Encrypt to a buffer
	var ciphertext bytes.Buffer
	encWriter, err := Encrypt(&ciphertext, []age.Recipient{recipient})
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := encWriter.Write(plaintext); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := encWriter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Decrypt from the buffer
	decReader, err := Decrypt(&ciphertext, []age.Identity{identity})
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	decrypted, err := io.ReadAll(decReader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("streaming round-trip mismatch: got %d bytes, want %d", len(decrypted), len(plaintext))
	}
}

func TestParseRecipients(t *testing.T) {
	_, recipient := generateTestIdentity(t)
	recipientStr := recipient.String()

	parsed, err := ParseRecipients(recipientStr)
	if err != nil {
		t.Fatalf("ParseRecipients failed: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(parsed))
	}

	// Test multiple recipients separated by newlines
	_, recipient2 := generateTestIdentity(t)
	parsed, err = ParseRecipients(recipientStr + "\n" + recipient2.String())
	if err != nil {
		t.Fatalf("ParseRecipients with newline-separated keys failed: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(parsed))
	}

	// Test invalid key
	_, err = ParseRecipients("not-a-valid-key")
	if err == nil {
		t.Error("expected error for invalid recipient key")
	}

	// Test empty input
	_, err = ParseRecipients("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseIdentity(t *testing.T) {
	identity, _ := generateTestIdentity(t)
	identityStr := identity.String()

	parsed, err := ParseIdentity(identityStr)
	if err != nil {
		t.Fatalf("ParseIdentity failed: %v", err)
	}

	// Round-trip with the parsed identity: encrypt to the original identity's
	// recipient, then decrypt with the parsed identity.
	var ciphertext bytes.Buffer
	encWriter, err := Encrypt(&ciphertext, []age.Recipient{identity.Recipient()})
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := encWriter.Write([]byte("test")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := encWriter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	decReader, err := Decrypt(&ciphertext, []age.Identity{parsed})
	if err != nil {
		t.Fatalf("Decrypt with parsed identity failed: %v", err)
	}
	decrypted, err := io.ReadAll(decReader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(decrypted) != "test" {
		t.Errorf("expected 'test', got %q", string(decrypted))
	}

	// Test invalid identity
	_, err = ParseIdentity("not-a-valid-identity")
	if err == nil {
		t.Error("expected error for invalid identity")
	}
}

func TestMultipleRecipients(t *testing.T) {
	id1, rec1 := generateTestIdentity(t)
	id2, rec2 := generateTestIdentity(t)
	id3, rec3 := generateTestIdentity(t)

	plaintext := []byte("encrypted for multiple recipients")

	// Encrypt to all three recipients
	var ciphertext bytes.Buffer
	encWriter, err := Encrypt(&ciphertext, []age.Recipient{rec1, rec2, rec3})
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := encWriter.Write(plaintext); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := encWriter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Each identity should be able to decrypt independently
	for i, id := range []age.Identity{id1, id2, id3} {
		decReader, err := Decrypt(bytes.NewReader(ciphertext.Bytes()), []age.Identity{id})
		if err != nil {
			t.Fatalf("Decrypt with identity %d failed: %v", i, err)
		}
		decrypted, err := io.ReadAll(decReader)
		if err != nil {
			t.Fatalf("ReadAll with identity %d failed: %v", i, err)
		}
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("identity %d: got %q, want %q", i, decrypted, plaintext)
		}
	}
}
