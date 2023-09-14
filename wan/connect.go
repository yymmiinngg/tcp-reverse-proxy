package wan

import (
	"net"
	"os"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"
)

type ServerInfo struct {
	serverAddress      string
	applicationAddress string
	handshaker         *core.Handshaker
	ioTimeout          int
	lanConns           chan net.Conn
	log                *logger.Logger
}

func MakeServerInfo(serverAddress, applicationAddress, handshakerKey string, ioTimeout int, log *logger.Logger) *ServerInfo {
	return &ServerInfo{
		serverAddress:      serverAddress,
		applicationAddress: applicationAddress,
		handshaker:         core.MakeHandshaker(handshakerKey),
		ioTimeout:          ioTimeout,
		lanConns:           make(chan net.Conn, 1024),
		log:                log,
	}
}

// 局域网的连接
func (it *ServerInfo) StartServer() {

	// 启动服务端监听
	go func() {
		server, err := net.Listen("tcp", it.serverAddress)
		if err != nil {
			it.log.Error(err, "listen server port error")
			os.Exit(1)
		}
		defer server.Close()
		it.log.Info("start server port:", it.serverAddress)
		for {
			lanConn, err := server.Accept()
			if err != nil {
				it.log.Error(err, "accept server connection error")
				time.Sleep(1000)
				continue
			}
			it.log.Debug("get a lan connection", lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
			it.lanConns <- lanConn
		}
	}()

	// 启动应用端监听
	server, err := net.Listen("tcp", it.applicationAddress)
	if err != nil {
		it.log.Error(err, "listen application port error")
		os.Exit(1)
	}
	defer server.Close()
	it.log.Info("start application port:", it.applicationAddress)
	for {
		clientConn, err := server.Accept()
		if err != nil {
			it.log.Error(err, "accept application connection error")
			time.Sleep(1000)
			continue
		}
		it.log.Debug("get a application connection", clientConn.LocalAddr().String(), "<-", clientConn.RemoteAddr().String())
		// 处理客户端连接
		go it.handlConn(clientConn, <-it.lanConns)
	}
}

func (it *ServerInfo) handlConn(clientConn, lanConn net.Conn) {
	defer func() {
		clientConn.Close()
		lanConn.Close()
		it.log.Debug("break", clientConn.RemoteAddr().String(), "</>", lanConn.RemoteAddr().String())
	}()

	// 发送握手指令
	handshake := it.handshaker.MakeHandshake()
	lanConn.Write(handshake[:])

	//  转发
	it.log.Debug("relay", clientConn.RemoteAddr().String(), "<->", lanConn.RemoteAddr().String())
	nets.Relay(clientConn, lanConn, it.ioTimeout)
}
