package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const prefix = "v1:"

var ErrKeyRequired = errors.New("encryption key is required")

type Box struct {
	key []byte
}

// New derives a stable AES-256 key from an operator-provided secret. The
// secret itself is never persisted by this package.
func New(operatorSecret string) (*Box, error) {
	if strings.TrimSpace(operatorSecret) == "" {
		return nil, ErrKeyRequired
	}
	digest := sha256.Sum256([]byte("axiops/credential/v1/" + operatorSecret))
	return &Box{key: digest[:]}, nil
}

func (b *Box) Encrypt(plaintext string) (string, error) {
	if b == nil || len(b.key) == 0 {
		return "", ErrKeyRequired
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (b *Box) Decrypt(value string) (string, error) {
	if b == nil || len(b.key) == 0 {
		return "", ErrKeyRequired
	}
	if !strings.HasPrefix(value, prefix) {
		return "", errors.New("unsupported credential ciphertext version")
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", fmt.Errorf("decode credential ciphertext: %w", err)
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("credential ciphertext is too short")
	}
	plaintext, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("credential ciphertext authentication failed")
	}
	return string(plaintext), nil
}
