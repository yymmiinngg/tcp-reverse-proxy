package wan

import (
	"crypto/tls"
	"net"
	"os"
	"tcp-tunnel/config"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
)

type BindServer struct {
	bindAddress *net.TCPAddr
	ioTimeout   int
	handshake   *core.Handshaker
	log         *logger.Logger
}

func StartBindServer(
	bindAddress *net.TCPAddr,
	ioTimeout int,
	handshakeKey string,
	log *logger.Logger,
	tlsCertificate string,
	tlsPrivateKey string,
) {

	// 实例化
	it := &BindServer{
		bindAddress: bindAddress,
		ioTimeout:   ioTimeout,
		handshake:   core.MakeHandshaker(handshakeKey),
		log:         log,
	}

	if tlsCertificate != "" {
		// 证书配置
		cert, err := tls.LoadX509KeyPair(tlsCertificate, tlsPrivateKey)
		if err != nil {
			it.log.Error(err, "load x509 key pair error")
			return
		}
		// TLSs监听服务端口
		server, err := tls.Listen("tcp", it.bindAddress.AddrPort().String(), &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true})
		if err != nil {
			it.log.Error(err, "listen tls bind server error")
			return
		}
		it.log.Info("start tls bind server at", it.bindAddress.AddrPort().String())
		it.accept(server)
	} else {
		// TCP监听服务端口
		server, err := net.Listen("tcp", it.bindAddress.AddrPort().String())
		if err != nil {
			it.log.Error(err, "listen tcp bind server error")
			os.Exit(1)
			return
		}
		it.log.Info("start tcp bind server at", it.bindAddress.AddrPort().String())
		it.accept(server)
	}
}

func (it *BindServer) accept(server net.Listener) {
	defer server.Close()
	// 处理请求
	for {
		bindConn, err := server.Accept()
		if err != nil {
			it.log.Debug("accept bind connection error:", err.Error())
			break
		}
		it.log.Debug("get a bind connection", bindConn.LocalAddr().String(), "<-", bindConn.RemoteAddr().String())
		go it.handleBindConn(bindConn)
	}
}

// 处理请求
func (it *BindServer) handleBindConn(bindConn net.Conn) {
	defer bindConn.Close()

	// 通信前握手
	err := it.handshake.WrHandshake(bindConn, config.IOTimeout)
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
	})
	if err != nil {
		it.log.Debug("response bind connection error:", err.Error())
		return
	}

	// 长连接，断开则关闭代理
	func() {
		defer bindConn.Close()
		buff := make([]byte, 32)
		for {
			_, err := bindConn.Read(buff)
			if err != nil {
				it.log.Debug("break bind:", err.Error())
				break
			}
		}
	}()
}
