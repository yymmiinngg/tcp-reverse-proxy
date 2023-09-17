package lan

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"
)

type Client struct {

	// 设置
	connectTimeout  int
	ioTimeout       int
	maxReadyConnect int

	// 地址
	serverAddress      *net.TCPAddr
	openPort           int
	applicationAddress *net.TCPAddr

	handshaker *core.Handshaker
	log        *logger.Logger

	// 待命连接计数
	readyConnect int
	readyLock    sync.Locker
}

func StartClient(
	serverAddress *net.TCPAddr,
	openPort int,
	applicationAddress *net.TCPAddr,
	handshakerKey string,
	maxReadyConnect int,
	connectTimeout,
	ioTimeout int,
	log *logger.Logger,
) {

	it := &Client{
		serverAddress:      serverAddress,
		connectTimeout:     connectTimeout,
		ioTimeout:          ioTimeout,
		maxReadyConnect:    maxReadyConnect,
		readyLock:          &sync.Mutex{},
		readyConnect:       0,
		openPort:           openPort,
		applicationAddress: applicationAddress,
		log:                log,
		handshaker:         core.MakeHandshaker(handshakerKey),
	}

	// 循环重试（直到绑定到服务端）
	for {

		// 是否关闭
		closed := false
		// 连接和绑定
		bindResponse := it.connectAndBind(func() { closed = true })
		if bindResponse == nil {
			time.Sleep(5 * time.Second) // 10秒后重试
			continue
		}

		// 运行循环器
		it.loopRelayConnect(bindResponse, &closed)
	}

}

func (it *Client) connectAndBind(bindCloseCallback func()) *core.BindResponse {

	// 连接绑定服务端
	bindConn, err := net.DialTimeout("tcp", it.serverAddress.AddrPort().String(), time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Error(err, "bind connect error")
		return nil
	}

	// 绑定连接的握手
	err = it.handshaker.RwHandshake(bindConn, it.ioTimeout)
	if err != nil {
		it.log.Debug("handshake error:", err.Error())
		bindConn.Close()
		return nil
	}

	// 发送绑定请求
	if err := core.WriteAny(bindConn, &core.BindRequest{
		Reqeust:    core.Reqeust{Action: "bind"},
		ClientName: bindConn.LocalAddr().String(),
		OpenPort:   it.openPort,
	}); err != nil {
		it.log.Error(err, "write bind request error")
		bindConn.Close()
		return nil
	}

	// 读取bind命令
	bindResponse := &core.BindResponse{}
	if err := core.ReadAny(bindConn, &bindResponse); err != nil {
		it.log.Error(err, "read bind response error")
		bindConn.Close()
		return nil
	}

	// 长连接，断开则关闭整个转发链路
	go func() {
		defer bindConn.Close()
		defer bindCloseCallback()
		for {
			_, err := bindConn.Write([]byte("heartbeat"))
			if err != nil {
				it.log.Debug("write heartbeat error:", err.Error())
				break
			}
			it.log.Debug("heartbeat")
			time.Sleep(time.Duration(bindResponse.Heartbeat) * time.Second)
		}
	}()
	go func() {
		defer bindConn.Close()
		defer bindCloseCallback()
		buff := make([]byte, 32)
		for {
			bindConn.SetReadDeadline(time.Now().Add(time.Duration(bindResponse.Heartbeat+it.ioTimeout) * time.Second))
			_, err := bindConn.Read(buff)
			if err != nil {
				it.log.Debug("read heartbeat error:", err.Error())
				break
			}
		}
	}()

	// 返回
	return bindResponse
}

//// 以下是转发连接的实现部分 /////////////////////////////////////////////////////////////////////////////////

type relayConnectionBundle struct {
	relayConn  net.Conn
	handshaker *core.Handshaker
}

// 循环尝试连接服务端转发端口
func (it *Client) loopRelayConnect(bindResponse *core.BindResponse, closed *bool) {
	var relayConn net.Conn
	var errCount = 0
	for !*closed {

		// 准备连接已满，等待
		if it.readyConnect >= it.maxReadyConnect {
			time.Sleep(80 * time.Millisecond)
			continue
		}

		// 连接服务端
		var err error
		relayConn, err = net.DialTimeout("tcp", net.JoinHostPort(it.serverAddress.IP.String(), strconv.Itoa(bindResponse.RelayPort)), time.Duration(it.connectTimeout)*time.Second)
		if err != nil { // 连接失败
			errCount++
			it.log.Error(err, "connect to relay server error", fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
			if errCount <= 3 {
				time.Sleep(100 * time.Millisecond)
			} else if errCount <= 8 {
				time.Sleep(1000 * time.Millisecond)
			} else {
				time.Sleep(5000 * time.Millisecond)
			}
			continue // 去重试
		}

		// 连接成功
		it.log.Debug("connect to relay server", relayConn.LocalAddr().String(), "->", relayConn.RemoteAddr().String(), fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))

		// 增待命连接数
		it.addReady()

		// 处理转发连接
		go it.handleRelayConnection(&relayConnectionBundle{relayConn: relayConn, handshaker: core.MakeHandshaker(bindResponse.HandshakeKey)})
	}

}

func (it *Client) handleRelayConnection(bundle *relayConnectionBundle) {

	defer bundle.relayConn.Close() // 关闭转发连接

	// 是否握手失败
	if func() bool {
		defer it.subReady() // 握手成功或失败后减少待命连接数
		err := bundle.handshaker.RwHandshake(bundle.relayConn, 0)
		if err != nil {
			it.log.Debug("handshake error:", err.Error())
			return true
		}
		return false
	}() {
		return
	}

	// 开始转发
	it.startRelay(bundle)
}

// 转发 relayAddress <-> applicationAddress
func (it *Client) startRelay(bundle *relayConnectionBundle) {

	// 请求应用服务器
	applicationConn, err := net.DialTimeout("tcp", it.applicationAddress.AddrPort().String(), time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Debug("connect to application error:", err.Error())
		return
	}
	it.log.Debug("connect to application", applicationConn.LocalAddr().String(), "->", applicationConn.RemoteAddr().String())

	// 退出转发
	defer func() {
		applicationConn.Close()
		it.log.Debug("break", bundle.relayConn.LocalAddr().String(), "</>", applicationConn.LocalAddr().String())
	}()

	// 转发
	it.log.Debug("relay", bundle.relayConn.LocalAddr().String(), "<->", applicationConn.LocalAddr().String())
	nets.Relay(bundle.relayConn, applicationConn, it.ioTimeout)
}

func (it *Client) addReady() {
	it.readyLock.Lock()
	defer it.readyLock.Unlock()
	it.readyConnect++
}

func (it *Client) subReady() {
	it.readyLock.Lock()
	defer it.readyLock.Unlock()
	it.readyConnect--
}
