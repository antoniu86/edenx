package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// File format:
//   [4]  magic: EDNX
//   [1]  version: 0x01
//   [32] Argon2 salt
//   [12] AES-256-GCM nonce
//   [N]  ciphertext (AES-256-GCM encrypted content)

const (
	Magic          = "EDNX"
	Version        = byte(0x01)
	SaltSize       = 32
	NonceSize      = 12
	HeaderSize     = 4 + 1 + SaltSize + NonceSize // 49 bytes
)

// Argon2 parameters (deliberately conservative for key security)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // 256-bit key for AES-256
)

var (
	ErrInvalidMagic   = errors.New("not an .ednx encrypted file")
	ErrInvalidVersion = errors.New("unsupported .ednx format version")
	ErrDecryptFailed  = errors.New("decryption failed: wrong password or corrupted file")
)

// IsEncrypted checks if data starts with the EDNX magic header
func IsEncrypted(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == Magic
}

// Encrypt encrypts plaintext with the given password and returns the .ednx binary blob
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Derive key using Argon2id
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Assemble output: magic + version + salt + nonce + ciphertext
	out := make([]byte, 0, HeaderSize+len(ciphertext))
	out = append(out, []byte(Magic)...)
	out = append(out, Version)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	_ = binary.LittleEndian // imported for future use

	return out, nil
}

// Decrypt decrypts an .ednx binary blob with the given password
func Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidMagic
	}

	// Verify magic
	if string(data[:4]) != Magic {
		return nil, ErrInvalidMagic
	}

	// Verify version
	if data[4] != Version {
		return nil, ErrInvalidVersion
	}

	// Extract salt, nonce, ciphertext
	offset := 5
	salt := data[offset : offset+SaltSize]
	offset += SaltSize
	nonce := data[offset : offset+NonceSize]
	offset += NonceSize
	ciphertext := data[offset:]

	// Derive key
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}

	return plaintext, nil
}
