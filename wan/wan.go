package wan

import (
	"fmt"
	"net"
	"os"
	"strings"
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

	+ -b, --bind-address          # Listen on a port for client connecting and binding, 
	#                               Like "0.0.0.0:3390" (Default: "0.0.0.0:3390")
	+ -k, --handshake-key         # Handshake key used for binding connections to protect the
	#                               server from unauthorized connection hijacking
	+ -i, --io-timeout            # Read/Write Timeout Duration in relaying (Unit: Seconds,
	#                               Default: 120)
	+ -C, --tls-x509-certificate  # The Certificate of tls
	+ -K, --tls-x509-key          # The private key of tls

    ?     --help                  # Show Help and Exit
`

	// 定义变量
	var bindAddress string
	var handshakeKey string
	var ioTimeout int
	var tlsCertificate, tlsPrivateKey string

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}

	// 绑定变量
	args.StringOption("-b", &bindAddress, "0.0.0.0:3390")
	args.IntOption("-i", &ioTimeout, 120)
	args.StringOption("-k", &handshakeKey, "")
	args.StringOption("-C", &tlsCertificate, "")
	args.StringOption("-K", &tlsPrivateKey, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.Has("--help", false) {
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
		return
	}

	if ioTimeout == 0 {
		fmt.Println("The io timeout duration cannot be less than 1")
		return
	}

	// 自动拼接IP
	if strings.HasPrefix(bindAddress, ":") {
		bindAddress = "0.0.0.0" + bindAddress
	}

	// 提取tcp地址
	bindAddr, err := net.ResolveTCPAddr("tcp", bindAddress)
	if err != nil {
		fmt.Println("resolve bind address error:", err.Error())
		return
	}

	// 证书和密钥必须成对出现
	if (tlsCertificate != "" && tlsPrivateKey == "") || (tlsCertificate == "" && tlsPrivateKey != "") {
		fmt.Println("tls certificate and private key must be pair")
		return
	}

	// 启动服务
	StartBindServer(
		bindAddr,
		ioTimeout,
		handshakeKey,
		log,
		tlsCertificate,
		tlsPrivateKey,
	)
}
