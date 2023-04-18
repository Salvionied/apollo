package TxBuilder

import (
	"Salvionied/apollo/serialization/PlutusData"
	"Salvionied/apollo/serialization/Redeemer"
)

type MintingScriptToRedeemer struct {
	Script      PlutusData.ScriptHashable
	Redeemer    Redeemer.Redeemer
	HasRedeemer bool
}
