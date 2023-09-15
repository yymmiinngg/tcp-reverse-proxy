package wan

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type RelayServer struct {
	relayBindHost      string
	applicationAddress string
	handshaker         *core.Handshaker
	ioTimeout          int
	lanConns           chan net.Conn
	lanConnsLock       sync.Locker
	log                *logger.Logger
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
	}

	// TODO 关闭正在转发的连接
}

// 局域网的连接
func StartRelayServer(
	relayBindHost,
	applicationAddress string,
	ioTimeout int,
	log *logger.Logger,
) *RelayServer {

	// 随机一个密钥
	handshakerKey := uuid.New().String()
	it := &RelayServer{
		relayBindHost:      relayBindHost,
		applicationAddress: applicationAddress,
		handshaker:         core.MakeHandshaker(handshakerKey),
		ioTimeout:          ioTimeout,
		lanConnsLock:       &sync.Mutex{},
		lanConns:           make(chan net.Conn, 10),
		log:                log,
	}

	// 转发端口监听
	relayListener, err := net.Listen("tcp", it.relayBindHost+":0")
	if err != nil {
		it.log.Error(err, "listen relay port error")
		return nil
	}

	// 应用端口监听
	applicationListener, err := net.Listen("tcp", it.applicationAddress)
	if err != nil {
		it.log.Error(err, "listen application port error")
		relayListener.Close() // 关闭转发监听
		return nil
	}

	// 保存
	it.relayListener = relayListener
	it.applicationListener = applicationListener

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
		it.log.Info("start application port:", it.applicationAddress)
		for {
			clientConn, err := applicationListener.Accept()
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
	lanConn, err := it.takeLanConn()
	if err != nil {
		it.log.Debug("waiting for connection error: " + err.Error())
		clientConn.Close()
		return
	}
	// 转发
	go it.relay(clientConn, lanConn)
}

func (it *RelayServer) takeLanConn() (net.Conn, error) {
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
		it.lanConnsLock.Lock()
		lanConn := <-it.lanConns
		it.lanConnsLock.Unlock()
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
