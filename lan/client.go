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
	}

	for {

		// 连接和绑定
		bindResponse := it.connectAndBind()
		if bindResponse == nil {
			time.Sleep(10 * time.Second)
		}

		// 运行循环器
		it.loop(bindResponse)

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

}

func (it *Client) connectAndBind() *core.BindResponse {
	// 连接绑定服务端
	bindConn, err := net.DialTimeout("tcp", it.serverAddress, time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Error(err, "bind connect error")
		return nil
	}
	defer bindConn.Close()

	// 发送绑定请求
	core.WriteAny(bindConn, &core.BindRequest{
		Reqeust:     core.Reqeust{Action: "bind"},
		ClientName:  bindConn.LocalAddr().String(),
		OpenAddress: it.openAddress,
	})

	// 读取bind命令
	bindResponse := &core.BindResponse{}
	if err := core.ReadAny(bindConn, &bindResponse); err != nil {
		it.log.Error(err, "read bind response error")
		return nil
	}
	return bindResponse
}

// 循环尝试连接服务端转发端口
func (it *Client) loop(bindResponse *core.BindResponse) {
	var relayConn net.Conn
	var errCount = 0
	for {
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
			// 去重试
			continue
		}
		// 连接成功
		it.log.Debug("connect to relay server", relayConn.LocalAddr().String(), "->", relayConn.RemoteAddr().String(), fmt.Sprintf("[%d/%d]", it.readyConnect+1, it.maxReadyConnect))
		break
	}

	// 处理转发连接
	go it.handleRelayConnection(&relayConnectionBundle{relayConn: relayConn, handshaker: core.MakeHandshaker(bindResponse.HandshakeKey)})
}

type relayConnectionBundle struct {
	relayConn  net.Conn
	handshaker *core.Handshaker
}

func (it *Client) handleRelayConnection(bundle *relayConnectionBundle) {
	// it := serverConnectionBundle{relayConn: lanConn, readyConnectionLooper: serverConnectionPoolInfo}
	defer bundle.relayConn.Close()

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

	it.addReady()       // 增待命连接数
	defer it.subReady() // 减少待命连接数

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

	// 请求目的服务器
	remoteConn, err := net.DialTimeout("tcp", it.applicationAddress, time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Debug("connect to application error:", err.Error())
		return
	}
	it.log.Debug("connect to application", remoteConn.LocalAddr().String(), "->", remoteConn.RemoteAddr().String())

	// 退出转发
	defer func() {
		remoteConn.Close()
		it.log.Debug("break", bundle.relayConn.LocalAddr().String(), "</>", remoteConn.LocalAddr().String())
	}()

	// 转发
	it.log.Debug("relay", bundle.relayConn.LocalAddr().String(), "<->", remoteConn.LocalAddr().String())
	nets.Relay(bundle.relayConn, remoteConn, it.ioTimeout)
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

func (it *Client) Close() {

}
