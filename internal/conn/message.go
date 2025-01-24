package conn

import "github.com/CaptainDno/btunnel-proxy/internal/proto"

type Message struct {
	Kind       proto.MessageKind
	Connection Connection
	Data       []byte
}

type ConnectionOpenMessage struct {
	Message
}
