package wan

import (
	"fmt"
	"io"
	"net"
)

/*


公网端：

clientConn -> client_connection_port <==> lan_connection_port -> lanConn

*/

var lanConns = make(chan net.Conn, 5)

func Start() {

	// 局域网的连接
	go func() {
		server, err := net.Listen("tcp", "127.0.0.1:6666")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer server.Close()
		fmt.Println("start lan server...")
		for {
			conn, err := server.Accept()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("get lan connection...")
			lanConns <- conn
		}
	}()

	// 客户端的连接
	func() {
		server, err := net.Listen("tcp", "127.0.0.1:80")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer server.Close()
		fmt.Println("start client server...")
		for {
			clientConn, err := server.Accept()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("get wan connection...")
			// 处理客户端连接
			go handlConn(clientConn, <-lanConns)
		}
	}()

}

func handlConn(clientConn, lanConn net.Conn) {
	defer clientConn.Close()
	defer lanConn.Close()
	fmt.Println("send CONNECT...")
	lanConn.Write([]byte("CONNECT"))
	go io.Copy(clientConn, lanConn)
	io.Copy(lanConn, clientConn)
}
