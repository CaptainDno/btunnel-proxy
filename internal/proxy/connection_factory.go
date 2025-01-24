package proxy

import (
	"fmt"
	"github.com/CaptainDno/btunnel"
	"math/rand"
	"sync/atomic"
)

// BTunnelConnectionFactory is used to create new connections
type BTunnelConnectionFactory interface {
	NewBTunnelConnection() (*btunnel.Connection, error)
}

type DefaultConnectionFactory struct {
	idCounter       atomic.Uint32
	prefixName      string
	addr            string
	clientID        string
	keyStore        btunnel.KeyStore
	availableKeyIDs [][]byte
}

func (f *DefaultConnectionFactory) NewBTunnelConnection() (*btunnel.Connection, error) {

	return btunnel.Connect(
		fmt.Sprintf("%s-%d", f.prefixName, f.idCounter.Add(1)),
		f.addr,
		f.availableKeyIDs[rand.Intn(len(f.availableKeyIDs))],
		f.clientID,
		f.keyStore,
	)
}
