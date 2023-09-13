package lan

import (
	"fmt"

	"github.com/yymmiinngg/goargs"
)

/*

局域网端：

lanConn -> localConn

*/

func Start(argsArr []string) {

	template := `
    Usage: {{COMMAND}} LAN {{OPTION}}

	+ -a, --application-address  # 映射应用的TCP地址（格式：”ip:port“, 默认：127.0.0.1:80）
	* -s, --server-address       # 关联的服务端TCP地址（ip:port）
	+ -r, --max-ready-connection # 最大准备连接数（默认：5）
	+ -c, --connect-timeout      # 连接超时时长（单位：秒，默认：10)
	+ -i, --io-timeout           # 读写超时时长（单位：秒，默认：120)
	+ -k, --handshake-key        # 握手密钥，防止WAN端被非法使用

    ? -h, --help                 # 显示帮助后退出
    ? -v, --version              # 显示版本后退出
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
	var handshakeKey string

	var maxReadyConnection, connectTimeout, ioTimeout int

	// 绑定变量
	args.StringOption("-a", &applicationAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverAddress, "")
	args.IntOption("-r", &maxReadyConnection, 5)
	args.IntOption("-c", &connectTimeout, 10)
	args.IntOption("-i", &ioTimeout, 120)
	args.StringOption("-k", &handshakeKey, "l3I19DqQ3T1eYjKn")

	// 处理参数
	err = args.Parse(argsArr)

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
		fmt.Println("--------------------------------------------------")
		fmt.Println(args.Usage())
		return
	}

	// 局域网的连接
	serverConnection := MakeServerConnectionPoolInfo(serverAddress, applicationAddress, handshakeKey, maxReadyConnection, connectTimeout, ioTimeout)
	for {
		serverConnection.GetServerConnect()
	}
}
