package lan

import (
	"fmt"
	"io"
	"net"
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"
)

type ServerConnectionPoolInfo struct {
	connectTimeout     int
	ioTimeout          int
	maxReadyConnect    int
	readyLock          sync.Locker
	readyConnect       int // 准备的连接
	relayAddress       string
	applicationAddress string
	handshaker         *core.Handshaker
	log                *logger.Logger
}

func MakeServerConnectionPoolInfo(serverAddress, applicationAddress, handshakerKey string, maxReadyConnect int, connectTimeout, ioTimeout int, log *logger.Logger) *ServerConnectionPoolInfo {
	return &ServerConnectionPoolInfo{
		connectTimeout:     connectTimeout,
		ioTimeout:          ioTimeout,
		maxReadyConnect:    maxReadyConnect,
		readyLock:          &sync.Mutex{},
		readyConnect:       0,
		relayAddress:       serverAddress,
		applicationAddress: applicationAddress,
		handshaker:         core.MakeHandshaker(handshakerKey),
		log:                log,
	}
}

func (it *ServerConnectionPoolInfo) addReady() {
	it.readyLock.Lock()
	defer it.readyLock.Unlock()
	it.readyConnect++
}

func (it *ServerConnectionPoolInfo) subReady() {
	it.readyLock.Lock()
	defer it.readyLock.Unlock()
	it.readyConnect--
}

func (it *ServerConnectionPoolInfo) GetServerConnect() {
	var lanConn net.Conn
	var errCount = 0
	for {
		// 准备连接已满，等待
		if it.readyConnect >= it.maxReadyConnect {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// 连接服务端
		var err error
		lanConn, err = net.DialTimeout("tcp", it.relayAddress, time.Duration(it.connectTimeout)*time.Second)
		if err != nil { // 连接失败
			errCount++
			it.log.Error(err, "connect to server error", fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
			if errCount <= 3 {
				time.Sleep(100 * time.Millisecond)
			} else if errCount <= 8 {
				time.Sleep(1000 * time.Millisecond)
			} else {
				time.Sleep(5000 * time.Millisecond)
			}
			// 去重试
			continue
		}
		// 连接成功
		it.log.Debug("connect to server", lanConn.LocalAddr().String(), "->", lanConn.RemoteAddr().String(), fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
		break
	}
	it.addReady()
	_serverConnectionBundle := serverConnectionBundle{lanConn: lanConn, serverConnectionPoolInfo: it}
	go _serverConnectionBundle.handConn()
}

//// ----------------------------------------------------------

type serverConnectionBundle struct {
	lanConn                  net.Conn
	serverConnectionPoolInfo *ServerConnectionPoolInfo
}

// 处理连接
func (it *serverConnectionBundle) handConn() {
	defer it.lanConn.Close()

	// 处理远程的握手
	var buff = make([]byte, core.HandshakeDataLength)
	_, err := io.ReadFull(it.lanConn, buff)
	if err != nil {
		it.serverConnectionPoolInfo.log.Debug("read from server error: " + err.Error())
		it.serverConnectionPoolInfo.subReady()
		return
	}
	if !it.serverConnectionPoolInfo.handshaker.CheckHandshake([64]byte(buff)) {
		it.serverConnectionPoolInfo.log.Debug("handshake fail")
		it.serverConnectionPoolInfo.subReady()
		return
	}
	// 原文响应
	it.lanConn.Write(buff)

	// 开始转发
	it.startRelay()
}

func (it *serverConnectionBundle) startRelay() {
	it.serverConnectionPoolInfo.subReady()
	// 请求目的服务器
	remoteConn, err := net.DialTimeout("tcp", it.serverConnectionPoolInfo.applicationAddress, time.Duration(it.serverConnectionPoolInfo.connectTimeout)*time.Second)
	if err != nil {
		it.serverConnectionPoolInfo.log.Debug("connect to application error: " + err.Error())
		return
	}
	it.serverConnectionPoolInfo.log.Debug("connect to application", remoteConn.LocalAddr().String(), "->", remoteConn.RemoteAddr().String())

	// 退出转发
	defer func() {
		remoteConn.Close()
		it.serverConnectionPoolInfo.log.Debug("break", it.lanConn.LocalAddr().String(), "</>", remoteConn.LocalAddr().String())
	}()

	// 转发
	it.serverConnectionPoolInfo.log.Debug("relay", it.lanConn.LocalAddr().String(), "<->", remoteConn.LocalAddr().String())
	nets.Relay(it.lanConn, remoteConn, it.serverConnectionPoolInfo.ioTimeout)
}
