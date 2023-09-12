package lan

import (
	"fmt"
	"io"
	"net"
	"sync"
	"tcp-tunnel/config"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"
)

var log = logger.Logger{Mode: "LAN"}

type ServerConnectionPoolInfo struct {
	connectTimeout     int
	ioTimeout          int
	maxReadyConnect    int
	readyLock          sync.Locker
	readyConnect       int // 准备的连接
	serverAddress      string
	applicationAddress string
}

func MakeServerConnectionPoolInfo(serverAddress, applicationAddress string, maxReadyConnect int, connectTimeout, ioTimeout int) *ServerConnectionPoolInfo {
	return &ServerConnectionPoolInfo{
		connectTimeout:     connectTimeout,
		ioTimeout:          ioTimeout,
		maxReadyConnect:    maxReadyConnect,
		readyLock:          &sync.Mutex{},
		readyConnect:       0,
		serverAddress:      serverAddress,
		applicationAddress: applicationAddress,
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
		lanConn, err = net.DialTimeout("tcp", it.serverAddress, time.Duration(it.connectTimeout)*time.Second)
		if err != nil {
			errCount++
			log.Error(err, "connect to server error", fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
			if errCount <= 3 {
				time.Sleep(100 * time.Millisecond)
			} else if errCount <= 8 {
				time.Sleep(1000 * time.Millisecond)
			} else {
				time.Sleep(5000 * time.Millisecond)
			}
			continue
		}
		log.Info("connect to server", lanConn.LocalAddr().String(), "->", lanConn.RemoteAddr().String(), fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
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

// 处理连接f
func (it *serverConnectionBundle) handConn() {

	defer it.lanConn.Close()

	// 处理远程的请求指令
	var buff = make([]byte, len(config.PCMD_CONNECT))
	size, err := io.ReadFull(it.lanConn, buff)
	if err != nil {
		log.Error(err, "read from server error")
		it.serverConnectionPoolInfo.subReady()
		return
	}
	if string(buff[:size]) != config.PCMD_CONNECT {
		log.Info("no command '" + config.PCMD_CONNECT + "' found")
		it.serverConnectionPoolInfo.subReady()
		return
	}

	// 开始转发
	it.startRelay()
}

func (it *serverConnectionBundle) startRelay() {
	it.serverConnectionPoolInfo.subReady()
	// 请求目的服务器
	remoteConn, err := net.DialTimeout("tcp", it.serverConnectionPoolInfo.applicationAddress, time.Duration(it.serverConnectionPoolInfo.connectTimeout)*time.Second)
	if err != nil {
		log.Error(err, "connect to application error")
		return
	}
	log.Info("connect to application", remoteConn.LocalAddr().String(), "->", remoteConn.RemoteAddr().String())

	// 退出转发
	defer func() {
		remoteConn.Close()
		log.Info("break", it.lanConn.LocalAddr().String(), "</>", remoteConn.LocalAddr().String())
	}()

	// 转发
	log.Info("relay", it.lanConn.LocalAddr().String(), "<->", remoteConn.LocalAddr().String())
	nets.Relay(it.lanConn, remoteConn, it.serverConnectionPoolInfo.ioTimeout)
}
