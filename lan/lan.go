package lan

import (
	"fmt"
	"os"
	"tcp-tunnel/logger"

	"github.com/yymmiinngg/goargs"
)

func Start(argsArr []string, log *logger.Logger) {

	template := `
    Usage: {{COMMAND}} LAN {{OPTION}}

	+ -a, --application-address  # Mapped TCP Address for the Application, (Format: ip:port,
	#                              Default: 127.0.0.1:80)
	* -s, --server-address       # Associated Server TCP Address (Format: ip:port)
	* -o, --open-address         # Associated Server TCP Address (Format: ip:port)
	+ -r, --max-ready-connection # Maximum Ready Connection Count (Default: 5), Ready
	#                              connections help improve client connection speed. The
	#                              quantity limit is 1024.
	+ -c, --connect-timeout      # Connection Timeout Duration (Unit: Seconds, Default: 10)
	+ -i, --io-timeout           # Read/Write Timeout Duration (Unit: Seconds, Default: 120)
	+ -k, --handshake-key        # Handshake Key, Preventing Unauthorized Use of WAN Port

    ? -h, --help                 # Show Help and Exit
    ? -v, --version              # Show Version and Exit
    `

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}

	// 定义变量
	var applicationAddress string
	var serverAddress string
	var openAddress string
	var handshakeKey string

	var maxReadyConnection, connectTimeout, ioTimeout int

	// 绑定变量
	args.StringOption("-a", &applicationAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverAddress, "")
	args.StringOption("-o", &openAddress, "")
	args.IntOption("-r", &maxReadyConnection, 5)
	args.IntOption("-c", &connectTimeout, 10)
	args.IntOption("-i", &ioTimeout, 120)
	args.StringOption("-k", &handshakeKey, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.HasItem("-h", "--help") {
		fmt.Println(args.Usage())
		return
	}

	// 显示版本
	if args.HasItem("-v", "--version") {
		fmt.Println("v0.0.1")
		return
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}

	if maxReadyConnection < 1 {
		fmt.Println("The minimum ready connection count is 1")
		os.Exit(1)
		return
	}

	if maxReadyConnection > 1024 {
		fmt.Println("The maximum ready connection count is 1024")
		os.Exit(1)
		return
	}

	if connectTimeout == 0 {
		fmt.Println("The connection timeout duration cannot be less than 1")
		os.Exit(1)
		return
	}

	if ioTimeout == 0 {
		fmt.Println("The io timeout duration cannot be less than 1")
		os.Exit(1)
		return
	}

	StartClient(serverAddress,
		openAddress,
		applicationAddress,
		handshakeKey,
		maxReadyConnection,
		connectTimeout,
		ioTimeout,
		log,
	)
}
