package TxBuilder

import (
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/SundaeSwap-finance/apollo/serialization/Redeemer"
)

type MintingScriptToRedeemer struct {
	Script      PlutusData.ScriptHashable
	Redeemer    Redeemer.Redeemer
	HasRedeemer bool
}
