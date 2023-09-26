package core

import (
	"crypto/md5"
	"io"
	"sync"

	"golang.org/x/crypto/chacha20"
)

const (
	// KeySize256 KeySize256
	KeySize256 int = 32
)

// Cryptor 加解密接口
type Cryptor interface {
	Encrypt(src, dest []byte)
	Decrypt(src, dest []byte)
}

// toCryptKey toCryptKey
func toCryptKey(key string, keySize int) string {
	keyLen := len(key)
	cryptKey := make([]byte, keySize)
	for i := 0; i < keySize; i++ {
		cryptKey[i] = key[i%keyLen]
	}
	return string(cryptKey)
}

// ChaCha20Cryptor ChaCha20Cryptor
type ChaCha20Cryptor struct {
	enCipher *chacha20.Cipher
	deCipher *chacha20.Cipher
	enLocker sync.Mutex
	deLocker sync.Mutex
}

// Encrypt Encrypt
func (c *ChaCha20Cryptor) Encrypt(src, dest []byte) {
	c.enLocker.Lock()
	defer c.enLocker.Unlock()
	c.enCipher.XORKeyStream(dest, src)
}

// Decrypt Decrypt
func (c *ChaCha20Cryptor) Decrypt(src, dest []byte) {
	c.deLocker.Lock()
	defer c.deLocker.Unlock()
	c.deCipher.XORKeyStream(dest, src)
}

// NewXChaCha20Crypto NewXChaCha20Crypto
func NewXChaCha20Crypto(key string) (*ChaCha20Cryptor, error) {
	realKey := toCryptKey(key, KeySize256)
	return newChaCha20Crypto(realKey)
}

// NewChaCha20Crypto NewChaCha20Crypto
func newChaCha20Crypto(key string) (*ChaCha20Cryptor, error) {
	var enCipher, deCipher *chacha20.Cipher
	var err error
	h := md5.New()
	io.WriteString(h, "2e8fvx6zbyf40ut"+key)
	iv := h.Sum(nil)[:12]
	enCipher, err = chacha20.NewUnauthenticatedCipher([]byte(key), iv)
	if err != nil {
		return nil, err
	}
	deCipher, err = chacha20.NewUnauthenticatedCipher([]byte(key), iv)
	if err != nil {
		return nil, err
	}
	cryptor := &ChaCha20Cryptor{
		enCipher: enCipher,
		deCipher: deCipher,
	}
	return cryptor, nil
}
