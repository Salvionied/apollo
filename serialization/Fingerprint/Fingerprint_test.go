package Fingerprint_test

import (
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/AssetName"
	"github.com/Salvionied/apollo/v2/serialization/Fingerprint"
	"github.com/Salvionied/apollo/v2/serialization/Policy"
)

type data struct {
	policyId     string
	assetNameHex string
	fingerprint  string
}

// data references from https://cips.cardano.org/cip/CIP-14
var dataSet = []data{
	{
		policyId:     "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc373",
		assetNameHex: "",
		fingerprint:  "asset1rjklcrnsdzqp65wjgrg55sy9723kw09mlgvlc3",
	},
	{
		policyId:     "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc37e",
		assetNameHex: "",
		fingerprint:  "asset1nl0puwxmhas8fawxp8nx4e2q3wekg969n2auw3",
	},
	{
		policyId:     "1e349c9bdea19fd6c147626a5260bc44b71635f398b67c59881df209",
		assetNameHex: "",
		fingerprint:  "asset1uyuxku60yqe57nusqzjx38aan3f2wq6s93f6ea",
	},
	{
		policyId:     "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc373",
		assetNameHex: "504154415445",
		fingerprint:  "asset13n25uv0yaf5kus35fm2k86cqy60z58d9xmde92",
	},
	{
		policyId:     "1e349c9bdea19fd6c147626a5260bc44b71635f398b67c59881df209",
		assetNameHex: "504154415445",
		fingerprint:  "asset1hv4p5tv2a837mzqrst04d0dcptdjmluqvdx9k3",
	},
	{
		policyId:     "1e349c9bdea19fd6c147626a5260bc44b71635f398b67c59881df209",
		assetNameHex: "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc373",
		fingerprint:  "asset1aqrdypg669jgazruv5ah07nuyqe0wxjhe2el6f",
	},
	{
		policyId:     "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc373",
		assetNameHex: "1e349c9bdea19fd6c147626a5260bc44b71635f398b67c59881df209",
		fingerprint:  "asset17jd78wukhtrnmjh3fngzasxm8rck0l2r4hhyyt",
	},
	{
		policyId:     "7eae28af2208be856f7a119668ae52a49b73725e326dc16579dcc373",
		assetNameHex: "0000000000000000000000000000000000000000000000000000000000000000",
		fingerprint:  "asset1pkpwyknlvul7az0xx8czhl60pyel45rpje4z8w",
	},
}

func TestFingerprintSet(t *testing.T) {
	for _, data := range dataSet {
		policyId, _ := Policy.New(data.policyId)
		assetName := AssetName.NewAssetNameFromHexString(data.assetNameHex)
		fingerprint := Fingerprint.New(policyId, assetName)
		if fingerprint.String() != data.fingerprint {
			t.Errorf(
				"\nPolicyId: %v\nAssetName: %v\nFingerprint: %v\nExpected: %v",
				data.policyId,
				data.assetNameHex,
				fingerprint.String(),
				data.fingerprint,
			)
		}
	}
}
