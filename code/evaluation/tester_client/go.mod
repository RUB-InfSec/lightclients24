module example.com/tester-client

go 1.22.1

replace example.com/light-client-signature => ../../implementation

require (
	example.com/light-client-signature v0.0.0-00010101000000-000000000000
	github.com/filecoin-project/go-jsonrpc v0.3.1
)

require (
	github.com/bits-and-blooms/bitset v1.7.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/ipfs/go-log/v2 v2.0.8 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
