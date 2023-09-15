package lan

import (
	"fmt"
	"net"
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	"time"
)

type Client struct {
	connectTimeout     int
	ioTimeout          int
	maxReadyConnect    int
	readyLock          sync.Locker
	readyConnect       int // 准备的连接
	serverAddress      string
	openAddress        string
	applicationAddress string
	handshaker         *core.Handshaker
	log                *logger.Logger
}

func MakeClient(serverAddress, openAddress, applicationAddress, handshakerKey string, maxReadyConnect int, connectTimeout, ioTimeout int, log *logger.Logger) *Client {
	return &Client{
		serverAddress:      serverAddress,
		connectTimeout:     connectTimeout,
		ioTimeout:          ioTimeout,
		maxReadyConnect:    maxReadyConnect,
		readyLock:          &sync.Mutex{},
		readyConnect:       0,
		openAddress:        openAddress,
		applicationAddress: applicationAddress,
		handshaker:         core.MakeHandshaker(handshakerKey),
		log:                log,
	}
}

func (it *Client) StartClient() {
	fmt.Println(it.serverAddress)
	// 连接绑定服务端
	bindConn, err := net.DialTimeout("tcp", it.serverAddress, time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Error(err, "bind connect error")
		return
	}

	// 发送绑定请求s
	core.WriteAny(bindConn, &core.BindRequest{
		Reqeust:     core.Reqeust{Action: "bind"},
		ClientName:  it.openAddress,
		OpenAddress: it.openAddress,
	})

	// 读取bind命令
	bindResponse := &core.BindResponse{}
	if err := core.ReadAny(bindConn, &bindResponse); err != nil {
		it.log.Error(err, "read bind response error")
		return
	}

	// 局域网的连接
	go func() {
		serverConnection := MakeServerConnectionPoolInfo(
			bindResponse.RelayAddress,
			it.applicationAddress,
			bindResponse.HandshakeKey,
			it.maxReadyConnect,
			it.connectTimeout,
			it.ioTimeout,
			it.log,
		)
		for {
			serverConnection.GetServerConnect()
		}
	}()

	// 长连接，断开则关闭整个转发链路
	buff := make([]byte, 32)
	for {

		size, err := bindConn.Read(buff)
		if err != nil {
			it.log.Error(err, "read hold connect error")
			break
		}

		_, err = bindConn.Write(buff[:size])
		if err != nil {
			it.log.Error(err, "write hold connect error")
			break
		}

	}

}
