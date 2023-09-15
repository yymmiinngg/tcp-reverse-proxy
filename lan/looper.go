package lan

import (
	"sync"
	"tcp-tunnel/core"
	"tcp-tunnel/logger"
)

type RelayConnectionLooper struct {
	maxReadyConnect    int
	readyLock          sync.Locker
	readyConnect       int // 准备的连接
	relayAddress       string
	applicationAddress string
	handshaker         *core.Handshaker
	log                *logger.Logger
	looping            bool
}
