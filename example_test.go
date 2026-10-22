package apollo_test

import (
	"encoding/hex"
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"

	apollo "github.com/Salvionied/apollo/v2"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// Example demonstrates the basic, externally visible builder setup. Transaction
// completion requires spendable UTxOs and is covered by the package's
// deterministic backend tests.
func Example() {
	rawAddress := make([]byte, 57)
	rawAddress[1] = 0xaa
	rawAddress[29] = 0xbb
	address, err := common.NewAddressFromBytes(rawAddress)
	if err != nil {
		panic(err)
	}

	chainContext := fixed.NewEmptyFixedChainContext()
	builder := apollo.New(chainContext).
		SetWallet(apollo.NewExternalWallet(address)).
		PayToAddress(address, 2_000_000)

	_ = builder
	fmt.Println(builder != nil)
	// Output: true
}

// ExampleNewScriptRef verifies the error-returning reference-script API from an
// external consumer's perspective.
func ExampleNewScriptRef() {
	script := common.PlutusV3Script{0x01}
	scriptRef, err := apollo.NewScriptRef(script)
	if err != nil {
		panic(err)
	}

	_ = scriptRef
	fmt.Println(scriptRef != nil)
	// Output: true
}

// ExampleApplyParamsToScript applies a byte-string parameter to a
// parameterized script, such as the compiledCode of an Aiken blueprint
// validator.
func ExampleApplyParamsToScript() {
	// (program 1.1.0 (lam x x)), CBOR-wrapped flat.
	compiledCode, err := hex.DecodeString("46010100200101")
	if err != nil {
		panic(err)
	}
	ownerKeyHash, err := hex.DecodeString("deadbeef")
	if err != nil {
		panic(err)
	}

	script, err := apollo.ApplyParamsToScript(
		common.PlutusV3Script(compiledCode),
		data.NewByteString(ownerKeyHash),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(script))
	// Output: 4f010100320014c10544deadbeef0001
}
