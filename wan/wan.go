package wan

import (
	"fmt"
	"os"
	"tcp-tunnel/logger"

	"github.com/yymmiinngg/goargs"
)

/*

	+ -a, --application-port  # Listen an open port for an application (Format ip:port,
	#                           Default: 127.0.0.1:80)

*/

func Start(argsArr []string, log *logger.Logger) {

	template := `
    Usage: {{COMMAND}} WAN {{OPTION}}

	+ -b, --bind-address      # Listen on a port for client connecting and binding, 
	#                           Like "0.0.0.0:3390" (Default: ":3390")
	+ -i, --io-timeout        # Read/Write Timeout Duration (Unit: Seconds, Default: 120)
	+ -k, --handshake-key     # Handshake key used for binding connections to protect the
	#                           server from unauthorized connection hijacking

    ? -h, --help              # Show Help and Exit
    ? -v, --version           # Show Version and Exit
`

	// 定义变量
	var bindAddress string
	var handshakeKey string
	var ioTimeout int

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}

	// 绑定变量
	args.StringOption("-b", &bindAddress, ":3390")
	args.IntOption("-i", &ioTimeout, 120)
	args.StringOption("-k", &handshakeKey, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.Has("-h", false) {
		fmt.Println(args.Usage())
		return
	}

	// 显示版本
	if args.Has("-v", false) {
		fmt.Println("v0.0.1")
		return
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}

	if ioTimeout == 0 {
		fmt.Println("The io timeout duration cannot be less than 1")
		os.Exit(1)
		return
	}

	// 启动服务
	StartBindServer(
		bindAddress,
		ioTimeout,
		handshakeKey,
		log,
	)
}
