package nets

import (
	"net"
	"tcp-tunnel/core"
	"time"
)

type DataProcessor func(src []byte) (dest []byte)

func Relay(conn1, conn2 net.Conn, ioTimeout int, cryptor core.Cryptor) {
	// 超时起始时间
	var lastIoTime time.Time = time.Now()
	// 下行
	go func() {
		defer conn1.Close()
		defer conn2.Close()
		buff := make([]byte, 1024)
		for {
			if ioTimeout != 0 {
				conn1.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
			}
			size, err := conn1.Read(buff)
			if err != nil {
				if ParseNetError(err) == NetErrorTimeout && time.Since(lastIoTime) < time.Duration(ioTimeout)*time.Second {
					continue
				}
				// fmt.Println(err.Error())
				break
			}
			lastIoTime = time.Now()

			// 数据处理
			var buff2 = buff[:size]
			if cryptor != nil {
				buff2 = make([]byte, size)
				cryptor.Encrypt(buff[:size], buff2)
			}

			conn2.Write(buff2)
		}
	}()

	// 上行
	func() {
		defer conn1.Close()
		defer conn2.Close()
		buff := make([]byte, 1024)
		for {
			if ioTimeout != 0 {
				conn2.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
			}
			size, err := conn2.Read(buff)
			if err != nil {
				if ParseNetError(err) == NetErrorTimeout && time.Since(lastIoTime) < time.Duration(ioTimeout)*time.Second {
					continue
				}
				// fmt.Println(err.Error())
				break
			}
			lastIoTime = time.Now()

			// 数据处理
			var buff2 = buff[:size]
			if cryptor != nil {
				buff2 = make([]byte, size)
				cryptor.Decrypt(buff[:size], buff2)
			}

			conn1.Write(buff2)
		}
	}()
}
