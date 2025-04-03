package main

import (
	"context"
	"fmt"
	"github.com/CaptainDno/btunnel"
	"github.com/CaptainDno/btunnel-proxy/internal/bridge"
	"github.com/CaptainDno/btunnel-proxy/internal/keys"
	"github.com/things-go/go-socks5"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"io"
	"net"
	"os"
)

const ServerKeyPath = "./keys.pgb"

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	cmd := &cli.Command{Commands: []*cli.Command{
		{
			Name: "server",
			Commands: []*cli.Command{
				{
					Name:        "start",
					Description: "accept connections on address:port",
					Action: func(ctx context.Context, command *cli.Command) error {
						logger.Info("starting server")
						serverKeys, err := keys.Open(ServerKeyPath, logger)
						if err != nil {
							return err
						}

						listener, err := net.Listen("tcp", command.Args().Get(0))

						for {
							conn, err := listener.Accept()
							if err != nil {
								return err
							}

							tun, err := btunnel.Accept(logger, conn, serverKeys, func(clientID string) bool {
								return true
							})

							if err != nil {
								logger.Error("failed to accept connection", zap.Error(err))
								continue
							}

							_, err = bridge.Open(ctx, logger, tun)

							if err != nil {
								logger.Error("failed to open bridge", zap.Error(err))
							}

						}
					},
				},
				{
					Name:        "keygen",
					Description: "Generate new key for client with provided id. Both server and client key files need to be present (will be created if not)",
					Flags: []cli.Flag{
						&cli.IntFlag{
							Name:  "n",
							Usage: "Number of keys to generate",
							Value: 1,
						},
					},
					Action: func(ctx context.Context, command *cli.Command) error {
						serverKeys, err := keys.Open(ServerKeyPath, logger)
						if err != nil {
							return err
						}

						clientKeys, err := keys.Open(fmt.Sprintf("%s.pgb", command.Args().Get(0)), logger)
						if err != nil {
							return err
						}

						for i := int64(0); i < command.Int("n"); i++ {
							id, key := keys.GenerateKey()
							err = serverKeys.SetKey(id, key)
							if err != nil {
								return err
							}
							err = clientKeys.SetKey(id, key)
							if err != nil {
								return err
							}
						}

						return nil
					},
				},
			},
		},
		{
			Name: "client",
			Commands: []*cli.Command{
				{
					Name: "connect",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "id",
							Usage:    "id of client",
							Required: true,
						},
						&cli.StringFlag{
							Name:        "listen",
							Usage:       "listen address",
							DefaultText: "127.0.0.1:9999",
						},
					},
					Action: func(ctx context.Context, command *cli.Command) error {
						clientID := command.String("id")

						kstore, err := keys.Open(fmt.Sprintf("%s.pgb", clientID), logger)
						if err != nil {
							return err
						}

						kid, key, err := kstore.GetRandom()
						if err != nil {
							return err
						}

						logger.Info("using random key", zap.Binary("id", kid), zap.Binary("key", key))
						conn, err := btunnel.Connect(logger, command.Args().Get(0), kid, clientID, kstore)

						if err != nil {
							return err
						}

						brd, err := bridge.Open(ctx, logger, conn)
						if err != nil {
							logger.Error("failed to open bridge", zap.Error(err))
						}

						ss := socks5.NewServer(socks5.WithConnectHandle(func(ctx context.Context, writer io.Writer, request *socks5.Request) error {
							brd.HandleConn(&SocksConn{
								reader: request.Reader,
								writer: writer,
							}, request.DestAddr.String())

							return nil
						}))

						return ss.ListenAndServe("tcp", command.String("listen"))
					},
				},
			},
		},
	}}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Fatal(err.Error())
	}
}

type SocksConn struct {
	reader io.Reader
	writer io.Writer
}

func (sc *SocksConn) Read(p []byte) (n int, err error) {
	return sc.reader.Read(p)
}

func (sc *SocksConn) Write(p []byte) (n int, err error) {
	return sc.writer.Write(p)
}

type closeWriter interface {
	CloseWrite() error
}

func (sc *SocksConn) Close() error {
	if tcpConn, ok := sc.writer.(closeWriter); ok {
		return tcpConn.CloseWrite()
	}
	return nil
}
