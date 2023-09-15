package lan

import (
	"fmt"
	"io"
	"net"
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
	serverAddress      string
	openAddress        string
	applicationAddress string

	log *logger.Logger

	// 待命连接计数
	readyConnect int
	readyLock    sync.Locker

	binded bool
}

func StartClient(
	serverAddress,
	openAddress,
	applicationAddress,
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
		openAddress:        openAddress,
		applicationAddress: applicationAddress,
		log:                log,
		binded:             false,
	}

	// 循环重试（直到绑定到服务端）
	for {

		// 连接和绑定
		bindResponse := it.connectAndBind()
		if bindResponse == nil {
			time.Sleep(10 * time.Second) // 10秒后重试
			continue
		}

		// 运行循环器
		it.loopRelayConnect(bindResponse)
	}

}

func (it *Client) connectAndBind() *core.BindResponse {
	// 未绑定
	it.binded = false

	// 连接绑定服务端
	bindConn, err := net.DialTimeout("tcp", it.serverAddress, time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Error(err, "bind connect error")
		return nil
	}

	// 发送绑定请求
	if err := core.WriteAny(bindConn, &core.BindRequest{
		Reqeust:     core.Reqeust{Action: "bind"},
		ClientName:  bindConn.LocalAddr().String(),
		OpenAddress: it.openAddress,
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

	// 绑定成功
	it.binded = true
	// 长连接，断开则关闭整个转发链路
	go func() {
		defer func() {
			it.binded = false // 绑定结束
			bindConn.Close()
		}()

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
	}()

	return bindResponse
}

//// 以下是转发连接的实现部分 /////////////////////////////////////////////////////////////////////////////////

type relayConnectionBundle struct {
	relayConn  net.Conn
	handshaker *core.Handshaker
}

// 循环尝试连接服务端转发端口
func (it *Client) loopRelayConnect(bindResponse *core.BindResponse) {
	var relayConn net.Conn
	var errCount = 0
	for it.binded {

		// 准备连接已满，等待
		if it.readyConnect >= it.maxReadyConnect {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// 连接服务端
		var err error
		relayConn, err = net.DialTimeout("tcp", bindResponse.RelayAddress, time.Duration(it.connectTimeout)*time.Second)
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

	// 握手
	err := it.handshake(bundle)
	if err != nil {
		it.log.Debug("handshake error:", err.Error())
		return
	}

	// 开始转发
	it.startRelay(bundle)
}

// 处理连接
func (it *Client) handshake(bundle *relayConnectionBundle) error {

	defer it.subReady() // 握手成功或失败后减少待命连接数

	// 处理远程的握手
	var buff = make([]byte, core.HandshakeDataLength)
	_, err := io.ReadFull(bundle.relayConn, buff)
	if err != nil {
		return err
	}
	if !bundle.handshaker.CheckHandshake([64]byte(buff)) {
		return fmt.Errorf("handshake fail")
	}

	// 握手响应
	_, err = bundle.relayConn.Write(buff)
	return err
}

// 转发 relayAddress <-> applicationAddress
func (it *Client) startRelay(bundle *relayConnectionBundle) {

	// 请求应用服务器
	applicationConn, err := net.DialTimeout("tcp", it.applicationAddress, time.Duration(it.connectTimeout)*time.Second)
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
