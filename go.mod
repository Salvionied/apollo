module github.com/Salvionied/apollo

go 1.20

require (
	github.com/Salvionied/cbor/v2 v2.6.0
	github.com/SundaeSwap-finance/kugo v0.1.5
	github.com/SundaeSwap-finance/ogmigo v0.8.0
	github.com/SundaeSwap-finance/ogmigo/v6 v6.0.0-20240117201106-ce491d0b031e
	github.com/tyler-smith/go-bip39 v1.1.0
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/text v0.9.0
)

require (
	github.com/aws/aws-sdk-go v1.44.197 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

require (
	filippo.io/edwards25519 v1.0.0
	github.com/maestro-org/go-sdk v1.1.3
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.8.0
)

replace github.com/maestro-org/go-sdk v1.1.2 => ./go-sdk
