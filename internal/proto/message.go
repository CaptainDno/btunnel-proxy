// Package proto ensures strict compliance of all messages with protocol.
package proto

import "encoding/binary"

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
	TCPDialError
	TCPDialSuccess
)

func Kind(msg []byte) MessageKind {
	return MessageKind(msg[0])
}

func SetKind(msg []byte, kind MessageKind) {
	msg[0] = byte(kind)
}

func SetCID(msg []byte, id uint32) {
	binary.BigEndian.PutUint32(msg[1:], id)
}

func GetCID(msg []byte) uint32 {
	return binary.BigEndian.Uint32(msg[1:])
}

const HeaderLength = 4 + 1
