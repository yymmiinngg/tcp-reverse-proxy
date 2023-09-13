package wan

import (
	"net"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
)

var log = logger.Logger{Mode: "WAN"}

type ServerInfo struct {
	serverAddress      string
	applicationAddress string
	handshaker         *core.Handshaker
	ioTimeout          int
	lanConns           chan net.Conn
}

func MakeServerInfo(serverAddress, applicationAddress, handshakerKey string, ioTimeout int) *ServerInfo {
	return &ServerInfo{
		serverAddress:      serverAddress,
		applicationAddress: applicationAddress,
		handshaker:         core.MakeHandshaker(handshakerKey),
		ioTimeout:          ioTimeout,
		lanConns:           make(chan net.Conn, 1024),
	}
}

// 局域网的连接
func (it *ServerInfo) StartServer() {

	// 启动服务端监听
	go func() {
		server, err := net.Listen("tcp", it.serverAddress)
		if err != nil {
			log.Error(err, "listen server port error")
			return
		}
		defer server.Close()
		log.Info("start server port...")
		for {
			lanConn, err := server.Accept()
			if err != nil {
				log.Error(err, "accept server connection error")
				break
			}
			log.Info("get a lan connection", lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
			it.lanConns <- lanConn
		}
	}()

	// 启动应用端监听
	server, err := net.Listen("tcp", it.applicationAddress)
	if err != nil {
		log.Error(err, "listen application port error")
		return
	}
	defer server.Close()
	log.Info("start client server...")
	for {
		clientConn, err := server.Accept()
		if err != nil {
			log.Error(err, "accept application connection error")
			break
		}
		log.Info("get a application connection", clientConn.LocalAddr().String(), "<-", clientConn.RemoteAddr().String())
		// 处理客户端连接
		go it.handlConn(clientConn, <-it.lanConns)
	}
}

func (it *ServerInfo) handlConn(clientConn, lanConn net.Conn) {
	defer func() {
		clientConn.Close()
		lanConn.Close()
		log.Info("break", clientConn.RemoteAddr().String(), "</>", lanConn.RemoteAddr().String())
	}()

	// 发送握手指令
	handshake := it.handshaker.MakeHandshake()
	lanConn.Write(handshake[:])

	//  转发
	log.Info("relay", clientConn.RemoteAddr().String(), "<->", lanConn.RemoteAddr().String())
	nets.Relay(clientConn, lanConn, it.ioTimeout)
}
