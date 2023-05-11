package TxBuilder

import (
	"github.com/github.com/salvionied/apollo/serialization/PlutusData"
	"github.com/github.com/salvionied/apollo/serialization/Redeemer"
)

type MintingScriptToRedeemer struct {
	Script      PlutusData.ScriptHashable
	Redeemer    Redeemer.Redeemer
	HasRedeemer bool
}
