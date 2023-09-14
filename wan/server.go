package wan

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"tcp-tunnel/logger"
	"time"

	"github.com/yymmiinngg/goargs"
)

type Server struct {
	serverPort   string
	ioTimeout    int
	handshakeKey string
	log          *logger.Logger
}

func MakeServer(
	serverPort string,
	ioTimeout int,
	handshakeKey string,
	log *logger.Logger,
) *Server {
	return &Server{
		serverPort:   serverPort,
		ioTimeout:    ioTimeout,
		handshakeKey: handshakeKey,
		log:          log,
	}
}

func (it *Server) StartServer() {

	server, err := net.Listen("tcp", it.serverPort)
	if err != nil {
		it.log.Error(err, "listen server port error")
		os.Exit(1)
	}
	defer server.Close()
	it.log.Info("start server port:", it.serverPort)
	for {
		holdConn, err := server.Accept()
		if err != nil {
			it.log.Error(err, "accept server connection error")
			time.Sleep(1000)
			continue
		}
		it.log.Debug("get a lan connection", holdConn.LocalAddr().String(), "<-", holdConn.RemoteAddr().String())
		it.HandlConn(holdConn)
	}
}

func (it *Server) HandlConn(holeConn net.Conn) {
	defer holeConn.Close()

	// 读取的第一行是转发参数
	br := bufio.NewReader(holeConn)
	line, err := br.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}

	argsArr := strings.Split(line, " ")

	template := `
		* -a 
		* -s 
	`

	// 定义变量
	var applicationAddress string
	var serverAddress string

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 绑定变量
	args.StringOption("-a", &applicationAddress, "127.0.0.1:80")
	args.StringOption("-s", &serverAddress, "")

	// 处理参数
	err = args.Parse(argsArr)

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	serverInfo := MakeServerInfo(serverAddress, applicationAddress, it.handshakeKey, it.ioTimeout, it.log)
	go serverInfo.StartServer()
	defer serverInfo.Close()

	// 长连接，断开则关闭整个转发链路
	buff := make([]byte, 32)
	for {

		size, err := holeConn.Read(buff)
		if err != nil {
			fmt.Println(err)
			break
		}

		_, err = holeConn.Write(buff[:size])
		if err != nil {
			fmt.Println(err)
			break
		}

	}

}
