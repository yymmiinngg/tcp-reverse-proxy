package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/exp/slices"
)

type Handshaker struct {
	UserKey string
}

const HandshakeDataLength = 64

func MakeHandshaker(key string) *Handshaker {
	return &Handshaker{
		UserKey: key,
	}
}

func (it *Handshaker) makeHandshake(data []byte) [HandshakeDataLength]byte {
	iv := RandBytes(32)
	tmp := []byte{}
	tmp = append(tmp, iv...)
	tmp = append(tmp, []byte(it.UserKey)...)
	tmp = append(tmp, data...)
	hash := getSha256(tmp)
	bytes := [HandshakeDataLength]byte(append(iv, hash...))
	return bytes
}

func (it *Handshaker) checkHandshake(handshakeData [HandshakeDataLength]byte, data []byte) bool {
	iv := handshakeData[:32]
	hash := handshakeData[32:]
	tmp := []byte{}
	tmp = append(tmp, iv...)
	tmp = append(tmp, []byte(it.UserKey)...)
	tmp = append(tmp, data...)
	hash2 := getSha256(tmp)
	return slices.Equal(hash, hash2)
}

// 处理连接
func (it *Handshaker) RwHandshake(conn net.Conn, ioTimeout int) error {
	// 处理远程的握手
	var handshakeData = make([]byte, HandshakeDataLength)
	if ioTimeout > 0 {
		defer conn.SetReadDeadline(time.Time{})
		conn.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
	}
	_, err := io.ReadFull(conn, handshakeData)
	if err != nil {
		return err
	}
	// 错误的握手数据
	if !it.checkHandshake([HandshakeDataLength]byte(handshakeData), nil) {
		return fmt.Errorf("handshake not match")
	}

	// 握手响应
	newHandshakeData := it.makeHandshake(handshakeData)
	if ioTimeout > 0 {
		defer conn.SetWriteDeadline(time.Time{})
		conn.SetWriteDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
	}
	_, err = conn.Write(newHandshakeData[:])
	return err
}

func (handshaker *Handshaker) WrHandshake(conn net.Conn, ioTimeout int) error {
	// 发送握手指令
	handshakeData := handshaker.makeHandshake([]byte{})
	if ioTimeout > 0 {
		defer conn.SetWriteDeadline(time.Time{})
		conn.SetWriteDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
	}
	_, err := conn.Write(handshakeData[:])
	if err != nil {
		return err
	}
	// 读握手响应
	newHandshakeData := make([]byte, HandshakeDataLength)
	if ioTimeout > 0 {
		defer conn.SetReadDeadline(time.Time{})
		conn.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
	}
	_, err = io.ReadFull(conn, newHandshakeData)
	if err != nil {
		return err
	}
	// 错误的响应
	if !handshaker.checkHandshake([HandshakeDataLength]byte(newHandshakeData), handshakeData[:]) {
		return fmt.Errorf("handshaker not match")
	}
	return nil
}

func getSha256(data []byte) []byte {
	m := sha256.New()
	defer m.Reset()
	m.Write(data)
	return m.Sum(nil)
}

func RandBytes(len int) (randbs []byte) {
	randBytes := make([]byte, len)
	_, err := io.ReadFull(rand.Reader, randBytes)
	if err != nil {
		fmt.Println(err)
	}
	return randBytes
}
