package bridge

import (
	"context"
	"github.com/CaptainDno/btunnel"
	"github.com/CaptainDno/btunnel-proxy/internal/proto"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
	"go.uber.org/zap"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

// Proxied connectioon writers should only be closed after reading TcpConnClose on Reader
// Readers automatically close themselves right after receiving TcpConnClose.

type Bridge struct {
	// Underlying BTUNNEL connection
	tun     *btunnel.Connection
	tunLock sync.Mutex

	// Logger used by this bridge
	logger *zap.Logger

	// Counter for generating unique connection IDs
	idCounter atomic.Uint32

	// Map of all connections
	connmap *csmap.CsMap[uint32, io.ReadWriteCloser]

	// Context
	ctx    context.Context
	cancel context.CancelCauseFunc
}

func Open(ctx context.Context, logger *zap.Logger, tun *btunnel.Connection) (*Bridge, error) {

	ctx, cancel := context.WithCancelCause(ctx)

	brd := &Bridge{
		logger:    logger,
		tun:       tun,
		tunLock:   sync.Mutex{},
		idCounter: atomic.Uint32{},
		ctx:       ctx,
		cancel:    cancel,
		connmap:   csmap.Create[uint32, io.ReadWriteCloser](),
	}

	go brd.serveTun()

	return brd, nil
}

func (brd *Bridge) tunWrite(msg []byte) error {
	brd.tunLock.Lock()
	err := brd.tun.WriteMessage(msg)
	brd.tunLock.Unlock()
	return err
}

func (brd *Bridge) Close() error {
	_ = brd.tun.Close()
	// Close all connections
	brd.connmap.Range(func(cid uint32, conn io.ReadWriteCloser) bool {
		_ = conn.Close()
		return false
	})
	brd.cancel(nil)
	return nil
}

func (brd *Bridge) HandleConn(conn io.ReadWriteCloser, targetAddr string) {
	cid := brd.idCounter.Add(1)
	brd.connmap.Store(cid, conn)

	brd.logger.Info("handling socks connection", zap.String("remote", targetAddr), zap.Uint32("cid", cid))

	// Allocate buffer only once
	buf := make([]byte, proto.HeaderLength+len(targetAddr))

	l := proto.WriteTCPOpenMessage(targetAddr, cid, buf)
	err := brd.tunWrite(buf[:l])

	if err != nil {
		brd.logger.Error("failed to send TCPOpen message via tunnel", zap.Uint32("cid", cid), zap.Error(err))
		// Close faulty bridge
		brd.cancel(err)
		_ = brd.Close()
		return
	}

	brd.serveConn(conn, cid)
	brd.connmap.Delete(cid)
}

func (brd *Bridge) serveConn(conn io.ReadWriteCloser, cid uint32) {
	buf := make([]byte, 16*1024)
	var l int
	for {
		n, err := conn.Read(buf[proto.HeaderLength:])
		if n > 0 {
			proto.WriteTCPDataMessageHeader(cid, buf)
			err = brd.tunWrite(buf[:proto.HeaderLength+n])
			if err != nil {
				brd.logger.Error("failed to send TCPDataMessage via tunnel", zap.Uint32("cid", cid), zap.Error(err))
				brd.cancel(err)
				_ = brd.Close()
			}
		}

		if err != nil {
			if err == io.EOF {
				_ = conn.Close()
				l = proto.WriteTCPCloseMessage(cid, buf)
				err = brd.tunWrite(buf[:l])
				if err != nil {
					brd.logger.Error("failed to send TCPCloseMessage via tunnel", zap.Uint32("cid", cid), zap.Error(err))
					brd.cancel(err)
					_ = brd.Close()
				}
				break
			}

			brd.logger.Warn("failed to read from tcp connection", zap.Uint32("cid", cid), zap.Error(err))
			_ = conn.Close()
			l = proto.WriteTCPCloseMessage(cid, buf)

			err = brd.tunWrite(buf[:l])
			if err != nil {
				brd.logger.Error("failed to send TCPCloseMessage via tunnel", zap.Uint32("cid", cid), zap.Error(err))
			}

			break
		}
	}
}

func (brd *Bridge) serveTun() {
	for {
		if brd.ctx.Err() != nil {
			brd.logger.Warn("context canceled", zap.Error(brd.ctx.Err()))
			_ = brd.Close()
			return
		}

		msg, err := brd.tun.ReadMessage()

		if err != nil {
			brd.logger.Error("failed to read message from tunnel", zap.Error(err))
			brd.cancel(err)
			_ = brd.Close()
			return
		}

		buf := make([]byte, 1024)

		switch proto.MessageKind(msg[0]) {
		case proto.TCPConnectionOpen:
			var address string
			cid, address := proto.ReadTCPOpenMessage(msg)

			go func() {
				conn, err := net.Dial("tcp", address)

				if err != nil {
					brd.logger.Warn("failed to dial tcp", zap.Uint32("cid", cid), zap.String("addr", address), zap.Error(err))
					l := proto.WriteTCPDialErrorMessage(cid, err.Error(), buf)
					err = brd.tunWrite(buf[:l])
					if err != nil {
						brd.logger.Error("failed to send TCPDialError message via tunnel", zap.Uint32("cid", cid), zap.Error(err))
						brd.cancel(err)
						_ = brd.Close()

					}
					return
				}

				brd.logger.Info("opened TCP connection successfully", zap.Uint32("cid", cid), zap.String("addr", address))

				brd.connmap.Store(cid, conn)

				conn.LocalAddr().String()
				l := proto.WriteTCPDialSuccessMessage(cid, buf, conn.LocalAddr())
				err = brd.tunWrite(buf[:l])
				if err != nil {
					brd.logger.Error("failed to send TCPDialSuccess message via tunnel", zap.Uint32("cid", cid), zap.Error(err))
					brd.cancel(err)
					_ = brd.Close()
					return
				}
				brd.serveConn(conn, cid)
			}()

			break
		case proto.TCPConnectionClose:
			cid := proto.ReadTCPCloseMessage(msg)
			if conn, ok := brd.connmap.Load(cid); ok {
				brd.connmap.Delete(cid)
				_ = conn.Close()
			}
			break
		case proto.TCPConnectionData:
			cid, data := proto.ReadTCPDataMessage(msg)
			if conn, ok := brd.connmap.Load(cid); ok {
				_, err = conn.Write(data)
				if err != nil {
					brd.logger.Error("failed to write data to tcp connection", zap.Uint32("cid", cid), zap.Error(err))
					brd.connmap.Delete(cid)
					// Connection will be closed, so the read error will be triggered, and that will cause TCPConnectionClose message to be sent
					// And that will force connection to be closed on other side
					_ = conn.Close()
				}
			} else {
				brd.logger.Warn("tried to write data to tcp connection, but no connection found", zap.Uint32("cid", cid))
			}
			break
		case proto.TCPDialSuccess:
			cid, addr := proto.ReadTCPDialSuccessMessage(msg)
			if conn, ok := brd.connmap.Load(cid); ok {
				if err := socks5.SendReply(conn, statute.RepSuccess, addr); err != nil {
					brd.logger.Error("failed to send socks reply", zap.Uint32("cid", cid), zap.Error(err))
				}
			}
			break
		case proto.TCPDialError:
			cid, errmsg := proto.ReadTCPDialErrorMessage(msg)

			if conn, ok := brd.connmap.Load(cid); ok {
				// Code from socks package
				resp := statute.RepHostUnreachable
				if strings.Contains(errmsg, "refused") {
					resp = statute.RepConnectionRefused
				} else if strings.Contains(errmsg, "network is unreachable") {
					resp = statute.RepNetworkUnreachable
				}
				if err := socks5.SendReply(conn, resp, nil); err != nil {
					brd.logger.Error("failed to send socks reply", zap.Uint32("cid", cid), zap.Error(err))
				}

				// Close connection and delete from map
				_ = conn.Close()
				brd.connmap.Delete(cid)
			}
			break
		default:
			brd.logger.Warn("Unknown message kind, ignoring", zap.Uint8("kind", msg[0]))
			break
		}

	}
}
