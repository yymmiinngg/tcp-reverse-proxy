package wan

import (
	"net"
	"os"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	"time"
)

type Server struct {
	serverAddress string
	ioTimeout     int
	handshakeKey  string
	log           *logger.Logger
}

func MakeServer(
	serverPort string,
	ioTimeout int,
	handshakeKey string,
	log *logger.Logger,
) *Server {
	return &Server{
		serverAddress: serverPort,
		ioTimeout:     ioTimeout,
		handshakeKey:  handshakeKey,
		log:           log,
	}
}

func (it *Server) StartServer() {
	// 监听服务端口
	server, err := net.Listen("tcp", it.serverAddress)
	if err != nil {
		it.log.Error(err, "listen server port error")
		os.Exit(1)
	}
	defer server.Close()
	it.log.Info("listen server port:", it.serverAddress)

	// 处理请求
	for {
		bindConn, err := server.Accept()
		if err != nil {
			it.log.Error(err, "accept bind connection error")
			time.Sleep(1000)
			continue
		}
		it.log.Debug("get a bind connection", bindConn.LocalAddr().String(), "<-", bindConn.RemoteAddr().String())
		it.HandlBindConn(bindConn)
	}
}

// 处理请求
func (it *Server) HandlBindConn(bindConn net.Conn) {
	defer bindConn.Close()

	// 读取bind命令
	bindRequest := &core.BindRequest{}
	if err := core.ReadAny(bindConn, &bindRequest); err != nil {
		it.log.Error(err, "read bind request error")
		return
	}

	// 提取tcp地址
	tcpAddr, err := net.ResolveTCPAddr("tcp", bindConn.LocalAddr().String())
	if err != nil {
		it.log.Error(err, "get local addr error")
		return
	}

	// 启动转发服务
	relayServer, handshakeKey := MakeRelayServer(tcpAddr.IP.String(), bindRequest.OpenAddress, it.ioTimeout, it.log)
	relayAddress := relayServer.StartServer()
	if relayAddress == "" {
		return
	}
	defer relayServer.Close()

	// 响应绑定连接
	err = core.WriteAny(bindConn, &core.BindResponse{
		Response:     core.Response{Message: "success"},
		ClientName:   bindRequest.ClientName,
		RelayAddress: relayAddress,
		HandshakeKey: handshakeKey,
	})
	if err != nil {
		it.log.Error(err, "response bind connection error")
		return
	}

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
