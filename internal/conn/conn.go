package conn

type Connection interface {
	GetID() uint32
	HandleMessage(message Message)
}
