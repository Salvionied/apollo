module github.com/SundaeSwap-finance/apollo

go 1.20

require (
	github.com/Salvionied/cbor/v2 v2.6.0
	github.com/SundaeSwap-finance/kugo v1.0.6-0.20231215030228-2eab7ae4f160
	github.com/SundaeSwap-finance/ogmigo/v6 v6.0.0-20240613041327-627a9f8c8240
	github.com/tyler-smith/go-bip39 v1.1.0
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/text v0.21.0
)

require (
	github.com/SundaeSwap-finance/ogmigo v0.10.0 // indirect
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

require (
	filippo.io/edwards25519 v1.0.0
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.32.0
)

replace github.com/SundaeSwap-finance/kugo => ../kugo
