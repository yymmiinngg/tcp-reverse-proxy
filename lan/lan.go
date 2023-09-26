package lan

import (
	"fmt"
	"net"
	"strconv"
	"tcp-tunnel/logger"

	"github.com/yymmiinngg/goargs"
)

func Start(argsArr []string, log *logger.Logger) {
	template := `
	Usage: {{COMMAND}} LAN {{OPTION}}

	+ -a, --application-address  # Mapped TCP Address for the Application, (Format: ip:port,
	#                              Default: 127.0.0.1:80)
	* -s, --server-bind-address  # Listen on a port for Client binding (Format: ip:port)
	+ -o, --open-address         # Instruct the server to open a port for relay traffic to
	#                              the client (Format: ip:port, Default is the same port of 
	#                              application-address, like ":port")
	
	+ -r, --ready-connection     # Ready Connection Count (Default: 5), Ready connections
	#                              help improve client connection speed. The quantity limit
	#                              is 1024.
	+ -c, --connect-timeout      # Connection Timeout Duration (Unit: Seconds, Default: 10)
	+ -i, --io-timeout           # Read/Write Timeout Duration in relaying (Unit: Seconds,
	#                              Default: 120)
    + -K, --keepalive            # Keepalive seconds, use to keep tcp connection always alive
	#                              (Default: 120)

	+ -k, --bind-handshake-key   # Handshake Key, Preventing Unauthorized Use of WAN Port
    + -e, --encrypt-key          # The key is used to encrypt the traffic. When the LAN side
	#                              uses the key, direct connections from client applications
	#                              to open ports on the WAN side will fail because the
	#                              traffic is encrypted, and the CLIENT side needs to decrypt
	#                              the traffic
	? -T, --tls                  # Use tls connect when WAN program used x509-certificate

	? -H, --help                 # Show Help and Exit
	`

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 定义变量
	var applicationAddress string
	var serverAddress string
	var openAddress string
	var bindHandshakeKey string
	var readyConnection, connectTimeout, relayIoTimeout int
	var tls bool
	var encryptKey string
	var keepaliveConnection int

	// 绑定变量
	args.StringOption("-a", &applicationAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverAddress, "")
	args.StringOption("-o", &openAddress, "")
	args.IntOption("-r", &readyConnection, 5)
	args.IntOption("-c", &connectTimeout, 10)
	args.IntOption("-i", &relayIoTimeout, 120)
	args.IntOption("-K", &keepaliveConnection, 120)
	args.StringOption("-k", &bindHandshakeKey, "")
	args.BoolOption("-T", &tls, false)
	args.StringOption("-e", &encryptKey, "")

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

	if readyConnection < 1 {
		fmt.Println("The minimum ready connection count is 1")
		return
	}

	if readyConnection > 1024 {
		fmt.Println("The maximum ready connection count is 1024")
		return
	}

	if connectTimeout == 0 {
		fmt.Println("The connection timeout duration cannot be less than 1")
		return
	}

	if relayIoTimeout == 0 {
		fmt.Println("The io timeout duration cannot be less than 1")
		return
	}

	// 提取tcp地址
	serverAddr, err := net.ResolveTCPAddr("tcp", serverAddress)
	if err != nil {
		fmt.Println("resolve server address error:", err.Error())
		return
	}

	// 提取tcp地址
	applicationAddr, err := net.ResolveTCPAddr("tcp", applicationAddress)
	if err != nil {
		fmt.Println("resolve server address error:", err.Error())
		return
	}

	// 默认与应用的端口一致
	if openAddress == "" {
		openAddress = ":" + strconv.Itoa(applicationAddr.Port)
	}

	StartClient(serverAddr,
		openAddress,
		applicationAddr,
		bindHandshakeKey,
		readyConnection,
		connectTimeout,
		relayIoTimeout,
		keepaliveConnection,
		log,
		tls,
		encryptKey,
	)
}
