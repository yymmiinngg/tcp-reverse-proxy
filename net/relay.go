package nets

import (
	"net"
	"time"
)

func Relay(conn1, conn2 net.Conn, ioTimeout int) {
	// 超时起始时间
	var lastIoTime time.Time = time.Now()
	// 下行
	go func() {
		defer conn1.Close()
		defer conn2.Close()
		buff := make([]byte, 1024)
		for {
			conn1.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
			size, err := conn1.Read(buff)
			if err != nil {
				if ParseNetError(err) == NetErrorTimeout && time.Since(lastIoTime) < time.Duration(ioTimeout)*time.Second {
					continue
				}
				break
			}
			lastIoTime = time.Now()
			conn2.Write(buff[:size])
		}
	}()

	// 上行
	func() {
		defer conn1.Close()
		defer conn2.Close()
		buff := make([]byte, 1024)
		for {
			conn2.SetReadDeadline(time.Now().Add(time.Duration(ioTimeout) * time.Second))
			size, err := conn2.Read(buff)
			if err != nil {
				if ParseNetError(err) == NetErrorTimeout && time.Since(lastIoTime) < time.Duration(ioTimeout)*time.Second {
					continue
				}
				break
			}
			lastIoTime = time.Now()
			conn1.Write(buff[:size])
		}
	}()
}
