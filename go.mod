module github.com/Salvionied/apollo/v2

go 1.24.0

require (
	connectrpc.com/connect v1.19.1
	filippo.io/edwards25519 v1.1.0
	github.com/SundaeSwap-finance/kugo v1.3.0
	github.com/SundaeSwap-finance/ogmigo/v6 v6.2.0
	github.com/blinklabs-io/gouroboros v0.148.0
	github.com/maestro-org/go-sdk v1.2.1
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/utxorpc/go-codegen v0.18.1
	github.com/utxorpc/go-sdk v0.0.1
	golang.org/x/crypto v0.46.0
	golang.org/x/text v0.32.0
)

// XXX: uncomment when testing local changes to gouroboros
// replace github.com/blinklabs-io/gouroboros => ../../blink/gouroboros

replace github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.24.2

require (
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/btcsuite/btcutil v1.0.2 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
