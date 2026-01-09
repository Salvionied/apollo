package TxBuilder

import (
	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/Redeemer"
)

type MintingScriptToRedeemer struct {
	Script      PlutusData.ScriptHashable
	Redeemer    Redeemer.Redeemer
	HasRedeemer bool
}
