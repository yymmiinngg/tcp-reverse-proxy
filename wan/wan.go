package wan

import (
	"fmt"
	"os"
	"tcp-tunnel/config"
	"tcp-tunnel/logger"

	"github.com/yymmiinngg/goargs"
)

func Start(argsArr []string, log *logger.Logger) {

	template := `
    Usage: {{COMMAND}} WAN {{OPTION}}

	+ -a, --application-port  # Listen an open port for an application (Format ip:port,
	#                           Default: 127.0.0.1:80)
	* -s, --server-port       # Listen a port for forwarding traffic from LAN to an 
	#                           open application port (Format ip:port)
	+ -i, --io-timeout        # Read/Write Timeout Duration (Unit: Seconds, Default: 120)
	+ -k, --handshake-key     # Handshake Key, Preventing Unauthorized Use of WAN Port

    ? -h, --help              # Show Help and Exit
    ? -v, --version           # Show Version and Exit
`

	// 定义变量
	var applicationAddress string
	var serverAddress string
	var handshakeKey string
	var ioTimeout int

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// 绑定变量
	args.StringOption("-a", &applicationAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverAddress, "")
	args.IntOption("-i", &ioTimeout, 120)
	args.StringOption("-k", &handshakeKey, config.DEFAULT_HANDSHAKE_KEY)

	// 处理参数
	err = args.Parse(argsArr)

	// 显示帮助
	if args.HasItem("-h", "--help") {
		fmt.Println(args.Usage())
		os.Exit(1)
	}

	// 显示版本
	if args.HasItem("-v", "--version") {
		fmt.Println("v0.0.1")
		os.Exit(1)
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if ioTimeout == 0 {
		fmt.Println("The io timeout duration cannot be less than 1")
		os.Exit(1)
	}

	serverInfo := MakeServerInfo(serverAddress, applicationAddress, handshakeKey, ioTimeout, log)
	serverInfo.StartServer()
}
