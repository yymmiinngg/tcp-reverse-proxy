package core

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/chacha20"
)

const (
	// KeySizeAuto KeySizeAuto
	KeySizeAuto int = 0
	// KeySize128 KeySize128
	KeySize128 int = 16
	// KeySize192 KeySize192
	KeySize192 int = 24
	// KeySize256 KeySize256
	KeySize256 int = 32
)

// Cryptor 加解密接口
type Cryptor interface {
	Encrypt(src, dest []byte)
	Decrypt(src, dest []byte)
}

// ToCryptKey ToCryptKey
func ToCryptKey(key []byte, keySize int) []byte {
	if len(key) == 0 {
		key = []byte(" ")
	}
	keyLen := len(key)
	switch keySize {
	case KeySizeAuto:
		if keyLen < KeySize128 {
			keySize = KeySize128
		} else if keyLen < KeySize192 {
			keySize = KeySize192
		} else {
			keySize = KeySize256
		}
	case KeySize128, KeySize192, KeySize256:
	default:
		panic(fmt.Errorf("unsupported keysize %d", keySize))
	}
	cryptKey := make([]byte, keySize)
	for i := 0; i < keySize; i++ {
		cryptKey[i] = key[i%keyLen]
	}
	return cryptKey
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
func NewXChaCha20Crypto(key []byte) (*ChaCha20Cryptor, error) {
	realKey := ToCryptKey(key, KeySize256)
	return newChaCha20Crypto(realKey)
}

// NewChaCha20Crypto NewChaCha20Crypto
func newChaCha20Crypto(key []byte) (*ChaCha20Cryptor, error) {
	var enCipher, deCipher *chacha20.Cipher
	var err error
	enCipher, err = chacha20.NewUnauthenticatedCipher(key, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})
	if err != nil {
		return nil, err
	}
	deCipher, err = chacha20.NewUnauthenticatedCipher(key, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})
	if err != nil {
		return nil, err
	}
	cryptor := &ChaCha20Cryptor{
		enCipher: enCipher,
		deCipher: deCipher,
	}
	return cryptor, nil
}
