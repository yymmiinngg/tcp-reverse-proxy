package wan

import (
	"net"
	"tcp-tunnel/config"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
)

type ServerInfo struct {
	ioTimeout          int
	serverAddress      string
	applicationAddress string
}

func MakeServerInfo(serverAddress, applicationAddress string, ioTimeout int) *ServerInfo {
	return &ServerInfo{
		ioTimeout:          ioTimeout,
		serverAddress:      serverAddress,
		applicationAddress: applicationAddress,
	}
}

// 局域网的连接
func (it *ServerInfo) StartServer() {

	// 启动服务端监听
	go func() {
		server, err := net.Listen("tcp", it.serverAddress)
		if err != nil {
			logger.Error(err, "listen server port error")
			return
		}
		defer server.Close()
		logger.Info("start server port...")
		for {
			lanConn, err := server.Accept()
			if err != nil {
				logger.Error(err, "accept server connection error")
				break
			}
			logger.Info("get a lan connection", lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
			lanConns <- lanConn
		}
	}()

	// 启动应用端监听
	server, err := net.Listen("tcp", it.applicationAddress)
	if err != nil {
		logger.Error(err, "listen application port error")
		return
	}
	defer server.Close()
	logger.Info("start client server...")
	for {
		clientConn, err := server.Accept()
		if err != nil {
			logger.Error(err, "accept application connection error")
			break
		}
		logger.Info("get a application connection", clientConn.LocalAddr().String(), "<-", clientConn.RemoteAddr().String())
		// 处理客户端连接
		go it.handlConn(clientConn, <-lanConns)
	}
}

func (it *ServerInfo) handlConn(clientConn, lanConn net.Conn) {
	defer func() {
		clientConn.Close()
		lanConn.Close()
		logger.Info("break", clientConn.RemoteAddr().String(), "</>", lanConn.RemoteAddr().String())
	}()

	// 发送连接指令
	lanConn.Write([]byte(config.PCMD_CONNECT))

	//  转发
	logger.Info("relay", clientConn.RemoteAddr().String(), "<->", lanConn.RemoteAddr().String())
	nets.Relay(clientConn, lanConn, it.ioTimeout)
}
