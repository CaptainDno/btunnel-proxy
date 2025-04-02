module github.com/CaptainDno/btunnel-proxy

go 1.23

// TODO Comment when pushing to master
replace github.com/CaptainDno/btunnel => ../BTUNNEL

require (
	github.com/CaptainDno/btunnel v0.0.0-20250123174047-3e9c4c9cd676
	go.uber.org/zap v1.27.0
)

require (
	github.com/akrylysov/pogreb v0.10.2 // indirect
	github.com/mhmtszr/concurrent-swiss-map v1.0.8 // indirect
	github.com/things-go/go-socks5 v0.0.5 // indirect
	github.com/urfave/cli/v3 v3.1.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
)
