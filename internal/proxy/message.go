package proxy

import "github.com/CaptainDno/btunnel-proxy/internal/proto"

var signals = [2]byte{
	byte(proto.SrvConnKeepAlive),
	byte(proto.SrvConnClose),
}

var (
	SrvKeepAliveMessage = signals[0:1]
	SrvCloseMessage     = signals[1:2]
)
