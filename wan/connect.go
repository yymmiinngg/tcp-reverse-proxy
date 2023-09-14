package wan

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"

	"golang.org/x/exp/slices"
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
		lanConns:           make(chan net.Conn, 10),
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
			it.log.Debug("get a lan connection", strconv.Itoa(len(it.lanConns)+1), lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
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
		it.handlConn(clientConn)
	}
}

func (it *ServerInfo) handlConn(clientConn net.Conn) {
	lanConn, err := it.takeLanConn()
	if err != nil {
		it.log.Debug("waiting for connection error: " + err.Error())
		clientConn.Close()
		return
	}
	// 转发
	go it.relay(clientConn, lanConn)
}

func (it *ServerInfo) takeLanConn() (net.Conn, error) {
	startTime := time.Now()
	// 获得现有或等待连接
	for {
		aliveCount := len(it.lanConns)
		if aliveCount == 0 {
			// 等待连接超时
			if time.Since(startTime) >= time.Duration(it.ioTimeout) {
				return nil, fmt.Errorf("timeout")
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// 获得lan端的连接
		lanConn := <-it.lanConns
		// 发送握手指令
		handshake := it.handshaker.MakeHandshake()
		_, err := lanConn.Write(handshake[:])
		if err != nil {
			lanConn.Close()
			it.log.Debug("write handshaker data error: " + err.Error())
			continue
		}
		// 读握手响应
		buff := make([]byte, len(handshake))
		lanConn.SetReadDeadline(time.Now().Add(time.Duration(it.ioTimeout) * time.Second))
		_, err = io.ReadFull(lanConn, buff)
		if err != nil {
			lanConn.Close()
			it.log.Debug("read handshaker data error: " + err.Error())
			continue
		}
		// 错误的响应
		if !slices.Equal(buff, handshake[:]) {
			lanConn.Close()
			return nil, fmt.Errorf("handshaker fail")
		}
		return lanConn, nil
	}
}

func (it *ServerInfo) relay(clientConn, lanConn net.Conn) {
	defer func() {
		clientConn.Close()
		lanConn.Close()
		it.log.Debug("break", clientConn.RemoteAddr().String(), "</>", lanConn.RemoteAddr().String())
	}()

	//  转发
	it.log.Debug("relay", clientConn.RemoteAddr().String(), "<->", lanConn.RemoteAddr().String())
	nets.Relay(clientConn, lanConn, it.ioTimeout)
}
