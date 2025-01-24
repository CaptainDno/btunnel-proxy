package proxy

import (
	"context"
	"errors"
	"github.com/CaptainDno/btunnel-proxy/internal/conn"
	cmap "github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap"
	"sync/atomic"
	"time"
)

// btun message format

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

func (b *ClientBridge) spawnTunnel(ctx context.Context) error {
	connection, err := b.bTunnelConnectionFactory.NewBTunnelConnection()
	// TODO Use service messages to negotiate connection close
	if err != nil {
		return err
	}

	go func() {
		msg := <-b.messageChannel

	}()

	go func() {

	}()
}

// ServerBridge accepts ClientBridge connections
type ServerBridge struct {
	bridgeBase
	maxActiveTunnelCount uint32
}
