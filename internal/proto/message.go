// Package proto ensures strict compliance of all messages with protocol.
package proto

type MessageKind uint8

// Service messages
const (
	SrvConnKeepAlive MessageKind = iota
	SrvConnClose
)

// TCP proxy messages
const (
	TCPConnectionOpen MessageKind = iota + 50
	TCPConnectionClose
	TCPConnectionData
)

// Serialize message kind. Wri
func (mk MessageKind) Serialize(msg []byte) {
	msg[0] = byte(mk)
}

func ReadMessageKind(msg []byte) MessageKind {
	return MessageKind(msg[0])
}
