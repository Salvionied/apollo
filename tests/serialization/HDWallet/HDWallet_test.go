package hdwallet_test

import (
	"testing"

	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/HDWallet"
	"github.com/SundaeSwap-finance/apollo/serialization/Key"
)

var MNEMONIC_12 = "test walk nut penalty hip pave soap entry language right filter choice"
var MNEMONIC_15 = "art forum devote street sure rather head chuckle guard poverty release quote oak craft enemy"
var MNEMONIC_24 = "excess behave track soul table wear ocean cash stay nature item turtle palm soccer lunch horror start stumble month panic right must lock dress"

var MNEMONIC_12_ENTROPY = "df9ed25ed146bf43336a5d7cf7395994"
var MNEMONIC_15_ENTROPY = "0ccb74f36b7da1649a8144675522d4d8097c6412"
var MNEMONIC_24_ENTROPY = "4e828f9a67ddcff0e6391ad4f26ddb7579f59ba14b6dd4baf63dcfdb9d2420da"

func TestIsMnemonic(t *testing.T) {
	res := HDWallet.IsMnemonic(MNEMONIC_12)
	if res != true {
		t.Errorf("IsMnemonic() failed")
	}
	res = HDWallet.IsMnemonic(MNEMONIC_15)
	if res != true {
		t.Errorf("IsMnemonic() failed")
	}
	res = HDWallet.IsMnemonic(MNEMONIC_24)
	if res != true {
		t.Errorf("IsMnemonic() failed")
	}

}

func TestIsMnemonicFail(t *testing.T) {
	res := HDWallet.IsMnemonic("test walk nut penalty hip pave soap entry language right filter choice test")
	if res != false {
		t.Errorf("IsMnemonic() failed")
	}
}

func TestMnemonicGeneration(t *testing.T) {
	mnemo, _ := HDWallet.GenerateMnemonic()
	res := HDWallet.IsMnemonic(mnemo)
	if res != true {
		t.Errorf("GenerateMnemonic() failed")
	}
}

func TestPaymentAddress12Reward(t *testing.T) {
	hd, _ := HDWallet.NewHDWalletFromMnemonic(MNEMONIC_12, "")
	hdWallet_stake, _ := hd.DerivePath("m/1852'/1815'/0'/2/0")
	stake_public_key := hdWallet_stake.XPrivKey.PublicKey()
	vkh, err := Key.VerificationKey{Payload: stake_public_key}.Hash()
	if err != nil {
		panic(err)
	}
	addr := Address.Address{PaymentPart: make([]byte, 0), StakingPart: vkh[:], Network: 1, AddressType: Address.NONE_KEY, HeaderByte: 0b11100001, Hrp: "stake"}
	if addr.String() != "stake1uyevw2xnsc0pvn9t9r9c7qryfqfeerchgrlm3ea2nefr9hqxdekzz" {
		t.Errorf("PaymentAddress12Reward() failed")
	}
}
