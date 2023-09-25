package lan

import (
	"fmt"
	"net"
	"tcp-tunnel/logger"
	nets "tcp-tunnel/net"
	"time"

	"github.com/yymmiinngg/goargs"
)

func Start(argsArr []string, log *logger.Logger) {
	template := `
	Usage: {{COMMAND}} CLIENT {{OPTION}}

	+ -l, --local-relay-address   # Mapped Local TCP Address for the Application, (Format: ip:port,
	#                               Default: 127.0.0.1:80)
	* -s, --server-opened-address # Listen on a port for Client binding (Format: ip:port)
	+ -k, --opened-key            # Handshake Key, Preventing Unauthorized Use of WAN Port

	? -h, --help                  # Show Help and Exit
	`

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 定义变量
	var localOpenAddress string
	var serverOpenedAddress string
	var openedKey string
	var connectTimeout int

	// 绑定变量
	args.StringOption("-l", &localOpenAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverOpenedAddress, "")
	args.IntOption("-c", &connectTimeout, 10)
	args.StringOption("-k", &openedKey, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.HasItem("-h", "--help") {
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
	serverAddr, err := net.ResolveTCPAddr("tcp", serverOpenedAddress)
	if err != nil {
		fmt.Println("resolve server address error:", err.Error())
		return
	}

	// 提取tcp地址
	applicationAddr, err := net.ResolveTCPAddr("tcp", localOpenAddress)
	if err != nil {
		fmt.Println("resolve server address error:", err.Error())
		return
	}

	// 默认与应用的端口一致

	localOpenListener, err := net.Listen("tcp", applicationAddr.AddrPort().String())
	if err != nil {
		fmt.Println("listen local open address error:", err.Error())
		return
	}

	for {
		localConn, err := localOpenListener.Accept()
		if err != nil {
			break
		}
		client := MakeClient(*serverAddr, connectTimeout, log)
		go client.handleLocalConn(localConn)
	}
}

type Client struct {
	serverAddr     net.TCPAddr
	connectTimeout int
	log            *logger.Logger
}

func MakeClient(serverAddr net.TCPAddr, connectTimeout int, log *logger.Logger) *Client {
	return &Client{
		serverAddr:     serverAddr,
		connectTimeout: connectTimeout,
		log:            log,
	}
}

func (it *Client) handleLocalConn(localConn net.Conn) {
	defer localConn.Close()
	serverConn, err := net.DialTimeout("tcp", it.serverAddr.AddrPort().String(), time.Duration(it.connectTimeout)*time.Second)
	if err != nil {
		it.log.Debug("connect to server open port error", err.Error())
	}
	defer func() {
		serverConn.Close()
		it.log.Debug("break", localConn.RemoteAddr().String(), "</>", serverConn.RemoteAddr().String())
	}()
	it.log.Debug("relay", localConn.RemoteAddr().String(), "<->", serverConn.RemoteAddr().String())
	nets.Relay(localConn, serverConn, 0)
}
