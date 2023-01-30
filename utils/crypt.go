package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Cryptography struct {
	key []byte
}

func NewCryptography(key string) (*Cryptography, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("please provide secret key")
	}
	return &Cryptography{key: []byte(key)}, nil
}

func (c *Cryptography) Encrypt(text string) (string, error) {
	ci, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(ci)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(text), nil)), nil
}

func (c *Cryptography) Decrypt(cypher string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return "", err
	}
	ci, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(ci)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}

func PaymentPlainText(id uint64) string {
	return fmt.Sprintf("payment:%d", id)
}
