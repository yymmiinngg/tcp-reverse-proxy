package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/exp/slices"
)

type Handshaker struct {
	staticKey string
	userKey   string
}

const HandshakeDataLength = 64

func MakeHandshaker(key string) *Handshaker {
	return &Handshaker{
		staticKey: "Gqmmh82CgEsRVQWz",
		userKey:   key,
	}
}

func (it *Handshaker) MakeHandshake() [HandshakeDataLength]byte {
	data := randBytes(32)
	tmp := []byte{}
	tmp = append(tmp, data...)
	tmp = append(tmp, []byte(it.staticKey)...)
	tmp = append(tmp, []byte(it.userKey)...)
	hash := getSha256(tmp)
	bytes := [HandshakeDataLength]byte(append(data, hash...))
	return bytes
}

func (it *Handshaker) CheckHandshake(handshakeData [HandshakeDataLength]byte) bool {
	data := handshakeData[:32]
	hash := handshakeData[32:]
	tmp := []byte{}
	tmp = append(tmp, data...)
	tmp = append(tmp, []byte(it.staticKey)...)
	tmp = append(tmp, []byte(it.userKey)...)
	hash2 := getSha256(tmp)
	return slices.Equal(hash, hash2)
}

func getSha256(data []byte) []byte {
	m := sha256.New()
	defer m.Reset()
	m.Write(data)
	return m.Sum(nil)
}

func randBytes(len int) (randbs []byte) {
	randBytes := make([]byte, len)
	_, err := io.ReadFull(rand.Reader, randBytes)
	if err != nil {
		fmt.Println(err)
	}
	return randBytes
}
