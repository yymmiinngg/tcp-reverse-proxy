package wan

import (
	"net"
	"os"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	"time"
)

type BindServer struct {
	bindAddress *net.TCPAddr
	ioTimeout   int
	handshake   *core.Handshaker
	heartbeat   int
	log         *logger.Logger
}

func StartBindServer(
	serverAddress *net.TCPAddr,
	ioTimeout int,
	handshakeKey string,
	heartbeat int,
	log *logger.Logger,
) {

	// 实例化
	it := &BindServer{
		bindAddress: serverAddress,
		ioTimeout:   ioTimeout,
		handshake:   core.MakeHandshaker(handshakeKey),
		heartbeat:   heartbeat,
		log:         log,
	}

	// 监听服务端口
	server, err := net.Listen("tcp", it.bindAddress.AddrPort().String())
	if err != nil {
		it.log.Error(err, "listen bind server error")
		os.Exit(1)
		return
	}
	defer server.Close()
	it.log.Info("start bind server at", it.bindAddress.AddrPort().String())

	// 处理请求
	for {
		bindConn, err := server.Accept()
		if err != nil {
			it.log.Debug("accept bind connection error:", err.Error())
			break
		}
		it.log.Debug("get a bind connection", bindConn.LocalAddr().String(), "<-", bindConn.RemoteAddr().String())
		go it.HandlBindConn(bindConn)
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

	// 启动转发服务
	relayServer := StartRelayServer(it.bindAddress.IP.String(), bindRequest.OpenPort, it.ioTimeout, it.log)
	if relayServer == nil {
		return
	}
	defer relayServer.Close()

	// 转发服务的地址
	relayAddr, err := net.ResolveTCPAddr("tcp", relayServer.relayListener.Addr().String())
	if err != nil {
		it.log.Debug("resolve relay address error:", err.Error())
		return
	}

	// 响应绑定连接
	err = core.WriteAny(bindConn, &core.BindResponse{
		Response:     core.Response{Message: "success"},
		ClientName:   bindRequest.ClientName,
		RelayPort:    relayAddr.Port, // 这里传端口是为了避免回传内网地址
		HandshakeKey: relayServer.handshaker.UserKey,
		Heartbeat:    it.heartbeat,
	})
	if err != nil {
		it.log.Debug("response bind connection error:", err.Error())
		return
	}

	// 长连接，断开则关闭整个转发链路
	go func() {
		defer bindConn.Close()
		for {
			_, err := bindConn.Write([]byte("heartbeat"))
			if err != nil {
				it.log.Debug("write heartbeat error:", err.Error())
				break
			}
			it.log.Debug("heartbeat")
			time.Sleep(time.Duration(it.heartbeat) * time.Second)
		}
	}()
	func() {
		defer bindConn.Close()
		buff := make([]byte, 32)
		for {
			bindConn.SetReadDeadline(time.Now().Add(time.Duration(it.heartbeat+it.ioTimeout) * time.Second))
			_, err := bindConn.Read(buff)
			if err != nil {
				it.log.Debug("read heartbeat error:", err.Error())
				break
			}
		}
	}()
}
