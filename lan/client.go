package lan

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync"
	"tcp-tunnel/config"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"
)

type Client struct {

	// 设置
	connectTimeout      int
	relayIoTimeout      int
	maxReadyConnect     int
	keepaliveConnection int
	// 地址
	serverAddress      *net.TCPAddr
	openPort           string
	applicationAddress *net.TCPAddr

	handshaker      *core.Handshaker
	log             *logger.Logger
	encryptKey      string
	relayHandshaker *core.Handshaker

	// 待命连接计数
	readyConnect int
	readyLock    sync.Locker
}

func StartClient(
	serverAddress *net.TCPAddr,
	openAddress string,
	applicationAddress *net.TCPAddr,
	handshakerKey string,
	maxReadyConnect int,
	connectTimeout,
	relayIoTimeout int,
	keepaliveConnection int,
	log *logger.Logger,
	useTls bool,
	encryptKey string,
) {

	it := &Client{
		serverAddress:       serverAddress,
		connectTimeout:      connectTimeout,
		relayIoTimeout:      relayIoTimeout,
		keepaliveConnection: keepaliveConnection,
		maxReadyConnect:     maxReadyConnect,
		readyLock:           &sync.Mutex{},
		readyConnect:        0,
		openPort:            openAddress,
		applicationAddress:  applicationAddress,
		log:                 log,
		handshaker:          core.MakeHandshaker(handshakerKey),
		encryptKey:          encryptKey,
		relayHandshaker: func() *core.Handshaker {
			if encryptKey != "" {
				return core.MakeHandshaker(encryptKey)
			}
			return nil
		}(),
	}

	// 循环重试（直到绑定到服务端）
	for {

		// 是否关闭
		closed := false
		// 连接和绑定
		bindResponse := it.connectAndBind(useTls, func() { closed = true })
		if bindResponse == nil {
			time.Sleep(5 * time.Second) // 重试
			continue
		}

		// 运行循环器
		it.loopRelayConnect(bindResponse, &closed)
	}

}

func (it *Client) connectAndBind(useTls bool, bindCloseCallback func()) *core.BindResponse {

	var bindConn net.Conn
	var err error

	// 连接绑定服务端
	if useTls {
		it.log.Debug("connect to tls bind server", it.serverAddress.AddrPort().String(), "-", it.openPort)
		d := &net.Dialer{Timeout: time.Duration(it.connectTimeout) * time.Second}
		bindConn, err = tls.DialWithDialer(d, "tcp", it.serverAddress.AddrPort().String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			it.log.Error(err, "tls bind connect error")
			return nil
		}
	} else {
		it.log.Debug("connect to tcp bind server", it.serverAddress.AddrPort().String())
		d := &net.Dialer{Timeout: time.Duration(it.connectTimeout) * time.Second}
		bindConn, err = d.Dial("tcp", it.serverAddress.AddrPort().String())
		if err != nil {
			it.log.Error(err, "tcp bind connect error")
			return nil
		}
	}

	// 绑定连接的握手
	err = it.handshaker.RwHandshake(bindConn, config.WaitTimeout)
	if err != nil {
		it.log.Debug("bind handshake error:", err.Error())
		bindConn.Close()
		return nil
	}

	// 发送绑定请求
	if err := core.WriteObject2Json(bindConn, &core.BindRequest{
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
	if err := core.ReadJson2Object(bindConn, &bindResponse); err != nil {
		it.log.Error(err, "read bind response error")
		bindConn.Close()
		return nil
	}

	// 长连接，断开则关闭代理
	go func() {
		defer bindConn.Close()
		defer bindCloseCallback()
		buff := make([]byte, 64)
		go func() {
			defer bindConn.Close()
			defer bindCloseCallback()
			for {
				time.Sleep(time.Duration(it.keepaliveConnection) * time.Second)
				_, err := bindConn.Write([]byte(time.Now().Local().String()))
				if err != nil {
					break
				}
			}
		}()
		for {
			size, err := bindConn.Read(buff)
			if err != nil {
				it.log.Debug("break bind:", err.Error())
				break
			}
			it.log.Debug("bind keepalive package:", string(buff[:size]))
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
			time.Sleep(100 * time.Millisecond)
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
		it.log.Debug("connect to relay server", relayConn.LocalAddr().String(), "->", relayConn.RemoteAddr().String(), fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect), "-", it.openPort)

		// 增待命连接数
		it.addReady()

		// 处理转发连接
		go it.handleRelayConnection(&relayConnectionBundle{relayConn: relayConn, handshaker: core.MakeHandshaker(bindResponse.HandshakeKey)})
	}

}

func (it *Client) handleRelayConnection(bundle *relayConnectionBundle) {
	// 关闭转发连接
	defer bundle.relayConn.Close()

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
	// 加解密处理器
	var cryptor core.Cryptor
	if it.encryptKey != "" {
		cryptor, err = core.NewXChaCha20Crypto(it.encryptKey)
		if err != nil {
			it.log.Debug("make cryptor error", err.Error())
			return
		}
		// 加密连接的握手
		err = it.relayHandshaker.WrHandshake(bundle.relayConn, config.WaitTimeout)
		if err != nil {
			it.log.Debug("relay handshake error:", err.Error())
			return
		}
	}
	nets.Relay(applicationConn, bundle.relayConn, it.relayIoTimeout, cryptor)
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
