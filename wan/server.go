package wan

import (
	"net"
	"os"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
)

type BindServer struct {
	serverAddress string
	ioTimeout     int
	handshake     *core.Handshaker
	log           *logger.Logger
}

func StartBindServer(
	serverPort string,
	ioTimeout int,
	handshakeKey string,
	log *logger.Logger,
) {

	// 实例化
	it := &BindServer{
		serverAddress: serverPort,
		ioTimeout:     ioTimeout,
		handshake:     core.MakeHandshaker(handshakeKey),
		log:           log,
	}

	// 监听服务端口
	server, err := net.Listen("tcp", it.serverAddress)
	if err != nil {
		it.log.Error(err, "listen bind server error")
		os.Exit(1)
		return
	}
	defer server.Close()
	it.log.Info("start bind server at", it.serverAddress)

	// 处理请求
	for {
		bindConn, err := server.Accept()
		if err != nil {
			it.log.Debug("accept bind connection error:", err.Error())
			break
		}
		it.log.Debug("get a bind connection", bindConn.LocalAddr().String(), "<-", bindConn.RemoteAddr().String())
		it.HandlBindConn(bindConn)
	}
}

// 处理请求
func (it *BindServer) HandlBindConn(bindConn net.Conn) {
	defer bindConn.Close()

	// 通信前握手
	err := it.handshake.WrHandshake(bindConn, it.ioTimeout)
	if err != nil {
		it.log.Debug("handshaker error:", err.Error())
		return
	}

	// 读取bind命令
	bindRequest := &core.BindRequest{}
	if err := core.ReadAny(bindConn, &bindRequest); err != nil {
		it.log.Debug("read bind request error:", err.Error())
		return
	}

	// 提取tcp地址
	tcpAddr, err := net.ResolveTCPAddr("tcp", bindConn.LocalAddr().String())
	if err != nil {
		it.log.Debug("resolve bind address error:", err.Error())
		return
	}

	// 启动转发服务
	relayServer := StartRelayServer(tcpAddr.IP.String(), bindRequest.OpenAddress, it.ioTimeout, it.log)
	if relayServer == nil {
		return
	}
	defer relayServer.Close()

	// 响应绑定连接
	err = core.WriteAny(bindConn, &core.BindResponse{
		Response:     core.Response{Message: "success"},
		ClientName:   bindRequest.ClientName,
		RelayAddress: relayServer.relayListener.Addr().String(),
		HandshakeKey: relayServer.handshaker.UserKey,
	})
	if err != nil {
		it.log.Debug("response bind connection error:", err.Error())
		return
	}

	// 长连接，断开则关闭整个转发链路
	buff := make([]byte, 32)
	for {

		size, err := bindConn.Read(buff)
		if err != nil {
			it.log.Debug("read bind heartbeat error:", err.Error())
			break
		}

		_, err = bindConn.Write(buff[:size])
		if err != nil {
			it.log.Debug("write bind heartbeat error:", err.Error())
			break
		}

	}

}
