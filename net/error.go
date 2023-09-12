package nets

import (
	"net"
	"os"
	"syscall"
)

// NetError NetError
type NetError int

const (
	// NetErrorNil NetErrorNil
	NetErrorNil NetError = 0
	// NetErrorUnknow NetErrorUnknow
	NetErrorUnknow NetError = 1
	// NetErrorTimeout NetErrorTimeout
	NetErrorTimeout NetError = 2
	// NetErrorConnectRefused NetErrorConnectRefused
	NetErrorConnectRefused NetError = 3
	// NetErrorDNS NetErrorDNS
	NetErrorDNS NetError = 4
	// NetErrorInvalidAddrError NetErrorInvalidAddrError
	NetErrorInvalidAddrError NetError = 5
	// NetErrorUnknownNetwork NetErrorUnknownNetwork
	NetErrorUnknownNetwork NetError = 6
	// NetErrorAddrError NetErrorAddrError
	NetErrorAddrError NetError = 7
	// NetErrorDNSConfigError NetErrorDNSConfigError
	NetErrorDNSConfigError NetError = 8
)

// ParseNetError ParseNetError
func ParseNetError(err error) NetError {
	if err == nil {
		return NetErrorNil
	}
	netErr, ok := err.(net.Error)
	if !ok {
		return NetErrorUnknow
	}
	if netErr.Timeout() {
		return NetErrorTimeout
	}
	opErr, ok := netErr.(*net.OpError)
	if !ok {
		return NetErrorUnknow
	}
	switch t := opErr.Err.(type) {
	case *net.DNSError:
		return NetErrorDNS
	case *net.InvalidAddrError:
		return NetErrorInvalidAddrError
	case *net.UnknownNetworkError:
		return NetErrorUnknownNetwork
	case *net.AddrError:
		return NetErrorAddrError
	case *net.DNSConfigError:
		return NetErrorDNSConfigError
	case *os.SyscallError:
		if errno, ok := t.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.ECONNREFUSED:
				return NetErrorConnectRefused
			case syscall.ETIMEDOUT:
				return NetErrorTimeout
			}
		}
	}
	return NetErrorUnknow
}
