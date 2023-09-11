package lan

import (
	"fmt"
	"io"
	"net"
	"time"
)

/*

局域网端：

lanConn -> localConn

*/

var lanConns = 0

func Start() {

	// 局域网的连接
	for {
		if lanConns >= 5 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		lanConn, err := net.DialTimeout("tcp", "127.0.0.1:6666", 5*time.Second)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Printf("connect %d \n", lanConns)
		lanConns++
		go handlConn(lanConn)
	}

}

// 转发
func handlConn(lanConn net.Conn) {
	defer lanConn.Close()

	// 远程的请求指令
	var buff = make([]byte, len("CONNECT"))
	size, err := lanConn.Read(buff)
	if err != nil {
		fmt.Println(err)
		lanConns--
		return
	}
	if string(buff[:size]) != "CONNECT" {
		fmt.Println("no CONNECT...")
		lanConns--
		return
	}

	fmt.Println("get CONNECT...")

	lanConns--

	// 请求目的服务器
	remoteConn, err := net.DialTimeout("tcp", "110.242.68.4:80", 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer remoteConn.Close()
	fmt.Println("req remote...")

	go io.Copy(lanConn, remoteConn)
	io.Copy(remoteConn, lanConn)
}
