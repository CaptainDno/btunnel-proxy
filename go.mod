module github.com/CaptainDno/btunnel-proxy

go 1.23

// TODO Comment when pushing to master
replace github.com/CaptainDno/btunnel => ../BTUNNEL


require (
	github.com/CaptainDno/btunnel v0.0.0-20250123174047-3e9c4c9cd676
	github.com/orcaman/concurrent-map/v2 v2.0.1
	go.uber.org/zap v1.27.0
)

require (
	github.com/things-go/go-socks5 v0.0.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
)
