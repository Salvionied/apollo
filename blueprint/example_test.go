package blueprint_test

import (
	"fmt"
	"math/big"

	"github.com/Salvionied/apollo/v2/blueprint"
	"github.com/blinklabs-io/plutigo/data"
)

func ExampleParse() {
	raw := []byte(`{
		"preamble": {"title": "example/escrow", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "escrow.spend",
			"datum": {
				"purpose": "spend",
				"schema": {
					"dataType": "constructor",
					"index": 0,
					"fields": [
						{"title": "owner", "dataType": "bytes"},
						{"title": "amount", "dataType": "integer"}
					]
				}
			},
			"redeemer": {"purpose": "spend", "schema": {"dataType": "integer"}},
			"compiledCode": "46010100200101"
		}]
	}`)

	bp, err := blueprint.Parse(raw)
	if err != nil {
		panic(err)
	}
	fmt.Println(bp.ValidatorNames())
	// Output:
	// [escrow.spend]
}

func ExampleBlueprint_EncodeDatum() {
	raw := []byte(`{
		"preamble": {"title": "example/escrow", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "escrow.spend",
			"datum": {
				"purpose": "spend",
				"schema": {
					"dataType": "constructor",
					"index": 0,
					"fields": [
						{"title": "owner", "dataType": "bytes"},
						{"title": "amount", "dataType": "integer"}
					]
				}
			},
			"redeemer": {"purpose": "spend", "schema": {"dataType": "integer"}},
			"compiledCode": "46010100200101"
		}]
	}`)

	type EscrowDatum struct {
		Owner  string
		Amount int64
	}

	bp, err := blueprint.Parse(raw)
	if err != nil {
		panic(err)
	}
	pd, err := bp.EncodeDatum("escrow.spend", EscrowDatum{Owner: "deadbeef", Amount: 42})
	if err != nil {
		panic(err)
	}
	if err := bp.ValidateDatum("escrow.spend", pd); err != nil {
		panic(err)
	}
	var decoded EscrowDatum
	if err := bp.DecodeDatum("escrow.spend", pd, &decoded); err != nil {
		panic(err)
	}
	fmt.Printf("%s %d\n", decoded.Owner, decoded.Amount)
	// Output:
	// deadbeef 42
}

func ExampleValidatorScript_ApplyParams() {
	raw := []byte(`{
		"preamble": {"title": "example/mint", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "policy.mint",
			"redeemer": {"purpose": "mint", "schema": {"dataType": "integer"}},
			"parameters": [
				{"title": "nonce", "purpose": "mint", "schema": {"dataType": "integer"}}
			],
			"compiledCode": "46010100200101"
		}]
	}`)

	bp, err := blueprint.Parse(raw)
	if err != nil {
		panic(err)
	}
	script, err := bp.Script("policy.mint")
	if err != nil {
		panic(err)
	}
	applied, err := script.ApplyParams(int64(42))
	if err != nil {
		panic(err)
	}
	fmt.Println(applied.Script.CborHex(), len(applied.Script.RemainingParameters()))
	// Output:
	// 4c010100320014c102182a0001 0
}

func ExampleBlueprint_ValidateDatum() {
	raw := []byte(`{
		"preamble": {"title": "example/check", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "check.spend",
			"datum": {
				"purpose": "spend",
				"schema": {"dataType": "integer", "minimum": 10}
			},
			"redeemer": {"purpose": "spend", "schema": {"dataType": "integer"}}
		}]
	}`)

	bp, err := blueprint.Parse(raw)
	if err != nil {
		panic(err)
	}
	okErr := bp.ValidateDatum("check.spend", data.NewInteger(big.NewInt(15)))
	badErr := bp.ValidateDatum("check.spend", data.NewInteger(big.NewInt(3)))
	fmt.Println(okErr == nil, badErr != nil)
	// Output:
	// true true
}
