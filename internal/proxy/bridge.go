package proxy

import (
	"context"
	"errors"
	"github.com/CaptainDno/btunnel"
	"github.com/CaptainDno/btunnel-proxy/internal/conn"
	"github.com/CaptainDno/btunnel-proxy/internal/proto"
	cmap "github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap"
	"sync/atomic"
	"time"
)

type Bridge interface {
	// Send message using the bridge
	Send(message conn.Message)
}

type bridgeBase struct {
	logger zap.Logger

	name string
	// Map of all connections handled by this bridge
	connections cmap.ConcurrentMap[uint32, conn.Connection]

	// Channel for sending and receiving messages
	messageChannel chan conn.Message

	// Current number of active tunnels
	activeTunnelCount atomic.Uint32
}

func (b *bridgeBase) Send(message conn.Message) {
	switch message.Kind {
	case conn.ConnectionOpen:
		b.connections.Set(message.Connection.GetID(), message.Connection)
		break

	case conn.ConnectionClose:
		// TODO Add removal
		b.connections.Remove(message.Connection.GetID())
		break
	}
	b.messageChannel <- message
}

// ClientBridge initiates connection to listening server bridge
type ClientBridge struct {
	bridgeBase
	// Connection factory used to create btunnel connections
	bTunnelConnectionFactory BTunnelConnectionFactory
	minActiveTunnelCount     uint32
	maxActiveTunnelCount     uint32
}

// This method automatically tries to spawn new tunnel when message channel is filled to certain capacity
func (b *ClientBridge) autoSpawnTunnels(ctx context.Context) {
	go func() {
		for {
			time.Sleep(time.Second)

			if err := ctx.Err(); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				} else {
					b.logger.Fatal("unexpected context error", zap.Error(err))
				}
			}

			if float32(len(b.messageChannel)/cap(b.messageChannel)) > 0.8 {
				if b.activeTunnelCount.Load() < b.maxActiveTunnelCount {
					if err := b.spawnTunnel(); err != nil {
						b.logger.Error("failed to spawn tunnel automatically", zap.Error(err))
					}
				} else {
					b.logger.Warn("cannot spawn new tunnel due to active tunnel count limit. this may impact bridge speed")
				}
			}
		}

	}()
}

const MaxIdle = time.Second * 10
const MsgReadThreshold = 5

var ProtoProxy = zap.String("proto", "btp")

func createTunnel(ctx context.Context, connection *btunnel.Connection, messageChannel chan conn.Message, logger *zap.Logger) error {
	ctx, cancel := context.WithCancelCause(ctx)
	var err error
	// This counter is used to terminate unused connections
	read := atomic.Uint32{}

	// Separate goroutines are used for reading and writing from connection
	// Important notes:
	// 1) Connection may only be closed inside writer goroutine using context (except when write error occurs - it triggers immediate close).
	// 2) Both client and server may initiate connection close (even simultaneously)

	go func() {
		var msg conn.Message
	writeLoop:
		for {
			select {
			case msg = <-messageChannel:
				// TODO Send message
				err = connection.WriteMessage()

				if err != nil {
					logger.Error("failed to send message", ProtoProxy, zap.Error(err))
					// Return message back to channel
					messageChannel <- msg
					break writeLoop
				}

				break
			case <-time.After(MaxIdle):
				if read.Swap(0) < MsgReadThreshold {
					// Send close command and return from goroutine
					err = connection.WriteMessage(SrvCloseMessage)
					if err != nil {
						logger.Error("failed to send SrvConnClose message", ProtoProxy, zap.Error(err))
						break writeLoop
					}
					// No more writes may be done after sending SrvConnClose
					// When other end responds with SrvConnClose, reader calls cancel()
					<-ctx.Done()
					// Now it is safe to close the connection - no more messages may be transmitted by both sides
					break writeLoop
				}
			case <-ctx.Done():
				// This code will run only in two cases:
				// 1) Other end of the tunnel requested connection close
				// 2) Error occurred while reading message from connection
				// In both cases reader goroutine has already returned => no active read operations on tunnel => it is safe to close

				// In the first case, we need to send the same message back
				if errors.Is(ctx.Err(), context.Canceled) {
					err = connection.WriteMessage(SrvCloseMessage)
					if err != nil {
						logger.Error("failed to send reply SrvConnClose message", ProtoProxy, zap.Error(err))
					}
				}

				// If message was sent, at this moment it is already delivered. so connection can be closed
				break writeLoop
			}
		}

		// Handle connection close
		if err = ctx.Err(); !errors.Is(context.Canceled, err) {
			logger.Info("closing tunnel because of error", ProtoProxy, zap.Error(err))
		} else {
			logger.Info("closing tunnel because of inactivity", ProtoProxy)
		}

		err = connection.Close()
		if err != nil {
			logger.Error("failed to close connection", ProtoProxy, zap.Error(err))
		}
	}()

	go func() {
		for {
			msg, err := connection.ReadMessage()
			if err != nil {
				// This leads to case 2 (see case <-ctx.Done() in writer)
				cancel(err)
				logger.Error("error in tunnel", ProtoProxy, zap.Error(err))
				return
			}
			// Increment count of read messages
			read.Add(1)
			// TODO Handle all messages
			switch proto.ReadMessageKind(msg) {
			case proto.SrvConnClose:
				// This leads to case 1 (see case <-ctx.Done() in writer)
				logger.Info("commanded to close tunnel", ProtoProxy)
				cancel(nil)
				return
			case proto.SrvConnKeepAlive:
				logger.Debug("received keep alive message", ProtoProxy)
				break
			}
		}
	}()
}

// ServerBridge accepts ClientBridge connections
type ServerBridge struct {
	bridgeBase
	maxActiveTunnelCount uint32
}
