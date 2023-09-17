package wan

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"

	"github.com/google/uuid"
)

type RelayServer struct {
	relayBindHost string
	openPort      int
	handshaker    *core.Handshaker
	ioTimeout     int
	lanConns      chan net.Conn
	lanConnsLock  sync.Locker
	log           *logger.Logger
	// 两个重要的监听器
	relayListener       net.Listener
	applicationListener net.Listener
}

func (it *RelayServer) Close() {

	it.lanConnsLock.Lock()
	defer it.lanConnsLock.Unlock()

	// 关闭监听器
	it.relayListener.Close()
	it.applicationListener.Close()

	// 关闭所有待命连接
	for len(it.lanConns) > 0 {
		lanConn := <-it.lanConns
		lanConn.Close()
		it.log.Debug("close ready relay connection", lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
	}

	// TODO 关闭正在转发的连接
}

// 局域网的连接
func StartRelayServer(
	relayBindHost string,
	openPort int,
	ioTimeout int,
	log *logger.Logger,
) *RelayServer {

	// 随机一个密钥
	handshakerKey := uuid.New().String()
	it := &RelayServer{
		relayBindHost: relayBindHost,
		openPort:      openPort,
		handshaker:    core.MakeHandshaker(handshakerKey),
		ioTimeout:     ioTimeout,
		lanConnsLock:  &sync.Mutex{},
		lanConns:      make(chan net.Conn, 10),
		log:           log,
	}

	// 转发端口监听
	relayListener, err := net.Listen("tcp", net.JoinHostPort(it.relayBindHost, "0")) // 任意端口
	if err != nil {
		it.log.Error(err, "listen relay port error")
		return nil
	}

	// 应用端口监听
	openListener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(openPort)))
	if err != nil {
		it.log.Error(err, "listen application port error")
		relayListener.Close() // 关闭转发监听
		return nil
	}

	// 保存
	it.relayListener = relayListener
	it.applicationListener = openListener

	// 处理转发连接
	go func() {
		it.log.Info("start relay port:", relayListener.Addr().String())
		for {
			lanConn, err := relayListener.Accept()
			if err != nil {
				it.log.Debug("accept relay connection error: " + err.Error())
				break
			}
			it.log.Debug("get a relay connection", strconv.Itoa(len(it.lanConns)+1), lanConn.LocalAddr().String(), "<-", lanConn.RemoteAddr().String())
			it.lanConnsLock.Lock()
			it.lanConns <- lanConn // 连接放入待命队列
			it.lanConnsLock.Unlock()
		}
	}()

	// 处理应用连接
	go func() {
		it.log.Info("start application port:", strconv.Itoa(it.openPort))
		for {
			clientConn, err := openListener.Accept()
			if err != nil {
				it.log.Debug("accept client connection error: " + err.Error())
				break
			}
			it.log.Debug("get a client connection", clientConn.LocalAddr().String(), "<-", clientConn.RemoteAddr().String())
			// 处理客户端连接
			it.handlClientConn(clientConn)
		}
	}()

	return it
}

// 处理客户端的应用请求
func (it *RelayServer) handlClientConn(clientConn net.Conn) {
	lanConn, err := it.takeRelayConn()
	if err != nil {
		it.log.Debug("take a relay connection error: " + err.Error())
		clientConn.Close()
		return
	}
	// 转发
	go it.relay(clientConn, lanConn)
}

func (it *RelayServer) takeRelayConn() (net.Conn, error) {
	startTime := time.Now()
	// 获得现有或等待连接
	for {

		// 无连接则等待
		aliveCount := len(it.lanConns)
		if aliveCount == 0 {
			// 等待连接超时
			if time.Since(startTime) >= time.Duration(it.ioTimeout) {
				return nil, fmt.Errorf("wait relay connection timeout")
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// 获得lan端的连接
		it.lanConnsLock.Lock()
		lanConn := <-it.lanConns
		it.lanConnsLock.Unlock()

		// 通信前握手
		err := it.handshaker.WrHandshake(lanConn, it.ioTimeout)
		if err != nil {
			lanConn.Close()
			it.log.Debug("handshaker error:", err.Error())
			continue
		}

		// 返回可用连接
		return lanConn, nil
	}
}

func (it *RelayServer) relay(clientConn, lanConn net.Conn) {
	defer func() {
		clientConn.Close()
		lanConn.Close()
		it.log.Debug("break", clientConn.RemoteAddr().String(), "</>", lanConn.RemoteAddr().String())
	}()

	//  转发
	it.log.Debug("relay", clientConn.RemoteAddr().String(), "<->", lanConn.RemoteAddr().String())
	nets.Relay(clientConn, lanConn, it.ioTimeout)
}
