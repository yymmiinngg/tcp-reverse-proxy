package client

import (
	"fmt"
	"net"
	"tcp-tunnel/config"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"

	"github.com/yymmiinngg/goargs"
)

func Start(argsArr []string, log *logger.Logger) {
	template := `
	Usage: {{COMMAND}} CLIENT {{OPTION}}

	* -s, --server-relay-address  # Request to server relay port (Format: ip:port)
	+ -l, --local-relay-address   # Listen on a port for Client access (Format: ip:port)
	#                               (Default: 127.0.0.1:80)
	+ -e, --relay-encrypt-key     # Keep the relay-encrypt-key consistent with the LAN side,
	#                               if they are not the same, correct transmission will not
	#                               be possible
	+ -c, --connect-timeout       # Connection Timeout Duration (Unit: Seconds, Default: 10)
	? -H, --help                  # Show Help and Exit
	`

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 定义变量
	var localRelayAddress string
	var serverRelayAddress string
	var relayEncryptKey string
	var connectTimeout int

	// 绑定变量
	args.StringOption("-l", &localRelayAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverRelayAddress, "")
	args.StringOption("-e", &relayEncryptKey, "")
	args.IntOption("-c", &connectTimeout, 10)

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.HasItem("-H", "--help") {
		fmt.Println(args.Usage())
		return
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if connectTimeout == 0 {
		fmt.Println("The connection timeout duration cannot be less than 1")
		return
	}

	// 提取tcp地址
	serverRelayAddr, err := net.ResolveTCPAddr("tcp", serverRelayAddress)
	if err != nil {
		fmt.Println("resolve server relay address error:", err.Error())
		return
	}

	// 提取tcp地址
	localRelayAddr, err := net.ResolveTCPAddr("tcp", localRelayAddress)
	if err != nil {
		fmt.Println("resolve local relay address error:", err.Error())
		return
	}

	// 本地监听器
	localRelayListener, err := net.Listen("tcp", localRelayAddr.AddrPort().String())
	if err != nil {
		fmt.Println("listen local relay address error:", err.Error())
		return
	}
	log.Debug("listen local relay address", localRelayAddr.AddrPort().String())

	for {
		localConn, err := localRelayListener.Accept()
		if err != nil {
			break
		}
		client, err := MakeClient(*serverRelayAddr, connectTimeout, relayEncryptKey, log)
		if err != nil {
			log.Debug("connect to server opened port error", err.Error())
		}
		go client.handleLocalConn(localConn)
	}
}

type Client struct {
	serverAddr      net.TCPAddr
	connectTimeout  int
	log             *logger.Logger
	encryptKey      string
	relayHandshaker *core.Handshaker
}

func MakeClient(serverAddr net.TCPAddr, connectTimeout int, encryptKey string, log *logger.Logger) (*Client, error) {
	return &Client{
		serverAddr:     serverAddr,
		connectTimeout: connectTimeout,
		log:            log,
		encryptKey:     encryptKey,
		relayHandshaker: func() *core.Handshaker {
			if encryptKey != "" {
				return core.MakeHandshaker(encryptKey)
			}
			return nil
		}(),
	}, nil
}

func (it *Client) handleLocalConn(localConn net.Conn) {
	defer localConn.Close()
	serverConn, err := net.DialTimeout("tcp", it.serverAddr.AddrPort().String(), time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Debug("connect to server opened port error", err.Error())
		return
	}
	defer func() {
		serverConn.Close()
		it.log.Debug("break", localConn.RemoteAddr().String(), "</>", serverConn.RemoteAddr().String())
	}()

	// 开始转发
	it.log.Debug("relay", localConn.RemoteAddr().String(), "<->", serverConn.RemoteAddr().String())
	// 加解密处理器
	var cryptor core.Cryptor
	if it.encryptKey != "" {
		cryptor, err = core.NewXChaCha20Crypto(it.encryptKey)
		if err != nil {
			it.log.Debug("make cryptor error", err.Error())
			return
		}
		// 加密连接的握手
		err = it.relayHandshaker.RwHandshake(serverConn, config.WaitTimeout)
		if err != nil {
			it.log.Debug("relay handshake error:", err.Error())
			return
		}
	}
	nets.Relay(localConn, serverConn, 0, cryptor)
}
